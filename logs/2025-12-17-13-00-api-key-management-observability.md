# Task Log - 2025-12-17 - API Key Management & Observability

## Actions
- Implemented API Key Management system:
    - Created `internal/domain/auth` with `APIKey` models and `Service`.
    - Created `internal/infrastructure/database/api_key_repository.go` with RLS support.
    - Created `internal/api/handlers/api_keys.go` for REST API.
    - Integrated into `cmd/api/main.go`.
    - Updated `internal/api/middleware/auth.go` to use SHA-256 hashing for API keys.
- Implemented Observability:
    - Created `internal/pkg/telemetry/metrics.go` with Prometheus metrics.
    - Added metrics middleware and `/metrics` endpoint to the API Gateway.
- Updated SDK and CLI:
    - Updated `sdk/types.go` and `sdk/auth.go` to support new API key endpoints.
    - Created `cli/internal/commands/keys.go` for `agentctl keys` command.
    - Registered `keys` command in `cli/internal/commands/root.go`.
- Updated `Makefile`:
    - Added `build-cli` target.

## Decisions
- Used SHA-256 for API key hashing to ensure security (only hashes are stored).
- Keys are prefixed with `sk_live_` for easy identification.
- RLS is automatically applied via `TenantMiddleware` and `TenantDB` wrapper.
- Prometheus metrics are exposed on `/metrics` without authentication.

## Next Steps
- Implement more granular RBAC roles for API keys (e.g., read-only keys).
- Add more metrics for database performance and worker queue depth.
- Implement automated rotation for API keys.

## Lessons/Insights
- The `TenantDB` wrapper pattern is very effective for ensuring RLS is applied consistently across the codebase without manual intervention in every repository.
- Huma v2 makes it very easy to add middlewares and document APIs simultaneously.
