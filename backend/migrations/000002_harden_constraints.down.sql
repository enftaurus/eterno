-- Migration 000002 Down: Revert constraint hardening

DROP INDEX IF EXISTS idx_submissions_user_time;
DROP INDEX IF EXISTS idx_sync_jobs_stale_recovery;
ALTER TABLE challenge_assignments DROP CONSTRAINT IF EXISTS chk_challenge_assignments_points_nonneg;
ALTER TABLE challenge_problems DROP CONSTRAINT IF EXISTS chk_challenge_problems_points_nonneg;
ALTER TABLE score_events DROP CONSTRAINT IF EXISTS chk_score_events_points_nonneg;
DROP INDEX IF EXISTS idx_score_events_completion_uniq;
