# AgentStack Platform Context

Essential context for creating AgentStack-specific skills.

## Platform Overview

AgentStack is a **Sovereign AI Agent Platform** for deploying, orchestrating, and scaling AI agents on any infrastructure with full data sovereignty.

## Technology Stack

| Layer | Components |
|-------|------------|
| Agent Frameworks | Google ADK, LangGraph, CrewAI, AutoGen |
| Agent Orchestration | kagent (CNCF Sandbox) |
| Evaluation | MLflow 3.x (Tracing, Scorers, Judges) |
| Serverless Runtime | Knative Serving 1.20+ |
| Ingress | Contour + Envoy (Gateway API) |
| Observability | OpenTelemetry, Prometheus, Grafana |
| Data | PostgreSQL 16, Redis 7, S3-compatible |
| Kubernetes | 1.28+ (any distribution) |

## Key Specifications

Reference these spec files when creating skills:

| Spec | Purpose |
|------|---------|
| `/spec/001-platform-overview.md` | Vision, goals, positioning |
| `/spec/002-architecture-layers.md` | 6-layer architecture |
| `/spec/003-agent-lifecycle.md` | Agent states, deployments |
| `/spec/004-api-design.md` | REST API structure |
| `/spec/010-agent-evaluation.md` | MLflow evaluation framework |

## API Specifications

Detailed API docs in `/spec/api/`:

| Document | Coverage |
|----------|----------|
| `010-multi-tenancy.md` | Tenant isolation, resource hierarchy |
| `011-authentication.md` | API keys, JWT, OAuth |
| `012-agents-endpoints.md` | Agent CRUD, deployments |
| `017-a2a-protocol.md` | Agent-to-Agent communication |
| `018-agui-protocol.md` | AG-UI streaming events |

## Implementation Plan

Phases in `/spec/plan/`:

| Phase | Focus |
|-------|-------|
| 0 | Foundation: Go project, CI/CD, data layer |
| 1 | Core API: Gateway, REST endpoints, auth |
| 2 | Agent Runtime: Lifecycle, A2A, streaming |
| 3 | Evaluation: MLflow integration, safety gates |
| 4 | Developer Experience: CLI, SDKs, docs |
| 5-6 | Enterprise: RBAC, quotas, production |

## Coding Standards

### Go Project Structure

```
agentstack/
├── cmd/                # Entry points (api, worker, agentctl)
├── internal/           # Private code
│   ├── api/           # Handlers, middleware, routes
│   ├── domain/        # Business logic (pure Go)
│   └── infrastructure/ # External integrations
├── pkg/               # Public libraries
└── deployments/       # Kubernetes manifests
```

### Go Dependencies

```go
// HTTP
github.com/gofiber/fiber/v2

// Database
github.com/jackc/pgx/v5

// Cache
github.com/redis/go-redis/v9

// Observability
go.opentelemetry.io/otel

// Validation
github.com/go-playground/validator/v10
```

## Resource Naming

| Resource | Prefix | Example |
|----------|--------|---------|
| Project | `prj_` | `prj_abc123` |
| Agent | `agt_` | `agt_xyz789` |
| Deployment | `dpl_` | `dpl_def456` |
| Session | `ses_` | `ses_ghi012` |
| API Key | `key_` | `key_jkl345` |

## Skill Creation Context

When creating AgentStack skills:

1. **Assume Kubernetes context**: All deployments target K8s
2. **Consider multi-tenancy**: Resources scoped to projects
3. **Include observability**: OpenTelemetry instrumentation
4. **Follow Go idioms**: gofmt, golangci-lint compliance
5. **Reference specs**: Point to authoritative spec documents
6. **Safety first**: Evaluation gates are mandatory
