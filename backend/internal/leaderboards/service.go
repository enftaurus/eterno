package leaderboards

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/scoring"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

type FilterParams struct {
	InstitutionID string
	DepartmentID  *string
	BatchID       *string
	GroupID       *string
	ChallengeID   *string
	Timeframe     string // "all", "monthly", "weekly", "daily"
	Limit         int
}

func (s *Service) GetLeaderboard(ctx context.Context, filter FilterParams) ([]models.LeaderboardEntry, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 100
	}

	var timeFilterSQL string
	switch filter.Timeframe {
	case "daily":
		timeFilterSQL = "AND se.created_at >= NOW() - INTERVAL '24 hours'"
	case "weekly":
		timeFilterSQL = "AND se.created_at >= NOW() - INTERVAL '7 days'"
	case "monthly":
		timeFilterSQL = "AND se.created_at >= NOW() - INTERVAL '30 days'"
	default:
		timeFilterSQL = ""
	}

	query := fmt.Sprintf(`
		SELECT 
			u.id AS user_id,
			u.display_name,
			u.avatar_url,
			d.name AS department_name,
			b.name AS batch_name,
			COALESCE(SUM(se.points), 0)::int AS total_points,
			COUNT(DISTINCT cc.id)::int AS verified_completions,
			MAX(se.created_at) AS last_verified_solve_at
		FROM users u
		JOIN institution_memberships im ON u.id = im.user_id AND im.institution_id = $1 AND im.role = 'STUDENT'
		LEFT JOIN departments d ON im.department_id = d.id
		LEFT JOIN batches b ON im.batch_id = b.id
		LEFT JOIN score_events se ON u.id = se.user_id AND se.institution_id = $1 %s
		LEFT JOIN challenge_completions cc ON u.id = cc.user_id
		WHERE im.is_active = TRUE
	`, timeFilterSQL)

	args := []interface{}{filter.InstitutionID}
	argIdx := 2

	if filter.DepartmentID != nil && *filter.DepartmentID != "" {
		query += fmt.Sprintf(" AND im.department_id = $%d", argIdx)
		args = append(args, *filter.DepartmentID)
		argIdx++
	}

	if filter.BatchID != nil && *filter.BatchID != "" {
		query += fmt.Sprintf(" AND im.batch_id = $%d", argIdx)
		args = append(args, *filter.BatchID)
		argIdx++
	}

	if filter.GroupID != nil && *filter.GroupID != "" {
		query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM group_memberships gm WHERE gm.group_id = $%d AND gm.user_id = u.id)", argIdx)
		args = append(args, *filter.GroupID)
		argIdx++
	}

	if filter.ChallengeID != nil && *filter.ChallengeID != "" {
		query += fmt.Sprintf(" AND se.challenge_id = $%d", argIdx)
		args = append(args, *filter.ChallengeID)
		argIdx++
	}

	query += `
		GROUP BY u.id, u.display_name, u.avatar_url, d.name, b.name
	`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []models.LeaderboardEntry
	for rows.Next() {
		var e models.LeaderboardEntry
		var avatarURL, deptName, batchName sql.NullString
		var lastSolve sql.NullTime

		if err := rows.Scan(
			&e.UserID, &e.DisplayName, &avatarURL, &deptName, &batchName,
			&e.TotalPoints, &e.VerifiedCompletions, &lastSolve,
		); err != nil {
			return nil, err
		}

		if avatarURL.Valid {
			e.AvatarURL = &avatarURL.String
		}
		if deptName.Valid {
			e.DepartmentName = &deptName.String
		}
		if batchName.Valid {
			e.BatchName = &batchName.String
		}
		if lastSolve.Valid {
			e.LastVerifiedSolveAt = &lastSolve.Time
		}

		entries = append(entries, e)
	}

	// Deterministic sorting via Go using scoring.CompareLeaderboardEntries
	sort.Slice(entries, func(i, j int) bool {
		return scoring.CompareLeaderboardEntries(entries[i], entries[j])
	})

	// Assign 1-based ranks
	if len(entries) > filter.Limit {
		entries = entries[:filter.Limit]
	}
	for i := range entries {
		entries[i].Rank = i + 1
	}

	return entries, nil
}
