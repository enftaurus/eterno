package tests

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/scoring"
	"github.com/eterno/backend/internal/submissions"
	"github.com/eterno/backend/internal/verification"
)

// Tests 22-44: Authoritative Verification, Time Boundaries, Multi-Challenge Matching, and Concurrency Idempotency

func createTestUserAndInst(t *testing.T, db *sql.DB) (string, string) {
	ctx := context.Background()
	var instID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO institutions (name, slug, timezone)
		VALUES ('Test College ' || gen_random_uuid()::text, 'inst-' || substr(gen_random_uuid()::text, 1, 8), 'UTC')
		RETURNING id
	`).Scan(&instID)
	if err != nil {
		t.Fatalf("failed to create institution: %v", err)
	}

	var userID string
	err = db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, display_name, role)
		VALUES ('user-' || gen_random_uuid()::text || '@college.edu', 'hash', 'Student Coder', 'STUDENT')
		RETURNING id
	`).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO institution_memberships (institution_id, user_id, role)
		VALUES ($1, $2, 'STUDENT')
	`, instID, userID)
	if err != nil {
		t.Fatalf("failed to create membership: %v", err)
	}

	return instID, userID
}

func createTestChallenge(t *testing.T, db *sql.DB, instID, status string, start, deadline time.Time, historicalAllowed bool) string {
	ctx := context.Background()
	var chID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO challenges (institution_id, title, description, status, start_time, deadline, historical_allowed)
		VALUES ($1, 'Challenge ' || gen_random_uuid()::text, 'Desc', $2, $3, $4, $5)
		RETURNING id
	`, instID, status, start, deadline, historicalAllowed).Scan(&chID)
	if err != nil {
		t.Fatalf("failed to create challenge: %v", err)
	}
	return chID
}

func addProblemToChallenge(t *testing.T, db *sql.DB, chID, slug string, points int) string {
	ctx := context.Background()
	var pID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO challenge_problems (challenge_id, platform, problem_url, problem_slug, title, difficulty, points)
		VALUES ($1, 'leetcode', 'https://leetcode.com/problems/' || $2, $2, 'Title', 'Medium', $3)
		RETURNING id
	`, chID, slug, points).Scan(&pID)
	if err != nil {
		t.Fatalf("failed to add problem: %v", err)
	}
	return pID
}

func assignStudentToChallenge(t *testing.T, db *sql.DB, chID, userID, instID string, baseline time.Time) string {
	ctx := context.Background()
	var aID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO challenge_assignments (challenge_id, user_id, institution_id, status, baseline_at)
		VALUES ($1, $2, $3, 'NOT_STARTED', $4)
		RETURNING id
	`, chID, userID, instID, baseline).Scan(&aID)
	if err != nil {
		t.Fatalf("failed to assign student: %v", err)
	}
	return aID
}

