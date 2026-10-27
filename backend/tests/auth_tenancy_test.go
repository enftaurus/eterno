package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/analytics"
	"github.com/eterno/backend/internal/audit"
	"github.com/eterno/backend/internal/auth"
	"github.com/eterno/backend/internal/challenges"
	"github.com/eterno/backend/internal/config"
	"github.com/eterno/backend/internal/groups"
	eternoHttp "github.com/eterno/backend/internal/http"
	"github.com/eterno/backend/internal/institutions"
	"github.com/eterno/backend/internal/jobs"
	"github.com/eterno/backend/internal/leaderboards"
	"github.com/eterno/backend/internal/middleware"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/platforms/leetcode"
	"github.com/eterno/backend/internal/scoring"
	"github.com/eterno/backend/internal/submissions"
	"github.com/eterno/backend/internal/verification"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Tests 76-90: Authentication, Token Integrity, RBAC, and Multi-Tenant Isolation

func setupTestRouter(t *testing.T) (*gin.Engine, *config.Config) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	queue := jobs.NewQueue(db)
	scoringSvc := scoring.NewService(db)
	subSvc := submissions.NewService(db)
	actSvc := activity.NewService(db)
	verifSvc := verification.NewService(db, scoringSvc)
	chSvc := challenges.NewService(db)
	lbSvc := leaderboards.NewService(db)
	analyticsSvc := analytics.NewService(db)
	instSvc := institutions.NewService(db)
	groupSvc := groups.NewService(db)
	auditSvc := audit.NewService(db)
	lcAdapter := leetcode.NewAdapter("", true)

	worker := jobs.NewWorker(queue, db, lcAdapter, subSvc, verifSvc, actSvc, 1*time.Second)

	services := &eternoHttp.Services{
		Config:          cfg,
		DB:              db,
		Queue:           queue,
		Worker:          worker,
		LCAdapter:       lcAdapter,
		ChallengeSvc:    chSvc,
		SubmissionSvc:   subSvc,
		ScoringSvc:      scoringSvc,
		VerificationSvc: verifSvc,
		LeaderboardSvc:  lbSvc,
		AnalyticsSvc:    analyticsSvc,
		ActivitySvc:     actSvc,
		InstitutionSvc:  instSvc,
		GroupSvc:        groupSvc,
		AuditSvc:        auditSvc,
	}

	router := eternoHttp.SetupRouter(services)
	return router, cfg
}

func TestAuthMissingToken(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for missing token, got %d", w.Code)
	}
}

func TestAuthMalformedToken(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-malformed-token-garbage")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for malformed token, got %d", w.Code)
	}
}

func TestAuthExpiredToken(t *testing.T) {
	router, cfg := setupTestRouter(t)

	// Construct token expired 1 hour ago
	claims := &auth.Claims{
		UserID:        "expired-user-id",
		Email:         "expired@college.edu",
		Role:          models.RoleStudent,
		InstitutionID: "inst-test",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "eterno-api",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := tok.SignedString([]byte(cfg.JWTSecret))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for expired token, got %d", w.Code)
	}
}

func TestAuthTamperedSignature(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Sign token with a different, rogue secret
	claims := &auth.Claims{
		UserID:        "rogue-user-id",
		Email:         "rogue@attacker.com",
		Role:          models.RolePlatformAdmin,
		InstitutionID: "inst-test",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "eterno-api",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	rogueTokenStr, _ := tok.SignedString([]byte("completely-wrong-untrusted-secret-key-32chars!!"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+rogueTokenStr)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for invalid signature, got %d", w.Code)
	}
}

func TestAuthUnsupportedAlgorithmNone(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Create unsigned token (alg: none attack)
	claims := &auth.Claims{
		UserID:        "attacker-id",
		Email:         "attacker@domain.com",
		Role:          models.RolePlatformAdmin,
		InstitutionID: "inst-test",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, _ := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for alg=none, got %d", w.Code)
	}
}

func TestRBACStudentBlockedFromAdminRoutes(t *testing.T) {
	router, cfg := setupTestRouter(t)
	db := setupTestDB(t)
	defer db.Close()

	instID, studentID := createTestUserAndInst(t, db)

	studentUser := &models.User{
		ID:    studentID,
		Email: "student@test.edu",
		Role:  models.RoleStudent,
	}
	tokenStr, err := auth.GenerateToken(cfg, studentUser, instID)
	if err != nil {
		t.Fatalf("failed to generate student token: %v", err)
	}

	adminEndpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/dashboard"},
		{http.MethodGet, "/api/v1/admin/students"},
		{http.MethodPost, "/api/v1/admin/challenges"},
		{http.MethodGet, "/api/v1/admin/audit-logs"},
		{http.MethodPost, "/api/v1/admin/groups"},
	}

	for _, ep := range adminEndpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(ep.method, ep.path, nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("RBAC FAILURE: Student accessed admin route %s %s with status %d (expected 403 Forbidden)",
				ep.method, ep.path, w.Code)
		}
	}
}

