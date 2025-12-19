# AgentStack Tech Stack

> The stack powering the sovereign, open-source GenAI agent platform.

AgentStack is built for European-grade sovereignty, production readiness, and extensibility. This document describes the primary technologies that carry the API, database, and security layers together so every contributor shares the same mental model.

## Goals

- Run on your own Kubernetes clusters (local, cloud, or on-prem) with minimal vendor lock-in.
- Deliver documented, type-safe APIs whose specs never drift.
- Enforce multi-tenant isolation and OAuth2 security without custom, brittle glue code.

## API Layer (Huma v2)

- **Why Huma**: Declarative Go APIs, FastAPI-inspired ergonomics, built-in OpenAPI/JSON Schema generation, and RFC 7807 problem details ensure the documentation and runtime behavior stay aligned. Example handlers remain router-agnostic thanks to Huma adapters for Chi, Fiber, or the stdlib mux.
- **Core benefits**: `huma.Register()` operations define inputs/outputs once, validation tags cover most business rules, and auto-generated docs (served at `/openapi.json` and `/docs`) supply SDKs or mocks for frontend/QA teams.
- **Usage guidance**: Keep business logic in handlers, let Huma handle serialization, and use `Resolve()` methods for cross-field validation. The spec/tech_stack/001-api-tech-stack.md document summarizes the working assumptions.

## Database Layer (pgx + sqlc + PostgreSQL)

- **Why pgx + sqlc**: pgx exposes PostgreSQL features with minimal overhead, while sqlc compiles raw SQL files into type-safe Go functions, keeping database code explicit yet ergonomic. The combination avoids ORMs and leaves you in full control of queries.
- **Multi-tenant safety**: Set session variables (e.g., `SET app.tenant_id`) on `pgxpool` connections before executing sqlc-generated queries so PostgreSQL Row-Level Security policies automatically scope every request, even if a handler accidentally misuses the API.
- **Recommended workflow**: Keep SQL in version-controlled `.sql` files with `-- name:` annotations, run `sqlc generate` in CI, and treat `db.Queries` as the single source of truth for CRUD logic. The spec/tech_stack/002-database-layer-stack.md file expands on the RLS pattern and lifecycle checklists.

## OAuth Layer (Ory Fosite / Hydra)

- **Why Fosite**: Security-first OAuth2/OpenID Connect framework that respects RFC 6749/6819. It ships extensible handlers (authorization code, client credentials, refresh tokens, introspection, etc.) and lets you plug in a custom PostgreSQL store for clients, tokens, and consents.
- **When to reach for Hydra**: If you need a standalone, production-ready authorization server, Hydra builds on Fosite and adds PKCE, consent UIs, and federation-ready tooling without re-implementing the OAuth stack.
- **Recommendation**: Start with Fosite as an embedded library for service-to-service auth or to issue tokens from AgentStack. Use Hydra when you prefer a dedicated OAuth2 service that can be secured, observed, and scaled independently. The spec/tech_stack/003-oauth-stack.md note captures the trade-offs.

## Polishing Notes

1. **Observability hooks**: Expose metrics (OpenTelemetry) off Huma middleware, pgx/tracing, and Fosite handlers so the infrastructure team can monitor latency, DB load, and token issuance.
2. **Automation**: The top-level Makefile and docs/ guides rely on this stack; match CLI flags (`SERVICE_PORT`, `DATABASE_URL`, `PGX_LOG_LEVEL`, `HYDRA_ADMIN_URL`) to the same env vars described in the detailed docs.
3. **Documentation**: Link to spec/README.md and to each tech stack note so readers can dive deeper.

## References

- API stack reference: [spec/tech_stack/001-api-tech-stack.md](spec/tech_stack/001-api-tech-stack.md)
- Database stack reference: [spec/tech_stack/002-database-layer-stack.md](spec/tech_stack/002-database-layer-stack.md)
- OAuth stack note: [spec/tech_stack/003-oauth-stack.md](spec/tech_stack/003-oauth-stack.md)
- Huma docs: https://huma.rocks/
- sqlc docs: https://docs.sqlc.dev/
- pgx docs: https://github.com/jackc/pgx
- Fosite docs: https://www.ory.sh/fosite/
- Platform spec context: [spec/README.md](spec/README.md)
