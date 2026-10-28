package codeforces

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eterno/backend/internal/models"
)

var (
	ErrUserNotFound    = errors.New("codeforces user not found")
	ErrRateLimited     = errors.New("codeforces rate limit exceeded (HTTP 429)")
	ErrInvalidResponse = errors.New("invalid or empty response from codeforces API")
)

type Adapter struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
	mockMode   bool
	mockStore  map[string][]models.SubmissionDTO
	mockUsers  map[string]map[string]interface{}
}

func NewAdapter(apiKey, apiSecret string, mockMode bool) *Adapter {
	return &Adapter{
		baseURL:    "https://codeforces.com/api",
		apiKey:     strings.TrimSpace(apiKey),
		apiSecret:  strings.TrimSpace(apiSecret),
		httpClient: &http.Client{Timeout: 15 * time.Second},
		mockMode:   mockMode,
		mockStore:  make(map[string][]models.SubmissionDTO),
		mockUsers:  make(map[string]map[string]interface{}),
	}
}

func (a *Adapter) PlatformName() string {
	return models.PlatformCodeforces
}

func (a *Adapter) SetMockSubmissions(handle string, subs []models.SubmissionDTO) {
	a.mockStore[strings.ToLower(strings.TrimSpace(handle))] = subs
}

func (a *Adapter) SetMockUser(handle string, data map[string]interface{}) {
	a.mockUsers[strings.ToLower(strings.TrimSpace(handle))] = data
}

// UserInfo represents the profile metadata returned from Codeforces
type UserInfo struct {
	Handle      string `json:"handle"`
	Rating      int    `json:"rating"`
	MaxRating   int    `json:"maxRating"`
	Rank        string `json:"rank"`
	MaxRank     string `json:"maxRank"`
	Contribution int   `json:"contribution"`
	AvatarURL   string `json:"avatar"`
}

// VerifyAccount checks whether a Codeforces handle exists and retrieves their current rating
func (a *Adapter) VerifyAccount(ctx context.Context, handle string) (bool, map[string]interface{}, error) {
	handle = strings.TrimSpace(handle)
	if handle == "" {
		return false, nil, errors.New("codeforces handle cannot be empty")
	}

	if a.mockMode {
		if meta, ok := a.mockUsers[strings.ToLower(handle)]; ok {
			return true, meta, nil
		}
		return false, nil, ErrUserNotFound
	}

	params := url.Values{}
	params.Set("handles", handle)

	var resp struct {
		Status  string     `json:"status"`
		Comment string     `json:"comment"`
		Result  []UserInfo `json:"result"`
	}

	err := a.doGet(ctx, "user.info", params, &resp)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return false, nil, ErrUserNotFound
		}
		return false, nil, err
	}

	if resp.Status != "OK" || len(resp.Result) == 0 {
		return false, nil, ErrUserNotFound
	}

	user := resp.Result[0]
	profileMeta := map[string]interface{}{
		"handle":       user.Handle,
		"rating":       user.Rating,
		"max_rating":   user.MaxRating,
		"rank":         user.Rank,
		"max_rank":     user.MaxRank,
		"avatar_url":   user.AvatarURL,
		"contribution": user.Contribution,
	}

	return true, profileMeta, nil
}

