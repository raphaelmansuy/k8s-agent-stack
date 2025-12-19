# API Gateway Architecture

[← Back to Master Architecture](../architecture.md)

The AgentStack API Gateway is the central entry point for all administrative and operational requests. It is designed for high performance, security, and developer productivity.

## Technology Stack

- **Language**: Go 1.22+
- **Web Framework**: [Chi](https://github.com/go-chi/chi) (Lightweight, idiomatic router)
- **API Framework**: [Huma v2](https://github.com/danielgtaylor/huma) (Type-safe, OpenAPI-first framework)
- **Logging**: [Zap](https://github.com/uber-go/zap) (Structured logging)
- **Telemetry**: OpenTelemetry (Tracing and Metrics)

## Core Responsibilities

### 1. Request Routing & Validation
The gateway uses `chi` for routing and `huma` for request/response validation. Every endpoint is defined with a Go struct that automatically generates JSON Schema for validation and OpenAPI 3.1 documentation.

### 2. Middleware Pipeline
The API applies a series of global and route-specific middlewares:

| Middleware | Responsibility | Implementation |
|------------|----------------|----------------|
| **RequestID** | Assigns a unique ID to every request | `chi/middleware.RequestID` |
| **RealIP** | Extracts the client's real IP | `chi/middleware.RealIP` |
| **Logger** | Logs request details and timing | `internal/api/middleware/logger.go` |
| **Recoverer** | Gracefully handles panics | `chi/middleware.Recoverer` |
| **Timeout** | Enforces a 60s request timeout | `chi/middleware.Timeout` |
| **Telemetry** | Injects tracing and records metrics | [Observability Architecture](observability.md) |
| **Auth** | Validates JWT or API Keys | [Security Architecture](security.md) |
| **Tenant** | Injects tenant context into the request | [Security Architecture](security.md) |
| **Audit** | Records administrative actions | [Observability Architecture](observability.md) |

### 3. OpenAPI Documentation
The gateway automatically serves an interactive OpenAPI documentation (Swagger UI) at `/docs`. This documentation is always in sync with the code because it is generated from the Go types used in the handlers.

## Internal Structure (`agentstack/internal/api`)

- **`handlers/`**: Contains the business logic for each API endpoint, grouped by domain (e.g., `agents`, `projects`, `keys`).
- **`middleware/`**: Contains custom middleware implementations for security, logging, and context injection.

## Data Access Pattern
Handlers do not interact with the database directly. Instead, they use **Domain Services** (e.g., `deployment.Service`, `rbac.Service`) which in turn use **Repositories** for data persistence. This ensures a clean separation of concerns and makes the code easier to test.

## Error Handling
The gateway uses a standardized error format defined by `huma`. Errors include a machine-readable code, a human-readable message, and optional validation details.

```json
{
  "$schema": "https://huma.rocks/error.json",
  "status": 400,
  "title": "Bad Request",
  "detail": "validation failed",
  "errors": [
    {
      "message": "expected string, but got number",
      "location": "body.name",
      "value": 123
    }
  ]
}
```
