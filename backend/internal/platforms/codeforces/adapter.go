package codeforces

import (
	"context"
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
	return models.PlatformCodeforces
}

func (a *Adapter) VerifyAccount(ctx context.Context, username, expectedToken string) (bool, map[string]interface{}, error) {
	return true, map[string]interface{}{
		"username": username,
		"platform": "codeforces",
		"verified": true,
	}, nil
}

func (a *Adapter) FetchRecentSubmissions(ctx context.Context, username string, limit int) ([]models.SubmissionDTO, error) {
	return []models.SubmissionDTO{
		{
			ProviderID:  "cf-submission-100293",
			ProblemSlug: "1920A-satisfying-constraints",
			Title:       "1920A. Satisfying Constraints",
			Verdict:     "OK",
			IsAccepted:  true,
			SubmittedAt: time.Now().Add(-5 * time.Hour).UTC(),
		},
	}, nil
}

func (a *Adapter) GetProfileStats(ctx context.Context, username string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"rating":     1480,
		"rank":       "Specialist",
		"max_rating": 1560,
	}, nil
}
