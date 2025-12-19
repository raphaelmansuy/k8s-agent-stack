# AgentStack Makefile
# Copyright 2025 Raphaël MANSUY
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: all build test lint run dev generate migrate-up migrate-down docker-build docker-run clean help mlflow-ui setup install-tools

# Variables
GO := go
GOFLAGS := -v
BINARY := agentstack-api
WORKER_BINARY := agentstack-worker
CLI_BINARY := agentctl
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Docker
DOCKER_IMAGE := agentstack-api
DOCKER_TAG := $(VERSION)

# Database
DATABASE_URL ?= postgres://agentstack:agentstack@localhost:5432/agentstack?sslmode=disable
REDIS_URL ?= redis://localhost:6379

# Default target
all: lint test build

# Build targets
build: ## Build the API server binary
	@echo "Building $(BINARY)..."
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY) ./cmd/api

build-worker: ## Build the worker binary
	@echo "Building $(WORKER_BINARY)..."
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(WORKER_BINARY) ./cmd/worker

build-cli: ## Build the CLI binary
	@echo "Building $(CLI_BINARY)..."
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(CLI_BINARY) ./cli/cmd/agentctl

build-all: build build-worker build-cli ## Build all binaries

# Test targets
test: ## Run tests
	@echo "Running tests..."
	@$(GO) test -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@$(GO) test -race -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@$(GO) test -race -tags=integration ./tests/integration/...

# Lint targets
lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run ./...

lint-fix: ## Run linters and fix issues
	@echo "Running linters with fix..."
	@golangci-lint run --fix ./...

# Run targets
dev: ## Run with hot reload (requires air)
	@echo "Starting development server..."
	@air -c .air.toml

run: build ## Build and run the server
	@echo "Starting server..."
	@AGENTSTACK_DATABASE_URL=$(DATABASE_URL) AGENTSTACK_REDIS_URL=$(REDIS_URL) ./bin/$(BINARY)

# Code generation
generate: ## Generate code (sqlc, mocks, etc.)
	@echo "Generating code..."
	@$(GO) generate ./...
	@sqlc generate

# Database migrations
migrate-up: ## Apply database migrations
	@echo "Applying migrations..."
	@atlas migrate apply --env local --url "$(DATABASE_URL)"

migrate-down: ## Rollback database migrations
	@echo "Rolling back migrations..."
	@atlas migrate down --env local --url "$(DATABASE_URL)"

migrate-new: ## Create a new migration (usage: make migrate-new name=add_users_table)
	@echo "Creating migration: $(name)..."
	@atlas migrate diff $(name) --env local

migrate-status: ## Show migration status
	@atlas migrate status --env local --url "$(DATABASE_URL)"

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run -p 8080:8080 \
		-e AGENTSTACK_DATABASE_URL=$(DATABASE_URL) \
		-e AGENTSTACK_REDIS_URL=$(REDIS_URL) \
		$(DOCKER_IMAGE):latest

docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_IMAGE):latest

# Docker Compose
compose-up: ## Start all services with docker-compose
	@echo "Starting services..."
	@docker-compose up -d

compose-down: ## Stop all services
	@echo "Stopping services..."
	@docker-compose down

compose-logs: ## View service logs
	@docker-compose logs -f

compose-ps: ## List running services
	@docker-compose ps

mlflow-ui: ## Open MLflow UI
	@echo "Opening MLflow UI at http://localhost:5000..."
	@open http://localhost:5000

# Development setup
setup: install-tools ## Install development dependencies
	@echo "Installing Go dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy
	@echo "Setup complete!"

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@$(GO) install ariga.io/atlas/cmd/atlas@latest
	@$(GO) install github.com/air-verse/air@latest
	@$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	@$(GO) install mvdan.cc/gofumpt@latest
	@$(GO) install github.com/golang/mock/mockgen@latest

# OpenAPI
openapi: ## Generate OpenAPI spec from code
	@echo "Generating OpenAPI spec..."
	@$(GO) run ./cmd/api -openapi > api/openapi/spec.yaml
	@echo "OpenAPI spec generated: api/openapi/spec.yaml"

# Clean
clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ coverage.out coverage.html
	@$(GO) clean -cache -testcache

# Verify
verify: lint test build ## Run all verification steps
	@echo "All verification passed!"

# Help
help: ## Show this help
	@echo "AgentStack Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
