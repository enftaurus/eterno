package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eterno/backend/internal/models"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Adapter struct {
	endpoint   string
	httpClient *http.Client
	userAgent  string
	mockMode   bool
	mockStore  map[string][]models.SubmissionDTO
}

func NewAdapter(endpoint string, mockMode bool) *Adapter {
	if endpoint == "" {
		endpoint = "https://leetcode.com/graphql"
	}
	return &Adapter{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		userAgent: "Mozilla/5.0 (compatible; EternoBot/1.0; +https://eterno.dev)",
		mockMode:  mockMode,
		mockStore: make(map[string][]models.SubmissionDTO),
	}
}

func (a *Adapter) SetMockSubmissions(username string, subs []models.SubmissionDTO) {
	a.mockStore[strings.ToLower(username)] = subs
}

func (a *Adapter) PlatformName() string {
	return models.PlatformLeetCode
}

func ParseProblemSlug(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("problem identifier or URL cannot be empty")
	}

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		u, err := url.Parse(trimmed)
		if err != nil {
			return "", fmt.Errorf("invalid problem URL format: %w", err)
		}

		host := strings.ToLower(u.Host)
		if !strings.Contains(host, "leetcode.com") {
			return "", errors.New("unsupported domain: problem URL must be from leetcode.com")
		}

		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 && parts[0] == "problems" {
			slug := strings.ToLower(parts[1])
			if slugRegex.MatchString(slug) {
				return slug, nil
			}
		}
		return "", errors.New("cannot extract problem slug from LeetCode URL")
	}

	lower := strings.ToLower(trimmed)
	if slugRegex.MatchString(lower) {
		return lower, nil
	}

	return "", fmt.Errorf("invalid problem slug: %q (must be lowercase alphanumeric separated by single hyphens)", raw)
}

func IsAccepted(verdict string) bool {
	v := strings.ToLower(strings.TrimSpace(verdict))
	return v == "accepted" || v == "ac" || v == "10"
}

func (a *Adapter) VerifyAccount(ctx context.Context, username, expectedToken string) (bool, map[string]interface{}, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, nil, errors.New("username cannot be empty")
	}

	if a.mockMode {
		meta := map[string]interface{}{
			"verified":    true,
			"username":    username,
			"mock":        true,
			"verified_at": time.Now().UTC().Format(time.RFC3339),
		}
		return true, meta, nil
	}

	query := `query userPublicProfile($username: String!) {
		matchedUser(username: $username) {
			username
			profile {
				aboutMe
				userAvatar
				ranking
				reputation
			}
		}
	}`

	reqBody := map[string]interface{}{
		"query":     query,
		"variables": map[string]string{"username": username},
	}

	data, err := a.doGraphQL(ctx, reqBody)
	if err != nil {
		return false, nil, fmt.Errorf("leetcode profile query failed: %w", err)
	}

	matchedUser, _ := data["matchedUser"].(map[string]interface{})
	if matchedUser == nil {
		return false, nil, fmt.Errorf("leetcode user %q not found or account is private", username)
	}

	profile, _ := matchedUser["profile"].(map[string]interface{})
	if profile == nil {
		return false, nil, errors.New("user profile data unavailable")
	}

	aboutMe, _ := profile["aboutMe"].(string)
	verified := strings.Contains(aboutMe, expectedToken)

	return verified, profile, nil
}

func (a *Adapter) FetchRecentSubmissions(ctx context.Context, username string, limit int) ([]models.SubmissionDTO, error) {
	username = strings.TrimSpace(username)
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	if a.mockMode {
		if subs, ok := a.mockStore[strings.ToLower(username)]; ok {
			return subs, nil
		}
		return []models.SubmissionDTO{}, nil
	}

	query := `query recentAcSubmissions($username: String!, $limit: Int!) {
		recentAcSubmissionList(username: $username, limit: $limit) {
			id
			title
			titleSlug
			timestamp
		}
	}`

	reqBody := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"username": username,
			"limit":    limit,
		},
	}

	data, err := a.doGraphQL(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("leetcode submission query failed: %w", err)
	}

	list, _ := data["recentAcSubmissionList"].([]interface{})
	results := make([]models.SubmissionDTO, 0, len(list))

	for _, item := range list {
		subMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		idStr := fmt.Sprintf("%v", subMap["id"])
		title, _ := subMap["title"].(string)
		titleSlug, _ := subMap["titleSlug"].(string)

		var subTime time.Time
		switch ts := subMap["timestamp"].(type) {
		case string:
			if sec, err := strconv.ParseInt(ts, 10, 64); err == nil {
				subTime = time.Unix(sec, 0).UTC()
			}
		case float64:
			subTime = time.Unix(int64(ts), 0).UTC()
		}

		if subTime.IsZero() {
			subTime = time.Now().UTC()
		}

		results = append(results, models.SubmissionDTO{
			ProviderID:  idStr,
			ProblemSlug: strings.ToLower(titleSlug),
			Title:       title,
			Verdict:     "Accepted",
			IsAccepted:  true,
			SubmittedAt: subTime,
		})
	}

	return results, nil
}

func (a *Adapter) GetProfileStats(ctx context.Context, username string) (map[string]interface{}, error) {
	if a.mockMode {
		return map[string]interface{}{
			"solved_total":  142,
			"solved_easy":   60,
			"solved_medium": 65,
			"solved_hard":   17,
			"ranking":       45200,
		}, nil
	}

	query := `query userProblemsSolved($username: String!) {
		matchedUser(username: $username) {
			submitStatsGlobal {
				acSubmissionNum {
					difficulty
					count
				}
			}
		}
	}`

	reqBody := map[string]interface{}{
		"query":     query,
		"variables": map[string]string{"username": username},
	}

	data, err := a.doGraphQL(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (a *Adapter) doGraphQL(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("leetcode returned HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var responsePayload struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&responsePayload); err != nil {
		return nil, fmt.Errorf("failed to decode response JSON: %w", err)
	}

	if len(responsePayload.Errors) > 0 {
		return nil, fmt.Errorf("leetcode API error: %s", responsePayload.Errors[0].Message)
	}

	return responsePayload.Data, nil
}
