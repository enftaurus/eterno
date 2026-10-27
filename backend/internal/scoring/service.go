package scoring

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/eterno/backend/internal/models"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) AwardPoints(ctx context.Context, tx *sql.Tx, userID, institutionID string, challengeID, completionID *string, eventType string, points int, description string) (*models.ScoreEvent, error) {
	query := `
		INSERT INTO score_events (
			user_id, institution_id, challenge_id, completion_id,
			event_type, points, policy_version, description, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, user_id, institution_id, challenge_id, completion_id,
		          event_type, points, policy_version, description, created_at
	`

	var chID, compID, desc sql.NullString
	if challengeID != nil && *challengeID != "" {
		chID.String = *challengeID
		chID.Valid = true
	}
	if completionID != nil && *completionID != "" {
		compID.String = *completionID
		compID.Valid = true
	}
	if description != "" {
		desc.String = description
		desc.Valid = true
	}

	var se models.ScoreEvent
	var execer interface {
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	} = s.db
	if tx != nil {
		execer = tx
	}

	err := execer.QueryRowContext(ctx, query,
		userID, institutionID, chID, compID,
		eventType, points, "v1", desc,
	).Scan(
		&se.ID, &se.UserID, &se.InstitutionID, &chID, &compID,
		&se.EventType, &se.Points, &se.PolicyVersion, &desc, &se.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to record score event: %w", err)
	}

	if chID.Valid {
		se.ChallengeID = &chID.String
	}
	if compID.Valid {
		se.CompletionID = &compID.String
	}
	if desc.Valid {
		se.Description = &desc.String
	}

	return &se, nil
}

func (s *Service) GetUserTotalPoints(ctx context.Context, userID, institutionID string) (int, error) {
	query := `
		SELECT COALESCE(SUM(points), 0)
		FROM score_events
		WHERE user_id = $1 AND institution_id = $2
	`
	var total int
	err := s.db.QueryRowContext(ctx, query, userID, institutionID).Scan(&total)
	return total, err
}

func (s *Service) GetUserScoreHistory(ctx context.Context, userID string, limit int) ([]models.ScoreEvent, error) {
	if limit <= 0 {
		limit = 30
	}
	query := `
		SELECT id, user_id, institution_id, challenge_id, completion_id,
		       event_type, points, policy_version, description, created_at
		FROM score_events
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.ScoreEvent
	for rows.Next() {
		var se models.ScoreEvent
		var chID, compID, desc sql.NullString
		if err := rows.Scan(
			&se.ID, &se.UserID, &se.InstitutionID, &chID, &compID,
			&se.EventType, &se.Points, &se.PolicyVersion, &desc, &se.CreatedAt,
		); err != nil {
			return nil, err
		}
		if chID.Valid {
			se.ChallengeID = &chID.String
		}
		if compID.Valid {
			se.CompletionID = &compID.String
		}
		if desc.Valid {
			se.Description = &desc.String
		}
		events = append(events, se)
	}
	return events, nil
}

func CalculateProblemPoints(difficulty string, basePoints int) int {
	if basePoints <= 0 {
		basePoints = 100
	}
	switch difficulty {
	case "Easy":
		return basePoints
	case "Medium":
		return int(float64(basePoints) * 1.5)
	case "Hard":
		return basePoints * 2
	default:
		return basePoints
	}
}

// ComputeTieBreak compares two student entries deterministically
// 1. Total eligible points (desc)
// 2. Number of verified challenge completions (desc)
// 3. Earlier achievement of the tied score (asc)
func CompareLeaderboardEntries(a, b models.LeaderboardEntry) bool {
	if a.TotalPoints != b.TotalPoints {
		return a.TotalPoints > b.TotalPoints
	}
	if a.VerifiedCompletions != b.VerifiedCompletions {
		return a.VerifiedCompletions > b.VerifiedCompletions
	}
	if a.LastVerifiedSolveAt != nil && b.LastVerifiedSolveAt != nil {
		if !a.LastVerifiedSolveAt.Equal(*b.LastVerifiedSolveAt) {
			return a.LastVerifiedSolveAt.Before(*b.LastVerifiedSolveAt)
		}
	} else if a.LastVerifiedSolveAt != nil {
		return true
	} else if b.LastVerifiedSolveAt != nil {
		return false
	}
	return a.UserID < b.UserID
}
