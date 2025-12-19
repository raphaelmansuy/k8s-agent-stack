// Package main is the entry point for the AgentStack background worker.
/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/telemetry"
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

	// Initialize Telemetry
	otelSvc, err := telemetry.NewFromConfig(context.Background(), cfg.Telemetry, "production")
	if err != nil {
		log.Warn("failed to initialize telemetry", zap.Error(err))
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = otelSvc.Shutdown(shutdownCtx)
		}()
	}

	// Initialize database connection
	dbPool, err := database.NewPool(context.Background(), cfg.Database.URL, otelSvc)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	// Initialize Redis client
	redisClient, err := cache.NewRedisClient(cfg.Redis.URL, otelSvc)
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
