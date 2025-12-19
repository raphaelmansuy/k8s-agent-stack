# AgentStack Architecture

AgentStack is a sovereign AI agent platform providing a "Cloud Run-like" experience for AI agents on Kubernetes. This document describes the architecture as implemented in the codebase.

## System Overview

```ascii
                                 +-------------------+
                                 |   User / Client   |
                                 +---------+---------+
                                           |
                                           | HTTP/JSON-RPC
                                           v
+---------------------------------------------------------------------------------------+
|                                  AgentStack API Gateway                               |
|  +-------------------+  +-------------------+  +-------------------+  +------------+  |
|  |   Auth & RBAC     |  |   Quota & Audit   |  |   A2A Protocol    |  | Deployment |  |
|  +---------+---------+  +---------+---------+  +---------+---------+  +-----+------+  |
|            |                      |                      |                  |         |
+------------|----------------------|----------------------|------------------|---------+
             |                      |                      |                  |         |
             v                      v                      v                  v
+-----------------------+  +-----------------------+  +---------------------------------+
|      PostgreSQL       |  |         Redis         |  |       Kubernetes / Knative      |
| (Multi-tenant State)  |  | (Cache / Task Queue)  |  | (Agent Runtime / Auto-scaling)  |
+-----------------------+  +-----------------------+  +---------------------------------+
             ^                                                 ^
             |                                                 |
             +-----------------------+-------------------------+
                                     |
                          +----------v----------+
                          |       MLflow        |
                          | (Evaluation/Traces) |
                          +---------------------+
```

## Core Components

### 1. API Gateway (`agentstack/cmd/api`)
The central entry point for all operations. Built with Go, `chi`, and `huma`.
- **Auth**: Implements JWT and API Key verification (`internal/api/middleware/auth.go`).
- **RBAC**: Resource-based access control for Teams and Projects (`internal/domain/rbac`).
- **Quota**: Rate limiting and resource usage tracking (`internal/domain/quota`).
- **Audit**: Event-driven audit logging for all administrative actions (`internal/domain/audit`).

### 2. Agent Runtime (`agentstack/internal/domain/deployment`)
Agents are managed through a dual-mode orchestration strategy that leverages both **kagent** and **Knative Serving**.

- **kagent Integration**: The stack natively uses **kagent** as its primary orchestration layer. If the `kagent.dev/v1alpha2` CRDs are present in the cluster, AgentStack manages agents via the `Agent` custom resource. This provides:
    - **Declarative Lifecycle**: Agents are defined as high-level Kubernetes objects.
    - **Unified Management**: Integration with the kagent UI and A2A protocol discovery.
- **Knative Fallback**: If `kagent` is not installed, the system falls back to direct management of **Knative Services**. This ensures:
    - **Scale-to-Zero**: Agents consume zero resources when idle.
    - **Auto-scaling**: Rapid scaling based on request concurrency (via Knative Pod Autoscaler).
- **Deployment Types**:
    - **BYO (Bring Your Own)**: Custom container images implementing the A2A protocol.
    - **LLM**: Pre-configured agents with specific LLM provider settings.
    - **ADK**: Agents built using the Google Agent Development Kit.

### 3. A2A Protocol (`agentstack/internal/domain/a2a`)
A standardized communication layer based on JSON-RPC 2.0 over HTTP/SSE.
- **Methods**: `message/send`, `message/stream`, `task/get`, `task/cancel`.
- **Streaming**: Real-time event delivery via Server-Sent Events (SSE).
- **Discovery**: `.well-known/agent.json` for agent metadata (Agent Card).

## Data Flow: Request Lifecycle

```ascii
1. Request  --> [ API Gateway ] --> 2. Auth/RBAC Check (Postgres/Redis)
                                       |
                                       v
4. Response <-- [ Agent Pod ] <--- 3. Knative Ingress (Envoy)
      ^             |
      |             +-------------> 5. Telemetry (OTEL)
      +---------------------------> 6. Audit/Quota Update (Postgres)
```

## Agent Implementation (`kagent-adk-agent`)
Reference implementation using Google ADK and FastAPI.
- **A2A Middleware**: Intercepts standard requests to provide A2A-compliant JSON-RPC and SSE streaming.
- **Runner**: Orchestrates the agent's reasoning loop and tool execution.
- **Telemetry**: Integrated OpenTelemetry for distributed tracing.

## Infrastructure Stack
- **Database**: PostgreSQL (via `pgx` and `sqlc`) for persistent multi-tenant data.
- **Cache**: Redis for fast quota checks and background task queuing (Evaluation).
- **Observability**: OpenTelemetry for tracing and metrics; MLflow for agent evaluation and trace tracking.

## kagent Usage
Yes, the stack is built around **kagent**. It serves as the "Cognitive Layer" of the platform, providing the standardized A2A (Agent-to-Agent) communication protocol and the declarative `Agent` CRD used for deployment. The `Makefile` includes targets for installing the `kagent` CLI and setting up the controller in the cluster.
