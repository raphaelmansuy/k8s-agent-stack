# AgentStack Platform Specifications

> **Mission**: Build the Sovereign GenAI Agent Platform for Europe and free nations.

## Overview

AgentStack is a Kubernetes-native Platform-as-a-Service for deploying, orchestrating, and scaling AI agents. Built on CNCF open-source components (Apache 2.0), it provides data sovereignty, vendor independence, and production-grade reliability.

## How to Navigate This Spec

**New to AgentStack?** Start with [001-platform-overview.md](001-platform-overview.md) for vision and principles, then [002-architecture-layers.md](002-architecture-layers.md) for the 6-layer platform model.

**Building integrations?** Jump to [api/README.md](api/README.md) for endpoint references and the [Protocol Integration](#protocol-integration) section for A2A, AG-UI, and A2UI.

**Implementing features?** Check [tech_stack/README.md](tech_stack/README.md) for API, database, and OAuth layer decisions, then the relevant spec document (e.g., [004-api-design](004-api-design.md) for API patterns, [005-data-architecture](005-data-architecture.md) for storage).

## Platform Specification Documents

| Document | Description | Status |
|----------|-------------|--------|
| [001-platform-overview](001-platform-overview.md) | Vision, principles, component stack | ✅ Complete |
| [002-architecture-layers](002-architecture-layers.md) | 6-layer architecture, request flow | ✅ Complete |
| [003-agent-lifecycle](003-agent-lifecycle.md) | Agent types, lifecycle, A2A protocol | ✅ Complete |
| [004-api-design](004-api-design.md) | OpenAPI spec, Go implementation | ✅ Complete |
| [005-data-architecture](005-data-architecture.md) | Storage, state, memory patterns | ✅ Complete |
| [006-security-governance](006-security-governance.md) | AuthN/AuthZ, secrets, compliance | ✅ Complete |
| [007-observability](007-observability.md) | Metrics, logging, tracing | ✅ Complete |
| [008-deployment-operations](008-deployment-operations.md) | CI/CD, scaling, multi-cloud | ✅ Complete |
| [009-developer-experience](009-developer-experience.md) | CLI, SDK, local dev | ✅ Complete |
| [010-agent-evaluation](010-agent-evaluation.md) | **MLflow safety & quality** | ✅ Complete |

### API Specification (Split)

The API design has been split into focused, modular documents:

| Document | Description |
|----------|-------------|
| [api/README.md](api/README.md) | API documentation index |
| [api/010-multi-tenancy.md](api/010-multi-tenancy.md) | Multi-tenant architecture |
| [api/011-authentication.md](api/011-authentication.md) | Auth, rate limiting, security |
| [api/012-agents-endpoints.md](api/012-agents-endpoints.md) | Agent CRUD & deployments |
| [api/013-chat-sessions.md](api/013-chat-sessions.md) | Chat API & streaming |
| [api/014-tools-models.md](api/014-tools-models.md) | Tools & model providers |
| [api/015-admin-endpoints.md](api/015-admin-endpoints.md) | Teams, projects, webhooks |
| [api/016-schemas.md](api/016-schemas.md) | Schema reference |

### Protocol Integration

AgentStack supports industry-standard agent protocols for interoperability:

| Document | Description |
|----------|-------------|
| [api/017-a2a-protocol.md](api/017-a2a-protocol.md) | A2A (Agent-to-Agent) protocol - Linux Foundation |
| [api/018-agui-protocol.md](api/018-agui-protocol.md) | AG-UI streaming events - Real-time communication |
| [api/019-a2ui-components.md](api/019-a2ui-components.md) | A2UI declarative UI - Google protocol |

### Content & Interactions

Universal content model and unified interaction interface:

| Document | Description |
|----------|-------------|
| [api/020-universal-content-model.md](api/020-universal-content-model.md) | UCM - Provider-agnostic multimodal content format |
| [api/021-interactions-api.md](api/021-interactions-api.md) | Unified interface for models/agents with server-side state |

## Implementation Tech Stack

The platform is implemented using battle-tested, open-source technologies chosen for **production reliability**, **data sovereignty**, and **extensibility**.

**See [tech_stack/README.md](tech_stack/README.md) for:**
- **API Layer** — Huma v2: Type-safe, auto-generated OpenAPI, no doc drift
- **Database** — pgx + sqlc + PostgreSQL RLS: Fast, multi-tenant safe, explicit SQL
- **OAuth** — Ory Fosite/Hydra: Security-first, RFC-compliant, extensible

Each layer includes working assumptions, quick-start checklists, and production patterns.

## Quick Reference

```text
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

┌─────────────────────────────────────────────────────────────────┐
│                    Protocol Stack                               │
├─────────────────────────────────────────────────────────────────┤
│  A2UI  │ Declarative UI components (Google)                     │
│  AG-UI │ Real-time streaming events (MIT)                       │
│  A2A   │ Agent-to-agent interoperability (Linux Foundation)     │
│  MCP   │ Tool integration (Anthropic)                           │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                Universal Content Model                          │
├─────────────────────────────────────────────────────────────────┤
│  UCM Part → OpenAI Adapter  → OpenAI API                        │
│          → Anthropic Adapter → Anthropic API                    │
│          → Gemini Adapter   → Google Gemini API                 │
│          → Ollama Adapter   → Ollama/Local Models               │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components (All Apache 2.0)

| Component | Purpose | License | CNCF Status |
|-----------|---------|---------|-------------|
| **Kubernetes** | Container orchestration | Apache 2.0 | Graduated |
| **Knative Serving** | Serverless runtime | Apache 2.0 | CNCF Incubating |
| **kagent** | Agent orchestration | Apache 2.0 | CNCF Sandbox |
| **Contour** | Ingress controller | Apache 2.0 | CNCF Incubating |
| **Envoy** | L7 proxy | Apache 2.0 | Graduated |
| **cert-manager** | Certificate management | Apache 2.0 | CNCF Graduated |
| **OpenTelemetry** | Observability | Apache 2.0 | CNCF Incubating |
| **MLflow** | Agent Evaluation & Tracing | Apache 2.0 | - |
| **PostgreSQL** | Primary database | PostgreSQL License | - |
| **Redis** | Cache/Queue | BSD-3 | - |

## Design Principles

1. **Sovereignty First**: Run anywhere—your infrastructure, your data
2. **Open Standards**: CloudEvents, OpenTelemetry, A2A Protocol
3. **Progressive Disclosure**: Simple defaults, powerful overrides
4. **Observable by Default**: Traces, metrics, logs built-in
5. **Developer Experience**: Vercel-like simplicity

## Official References

### Infrastructure & Orchestration
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Knative Serving Documentation](https://knative.dev/docs/)
- [kagent Documentation](https://kagent.dev/docs/)
- [Contour Ingress Controller](https://projectcontour.io/)

### Implementation Guides
- [Huma API Framework](https://huma.rocks/)
- [sqlc SQL Code Generator](https://docs.sqlc.dev/)
- [pgx PostgreSQL Driver](https://github.com/jackc/pgx)
- [Ory Fosite OAuth2 Framework](https://www.ory.sh/fosite/)
- [Ory Hydra Identity Provider](https://www.ory.sh/hydra/)
- [MLflow Evaluation](https://mlflow.org/docs/latest/llms/llm-evaluate/)
- [MLflow Tracing](https://mlflow.org/docs/latest/llms/tracing/)

### Standards & Protocols
- [CloudEvents Specification](https://cloudevents.io/)
- [OpenTelemetry](https://opentelemetry.io/)
- [OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0)
- [JSON Schema](https://json-schema.org/)

---

**Copyright © 2025 | Apache License 2.0**

Built with ❤️ for European data sovereignty and open standards.
