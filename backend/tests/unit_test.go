package tests

import (
	"testing"
	"time"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/auth"
	"github.com/eterno/backend/internal/config"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/platforms/leetcode"
	"github.com/eterno/backend/internal/scoring"
)

func TestLeetCodeSlugParsing(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		shouldError bool
	}{
		{"two-sum", "two-sum", false},
		{"https://leetcode.com/problems/two-sum/", "two-sum", false},
		{"https://leetcode.com/problems/trapping-rain-water/description/", "trapping-rain-water", false},
		{"https://leetcode.com/problems/3sum/", "3sum", false},
		{"https://leetcode.com/problems/longest-substring-without-repeating-characters", "longest-substring-without-repeating-characters", false},
		{"http://leetcode.com/problems/median-of-two-sorted-arrays/", "median-of-two-sorted-arrays", false},
		{"https://hackerrank.com/challenges/solve-me-first", "", true},
		{"", "", true},
		{"two_sum_invalid", "", true},
		{"---invalid---", "", true},
	}

	for _, tt := range tests {
		slug, err := leetcode.ParseProblemSlug(tt.input)
		if tt.shouldError {
			if err == nil {
				t.Errorf("expected error for input %q, got slug %q", tt.input, slug)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
			}
			if slug != tt.expected {
				t.Errorf("for input %q, expected %q, got %q", tt.input, tt.expected, slug)
			}
		}
	}
}

func TestLeetCodeVerdictAcceptance(t *testing.T) {
	if !leetcode.IsAccepted("Accepted") {
		t.Error("expected 'Accepted' to be accepted")
	}
	if !leetcode.IsAccepted("ac") {
		t.Error("expected 'ac' to be accepted")
	}
	if !leetcode.IsAccepted("ACCEPTED") {
		t.Error("expected 'ACCEPTED' to be accepted")
	}
	if leetcode.IsAccepted("Wrong Answer") {
		t.Error("expected 'Wrong Answer' to NOT be accepted")
	}
	if leetcode.IsAccepted("Time Limit Exceeded") {
		t.Error("expected 'Time Limit Exceeded' to NOT be accepted")
	}
	if leetcode.IsAccepted("Memory Limit Exceeded") {
		t.Error("expected 'Memory Limit Exceeded' to NOT be accepted")
	}
}

func TestDeterministicLeaderboardTieBreaking(t *testing.T) {
	t1 := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 10, 27, 11, 0, 0, 0, time.UTC)

	// User A has more points
	a := models.LeaderboardEntry{UserID: "a", TotalPoints: 200, VerifiedCompletions: 2, LastVerifiedSolveAt: &t2}
	b := models.LeaderboardEntry{UserID: "b", TotalPoints: 100, VerifiedCompletions: 3, LastVerifiedSolveAt: &t1}
	if !scoring.CompareLeaderboardEntries(a, b) {
		t.Error("User with higher points should rank higher")
	}

	// Tied points, User C has more verified completions
	c := models.LeaderboardEntry{UserID: "c", TotalPoints: 200, VerifiedCompletions: 3, LastVerifiedSolveAt: &t2}
	if !scoring.CompareLeaderboardEntries(c, a) {
		t.Error("User with tied points but more verified completions should rank higher")
	}

	// Tied points and completions, User D solved earlier
	d := models.LeaderboardEntry{UserID: "d", TotalPoints: 200, VerifiedCompletions: 3, LastVerifiedSolveAt: &t1}
	if !scoring.CompareLeaderboardEntries(d, c) {
		t.Error("User with earlier solve timestamp should win tie-break")
	}
}

func TestActivityIntensityCalculation(t *testing.T) {
	cases := []struct {
		count     int
		intensity int
	}{
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 2},
		{5, 2},
		{6, 3},
		{9, 3},
		{10, 4},
		{45, 4},
	}

	for _, c := range cases {
		got := activity.CalculateIntensity(c.count)
		if got != c.intensity {
			t.Errorf("count %d: expected intensity %d, got %d", c.count, c.intensity, got)
		}
	}
}

func TestPasswordHashingAndAuthTokens(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:          "test-jwt-secret-key-32-bytes-long!",
		JWTExpirationHours: 24,
	}

	password := "SecretP@ssword2026"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !auth.CheckPassword(hash, password) {
		t.Error("expected password to match hash")
	}
	if auth.CheckPassword(hash, "WrongPassword") {
		t.Error("expected wrong password to fail check")
	}

	user := &models.User{
		ID:    "user-uuid-1234",
		Email: "student@college.edu",
		Role:  models.RoleStudent,
	}
	token, err := auth.GenerateToken(cfg, user, "inst-uuid-5678")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := auth.ValidateToken(cfg, token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != user.ID || claims.Role != models.RoleStudent || claims.InstitutionID != "inst-uuid-5678" {
		t.Errorf("token claims mismatch: %+v", claims)
	}
}