func TestVerificationTimeBoundaries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, userID := createTestUserAndInst(t, db)
	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	baseline := now
	deadline := now.Add(2 * time.Hour)

	chID := createTestChallenge(t, db, instID, models.ChallengeActive, start, deadline, false)
	slug := "test-slug-time-" + fmt.Sprintf("%d", time.Now().UnixNano())
	addProblemToChallenge(t, db, chID, slug, 100)
	assignStudentToChallenge(t, db, chID, userID, instID, baseline)

	subSvc := submissions.NewService(db)
	scoreSvc := scoring.NewService(db)
	verifSvc := verification.NewService(db, scoreSvc)

	// Case 1: Submission exactly at baseline -> Valid
	subDTOAtBaseline := models.SubmissionDTO{
		ProviderID:  "sub-at-base-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		ProblemSlug: slug,
		Title:       "Title",
		Verdict:     "Accepted",
		IsAccepted:  true,
		SubmittedAt: baseline,
	}
	sub1, _, err := subSvc.IngestSubmission(ctx, userID, models.PlatformLeetCode, subDTOAtBaseline)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	res1, err := verifSvc.VerifySubmission(ctx, sub1)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if res1.Status != "VERIFIED" {
		t.Errorf("expected submission exactly at baseline to be VERIFIED, got: %s (reason: %s)", res1.Status, res1.VerificationReason)
	}

	// Case 2: Submission before baseline (for another problem) -> Strictly Rejected
	slug2 := "test-slug-before-" + fmt.Sprintf("%d", time.Now().UnixNano())
	addProblemToChallenge(t, db, chID, slug2, 100)
	subDTOBefore := models.SubmissionDTO{
		ProviderID:  "sub-before-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		ProblemSlug: slug2,
		Title:       "Title",
		Verdict:     "Accepted",
		IsAccepted:  true,
		SubmittedAt: baseline.Add(-1 * time.Minute),
	}
	sub2, _, err := subSvc.IngestSubmission(ctx, userID, models.PlatformLeetCode, subDTOBefore)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	res2, err := verifSvc.VerifySubmission(ctx, sub2)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if res2.Status != "REJECTED_TIME" {
		t.Errorf("expected submission before baseline to be REJECTED_TIME, got: %s", res2.Status)
	}

	// Case 3: Submission after deadline -> Strictly Rejected
	pastStart := now.Add(-3 * time.Hour)
	pastDeadline := now.Add(-1 * time.Hour)
	pastBaseline := now.Add(-3 * time.Hour)
	chPastID := createTestChallenge(t, db, instID, models.ChallengeActive, pastStart, pastDeadline, false)
	assignStudentToChallenge(t, db, chPastID, userID, instID, pastBaseline)

	slug3 := "test-slug-after-" + fmt.Sprintf("%d", time.Now().UnixNano())
	addProblemToChallenge(t, db, chPastID, slug3, 100)
	subDTOAfter := models.SubmissionDTO{
		ProviderID:  "sub-after-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		ProblemSlug: slug3,
		Title:       "Title",
		Verdict:     "Accepted",
		IsAccepted:  true,
		SubmittedAt: pastDeadline.Add(10 * time.Minute), // 50 mins ago (after pastDeadline, but not in future)
	}
	sub3, _, err := subSvc.IngestSubmission(ctx, userID, models.PlatformLeetCode, subDTOAfter)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	res3, err := verifSvc.VerifySubmission(ctx, sub3)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if res3.Status != "REJECTED_TIME" {
		t.Errorf("expected submission after deadline to be REJECTED_TIME, got: %s (reason: %s)", res3.Status, res3.VerificationReason)
	}

	// Case 4: Future-dated submission (> 10 min clock skew) -> REJECTED_TIMESTAMP
	slug4 := "test-slug-future-" + fmt.Sprintf("%d", time.Now().UnixNano())
	addProblemToChallenge(t, db, chID, slug4, 100)
	subDTOFuture := models.SubmissionDTO{
		ProviderID:  "sub-future-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		ProblemSlug: slug4,
		Title:       "Title",
		Verdict:     "Accepted",
		IsAccepted:  true,
		SubmittedAt: now.Add(15 * time.Minute),
	}
	sub4, _, err := subSvc.IngestSubmission(ctx, userID, models.PlatformLeetCode, subDTOFuture)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	res4, err := verifSvc.VerifySubmission(ctx, sub4)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if res4.Status != "REJECTED_TIMESTAMP" {
		t.Errorf("expected future submission to be REJECTED_TIMESTAMP, got: %s (reason: %s)", res4.Status, res4.VerificationReason)
	}
}

func TestMultiChallengeVerification(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, userID := createTestUserAndInst(t, db)
	now := time.Now().UTC()
	slug := "multi-ch-slug-" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Challenge 1: Daily Sprint (Active)
	ch1 := createTestChallenge(t, db, instID, models.ChallengeActive, now.Add(-1*time.Hour), now.Add(2*time.Hour), false)
	addProblemToChallenge(t, db, ch1, slug, 100)
	assignStudentToChallenge(t, db, ch1, userID, instID, now.Add(-30*time.Minute))

	// Challenge 2: Weekly Contest (Also Active and contains the same problem)
	ch2 := createTestChallenge(t, db, instID, models.ChallengeActive, now.Add(-2*time.Hour), now.Add(24*time.Hour), false)
	addProblemToChallenge(t, db, ch2, slug, 150)
	assignStudentToChallenge(t, db, ch2, userID, instID, now.Add(-1*time.Hour))

	subSvc := submissions.NewService(db)
	scoreSvc := scoring.NewService(db)
	verifSvc := verification.NewService(db, scoreSvc)

	subDTO := models.SubmissionDTO{
		ProviderID:  "multi-sub-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		ProblemSlug: slug,
		Title:       "Two Sum",
		Verdict:     "Accepted",
		IsAccepted:  true,
		SubmittedAt: now.Add(5 * time.Minute),
	}
	sub, _, err := subSvc.IngestSubmission(ctx, userID, models.PlatformLeetCode, subDTO)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// Verify all matching challenges
	results, err := verifSvc.VerifySubmissionAll(ctx, sub)
	if err != nil {
		t.Fatalf("verify all failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 challenges evaluated, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != "VERIFIED" {
			t.Errorf("expected both challenges to be VERIFIED, got status %s for challenge %s", r.Status, r.ChallengeID)
		}
	}

	// Verify total points awarded in database: 100 + 150 = 250 points
	var totalPts int
	err = db.QueryRowContext(ctx, `SELECT COALESCE(SUM(points), 0) FROM score_events WHERE user_id = $1`, userID).Scan(&totalPts)
	if err != nil || totalPts != 250 {
		t.Errorf("expected 250 total points awarded across both challenges, got %d (err: %v)", totalPts, err)
	}
}

