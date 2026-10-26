package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/eterno/backend/internal/audit"
	"github.com/eterno/backend/internal/auth"
	"github.com/eterno/backend/internal/config"
	"github.com/eterno/backend/internal/database"
	"github.com/eterno/backend/internal/models"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	db       *sql.DB
	cfg      *config.Config
	auditSvc *audit.Service
}

func NewAuthHandler(db *sql.DB, cfg *config.Config, auditSvc *audit.Service) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg, auditSvc: auditSvc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := req.Role
	if role == "" {
		role = models.RoleStudent
	}

	var inst models.Institution
	queryInst := `SELECT id, name, slug, domain, timezone FROM institutions WHERE slug = $1`
	err := h.db.QueryRowContext(c.Request.Context(), queryInst, req.InstitutionSlug).Scan(
		&inst.ID, &inst.Name, &inst.Slug, &inst.Domain, &inst.Timezone,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Institution not found with provided slug"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var newUser models.User
	var departmentID, batchID *string

	err = database.WithTx(c.Request.Context(), h.db, func(tx *sql.Tx) error {
		// Insert User
		userQuery := `
			INSERT INTO users (email, password_hash, display_name, role)
			VALUES ($1, $2, $3, $4)
			RETURNING id, email, display_name, role, is_active, created_at, updated_at
		`
		err := tx.QueryRowContext(c.Request.Context(), userQuery,
			strings.ToLower(req.Email), hash, req.DisplayName, role,
		).Scan(
			&newUser.ID, &newUser.Email, &newUser.DisplayName, &newUser.Role,
			&newUser.IsActive, &newUser.CreatedAt, &newUser.UpdatedAt,
		)
		if err != nil {
			return err
		}

		// Department
		var dID sql.NullString
		if req.DepartmentName != "" {
			dQuery := `
				INSERT INTO departments (institution_id, name)
				VALUES ($1, $2)
				ON CONFLICT (institution_id, name) DO UPDATE SET name = EXCLUDED.name
				RETURNING id
			`
			_ = tx.QueryRowContext(c.Request.Context(), dQuery, inst.ID, req.DepartmentName).Scan(&dID)
			if dID.Valid {
				departmentID = &dID.String
			}
		}

		// Batch
		var bID sql.NullString
		if req.BatchName != "" {
			bQuery := `
				INSERT INTO batches (institution_id, name, graduation_year)
				VALUES ($1, $2, 2026)
				ON CONFLICT (institution_id, name) DO UPDATE SET name = EXCLUDED.name
				RETURNING id
			`
			_ = tx.QueryRowContext(c.Request.Context(), bQuery, inst.ID, req.BatchName).Scan(&bID)
			if bID.Valid {
				batchID = &bID.String
			}
		}

		// Membership
		memQuery := `
			INSERT INTO institution_memberships (
				institution_id, user_id, department_id, batch_id, student_id_number, role
			) VALUES ($1, $2, $3, $4, $5, $6)
		`
		var sid sql.NullString
		if req.StudentIDNumber != "" {
			sid.String = req.StudentIDNumber
			sid.Valid = true
		}
		var didParam interface{} = nil
		if departmentID != nil {
			didParam = *departmentID
		}
		var bidParam interface{} = nil
		if batchID != nil {
			bidParam = *batchID
		}

		_, err = tx.ExecContext(c.Request.Context(), memQuery, inst.ID, newUser.ID, didParam, bidParam, sid, role)
		return err
	})

	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User registration failed (email may already be registered)"})
		return
	}

	token, err := auth.GenerateToken(h.cfg, &newUser, inst.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate authorization token"})
		return
	}

	c.JSON(http.StatusCreated, models.AuthResponse{
		Token:        token,
		User:         newUser,
		Institution:  &inst,
		DepartmentID: departmentID,
		BatchID:      batchID,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		SELECT u.id, u.email, u.password_hash, u.display_name, u.avatar_url, u.role, u.is_active, u.created_at, u.updated_at,
		       im.institution_id, i.name, i.slug, i.timezone, im.department_id, im.batch_id
		FROM users u
		LEFT JOIN institution_memberships im ON u.id = im.user_id AND im.is_active = TRUE
		LEFT JOIN institutions i ON im.institution_id = i.id
		WHERE u.email = $1
	`
	var user models.User
	var avatarURL, instID, instName, instSlug, instTZ, deptID, batchID sql.NullString

	err := h.db.QueryRowContext(c.Request.Context(), query, strings.ToLower(req.Email)).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &avatarURL, &user.Role,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&instID, &instName, &instSlug, &instTZ, &deptID, &batchID,
	)

	if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account has been deactivated"})
		return
	}

	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}

	var inst *models.Institution
	if instID.Valid {
		inst = &models.Institution{
			ID:       instID.String,
			Name:     instName.String,
			Slug:     instSlug.String,
			Timezone: instTZ.String,
		}
	}

	tokenInstID := ""
	if inst != nil {
		tokenInstID = inst.ID
	}

	token, err := auth.GenerateToken(h.cfg, &user, tokenInstID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token creation failure"})
		return
	}

	var dID, bID *string
	if deptID.Valid {
		dID = &deptID.String
	}
	if batchID.Valid {
		bID = &batchID.String
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		Token:        token,
		User:         user,
		Institution:  inst,
		DepartmentID: dID,
		BatchID:      bID,
	})
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	userID := c.GetString("user_id")

	query := `
		SELECT u.id, u.email, u.display_name, u.avatar_url, u.role, u.is_active, u.created_at, u.updated_at,
		       im.institution_id, i.name, i.slug, i.timezone,
		       d.id, d.name, b.id, b.name, im.student_id_number
		FROM users u
		LEFT JOIN institution_memberships im ON u.id = im.user_id AND im.is_active = TRUE
		LEFT JOIN institutions i ON im.institution_id = i.id
		LEFT JOIN departments d ON im.department_id = d.id
		LEFT JOIN batches b ON im.batch_id = b.id
		WHERE u.id = $1
	`
	var user models.User
	var avatarURL, instID, instName, instSlug, instTZ, deptID, deptName, batchID, batchName, sidNum sql.NullString

	err := h.db.QueryRowContext(c.Request.Context(), query, userID).Scan(
		&user.ID, &user.Email, &user.DisplayName, &avatarURL, &user.Role,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&instID, &instName, &instSlug, &instTZ, &deptID, &deptName, &batchID, &batchName, &sidNum,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}

	response := gin.H{
		"user": user,
	}
	if instID.Valid {
		response["institution"] = gin.H{
			"id":       instID.String,
			"name":     instName.String,
			"slug":     instSlug.String,
			"timezone": instTZ.String,
		}
	}
	if deptID.Valid {
		response["department"] = gin.H{
			"id":   deptID.String,
			"name": deptName.String,
		}
	}
	if batchID.Valid {
		response["batch"] = gin.H{
			"id":   batchID.String,
			"name": batchName.String,
		}
	}
	if sidNum.Valid {
		response["student_id_number"] = sidNum.String
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		DisplayName string  `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		UPDATE users
		SET display_name = COALESCE(NULLIF($1, ''), display_name),
		    avatar_url = COALESCE($2, avatar_url),
		    updated_at = NOW()
		WHERE id = $3
		RETURNING id, email, display_name, avatar_url, role, is_active, updated_at
	`
	var u models.User
	var avatar sql.NullString
	err := h.db.QueryRowContext(c.Request.Context(), query, req.DisplayName, req.AvatarURL, userID).Scan(
		&u.ID, &u.Email, &u.DisplayName, &avatar, &u.Role, &u.IsActive, &u.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}
	if avatar.Valid {
		u.AvatarURL = &avatar.String
	}

	c.JSON(http.StatusOK, u)
}

func (h *AuthHandler) GetNotifications(c *gin.Context) {
	userID := c.GetString("user_id")
	query := `
		SELECT id, user_id, title, message, type, link, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`
	rows, err := h.db.QueryContext(c.Request.Context(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var notifs []models.Notification
	for rows.Next() {
		var n models.Notification
		var link sql.NullString
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Type, &link, &n.IsRead, &n.CreatedAt); err == nil {
			if link.Valid {
				n.Link = &link.String
			}
			notifs = append(notifs, n)
		}
	}

	// Mark notifications as read
	_, _ = h.db.ExecContext(c.Request.Context(), `UPDATE notifications SET is_read = TRUE WHERE user_id = $1`, userID)

	c.JSON(http.StatusOK, notifs)
}

func (h *AuthHandler) ListInstitutions(c *gin.Context) {
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT id, name, slug, timezone FROM institutions ORDER BY name ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []gin.H
	for rows.Next() {
		var id, name, slug, tz string
		if err := rows.Scan(&id, &name, &slug, &tz); err == nil {
			list = append(list, gin.H{
				"id":       id,
				"name":     name,
				"slug":     slug,
				"timezone": tz,
			})
		}
	}
	c.JSON(http.StatusOK, list)
}
