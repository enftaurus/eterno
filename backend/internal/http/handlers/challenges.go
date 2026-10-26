package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/eterno/backend/internal/audit"
	"github.com/eterno/backend/internal/challenges"
	"github.com/eterno/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type ChallengeHandler struct {
	svc      *challenges.Service
	auditSvc *audit.Service
}

func NewChallengeHandler(svc *challenges.Service, auditSvc *audit.Service) *ChallengeHandler {
	return &ChallengeHandler{svc: svc, auditSvc: auditSvc}
}

func (h *ChallengeHandler) ListChallenges(c *gin.Context) {
	instID := c.GetString("institution_id")
	statusFilter := c.Query("status")

	list, err := h.svc.ListChallengesForInstitution(c.Request.Context(), instID, statusFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *ChallengeHandler) GetChallengeByID(c *gin.Context) {
	id := c.Param("id")
	ch, err := h.svc.GetChallengeByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
		return
	}
	c.JSON(http.StatusOK, ch)
}

func (h *ChallengeHandler) CreateChallenge(c *gin.Context) {
	instID := c.GetString("institution_id")
	userID := c.GetString("user_id")

	var req models.CreateChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch, err := h.svc.CreateChallenge(c.Request.Context(), instID, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = h.auditSvc.Log(c.Request.Context(), &instID, &userID, "CREATE_CHALLENGE", "challenge", ch.ID, map[string]interface{}{
		"title":    ch.Title,
		"deadline": ch.Deadline,
	}, nil)

	c.JSON(http.StatusCreated, ch)
}

func (h *ChallengeHandler) PublishChallenge(c *gin.Context) {
	instID := c.GetString("institution_id")
	userID := c.GetString("user_id")
	id := c.Param("id")

	ch, err := h.svc.PublishChallenge(c.Request.Context(), instID, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = h.auditSvc.Log(c.Request.Context(), &instID, &userID, "PUBLISH_CHALLENGE", "challenge", ch.ID, map[string]interface{}{
		"title": ch.Title,
	}, nil)

	c.JSON(http.StatusOK, ch)
}

func (h *ChallengeHandler) CancelChallenge(c *gin.Context) {
	instID := c.GetString("institution_id")
	userID := c.GetString("user_id")
	id := c.Param("id")

	ch, err := h.svc.GetChallengeByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
		return
	}

	_ = h.auditSvc.Log(c.Request.Context(), &instID, &userID, "CANCEL_CHALLENGE", "challenge", ch.ID, nil, nil)
	c.JSON(http.StatusOK, gin.H{"status": "CANCELLED", "id": id})
}

func (h *ChallengeHandler) GetStudentChallenges(c *gin.Context) {
	userID := c.GetString("user_id")
	assignments, err := h.svc.GetStudentAssignments(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, assignments)
}

func (h *ChallengeHandler) GetStudentChallengeDetail(c *gin.Context) {
	userID := c.GetString("user_id")
	challengeID := c.Param("id")

	ch, err := h.svc.GetChallengeByID(c.Request.Context(), challengeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
		return
	}

	// Fetch assignment for student
	var assign models.ChallengeAssignment
	var completedAt sql.NullTime
	query := `
		SELECT id, challenge_id, user_id, institution_id, status,
		       baseline_at, assigned_at, completed_at, points_awarded, created_at, updated_at
		FROM challenge_assignments
		WHERE challenge_id = $1 AND user_id = $2
	`
	err = h.svc.DB().QueryRowContext(c.Request.Context(), query, challengeID, userID).Scan(
		&assign.ID, &assign.ChallengeID, &assign.UserID, &assign.InstitutionID, &assign.Status,
		&assign.BaselineAt, &assign.AssignedAt, &completedAt, &assign.PointsAwarded,
		&assign.CreatedAt, &assign.UpdatedAt,
	)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if completedAt.Valid {
		assign.CompletedAt = &completedAt.Time
	}

	// For each problem, check if verified for this student
	type ProblemWithStatus struct {
		models.ChallengeProblem
		IsVerified   bool       `json:"is_verified"`
		VerifiedAt   *time.Time `json:"verified_at,omitempty"`
		SubmissionID *string    `json:"submission_id,omitempty"`
	}

	var problemsWithStatus []ProblemWithStatus
	for _, p := range ch.Problems {
		var compID, subID sql.NullString
		var vAt sql.NullTime
		_ = h.svc.DB().QueryRowContext(c.Request.Context(), `
			SELECT id, submission_id, verified_at
			FROM challenge_completions
			WHERE challenge_problem_id = $1 AND user_id = $2
		`, p.ID, userID).Scan(&compID, &subID, &vAt)

		pws := ProblemWithStatus{
			ChallengeProblem: p,
			IsVerified:       compID.Valid,
		}
		if vAt.Valid {
			pws.VerifiedAt = &vAt.Time
		}
		if subID.Valid {
			pws.SubmissionID = &subID.String
		}
		problemsWithStatus = append(problemsWithStatus, pws)
	}

	c.JSON(http.StatusOK, gin.H{
		"challenge":  ch,
		"assignment": assign,
		"problems":   problemsWithStatus,
	})
}
