package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/eterno/backend/internal/jobs"
	"github.com/eterno/backend/internal/models"
)

// Tests 45-60: Background Worker & Sync Queue Hardening, Concurrency, and Stale Recovery

func TestWorkerClaimNextJobLocking(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, userID := createTestUserAndInst(t, db)
	queue := jobs.NewQueue(db)

	jobType := "TEST_LOCK_" + fmt.Sprintf("%d", time.Now().UnixNano())
	enqueued, err := queue.Enqueue(ctx, userID, models.PlatformLeetCode, jobType, map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("failed to enqueue job: %v", err)
	}

	workerID := "worker-lock-alpha"
	claimed, err := queue.ClaimNextJob(ctx, workerID)
	if err != nil {
		t.Fatalf("failed to claim job: %v", err)
	}
	if claimed == nil {
		t.Fatalf("expected to claim a job, got nil")
	}

	// Verify claimed job matches enqueued job
	if claimed.ID != enqueued.ID {
		t.Errorf("expected claimed job ID %s, got %s", enqueued.ID, claimed.ID)
	}
	if claimed.Status != "RUNNING" {
		t.Errorf("expected job status RUNNING, got %s", claimed.Status)
	}

	// Database state check: locked_by and locked_at must be set
	var dbStatus, dbLockedBy string
	var dbLockedAt time.Time
	err = db.QueryRowContext(ctx, "SELECT status, locked_by, locked_at FROM sync_jobs WHERE id = $1", claimed.ID).
		Scan(&dbStatus, &dbLockedBy, &dbLockedAt)
	if err != nil {
		t.Fatalf("failed to query job: %v", err)
	}
	if dbStatus != "RUNNING" || dbLockedBy != workerID || dbLockedAt.IsZero() {
		t.Errorf("invalid DB state: status=%s, locked_by=%s, locked_at=%v", dbStatus, dbLockedBy, dbLockedAt)
	}

	// Second worker should NOT be able to claim this running job
	secondClaim, err := queue.ClaimNextJob(ctx, "worker-lock-beta")
	if err != nil {
		t.Fatalf("error claiming second job: %v", err)
	}
	if secondClaim != nil && secondClaim.ID == claimed.ID {
		t.Errorf("second worker claimed already running job %s", secondClaim.ID)
	}

	// Clean up job
	_ = queue.CompleteJob(ctx, claimed.ID, 0, 0)
}

func TestWorkerConcurrentDisjointClaims(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	queue := jobs.NewQueue(db)
	numJobs := 10
	numWorkers := 5

	var jobIDs []string
	prefix := fmt.Sprintf("CONCURRENT_%d_", time.Now().UnixNano())

	// Enqueue 10 distinct jobs
	for i := 0; i < numJobs; i++ {
		_, uid := createTestUserAndInst(t, db)
		j, err := queue.Enqueue(ctx, uid, models.PlatformLeetCode, prefix+fmt.Sprintf("%d", i), nil)
		if err != nil {
			t.Fatalf("failed to enqueue job %d: %v", i, err)
		}
		jobIDs = append(jobIDs, j.ID)
	}

	var mu sync.Mutex
	claimedMap := make(map[string]string) // jobID -> workerID
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		workerName := fmt.Sprintf("worker-pool-%d", w)
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			for {
				job, err := queue.ClaimNextJob(ctx, name)
				if err != nil {
					t.Errorf("worker %s error claiming job: %v", name, err)
					return
				}
				if job == nil {
					return // no more jobs
				}

				mu.Lock()
				if prevWorker, exists := claimedMap[job.ID]; exists {
					t.Errorf("RACE CONDITION: Job %s claimed by both %s and %s", job.ID, prevWorker, name)
				}
				claimedMap[job.ID] = name
				mu.Unlock()

				// Complete the job to simulate short work
				_ = queue.CompleteJob(ctx, job.ID, 1, 1)
			}
		}(workerName)
	}

	wg.Wait()

	// Verify all 10 jobs were claimed exactly once
	for _, jid := range jobIDs {
		if _, ok := claimedMap[jid]; !ok {
			t.Errorf("Job %s was never claimed by any worker", jid)
		}
	}
}

