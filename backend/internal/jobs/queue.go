package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/eterno/backend/internal/models"
)

type Queue struct {
	db *sql.DB
}

func NewQueue(db *sql.DB) *Queue {
	return &Queue{db: db}
}

func (q *Queue) Enqueue(ctx context.Context, userID, platform, jobType string, payload map[string]interface{}) (*models.SyncJob, error) {
	payloadJSON, _ := json.Marshal(payload)
	query := `
		INSERT INTO sync_jobs (user_id, platform, job_type, status, next_run_at, payload)
		VALUES ($1, $2, $3, 'PENDING', NOW(), $4)
		RETURNING id, user_id, platform, job_type, status, retry_count, max_retries,
		          next_run_at, locked_at, locked_by, payload, created_at, updated_at
	`
	var job models.SyncJob
	var lockedAt sql.NullTime
	var lockedBy sql.NullString
	var rawPayload []byte

	err := q.db.QueryRowContext(ctx, query, userID, platform, jobType, payloadJSON).Scan(
		&job.ID, &job.UserID, &job.Platform, &job.JobType, &job.Status,
		&job.RetryCount, &job.MaxRetries, &job.NextRunAt, &lockedAt,
		&lockedBy, &rawPayload, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue sync job: %w", err)
	}

	job.Payload = rawPayload
	return &job, nil
}

func (q *Queue) ClaimNextJob(ctx context.Context, workerID string) (*models.SyncJob, error) {
	tx, err := q.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, user_id, platform, job_type, status, retry_count, max_retries,
		       next_run_at, locked_at, locked_by, payload, created_at, updated_at
		FROM sync_jobs
		WHERE status = 'PENDING'
		  AND next_run_at <= NOW()
		ORDER BY next_run_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var job models.SyncJob
	var lockedAt sql.NullTime
	var lockedBy sql.NullString
	var rawPayload []byte

	err = tx.QueryRowContext(ctx, query).Scan(
		&job.ID, &job.UserID, &job.Platform, &job.JobType, &job.Status,
		&job.RetryCount, &job.MaxRetries, &job.NextRunAt, &lockedAt,
		&lockedBy, &rawPayload, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No jobs available
		}
		return nil, err
	}

	updateQuery := `
		UPDATE sync_jobs
		SET status = 'RUNNING', locked_at = NOW(), locked_by = $1, updated_at = NOW()
		WHERE id = $2
	`
	if _, err := tx.ExecContext(ctx, updateQuery, workerID, job.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	job.Status = "RUNNING"
	job.Payload = rawPayload
	return &job, nil
}

func (q *Queue) CompleteJob(ctx context.Context, jobID string, recordsIngested, recordsVerified int) error {
	tx, err := q.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	updateJob := `
		UPDATE sync_jobs
		SET status = 'COMPLETED', locked_at = NULL, locked_by = NULL, updated_at = NOW()
		WHERE id = $1
	`
	if _, err := tx.ExecContext(ctx, updateJob, jobID); err != nil {
		return err
	}

	insertRun := `
		INSERT INTO sync_runs (job_id, status, records_ingested, records_verified, started_at, completed_at)
		VALUES ($1, 'SUCCESS', $2, $3, NOW(), NOW())
	`
	if _, err := tx.ExecContext(ctx, insertRun, jobID, recordsIngested, recordsVerified); err != nil {
		return err
	}

	return tx.Commit()
}

func (q *Queue) FailJob(ctx context.Context, jobID string, errMsg string, backoffSec int) error {
	tx, err := q.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	updateJob := `
		UPDATE sync_jobs
		SET retry_count = retry_count + 1,
		    status = CASE WHEN retry_count + 1 >= max_retries THEN 'FAILED' ELSE 'PENDING' END,
		    next_run_at = NOW() + ($1 * INTERVAL '1 second'),
		    locked_at = NULL,
		    locked_by = NULL,
		    updated_at = NOW()
		WHERE id = $2
	`
	if _, err := tx.ExecContext(ctx, updateJob, backoffSec, jobID); err != nil {
		return err
	}

	insertRun := `
		INSERT INTO sync_runs (job_id, status, error_message, started_at, completed_at)
		VALUES ($1, 'FAILED', $2, NOW(), NOW())
	`
	if _, err := tx.ExecContext(ctx, insertRun, jobID, errMsg); err != nil {
		return err
	}

	return tx.Commit()
}
