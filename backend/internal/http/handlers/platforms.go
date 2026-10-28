package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/auth"
	"github.com/eterno/backend/internal/config"
	"github.com/eterno/backend/internal/groups"
	"github.com/eterno/backend/internal/jobs"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/platforms/codeforces"
	"github.com/eterno/backend/internal/platforms/github"
	"github.com/eterno/backend/internal/platforms/leetcode"
	"github.com/eterno/backend/internal/scoring"
	"github.com/eterno/backend/internal/submissions"
	"github.com/eterno/backend/internal/verification"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PlatformHandler struct {
	db           *sql.DB
	cfg          *config.Config
	queue        *jobs.Queue
	lcAdapter    *leetcode.Adapter
	cfAdapter    *codeforces.Adapter
	ghAdapter    *github.Adapter
	subService   *submissions.Service
	verifService *verification.Service
	actService   *activity.Service
	scoringSvc   *scoring.Service
	groupSvc     *groups.Service
}

func NewPlatformHandler(
	db *sql.DB,
	cfg *config.Config,
	queue *jobs.Queue,
	lcAdapter *leetcode.Adapter,
	cfAdapter *codeforces.Adapter,
	ghAdapter *github.Adapter,
	subService *submissions.Service,
	verifService *verification.Service,
	actService *activity.Service,
	scoringSvc *scoring.Service,
	groupSvc *groups.Service,
) *PlatformHandler {
	return &PlatformHandler{
		db:           db,
		cfg:          cfg,
		queue:        queue,
		lcAdapter:    lcAdapter,
		cfAdapter:    cfAdapter,
		ghAdapter:    ghAdapter,
		subService:   subService,
		verifService: verifService,
		actService:   actService,
		scoringSvc:   scoringSvc,
		groupSvc:     groupSvc,
	}
}

func (h *PlatformHandler) LinkLeetCode(c *gin.Context) {
	userID := c.GetString("user_id")
	var req models.LinkAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "LeetCode username cannot be empty"})
		return
	}

	token := "eterno-verify-" + uuid.New().String()[:8]

	query := `
		INSERT INTO platform_accounts (
			user_id, platform, username, verification_status, verification_token,
			verification_attempt_at, updated_at
		) VALUES ($1, 'leetcode', $2, 'PENDING', $3, NOW(), NOW())
		ON CONFLICT (user_id, platform) DO UPDATE
		SET username = EXCLUDED.username,
		    verification_status = 'PENDING',
		    verification_token = EXCLUDED.verification_token,
		    verification_attempt_at = NOW(),
		    updated_at = NOW()
		RETURNING id, user_id, platform, username, verification_status, verification_token
	`

	var pa models.PlatformAccount
	err := h.db.QueryRowContext(c.Request.Context(), query, userID, username, token).Scan(
		&pa.ID, &pa.UserID, &pa.Platform, &pa.Username, &pa.VerificationStatus, &pa.VerificationToken,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link LeetCode account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account":            pa,
		"verification_token": token,
		"instructions":       "Paste this verification token into your LeetCode profile 'About Me' section, then click 'Verify Account'.",
	})
}

