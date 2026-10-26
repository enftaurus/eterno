package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/platforms/leetcode"
	"github.com/eterno/backend/internal/submissions"
	"github.com/eterno/backend/internal/verification"
	"github.com/google/uuid"
)

type Worker struct {
	queue        *Queue
	db           *sql.DB
	lcAdapter    *leetcode.Adapter
	subService   *submissions.Service
	verifService *verification.Service
	actService   *activity.Service
	workerID     string
	pollInterval time.Duration
}

func NewWorker(
	queue *Queue,
	db *sql.DB,
	lcAdapter *leetcode.Adapter,
	subService *submissions.Service,
	verifService *verification.Service,
	actService *activity.Service,
	pollInterval time.Duration,
) *Worker {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	return &Worker{
		queue:        queue,
		db:           db,
		lcAdapter:    lcAdapter,
		subService:   subService,
		verifService: verifService,
		actService:   actService,
		workerID:     "worker-" + uuid.New().String()[:8],
		pollInterval: pollInterval,
	}
}

func (w *Worker) Start(ctx context.Context) {
	slog.Info("Starting persistent background worker", "worker_id", w.workerID, "poll_interval", w.pollInterval)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker context canceled, shutting down", "worker_id", w.workerID)
			return
		case <-ticker.C:
			w.ProcessNextJob(ctx)
		}
	}
}

func (w *Worker) ProcessNextJob(ctx context.Context) {
	job, err := w.queue.ClaimNextJob(ctx, w.workerID)
	if err != nil {
		slog.Error("Failed to claim background job", "error", err)
		return
	}
	if job == nil {
		return // No pending jobs
	}

	slog.Info("Processing job", "job_id", job.ID, "job_type", job.JobType, "user_id", job.UserID)

	switch job.JobType {
	case "ACCOUNT_VERIFY":
		w.handleAccountVerify(ctx, job)
	case "USER_SYNC":
		w.handleUserSync(ctx, job)
	default:
		slog.Warn("Unknown job type encountered", "job_type", job.JobType)
		_ = w.queue.CompleteJob(ctx, job.ID, 0, 0)
	}
}

func (w *Worker) handleAccountVerify(ctx context.Context, job *models.SyncJob) {
	var payload struct {
		Username string `json:"username"`
		Token    string `json:"token"`
	}
	_ = json.Unmarshal(job.Payload, &payload)

	verified, profileMeta, err := w.lcAdapter.VerifyAccount(ctx, payload.Username, payload.Token)
	if err != nil {
		slog.Error("Account verification error", "error", err, "username", payload.Username)
		_ = w.queue.FailJob(ctx, job.ID, err.Error(), 10)
		return
	}

	if verified {
		profileJSON, _ := json.Marshal(profileMeta)
		updateQuery := `
			UPDATE platform_accounts
			SET verification_status = 'VERIFIED', verified_at = NOW(),
			    last_sync_status = 'IDLE', sync_error_message = NULL,
			    profile_metadata = $1, updated_at = NOW()
			WHERE user_id = $2 AND platform = 'leetcode'
		`
		_, _ = w.db.ExecContext(ctx, updateQuery, profileJSON, job.UserID)

		// Create notification
		notifQuery := `
			INSERT INTO notifications (user_id, title, message, type, link)
			VALUES ($1, 'LeetCode Account Verified!', 'Your LeetCode profile has been successfully verified.', 'VERIFICATION_SUCCESS', '/settings')
		`
		_, _ = w.db.ExecContext(ctx, notifQuery, job.UserID)

		_ = w.queue.CompleteJob(ctx, job.ID, 0, 0)
		slog.Info("LeetCode account verified successfully", "user_id", job.UserID, "username", payload.Username)
	} else {
		updateQuery := `
			UPDATE platform_accounts
			SET verification_status = 'FAILED',
			    sync_error_message = 'Verification token not found in LeetCode About Me bio',
			    updated_at = NOW()
			WHERE user_id = $1 AND platform = 'leetcode'
		`
		_, _ = w.db.ExecContext(ctx, updateQuery, job.UserID)

		_ = w.queue.FailJob(ctx, job.ID, "token not found in profile bio", 30)
		slog.Warn("LeetCode verification token missing from profile", "user_id", job.UserID, "username", payload.Username)
	}
}

func (w *Worker) handleUserSync(ctx context.Context, job *models.SyncJob) {
	// Look up username
	var username string
	query := `SELECT username FROM platform_accounts WHERE user_id = $1 AND platform = 'leetcode'`
	err := w.db.QueryRowContext(ctx, query, job.UserID).Scan(&username)
	if err != nil {
		slog.Error("User has no linked LeetCode account", "error", err, "user_id", job.UserID)
		_ = w.queue.FailJob(ctx, job.ID, "no linked account found", 60)
		return
	}

	// Mark syncing
	_, _ = w.db.ExecContext(ctx, `
		UPDATE platform_accounts
		SET last_sync_status = 'SYNCING', updated_at = NOW()
		WHERE user_id = $1 AND platform = 'leetcode'
	`, job.UserID)

	// Fetch recent submissions
	submissionsList, err := w.lcAdapter.FetchRecentSubmissions(ctx, username, 30)
	if err != nil {
		slog.Error("Failed to fetch LeetCode submissions", "error", err, "username", username)
		_, _ = w.db.ExecContext(ctx, `
			UPDATE platform_accounts
			SET last_sync_status = 'FAILED', sync_error_message = $1, updated_at = NOW()
			WHERE user_id = $2 AND platform = 'leetcode'
		`, err.Error(), job.UserID)
		_ = w.queue.FailJob(ctx, job.ID, err.Error(), 15)
		return
	}

	recordsIngested := 0
	recordsVerified := 0

	for _, subDTO := range submissionsList {
		// 1. Ingest submission idempotently
		sub, isNew, err := w.subService.IngestSubmission(ctx, job.UserID, models.PlatformLeetCode, subDTO)
		if err != nil {
			slog.Error("Failed to ingest submission record", "error", err, "slug", subDTO.ProblemSlug)
			continue
		}
		if isNew {
			recordsIngested++
		}

		// 2. Ingest into canonical activity events (idempotent ON CONFLICT DO NOTHING)
		actMeta := map[string]interface{}{
			"slug":       subDTO.ProblemSlug,
			"title":      subDTO.Title,
			"verdict":    subDTO.Verdict,
			"is_accepted": subDTO.IsAccepted,
		}
		_ = w.actService.RecordEvent(ctx, job.UserID, "leetcode", subDTO.ProviderID, "accepted_submission", subDTO.SubmittedAt, actMeta)

		// 3. Run verification engine
		verifResult, err := w.verifService.VerifySubmission(ctx, sub)
		if err != nil {
			slog.Error("Verification check failed", "error", err, "slug", sub.ProblemSlug)
			continue
		}
		if verifResult != nil && verifResult.Status == "VERIFIED" {
			recordsVerified++
		}
	}

	// Update platform account last sync
	_, _ = w.db.ExecContext(ctx, `
		UPDATE platform_accounts
		SET last_sync_at = NOW(), last_sync_status = 'SUCCESS', sync_error_message = NULL, updated_at = NOW()
		WHERE user_id = $1 AND platform = 'leetcode'
	`, job.UserID)

	_ = w.queue.CompleteJob(ctx, job.ID, recordsIngested, recordsVerified)
	slog.Info("Completed user synchronization job",
		"user_id", job.UserID,
		"records_ingested", recordsIngested,
		"records_verified", recordsVerified,
	)
}
