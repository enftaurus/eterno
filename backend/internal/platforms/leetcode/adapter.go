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

var (
	slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

	ErrRateLimited     = errors.New("leetcode API rate limit exceeded (HTTP 429)")
	ErrUserNotFound    = errors.New("leetcode user not found or profile is private")
	ErrUnauthorized    = errors.New("leetcode API unauthorized access (HTTP 401/403)")
	ErrInvalidResponse = errors.New("invalid or empty response from leetcode API")
	ErrProfileMissing  = errors.New("leetcode user profile data unavailable")
)

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
			Timeout: 15 * time.Second,
		},
		userAgent: "Mozilla/5.0 (compatible; EternoBot/1.0; +https://eterno.dev)",
		mockMode:  mockMode,
		mockStore: make(map[string][]models.SubmissionDTO),
	}
}

func (a *Adapter) SetMockSubmissions(username string, subs []models.SubmissionDTO) {
	a.mockStore[strings.ToLower(strings.TrimSpace(username))] = subs
}

func (a *Adapter) SetProxy(rawURL string) error {
	if rawURL == "" {
		a.httpClient.Transport = http.DefaultTransport
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid proxy URL %q: %w", rawURL, err)
	}
	a.httpClient.Transport = &http.Transport{
		Proxy: http.ProxyURL(parsed),
	}
	return nil
}

func (a *Adapter) PlatformName() string {
	return models.PlatformLeetCode
}

// ParseProblemSlug extracts and normalizes the problem slug from a raw string or URL.
// Hostnames must strictly match leetcode.com or www.leetcode.com. Substring matches are rejected.
func ParseProblemSlug(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("problem identifier or URL cannot be empty")
	}
	if len(trimmed) > 1000 {
		return "", errors.New("problem identifier or URL exceeds maximum allowed length")
	}

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		u, err := url.Parse(trimmed)
		if err != nil {
			return "", fmt.Errorf("invalid problem URL format: %w", err)
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return "", fmt.Errorf("unsupported URL scheme: %q", u.Scheme)
		}

		// Security: Reject userinfo (e.g. leetcode.com@evil.example)
		if u.User != nil {
			return "", errors.New("URLs with userinfo credentials are not permitted")
		}

		hostname := strings.ToLower(u.Hostname())
		if hostname != "leetcode.com" && hostname != "www.leetcode.com" {
			return "", fmt.Errorf("unsupported host %q: problem URL must be from leetcode.com", hostname)
		}

		// Security: Disallow unexpected non-standard ports in production
		if u.Port() != "" && u.Port() != "80" && u.Port() != "443" {
			// Allow custom port only if pointing to localhost in tests
			if hostname != "127.0.0.1" && hostname != "localhost" {
				return "", fmt.Errorf("non-standard port %q not allowed for leetcode.com", u.Port())
			}
		}

		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 && strings.ToLower(parts[0]) == "problems" {
			slug := strings.ToLower(parts[1])
			if slugRegex.MatchString(slug) {
				return slug, nil
			}
			return "", fmt.Errorf("invalid problem slug format extracted from URL: %q", slug)
		}
		return "", errors.New("cannot extract problem slug from LeetCode URL (path must be /problems/{slug})")
	}

	lower := strings.ToLower(trimmed)
	if slugRegex.MatchString(lower) {
		return lower, nil
	}

	return "", fmt.Errorf("invalid problem slug %q: must be lowercase alphanumeric separated by single hyphens", raw)
}

func IsAccepted(verdict string) bool {
	v := strings.ToLower(strings.TrimSpace(verdict))
	return v == "accepted" || v == "ac" || v == "10"
}