func (h *PlatformHandler) VerifyLeetCode(c *gin.Context) {
	userID := c.GetString("user_id")

	var username, expectedToken string
	query := `
		SELECT username, verification_token
		FROM platform_accounts
		WHERE user_id = $1 AND platform = 'leetcode'
	`
	err := h.db.QueryRowContext(c.Request.Context(), query, userID).Scan(&username, &expectedToken)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No LeetCode account linked for verification"})
		return
	}

	// Immediate verification check via adapter
	verified, profileMeta, err := h.lcAdapter.VerifyAccount(c.Request.Context(), username, expectedToken)
	if err != nil {
		// Enqueue job for background retry
		_, _ = h.queue.Enqueue(c.Request.Context(), userID, "leetcode", "ACCOUNT_VERIFY", map[string]interface{}{
			"username": username,
			"token":    expectedToken,
		})
		c.JSON(http.StatusOK, gin.H{
			"status":   "PENDING",
			"message":  "Verification request queued for background processing",
			"username": username,
		})
		return
	}

	if verified {
		_, _ = h.db.ExecContext(c.Request.Context(), `
			UPDATE platform_accounts
			SET verification_status = 'VERIFIED', verified_at = NOW(),
			    last_sync_status = 'IDLE', sync_error_message = NULL, updated_at = NOW()
			WHERE user_id = $1 AND platform = 'leetcode'
		`, userID)

		c.JSON(http.StatusOK, gin.H{
			"status":   "VERIFIED",
			"message":  "LeetCode account verified successfully!",
			"username": username,
			"profile":  profileMeta,
		})
	} else {
		_, _ = h.db.ExecContext(c.Request.Context(), `
			UPDATE platform_accounts
			SET verification_status = 'FAILED',
			    sync_error_message = 'Verification token not found in profile bio',
			    updated_at = NOW()
			WHERE user_id = $1 AND platform = 'leetcode'
		`, userID)

		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "FAILED",
			"error":   "Verification token was not found in your LeetCode profile 'About Me' bio. Please update your profile and try again.",
			"token":   expectedToken,
		})
	}
}

func (h *PlatformHandler) SyncLeetCode(c *gin.Context) {
	userID := c.GetString("user_id")

	var username, verifStatus string
	var lastSyncAt sql.NullTime
	query := `
		SELECT username, verification_status, last_sync_at
		FROM platform_accounts
		WHERE user_id = $1 AND platform = 'leetcode'
	`
	err := h.db.QueryRowContext(c.Request.Context(), query, userID).Scan(&username, &verifStatus, &lastSyncAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No linked LeetCode account found"})
		return
	}

	if verifStatus != "VERIFIED" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account must be verified before synchronizing submissions",
			"status": verifStatus,
		})
		return
	}

	// Rate-limit protection: Prevent spamming LeetCode by enforcing a 30-second cooldown per student
	if lastSyncAt.Valid && time.Since(lastSyncAt.Time) < 30*time.Second {
		remaining := int(30 - time.Since(lastSyncAt.Time).Seconds())
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": fmt.Sprintf("Sync cooldown active. Please wait %d seconds before requesting another sync.", remaining),
			"retry_after_seconds": remaining,
		})
		return
	}

	// Execute synchronous sync pass for immediate response
	submissionsList, err := h.lcAdapter.FetchRecentSubmissions(c.Request.Context(), username, 30)
	if err != nil {
		// Queue background job
		_, _ = h.queue.Enqueue(c.Request.Context(), userID, "leetcode", "USER_SYNC", map[string]interface{}{"username": username})
		c.JSON(http.StatusAccepted, gin.H{
			"status":  "QUEUED",
			"message": "Sync queued for background retry due to temporary provider delay",
		})
		return
	}

	recordsIngested := 0
	recordsVerified := 0

	for _, subDTO := range submissionsList {
		sub, isNew, err := h.subService.IngestSubmission(c.Request.Context(), userID, models.PlatformLeetCode, subDTO)
		if err != nil {
			continue
		}
		if isNew {
			recordsIngested++
		}

		actMeta := map[string]interface{}{
			"slug":        subDTO.ProblemSlug,
			"title":       subDTO.Title,
			"verdict":     subDTO.Verdict,
			"is_accepted": subDTO.IsAccepted,
		}
		_ = h.actService.RecordEvent(c.Request.Context(), userID, "leetcode", subDTO.ProviderID, "accepted_submission", subDTO.SubmittedAt, actMeta)

		verifRes, err := h.verifService.VerifySubmission(c.Request.Context(), sub)
		if err == nil && verifRes != nil && verifRes.Status == "VERIFIED" {
			recordsVerified++
		}
	}

	// Update platform accounts
	_, _ = h.db.ExecContext(c.Request.Context(), `
		UPDATE platform_accounts
		SET last_sync_at = NOW(), last_sync_status = 'SUCCESS', sync_error_message = NULL, updated_at = NOW()
		WHERE user_id = $1 AND platform = 'leetcode'
	`, userID)

	c.JSON(http.StatusOK, gin.H{
		"status":           "SUCCESS",
		"username":         username,
		"records_ingested": recordsIngested,
		"records_verified": recordsVerified,
		"synced_at":        time.Now().UTC(),
	})
}