func TestCancelledChallengeDoesNotAwardPoints(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, userID := createTestUserAndInst(t, db)
	now := time.Now().UTC()
	slug := "cancel-slug-" + fmt.Sprintf("%d", time.Now().UnixNano())

	chID := createTestChallenge(t, db, instID, models.ChallengeCancelled, now.Add(-1*time.Hour), now.Add(2*time.Hour), false)
	addProblemToChallenge(t, db, chID, slug, 100)
	assignStudentToChallenge(t, db, chID, userID, instID, now.Add(-30*time.Minute))

	subSvc := submissions.NewService(db)
	scoreSvc := scoring.NewService(db)
	verifSvc := verification.NewService(db, scoreSvc)

	subDTO := models.SubmissionDTO{
		ProviderID:  "cancel-sub-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		ProblemSlug: slug,
		Title:       "Title",
		Verdict:     "Accepted",
		IsAccepted:  true,
		SubmittedAt: now,
	}
	sub, _, err := subSvc.IngestSubmission(ctx, userID, models.PlatformLeetCode, subDTO)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	res, err := verifSvc.VerifySubmission(ctx, sub)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if res.Status != "REJECTED_STATUS" {
		t.Errorf("expected REJECTED_STATUS for cancelled challenge, got: %s", res.Status)
	}

	var scoreCount int
	db.QueryRowContext(ctx, `SELECT count(*) FROM score_events WHERE user_id = $1`, userID).Scan(&scoreCount)
	if scoreCount != 0 {
		t.Errorf("cancelled challenge must never award score events, got %d", scoreCount)
	}
}

func TestConcurrentVerificationIdempotency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, userID := createTestUserAndInst(t, db)
	now := time.Now().UTC()
	slug := "concurrent-slug-" + fmt.Sprintf("%d", time.Now().UnixNano())

	chID := createTestChallenge(t, db, instID, models.ChallengeActive, now.Add(-1*time.Hour), now.Add(2*time.Hour), false)
	addProblemToChallenge(t, db, chID, slug, 100)
	assignStudentToChallenge(t, db, chID, userID, instID, now.Add(-30*time.Minute))

	subSvc := submissions.NewService(db)
	scoreSvc := scoring.NewService(db)
	verifSvc := verification.NewService(db, scoreSvc)

	subDTO := models.SubmissionDTO{
		ProviderID:  "concurrent-sub-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		ProblemSlug: slug,
		Title:       "Two Sum",
		Verdict:     "Accepted",
		IsAccepted:  true,
		SubmittedAt: now,
	}
	sub, _, err := subSvc.IngestSubmission(ctx, userID, models.PlatformLeetCode, subDTO)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// Launch 5 concurrent workers verifying the exact same submission
	var wg sync.WaitGroup
	var verifiedCount, alreadyVerifiedCount int
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := verifSvc.VerifySubmission(context.Background(), sub)
			if err != nil {
				return
			}
			mu.Lock()
			if res.Status == "VERIFIED" {
				verifiedCount++
			} else if res.Status == "ALREADY_VERIFIED" {
				alreadyVerifiedCount++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if verifiedCount != 1 {
		t.Errorf("expected exactly 1 VERIFIED result under concurrency, got %d", verifiedCount)
	}
	if alreadyVerifiedCount != 4 {
		t.Errorf("expected 4 ALREADY_VERIFIED results under concurrency, got %d", alreadyVerifiedCount)
	}

	// Verify database records in PostgreSQL: exactly 1 completion and 1 score event
	var completionsCount, scoreEventsCount int
	db.QueryRowContext(ctx, `SELECT count(*) FROM challenge_completions WHERE user_id = $1`, userID).Scan(&completionsCount)
	db.QueryRowContext(ctx, `SELECT count(*) FROM score_events WHERE user_id = $1`, userID).Scan(&scoreEventsCount)

	if completionsCount != 1 {
		t.Errorf("expected 1 completion in DB, got %d", completionsCount)
	}
	if scoreEventsCount != 1 {
		t.Errorf("expected 1 score event in DB, got %d", scoreEventsCount)
	}
}
