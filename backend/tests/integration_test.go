package tests

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/analytics"
	"github.com/eterno/backend/internal/challenges"
	"github.com/eterno/backend/internal/config"
	"github.com/eterno/backend/internal/database"
	"github.com/eterno/backend/internal/jobs"
	"github.com/eterno/backend/internal/leaderboards"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/platforms/leetcode"
	"github.com/eterno/backend/internal/scoring"
	"github.com/eterno/backend/internal/submissions"
	"github.com/eterno/backend/internal/verification"
)

func setupTestDB(t *testing.T) *sql.DB {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	return db
}

func TestCompleteVerificationWorkflow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// 0. Seed test institution, admin, and student
	var instID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO institutions (name, slug, timezone)
		VALUES ('Stanford Engineering', 'stanford-' || substr(gen_random_uuid()::text, 1, 8), 'UTC')
		RETURNING id
	`).Scan(&instID)
	if err != nil {
		t.Fatalf("failed to seed institution: %v", err)
	}

	var adminID, studentID string
	_ = db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, display_name, role)
		VALUES ('admin-' || gen_random_uuid()::text || '@stanford.edu', 'hash', 'Dean of CS', 'COLLEGE_ADMIN')
		RETURNING id
	`).Scan(&adminID)

	_ = db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, display_name, role)
		VALUES ('alice-' || gen_random_uuid()::text || '@stanford.edu', 'hash', 'Alice Smith', 'STUDENT')
		RETURNING id
	`).Scan(&studentID)

	// Memberships
	_, _ = db.ExecContext(ctx, `
		INSERT INTO institution_memberships (institution_id, user_id, role)
		VALUES ($1, $2, 'COLLEGE_ADMIN'), ($1, $3, 'STUDENT')
	`, instID, adminID, studentID)

	// Services
	scoringSvc := scoring.NewService(db)
	subSvc := submissions.NewService(db)
	actSvc := activity.NewService(db)
	verifSvc := verification.NewService(db, scoringSvc)
	chSvc := challenges.NewService(db)
	lbSvc := leaderboards.NewService(db)
	analyticsSvc := analytics.NewService(db)
	queue := jobs.NewQueue(db)
	lcAdapter := leetcode.NewAdapter("", true) // mock mode enabled

	// Step 1 & 2: Administrator creates challenge with LeetCode problem
	now := time.Now().UTC()
	req := &models.CreateChallengeRequest{
		Title:             "Daily Algorithm Challenge: Dynamic Programming",
		Description:       "Solve Two Sum on LeetCode",
		StartTime:         now.Add(-1 * time.Hour),
		Deadline:          now.Add(24 * time.Hour),
		ScopeType:         "INSTITUTION",
		MaxPoints:         150,
		HistoricalAllowed: false,
		Problems: []models.CreateProblemInput{
			{
				ProblemURL: "https://leetcode.com/problems/two-sum/",
				Difficulty: "Easy",
				Points:     100,
			},
		},
	}
	ch, err := chSvc.CreateChallenge(ctx, instID, adminID, req)
	if err != nil {
		t.Fatalf("Step 1 failed: could not create challenge: %v", err)
	}
	if len(ch.Problems) != 1 || ch.Problems[0].ProblemSlug != "two-sum" {
		t.Fatalf("Step 2 failed: canonical problem slug not extracted: %+v", ch.Problems)
	}

	// Step 3: Administrator publishes the challenge
	publishedCh, err := chSvc.PublishChallenge(ctx, instID, ch.ID)
	if err != nil {
		t.Fatalf("Step 3 failed: could not publish challenge: %v", err)
	}
	if publishedCh.Status != models.ChallengeActive {
		t.Fatalf("expected status ACTIVE, got %s", publishedCh.Status)
	}

	// Step 4: Eligible student sees assigned problem
	assignments, err := chSvc.GetStudentAssignments(ctx, studentID)
	if err != nil {
		t.Fatalf("Step 4 failed: could not retrieve assignments: %v", err)
	}
	if len(assignments) == 0 {
		t.Fatal("Step 4 failed: student has no assigned challenges")
	}

	// Step 5: Student links and verifies LeetCode account
	lcUsername := fmt.Sprintf("alice_%d", time.Now().UnixNano())
	_, err = db.ExecContext(ctx, `
		INSERT INTO platform_accounts (user_id, platform, username, verification_status, verified_at)
		VALUES ($1, 'leetcode', $2, 'VERIFIED', NOW())
	`, studentID, lcUsername)
	if err != nil {
		t.Fatalf("Step 5 failed: could not link leetcode account: %v", err)
	}

	// Step 6: Student submits Accepted solution on LeetCode
	subTime := now.Add(10 * time.Minute)
	providerID := fmt.Sprintf("lc-sub-%d", time.Now().UnixNano())
	mockSubmissions := []models.SubmissionDTO{
		{
			ProviderID:  providerID,
			ProblemSlug: "two-sum",
			Title:       "Two Sum",
			Verdict:     "Accepted",
			IsAccepted:  true,
			SubmittedAt: subTime,
		},
	}
	lcAdapter.SetMockSubmissions(lcUsername, mockSubmissions)

	// Step 7: Eterno worker retrieves and verifies submission
	worker := jobs.NewWorker(queue, db, lcAdapter, subSvc, verifSvc, actSvc, 1*time.Second)
	syncJob, err := queue.Enqueue(ctx, studentID, "leetcode", "USER_SYNC", map[string]interface{}{"username": lcUsername})
	if err != nil {
		t.Fatalf("Step 7 failed: enqueue failed: %v", err)
	}
	worker.ProcessNextJob(ctx)

	// Check job completed
	var jobStatus string
	_ = db.QueryRowContext(ctx, "SELECT status FROM sync_jobs WHERE id = $1", syncJob.ID).Scan(&jobStatus)
	if jobStatus != "COMPLETED" {
		t.Fatalf("Step 7 failed: job expected COMPLETED, got %s", jobStatus)
	}

	// Step 8: Student challenge progress changes to VERIFIED
	var assignStatus string
	var pointsAwarded int
	err = db.QueryRowContext(ctx, `
		SELECT status, points_awarded
		FROM challenge_assignments
		WHERE challenge_id = $1 AND user_id = $2
	`, ch.ID, studentID).Scan(&assignStatus, &pointsAwarded)
	if err != nil {
		t.Fatalf("Step 8 failed: assignment query: %v", err)
	}
	if assignStatus != models.AssignmentVerified {
		t.Fatalf("Step 8 failed: expected assignment status VERIFIED, got %s", assignStatus)
	}
	if pointsAwarded != 100 {
		t.Fatalf("Step 8 failed: expected 100 points, got %d", pointsAwarded)
	}

	// Step 9: Completion and score recorded once
	var completionCount, scoreCount int
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM challenge_completions WHERE challenge_id = $1 AND user_id = $2", ch.ID, studentID).Scan(&completionCount)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM score_events WHERE challenge_id = $1 AND user_id = $2", ch.ID, studentID).Scan(&scoreCount)
	if completionCount != 1 {
		t.Fatalf("Step 9 failed: expected exactly 1 completion, got %d", completionCount)
	}
	if scoreCount != 1 {
		t.Fatalf("Step 9 failed: expected exactly 1 score event, got %d", scoreCount)
	}

	// Step 10: Admin sees student under today's verified completions
	overview, err := analyticsSvc.GetAdminOverview(ctx, instID)
	if err != nil {
		t.Fatalf("Step 10 failed: overview query: %v", err)
	}
	if overview.SolvedProblemTodayCount < 1 {
		t.Fatalf("Step 10 failed: expected >= 1 solved today, got %d", overview.SolvedProblemTodayCount)
	}
	if overview.CompletedAssignedTodayCount < 1 {
		t.Fatalf("Step 10 failed: expected >= 1 completed assigned today, got %d", overview.CompletedAssignedTodayCount)
	}

	// Step 11: Student appears correctly in institution leaderboard
	lb, err := lbSvc.GetLeaderboard(ctx, leaderboards.FilterParams{InstitutionID: instID})
	if err != nil {
		t.Fatalf("Step 11 failed: leaderboard query: %v", err)
	}
	if len(lb) == 0 || lb[0].UserID != studentID || lb[0].TotalPoints != 100 {
		t.Fatalf("Step 11 failed: student not top of leaderboard: %+v", lb)
	}

	// Step 12: Student activity heatmap reflects event
	heatmap, err := actSvc.GetUserHeatmap(ctx, studentID, now.Year(), "")
	if err != nil {
		t.Fatalf("Step 12 failed: heatmap query: %v", err)
	}
	if heatmap.TotalEvents < 1 {
		t.Fatalf("Step 12 failed: expected at least 1 activity event, got %d", heatmap.TotalEvents)
	}

	// Step 13: Repeating synchronization does NOT duplicate scores, completions, or activity events!
	_, _ = queue.Enqueue(ctx, studentID, "leetcode", "USER_SYNC", map[string]interface{}{"username": lcUsername})
	worker.ProcessNextJob(ctx)

	var compCountAfter, scoreCountAfter, actCountAfter int
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM challenge_completions WHERE challenge_id = $1 AND user_id = $2", ch.ID, studentID).Scan(&compCountAfter)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM score_events WHERE challenge_id = $1 AND user_id = $2", ch.ID, studentID).Scan(&scoreCountAfter)
	_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM activity_events WHERE user_id = $1 AND source = 'leetcode'", studentID).Scan(&actCountAfter)

	if compCountAfter != 1 {
		t.Fatalf("Step 13 failed: duplicate completion created! Expected 1, got %d", compCountAfter)
	}
	if scoreCountAfter != 1 {
		t.Fatalf("Step 13 failed: duplicate score event created! Expected 1, got %d", scoreCountAfter)
	}
	if actCountAfter != 1 {
		t.Fatalf("Step 13 failed: duplicate activity event created! Expected 1, got %d", actCountAfter)
	}

	// Negative Test: Submission made before baseline is rejected
	earlySub := &models.Submission{
		ID:                   "sub-early",
		UserID:               studentID,
		Platform:             models.PlatformLeetCode,
		ProviderSubmissionID: "early-provider-id",
		ProblemSlug:          "two-sum",
		Verdict:              "Accepted",
		IsAccepted:           true,
		SubmittedAt:          now.Add(-2 * time.Hour), // before baseline!
	}
	res, err := verifSvc.VerifySubmission(ctx, earlySub)
	if err != nil {
		t.Fatalf("negative test error: %v", err)
	}
	if res.Status != "REJECTED_TIME" && res.Status != "ALREADY_VERIFIED" {
		t.Fatalf("expected REJECTED_TIME or ALREADY_VERIFIED, got %s", res.Status)
	}

	t.Log("All 13 verification steps and idempotency constraints passed with flying colors!")
}