func TestWorkerStaleJobLeaseRecovery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, userID := createTestUserAndInst(t, db)
	queue := jobs.NewQueue(db)

	jobType := "STALE_JOB_" + fmt.Sprintf("%d", time.Now().UnixNano())
	job, err := queue.Enqueue(ctx, userID, models.PlatformLeetCode, jobType, nil)
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// Claim it initially
	claimed, err := queue.ClaimNextJob(ctx, "crashed-worker")
	if err != nil || claimed == nil {
		t.Fatalf("failed to claim initial job: %v", err)
	}

	// Simulate crash: worker held lease and crashed. Force locked_at to 15 minutes ago.
	staleTime := time.Now().UTC().Add(-15 * time.Minute)
	_, err = db.ExecContext(ctx, "UPDATE sync_jobs SET locked_at = $1 WHERE id = $2", staleTime, job.ID)
	if err != nil {
		t.Fatalf("failed to backdate locked_at: %v", err)
	}

	// Run stale recovery with 5-minute lease threshold
	recovered, err := queue.RecoverStaleJobs(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("RecoverStaleJobs failed: %v", err)
	}
	if recovered < 1 {
		t.Errorf("expected at least 1 recovered job, got %d", recovered)
	}

	// Verify job is back in PENDING status, locked_by and locked_at cleared, retry_count incremented
	var status string
	var retryCount int
	var lockedBy *string
	var lockedAt *time.Time
	err = db.QueryRowContext(ctx, "SELECT status, retry_count, locked_by, locked_at FROM sync_jobs WHERE id = $1", job.ID).
		Scan(&status, &retryCount, &lockedBy, &lockedAt)
	if err != nil {
		t.Fatalf("failed to inspect recovered job: %v", err)
	}
	if status != "PENDING" {
		t.Errorf("expected recovered job to be PENDING, got %s", status)
	}
	if retryCount != 1 {
		t.Errorf("expected retry_count = 1, got %d", retryCount)
	}
	if lockedBy != nil || lockedAt != nil {
		t.Errorf("expected locks to be cleared, got lockedBy=%v, lockedAt=%v", lockedBy, lockedAt)
	}

	// Healthy worker claims recovered job
	reclaimed, err := queue.ClaimNextJob(ctx, "healthy-worker")
	if err != nil || reclaimed == nil {
		t.Fatalf("failed to reclaim recovered job: %v", err)
	}
	if reclaimed.ID != job.ID {
		t.Errorf("expected to reclaim job %s, got %s", job.ID, reclaimed.ID)
	}

	_ = queue.CompleteJob(ctx, reclaimed.ID, 0, 0)
}

func TestWorkerDeduplication(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, userID := createTestUserAndInst(t, db)
	queue := jobs.NewQueue(db)

	jobType := "DEDUP_TEST_" + fmt.Sprintf("%d", time.Now().UnixNano())

	// First enqueue -> creates PENDING job
	job1, err := queue.Enqueue(ctx, userID, models.PlatformLeetCode, jobType, map[string]interface{}{"run": 1})
	if err != nil {
		t.Fatalf("first enqueue failed: %v", err)
	}
	if job1.Status != "PENDING" {
		t.Fatalf("expected job1 to be PENDING, got %s", job1.Status)
	}

	// Second enqueue while PENDING -> returns existing job1
	job2, err := queue.Enqueue(ctx, userID, models.PlatformLeetCode, jobType, map[string]interface{}{"run": 2})
	if err != nil {
		t.Fatalf("second enqueue failed: %v", err)
	}
	if job2.ID != job1.ID {
		t.Errorf("expected duplicate enqueue to return same job ID, got %s vs %s", job1.ID, job2.ID)
	}

	// Claim job -> becomes RUNNING
	claimed, err := queue.ClaimNextJob(ctx, "worker-dedup")
	if err != nil || claimed == nil {
		t.Fatalf("claim failed: %v", err)
	}

	// Third enqueue while RUNNING -> returns existing job1
	job3, err := queue.Enqueue(ctx, userID, models.PlatformLeetCode, jobType, map[string]interface{}{"run": 3})
	if err != nil {
		t.Fatalf("third enqueue failed: %v", err)
	}
	if job3.ID != job1.ID {
		t.Errorf("expected enqueue while RUNNING to return same job ID, got %s vs %s", job1.ID, job3.ID)
	}

	// Complete the job
	err = queue.CompleteJob(ctx, job1.ID, 10, 5)
	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	// Fourth enqueue after COMPLETED -> should create a brand new job!
	job4, err := queue.Enqueue(ctx, userID, models.PlatformLeetCode, jobType, map[string]interface{}{"run": 4})
	if err != nil {
		t.Fatalf("fourth enqueue failed: %v", err)
	}
	if job4.ID == job1.ID {
		t.Errorf("expected new job ID after completion, but got same ID: %s", job4.ID)
	}

	_ = queue.CompleteJob(ctx, job4.ID, 0, 0)
}