// FetchRecentSubmissions retrieves the user's recent accepted solutions from Codeforces
func (a *Adapter) FetchRecentSubmissions(ctx context.Context, handle string, count int) ([]models.SubmissionDTO, error) {
	handle = strings.TrimSpace(handle)
	if handle == "" {
		return nil, errors.New("codeforces handle cannot be empty")
	}
	if count <= 0 || count > 100 {
		count = 50
	}

	if a.mockMode {
		if subs, ok := a.mockStore[strings.ToLower(handle)]; ok {
			return subs, nil
		}
		return []models.SubmissionDTO{}, nil
	}

	params := url.Values{}
	params.Set("handle", handle)
	params.Set("from", "1")
	params.Set("count", strconv.Itoa(count))

	var resp struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
		Result  []struct {
			ID                  int64  `json:"id"`
			CreationTimeSeconds int64  `json:"creationTimeSeconds"`
			Verdict             string `json:"verdict"`
			Problem             struct {
				ContestID int      `json:"contestId"`
				Index     string   `json:"index"`
				Name      string   `json:"name"`
				Tags      []string `json:"tags"`
				Rating    int      `json:"rating"`
			} `json:"problem"`
		} `json:"result"`
	}

	err := a.doGet(ctx, "user.status", params, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Status != "OK" {
		return nil, fmt.Errorf("codeforces API error: %s", resp.Comment)
	}

	now := time.Now().UTC()
	var subs []models.SubmissionDTO

	for _, item := range resp.Result {
		// Only accepted solutions count as solves
		if strings.ToUpper(strings.TrimSpace(item.Verdict)) != "OK" {
			continue
		}

		if item.CreationTimeSeconds <= 0 {
			continue
		}

		subTime := time.Unix(item.CreationTimeSeconds, 0).UTC()
		if subTime.After(now.Add(10 * time.Minute)) {
			continue // Reject clock skewed / future timestamps
		}

		slug := strings.ToLower(fmt.Sprintf("%d-%s", item.Problem.ContestID, item.Problem.Index))
		if item.Problem.ContestID == 0 {
			slug = strings.ToLower(strings.ReplaceAll(item.Problem.Name, " ", "-"))
		}

		subs = append(subs, models.SubmissionDTO{
			ProviderID:  strconv.FormatInt(item.ID, 10),
			ProblemSlug: slug,
			Title:       fmt.Sprintf("%s (%d%s)", item.Problem.Name, item.Problem.ContestID, item.Problem.Index),
			Verdict:     "Accepted",
			IsAccepted:  true,
			SubmittedAt: subTime,
		})
	}

	return subs, nil
}

// FetchContestRatingHistory retrieves rating points history over completed contests
func (a *Adapter) FetchContestRatingHistory(ctx context.Context, handle string) ([]map[string]interface{}, error) {
	handle = strings.TrimSpace(handle)
	if handle == "" {
		return nil, errors.New("handle cannot be empty")
	}

	if a.mockMode {
		return []map[string]interface{}{}, nil
	}

	params := url.Values{}
	params.Set("handle", handle)

	var resp struct {
		Status string `json:"status"`
		Result []struct {
			ContestID       int    `json:"contestId"`
			ContestName     string `json:"contestName"`
			Rank            int    `json:"rank"`
			OldRating       int    `json:"oldRating"`
			NewRating       int    `json:"newRating"`
			RatingUpdateTime int64 `json:"ratingUpdateTimeSeconds"`
		} `json:"result"`
	}

	err := a.doGet(ctx, "user.rating", params, &resp)
	if err != nil {
		return nil, err
	}

	var history []map[string]interface{}
	for _, r := range resp.Result {
		history = append(history, map[string]interface{}{
			"contest_id":   r.ContestID,
			"contest_name": r.ContestName,
			"rank":         r.Rank,
			"old_rating":   r.OldRating,
			"new_rating":   r.NewRating,
			"updated_at":   time.Unix(r.RatingUpdateTime, 0).UTC(),
		})
	}

	return history, nil
}

func (a *Adapter) doGet(ctx context.Context, methodName string, params url.Values, target interface{}) error {
	reqURL := fmt.Sprintf("%s/%s", a.baseURL, methodName)

	// If apiKey and apiSecret are provided, sign with SHA-512 apiSig
	if a.apiKey != "" && a.apiSecret != "" {
		nowUnix := strconv.FormatInt(time.Now().Unix(), 10)
		params.Set("apiKey", a.apiKey)
		params.Set("time", nowUnix)

		randBytes := make([]byte, 3)
		_, _ = rand.Read(randBytes)
		randStr := hex.EncodeToString(randBytes) // 6 hex characters

		// Sort query params alphabetically by key, then value
		var keys []string
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var sortedPairs []string
		for _, k := range keys {
			vals := params[k]
			sort.Strings(vals)
			for _, v := range vals {
				sortedPairs = append(sortedPairs, fmt.Sprintf("%s=%s", k, v))
			}
		}
		paramStr := strings.Join(sortedPairs, "&")

		// String to hash: <rand>/<methodName>?<sortedParams>#<secret>
		toHash := fmt.Sprintf("%s/%s?%s#%s", randStr, methodName, paramStr, a.apiSecret)
		h := sha512.Sum512([]byte(toHash))
		sig := randStr + hex.EncodeToString(h[:])

		params.Set("apiSig", sig)
	}

	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "CodeCrusadersBot/1.0 (+https://eterno.dev)")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("codeforces server returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return fmt.Errorf("failed reading response body: %w", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("failed decoding response: %w", err)
	}

	return nil
}
