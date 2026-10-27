package institutions

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/eterno/backend/internal/auth"
	"github.com/eterno/backend/internal/database"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

type StudentFilter struct {
	InstitutionID  string
	SearchQuery    string
	DepartmentID   *string
	BatchID        *string
	ActivityStatus string // "all", "active", "inactive"
	Limit          int
	Offset         int
}

type StudentListItem struct {
	ID                     string     `json:"id"`
	Email                  string     `json:"email"`
	DisplayName            string     `json:"display_name"`
	AvatarURL              *string    `json:"avatar_url"`
	DepartmentID           *string    `json:"department_id"`
	DepartmentName         *string    `json:"department_name"`
	BatchID                *string    `json:"batch_id"`
	BatchName              *string    `json:"batch_name"`
	StudentIDNumber        *string    `json:"student_id_number"`
	LeetCodeUsername       *string    `json:"leetcode_username"`
	LeetCodeVerification   *string    `json:"leetcode_verification"`
	LastSyncAt             *time.Time `json:"last_sync_at"`
	VerifiedSolvesCount    int        `json:"verified_solves_count"`
	TotalPoints            int        `json:"total_points"`
	CompletedAssignedCount int        `json:"completed_assigned_count"`
	JoinedAt               time.Time  `json:"joined_at"`
}