// VerifyAccount verifies ownership of a LeetCode account by inspecting the user's public bio.
// In test/mock mode, pre-configured submissions or explicit test checks are used.
func (a *Adapter) VerifyAccount(ctx context.Context, username, expectedToken string) (bool, map[string]interface{}, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" {
		return false, nil, errors.New("username cannot be empty")
	}

	expectedToken = strings.TrimSpace(expectedToken)
	if expectedToken == "" {
		return false, nil, errors.New("expected verification token cannot be empty")
	}

	// If mock mode is explicitly configured for isolated testing
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

	matchedUser, ok := data["matchedUser"].(map[string]interface{})
	if !ok || matchedUser == nil {
		return false, nil, ErrUserNotFound
	}

	returnedUsername, _ := matchedUser["username"].(string)
	if strings.ToLower(strings.TrimSpace(returnedUsername)) != username {
		return false, nil, fmt.Errorf("username mismatch: requested %q, provider returned %q", username, returnedUsername)
	}

	profile, ok := matchedUser["profile"].(map[string]interface{})
	if !ok || profile == nil {
		return false, nil, ErrProfileMissing
	}

	aboutMe, _ := profile["aboutMe"].(string)
	// Security: Require exact token boundary matching rather than unsafe substring containment
	tokenPattern := regexp.MustCompile(`(?i)(?:^|[\s,.;:!?"'()\[\]{}])` + regexp.QuoteMeta(expectedToken) + `(?:$|[\s,.;:!?"'()\[\]{}])`)
	verified := tokenPattern.MatchString(aboutMe)

	return verified, profile, nil
}

// FetchRecentSubmissions retrieves the user's recent accepted submission history from LeetCode.
// Note: LeetCode's public GraphQL API provides accepted submission history via recentAcSubmissionList.
// Records without valid IDs, valid slugs, or trustworthy timestamps are rejected.
func (a *Adapter) FetchRecentSubmissions(ctx context.Context, username string, limit int) ([]models.SubmissionDTO, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" {
		return nil, errors.New("username cannot be empty")
	}

	if limit <= 0 || limit > 50 {
		limit = 20
	}

	if a.mockMode {
		if subs, ok := a.mockStore[username]; ok {
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

	list, ok := data["recentAcSubmissionList"].([]interface{})
	if !ok {
		return []models.SubmissionDTO{}, nil
	}

	results := make([]models.SubmissionDTO, 0, len(list))
	now := time.Now().UTC()

	for _, item := range list {
		subMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		idRaw := subMap["id"]
		if idRaw == nil {
			continue // Reject: Missing provider submission ID
		}
		idStr := strings.TrimSpace(fmt.Sprintf("%v", idRaw))
		if idStr == "" || idStr == "<nil>" {
			continue // Reject: Empty ID
		}

		title, _ := subMap["title"].(string)
		titleSlugRaw, _ := subMap["titleSlug"].(string)
		titleSlug := strings.ToLower(strings.TrimSpace(titleSlugRaw))
		if titleSlug == "" || !slugRegex.MatchString(titleSlug) {
			continue // Reject: Missing or malformed slug
		}

		// Parse timestamp strictly
		var subTime time.Time
		switch ts := subMap["timestamp"].(type) {
		case string:
			if sec, err := strconv.ParseInt(ts, 10, 64); err == nil && sec > 0 {
				subTime = time.Unix(sec, 0).UTC()
			}
		case float64:
			if ts > 0 {
				subTime = time.Unix(int64(ts), 0).UTC()
			}
		}

		// Security: Never fabricate a timestamp; reject if zero or future-dated (>10m clock skew)
		if subTime.IsZero() {
			continue // Reject: Missing or unparseable timestamp
		}
		if subTime.After(now.Add(10 * time.Minute)) {
			continue // Reject: Future-dated timestamp
		}

		results = append(results, models.SubmissionDTO{
			ProviderID:  idStr,
			ProblemSlug: titleSlug,
			Title:       title,
			Verdict:     "Accepted",
			IsAccepted:  true,
			SubmittedAt: subTime,
		})
	}

	return results, nil
}

func (a *Adapter) GetProfileStats(ctx context.Context, username string) (map[string]interface{}, error) {
	username = strings.ToLower(strings.TrimSpace(username))
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
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, context.Canceled
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, context.DeadlineExceeded
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("leetcode upstream server error: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leetcode returned unexpected HTTP %d", resp.StatusCode)
	}

	// Limit reader to 5MB to prevent memory exhaustion from oversized responses
	limitedReader := io.LimitReader(resp.Body, 5*1024*1024)

	var responsePayload struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(limitedReader).Decode(&responsePayload); err != nil {
		return nil, fmt.Errorf("failed to decode response JSON: %w", err)
	}

	if len(responsePayload.Errors) > 0 {
		if responsePayload.Data == nil {
			return nil, fmt.Errorf("leetcode GraphQL error: %s", responsePayload.Errors[0].Message)
		}
	}

	if responsePayload.Data == nil {
		return nil, ErrInvalidResponse
	}

	return responsePayload.Data, nil
}