func (h *PlatformHandler) ListPlatformAccounts(c *gin.Context) {
	userID := c.GetString("user_id")
	query := `
		SELECT id, user_id, platform, username, external_user_id,
		       verification_status, verification_token, verified_at,
		       last_sync_at, last_sync_status, sync_error_message, created_at, updated_at
		FROM platform_accounts
		WHERE user_id = $1
		ORDER BY platform ASC
	`
	rows, err := h.db.QueryContext(c.Request.Context(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var accounts []models.PlatformAccount
	for rows.Next() {
		var pa models.PlatformAccount
		var extID, vToken, errMsg sql.NullString
		var vAt, lastSync sql.NullTime

		if err := rows.Scan(
			&pa.ID, &pa.UserID, &pa.Platform, &pa.Username, &extID,
			&pa.VerificationStatus, &vToken, &vAt,
			&lastSync, &pa.LastSyncStatus, &errMsg, &pa.CreatedAt, &pa.UpdatedAt,
		); err == nil {
			if extID.Valid {
				pa.ExternalUserID = &extID.String
			}
			if vToken.Valid {
				pa.VerificationToken = &vToken.String
			}
			if vAt.Valid {
				pa.VerifiedAt = &vAt.Time
			}
			if lastSync.Valid {
				pa.LastSyncAt = &lastSync.Time
			}
			if errMsg.Valid {
				pa.SyncErrorMessage = &errMsg.String
			}
			accounts = append(accounts, pa)
		}
	}

	c.JSON(http.StatusOK, accounts)
}

func (h *PlatformHandler) GetStudentSubmissions(c *gin.Context) {
	userID := c.GetString("user_id")
	list, err := h.subService.GetSubmissionsForUser(c.Request.Context(), userID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// LinkCodeforces links a user's Codeforces handle and verifies its existence
func (h *PlatformHandler) LinkCodeforces(c *gin.Context) {
	userID := c.GetString("user_id")
	var req models.LinkAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	handle := strings.TrimSpace(req.Username)
	if handle == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Codeforces handle cannot be empty"})
		return
	}

	verified, profileMeta, err := h.cfAdapter.VerifyAccount(c.Request.Context(), handle)
	if err != nil || !verified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Codeforces handle not found or API unavailable"})
		return
	}

	metaJSON, _ := json.Marshal(profileMeta)
	query := `
		INSERT INTO platform_accounts (
			user_id, platform, username, verification_status, verified_at,
			profile_metadata, updated_at
		) VALUES ($1, 'codeforces', $2, 'VERIFIED', NOW(), $3, NOW())
		ON CONFLICT (user_id, platform) DO UPDATE
		SET username = EXCLUDED.username,
		    verification_status = 'VERIFIED',
		    verified_at = NOW(),
		    profile_metadata = EXCLUDED.profile_metadata,
		    updated_at = NOW()
		RETURNING id, user_id, platform, username, verification_status
	`

	var pa models.PlatformAccount
	err = h.db.QueryRowContext(c.Request.Context(), query, userID, handle, metaJSON).Scan(
		&pa.ID, &pa.UserID, &pa.Platform, &pa.Username, &pa.VerificationStatus,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed saving Codeforces account link"})
		return
	}

	// Trigger initial async sync of submissions and rating
	go h.syncCodeforcesInternal(userID, handle)

	c.JSON(http.StatusOK, gin.H{
		"message": "Codeforces handle successfully verified and linked!",
		"account": pa,
		"profile": profileMeta,
	})
}

// SyncCodeforces synchronizes recent accepted submissions and rating history
func (h *PlatformHandler) SyncCodeforces(c *gin.Context) {
	userID := c.GetString("user_id")

	var handle string
	err := h.db.QueryRowContext(c.Request.Context(), `
		SELECT username FROM platform_accounts 
		WHERE user_id = $1 AND platform = 'codeforces' AND verification_status = 'VERIFIED'
	`, userID).Scan(&handle)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No verified Codeforces account linked"})
		return
	}

	ingested, verified := h.syncCodeforcesInternal(userID, handle)

	c.JSON(http.StatusOK, gin.H{
		"message":          "Codeforces synchronization complete",
		"records_ingested": ingested,
		"records_verified": verified,
	})
}

func (h *PlatformHandler) syncCodeforcesInternal(userID, handle string) (int, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subs, err := h.cfAdapter.FetchRecentSubmissions(ctx, handle, 50)
	if err != nil {
		return 0, 0
	}

	ingested := 0
	for _, s := range subs {
		sub, isNew, err := h.subService.IngestSubmission(ctx, userID, models.PlatformCodeforces, s)
		if err == nil {
			if isNew {
				ingested++
			}
			// Record into canonical activity_events
			_ = h.actService.RecordEvent(ctx, userID, "codeforces", s.ProviderID, "accepted_submission", s.SubmittedAt, map[string]interface{}{
				"slug":  s.ProblemSlug,
				"title": s.Title,
			})
			// Verify if matches any challenge problem
			_, _ = h.verifService.VerifySubmission(ctx, sub)
		}
	}

	// Also sync rating history
	ratings, err := h.cfAdapter.FetchContestRatingHistory(ctx, handle)
	if err == nil {
		for _, r := range ratings {
			contestID := fmt.Sprintf("%v", r["contest_id"])
			if updatedAt, ok := r["updated_at"].(time.Time); ok {
				_ = h.actService.RecordEvent(ctx, userID, "codeforces", "contest-"+contestID, "contest", updatedAt, r)
			}
		}
	}

	// Update platform_accounts last sync
	_, _ = h.db.ExecContext(ctx, `
		UPDATE platform_accounts 
		SET last_sync_at = NOW(), last_sync_status = 'SUCCESS', updated_at = NOW()
		WHERE user_id = $1 AND platform = 'codeforces'
	`, userID)

	// Recalculate Developer Score
	_, _ = h.scoringSvc.CalculateDeveloperScore(ctx, userID)

	return ingested, len(subs)
}

// GetGitHubOAuthURL generates the OAuth authorization redirect link
func (h *PlatformHandler) GetGitHubOAuthURL(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		if authHeader := c.GetHeader("Authorization"); authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" && h.cfg != nil {
				if claims, err := auth.ValidateToken(h.cfg, parts[1]); err == nil {
					userID = claims.UserID
				}
			}
		}
	}
	if userID == "" {
		userID = c.Query("state")
	}
	url := h.ghAdapter.GetOAuthURL(userID)
	c.JSON(http.StatusOK, gin.H{"auth_url": url})
}