func (s *Service) ListStudents(ctx context.Context, f StudentFilter) ([]StudentListItem, int, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}

	where := "WHERE im.institution_id = $1 AND im.role = 'STUDENT' AND im.is_active = TRUE"
	args := []interface{}{f.InstitutionID}
	argIdx := 2

	if f.SearchQuery != "" {
		where += fmt.Sprintf(" AND (u.display_name ILIKE $%d OR u.email ILIKE $%d OR im.student_id_number ILIKE $%d OR pa.username ILIKE $%d)", argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+f.SearchQuery+"%")
		argIdx++
	}

	if f.DepartmentID != nil && *f.DepartmentID != "" {
		where += fmt.Sprintf(" AND im.department_id = $%d", argIdx)
		args = append(args, *f.DepartmentID)
		argIdx++
	}

	if f.BatchID != nil && *f.BatchID != "" {
		where += fmt.Sprintf(" AND im.batch_id = $%d", argIdx)
		args = append(args, *f.BatchID)
		argIdx++
	}

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		JOIN institution_memberships im ON u.id = im.user_id
		LEFT JOIN platform_accounts pa ON u.id = pa.user_id AND pa.platform = 'leetcode'
		%s
	`, where)

	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count students: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT 
			u.id, u.email, u.display_name, u.avatar_url,
			im.department_id, d.name AS department_name,
			im.batch_id, b.name AS batch_name,
			im.student_id_number,
			pa.username AS lc_username,
			pa.verification_status AS lc_status,
			pa.last_sync_at,
			COUNT(DISTINCT cc.id)::int AS verified_solves,
			COALESCE(SUM(se.points), 0)::int AS total_points,
			COUNT(DISTINCT CASE WHEN ca.status = 'VERIFIED' THEN ca.id END)::int AS completed_assigned,
			im.joined_at
		FROM users u
		JOIN institution_memberships im ON u.id = im.user_id
		LEFT JOIN departments d ON im.department_id = d.id
		LEFT JOIN batches b ON im.batch_id = b.id
		LEFT JOIN platform_accounts pa ON u.id = pa.user_id AND pa.platform = 'leetcode'
		LEFT JOIN challenge_completions cc ON u.id = cc.user_id
		LEFT JOIN score_events se ON u.id = se.user_id AND se.institution_id = $1
		LEFT JOIN challenge_assignments ca ON u.id = ca.user_id
		%s
		GROUP BY u.id, u.email, u.display_name, u.avatar_url,
		         im.department_id, d.name, im.batch_id, b.name,
		         im.student_id_number, pa.username, pa.verification_status,
		         pa.last_sync_at, im.joined_at
		ORDER BY total_points DESC, u.display_name ASC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, f.Limit, f.Offset)

	rows, err := s.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query students: %w", err)
	}
	defer rows.Close()

	var students []StudentListItem
	for rows.Next() {
		var sItem StudentListItem
		var avatarURL, deptID, deptName, batchID, batchName, studentNum, lcUser, lcStatus sql.NullString
		var lastSync sql.NullTime

		if err := rows.Scan(
			&sItem.ID, &sItem.Email, &sItem.DisplayName, &avatarURL,
			&deptID, &deptName, &batchID, &batchName,
			&studentNum, &lcUser, &lcStatus, &lastSync,
			&sItem.VerifiedSolvesCount, &sItem.TotalPoints, &sItem.CompletedAssignedCount,
			&sItem.JoinedAt,
		); err != nil {
			return nil, 0, err
		}

		if avatarURL.Valid {
			sItem.AvatarURL = &avatarURL.String
		}
		if deptID.Valid {
			sItem.DepartmentID = &deptID.String
		}
		if deptName.Valid {
			sItem.DepartmentName = &deptName.String
		}
		if batchID.Valid {
			sItem.BatchID = &batchID.String
		}
		if batchName.Valid {
			sItem.BatchName = &batchName.String
		}
		if studentNum.Valid {
			sItem.StudentIDNumber = &studentNum.String
		}
		if lcUser.Valid {
			sItem.LeetCodeUsername = &lcUser.String
		}
		if lcStatus.Valid {
			sItem.LeetCodeVerification = &lcStatus.String
		}
		if lastSync.Valid {
			sItem.LastSyncAt = &lastSync.Time
		}

		students = append(students, sItem)
	}

	return students, totalCount, nil
}

type CSVImportResult struct {
	TotalRows int      `json:"total_rows"`
	Imported  int      `json:"imported"`
	Skipped   int      `json:"skipped"`
	Errors    []string `json:"errors"`
}

func (s *Service) ImportStudentsCSV(ctx context.Context, institutionID string, r io.Reader) (*CSVImportResult, error) {
	// Bounded read: Max 2MB CSV file to prevent memory exhaustion
	limitReader := io.LimitReader(r, 2*1024*1024)
	csvReader := csv.NewReader(limitReader)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV format or file exceeds 2MB limit: %w", err)
	}

	if len(records) < 2 {
		return &CSVImportResult{TotalRows: 0, Imported: 0, Skipped: 0, Errors: []string{"CSV must contain a header row and at least one student data row"}}, nil
	}

	if len(records) > 1001 {
		return nil, errors.New("CSV exceeds maximum batch limit of 1000 student records")
	}

	result := &CSVImportResult{
		TotalRows: len(records) - 1,
		Errors:    make([]string, 0),
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	seenEmails := make(map[string]int)

	for i, row := range records[1:] {
		rowNum := i + 2
		if len(row) < 2 {
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: missing required columns (email, display_name)", rowNum))
			result.Skipped++
			continue
		}

		email := strings.TrimSpace(strings.ToLower(row[0]))
		displayName := strings.TrimSpace(row[1])
		deptName := ""
		batchName := ""
		studentIDNum := ""
		lcUsername := ""

		if len(row) > 2 {
			deptName = strings.TrimSpace(row[2])
		}
		if len(row) > 3 {
			batchName = strings.TrimSpace(row[3])
		}
		if len(row) > 4 {
			studentIDNum = strings.TrimSpace(row[4])
		}
		if len(row) > 5 {
			lcUsername = strings.TrimSpace(row[5])
		}

		if email == "" || displayName == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: email and display_name must not be empty", rowNum))
			result.Skipped++
			continue
		}

		// Security: Validate email format
		if !emailRegex.MatchString(email) {
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: invalid email address %q", rowNum, email))
			result.Skipped++
			continue
		}

		// Security: Prevent duplicate emails within same CSV file
		if prevRow, exists := seenEmails[email]; exists {
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: duplicate email %q (already seen at row %d)", rowNum, email, prevRow))
			result.Skipped++
			continue
		}
		seenEmails[email] = rowNum

		// Security: Sanitize formula injection prefixes in text fields
		for _, prefix := range []string{"=", "+", "-", "@", "\t", "\r"} {
			if strings.HasPrefix(displayName, prefix) {
				displayName = "'" + displayName
				break
			}
		}

		err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
			// Find or create user
			var userID string
			userQuery := `SELECT id FROM users WHERE email = $1`
			err := tx.QueryRowContext(ctx, userQuery, email).Scan(&userID)
			if err == sql.ErrNoRows {
				defaultHash, _ := auth.HashPassword("ChangeMe123!")
				insertUser := `
					INSERT INTO users (email, password_hash, display_name, role)
					VALUES ($1, $2, $3, 'STUDENT')
					RETURNING id
				`
				err = tx.QueryRowContext(ctx, insertUser, email, defaultHash, displayName).Scan(&userID)
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			// Find department
			var deptID sql.NullString
			if deptName != "" {
				deptQuery := `SELECT id FROM departments WHERE institution_id = $1 AND name ILIKE $2`
				_ = tx.QueryRowContext(ctx, deptQuery, institutionID, deptName).Scan(&deptID)
			}

			// Find batch
			var bID sql.NullString
			if batchName != "" {
				batchQuery := `SELECT id FROM batches WHERE institution_id = $1 AND name ILIKE $2`
				_ = tx.QueryRowContext(ctx, batchQuery, institutionID, batchName).Scan(&bID)
			}

			// Insert membership
			imQuery := `
				INSERT INTO institution_memberships (
					institution_id, user_id, department_id, batch_id, student_id_number, role
				) VALUES ($1, $2, $3, $4, $5, 'STUDENT')
				ON CONFLICT (institution_id, user_id) DO UPDATE
				SET department_id = COALESCE(EXCLUDED.department_id, institution_memberships.department_id),
				    batch_id = COALESCE(EXCLUDED.batch_id, institution_memberships.batch_id),
				    student_id_number = COALESCE(EXCLUDED.student_id_number, institution_memberships.student_id_number)
			`
			var sidParam interface{} = nil
			if studentIDNum != "" {
				sidParam = studentIDNum
			}
			var didParam interface{} = nil
			if deptID.Valid {
				didParam = deptID.String
			}
			var bidParam interface{} = nil
			if bID.Valid {
				bidParam = bID.String
			}

			_, err = tx.ExecContext(ctx, imQuery, institutionID, userID, didParam, bidParam, sidParam)
			if err != nil {
				return err
			}

			// Link platform account if provided
			if lcUsername != "" {
				paQuery := `
					INSERT INTO platform_accounts (user_id, platform, username, verification_status)
					VALUES ($1, 'leetcode', $2, 'UNVERIFIED')
					ON CONFLICT (user_id, platform) DO NOTHING
				`
				_, _ = tx.ExecContext(ctx, paQuery, userID, lcUsername)
			}

			return nil
		})

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: %v", rowNum, err))
			result.Skipped++
		} else {
			result.Imported++
		}
	}

	return result, nil
}
