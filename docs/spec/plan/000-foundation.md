# Phase 0: Foundation

> Project Structure, CI/CD Pipeline, Data Layer Setup

**Duration**: 2 weeks | **Status**: Not Started | **Priority**: Critical

---

## Objectives

1. Establish Go project structure with clean architecture
2. Deploy PostgreSQL and Redis infrastructure
3. Set up CI/CD pipeline (GitHub Actions)
4. Create development environment automation
5. Establish coding standards and tooling

---

## Week 1: Project Structure & Tooling

### 1.1 Repository Structure

```text
agentstack/
├── cmd/                          # Application entry points
│   ├── api/                      # API Gateway server
│   │   └── main.go
│   ├── worker/                   # Background job processor
│   │   └── main.go
│   └── agentctl/                 # CLI tool (Phase 4)
│       └── main.go
├── internal/                     # Private application code
│   ├── api/                      # API layer
│   │   ├── handlers/             # HTTP handlers
│   │   ├── middleware/           # Auth, logging, rate limiting
│   │   ├── routes/               # Route definitions
│   │   └── server.go             # Server setup
│   ├── domain/                   # Business logic (pure Go)
│   │   ├── agent/                # Agent entity & service
│   │   ├── project/              # Project/tenant logic
│   │   ├── chat/                 # Chat session logic
│   │   └── evaluation/           # Evaluation logic
│   ├── infrastructure/           # External integrations
│   │   ├── database/             # PostgreSQL repository
│   │   ├── cache/                # Redis client
│   │   ├── queue/                # Job queue
│   │   ├── k8s/                  # Kubernetes client
│   │   └── llm/                  # LLM provider clients
│   ├── config/                   # Configuration loading
│   └── pkg/                      # Shared utilities
│       ├── logger/               # Structured logging
│       ├── errors/               # Error types
│       ├── validator/            # Input validation
│       └── telemetry/            # OpenTelemetry setup
├── pkg/                          # Public libraries (SDK later)
│   └── client/                   # Go SDK client
├── api/                          # API specifications
│   └── openapi/                  # OpenAPI 3.1 specs
├── migrations/                   # Database migrations
│   └── atlas/                    # Atlas migration files
├── deployments/                  # Kubernetes manifests
│   ├── base/                     # Base manifests
│   ├── dev/                      # Dev overlays
│   └── prod/                     # Prod overlays
├── scripts/                      # Build/dev scripts
├── tests/                        # Integration/E2E tests
│   ├── integration/
│   └── e2e/
├── docs/                         # Developer documentation
├── .github/                      # GitHub Actions workflows
│   └── workflows/
├── Makefile                      # Build automation
├── Dockerfile                    # Multi-stage build
├── docker-compose.yaml           # Local dev environment
├── go.mod
├── go.sum
└── README.md
```

### 1.2 Go Module Initialization

```bash
# Initialize module
go mod init github.com/raphaelmansuy/agentstack

# Core dependencies
go get github.com/gofiber/fiber/v2                    # HTTP framework
go get github.com/go-chi/chi/v5                       # Alternative router
go get github.com/jackc/pgx/v5                        # PostgreSQL driver
go get github.com/redis/go-redis/v9                   # Redis client
go get github.com/golang-jwt/jwt/v5                   # JWT handling
go get go.opentelemetry.io/otel                       # OpenTelemetry
go get github.com/go-playground/validator/v10         # Validation
go get github.com/spf13/viper                         # Configuration
go get go.uber.org/zap                                # Structured logging
go get github.com/stretchr/testify                    # Testing
go get github.com/golang/mock                         # Mocking
```

### 1.3 Development Tooling

**Required Tools**:
```bash
# Linting
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Code generation
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest

# Database migrations
go install ariga.io/atlas/cmd/atlas@latest

# Hot reload for development
go install github.com/cosmtrek/air@latest

# Mock generation
go install github.com/golang/mock/mockgen@latest
```

**golangci-lint configuration** (`.golangci.yml`):
```yaml
run:
  timeout: 5m
  go: "1.22"

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell
    - unconvert
    - gocritic
    - revive

linters-settings:
  govet:
    enable-all: true
  goimports:
    local-prefixes: github.com/raphaelmansuy/agentstack

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
```

### 1.4 Makefile Targets

