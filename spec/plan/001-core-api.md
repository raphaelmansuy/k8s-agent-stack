# Phase 1: Core API Gateway

> REST API Implementation, Authentication, Multi-Tenancy Foundation

**Duration**: 3 weeks | **Status**: Not Started | **Priority**: Critical  
**Depends On**: Phase 0 (Foundation)

---

## Objectives

1. Implement core REST API endpoints (agents, projects, health)
2. Implement authentication (API keys, JWT)
3. Implement multi-tenancy (team/project isolation)
4. Implement rate limiting and request validation
5. Implement OpenTelemetry instrumentation

---

## Architecture Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                         API Gateway Architecture                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      HTTP Server (Fiber/Chi)                     │   │
│  │  Port: 8080                                                      │   │
│  └───────────────────────────────┬─────────────────────────────────┘   │
│                                  │                                      │
│  ┌───────────────────────────────▼─────────────────────────────────┐   │
│  │                      Middleware Stack                            │   │
│  │  1. Request ID │ 2. Logger │ 3. Recover │ 4. CORS │ 5. Timeout  │   │
│  │  6. Auth │ 7. RateLimit │ 8. Project Context │ 9. Tracing       │   │
│  └───────────────────────────────┬─────────────────────────────────┘   │
│                                  │                                      │
│  ┌───────────────────────────────▼─────────────────────────────────┐   │
│  │                         Route Groups                             │   │
│  │                                                                  │   │
│  │  /health          → Health handlers (no auth)                   │   │
│  │  /v1/projects     → Project CRUD                                │   │
│  │  /v1/agents       → Agent CRUD                                  │   │
│  │  /v1/api-keys     → API key management                          │   │
│  │  /v1/admin        → Admin endpoints (superuser)                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Week 1: HTTP Server & Middleware

### 1.1 Server Setup

**`cmd/api/main.go`**:
```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/api"
	"github.com/raphaelmansuy/agentstack/internal/config"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
	"github.com/raphaelmansuy/agentstack/internal/pkg/logger"
	"github.com/raphaelmansuy/agentstack/internal/pkg/telemetry"
)

var version = "dev"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel, cfg.Environment)
	defer log.Sync()

	log.Info("starting agentstack api",
		"version", version,
		"environment", cfg.Environment,
	)

	// Initialize telemetry
	shutdown, err := telemetry.Init(cfg.OTelEndpoint, "agentstack-api", version)
	if err != nil {
		log.Error("failed to init telemetry", "error", err)
	}
	defer shutdown(context.Background())

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("failed to connect to database", "error", err)
	}
	defer db.Close()

	// Connect to Redis
	redisClient, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatal("failed to connect to redis", "error", err)
	}
	defer redisClient.Close()

	// Create and start server
	server := api.NewServer(cfg, log, db, redisClient)

	// Graceful shutdown
	go func() {
		if err := server.Start(cfg.Port); err != nil {
			log.Error("server error", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "error", err)
	}

	log.Info("server stopped")
}
```

### 1.2 Server Configuration

