package handlers

import (
	"net/http"
	"strconv"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/analytics"
	"github.com/eterno/backend/internal/audit"
	"github.com/eterno/backend/internal/groups"
	"github.com/eterno/backend/internal/institutions"
	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsSvc   *analytics.Service
	activitySvc    *activity.Service
	institutionSvc *institutions.Service
	groupSvc       *groups.Service
	auditSvc       *audit.Service
}

func NewAnalyticsHandler(
	analyticsSvc *analytics.Service,
	activitySvc *activity.Service,
	institutionSvc *institutions.Service,
	groupSvc *groups.Service,
	auditSvc *audit.Service,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsSvc:   analyticsSvc,
		activitySvc:    activitySvc,
		institutionSvc: institutionSvc,
		groupSvc:       groupSvc,
		auditSvc:       auditSvc,
	}
}

func (h *AnalyticsHandler) GetAdminDashboard(c *gin.Context) {
	instID := c.GetString("institution_id")
	metrics, err := h.analyticsSvc.GetAdminOverview(c.Request.Context(), instID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) ListStudents(c *gin.Context) {
	instID := c.GetString("institution_id")
	search := c.Query("search")
	deptID := c.Query("department_id")
	batchID := c.Query("batch_id")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	filter := institutions.StudentFilter{
		InstitutionID: instID,
		SearchQuery:   search,
		Limit:         limit,
		Offset:        offset,
	}
	if deptID != "" {
		filter.DepartmentID = &deptID
	}
	if batchID != "" {
		filter.BatchID = &batchID
	}

	students, total, err := h.institutionSvc.ListStudents(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"students": students,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *AnalyticsHandler) ImportStudentsCSV(c *gin.Context) {
	instID := c.GetString("institution_id")
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid CSV file"})
		return
	}
	defer file.Close()

	result, err := h.institutionSvc.ImportStudentsCSV(c.Request.Context(), instID, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AnalyticsHandler) ListAuditLogs(c *gin.Context) {
	instID := c.GetString("institution_id")
	logs, err := h.auditSvc.List(c.Request.Context(), instID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *AnalyticsHandler) GetStudentActivity(c *gin.Context) {
	userID := c.GetString("user_id")
	yearStr := c.DefaultQuery("year", strconv.Itoa(2026))
	year, _ := strconv.Atoi(yearStr)
	source := c.Query("source")

	heatmap, err := h.activitySvc.GetUserHeatmap(c.Request.Context(), userID, year, source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, heatmap)
}

func (h *AnalyticsHandler) GetStudentHeatmap(c *gin.Context) {
	h.GetStudentActivity(c)
}

func (h *AnalyticsHandler) GetStudentStreaks(c *gin.Context) {
	userID := c.GetString("user_id")
	streaks, err := h.analyticsSvc.GetUserStreakAnalytics(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, streaks)
}

func (h *AnalyticsHandler) ListGroups(c *gin.Context) {
	instID := c.Param("institution_id")
	if instID == "" {
		instID = c.GetString("institution_id")
	}

	list, err := h.groupSvc.ListGroupsForInstitution(c.Request.Context(), instID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *AnalyticsHandler) CreateGroup(c *gin.Context) {
	instID := c.GetString("institution_id")
	userID := c.GetString("user_id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		GroupType   string `json:"group_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g, err := h.groupSvc.CreateGroup(c.Request.Context(), instID, req.Name, req.Description, req.GroupType, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, g)
}