func TestRBACCollegeAdminAllowedOnAdminRoutes(t *testing.T) {
	router, cfg := setupTestRouter(t)
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	instID, _ := createTestUserAndInst(t, db)

	// Create College Admin
	var adminID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, display_name, role)
		VALUES ('admin-' || gen_random_uuid()::text || '@college.edu', 'hash', 'Dean Admin', 'COLLEGE_ADMIN')
		RETURNING id
	`).Scan(&adminID)
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	adminUser := &models.User{
		ID:    adminID,
		Email: "admin@college.edu",
		Role:  models.RoleCollegeAdmin,
	}
	tokenStr, err := auth.GenerateToken(cfg, adminUser, instID)
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	// GET /api/v1/admin/dashboard
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for College Admin dashboard, got %d (body: %s)", w.Code, w.Body.String())
	}

	// GET /api/v1/admin/students
	wStudents := httptest.NewRecorder()
	reqStudents, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/students", nil)
	reqStudents.Header.Set("Authorization", "Bearer "+tokenStr)
	router.ServeHTTP(wStudents, reqStudents)

	if wStudents.Code != http.StatusOK {
		t.Errorf("expected 200 OK for College Admin students list, got %d (body: %s)", wStudents.Code, wStudents.Body.String())
	}
}

func TestMultiTenantChallengeDataIsolation(t *testing.T) {
	router, cfg := setupTestRouter(t)
	db := setupTestDB(t)
	defer db.Close()
	now := time.Now().UTC()

	// 1. Create Institution A and Student A
	instA, studentA := createTestUserAndInst(t, db)
	chA := createTestChallenge(t, db, instA, models.ChallengeActive, now.Add(-1*time.Hour), now.Add(2*time.Hour), false)

	// 2. Create Institution B and Student B
	instB, studentB := createTestUserAndInst(t, db)
	chB := createTestChallenge(t, db, instB, models.ChallengeActive, now.Add(-1*time.Hour), now.Add(2*time.Hour), false)

	// Generate Token for Student A (Institution A)
	tokA, _ := auth.GenerateToken(cfg, &models.User{ID: studentA, Role: models.RoleStudent}, instA)

	// Generate Token for Student B (Institution B)
	tokB, _ := auth.GenerateToken(cfg, &models.User{ID: studentB, Role: models.RoleStudent}, instB)

	// Student A queries challenges
	wA := httptest.NewRecorder()
	reqA, _ := http.NewRequest(http.MethodGet, "/api/v1/challenges", nil)
	reqA.Header.Set("Authorization", "Bearer "+tokA)
	router.ServeHTTP(wA, reqA)

	if wA.Code != http.StatusOK {
		t.Fatalf("Student A request failed with status %d", wA.Code)
	}

	var listA []map[string]interface{}
	_ = json.Unmarshal(wA.Body.Bytes(), &listA)

	for _, item := range listA {
		if item["id"] == chB {
			t.Errorf("TENANT LEAK: Student from Institution A saw Challenge %s belonging to Institution B!", chB)
		}
	}

	// Student B queries challenges
	wB := httptest.NewRecorder()
	reqB, _ := http.NewRequest(http.MethodGet, "/api/v1/challenges", nil)
	reqB.Header.Set("Authorization", "Bearer "+tokB)
	router.ServeHTTP(wB, reqB)

	if wB.Code != http.StatusOK {
		t.Fatalf("Student B request failed with status %d", wB.Code)
	}

	var listB []map[string]interface{}
	_ = json.Unmarshal(wB.Body.Bytes(), &listB)

	for _, item := range listB {
		if item["id"] == chA {
			t.Errorf("TENANT LEAK: Student from Institution B saw Challenge %s belonging to Institution A!", chA)
		}
	}
}

func TestEnsureTenantScopeMiddlewareTampering(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Handler that applies EnsureTenantScope
	testApp := gin.New()
	testApp.Use(func(c *gin.Context) {
		// Mock token auth setting context
		role := c.GetHeader("X-Mock-Role")
		instID := c.GetHeader("X-Mock-Inst")
		if role != "" {
			c.Set(middleware.ContextKeyRole, role)
		}
		if instID != "" {
			c.Set(middleware.ContextKeyInstitutionID, instID)
		}
		c.Next()
	})
	testApp.GET("/institutions/:institution_id/resource", middleware.EnsureTenantScope(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Case 1: Same institution -> Allowed
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/institutions/inst-123/resource", nil)
	req1.Header.Set("X-Mock-Role", models.RoleCollegeAdmin)
	req1.Header.Set("X-Mock-Inst", "inst-123")
	testApp.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("expected 200 OK for matched tenant, got %d", w1.Code)
	}

	// Case 2: Cross-institution parameter tampering -> 403 Forbidden
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/institutions/inst-TARGET-OTHER/resource", nil)
	req2.Header.Set("X-Mock-Role", models.RoleCollegeAdmin)
	req2.Header.Set("X-Mock-Inst", "inst-ATTACKER-OWN")
	testApp.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for cross-tenant access tampering, got %d", w2.Code)
	}

	// Case 3: Platform Admin override -> Allowed
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/institutions/inst-ANY/resource", nil)
	req3.Header.Set("X-Mock-Role", models.RolePlatformAdmin)
	req3.Header.Set("X-Mock-Inst", "inst-ANY")
	testApp.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("expected 200 OK for platform admin, got %d", w3.Code)
	}
}