**`internal/api/server.go`**:
```go
package api

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/raphaelmansuy/agentstack/internal/api/handlers"
	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/config"
	"github.com/raphaelmansuy/agentstack/internal/domain/agent"
	"github.com/raphaelmansuy/agentstack/internal/domain/project"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
	"github.com/raphaelmansuy/agentstack/internal/pkg/logger"
)

type Server struct {
	app    *fiber.App
	config *config.Config
	log    *logger.Logger
}

func NewServer(cfg *config.Config, log *logger.Logger, db *database.DB, redis *cache.RedisClient) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "AgentStack API",
		DisableStartupMessage: cfg.Environment == "production",
		ErrorHandler:          errorHandler,
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
	})

	// Repositories
	projectRepo := database.NewProjectRepository(db)
	agentRepo := database.NewAgentRepository(db)
	apiKeyRepo := database.NewAPIKeyRepository(db)

	// Services
	projectSvc := project.NewService(projectRepo)
	agentSvc := agent.NewService(agentRepo, projectRepo)

	// Auth service
	authSvc := middleware.NewAuthService(apiKeyRepo, redis, cfg.JWTSecret)

	// Handlers
	healthHandler := handlers.NewHealthHandler(db, redis)
	projectHandler := handlers.NewProjectHandler(projectSvc, log)
	agentHandler := handlers.NewAgentHandler(agentSvc, log)
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeyRepo, log)

	// Global middleware
	app.Use(requestid.New())
	app.Use(middleware.Logger(log))
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-API-Key,X-Project-ID,X-Request-ID,Idempotency-Key",
		AllowCredentials: true,
	}))
	app.Use(middleware.Tracing())

	// Health endpoints (no auth)
	app.Get("/health", healthHandler.Health)
	app.Get("/health/ready", healthHandler.Ready)
	app.Get("/health/live", healthHandler.Live)

	// API v1 group
	v1 := app.Group("/v1")

	// Rate limiting (after auth to use user context)
	v1.Use(middleware.RateLimit(redis, cfg.RateLimitPerMinute))

	// Authenticated routes
	v1.Use(middleware.Auth(authSvc))

	// Project context middleware
	v1.Use(middleware.ProjectContext(projectSvc))

	// Projects
	projects := v1.Group("/projects")
	projects.Get("/", projectHandler.List)
	projects.Post("/", projectHandler.Create)
	projects.Get("/:projectId", projectHandler.Get)
	projects.Patch("/:projectId", projectHandler.Update)
	projects.Delete("/:projectId", projectHandler.Delete)

	// Agents
	agents := v1.Group("/agents")
	agents.Get("/", agentHandler.List)
	agents.Post("/", agentHandler.Create)
	agents.Get("/:agentId", agentHandler.Get)
	agents.Patch("/:agentId", agentHandler.Update)
	agents.Delete("/:agentId", agentHandler.Delete)

	// API Keys
	apiKeys := v1.Group("/api-keys")
	apiKeys.Get("/", apiKeyHandler.List)
	apiKeys.Post("/", apiKeyHandler.Create)
	apiKeys.Delete("/:keyId", apiKeyHandler.Revoke)

	return &Server{
		app:    app,
		config: cfg,
		log:    log,
	}
}

func (s *Server) Start(port int) error {
	return s.app.Listen(fmt.Sprintf(":%d", port))
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}
```

### 1.3 Middleware Stack

