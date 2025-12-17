# 002 - Architecture Layers

> 5-Layer Platform Architecture for AgentStack

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Architecture Overview

```text
┌─────────────────────────────────────────────────────────────────────┐
│                         AgentStack Platform                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  LAYER 5: GOVERNANCE                                          │  │
│  │  RBAC │ Quotas │ Audit │ Policy │ Compliance                  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                               │                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  LAYER 4: INTERFACE                                           │  │
│  │  API Gateway │ CLI │ SDK │ UI │ Webhooks                      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                               │                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  LAYER 3: COGNITIVE                                           │  │
│  │  kagent │ A2A Protocol │ MCP Tools │ Memory │ Multi-Agent     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                               │                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  LAYER 2: RUNTIME                                             │  │
│  │  Knative Serving │ Autoscaler │ Activator │ Queue-Proxy       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                               │                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  LAYER 1: INFRASTRUCTURE                                      │  │
│  │  Kubernetes │ Contour/Envoy │ Storage │ Networking            │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Layer 1: Infrastructure

### Components

| Component | Purpose | Implementation |
|-----------|---------|----------------|
| **Kubernetes** | Container orchestration | K8s 1.28+, any distribution |
| **Ingress** | L7 routing, TLS | Contour + Envoy |
| **DNS** | Service discovery | CoreDNS, sslip.io (dev) |
| **Storage** | Persistent data | CSI drivers, S3-compatible |
| **Networking** | CNI, policies | Cilium or Calico |

### Kubernetes Requirements

```yaml
# Minimum cluster requirements
kubernetes:
  version: "1.28+"
  nodes:
    control_plane: 3  # HA production
    workers: 3+
  resources_per_worker:
    cpu: 4
    memory: 16Gi
    storage: 100Gi
  features:
    - Gateway API v1.0
    - CSI drivers
    - RBAC enabled
```

### Ingress Architecture

```text
                    Internet
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                    Cloud Load Balancer                       │
│                    (L4: TCP/UDP)                            │
└─────────────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                    Envoy Proxy                               │
│                    (L7: HTTP/gRPC)                          │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Features:                                            │    │
│  │ • TLS termination (cert-manager)                     │    │
│  │ • Rate limiting                                      │    │
│  │ • Header-based routing                               │    │
│  │ • Circuit breaking                                   │    │
│  │ • Request mirroring                                  │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                        │
                        ▼
                Knative Services
```

### Trade-offs

| Decision | Chosen | Alternative | Rationale |
|----------|--------|-------------|-----------|
| Ingress | Contour | Istio, Kong | Lighter weight, Envoy-native |
| CNI | Cilium | Calico | eBPF performance, observability |
| Storage | CSI | Local PV | Portability across clouds |

---

## 3. Layer 2: Runtime (Knative Serving)

### Core Components

```text
┌─────────────────────────────────────────────────────────────────┐
│                     Knative Serving                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │   Service    │  │   Route      │  │   Configuration      │   │
│  │  (ksvc)      │──│  (Traffic)   │──│   (Desired State)    │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
│         │                                        │               │
│         ▼                                        ▼               │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Revisions (Immutable)                  │   │
│  │   rev-001 (10%)    rev-002 (90%)    rev-003 (0%)         │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │  Activator   │  │  Autoscaler  │  │   Queue-Proxy        │   │
│  │  (Cold start)│  │  (KPA/HPA)   │  │   (Sidecar)          │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Request Flow

```text
                         Request
                            │
                            ▼
┌────────────────────────────────────────────────────────────────┐
│                         Envoy                                   │
│                    (L7 Load Balancer)                          │
└────────────────────────────────────────────────────────────────┘
                            │
              ┌─────────────┴─────────────┐
              │                           │
        pods == 0                   pods > 0
              │                           │
              ▼                           │
┌─────────────────────────┐               │
│       Activator         │               │
│  • Buffers request      │               │
│  • Triggers scale-up    │               │
│  • Retries on timeout   │               │
└─────────────────────────┘               │
              │                           │
              └─────────────┬─────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────────────┐
│                      Queue-Proxy                                │
│  • Concurrency enforcement                                      │
│  • Request metrics collection                                   │
│  • Health check proxy                                          │
└────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────────────┐
│                     Agent Container                             │
│               (Your code @ port 8080)                          │
└────────────────────────────────────────────────────────────────┘
```

