// Package main is the entry point for the AgentStack API Gateway.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/raphaelmansuy/agentstack/internal/api/handlers"
	"github.com/raphaelmansuy/agentstack/internal/config"
	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
	"github.com/raphaelmansuy/agentstack/internal/pkg/logger"
	"github.com/raphaelmansuy/agentstack/internal/pkg/telemetry"
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

	log.Info("Starting AgentStack API Gateway",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("build_time", buildTime),
	)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load configuration", zap.Error(err))
	}

	// Initialize OpenTelemetry
	shutdown, err := telemetry.InitTracer(cfg.Telemetry)
	if err != nil {
		log.Warn("failed to initialize telemetry", zap.Error(err))
	} else {
		defer func() { _ = shutdown(context.Background()) }()
	}

	// Initialize database connection
	db, err := database.NewPool(context.Background(), cfg.Database.URL)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis client
	redisClient, err := cache.NewRedisClient(cfg.Redis.URL)
	if err != nil {
		log.Warn("failed to connect to Redis", zap.Error(err))
		// Redis is optional for development, continue without it
	} else {
		defer func() { _ = redisClient.Close() }()
	}

	// Create Chi router
	router := chi.NewRouter()

	// Apply middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	// Create Huma API
	api := humachi.New(router, huma.DefaultConfig("AgentStack API", version))

	// Configure OpenAPI
	api.OpenAPI().Info.Description = "Sovereign GenAI Agent Platform API"
	api.OpenAPI().Info.Contact = &huma.Contact{
		Name:  "AgentStack Team",
		Email: "team@agentstack.dev",
	}
	api.OpenAPI().Info.License = &huma.License{
		Name: "Apache-2.0",
		URL:  "https://www.apache.org/licenses/LICENSE-2.0",
	}

	// Initialize A2A service
	a2aService := a2a.NewService()

	// Register routes
	handlers.RegisterHealthRoutes(api)
	handlers.RegisterAgentRoutes(api, db)
	handlers.RegisterProjectRoutes(api, db)
	handlers.RegisterChatRoutes(api, db, redisClient, a2aService)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Info("Server listening",
			zap.Int("port", cfg.Server.Port),
			zap.String("env", cfg.Environment),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exited gracefully")
}