**`internal/api/middleware/auth.go`**:
```go
package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/raphaelmansuy/agentstack/internal/domain/apikey"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
)

type AuthService struct {
	apiKeyRepo apikey.Repository
	cache      *cache.RedisClient
	jwtSecret  []byte
}

type AuthContext struct {
	TeamID    string
	UserID    string
	ProjectID string
	Scopes    []string
	AuthType  string // "api_key" or "jwt"
}

const AuthContextKey = "auth"

func NewAuthService(repo apikey.Repository, cache *cache.RedisClient, jwtSecret string) *AuthService {
	return &AuthService{
		apiKeyRepo: repo,
		cache:      cache,
		jwtSecret:  []byte(jwtSecret),
	}
}

func Auth(authSvc *AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try API Key first
		apiKey := c.Get("X-API-Key")
		if apiKey != "" {
			authCtx, err := authSvc.validateAPIKey(c.Context(), apiKey)
			if err != nil {
				return fiber.NewError(fiber.StatusUnauthorized, "invalid API key")
			}
			c.Locals(AuthContextKey, authCtx)
			return c.Next()
		}

		// Try Bearer token
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			authCtx, err := authSvc.validateJWT(token)
			if err != nil {
				return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
			}
			c.Locals(AuthContextKey, authCtx)
			return c.Next()
		}

		return fiber.NewError(fiber.StatusUnauthorized, "missing authentication")
	}
}

func (s *AuthService) validateAPIKey(ctx context.Context, key string) (*AuthContext, error) {
	// Check prefix format: ask_xxxxx
	if !strings.HasPrefix(key, "ask_") {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid key format")
	}

	// Hash the key for lookup
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	// Try cache first
	cached, found, _ := s.cache.GetSession(ctx, "apikey:"+keyHash)
	if found {
		// Deserialize cached auth context
		// ... (implementation)
	}

	// Lookup in database
	apiKeyRecord, err := s.apiKeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	// Check expiration
	if apiKeyRecord.ExpiresAt != nil && apiKeyRecord.ExpiresAt.Before(time.Now()) {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "API key expired")
	}

	authCtx := &AuthContext{
		TeamID:    apiKeyRecord.TeamID,
		ProjectID: apiKeyRecord.ProjectID,
		Scopes:    apiKeyRecord.Scopes,
		AuthType:  "api_key",
	}

	// Cache for 5 minutes
	// s.cache.SetSession(ctx, "apikey:"+keyHash, serialized, 5*time.Minute)

	// Update last used timestamp (async)
	go s.apiKeyRepo.UpdateLastUsed(context.Background(), apiKeyRecord.ID)

	return authCtx, nil
}

func (s *AuthService) validateJWT(tokenString string) (*AuthContext, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid claims")
	}

	return &AuthContext{
		TeamID:   claims["team_id"].(string),
		UserID:   claims["sub"].(string),
		AuthType: "jwt",
	}, nil
}

// GetAuthContext extracts auth context from fiber context
func GetAuthContext(c *fiber.Ctx) *AuthContext {
	auth, ok := c.Locals(AuthContextKey).(*AuthContext)
	if !ok {
		return nil
	}
	return auth
}
```

**`internal/api/middleware/ratelimit.go`**:
```go
package middleware

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
)

func RateLimit(redis *cache.RedisClient, limitPerMinute int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := GetAuthContext(c)
		if auth == nil {
			return c.Next() // Auth middleware will handle
		}

		// Rate limit key based on team
		key := fmt.Sprintf("ratelimit:%s:%d", auth.TeamID, time.Now().Unix()/60)

		count, err := redis.IncrementRateLimit(c.Context(), key, time.Minute)
		if err != nil {
			// Log error but don't block request
			return c.Next()
		}

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(limitPerMinute))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(max(0, limitPerMinute-int(count))))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Unix()/60*60+60, 10))

		if int(count) > limitPerMinute {
			c.Set("Retry-After", "60")
			return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded")
		}

		return c.Next()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
```

**`internal/api/middleware/tracing.go`**:
```go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("agentstack-api")

func Tracing() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract parent context from headers
		ctx := otel.GetTextMapPropagator().Extract(
			c.Context(),
			propagation.HeaderCarrier(c.GetReqHeaders()),
		)

		// Start span
		ctx, span := tracer.Start(ctx, c.Method()+" "+c.Path(),
			trace.WithAttributes(
				attribute.String("http.method", c.Method()),
				attribute.String("http.url", c.OriginalURL()),
				attribute.String("http.route", c.Route().Path),
				attribute.String("http.request_id", c.Get("X-Request-ID")),
			),
		)
		defer span.End()

		// Store in fiber context
		c.SetUserContext(ctx)

		// Execute handler
		err := c.Next()

		// Record response status
		span.SetAttributes(
			attribute.Int("http.status_code", c.Response().StatusCode()),
		)

		if err != nil {
			span.RecordError(err)
		}

		return err
	}
}
```

---

## Week 2: Domain Layer & Handlers

### 2.1 Domain Models

**`internal/domain/agent/model.go`**:
```go
package agent

import (
	"time"
)

// Agent represents an AI agent in the system
type Agent struct {
	ID                  string            `json:"id"`
	ProjectID           string            `json:"project_id"`
	Name                string            `json:"name"`
	Slug                string            `json:"slug"`
	Description         string            `json:"description,omitempty"`
	Status              Status            `json:"status"`
	Framework           Framework         `json:"framework"`
	Config              Config            `json:"config"`
	Source              *Source           `json:"source,omitempty"`
	Tags                []string          `json:"tags,omitempty"`
	CurrentDeploymentID string            `json:"current_deployment_id,omitempty"`
	URLs                *URLs             `json:"urls,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

