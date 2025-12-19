# AgentStack Architecture

AgentStack is a sovereign AI agent platform providing a "Cloud Run-like" experience for AI agents on Kubernetes. This document describes the architecture as implemented in the codebase.

## 🗺️ Architecture Map

This document serves as the master architecture reference. For deep dives into specific subsystems, refer to the following specialized documents:

| Component | Deep Dive Document | Focus Area |
|-----------|-------------------|------------|
| **Agent Runtime** | [kagent-adk-a2a-architecture.md](kagent-adk-a2a-architecture.md) | Kagent, Google ADK, and A2A Protocol |
| **Deployment** | [deployment-guide.md](deployment-guide.md) | Knative, Scaling, and Traffic Management |
| **Operations** | [quick-reference.md](quick-reference.md) | CLI commands, Port-forwarding, and Troubleshooting |
| **Agent Building** | [building-google-adk-agents-for-kagent.md](building-google-adk-agents-for-kagent.md) | SDK usage and Tool definitions |

---

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

- **kagent Integration**: The stack natively uses **kagent** as its primary orchestration layer. If the `kagent.dev/v1alpha2` CRDs are present in the cluster, AgentStack manages agents via the `Agent` custom resource.
    - **Declarative Lifecycle**: Agents are defined as high-level Kubernetes objects.
    - **Unified Management**: Integration with the kagent UI and A2A protocol discovery.
    - See [kagent-adk-a2a-architecture.md](kagent-adk-a2a-architecture.md) for the runtime specification.
- **Knative Fallback**: If `kagent` is not installed, the system falls back to direct management of **Knative Services**.
    - **Scale-to-Zero**: Agents consume zero resources when idle.
    - **Auto-scaling**: Rapid scaling based on request concurrency (via Knative Pod Autoscaler).
    - See [deployment-guide.md](deployment-guide.md) for scaling and traffic management details.
- **Deployment Types**:
    - **BYO (Bring Your Own)**: Custom container images implementing the A2A protocol.
    - **LLM**: Pre-configured agents with specific LLM provider settings.
    - **ADK**: Agents built using the Google Agent Development Kit.

### 3. A2A Protocol (`agentstack/internal/domain/a2a`)
A standardized communication layer based on JSON-RPC 2.0 over HTTP/SSE.
- **Methods**: `message/send`, `message/stream`, `task/get`, `task/cancel`.
- **Streaming**: Real-time event delivery via Server-Sent Events (SSE).
- **Discovery**: `.well-known/agent.json` for agent metadata (Agent Card).
- See [kagent-adk-a2a-architecture.md](kagent-adk-a2a-architecture.md) for the full protocol specification.

### 4. Control Plane (`agentstack/internal/domain/controlplane`)
The Control Plane acts as the brain of the system, coordinating between the API, the database, and the Kubernetes cluster.

- **State Management**: Uses **PostgreSQL** (via `sqlc` generated Go code) to track agent metadata, deployment status, and user configurations.
- **Kubernetes Controller**: A custom controller loop that watches for changes in the database and reconciles the desired state with the cluster (creating/updating `Agent` or `Service` resources).
- **A2A Protocol Bridge**: Facilitates communication between the UI and agents by managing discovery and routing.

### 5. Kagent Web UI Integration
The platform integrates the official Kagent Web UI (`cr.kagent.dev/kagent-dev/kagent/ui`) for cluster administration and agent interaction.

- **Architecture**: The UI is deployed as a standalone Next.js application with an internal Nginx proxy.
- **Multi-Port Forwarding**: To support the full feature set (SSR, A2A Streaming, WebSockets) through a single CLI command, `agentctl ui` performs a quadruple port-forward:
    - `3000`: Main UI entry point.
    - `8080`: Next.js SSR backend (required for model loading).
    - `8083`: A2A/Backend API proxy.
    - `8081`: WebSocket gateway for real-time updates.
- **Streaming Optimization**: The internal Nginx proxy is configured with `proxy_buffering off` and custom `rewrite` rules to handle A2A SSE streams without method-stripping redirects (POST to GET).
- **Cross-Namespace Proxying**: Uses `socat` sidecars within the UI pod to bridge communication between the `agentstack` namespace and the `kagent-controller` in the `kagent` namespace.

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
