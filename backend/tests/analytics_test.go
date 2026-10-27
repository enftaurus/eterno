package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/analytics"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/scoring"
)

// Tests 91-110: Analytics Robustness, Empty States, Streak State Machine, and Leaderboard Tie-Breaking

func TestEmptyInstitutionMetricsSafeZero(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create empty institution with zero students and zero challenges
	var instID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO institutions (name, slug, timezone)
		VALUES ('Empty University ' || gen_random_uuid()::text, 'empty-' || substr(gen_random_uuid()::text, 1, 8), 'UTC')
		RETURNING id
	`).Scan(&instID)
	if err != nil {
		t.Fatalf("failed to create institution: %v", err)
	}

	analyticsSvc := analytics.NewService(db)
	overview, err := analyticsSvc.GetAdminOverview(ctx, instID)
	if err != nil {
		t.Fatalf("GetAdminOverview failed on empty institution: %v", err)
	}

	if overview.TotalRegisteredStudents != 0 {
		t.Errorf("expected 0 total registered students, got %d", overview.TotalRegisteredStudents)
	}
	if overview.SolvedProblemTodayCount != 0 {
		t.Errorf("expected 0 solved today, got %d", overview.SolvedProblemTodayCount)
	}
	if overview.CompletedAssignedTodayCount != 0 {
		t.Errorf("expected 0 completed assigned today, got %d", overview.CompletedAssignedTodayCount)
	}
	if overview.InactiveStudentsCount != 0 {
		t.Errorf("expected 0 inactive students, got %d", overview.InactiveStudentsCount)
	}
	if overview.ActiveChallengesCount != 0 {
		t.Errorf("expected 0 active challenges, got %d", overview.ActiveChallengesCount)
	}
	if overview.ApproachingDeadlinesCount != 0 {
		t.Errorf("expected 0 approaching deadlines, got %d", overview.ApproachingDeadlinesCount)
	}
	if overview.VerifiedSolvesThisWeek != 0 {
		t.Errorf("expected 0 solves this week, got %d", overview.VerifiedSolvesThisWeek)
	}
	if overview.VerifiedSolvesThisMonth != 0 {
		t.Errorf("expected 0 solves this month, got %d", overview.VerifiedSolvesThisMonth)
	}
	if overview.ChallengeCompletionRate != 0.0 {
		t.Errorf("expected 0.0 completion rate, got %f", overview.ChallengeCompletionRate)
	}
	if overview.AvgProblemsPerStudent != 0.0 {
		t.Errorf("expected 0.0 avg problems, got %f", overview.AvgProblemsPerStudent)
	}
	if len(overview.RecentSubmissions) != 0 {
		t.Errorf("expected empty recent submissions, got %d", len(overview.RecentSubmissions))
	}
}

func TestEmptyStudentStreaksSafeZero(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, userID := createTestUserAndInst(t, db)
	analyticsSvc := analytics.NewService(db)

	streaks, err := analyticsSvc.GetUserStreakAnalytics(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserStreakAnalytics failed on new student: %v", err)
	}

	if streaks.CurrentStreak != 0 {
		t.Errorf("expected current streak 0, got %d", streaks.CurrentStreak)
	}
	if streaks.LongestStreak != 0 {
		t.Errorf("expected longest streak 0, got %d", streaks.LongestStreak)
	}
	if streaks.ActiveDaysLast7 != 0 || streaks.ActiveDaysLast30 != 0 || streaks.ActiveDaysLast90 != 0 {
		t.Errorf("expected 0 active days, got 7d=%d, 30d=%d, 90d=%d",
			streaks.ActiveDaysLast7, streaks.ActiveDaysLast30, streaks.ActiveDaysLast90)
	}
	if streaks.WeeklyParticipationRate != 0.0 {
		t.Errorf("expected 0.0 weekly participation rate, got %f", streaks.WeeklyParticipationRate)
	}
	if streaks.ChallengeCompletionRate != 0.0 {
		t.Errorf("expected 0.0 challenge completion rate, got %f", streaks.ChallengeCompletionRate)
	}
	if streaks.AvgVerifiedPerDay != 0.0 {
		t.Errorf("expected 0.0 avg verified per day, got %f", streaks.AvgVerifiedPerDay)
	}
}

func TestStudentStreakCalculationAndBreak(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, userID := createTestUserAndInst(t, db)
	actSvc := activity.NewService(db)
	analyticsSvc := analytics.NewService(db)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)

	// Step 1: Record 4 consecutive active days (today, yesterday, 2d ago, 3d ago)
	for d := 0; d < 4; d++ {
		eventTime := today.AddDate(0, 0, -d)
		extID := fmt.Sprintf("event-streak-%s-%d", userID, d)
		err := actSvc.RecordEvent(ctx, userID, models.PlatformLeetCode, extID, "PROBLEM_SOLVED", eventTime, nil)
		if err != nil {
			t.Fatalf("failed to record activity event: %v", err)
		}
	}

	streaks, err := analyticsSvc.GetUserStreakAnalytics(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get streaks: %v", err)
	}
	if streaks.CurrentStreak != 4 {
		t.Errorf("expected current streak 4, got %d", streaks.CurrentStreak)
	}
	if streaks.LongestStreak != 4 {
		t.Errorf("expected longest streak 4, got %d", streaks.LongestStreak)
	}

	// Step 2: Add an older historical streak of 7 days ending 20 days ago (gap between 3d ago and 20d ago)
	for d := 20; d < 27; d++ {
		eventTime := today.AddDate(0, 0, -d)
		extID := fmt.Sprintf("event-hist-%s-%d", userID, d)
		err := actSvc.RecordEvent(ctx, userID, models.PlatformLeetCode, extID, "PROBLEM_SOLVED", eventTime, nil)
		if err != nil {
			t.Fatalf("failed to record historical event: %v", err)
		}
	}

	streaksAfterHist, err := analyticsSvc.GetUserStreakAnalytics(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get streaks: %v", err)
	}
	// Current streak is still 4
	if streaksAfterHist.CurrentStreak != 4 {
		t.Errorf("expected current streak to remain 4, got %d", streaksAfterHist.CurrentStreak)
	}
	// Longest streak is now 7 from historical block
	if streaksAfterHist.LongestStreak != 7 {
		t.Errorf("expected longest streak to be 7, got %d", streaksAfterHist.LongestStreak)
	}

	// Step 3: Test a student whose last solve was 3 days ago (gap breaks current streak)
	_, userBrokenID := createTestUserAndInst(t, db)
	// Solved 3 days ago and 4 days ago
	for d := 3; d <= 4; d++ {
		eventTime := today.AddDate(0, 0, -d)
		extID := fmt.Sprintf("event-broken-%s-%d", userBrokenID, d)
		_ = actSvc.RecordEvent(ctx, userBrokenID, models.PlatformLeetCode, extID, "PROBLEM_SOLVED", eventTime, nil)
	}

	brokenStreaks, err := analyticsSvc.GetUserStreakAnalytics(ctx, userBrokenID)
	if err != nil {
		t.Fatalf("failed to get broken streaks: %v", err)
	}
	if brokenStreaks.CurrentStreak != 0 {
		t.Errorf("expected current streak 0 due to 2-day gap, got %d", brokenStreaks.CurrentStreak)
	}
	if brokenStreaks.LongestStreak != 2 {
		t.Errorf("expected longest streak 2, got %d", brokenStreaks.LongestStreak)
	}
}

func TestDeterministicLeaderboardTieBreakingMultiCriteria(t *testing.T) {
	now := time.Now().UTC()
	tEarly := now.Add(-2 * time.Hour)
	tLate := now.Add(-1 * time.Hour)

	// Case 1: Point superiority dominates
	highPoints := models.LeaderboardEntry{UserID: "u1", TotalPoints: 500, VerifiedCompletions: 1}
	lowPoints := models.LeaderboardEntry{UserID: "u2", TotalPoints: 400, VerifiedCompletions: 10}
	if !scoring.CompareLeaderboardEntries(highPoints, lowPoints) {
		t.Errorf("expected highPoints to rank above lowPoints")
	}

	// Case 2: Points tied -> more verified completions wins
	moreCompletions := models.LeaderboardEntry{UserID: "u1", TotalPoints: 300, VerifiedCompletions: 5}
	fewerCompletions := models.LeaderboardEntry{UserID: "u2", TotalPoints: 300, VerifiedCompletions: 3}
	if !scoring.CompareLeaderboardEntries(moreCompletions, fewerCompletions) {
		t.Errorf("expected moreCompletions to rank above fewerCompletions on tied points")
	}
	if scoring.CompareLeaderboardEntries(fewerCompletions, moreCompletions) {
		t.Errorf("fewerCompletions should not rank above moreCompletions")
	}

	// Case 3: Points tied & completions tied -> earlier solve time wins
	earlySolve := models.LeaderboardEntry{UserID: "u1", TotalPoints: 300, VerifiedCompletions: 5, LastVerifiedSolveAt: &tEarly}
	lateSolve := models.LeaderboardEntry{UserID: "u2", TotalPoints: 300, VerifiedCompletions: 5, LastVerifiedSolveAt: &tLate}
	if !scoring.CompareLeaderboardEntries(earlySolve, lateSolve) {
		t.Errorf("expected earlySolve to rank above lateSolve")
	}
	if scoring.CompareLeaderboardEntries(lateSolve, earlySolve) {
		t.Errorf("lateSolve should not rank above earlySolve")
	}

	// Case 4: Points tied, completions tied, solve times identical -> deterministic UserID comparison
	userA := models.LeaderboardEntry{UserID: "user-alpha", TotalPoints: 300, VerifiedCompletions: 5, LastVerifiedSolveAt: &tEarly}
	userB := models.LeaderboardEntry{UserID: "user-beta", TotalPoints: 300, VerifiedCompletions: 5, LastVerifiedSolveAt: &tEarly}
	if !scoring.CompareLeaderboardEntries(userA, userB) {
		t.Errorf("expected user-alpha to tie-break ahead of user-beta")
	}
}

func TestActivityIntensityScale(t *testing.T) {
	testCases := []struct {
		count     int
		intensity int
	}{
		{-10, 0},
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 2},
		{4, 2},
		{5, 2},
		{6, 3},
		{7, 3},
		{8, 3},
		{9, 3},
		{10, 4},
		{25, 4},
		{500, 4},
	}

	for _, tc := range testCases {
		got := activity.CalculateIntensity(tc.count)
		if got != tc.intensity {
			t.Errorf("CalculateIntensity(%d) = %d; expected %d", tc.count, got, tc.intensity)
		}
	}
}

func TestHeatmapFilteringByYearAndSource(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	_, userID := createTestUserAndInst(t, db)
	actSvc := activity.NewService(db)

	// Insert events in 2026
	time2026 := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	_ = actSvc.RecordEvent(ctx, userID, models.PlatformLeetCode, fmt.Sprintf("lc-2026-%s", userID), "SOLVE", time2026, nil)
	_ = actSvc.RecordEvent(ctx, userID, models.PlatformGitHub, fmt.Sprintf("gh-2026-%s", userID), "COMMIT", time2026, nil)

	// Insert event in 2025
	time2025 := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	_ = actSvc.RecordEvent(ctx, userID, models.PlatformLeetCode, fmt.Sprintf("lc-2025-%s", userID), "SOLVE", time2025, nil)

	// Query 2026 (all sources)
	res2026, err := actSvc.GetUserHeatmap(ctx, userID, 2026, "")
	if err != nil {
		t.Fatalf("GetUserHeatmap 2026 failed: %v", err)
	}
	if res2026.TotalEvents != 2 {
		t.Errorf("expected 2 events in 2026, got %d", res2026.TotalEvents)
	}

	// Query 2026 filtered by source 'leetcode'
	resLC, err := actSvc.GetUserHeatmap(ctx, userID, 2026, models.PlatformLeetCode)
	if err != nil {
		t.Fatalf("GetUserHeatmap LC failed: %v", err)
	}
	if resLC.TotalEvents != 1 {
		t.Errorf("expected 1 LeetCode event in 2026, got %d", resLC.TotalEvents)
	}

	// Query 2025 (all sources)
	res2025, err := actSvc.GetUserHeatmap(ctx, userID, 2025, "")
	if err != nil {
		t.Fatalf("GetUserHeatmap 2025 failed: %v", err)
	}
	if res2025.TotalEvents != 1 {
		t.Errorf("expected 1 event in 2025, got %d", res2025.TotalEvents)
	}
}