// HandleGitHubOAuthCallback processes the GitHub OAuth redirect from browser
func (h *PlatformHandler) HandleGitHubOAuthCallback(c *gin.Context) {
	frontendURL := "http://localhost:5173"
	if h.cfg != nil && h.cfg.FrontendURL != "" {
		frontendURL = h.cfg.FrontendURL
	}

	code := c.Query("code")
	userID := c.Query("state")

	if code == "" || userID == "" {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/settings/connected-accounts?error=%s", frontendURL, url.QueryEscape("Missing code or state in OAuth callback")))
		return
	}

	token, err := h.ghAdapter.ExchangeOAuthCode(c.Request.Context(), code)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/settings/connected-accounts?error=%s", frontendURL, url.QueryEscape("OAuth token exchange failed: "+err.Error())))
		return
	}

	user, err := h.ghAdapter.FetchUserProfile(c.Request.Context(), token)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/settings/connected-accounts?error=%s", frontendURL, url.QueryEscape("Failed to fetch GitHub profile: "+err.Error())))
		return
	}

	metaJSON, _ := json.Marshal(map[string]interface{}{
		"login":        user.Login,
		"name":         user.Name,
		"avatar_url":   user.AvatarURL,
		"public_repos": user.PublicRepos,
		"access_token": token,
	})

	query := `
		INSERT INTO platform_accounts (
			user_id, platform, username, external_user_id, verification_status,
			verified_at, profile_metadata, updated_at
		) VALUES ($1, 'github', $2, $3, 'VERIFIED', NOW(), $4, NOW())
		ON CONFLICT (user_id, platform) DO UPDATE
		SET username = EXCLUDED.username,
		    external_user_id = EXCLUDED.external_user_id,
		    verification_status = 'VERIFIED',
		    verified_at = NOW(),
		    profile_metadata = EXCLUDED.profile_metadata,
		    updated_at = NOW()
	`
	_, err = h.db.ExecContext(c.Request.Context(), query, userID, user.Login, fmt.Sprintf("%d", user.ID), metaJSON)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/settings/connected-accounts?error=%s", frontendURL, url.QueryEscape("Failed saving GitHub account link")))
		return
	}

	// Trigger initial sync of commits
	go h.syncGitHubInternal(userID, user.Login, token)

	// Redirect back to settings page on frontend
	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/settings/connected-accounts?status=github_connected", frontendURL))
}

