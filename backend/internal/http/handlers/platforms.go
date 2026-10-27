package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/jobs"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/platforms/leetcode"
	"github.com/eterno/backend/internal/submissions"
	"github.com/eterno/backend/internal/verification"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PlatformHandler struct {
	db           *sql.DB
	queue        *jobs.Queue
	lcAdapter    *leetcode.Adapter
	subService   *submissions.Service
	verifService *verification.Service
	actService   *activity.Service
}

func NewPlatformHandler(
	db *sql.DB,
	queue *jobs.Queue,
	lcAdapter *leetcode.Adapter,
	subService *submissions.Service,
	verifService *verification.Service,
	actService *activity.Service,
) *PlatformHandler {
	return &PlatformHandler{
		db:           db,
		queue:        queue,
		lcAdapter:    lcAdapter,
		subService:   subService,
		verifService: verifService,
		actService:   actService,
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
