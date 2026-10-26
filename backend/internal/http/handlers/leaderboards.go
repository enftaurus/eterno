package handlers

import (
	"net/http"
	"strconv"

	"github.com/eterno/backend/internal/leaderboards"
	"github.com/gin-gonic/gin"
)

type LeaderboardHandler struct {
	svc *leaderboards.Service
}

func NewLeaderboardHandler(svc *leaderboards.Service) *LeaderboardHandler {
	return &LeaderboardHandler{svc: svc}
}

func (h *LeaderboardHandler) GetLeaderboard(c *gin.Context) {
	instID := c.GetString("institution_id")
	timeframe := c.DefaultQuery("timeframe", "all")
	deptIDStr := c.Query("department_id")
	batchIDStr := c.Query("batch_id")
	groupIDStr := c.Query("group_id")
	challengeIDStr := c.Query("challenge_id")
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	filter := leaderboards.FilterParams{
		InstitutionID: instID,
		Timeframe:     timeframe,
		Limit:         limit,
	}

	if deptIDStr != "" {
		filter.DepartmentID = &deptIDStr
	}
	if batchIDStr != "" {
		filter.BatchID = &batchIDStr
	}
	if groupIDStr != "" {
		filter.GroupID = &groupIDStr
	}
	if challengeIDStr != "" {
		filter.ChallengeID = &challengeIDStr
	}

	entries, err := h.svc.GetLeaderboard(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, entries)
}
