package http

import (
	"database/sql"
	"net/http"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/analytics"
	"github.com/eterno/backend/internal/audit"
	"github.com/eterno/backend/internal/challenges"
	"github.com/eterno/backend/internal/config"
	"github.com/eterno/backend/internal/groups"
	"github.com/eterno/backend/internal/http/handlers"
	"github.com/eterno/backend/internal/institutions"
	"github.com/eterno/backend/internal/jobs"
	"github.com/eterno/backend/internal/leaderboards"
	"github.com/eterno/backend/internal/middleware"
	"github.com/eterno/backend/internal/models"
	"github.com/eterno/backend/internal/platforms/codeforces"
	"github.com/eterno/backend/internal/platforms/github"
	"github.com/eterno/backend/internal/platforms/leetcode"
	"github.com/eterno/backend/internal/scoring"
	"github.com/eterno/backend/internal/submissions"
	"github.com/eterno/backend/internal/verification"
	"github.com/gin-gonic/gin"
)

type Services struct {
	Config          *config.Config
	DB              *sql.DB
	Queue           *jobs.Queue
	Worker          *jobs.Worker
	LCAdapter       *leetcode.Adapter
	CFAdapter       *codeforces.Adapter
	GHAdapter       *github.Adapter
	ChallengeSvc    *challenges.Service
	SubmissionSvc   *submissions.Service
	ScoringSvc      *scoring.Service
	VerificationSvc *verification.Service
	LeaderboardSvc  *leaderboards.Service
	AnalyticsSvc    *analytics.Service
	ActivitySvc     *activity.Service
	InstitutionSvc  *institutions.Service
	GroupSvc        *groups.Service
	AuditSvc        *audit.Service
}

func SetupRouter(s *Services) *gin.Engine {
	if s.Config.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(s.Config.CORSAllowedOrigins))

	// Health and readiness endpoints
	r.GET("/healthz", func(c *gin.Context) {
		if err := s.DB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "db": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "db": "up"})
	})
	r.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	api := r.Group("/api/v1")
	{
		// Handlers instantiation
		authH := handlers.NewAuthHandler(s.DB, s.Config, s.AuditSvc)
		challengeH := handlers.NewChallengeHandler(s.ChallengeSvc, s.AuditSvc)
		platformH := handlers.NewPlatformHandler(s.DB, s.Config, s.Queue, s.LCAdapter, s.CFAdapter, s.GHAdapter, s.SubmissionSvc, s.VerificationSvc, s.ActivitySvc, s.ScoringSvc, s.GroupSvc)
		leaderboardH := handlers.NewLeaderboardHandler(s.LeaderboardSvc)
		analyticsH := handlers.NewAnalyticsHandler(s.AnalyticsSvc, s.ActivitySvc, s.InstitutionSvc, s.GroupSvc, s.AuditSvc)

		// GitHub OAuth routes (available both publicly for browser redirect and within authProtected)
		api.GET("/platforms/github/authorize", platformH.GetGitHubOAuthURL)
		api.GET("/platforms/github/callback", platformH.HandleGitHubOAuthCallback)
		api.POST("/platforms/github/callback", platformH.ExchangeGitHubOAuthCode)
		api.POST("/platforms/github/exchange", platformH.ExchangeGitHubOAuthCode)

		// Public Auth routes
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authH.Register)
			authGroup.POST("/login", authH.Login)
			authGroup.GET("/institutions", authH.ListInstitutions)
		}

		// Authenticated routes
		authProtected := api.Group("")
		authProtected.Use(middleware.Auth(s.Config))
		{
			// User identity
			authProtected.GET("/me", authH.GetMe)
			authProtected.PATCH("/me/profile", authH.UpdateProfile)
			authProtected.GET("/me/notifications", authH.GetNotifications)

			// Student Challenge Progress
			authProtected.GET("/me/challenges", challengeH.GetStudentChallenges)
			authProtected.GET("/me/challenges/:id", challengeH.GetStudentChallengeDetail)
			authProtected.GET("/me/submissions", platformH.GetStudentSubmissions)

			// Account Connections & Platform Sync (LeetCode, Codeforces, GitHub)
			authProtected.POST("/me/platform-accounts/leetcode", platformH.LinkLeetCode)
			authProtected.POST("/me/platform-accounts/leetcode/verify", platformH.VerifyLeetCode)
			authProtected.POST("/me/platform-accounts/leetcode/sync", platformH.SyncLeetCode)
			authProtected.POST("/me/platform-accounts/codeforces", platformH.LinkCodeforces)
			authProtected.POST("/me/platform-accounts/codeforces/sync", platformH.SyncCodeforces)
			authProtected.POST("/me/platform-accounts/github", platformH.LinkGitHub)
			authProtected.POST("/me/platform-accounts/github/sync", platformH.SyncGitHub)
			authProtected.GET("/me/platform-accounts", platformH.ListPlatformAccounts)

			// Developer Score & Intelligence
			authProtected.GET("/me/developer-score", platformH.GetDeveloperScore)

			// Student Peer Groups (Code Crusaders Squads)
			authProtected.GET("/me/groups", platformH.ListStudentGroups)
			authProtected.POST("/me/groups/:id/join", platformH.JoinStudentGroup)

			// Activity Heatmap & Streaks
			authProtected.GET("/me/activity", analyticsH.GetStudentActivity)
			authProtected.GET("/me/heatmap", analyticsH.GetStudentHeatmap)
			authProtected.GET("/me/streaks", analyticsH.GetStudentStreaks)

			// Challenges (Read-only for all logged in)
			authProtected.GET("/challenges", challengeH.ListChallenges)
			authProtected.GET("/challenges/:id", challengeH.GetChallengeByID)

			// Leaderboards
			authProtected.GET("/leaderboards", leaderboardH.GetLeaderboard)

			// Groups Directory
			authProtected.GET("/institutions/:institution_id/groups", analyticsH.ListGroups)

			// College Administrator Only
			adminProtected := authProtected.Group("/admin")
			adminProtected.Use(middleware.RequireRoles(models.RoleCollegeAdmin, models.RolePlatformAdmin))
			{
				adminProtected.GET("/dashboard", analyticsH.GetAdminDashboard)
				adminProtected.GET("/students", analyticsH.ListStudents)
				adminProtected.POST("/students/import", analyticsH.ImportStudentsCSV)
				adminProtected.POST("/challenges", challengeH.CreateChallenge)
				adminProtected.POST("/challenges/:id/publish", challengeH.PublishChallenge)
				adminProtected.POST("/challenges/:id/cancel", challengeH.CancelChallenge)
				adminProtected.GET("/audit-logs", analyticsH.ListAuditLogs)
				adminProtected.POST("/groups", analyticsH.CreateGroup)
			}
		}
	}

	return r
}