### Autoscaling Configuration

```yaml
# Knative Pod Autoscaler (KPA) settings
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: my-agent
  annotations:
    # Scale-to-zero settings
    autoscaling.knative.dev/min-scale: "0"
    autoscaling.knative.dev/max-scale: "100"
    
    # Scaling triggers
    autoscaling.knative.dev/metric: "concurrency"  # or "rps"
    autoscaling.knative.dev/target: "10"
    
    # Scale-down delay (prevents flapping)
    autoscaling.knative.dev/scale-down-delay: "30s"
    
    # Window for scaling decisions
    autoscaling.knative.dev/window: "60s"
```

### Trade-offs

| Decision | Chosen | Alternative | Rationale |
|----------|--------|-------------|-----------|
| Autoscaler | KPA | HPA | Better scale-to-zero, concurrency-based |
| Metric | Concurrency | RPS | Better for LLM workloads (long requests) |
| Cold Start | Accept 2-3s | Keep-warm | Cost vs latency trade-off |

---

## 4. Layer 3: Cognitive (kagent)

### kagent Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                       kagent Namespace                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────┐  ┌──────────────┐  ┌──────────────────┐    │
│  │   Controller    │  │     UI       │  │  KMCP Controller │    │
│  │                 │  │  (Web App)   │  │  (Tool Server)   │    │
│  │  • Watch CRDs   │  │              │  │                  │    │
│  │  • Reconcile    │  │  • Chat      │  │  • MCP Protocol  │    │
│  │  • Manage pods  │  │  • Manage    │  │  • Tool registry │    │
│  └────────┬────────┘  └──────────────┘  └──────────────────┘    │
│           │                                                      │
│           ▼                                                      │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                     Custom Resources                      │   │
│  │  Agent │ ModelConfig │ ToolServer │ Conversation          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                     Agent Pods                            │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────────────┐ │   │
│  │  │ Agent-1 │ │ Agent-2 │ │ Agent-3 │ │ Your BYO Agent  │ │   │
│  │  │ (ADK)   │ │ (ADK)   │ │ (Custom)│ │                 │ │   │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────────────┘ │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Custom Resource Definitions

```yaml
# Agent CRD
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: customer-support
  namespace: agents
spec:
  type: Declarative  # or BYO
  declarative:
    modelConfig: gpt-4o-config
    systemMessage: |
      You are a helpful customer support agent.
      Be concise and friendly.
    tools:
      - mcpServer:
          name: kagent-tools
          tools: [search-kb, create-ticket]
    memory:
      type: conversation
      ttl: 24h

---
# ModelConfig CRD
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: gpt-4o-config
spec:
  provider: openai
  model: gpt-4o
  apiKeySecretRef:
    name: openai-credentials
    key: api-key
  parameters:
    temperature: 0.7
    maxTokens: 4096
```

### A2A Protocol Flow

```text
┌─────────────┐                    ┌─────────────┐
│  Agent A    │                    │  Agent B    │
│ (Requester) │                    │  (Provider) │
└──────┬──────┘                    └──────┬──────┘
       │                                  │
       │  1. Discover Agent B             │
       │─────────────────────────────────►│
       │     GET /.well-known/agent.json  │
       │◄─────────────────────────────────│
       │     {capabilities, endpoints}     │
       │                                  │
       │  2. Send Task                    │
       │─────────────────────────────────►│
       │     POST /a2a/tasks              │
       │     {task_id, input, context}    │
       │◄─────────────────────────────────│
       │     {task_id, status: accepted}  │
       │                                  │
       │  3. Stream Results (SSE)         │
       │─────────────────────────────────►│
       │     GET /a2a/tasks/{id}/stream   │
       │◄─────────────────────────────────│
       │     event: progress              │
       │     event: result                │
       │     event: done                  │
       │                                  │
```

---

## 5. Layer 4: Interface

### API Gateway

