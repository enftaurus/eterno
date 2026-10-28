package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/platforms/github"
)

func TestGitHubAdapterOAuthURL(t *testing.T) {
	adapter := github.NewAdapter("test-client-id-123", "test-secret", true)

	url := adapter.GetOAuthURL("user-uuid-xyz")
	if !strings.Contains(url, "client_id=test-client-id-123") {
		t.Errorf("OAuth URL missing client_id: %s", url)
	}
	if !strings.Contains(url, "state=user-uuid-xyz") {
		t.Errorf("OAuth URL missing state: %s", url)
	}
	if !strings.Contains(url, "scope=read%3Auser%2Crepo") && !strings.Contains(url, "scope=read:user,repo") {
		t.Errorf("OAuth URL missing scopes: %s", url)
	}

	if adapter.PlatformName() != models.PlatformGitHub {
		t.Errorf("expected platform name github, got %s", adapter.PlatformName())
	}
}

func TestGitHubAdapterMockOperations(t *testing.T) {
	adapter := github.NewAdapter("test-client", "test-secret", true)
	ctx := context.Background()

	// Token exchange mock
	token, err := adapter.ExchangeOAuthCode(ctx, "mock-code-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty access token")
	}

	// User profile mock
	profile, err := adapter.FetchUserProfile(ctx, token)
	if err != nil {
		t.Fatalf("unexpected error fetching profile: %v", err)
	}
	if profile.Login != "mock-dev" {
		t.Errorf("expected login mock-dev, got %s", profile.Login)
	}

	// Commits mock
	adapter.SetMockCommits("mock-dev", []github.GitHubCommitDTO{
		{
			SHA:         "abc12345",
			Message:     "feat: implement developer intelligence engine",
			RepoName:    "eterno/code-crusaders",
			CommittedAt: time.Now().UTC(),
		},
	})

	commits, err := adapter.FetchUserCommitsAndActivity(ctx, "mock-dev", token)
	if err != nil {
		t.Fatalf("unexpected error fetching commits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].SHA != "abc12345" {
		t.Errorf("expected commit SHA abc12345, got %s", commits[0].SHA)
	}
}
