package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eterno/backend/internal/activity"
	"github.com/eterno/backend/internal/config"
	"github.com/eterno/backend/internal/database"
	"github.com/eterno/backend/internal/jobs"
	"github.com/eterno/backend/internal/platforms/leetcode"
	"github.com/eterno/backend/internal/scoring"
	"github.com/eterno/backend/internal/submissions"
	"github.com/eterno/backend/internal/verification"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Starting Eterno Background Verification Worker Daemon...")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Worker configuration validation failure", "error", err)
		os.Exit(1)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		slog.Error("Worker database connection failure", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize adapters & services
	lcAdapter := leetcode.NewAdapter(cfg.LeetCodeGraphQLEndpoint, cfg.MockExternalProviders)
	queue := jobs.NewQueue(db)
	scoringSvc := scoring.NewService(db)
	submissionSvc := submissions.NewService(db)
	verifSvc := verification.NewService(db, scoringSvc)
	actSvc := activity.NewService(db)

	worker := jobs.NewWorker(
		queue,
		db,
		lcAdapter,
		submissionSvc,
		verifSvc,
		actSvc,
		time.Duration(cfg.WorkerPollIntervalSeconds)*time.Second,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch worker
	go worker.Start(ctx)

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down Eterno Worker daemon...")
	cancel()
	time.Sleep(1 * time.Second)
	slog.Info("Worker daemon stopped cleanly")
}
