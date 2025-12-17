# AgentStack Platform Specifications

> **Mission**: Build the Sovereign GenAI Agent Platform for Europe and free nations.

## Overview

AgentStack is a Kubernetes-native Platform-as-a-Service for deploying, orchestrating, and scaling AI agents. Built on CNCF open-source components (Apache 2.0), it provides data sovereignty, vendor independence, and production-grade reliability.

## Specification Documents

| Document | Description | Status |
|----------|-------------|--------|
| [001-platform-overview](001-platform-overview.md) | Vision, principles, component stack | ✅ Complete |
| [002-architecture-layers](002-architecture-layers.md) | 5-layer architecture, request flow | ✅ Complete |
| [003-agent-lifecycle](003-agent-lifecycle.md) | Agent types, lifecycle, A2A protocol | ✅ Complete |
| [004-api-design](004-api-design.md) | OpenAPI spec, Go implementation | ✅ Complete |
| [005-data-architecture](005-data-architecture.md) | Storage, state, memory patterns | ✅ Complete |
| [006-security-governance](006-security-governance.md) | AuthN/AuthZ, secrets, compliance | ✅ Complete |
| [007-observability](007-observability.md) | Metrics, logging, tracing | ✅ Complete |
| [008-deployment-operations](008-deployment-operations.md) | CI/CD, scaling, multi-cloud | ✅ Complete |
| [009-developer-experience](009-developer-experience.md) | CLI, SDK, local dev | ✅ Complete |

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

## Quick Reference

```text
┌─────────────────────────────────────────────────────────────────┐
│                    AgentStack Platform                          │
├─────────────────────────────────────────────────────────────────┤
│  Layer 5: Governance    │ RBAC, Quotas, Audit, Compliance       │
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
| **PostgreSQL** | Primary database | PostgreSQL License | - |
| **Redis** | Cache/Queue | BSD-3 | - |

## Design Principles

1. **Sovereignty First**: Run anywhere—your infrastructure, your data
2. **Open Standards**: CloudEvents, OpenTelemetry, A2A Protocol
3. **Progressive Disclosure**: Simple defaults, powerful overrides
4. **Observable by Default**: Traces, metrics, logs built-in
5. **Developer Experience**: Vercel-like simplicity

## Official References

- [Knative Documentation](https://knative.dev/docs/)
- [kagent Documentation](https://kagent.dev/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [CloudEvents Specification](https://cloudevents.io/)
- [OpenTelemetry](https://opentelemetry.io/)

---

**Copyright © 2025 | Apache License 2.0**