func TestWorkerMaxRetriesTerminalFailed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, userID := createTestUserAndInst(t, db)
	queue := jobs.NewQueue(db)

	jobType := "RETRY_FAIL_" + fmt.Sprintf("%d", time.Now().UnixNano())
	job, err := queue.Enqueue(ctx, userID, models.PlatformLeetCode, jobType, nil)
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	// Set max_retries = 3 explicitly
	_, err = db.ExecContext(ctx, "UPDATE sync_jobs SET max_retries = 3 WHERE id = $1", job.ID)
	if err != nil {
		t.Fatalf("update max_retries failed: %v", err)
	}

	// Retry 1: Fail with backoff 0
	claimed1, _ := queue.ClaimNextJob(ctx, "worker-retry")
	if claimed1 == nil {
		t.Fatalf("claimed1 nil")
	}
	_ = queue.FailJob(ctx, claimed1.ID, "transient error 1", 0)

	// Verify retry_count = 1, status = PENDING
	var status string
	var retries int
	_ = db.QueryRowContext(ctx, "SELECT status, retry_count FROM sync_jobs WHERE id = $1", job.ID).Scan(&status, &retries)
	if status != "PENDING" || retries != 1 {
		t.Errorf("attempt 1: status=%s, retries=%d", status, retries)
	}

	// Retry 2: Fail again
	claimed2, _ := queue.ClaimNextJob(ctx, "worker-retry")
	if claimed2 == nil {
		t.Fatalf("claimed2 nil")
	}
	_ = queue.FailJob(ctx, claimed2.ID, "transient error 2", 0)

	_ = db.QueryRowContext(ctx, "SELECT status, retry_count FROM sync_jobs WHERE id = $1", job.ID).Scan(&status, &retries)
	if status != "PENDING" || retries != 2 {
		t.Errorf("attempt 2: status=%s, retries=%d", status, retries)
	}

	// Retry 3: Fail final time (retries reach max_retries 3) -> should become FAILED
	claimed3, _ := queue.ClaimNextJob(ctx, "worker-retry")
	if claimed3 == nil {
		t.Fatalf("claimed3 nil")
	}
	_ = queue.FailJob(ctx, claimed3.ID, "permanent fatal error", 0)

	_ = db.QueryRowContext(ctx, "SELECT status, retry_count FROM sync_jobs WHERE id = $1", job.ID).Scan(&status, &retries)
	if status != "FAILED" || retries != 3 {
		t.Errorf("attempt 3 (terminal): expected FAILED/3, got status=%s, retries=%d", status, retries)
	}

	// Ensure FAILED job is never claimed again
	claimedFinal, _ := queue.ClaimNextJob(ctx, "worker-retry")
	if claimedFinal != nil && claimedFinal.ID == job.ID {
		t.Errorf("claimed FAILED job %s", claimedFinal.ID)
	}
}

func TestWorkerCompleteJobWithRunsAudit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, userID := createTestUserAndInst(t, db)
	queue := jobs.NewQueue(db)

	jobType := "AUDIT_RUN_" + fmt.Sprintf("%d", time.Now().UnixNano())
	job, err := queue.Enqueue(ctx, userID, models.PlatformLeetCode, jobType, nil)
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	claimed, err := queue.ClaimNextJob(ctx, "worker-auditor")
	if err != nil || claimed == nil {
		t.Fatalf("claim failed: %v", err)
	}

	ingested := 42
	verified := 18
	err = queue.CompleteJob(ctx, claimed.ID, ingested, verified)
	if err != nil {
		t.Fatalf("CompleteJob failed: %v", err)
	}

	// Check sync_runs audit table
	var runStatus string
	var dbIngested, dbVerified int
	err = db.QueryRowContext(ctx, `
		SELECT status, records_ingested, records_verified
		FROM sync_runs
		WHERE job_id = $1
		ORDER BY completed_at DESC
		LIMIT 1
	`, job.ID).Scan(&runStatus, &dbIngested, &dbVerified)
	if err != nil {
		t.Fatalf("failed to query sync_runs: %v", err)
	}

	if runStatus != "SUCCESS" || dbIngested != ingested || dbVerified != verified {
		t.Errorf("mismatched sync_run audit: status=%s, ingested=%d, verified=%d", runStatus, dbIngested, dbVerified)
	}
}
