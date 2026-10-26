package challenges

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eterno/backend/internal/database"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/platforms/leetcode"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateChallenge(ctx context.Context, institutionID, userID string, req *models.CreateChallengeRequest) (*models.Challenge, error) {
	if req.StartTime.After(req.Deadline) {
		return nil, errors.New("challenge start time must be before deadline")
	}

	scopeType := req.ScopeType
	if scopeType == "" {
		scopeType = "INSTITUTION"
	}

	var challenge models.Challenge
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		query := `
			INSERT INTO challenges (
				institution_id, title, description, instructions,
				challenge_type, status, start_time, deadline,
				scope_type, scope_id, max_points, historical_allowed, created_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING id, institution_id, title, description, instructions, challenge_type,
			          status, start_time, deadline, scope_type, scope_id, max_points,
			          historical_allowed, created_by, created_at, updated_at
		`
		var instructions, scopeID, createdBy sql.NullString
		if req.Instructions != "" {
			instructions.String = req.Instructions
			instructions.Valid = true
		}
		if req.ScopeID != nil && *req.ScopeID != "" {
			scopeID.String = *req.ScopeID
			scopeID.Valid = true
		}
		if userID != "" {
			createdBy.String = userID
			createdBy.Valid = true
		}

		status := models.ChallengeDraft
		now := time.Now().UTC()
		if req.StartTime.Before(now) || req.StartTime.Equal(now) {
			status = models.ChallengeDraft // created as draft until explicitly published
		}

		err := tx.QueryRowContext(ctx, query,
			institutionID, req.Title, req.Description, instructions,
			req.ChallengeType, status, req.StartTime, req.Deadline,
			scopeType, scopeID, req.MaxPoints, req.HistoricalAllowed, createdBy,
		).Scan(
			&challenge.ID, &challenge.InstitutionID, &challenge.Title, &challenge.Description,
			&instructions, &challenge.ChallengeType, &challenge.Status, &challenge.StartTime,
			&challenge.Deadline, &challenge.ScopeType, &scopeID, &challenge.MaxPoints,
			&challenge.HistoricalAllowed, &createdBy, &challenge.CreatedAt, &challenge.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert challenge: %w", err)
		}

		if instructions.Valid {
			challenge.Instructions = &instructions.String
		}
		if scopeID.Valid {
			challenge.ScopeID = &scopeID.String
		}
		if createdBy.Valid {
			challenge.CreatedBy = &createdBy.String
		}

		// Insert Problems
		probQuery := `
			INSERT INTO challenge_problems (
				challenge_id, platform, problem_url, problem_slug, title,
				difficulty, points, is_bonus, order_index, metadata
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id, challenge_id, platform, problem_url, problem_slug, title,
			          difficulty, points, is_bonus, order_index, metadata, created_at
		`
		for i, p := range req.Problems {
			slug, err := leetcode.ParseProblemSlug(p.ProblemURL)
			if err != nil {
				return fmt.Errorf("problem %d has invalid URL: %w", i+1, err)
			}

			diff := p.Difficulty
			if diff == "" {
				diff = "Medium"
			}
			points := p.Points
			if points <= 0 {
				points = 100
			}

			title := stringsTitle(slug)
			metaJSON, _ := json.Marshal(map[string]interface{}{"source": "manual_entry"})

			var cp models.ChallengeProblem
			var metaBytes []byte
			err = tx.QueryRowContext(ctx, probQuery,
				challenge.ID, models.PlatformLeetCode, p.ProblemURL, slug, title,
				diff, points, p.IsBonus, i, metaJSON,
			).Scan(
				&cp.ID, &cp.ChallengeID, &cp.Platform, &cp.ProblemURL, &cp.ProblemSlug, &cp.Title,
				&cp.Difficulty, &cp.Points, &cp.IsBonus, &cp.OrderIndex, &metaBytes, &cp.CreatedAt,
			)
			if err != nil {
				return fmt.Errorf("failed to insert challenge problem %s: %w", slug, err)
			}
			cp.Metadata = metaBytes
			challenge.Problems = append(challenge.Problems, cp)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (s *Service) PublishChallenge(ctx context.Context, institutionID, challengeID string) (*models.Challenge, error) {
	var ch models.Challenge
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		// Update challenge status
		query := `
			UPDATE challenges
			SET status = $1, updated_at = NOW()
			WHERE id = $2 AND institution_id = $3
			RETURNING id, institution_id, title, description, instructions, challenge_type,
			          status, start_time, deadline, scope_type, scope_id, max_points,
			          historical_allowed, created_by, created_at, updated_at
		`
		var instructions, scopeID, createdBy sql.NullString
		err := tx.QueryRowContext(ctx, query, models.ChallengeActive, challengeID, institutionID).Scan(
			&ch.ID, &ch.InstitutionID, &ch.Title, &ch.Description, &instructions,
			&ch.ChallengeType, &ch.Status, &ch.StartTime, &ch.Deadline, &ch.ScopeType,
			&scopeID, &ch.MaxPoints, &ch.HistoricalAllowed, &createdBy, &ch.CreatedAt, &ch.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("challenge not found or unable to publish: %w", err)
		}

		if instructions.Valid {
			ch.Instructions = &instructions.String
		}
		if scopeID.Valid {
			ch.ScopeID = &scopeID.String
		}
		if createdBy.Valid {
			ch.CreatedBy = &createdBy.String
		}

		// Create challenge assignments for all matching eligible students
		assignQuery := `
			INSERT INTO challenge_assignments (challenge_id, user_id, institution_id, status, baseline_at, assigned_at)
			SELECT $1, im.user_id, $2, $3, NOW(), NOW()
			FROM institution_memberships im
			WHERE im.institution_id = $2
			  AND im.is_active = TRUE
			  AND im.role = 'STUDENT'
			  AND (
			    $4 = 'INSTITUTION'
			    OR ($4 = 'DEPARTMENT' AND im.department_id = $5::uuid)
			    OR ($4 = 'BATCH' AND im.batch_id = $5::uuid)
			    OR ($4 = 'GROUP' AND EXISTS (
			         SELECT 1 FROM group_memberships gm WHERE gm.group_id = $5::uuid AND gm.user_id = im.user_id
			       ))
			  )
			ON CONFLICT (challenge_id, user_id) DO NOTHING
		`
		var scopeIDParam interface{} = nil
		if ch.ScopeID != nil && *ch.ScopeID != "" {
			scopeIDParam = *ch.ScopeID
		}

		_, err = tx.ExecContext(ctx, assignQuery, ch.ID, institutionID, models.AssignmentNotStarted, ch.ScopeType, scopeIDParam)
		if err != nil {
			return fmt.Errorf("failed to generate student assignments: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	problems, err := s.GetProblemsForChallenge(ctx, ch.ID)
	if err == nil {
		ch.Problems = problems
	}

	return &ch, nil
}

func (s *Service) GetProblemsForChallenge(ctx context.Context, challengeID string) ([]models.ChallengeProblem, error) {
	query := `
		SELECT id, challenge_id, platform, problem_url, problem_slug, title,
		       difficulty, points, is_bonus, order_index, metadata, created_at
		FROM challenge_problems
		WHERE challenge_id = $1
		ORDER BY order_index ASC
	`
	rows, err := s.db.QueryContext(ctx, query, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []models.ChallengeProblem
	for rows.Next() {
		var cp models.ChallengeProblem
		var metaBytes []byte
		if err := rows.Scan(
			&cp.ID, &cp.ChallengeID, &cp.Platform, &cp.ProblemURL, &cp.ProblemSlug, &cp.Title,
			&cp.Difficulty, &cp.Points, &cp.IsBonus, &cp.OrderIndex, &metaBytes, &cp.CreatedAt,
		); err != nil {
			return nil, err
		}
		cp.Metadata = metaBytes
		problems = append(problems, cp)
	}
	return problems, nil
}

func (s *Service) ListChallengesForInstitution(ctx context.Context, institutionID string, statusFilter string) ([]models.Challenge, error) {
	query := `
		SELECT id, institution_id, title, description, instructions, challenge_type,
		       status, start_time, deadline, scope_type, scope_id, max_points,
		       historical_allowed, created_by, created_at, updated_at
		FROM challenges
		WHERE institution_id = $1
	`
	args := []interface{}{institutionID}
	if statusFilter != "" {
		query += " AND status = $2"
		args = append(args, statusFilter)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Challenge
	for rows.Next() {
		var ch models.Challenge
		var instructions, scopeID, createdBy sql.NullString
		if err := rows.Scan(
			&ch.ID, &ch.InstitutionID, &ch.Title, &ch.Description, &instructions,
			&ch.ChallengeType, &ch.Status, &ch.StartTime, &ch.Deadline, &ch.ScopeType,
			&scopeID, &ch.MaxPoints, &ch.HistoricalAllowed, &createdBy, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if instructions.Valid {
			ch.Instructions = &instructions.String
		}
		if scopeID.Valid {
			ch.ScopeID = &scopeID.String
		}
		if createdBy.Valid {
			ch.CreatedBy = &createdBy.String
		}
		list = append(list, ch)
	}
	return list, nil
}

func (s *Service) GetChallengeByID(ctx context.Context, challengeID string) (*models.Challenge, error) {
	query := `
		SELECT id, institution_id, title, description, instructions, challenge_type,
		       status, start_time, deadline, scope_type, scope_id, max_points,
		       historical_allowed, created_by, created_at, updated_at
		FROM challenges
		WHERE id = $1
	`
	var ch models.Challenge
	var instructions, scopeID, createdBy sql.NullString
	err := s.db.QueryRowContext(ctx, query, challengeID).Scan(
		&ch.ID, &ch.InstitutionID, &ch.Title, &ch.Description, &instructions,
		&ch.ChallengeType, &ch.Status, &ch.StartTime, &ch.Deadline, &ch.ScopeType,
		&scopeID, &ch.MaxPoints, &ch.HistoricalAllowed, &createdBy, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if instructions.Valid {
		ch.Instructions = &instructions.String
	}
	if scopeID.Valid {
		ch.ScopeID = &scopeID.String
	}
	if createdBy.Valid {
		ch.CreatedBy = &createdBy.String
	}

	problems, err := s.GetProblemsForChallenge(ctx, ch.ID)
	if err == nil {
		ch.Problems = problems
	}

	return &ch, nil
}

func (s *Service) GetStudentAssignments(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT ca.id, ca.challenge_id, ca.user_id, ca.institution_id, ca.status,
		       ca.baseline_at, ca.assigned_at, ca.completed_at, ca.points_awarded,
		       c.title, c.description, c.challenge_type, c.status AS challenge_status,
		       c.start_time, c.deadline, c.max_points
		FROM challenge_assignments ca
		JOIN challenges c ON ca.challenge_id = c.id
		WHERE ca.user_id = $1
		ORDER BY c.deadline ASC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var caID, chID, uID, instID, caStatus string
		var baselineAt, assignedAt time.Time
		var completedAt sql.NullTime
		var pointsAwarded int
		var title, desc, chType, chStatus string
		var startTime, deadline time.Time
		var maxPoints int

		if err := rows.Scan(
			&caID, &chID, &uID, &instID, &caStatus,
			&baselineAt, &assignedAt, &completedAt, &pointsAwarded,
			&title, &desc, &chType, &chStatus,
			&startTime, &deadline, &maxPoints,
		); err != nil {
			return nil, err
		}

		item := map[string]interface{}{
			"id":               caID,
			"challenge_id":     chID,
			"user_id":          uID,
			"institution_id":   instID,
			"status":           caStatus,
			"baseline_at":      baselineAt,
			"assigned_at":      assignedAt,
			"points_awarded":   pointsAwarded,
			"title":            title,
			"description":      desc,
			"challenge_type":   chType,
			"challenge_status": chStatus,
			"start_time":       startTime,
			"deadline":         deadline,
			"max_points":       maxPoints,
		}
		if completedAt.Valid {
			item["completed_at"] = completedAt.Time
		}
		results = append(results, item)
	}
	return results, nil
}

func stringsTitle(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
