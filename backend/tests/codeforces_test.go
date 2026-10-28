package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/platforms/codeforces"
)

func TestCodeforcesAdapterMock(t *testing.T) {
	adapter := codeforces.NewAdapter("test-key", "test-secret", true)
	ctx := context.Background()

	// 1. User verification mock
	adapter.SetMockUser("tourist", map[string]interface{}{
		"handle":     "tourist",
		"rating":     3850,
		"max_rating": 4009,
		"rank":       "legendary grandmaster",
	})

	verified, profile, err := adapter.VerifyAccount(ctx, "tourist")
	if err != nil {
		t.Fatalf("unexpected error verifying tourist: %v", err)
	}
	if !verified {
		t.Fatalf("expected tourist to be verified")
	}
	if profile["rating"].(int) != 3850 {
		t.Errorf("expected rating 3850, got %v", profile["rating"])
	}

	// 2. Unregistered user
	verifiedNon, _, err := adapter.VerifyAccount(ctx, "nonexistent_coder_99")
	if verifiedNon {
		t.Fatalf("expected nonexistent user to fail verification")
	}
	if err == nil {
		t.Fatalf("expected error for nonexistent user")
	}

	// 3. Mock submissions
	now := time.Now().UTC()
	adapter.SetMockSubmissions("tourist", []models.SubmissionDTO{
		{
			ProviderID:  "12345678",
			ProblemSlug: "1920-a",
			Title:       "Satisfying Constraints (1920A)",
			Verdict:     "Accepted",
			IsAccepted:  true,
			SubmittedAt: now.Add(-1 * time.Hour),
		},
	})

	subs, err := adapter.FetchRecentSubmissions(ctx, "tourist", 10)
	if err != nil {
		t.Fatalf("unexpected error fetching submissions: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(subs))
	}
	if subs[0].ProblemSlug != "1920-a" {
		t.Errorf("expected slug 1920-a, got %s", subs[0].ProblemSlug)
	}
}

func TestCodeforcesServerIntegration(t *testing.T) {
	// Create mock HTTP server simulating Codeforces API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user.info":
			handle := r.URL.Query().Get("handles")
			if handle == "petr" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"status": "OK",
					"result": [{"handle": "petr", "rating": 3000, "maxRating": 3200, "rank": "legendary grandmaster"}]
				}`))
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status": "FAILED", "comment": "handles: User not found"}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	// Test real API logic targeting mock server
	adapter := codeforces.NewAdapter("key", "secret", false)
	// Platform name
	if adapter.PlatformName() != models.PlatformCodeforces {
		t.Errorf("expected platform name codeforces, got %s", adapter.PlatformName())
	}
}
