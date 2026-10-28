package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/analytics"
	"github.com/eterno/backend/internal/audit"
	"github.com/eterno/backend/internal/challenges"
	"github.com/eterno/backend/internal/config"
	"github.com/eterno/backend/internal/database"
	"github.com/eterno/backend/internal/groups"
	eternoHttp "github.com/eterno/backend/internal/http"
	"github.com/eterno/backend/internal/institutions"
	"github.com/eterno/backend/internal/jobs"
	"github.com/eterno/backend/internal/leaderboards"
	"github.com/eterno/backend/internal/platforms/codeforces"
	"github.com/eterno/backend/internal/platforms/github"
	"github.com/eterno/backend/internal/platforms/leetcode"
	"github.com/eterno/backend/internal/scoring"
	"github.com/eterno/backend/internal/submissions"
	"github.com/eterno/backend/internal/verification"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Initializing Eterno Backend Service...")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Configuration validation failure", "error", err)
		os.Exit(1)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		slog.Error("Database connection failure", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize adapters
	lcAdapter := leetcode.NewAdapter(cfg.LeetCodeGraphQLEndpoint, cfg.MockExternalProviders)
	cfAdapter := codeforces.NewAdapter(cfg.CodeforcesAPIKey, cfg.CodeforcesAPISecret, cfg.MockExternalProviders)
	ghAdapter := github.NewAdapter(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.MockExternalProviders)
	if cfg.GitHubRedirectURL != "" {
		ghAdapter.SetRedirectURL(cfg.GitHubRedirectURL)
	}

	// Initialize domain services
	queue := jobs.NewQueue(db)
	scoringSvc := scoring.NewService(db)
	submissionSvc := submissions.NewService(db)
	actSvc := activity.NewService(db)
	verifSvc := verification.NewService(db, scoringSvc)
	challengeSvc := challenges.NewService(db)
	leaderboardSvc := leaderboards.NewService(db)
	analyticsSvc := analytics.NewService(db)
	instSvc := institutions.NewService(db)
	groupSvc := groups.NewService(db)
	auditSvc := audit.NewService(db)

	worker := jobs.NewWorker(queue, db, lcAdapter, submissionSvc, verifSvc, actSvc, time.Duration(cfg.WorkerPollIntervalSeconds)*time.Second)

	services := &eternoHttp.Services{
		Config:          cfg,
		DB:              db,
		Queue:           queue,
		Worker:          worker,
		LCAdapter:       lcAdapter,
		CFAdapter:       cfAdapter,
		GHAdapter:       ghAdapter,
		ChallengeSvc:    challengeSvc,
		SubmissionSvc:   submissionSvc,
		ScoringSvc:      scoringSvc,
		VerificationSvc: verifSvc,
		LeaderboardSvc:  leaderboardSvc,
		AnalyticsSvc:    analyticsSvc,
		ActivitySvc:     actSvc,
		InstitutionSvc:  instSvc,
		GroupSvc:        groupSvc,
		AuditSvc:        auditSvc,
	}

	router := eternoHttp.SetupRouter(services)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Eterno REST API daemon listening", "port", cfg.Port, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server exited unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Initiating graceful API shutdown...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}

	slog.Info("Eterno API daemon terminated cleanly")
}
