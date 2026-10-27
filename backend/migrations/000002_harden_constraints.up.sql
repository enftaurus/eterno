-- Migration 000002: Harden constraints, idempotency keys, and performance indexes

-- 1. Idempotency constraint on score events: a completion can only be awarded points once
CREATE UNIQUE INDEX IF NOT EXISTS idx_score_events_completion_uniq 
ON score_events (completion_id) 
WHERE completion_id IS NOT NULL;

-- 2. Non-negative constraints for points
ALTER TABLE score_events ADD CONSTRAINT chk_score_events_points_nonneg CHECK (points >= 0);
ALTER TABLE challenge_problems ADD CONSTRAINT chk_challenge_problems_points_nonneg CHECK (points >= 0);
ALTER TABLE challenge_assignments ADD CONSTRAINT chk_challenge_assignments_points_nonneg CHECK (points_awarded >= 0);

-- 3. Stale job recovery index for background workers
CREATE INDEX IF NOT EXISTS idx_sync_jobs_stale_recovery 
ON sync_jobs (status, locked_at) 
WHERE status = 'RUNNING';

-- 4. User submissions query optimization
CREATE INDEX IF NOT EXISTS idx_submissions_user_time 
ON submissions (user_id, submitted_at DESC);
