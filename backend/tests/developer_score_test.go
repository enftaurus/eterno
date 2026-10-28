package tests

import (
	"context"
	"testing"
	"time"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/scoring"
	"github.com/eterno/backend/internal/submissions"
)

func TestLogNormalizeProperties(t *testing.T) {
	// 1. Boundary cases
	if val := scoring.LogNormalize(0, 100); val != 0.0 {
		t.Errorf("expected 0 for x=0, got %f", val)
	}
	if val := scoring.LogNormalize(-5, 100); val != 0.0 {
		t.Errorf("expected 0 for negative x, got %f", val)
	}
	if val := scoring.LogNormalize(100, 100); val != 100.0 {
		t.Errorf("expected 100 for x=K, got %f", val)
	}
	if val := scoring.LogNormalize(500, 100); val != 100.0 {
		t.Errorf("expected 100 for x > K, got %f", val)
	}

	// 2. Monotonicity: LogNormalize should be non-decreasing
	prev := 0.0
	for x := 0.0; x <= 200.0; x += 5.0 {
		curr := scoring.LogNormalize(x, 100.0)
		if curr < prev {
			t.Errorf("monotonicity violated at x=%f: curr %f < prev %f", x, curr, prev)
		}
		if curr > 100.0 || curr < 0.0 {
			t.Errorf("score bounds [0, 100] violated at x=%f: got %f", x, curr)
		}
		prev = curr
	}
}

func TestDeveloperScoreCalculation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, userID := createTestUserAndInst(t, db)

	scoringSvc := scoring.NewService(db)
	subSvc := submissions.NewService(db)
	actSvc := activity.NewService(db)

	// 1. Zero activity test
	res, err := scoringSvc.CalculateDeveloperScore(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error on zero activity: %v", err)
	}
	if res.TotalScore < 0 || res.TotalScore > 100 {
		t.Errorf("expected score between 0 and 100, got %d", res.TotalScore)
	}
	if len(res.Recommendations) == 0 {
		t.Errorf("expected recommendations for inactive user, got 0")
	}

	// 2. Add LeetCode solves
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		slug := "problem-" + time.Now().Format("150405.000000")
		_, _, err := subSvc.IngestSubmission(ctx, userID, models.PlatformLeetCode, models.SubmissionDTO{
			ProviderID:  "lc-sub-" + slug,
			ProblemSlug: slug,
			Title:       "Problem " + slug,
			Verdict:     "Accepted",
			IsAccepted:  true,
			SubmittedAt: now.Add(-time.Duration(i*24) * time.Hour),
		})
		if err != nil {
			t.Fatalf("failed ingesting submission: %v", err)
		}

		_ = actSvc.RecordEvent(ctx, userID, "leetcode", "lc-sub-"+slug, "accepted_submission", now.Add(-time.Duration(i*24)*time.Hour), map[string]interface{}{
			"slug": slug,
		})
	}

	// 3. Add GitHub commits
	for i := 0; i < 8; i++ {
		sha := "sha-" + time.Now().Format("150405.000000") + string(rune('a'+i))
		_ = actSvc.RecordEvent(ctx, userID, "github", sha, "commit", now.Add(-time.Duration(i*12)*time.Hour), map[string]interface{}{
			"repo": "eterno/code-crusaders",
			"sha":  sha,
		})
	}

	// 4. Recalculate score
	resUpdated, err := scoringSvc.CalculateDeveloperScore(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error calculating score: %v", err)
	}

	if resUpdated.TotalScore <= res.TotalScore {
		t.Errorf("expected updated score (%d) to be greater than initial score (%d)", resUpdated.TotalScore, res.TotalScore)
	}
	if resUpdated.ProblemSolving.Score <= 0 {
		t.Errorf("expected positive problem solving score, got %f", resUpdated.ProblemSolving.Score)
	}
	if resUpdated.Engineering.Score <= 0 {
		t.Errorf("expected positive engineering score, got %f", resUpdated.Engineering.Score)
	}
	if len(resUpdated.SkillRadar) == 0 {
		t.Errorf("expected non-empty skill radar")
	}
}