```makefile
.PHONY: all build test lint run dev

# Variables
GO := go
GOFLAGS := -v
BINARY := agentstack-api
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Default target
all: lint test build

# Build
build:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY) ./cmd/api

# Run tests
test:
	$(GO) test -race -cover ./...

# Run tests with coverage report
test-coverage:
	$(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	golangci-lint run ./...

# Run locally (with hot reload)
dev:
	air -c .air.toml

# Run locally (without hot reload)
run: build
	./bin/$(BINARY)

# Database migrations
migrate-up:
	atlas migrate apply --env local

migrate-down:
	atlas migrate down --env local

migrate-new:
	atlas migrate diff $(name) --env local

# Generate code
generate:
	go generate ./...
	sqlc generate
	oapi-codegen -generate types,chi-server -package api -o internal/api/generated.go api/openapi/spec.yaml

# Docker
docker-build:
	docker build -t agentstack-api:$(VERSION) .

docker-run:
	docker-compose up -d

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html
```

---

## Week 1: CI/CD Pipeline

### 1.5 GitHub Actions Workflow

**`.github/workflows/ci.yml`**:
```yaml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  GO_VERSION: "1.22"

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      
      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  test:
    name: Test
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: agentstack_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      
      - name: Run tests
        run: make test-coverage
        env:
          DATABASE_URL: postgres://test:test@localhost:5432/agentstack_test?sslmode=disable
          REDIS_URL: redis://localhost:6379
      
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: coverage.out

  build:
    name: Build
    runs-on: ubuntu-latest
    needs: [lint, test]
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      
      - name: Build
        run: make build
      
      - name: Build Docker image
        run: make docker-build

  security:
    name: Security Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          severity: 'CRITICAL,HIGH'
```

---

## Week 2: Data Layer

### 2.1 PostgreSQL Schema (Atlas)

**`migrations/atlas/schema.hcl`**:
```hcl
# Teams (multi-tenancy root)
table "teams" {
  schema = schema.public
  
  column "id" {
    type = text
    null = false
  }
  column "name" {
    type = text
    null = false
  }
  column "slug" {
    type = text
    null = false
  }
  column "plan" {
    type    = text
    null    = false
    default = "free"
  }
  column "settings" {
    type    = jsonb
    null    = false
    default = "{}"
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  
  primary_key {
    columns = [column.id]
  }
  
  index "idx_teams_slug" {
    columns = [column.slug]
    unique  = true
  }
}

# Projects
table "projects" {
  schema = schema.public
  
  column "id" {
    type = text
    null = false
  }
  column "team_id" {
    type = text
    null = false
  }
  column "name" {
    type = text
    null = false
  }
  column "slug" {
    type = text
    null = false
  }
  column "settings" {
    type    = jsonb
    null    = false
    default = "{}"
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  
  primary_key {
    columns = [column.id]
  }
  
  foreign_key "fk_team" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_delete   = CASCADE
  }
  
  index "idx_projects_team_slug" {
    columns = [column.team_id, column.slug]
    unique  = true
  }
}

# Agents
table "agents" {
  schema = schema.public
  
  column "id" {
    type = text
    null = false
  }
  column "project_id" {
    type = text
    null = false
  }
  column "name" {
    type = text
    null = false
  }
  column "slug" {
    type = text
    null = false
  }
  column "description" {
    type = text
  }
  column "status" {
    type    = text
    null    = false
    default = "inactive"
  }
  column "framework" {
    type = text
    null = false
  }
  column "config" {
    type    = jsonb
    null    = false
    default = "{}"
  }
  column "source" {
    type = jsonb
  }
  column "tags" {
    type    = sql("text[]")
    default = "{}"
  }
  column "current_deployment_id" {
    type = text
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  
  primary_key {
    columns = [column.id]
  }
  
  foreign_key "fk_project" {
    columns     = [column.project_id]
    ref_columns = [table.projects.column.id]
    on_delete   = CASCADE
  }
  
  index "idx_agents_project_slug" {
    columns = [column.project_id, column.slug]
    unique  = true
  }
  
  index "idx_agents_status" {
    columns = [column.project_id, column.status]
  }
  
  index "idx_agents_tags" {
    columns = [column.tags]
    type    = GIN
  }
}

# API Keys
table "api_keys" {
  schema = schema.public
  
  column "id" {
    type = text
    null = false
  }
  column "team_id" {
    type = text
    null = false
  }
  column "project_id" {
    type = text
  }
  column "name" {
    type = text
    null = false
  }
  column "key_hash" {
    type = text
    null = false
  }
  column "key_prefix" {
    type = text
    null = false
  }
  column "scopes" {
    type    = sql("text[]")
    null    = false
    default = "{}"
  }
  column "last_used_at" {
    type = timestamptz
  }
  column "expires_at" {
    type = timestamptz
  }
  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  
  primary_key {
    columns = [column.id]
  }
  
  foreign_key "fk_team" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_delete   = CASCADE
  }
  
  index "idx_api_keys_hash" {
    columns = [column.key_hash]
    unique  = true
  }
}
```

