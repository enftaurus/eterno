package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eterno/backend/internal/models"
)

var (
	ErrUserNotFound    = errors.New("github user not found")
	ErrUnauthorized    = errors.New("github unauthorized or token invalid")
	ErrRateLimited     = errors.New("github API rate limit exceeded")
	ErrInvalidResponse = errors.New("invalid or empty response from github API")
)

type Adapter struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	mockMode     bool
	mockEvents   map[string][]GitHubCommitDTO
}

func NewAdapter(clientID, clientSecret string, mockMode bool) *Adapter {
	return &Adapter{
		clientID:     strings.TrimSpace(clientID),
		clientSecret: strings.TrimSpace(clientSecret),
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		mockMode:     mockMode,
		mockEvents:   make(map[string][]GitHubCommitDTO),
	}
}

func (a *Adapter) PlatformName() string {
	return models.PlatformGitHub
}

func (a *Adapter) SetMockCommits(username string, commits []GitHubCommitDTO) {
	a.mockEvents[strings.ToLower(strings.TrimSpace(username))] = commits
}

type GitHubUser struct {
	Login       string `json:"login"`
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	AvatarURL   string `json:"avatar_url"`
	PublicRepos int    `json:"public_repos"`
	Bio         string `json:"bio"`
	CreatedAt   string `json:"created_at"`
}

type GitHubCommitDTO struct {
	SHA         string    `json:"sha"`
	Message     string    `json:"message"`
	RepoName    string    `json:"repo_name"`
	CommittedAt time.Time `json:"committed_at"`
}

func (a *Adapter) GetOAuthURL(state string) string {
	u := "https://github.com/login/oauth/authorize"
	params := url.Values{}
	params.Set("client_id", a.clientID)
	params.Set("scope", "read:user,repo")
	if state != "" {
		params.Set("state", state)
	}
	return u + "?" + params.Encode()
}

// ExchangeOAuthCode swaps the temporary authorization code for an OAuth access token
func (a *Adapter) ExchangeOAuthCode(ctx context.Context, code string) (string, error) {
	if a.mockMode {
		return "mock-gh-access-token-xyz", nil
	}

	payload := map[string]string{
		"client_id":     a.clientID,
		"client_secret": a.clientSecret,
		"code":          code,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CodeCrusadersBot/1.0 (+https://eterno.dev)")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode oauth response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("github oauth error: %s (%s)", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", errors.New("no access token received from github")
	}

	return tokenResp.AccessToken, nil
}

// FetchUserProfile retrieves the authenticated user's profile info
func (a *Adapter) FetchUserProfile(ctx context.Context, accessToken string) (*GitHubUser, error) {
	if a.mockMode {
		return &GitHubUser{
			Login:       "mock-dev",
			Name:        "Mock Developer",
			AvatarURL:   "https://avatars.githubusercontent.com/u/0",
			PublicRepos: 18,
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "CodeCrusadersBot/1.0 (+https://eterno.dev)")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user API returned HTTP %d", resp.StatusCode)
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// FetchUserCommitsAndActivity queries user public and repository events
func (a *Adapter) FetchUserCommitsAndActivity(ctx context.Context, username, accessToken string) ([]GitHubCommitDTO, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.New("username cannot be empty")
	}

	if a.mockMode {
		if c, ok := a.mockEvents[strings.ToLower(username)]; ok {
			return c, nil
		}
		return []GitHubCommitDTO{}, nil
	}

	reqURL := fmt.Sprintf("https://api.github.com/users/%s/events?per_page=100", url.PathEscape(username))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "CodeCrusadersBot/1.0 (+https://eterno.dev)")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	} else if a.clientID != "" && a.clientSecret != "" {
		req.SetBasicAuth(a.clientID, a.clientSecret)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github events query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrUserNotFound
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, err
	}

	var events []struct {
		ID        string    `json:"id"`
		Type      string    `json:"type"`
		CreatedAt time.Time `json:"created_at"`
		Repo      struct {
			Name string `json:"name"`
		} `json:"repo"`
		Payload struct {
			Commits []struct {
				SHA     string `json:"sha"`
				Message string `json:"message"`
			} `json:"commits"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(body, &events); err != nil {
		return nil, fmt.Errorf("failed decoding github events: %w", err)
	}

	var results []GitHubCommitDTO
	for _, ev := range events {
		if ev.Type == "PushEvent" {
			for _, c := range ev.Payload.Commits {
				results = append(results, GitHubCommitDTO{
					SHA:         c.SHA,
					Message:     c.Message,
					RepoName:    ev.Repo.Name,
					CommittedAt: ev.CreatedAt,
				})
			}
		}
	}

	return results, nil
}