type Status string

const (
	StatusInactive  Status = "inactive"
	StatusDeploying Status = "deploying"
	StatusActive    Status = "active"
	StatusFailed    Status = "failed"
	StatusSuspended Status = "suspended"
)

type Framework string

const (
	FrameworkGoogleADK Framework = "google-adk"
	FrameworkLangGraph Framework = "langgraph"
	FrameworkCrewAI    Framework = "crewai"
	FrameworkCustom    Framework = "custom"
)

type Config struct {
	Model        string                 `json:"model,omitempty"`
	SystemPrompt string                 `json:"system_prompt,omitempty"`
	Tools        []string               `json:"tools,omitempty"`
	Temperature  float64                `json:"temperature,omitempty"`
	MaxTokens    int                    `json:"max_tokens,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

type Source struct {
	Type   string `json:"type"` // "git", "image", "inline"
	URL    string `json:"url,omitempty"`
	Ref    string `json:"ref,omitempty"`
	Image  string `json:"image,omitempty"`
	Inline string `json:"inline,omitempty"`
}

type URLs struct {
	Chat   string `json:"chat,omitempty"`
	API    string `json:"api,omitempty"`
	Health string `json:"health,omitempty"`
}

// CreateRequest for creating a new agent
type CreateRequest struct {
	Name        string    `json:"name" validate:"required,min=1,max=64"`
	Description string    `json:"description" validate:"max=500"`
	Framework   Framework `json:"framework" validate:"required,oneof=google-adk langgraph crewai custom"`
	Config      Config    `json:"config"`
	Source      *Source   `json:"source"`
	Tags        []string  `json:"tags" validate:"max=10,dive,max=32"`
	AutoDeploy  bool      `json:"auto_deploy"`
}

// UpdateRequest for updating an agent
type UpdateRequest struct {
	Name        *string   `json:"name" validate:"omitempty,min=1,max=64"`
	Description *string   `json:"description" validate:"omitempty,max=500"`
	Config      *Config   `json:"config"`
	Tags        *[]string `json:"tags" validate:"omitempty,max=10,dive,max=32"`
}

// ListFilter for querying agents
type ListFilter struct {
	ProjectID string
	Status    *Status
	Framework *Framework
	Tags      []string
	Search    string
	Cursor    string
	Limit     int
}
```

**`internal/domain/agent/service.go`**:
```go
package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/project"
	"github.com/raphaelmansuy/agentstack/internal/pkg/id"
)

type Service struct {
	repo        Repository
	projectRepo project.Repository
}

type Repository interface {
	Create(ctx context.Context, agent *Agent) error
	GetByID(ctx context.Context, id string) (*Agent, error)
	GetBySlug(ctx context.Context, projectID, slug string) (*Agent, error)
	Update(ctx context.Context, agent *Agent) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter ListFilter) ([]*Agent, string, error)
}

func NewService(repo Repository, projectRepo project.Repository) *Service {
	return &Service{
		repo:        repo,
		projectRepo: projectRepo,
	}
}

func (s *Service) Create(ctx context.Context, projectID string, req CreateRequest) (*Agent, error) {
	// Verify project exists
	_, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Generate slug from name
	slug := slugify(req.Name)

	// Check for duplicate slug
	existing, _ := s.repo.GetBySlug(ctx, projectID, slug)
	if existing != nil {
		return nil, fmt.Errorf("agent with slug '%s' already exists", slug)
	}

	agent := &Agent{
		ID:          id.Generate("agt"),
		ProjectID:   projectID,
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Status:      StatusInactive,
		Framework:   req.Framework,
		Config:      req.Config,
		Source:      req.Source,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	// TODO: If AutoDeploy, trigger deployment

	return agent, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Agent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*Agent, error) {
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		agent.Name = *req.Name
		agent.Slug = slugify(*req.Name)
	}
	if req.Description != nil {
		agent.Description = *req.Description
	}
	if req.Config != nil {
		agent.Config = *req.Config
	}
	if req.Tags != nil {
		agent.Tags = *req.Tags
	}

	agent.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	return agent, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// TODO: Stop running deployments first

	if agent.Status == StatusActive {
		return fmt.Errorf("cannot delete active agent, suspend first")
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]*Agent, string, error) {
	if filter.Limit == 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	return s.repo.List(ctx, filter)
}

func slugify(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters
	// ... (more sanitization)
	return slug
}
```

### 2.2 HTTP Handlers

**`internal/api/handlers/agent.go`**:
```go
package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/go-playground/validator/v10"

	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/agent"
	"github.com/raphaelmansuy/agentstack/internal/pkg/logger"
)

type AgentHandler struct {
	svc       *agent.Service
	log       *logger.Logger
	validator *validator.Validate
}

func NewAgentHandler(svc *agent.Service, log *logger.Logger) *AgentHandler {
	return &AgentHandler{
		svc:       svc,
		log:       log,
		validator: validator.New(),
	}
}

// List godoc
// @Summary List agents
// @Description Get all agents for the current project
// @Tags agents
// @Accept json
// @Produce json
// @Param status query string false "Filter by status"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param search query string false "Search by name"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Page size (default 20, max 100)"
// @Success 200 {object} ListAgentsResponse
// @Router /v1/agents [get]
func (h *AgentHandler) List(c *fiber.Ctx) error {
	auth := middleware.GetAuthContext(c)

	filter := agent.ListFilter{
		ProjectID: auth.ProjectID,
		Search:    c.Query("search"),
		Cursor:    c.Query("cursor"),
		Limit:     c.QueryInt("limit", 20),
	}

	if status := c.Query("status"); status != "" {
		s := agent.Status(status)
		filter.Status = &s
	}

	agents, nextCursor, err := h.svc.List(c.UserContext(), filter)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"data": agents,
		"pagination": fiber.Map{
			"next_cursor": nextCursor,
			"has_more":    nextCursor != "",
		},
	})
}

// Create godoc
// @Summary Create agent
// @Description Create a new AI agent
// @Tags agents
// @Accept json
// @Produce json
// @Param agent body agent.CreateRequest true "Agent configuration"
// @Success 201 {object} agent.Agent
// @Router /v1/agents [post]
func (h *AgentHandler) Create(c *fiber.Ctx) error {
	auth := middleware.GetAuthContext(c)

	var req agent.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.validator.Struct(req); err != nil {
		return validationError(err)
	}

	created, err := h.svc.Create(c.UserContext(), auth.ProjectID, req)
	if err != nil {
		h.log.Error("failed to create agent", "error", err)
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

// Get godoc
// @Summary Get agent
// @Description Get agent by ID
// @Tags agents
// @Accept json
// @Produce json
// @Param agentId path string true "Agent ID"
// @Success 200 {object} agent.Agent
// @Router /v1/agents/{agentId} [get]
func (h *AgentHandler) Get(c *fiber.Ctx) error {
	agentID := c.Params("agentId")

	agent, err := h.svc.GetByID(c.UserContext(), agentID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "agent not found")
	}

	// TODO: Verify agent belongs to current project

	return c.JSON(agent)
}

// Update godoc
// @Summary Update agent
// @Description Update agent configuration
// @Tags agents
// @Accept json
// @Produce json
// @Param agentId path string true "Agent ID"
// @Param agent body agent.UpdateRequest true "Updated configuration"
// @Success 200 {object} agent.Agent
// @Router /v1/agents/{agentId} [patch]
func (h *AgentHandler) Update(c *fiber.Ctx) error {
	agentID := c.Params("agentId")

	var req agent.UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.validator.Struct(req); err != nil {
		return validationError(err)
	}

	updated, err := h.svc.Update(c.UserContext(), agentID, req)
	if err != nil {
		return err
	}

	return c.JSON(updated)
}

// Delete godoc
// @Summary Delete agent
// @Description Delete an agent
// @Tags agents
// @Accept json
// @Produce json
// @Param agentId path string true "Agent ID"
// @Success 204 "No Content"
// @Router /v1/agents/{agentId} [delete]
func (h *AgentHandler) Delete(c *fiber.Ctx) error {
	agentID := c.Params("agentId")

	if err := h.svc.Delete(c.UserContext(), agentID); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func validationError(err error) error {
	// Convert validation errors to RFC 7807 format
	// ... (implementation)
	return fiber.NewError(fiber.StatusBadRequest, err.Error())
}
```

---

## Week 3: Testing & Documentation

### 3.1 Integration Tests

**`tests/integration/agent_test.go`**:
```go
package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphaelmansuy/agentstack/internal/api"
	"github.com/raphaelmansuy/agentstack/internal/domain/agent"
)

func TestAgentCRUD(t *testing.T) {
	// Setup test server
	server := setupTestServer(t)

	// Create agent
	t.Run("Create", func(t *testing.T) {
		body := `{
			"name": "Test Agent",
			"framework": "google-adk",
			"config": {
				"model": "gpt-4o-mini",
				"system_prompt": "You are helpful"
			}
		}`

		req := httptest.NewRequest(http.MethodPost, "/v1/agents", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", testAPIKey)
		req.Header.Set("X-Project-ID", testProjectID)

		resp, err := server.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created agent.Agent
		err = json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)
		assert.Equal(t, "Test Agent", created.Name)
		assert.Equal(t, agent.StatusInactive, created.Status)
	})

	// List agents
	t.Run("List", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/agents", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		req.Header.Set("X-Project-ID", testProjectID)

		resp, err := server.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// ... more tests
}
```

### 3.2 OpenAPI Specification

Generate from code annotations:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/api/main.go -o api/openapi
```

---

## Deliverables Checklist

### Week 1
- [ ] HTTP server with Fiber/Chi
- [ ] Middleware stack (auth, rate limit, tracing)
- [ ] Health endpoints
- [ ] Request/response logging
- [ ] Error handling (RFC 7807)

### Week 2
- [ ] Agent domain model
- [ ] Agent service (CRUD)
- [ ] Agent repository (PostgreSQL)
- [ ] Project service and handler
- [ ] API key management

### Week 3
- [ ] Integration tests (>80% coverage)
- [ ] OpenAPI spec generation
- [ ] Postman/Insomnia collection
- [ ] API documentation
- [ ] Load test baseline

---

## Definition of Done

- [ ] All endpoints match `/spec/api/012-agents-endpoints.md`
- [ ] Authentication works with API keys and JWT
- [ ] Rate limiting functional per team
- [ ] Tracing visible in Jaeger/Tempo
- [ ] Integration tests pass
- [ ] OpenAPI spec generated and validated
- [ ] Performance: <100ms P99 for simple GET

---

## Sage AI Guidance

### API Design Principles

1. **Consistency**: All endpoints follow the same patterns
   - Resource-based URLs
   - Standard HTTP methods
   - RFC 7807 error responses

2. **Versioning**: Always prefix with `/v1`

3. **Pagination**: Use cursor-based pagination for lists

### Testing Strategy

```text
Unit Tests (pkg/**)          → 90% coverage
Domain Tests (domain/**)     → 85% coverage  
Handler Tests (api/**)       → 80% coverage
Integration Tests (tests/)   → Critical paths
```

### Common Mistakes to Avoid

| Mistake | Impact | Prevention |
|---------|--------|------------|
| N+1 queries | Performance | Use repository methods with joins |
| Missing auth checks | Security | Middleware handles auth, handlers verify ownership |
| Swallowing errors | Debugging | Always wrap with context |
| Blocking on I/O | Throughput | Use goroutines for fire-and-forget |

---

**Next Phase**: [002-agent-runtime.md](002-agent-runtime.md) - Agent lifecycle, A2A protocol, streaming