// ExchangeGitHubOAuthCode handles direct JSON code exchange from frontend
func (h *PlatformHandler) ExchangeGitHubOAuthCode(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		if authHeader := c.GetHeader("Authorization"); authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" && h.cfg != nil {
				if claims, err := auth.ValidateToken(h.cfg, parts[1]); err == nil {
					userID = claims.UserID
				}
			}
		}
	}
	var req struct {
		Code  string `json:"code"`
		State string `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code is required"})
		return
	}

	if userID == "" && req.State != "" {
		userID = req.State
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required to link GitHub account"})
		return
	}

	token, err := h.ghAdapter.ExchangeOAuthCode(c.Request.Context(), req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth token exchange failed: " + err.Error()})
		return
	}

	user, err := h.ghAdapter.FetchUserProfile(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to fetch GitHub user profile: " + err.Error()})
		return
	}

	metaJSON, _ := json.Marshal(map[string]interface{}{
		"login":        user.Login,
		"name":         user.Name,
		"avatar_url":   user.AvatarURL,
		"public_repos": user.PublicRepos,
		"access_token": token,
	})

	query := `
		INSERT INTO platform_accounts (
			user_id, platform, username, external_user_id, verification_status,
			verified_at, profile_metadata, updated_at
		) VALUES ($1, 'github', $2, $3, 'VERIFIED', NOW(), $4, NOW())
		ON CONFLICT (user_id, platform) DO UPDATE
		SET username = EXCLUDED.username,
		    external_user_id = EXCLUDED.external_user_id,
		    verification_status = 'VERIFIED',
		    verified_at = NOW(),
		    profile_metadata = EXCLUDED.profile_metadata,
		    updated_at = NOW()
	`
	_, err = h.db.ExecContext(c.Request.Context(), query, userID, user.Login, fmt.Sprintf("%d", user.ID), metaJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed saving GitHub account link"})
		return
	}

	// Trigger initial sync of commits
	go h.syncGitHubInternal(userID, user.Login, token)

	c.JSON(http.StatusOK, gin.H{
		"message":  "GitHub account connected via OAuth successfully!",
		"username": user.Login,
	})
}

// LinkGitHub manually connects a public GitHub username
func (h *PlatformHandler) LinkGitHub(c *gin.Context) {
	userID := c.GetString("user_id")
	var req models.LinkAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "GitHub username cannot be empty"})
		return
	}

	metaJSON, _ := json.Marshal(map[string]interface{}{
		"login": username,
	})

	query := `
		INSERT INTO platform_accounts (
			user_id, platform, username, verification_status, verified_at,
			profile_metadata, updated_at
		) VALUES ($1, 'github', $2, 'VERIFIED', NOW(), $3, NOW())
		ON CONFLICT (user_id, platform) DO UPDATE
		SET username = EXCLUDED.username,
		    verification_status = 'VERIFIED',
		    verified_at = NOW(),
		    profile_metadata = EXCLUDED.profile_metadata,
		    updated_at = NOW()
		RETURNING id, user_id, platform, username, verification_status
	`

	var pa models.PlatformAccount
	err := h.db.QueryRowContext(c.Request.Context(), query, userID, username, metaJSON).Scan(
		&pa.ID, &pa.UserID, &pa.Platform, &pa.Username, &pa.VerificationStatus,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed saving GitHub account link"})
		return
	}

	// Trigger async sync
	go h.syncGitHubInternal(userID, username, "")

	c.JSON(http.StatusOK, gin.H{
		"message": "GitHub account successfully linked!",
		"account": pa,
	})
}

// SyncGitHub fetches recent commits and repositories
func (h *PlatformHandler) SyncGitHub(c *gin.Context) {
	userID := c.GetString("user_id")

	var username string
	var metaRaw []byte
	err := h.db.QueryRowContext(c.Request.Context(), `
		SELECT username, profile_metadata FROM platform_accounts 
		WHERE user_id = $1 AND platform = 'github' AND verification_status = 'VERIFIED'
	`, userID).Scan(&username, &metaRaw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No verified GitHub account linked"})
		return
	}

	token := ""
	if len(metaRaw) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(metaRaw, &m); err == nil {
			if t, ok := m["access_token"].(string); ok {
				token = t
			}
		}
	}

	ingested := h.syncGitHubInternal(userID, username, token)

	c.JSON(http.StatusOK, gin.H{
		"message":          "GitHub synchronization complete",
		"records_ingested": ingested,
	})
}

func (h *PlatformHandler) syncGitHubInternal(userID, username, token string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	commits, err := h.ghAdapter.FetchUserCommitsAndActivity(ctx, username, token)
	if err != nil {
		return 0
	}

	ingested := 0
	for _, c := range commits {
		meta := map[string]interface{}{
			"repo":    c.RepoName,
			"message": c.Message,
			"sha":     c.SHA,
		}
		err := h.actService.RecordEvent(ctx, userID, "github", c.SHA, "commit", c.CommittedAt, meta)
		if err == nil {
			ingested++
		}
	}

	// Update platform account
	_, _ = h.db.ExecContext(ctx, `
		UPDATE platform_accounts 
		SET last_sync_at = NOW(), last_sync_status = 'SUCCESS', updated_at = NOW()
		WHERE user_id = $1 AND platform = 'github'
	`, userID)

	// Recalculate Developer Score
	_, _ = h.scoringSvc.CalculateDeveloperScore(ctx, userID)

	return ingested
}

// GetDeveloperScore calculates and returns the 0-100 Developer Score
func (h *PlatformHandler) GetDeveloperScore(c *gin.Context) {
	userID := c.GetString("user_id")
	score, err := h.scoringSvc.CalculateDeveloperScore(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed calculating developer score: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, score)
}

// ListStudentGroups lists peer groups/clubs available for students
func (h *PlatformHandler) ListStudentGroups(c *gin.Context) {
	instID := c.GetString("institution_id")
	if instID == "" {
		// Look up user's institution
		userID := c.GetString("user_id")
		_ = h.db.QueryRowContext(c.Request.Context(), `
			SELECT institution_id FROM institution_memberships WHERE user_id = $1 LIMIT 1
		`, userID).Scan(&instID)
	}

	groups, err := h.groupSvc.ListGroupsForInstitution(c.Request.Context(), instID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}

// JoinStudentGroup allows a student to join a peer group
func (h *PlatformHandler) JoinStudentGroup(c *gin.Context) {
	userID := c.GetString("user_id")
	groupID := c.Param("id")

	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID is required"})
		return
	}

	err := h.groupSvc.AddMember(c.Request.Context(), groupID, userID, "MEMBER")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join peer group: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully joined group!"})
}

