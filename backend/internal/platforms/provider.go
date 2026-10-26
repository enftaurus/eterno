package platforms

import (
	"context"

	"github.com/eterno/backend/internal/models"
)

type PlatformAdapter interface {
	PlatformName() string
	VerifyAccount(ctx context.Context, username, expectedToken string) (bool, map[string]interface{}, error)
	FetchRecentSubmissions(ctx context.Context, username string, limit int) ([]models.SubmissionDTO, error)
	GetProfileStats(ctx context.Context, username string) (map[string]interface{}, error)
}