```text
┌─────────────────────────────────────────────────────────────────┐
│                        API Gateway                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Public Endpoints                       │   │
│  │                                                           │   │
│  │  /v1/agents/*        → Agent CRUD, deployments           │   │
│  │  /v1/chat/*          → Chat interactions                 │   │
│  │  /v1/tools/*         → Tool management                   │   │
│  │  /v1/projects/*      → Project management                │   │
│  │  /v1/analytics/*     → Usage metrics                     │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Middleware Stack                       │   │
│  │                                                           │   │
│  │  1. Request ID injection                                  │   │
│  │  2. Authentication (JWT / API Key)                        │   │
│  │  3. Rate limiting                                         │   │
│  │  4. Project context resolution                            │   │
│  │  5. Request logging                                       │   │
│  │  6. CORS handling                                         │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Interface Components

| Interface | Purpose | Users |
|-----------|---------|-------|
| **REST API** | Programmatic access | All |
| **CLI** | Developer workflows | AI Engineers |
| **SDK** | Application integration | App Developers |
| **UI** | Visual management | All |
| **Webhooks** | Event notifications | Systems |

---

## 6. Layer 5: Governance

### RBAC Model

```text
┌─────────────────────────────────────────────────────────────────┐
│                      RBAC Hierarchy                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Organization (Tenant)                                          │
│  └── Team                                                       │
│      ├── Owner   → Full control                                 │
│      ├── Admin   → Manage members, settings                     │
│      ├── Member  → Deploy, manage agents                        │
│      └── Viewer  → Read-only access                             │
│                                                                  │
│  Project (within Team)                                          │
│  └── Agents, Secrets, Tools, Deployments                        │
│                                                                  │
│  Resource-level permissions:                                     │
│  ├── agents:read, agents:write, agents:delete                   │
│  ├── secrets:read, secrets:write                                │
│  ├── deployments:create, deployments:rollback                   │
│  └── analytics:read                                             │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Governance Components

| Component | Purpose |
|-----------|---------|
| **Quotas** | Limit agents, requests, tokens per project |
| **Audit Log** | Track all API calls and changes |
| **Policy** | OPA/Gatekeeper for K8s policies |
| **Compliance** | SOC2, GDPR data handling |

---

## 7. Cross-Cutting Concerns

### Networking

```yaml
# Network flow for agent request
request_path:
  1_ingress: "Envoy (TLS termination)"
  2_routing: "Knative Activator/Route"
  3_sidecar: "Queue-Proxy (metrics)"
  4_container: "Agent (port 8080)"
  
# Internal service mesh (optional)
service_mesh:
  enabled: false  # Trade-off: complexity vs features
  alternative: "Knative internal networking"
```

### Configuration Management

```yaml
# Configuration sources (precedence high→low)
config_sources:
  1: "Runtime API calls"
  2: "Environment variables"
  3: "ConfigMaps"
  4: "Agent spec in CRD"
  5: "Platform defaults"
```

---

## 8. Deployment Topologies

### Single Cluster (Development/Small)

```text
┌─────────────────────────────────────────┐
│           Single K8s Cluster            │
│  ┌─────────────────────────────────┐   │
│  │  kagent + Knative + Agents      │   │
│  └─────────────────────────────────┘   │
│  ┌─────────────────────────────────┐   │
│  │  PostgreSQL + Redis             │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

### Multi-Cluster (Production)

```text
┌─────────────────┐     ┌─────────────────┐
│  Control Plane  │     │   Data Plane    │
│  Cluster        │     │   Cluster(s)    │
│  ┌───────────┐  │     │  ┌───────────┐  │
│  │ API       │◄─┼─────┼─►│ Agents    │  │
│  │ UI        │  │     │  │ Knative   │  │
│  │ Controller│  │     │  │ kagent    │  │
│  └───────────┘  │     │  └───────────┘  │
└─────────────────┘     └─────────────────┘
         │                      │
         └──────────┬───────────┘
                    ▼
         ┌─────────────────────┐
         │   Shared Services   │
         │  PostgreSQL, Redis  │
         │  Object Storage     │
         └─────────────────────┘
```

---

## 9. References

- [Knative Architecture](https://knative.dev/docs/serving/architecture/)
- [kagent Architecture](https://kagent.dev/docs/architecture)
- [Envoy Proxy](https://www.envoyproxy.io/docs)
- [Kubernetes Architecture](https://kubernetes.io/docs/concepts/architecture/)

---

**Previous**: [001-platform-overview.md](001-platform-overview.md)  
**Next**: [003-agent-lifecycle.md](003-agent-lifecycle.md)
