package submissions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/eterno/backend/internal/models"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) IngestSubmission(ctx context.Context, userID string, platform string, sub models.SubmissionDTO) (*models.Submission, bool, error) {
	query := `
		INSERT INTO submissions (
			user_id, platform, provider_submission_id, problem_slug, title,
			verdict, is_accepted, language, runtime_ms, memory_mb, submitted_at, raw_payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (platform, provider_submission_id) DO UPDATE
		SET raw_payload = EXCLUDED.raw_payload
		RETURNING id, user_id, platform, provider_submission_id, problem_slug,
		          title, verdict, is_accepted, language, runtime_ms, memory_mb,
		          submitted_at, raw_payload, created_at, (xmax = 0) AS is_inserted
	`

	rawPayload, _ := json.Marshal(sub)

	var title, language sql.NullString
	if sub.Title != "" {
		title.String = sub.Title
		title.Valid = true
	}
	if sub.Language != "" {
		language.String = sub.Language
		language.Valid = true
	}

	var runtimeMs sql.NullInt32
	if sub.RuntimeMs > 0 {
		runtimeMs.Int32 = int32(sub.RuntimeMs)
		runtimeMs.Valid = true
	}

	var memoryMB sql.NullFloat64
	if sub.MemoryMB > 0 {
		memoryMB.Float64 = sub.MemoryMB
		memoryMB.Valid = true
	}

	var record models.Submission
	var payloadBytes []byte
	var isInserted bool

	err := s.db.QueryRowContext(ctx, query,
		userID, platform, sub.ProviderID, sub.ProblemSlug, title,
		sub.Verdict, sub.IsAccepted, language, runtimeMs, memoryMB,
		sub.SubmittedAt, rawPayload,
	).Scan(
		&record.ID, &record.UserID, &record.Platform, &record.ProviderSubmissionID, &record.ProblemSlug,
		&title, &record.Verdict, &record.IsAccepted, &language, &runtimeMs, &memoryMB,
		&record.SubmittedAt, &payloadBytes, &record.CreatedAt, &isInserted,
	)

	if err != nil {
		return nil, false, fmt.Errorf("failed to ingest submission: %w", err)
	}

	if title.Valid {
		record.Title = &title.String
	}
	if language.Valid {
		record.Language = &language.String
	}
	if runtimeMs.Valid {
		v := int(runtimeMs.Int32)
		record.RuntimeMs = &v
	}
	if memoryMB.Valid {
		record.MemoryMB = &memoryMB.Float64
	}
	record.RawPayload = payloadBytes

	return &record, isInserted, nil
}

func (s *Service) GetSubmissionsForUser(ctx context.Context, userID string, limit int) ([]models.Submission, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, user_id, platform, provider_submission_id, problem_slug,
		       title, verdict, is_accepted, language, runtime_ms, memory_mb,
		       submitted_at, raw_payload, created_at
		FROM submissions
		WHERE user_id = $1
		ORDER BY submitted_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Submission
	for rows.Next() {
		var sub models.Submission
		var title, language sql.NullString
		var runtimeMs sql.NullInt32
		var memoryMB sql.NullFloat64
		var payloadBytes []byte

		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.Platform, &sub.ProviderSubmissionID, &sub.ProblemSlug,
			&title, &sub.Verdict, &sub.IsAccepted, &language, &runtimeMs, &memoryMB,
			&sub.SubmittedAt, &payloadBytes, &sub.CreatedAt,
		); err != nil {
			return nil, err
		}

		if title.Valid {
			sub.Title = &title.String
		}
		if language.Valid {
			sub.Language = &language.String
		}
		if runtimeMs.Valid {
			v := int(runtimeMs.Int32)
			sub.RuntimeMs = &v
		}
		if memoryMB.Valid {
			sub.MemoryMB = &memoryMB.Float64
		}
		sub.RawPayload = payloadBytes

		list = append(list, sub)
	}

	return list, nil
}