### 2.2 Docker Compose for Local Development

**`docker-compose.yaml`**:
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: agentstack
      POSTGRES_PASSWORD: agentstack
      POSTGRES_DB: agentstack
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U agentstack"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  # MLflow for evaluation (Phase 3)
  mlflow:
    image: ghcr.io/mlflow/mlflow:v2.17.0
    ports:
      - "5000:5000"
    environment:
      MLFLOW_TRACKING_URI: postgresql://agentstack:agentstack@postgres:5432/mlflow
    depends_on:
      - postgres
    command: >
      mlflow server
      --host 0.0.0.0
      --port 5000
      --backend-store-uri postgresql://agentstack:agentstack@postgres:5432/mlflow

volumes:
  postgres_data:
  redis_data:
```

### 2.3 Redis Configuration

**`internal/infrastructure/cache/redis.go`**:
```go
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(url string) (*RedisClient, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisClient{client: client}, nil
}

// Session state management
func (r *RedisClient) SetSession(ctx context.Context, sessionID string, data []byte, ttl time.Duration) error {
	return r.client.Set(ctx, "session:"+sessionID, data, ttl).Err()
}

func (r *RedisClient) GetSession(ctx context.Context, sessionID string) ([]byte, error) {
	return r.client.Get(ctx, "session:"+sessionID).Bytes()
}

// Rate limiting
func (r *RedisClient) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// Idempotency keys
func (r *RedisClient) SetIdempotencyKey(ctx context.Context, key string, response []byte) error {
	return r.client.Set(ctx, "idem:"+key, response, 24*time.Hour).Err()
}

func (r *RedisClient) GetIdempotencyKey(ctx context.Context, key string) ([]byte, bool, error) {
	data, err := r.client.Get(ctx, "idem:"+key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
```

---

## Deliverables Checklist

### Week 1
- [ ] Go module initialized with dependencies
- [ ] Project structure created (cmd, internal, pkg)
- [ ] Makefile with all targets
- [ ] golangci-lint configuration
- [ ] Air (hot reload) configuration
- [ ] GitHub Actions CI workflow
- [ ] Dockerfile (multi-stage build)

### Week 2
- [ ] PostgreSQL schema (Atlas migrations)
- [ ] Docker Compose for local dev
- [ ] Database connection package
- [ ] Redis client package
- [ ] Configuration management (Viper)
- [ ] Structured logging (Zap)
- [ ] OpenTelemetry setup (basic)

---

## Definition of Done

- [ ] `make lint` passes with no errors
- [ ] `make test` passes with >70% coverage
- [ ] `make build` produces working binary
- [ ] `docker-compose up` starts all services
- [ ] `make migrate-up` creates database schema
- [ ] CI pipeline runs successfully on GitHub
- [ ] README with setup instructions

---

## Sage AI Guidance

### Critical Success Factors

1. **Project Structure Discipline**: Follow the structure exactly. Resist the temptation to "simplify" by putting everything in one package.

2. **Dependency Injection**: Use interfaces for all external dependencies (database, cache, LLM clients). This enables testing and future flexibility.

3. **Error Handling**: Establish error handling patterns from Day 1. Use structured errors with context.

4. **Configuration**: Use environment variables with sensible defaults. Never hardcode secrets.

### Common Pitfalls to Avoid

| Pitfall | Impact | Prevention |
|---------|--------|------------|
| Skipping tests | Technical debt | Require >70% coverage in CI |
| Global state | Testing nightmare | Use dependency injection |
| Ignoring linter | Code quality | Block merge on lint failures |
| Missing context propagation | Broken tracing | Always pass `context.Context` |

### Go Idioms to Follow

```go
// ✅ Good: Interface for dependency
type AgentRepository interface {
    Create(ctx context.Context, agent *Agent) error
    GetByID(ctx context.Context, id string) (*Agent, error)
}

// ❌ Bad: Concrete dependency
type AgentService struct {
    db *sql.DB  // Should be interface
}

// ✅ Good: Error wrapping with context
if err != nil {
    return fmt.Errorf("failed to create agent %s: %w", id, err)
}

// ✅ Good: Context propagation
func (s *Service) DoSomething(ctx context.Context) error {
    span := trace.SpanFromContext(ctx)
    // ...
}
```

---

**Next Phase**: [001-core-api.md](001-core-api.md) - API Gateway, REST endpoints, authentication
