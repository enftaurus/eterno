package tests

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eterno/backend/internal/platforms/leetcode"
)

// Tests 1-21: External Provider Edge Cases, HTTP/GraphQL handling, Account Verification, URL Parsing

func TestProviderSuccessfulAcceptedRecord(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"recentAcSubmissionList": []map[string]interface{}{
					{
						"id":        "sub-1001",
						"title":     "Two Sum",
						"titleSlug": "two-sum",
						"timestamp": "1700000000",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	subs, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(subs))
	}
	if subs[0].ProviderID != "sub-1001" || subs[0].ProblemSlug != "two-sum" || !subs[0].IsAccepted {
		t.Errorf("unexpected submission DTO: %+v", subs[0])
	}
	if subs[0].SubmittedAt.Unix() != 1700000000 {
		t.Errorf("timestamp mismatch: expected 1700000000, got %d", subs[0].SubmittedAt.Unix())
	}
}

func TestProviderMissingSubmissionID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"recentAcSubmissionList": []map[string]interface{}{
					{
						"id":        nil, // Missing ID
						"title":     "Two Sum",
						"titleSlug": "two-sum",
						"timestamp": "1700000000",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	subs, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 submissions due to missing ID, got %d", len(subs))
	}
}

func TestProviderMissingProblemSlug(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"recentAcSubmissionList": []map[string]interface{}{
					{
						"id":        "sub-1002",
						"title":     "Two Sum",
						"titleSlug": "", // Empty slug
						"timestamp": "1700000000",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	subs, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 submissions due to missing slug, got %d", len(subs))
	}
}

func TestProviderMissingAndZeroTimestamp(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"recentAcSubmissionList": []map[string]interface{}{
					{
						"id":        "sub-1003",
						"titleSlug": "two-sum",
						"timestamp": "0", // Zero timestamp
					},
					{
						"id":        "sub-1004",
						"titleSlug": "two-sum",
						// Missing timestamp entirely
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	subs, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 submissions because zero/missing timestamps must never be fabricated, got %d", len(subs))
	}
}

func TestProviderFutureDatedTimestamp(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		futureSec := time.Now().Add(24 * time.Hour).Unix()
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"recentAcSubmissionList": []map[string]interface{}{
					{
						"id":        "sub-1005",
						"titleSlug": "two-sum",
						"timestamp": futureSec,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	subs, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected future-dated submission to be rejected, got %d", len(subs))
	}
}

func TestProviderEmptySubmissionList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"recentAcSubmissionList": []interface{}{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	subs, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 submissions, got %d", len(subs))
	}
}

func TestProviderInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{ invalid json... `))
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	_, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if err == nil {
		t.Fatal("expected error on malformed JSON response, got nil")
	}
}

func TestProviderGraphQLErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": nil,
			"errors": []map[string]interface{}{
				{"message": "Rate limit reached for anonymous queries"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	_, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if err == nil {
		t.Fatal("expected error on GraphQL error response, got nil")
	}
	if !strings.Contains(err.Error(), "Rate limit") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestProviderRateLimitedHTTP429(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`Too Many Requests`))
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	_, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if !errors.Is(err, leetcode.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got: %v", err)
	}
}

func TestProviderUnauthorizedHTTP401(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`Unauthorized`))
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	_, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if !errors.Is(err, leetcode.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestProviderServerErrorHTTP500(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`Internal Server Error`))
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	_, err := adapter.FetchRecentSubmissions(context.Background(), "alice", 20)
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("expected HTTP 500 error, got: %v", err)
	}
}

func TestProviderContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := adapter.FetchRecentSubmissions(ctx, "alice", 20)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
}

func TestProviderAccountVerificationTokenExactMatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"matchedUser": map[string]interface{}{
					"username": "alice",
					"profile": map[string]interface{}{
						"aboutMe": "Stanford CS student. Eterno: eterno-verify-9876543210 please verify me!",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)

	// Case 1: Exact token present in bio
	verified, _, err := adapter.VerifyAccount(context.Background(), "alice", "eterno-verify-9876543210")
	if err != nil || !verified {
		t.Errorf("expected account to verify with exact token in bio, got %v, err: %v", verified, err)
	}

	// Case 2: Wrong token
	verified, _, err = adapter.VerifyAccount(context.Background(), "alice", "eterno-verify-WRONGTOKEN")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if verified {
		t.Error("expected account verification to fail with wrong token")
	}

	// Case 3: Empty token rejected
	_, _, err = adapter.VerifyAccount(context.Background(), "alice", "")
	if err == nil {
		t.Error("expected error with empty token")
	}
}

func TestProviderAccountVerificationUserNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"matchedUser": nil, // User not found
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter := leetcode.NewAdapter(ts.URL, false)
	verified, _, err := adapter.VerifyAccount(context.Background(), "nonexistent_user", "token-123")
	if !errors.Is(err, leetcode.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
	if verified {
		t.Error("expected verified=false for nonexistent user")
	}
}

func TestProviderURLParsingSecurityAndEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectSlug  string
		expectError bool
	}{
		{"Canonical problem URL", "https://leetcode.com/problems/two-sum/", "two-sum", false},
		{"URL with description suffix", "https://leetcode.com/problems/trapping-rain-water/description/", "trapping-rain-water", false},
		{"URL with query params", "https://leetcode.com/problems/two-sum/?envType=daily-question", "two-sum", false},
		{"URL with fragment", "https://leetcode.com/problems/two-sum/#solution", "two-sum", false},
		{"www prefix", "https://www.leetcode.com/problems/invert-binary-tree/", "invert-binary-tree", false},
		{"Trailing slash absent", "https://leetcode.com/problems/reverse-linked-list", "reverse-linked-list", false},
		{"Plain slug", "two-sum", "two-sum", false},
		{"Uppercase slug in URL normalized", "https://leetcode.com/problems/TWO-SUM/", "two-sum", false},
		{"Plain uppercase slug normalized", "TWO-SUM", "two-sum", false},
		{"Whitespace around input", "   https://leetcode.com/problems/two-sum/   ", "two-sum", false},

		// Adversarial and invalid inputs
		{"Spoofed domain suffix", "https://leetcode.com.evil.example/problems/two-sum", "", true},
		{"Evil domain with leetcode in name", "https://notleetcode.com/problems/two-sum", "", true},
		{"Userinfo spoofing", "https://leetcode.com@evil.example/problems/two-sum", "", true},
		{"Non-LeetCode host", "https://hackerrank.com/challenges/solve-me-first", "", true},
		{"Non-problems path", "https://leetcode.com/contest/weekly-contest-300", "", true},
		{"Non-standard port in production", "https://leetcode.com:8443/problems/two-sum", "", true},
		{"Empty string", "", "", true},
		{"Empty slug segment", "https://leetcode.com/problems//", "", true},
		{"Double hyphens in slug", "two--sum", "", true},
		{"Leading hyphen in slug", "-two-sum", "", true},
		{"Trailing hyphen in slug", "two-sum-", "", true},
		{"Invalid characters", "two_sum$", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, err := leetcode.ParseProblemSlug(tt.input)
			if tt.expectError && err == nil {
				t.Errorf("expected error for input %q, got slug %q", tt.input, slug)
			}
			if !tt.expectError {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if slug != tt.expectSlug {
					t.Errorf("slug mismatch for input %q: expected %q, got %q", tt.input, tt.expectSlug, slug)
				}
			}
		})
	}
}
