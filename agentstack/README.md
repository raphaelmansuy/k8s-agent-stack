# AgentStack

> Sovereign GenAI Agent Platform for Europe and Free Nations

AgentStack is a Kubernetes-native Platform-as-a-Service for deploying, orchestrating, and scaling AI agents. Built on CNCF open-source components (Apache 2.0), it provides data sovereignty, vendor independence, and production-grade reliability.

## Features

- **🔐 Data Sovereignty**: Run on your infrastructure, your data stays with you
- **🚀 Serverless Agents**: Scale-to-zero with Knative, pay only for what you use
- **🤖 Multi-Framework Support**: Google ADK, LangChain, CrewAI, AutoGen, and custom agents
- **🔄 A2A Protocol**: Agent-to-agent communication following Linux Foundation standards
- **📊 Built-in Evaluation**: MLflow integration for safety and quality gates
- **🔭 Observable by Default**: OpenTelemetry tracing, Prometheus metrics, structured logging
- **👥 Multi-Tenant**: Team and project isolation with RBAC

## Quick Start

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- Make

### Local Development

```bash
# Clone the repository
git clone https://github.com/raphaelmansuy/agentstack.git
cd agentstack

# Install development tools
make setup

# Start infrastructure (PostgreSQL, Redis, MLflow)
make compose-up

# Run the API server with hot reload
make dev
```

The API will be available at `http://localhost:8080`.

- OpenAPI spec: `http://localhost:8080/docs`
- Health check: `http://localhost:8080/health`

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run linters
make lint
```

### Building

```bash
# Build binary
make build

# Build Docker image
make docker-build

# Build and run
make run
```

## Deploying Agents

To deploy a real agent to AgentStack, you need to build its container image first.

### Building a Google ADK Agent

```bash
cd kagent-adk-agent
docker build -t kagent-adk-agent:latest .
```

### Deploying via CLI

Once the image is built, you can deploy it using `agentctl`:

```bash
agentctl deploy create --name "my-agent" --image "kagent-adk-agent:latest"
```

## Project Structure

```
agentstack/
├── cmd/                          # Application entry points
│   ├── api/                      # API Gateway server
│   └── worker/                   # Background job processor
├── internal/                     # Private application code
│   ├── api/                      # API layer
│   │   ├── handlers/             # HTTP handlers
│   │   ├── middleware/           # Auth, logging, rate limiting
│   │   └── routes/               # Route definitions
│   ├── domain/                   # Business logic
│   ├── infrastructure/           # External integrations
│   │   ├── database/             # PostgreSQL
│   │   ├── cache/                # Redis
│   │   └── k8s/                  # Kubernetes client
│   ├── config/                   # Configuration
│   └── pkg/                      # Shared utilities
├── migrations/                   # Database migrations
├── queries/                      # SQL queries (sqlc)
├── tests/                        # Integration/E2E tests
├── Dockerfile                    # Multi-stage Docker build
├── docker-compose.yaml           # Local development stack
└── Makefile                      # Build automation
```

## Configuration

AgentStack is configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `AGENTSTACK_ENVIRONMENT` | Environment (development/production) | `development` |
| `AGENTSTACK_SERVER_PORT` | API server port | `8080` |
| `AGENTSTACK_DATABASE_URL` | PostgreSQL connection string | `postgres://...` |
| `AGENTSTACK_REDIS_URL` | Redis connection string | `redis://localhost:6379` |
| `AGENTSTACK_TELEMETRY_ENABLED` | Enable OpenTelemetry | `false` |
| `AGENTSTACK_AUTH_JWT_SECRET` | JWT signing secret | Required in production |

See [config.go](internal/config/config.go) for all options.

## API Endpoints

### Health
- `GET /health` - Health check with component status
- `GET /ready` - Kubernetes readiness probe
- `GET /live` - Kubernetes liveness probe

### Projects
- `GET /v1/projects` - List projects
- `POST /v1/projects` - Create project
- `GET /v1/projects/{id}` - Get project
- `PATCH /v1/projects/{id}` - Update project
- `DELETE /v1/projects/{id}` - Delete project

### Agents
- `GET /v1/projects/{id}/agents` - List agents
- `POST /v1/projects/{id}/agents` - Create agent
- `GET /v1/agents/{id}` - Get agent
- `PATCH /v1/agents/{id}` - Update agent
- `DELETE /v1/agents/{id}` - Delete agent

### Chat
- `POST /v1/agents/{id}/chat` - Create chat session
- `POST /v1/chat/{session_id}/messages` - Send message
- `GET /v1/chat/{session_id}` - Get session with messages

Full API documentation: `http://localhost:8080/docs`

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    AgentStack Platform                          │
├─────────────────────────────────────────────────────────────────┤
│  Layer 6: Governance    │ RBAC, Quotas, Audit, Compliance       │
│  Layer 5: Evaluation    │ MLflow Safety, Scorers, Quality Gates │
│  Layer 4: Interface     │ API Gateway, CLI, SDK, UI             │
│  Layer 3: Cognitive     │ kagent, A2A Protocol, MCP Tools       │
│  Layer 2: Runtime       │ Knative Serving, Autoscaling          │
│  Layer 1: Infrastructure│ Kubernetes, Contour/Envoy, Storage    │
└─────────────────────────────────────────────────────────────────┘
```

## Technology Stack

- **API Framework**: [Huma v2](https://huma.rocks/) - Type-safe Go APIs with auto-OpenAPI
- **Database**: PostgreSQL 16 with [pgx](https://github.com/jackc/pgx) and [sqlc](https://sqlc.dev/)
- **Cache**: Redis 7 for sessions, rate limiting, and pub/sub
- **Observability**: OpenTelemetry, Prometheus, Grafana
- **Runtime**: Knative Serving for serverless agents
- **Evaluation**: MLflow for agent quality and safety

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0 - See [LICENSE](../LICENSE)

---

Built with ❤️ for European data sovereignty and open standards.
