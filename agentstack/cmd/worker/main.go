// Package main is the entry point for the AgentStack background worker.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/raphaelmansuy/agentstack/internal/config"
	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/idgen"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/mlflow"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/worker"
	"github.com/raphaelmansuy/agentstack/internal/pkg/logger"
)

// Version information (set via ldflags during build).
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	// Initialize logger
	log, err := logger.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	log.Info("Starting AgentStack Worker",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("build_time", buildTime),
	)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load configuration", zap.Error(err))
	}

	// Initialize database connection
	dbPool, err := database.NewPool(context.Background(), cfg.Database.URL)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	// Initialize Redis client
	redisClient, err := cache.NewRedisClient(cfg.Redis.URL)
	if err != nil {
		log.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer func() { _ = redisClient.Close() }()

	// Initialize Database Queries
	queries := db.New(dbPool)

	// Initialize Repositories
	evalRepo := database.NewEvaluationRepository(queries)

	// Initialize Caches/Queues
	evalQueue := cache.NewEvaluationQueue(redisClient)

	// Initialize MLflow
	mlflowClient := mlflow.NewClient(cfg.MLflow.URL)
	mlflowAdapter := mlflow.NewMLflowAdapter(mlflowClient)

	// Initialize ID Generator
	idGenerator := idgen.NewUUIDGenerator()

	// Initialize Evaluation Service
	evaluationService := evaluation.NewService(mlflowAdapter, evalQueue, evalRepo, idGenerator)

	// Initialize and start Evaluation Worker
	evalWorker := worker.NewEvaluationWorker(redisClient.GetRDB(), evaluationService, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker in goroutine
	go evalWorker.Start(ctx)

	log.Info("Worker started successfully")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down worker...")
	cancel()

	// Give worker some time to finish current task
	time.Sleep(2 * time.Second)

	log.Info("Worker exited gracefully")
}
