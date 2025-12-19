# API Specifications

> AgentStack Platform API Documentation

**Version**: 1.0.0 | **Last Updated**: 2025-12-17

---

## Overview

This directory contains the split API specifications for the AgentStack Platform. The API is designed following multi-tenant PaaS best practices inspired by Stripe, Vercel, and Azure's multi-tenant architecture patterns.

## API Documents

### Core API

| Document | Description |
|----------|-------------|
| [010-multi-tenancy.md](010-multi-tenancy.md) | Multi-tenant architecture, isolation models, resource hierarchy |
| [011-authentication.md](011-authentication.md) | API keys, JWT, OAuth, rate limiting, security |
| [012-agents-endpoints.md](012-agents-endpoints.md) | Agent CRUD, deployments, revisions, traffic management |
| [013-chat-sessions.md](013-chat-sessions.md) | Chat API, streaming, sessions, conversation history |
| [014-tools-models.md](014-tools-models.md) | Tool registry, model providers, MCP integration |
| [015-admin-endpoints.md](015-admin-endpoints.md) | Teams, projects, secrets, domains, webhooks, billing |
| [016-schemas.md](016-schemas.md) | OpenAPI schemas, error handling (RFC 7807) |

### Protocol Integration

| Document | Description |
|----------|-------------|
| [017-a2a-protocol.md](017-a2a-protocol.md) | A2A (Agent-to-Agent) protocol for agent interoperability |
| [018-agui-protocol.md](018-agui-protocol.md) | AG-UI streaming events for real-time agent-frontend communication |
| [019-a2ui-components.md](019-a2ui-components.md) | A2UI declarative UI components for rich agent responses |

### Content & Interactions

| Document | Description |
|----------|-------------|
| [020-universal-content-model.md](020-universal-content-model.md) | Universal Content Model (UCM) - Provider-agnostic multimodal content format |
| [021-interactions-api.md](021-interactions-api.md) | Interactions API - Unified interface for models/agents with server-side state |

## Resource Hierarchy

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Resource Hierarchy                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Organization (org_xxx)                                         │
│  └── Team (team_xxx)                                            │
│      └── Project (prj_xxx)           ◄── Tenant Boundary        │
│          ├── Agents (agt_xxx)                                   │
│          │   ├── Deployments (dpl_xxx)                          │
│          │   ├── Revisions (rev_xxx)                            │
│          │   └── Sessions (ses_xxx)                             │
│          ├── Tools (tol_xxx)                                    │
│          ├── Secrets (SECRET_NAME)                              │
│          ├── API Keys (key_xxx)                                 │
│          └── Webhooks (whk_xxx)                                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Multi-Tenant Design Principles

| Principle | Implementation |
|-----------|----------------|
| **Tenant Isolation** | Project-level data partitioning |
| **Resource Scoping** | All resources scoped to project |
| **Explicit Context** | `X-Project-ID` header or path prefix |
| **Quota Management** | Per-project and per-org limits |
| **Audit Logging** | All mutations logged with tenant context |

## API Design Standards

### URL Structure

```text
# Pattern 1: Header-based tenant context
GET /v1/agents
X-Project-ID: prj_abc123

# Pattern 2: Path-based tenant context (alternative)
GET /v1/projects/prj_abc123/agents

# Pattern 3: Inferred from API key (single-project keys)
GET /v1/agents
X-API-Key: as_xxx  # Key scoped to specific project
```

### ID Prefixes

| Resource | Prefix | Example |
|----------|--------|---------|
| Organization | `org_` | `org_2xKj9mNpQr` |
| Team | `team_` | `team_3yLk0nOpRs` |
| Project | `prj_` | `prj_4zMl1oQqSt` |
| Agent | `agt_` | `agt_5aNm2pRrTu` |
| Deployment | `dpl_` | `dpl_6bOn3qSsUv` |
| Revision | `rev_` | `rev_7cPo4rTtVw` |
| Session | `ses_` | `ses_8dQp5sUuWx` |
| Interaction | `int_` | `int_9fXy7zAbCd` |
| Tool | `tol_` | `tol_9eRq6tVvXy` |
| Webhook | `whk_` | `whk_0fSr7uWwYz` |
| API Key | `key_` | `key_1gTs8vXxZa` |
| Domain | `dom_` | `dom_2hUt9wYyAb` |
| File | `file_` | `file_3iVu0xZzBc` |

### Error Format (RFC 7807)

```json
{
  "type": "https://api.agentstack.io/errors/validation",
  "title": "Validation Error",
  "status": 400,
  "detail": "Request body contains invalid fields",
  "instance": "/v1/agents",
  "trace_id": "abc123xyz",
  "errors": [
    {"field": "name", "message": "must be 3-64 characters"}
  ]
}
```

## Rate Limiting

| Plan | Requests/min | Burst | Agents | Concurrent |
|------|--------------|-------|--------|------------|
| **Free** | 60 | 100 | 3 | 5 |
| **Pro** | 600 | 1,000 | 50 | 100 |
| **Enterprise** | 6,000 | 10,000 | Unlimited | 1,000 |

## Quick Links

- [Architecture](../002-architecture-layers.md) - Platform architecture
- [Security](../006-security-governance.md) - Authentication & authorization details
- [A2A Protocol](017-a2a-protocol.md) - Agent-to-agent interoperability
- [AG-UI Protocol](018-agui-protocol.md) - Real-time streaming events
- [A2UI Components](019-a2ui-components.md) - Declarative UI generation
- [Universal Content Model](020-universal-content-model.md) - Multimodal content format
- [Interactions API](021-interactions-api.md) - Server-side state management

---

**Parent**: [Specification Index](../README.md)
