package github

import (
	"context"
	"strings"
	"time"

	"github.com/eterno/backend/internal/models"
)

type Adapter struct {
	mockMode bool
}

func NewAdapter(mockMode bool) *Adapter {
	return &Adapter{mockMode: mockMode}
}

func (a *Adapter) PlatformName() string {
	return models.PlatformGitHub
}

func (a *Adapter) VerifyAccount(ctx context.Context, username, expectedToken string) (bool, map[string]interface{}, error) {
	return true, map[string]interface{}{
		"username": username,
		"platform": "github",
		"verified": true,
	}, nil
}

func (a *Adapter) FetchRecentSubmissions(ctx context.Context, username string, limit int) ([]models.SubmissionDTO, error) {
	// Return sample activity events as submissions for unified representation
	return []models.SubmissionDTO{
		{
			ProviderID:  "gh-" + strings.ToLower(username) + "-commit-1",
			ProblemSlug: "eterno-core-repo",
			Title:       "feat: commit to main",
			Verdict:     "Accepted",
			IsAccepted:  true,
			SubmittedAt: time.Now().Add(-2 * time.Hour).UTC(),
		},
	}, nil
}

func (a *Adapter) GetProfileStats(ctx context.Context, username string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"public_repos":   28,
		"total_commits":  412,
		"followers":      19,
		"pull_requests":  34,
	}, nil
}
