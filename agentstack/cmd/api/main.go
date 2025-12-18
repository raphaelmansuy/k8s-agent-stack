// Package main is the entry point for the AgentStack API Gateway.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/raphaelmansuy/agentstack/internal/api/handlers"
	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/config"
	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
	"github.com/raphaelmansuy/agentstack/internal/domain/auth"
	"github.com/raphaelmansuy/agentstack/internal/domain/deployment"
	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
	"github.com/raphaelmansuy/agentstack/internal/domain/quota"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/idgen"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/mlflow"
	infraTelemetry "github.com/raphaelmansuy/agentstack/internal/infrastructure/telemetry"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/worker"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	otelSvc, err := infraTelemetry.NewFromConfig(ctx, cfg.Telemetry, cfg.Environment)
	if err != nil {
		log.Warn("failed to initialize telemetry", zap.Error(err))
	} else {
		defer func() { _ = otelSvc.Shutdown(context.Background()) }()
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
		log.Warn("failed to connect to Redis", zap.Error(err))
		// Redis is optional for development, continue without it
	} else {
		defer func() { _ = redisClient.Close() }()
	}

	// Create Chi router
	router := chi.NewRouter()

	// Apply middleware
	router.Use(chiMiddleware.RequestID)
	router.Use(chiMiddleware.RealIP)
	router.Use(middleware.Logger(log))
	router.Use(chiMiddleware.Recoverer)
	router.Use(chiMiddleware.Timeout(60 * time.Second))
	router.Use(otelSvc.HTTPMiddleware())
	router.Use(telemetry.MetricsMiddleware)

	// Initialize Database Queries
	queries := db.New(dbPool.NewTenantDB())

	// Initialize Repositories
	auditRepo := database.NewAuditRepository(queries)
	quotaRepo := database.NewQuotaRepository(queries)
	rbacRepo := database.NewRBACRepository(queries)
	evalRepo := database.NewEvaluationRepository(queries)
	apiKeyRepo := database.NewAPIKeyRepository(dbPool)

	// Initialize Caches/Queues
	var quotaCache *cache.QuotaCache
	var rbacCache *cache.RBACCache
	var evalQueue *cache.EvaluationQueue
	if redisClient != nil {
		quotaCache = cache.NewQuotaCache(redisClient)
		rbacCache = cache.NewRBACCache(redisClient)
		evalQueue = cache.NewEvaluationQueue(redisClient)
	}

	// Initialize Services
	auditService := audit.NewService(auditRepo)
	quotaService := quota.NewService(quotaRepo, quotaCache)
	rbacService := rbac.NewService(rbacRepo, rbacCache)
	authService := auth.NewService(apiKeyRepo)

	// Initialize API Key lookup
	apiKeyLookup := func(ctx context.Context, keyHash string) (*middleware.APIKeyInfo, error) {
		key, err := authService.VerifyKey(ctx, keyHash)
		if err != nil {
			return nil, err
		}
		return &middleware.APIKeyInfo{
			TeamID:    key.TeamID,
			ProjectID: key.ProjectID,
			Scopes:    key.Scopes,
		}, nil
	}

	// Initialize Middlewares
	auditMiddleware := middleware.NewAuditMiddleware(auditService)

	// Apply global middleware
	router.Use(middleware.Auth(middleware.AuthConfig{
		JWTSecret:    cfg.Auth.JWTSecret,
		APIKeyLookup: apiKeyLookup,
	}))
	router.Use(dbPool.TenantMiddleware())
	router.Use(auditMiddleware.RequestLogger())

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

	// Initialize Deployment service
	deploymentLogger := zap.NewStdLog(log).Writer()
	// Note: In a real production app, we'd use a more sophisticated logger adapter
	// but for now we'll use a simple slog logger for the deployment service
	slogLogger := slog.New(slog.NewJSONHandler(deploymentLogger, nil))
	deploymentService := deployment.NewService(slogLogger)

	// Initialize MLflow
	mlflowClient := mlflow.NewClient(cfg.MLflow.URL)
	mlflowAdapter := mlflow.NewMLflowAdapter(mlflowClient)

	// Initialize ID Generator
	idGenerator := idgen.NewUUIDGenerator()

	// Initialize Services that depend on MLflow/IDGen
	evaluationService := evaluation.NewService(mlflowAdapter, evalQueue, evalRepo, idGenerator)

	// Initialize remaining Middlewares
	rbacMiddleware := middleware.NewRBACMiddleware(api, rbacService)
	quotaMiddleware := middleware.NewQuotaMiddleware(api, quotaService)

	// Register routes
	router.Handle("/metrics", telemetry.MetricsHandler())
	handlers.RegisterHealthRoutes(api)
	handlers.RegisterHealthRoutesWithDeps(api, dbPool, redisClient)
	handlers.RegisterAgentRoutes(api, dbPool, rbacMiddleware, quotaMiddleware, auditMiddleware)
	handlers.RegisterProjectRoutes(api, dbPool, rbacMiddleware, auditMiddleware)
	handlers.RegisterChatRoutes(api, dbPool, redisClient, a2aService, evaluationService, rbacMiddleware, quotaMiddleware, auditMiddleware)

	// Register Deployment routes
	handlers.RegisterDeploymentRoutes(api, deploymentService, rbacMiddleware, quotaMiddleware, auditMiddleware)

	// Register A2A routes
	handlers.RegisterA2ARoutes(api, a2aService, rbacMiddleware, auditMiddleware)

	// Register Enterprise routes
	handlers.RegisterAuditRoutes(api, auditService, rbacMiddleware)
	handlers.RegisterQuotaRoutes(api, quotaService, rbacMiddleware)
	handlers.RegisterRBACRoutes(api, rbacService, rbacMiddleware, auditMiddleware)
	handlers.RegisterEvaluationRoutes(api, evaluationService, rbacMiddleware, auditMiddleware)
	handlers.RegisterAPIKeyRoutes(api, authService, rbacMiddleware, auditMiddleware)

	// Start Evaluation Worker if Redis is available
	if redisClient != nil {
		evalWorker := worker.NewEvaluationWorker(redisClient.GetRDB(), evaluationService, log)
		go evalWorker.Start(ctx)
	}

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
	cancel() // Stop background workers

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exited gracefully")
}
