# Architecture Overview

k8s-agent-stack is a multi-layer platform for deploying sovereign AI agents on Kubernetes.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         k8s-agent-stack                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │            YOUR AGENTS                                      │   │
│   │   Google ADK  │  LangGraph  │  CrewAI  │  Custom BYO        │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│   ┌──────────────────────────▼──────────────────────────────────┐   │
│   │                      kagent                                 │   │
│   │   • Agent CRDs     • A2A Protocol    • MCP Tools           │   │
│   │   • UI Dashboard   • Multi-LLM       • Lifecycle Mgmt      │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│   ┌──────────────────────────▼──────────────────────────────────┐   │
│   │                  Knative Serving                            │   │
│   │   • Scale-to-Zero   • Auto-Scaling   • Traffic Management   │   │
│   │   • Revisions       • Canary Deploys • Request Buffering    │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│   ┌──────────────────────────▼──────────────────────────────────┐   │
│   │                 Contour + Envoy                             │   │
│   │   • L7 Routing     • Load Balancing   • TLS Termination    │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│   ┌──────────────────────────▼──────────────────────────────────┐   │
│   │                    Kubernetes                               │   │
│   │   OrbStack  │  GKE  │  EKS  │  AKS  │  On-Premises         │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Component Overview

### Layer 1: Kubernetes (Infrastructure)

| Component | Purpose | Status |
|-----------|---------|--------|
| **Kubernetes** | Container orchestration | ✅ Production |
| **OrbStack** | Local development (macOS) | ✅ Recommended |
| **kind/minikube** | Alternative local options | ✅ Supported |
| **GKE/EKS/AKS** | Cloud production | ✅ Supported |

### Layer 2: Ingress (Contour + Envoy)

| Component | Purpose | Status |
|-----------|---------|--------|
| **Contour** | Ingress controller | ✅ Production |
| **Envoy** | L7 proxy, load balancer | ✅ Production |
| **sslip.io** | Magic DNS for local dev | ✅ Production |

### Layer 3: Serverless (Knative Serving)

| Component | Purpose | Status |
|-----------|---------|--------|
| **Knative Service** | Agent deployment abstraction | ✅ Production |
| **Activator** | Buffers requests during scale-up | ✅ Production |
| **Autoscaler** | Scales pods based on metrics | ✅ Production |
| **Queue-Proxy** | Sidecar for metrics collection | ✅ Production |

### Layer 4: Agent Orchestration (kagent)

| Component | Purpose | Status |
|-----------|---------|--------|
| **kagent Controller** | Agent lifecycle management | ✅ Production |
| **kagent UI** | Web dashboard | ✅ Production |
| **Agent CRD** | Declarative agent definition | ✅ Production |
| **ModelConfig CRD** | LLM provider configuration | ✅ Production |
| **ToolServer CRD** | MCP tool integration | ✅ Production |

### Layer 5: Agent Frameworks

| Framework | Purpose | Status |
|-----------|---------|--------|
| **Google ADK** | Agent development kit | ✅ Production |
| **LangGraph** | Complex workflows | 📋 Planned |
| **CrewAI** | Multi-agent orchestration | 📋 Planned |
| **Custom BYO** | Bring your own container | ✅ Production |

---

## Request Flow

```
                                User Request
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────┐
│                              Envoy                                  │
│                    (L7 Load Balancer @ :80)                         │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │                           │
              If pods = 0                  If pods > 0
                    │                           │
                    ▼                           │
┌─────────────────────────────────┐             │
│           Activator             │             │
│   (Buffers & triggers scale)    │             │
└─────────────────────────────────┘             │
                    │                           │
                    └─────────────┬─────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Queue-Proxy                                │
│               (Sidecar in each pod, reports metrics)                │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Your Agent                                 │
│              (Google ADK / Custom Container @ :8080)                │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Agent Types

### Declarative Agents

Defined entirely in YAML. kagent manages the container.

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: my-agent
spec:
  type: Declarative
  declarative:
    modelConfig: default-model-config
    systemMessage: "You are a helpful assistant"
    tools:
      - mcpServer:
          name: kagent-tools
```

### BYO (Bring Your Own) Agents

You provide the container image. kagent manages deployment.

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: my-custom-agent
spec:
  type: BYO
  byo:
    deployment:
      image: dev.local/my-agent:v1
      resources:
        requests:
          cpu: 250m
          memory: 512Mi
```

---

## kagent Components

```
┌─────────────────────────────────────────────────────────────────────┐
│                         kagent Namespace                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │
│  │   Controller    │  │      UI         │  │   KMCP Controller   │  │
│  │  (Reconciles    │  │  (Web Dashboard │  │  (MCP Tool Server   │  │
│  │   Agent CRDs)   │  │   @ :8080)      │  │    Management)      │  │
│  └────────┬────────┘  └─────────────────┘  └─────────────────────┘  │
│           │                                                         │
│           ▼                                                         │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                     Agent Pods                              │    │
│  │  k8s-agent │ helm-agent │ istio-agent │ your-custom-agent   │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    Tool Servers                             │    │
│  │  kagent-tools │ grafana-mcp │ querydoc                      │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Custom Resource Definitions (CRDs)

| CRD | Purpose |
|-----|---------|
| `Agent` | Defines an AI agent (Declarative or BYO) |
| `ModelConfig` | LLM provider configuration (OpenAI, Anthropic, etc.) |
| `ToolServer` | MCP tool server definition |
| `RemoteMCPServer` | External MCP server connection |
| `Memory` | Agent memory/state configuration |
| `MCPServer` | Managed MCP server definition |

---

## Scaling Behavior

```
         Requests/sec
              │
    10 ───────┼─────────────────────────────────── Max Scale (10)
              │                    ████████████
     8 ───────┼───────────────────█████████████
              │              █████████████████
     6 ───────┼─────────────██████████████████
              │         ████████████████████
     4 ───────┼────────█████████████████████
              │    ████████████████████████
     2 ───────┼───█████████████████████████
              │  ██████████████████████████
     0 ───────┼█████████████████████████████──── Min Scale (0 or 1)
              │
              └────────────────────────────────── Time
               Scale from Zero  │  Under Load  │  Scale Down
               (~100-500ms)     │  (Instant)   │  (60s delay)
```

**Key Settings:**
- `minScale: 0` - Scale to zero when idle (cost savings)
- `minScale: 1` - Keep warm pod (no cold starts)
- `maxScale: 10` - Maximum concurrent pods
- `concurrency-target: 10` - Requests per pod before scaling

---

## Security Model

```
┌──────────────────────────────────────────────────────────────┐
│                      Kubernetes RBAC                         │
├──────────────────────────────────────────────────────────────┤
│  Namespace: kagent                                           │
│  ├── ServiceAccount: kagent-controller                       │
│  │   └── ClusterRole: manages Agent, ModelConfig, etc.      │
│  ├── ServiceAccount: google-adk-agent                        │
│  │   └── Role: limited to configmaps in kagent              │
│  └── Secret: openai-api-key                                  │
│      └── Referenced by ModelConfig                           │
└──────────────────────────────────────────────────────────────┘
```

---

## Network Architecture

| Component | Ports | Access |
|-----------|-------|--------|
| **Envoy** | 80, 443 | External (LoadBalancer) |
| **Knative Services** | 80 | Via Envoy only |
| **kagent UI** | 8080 | Port-forward |
| **kagent Controller** | 8083 | Cluster internal |
| **Agent Pods** | 8080 | Via Queue-Proxy |

---

## Next Steps

- [Getting Started](getting-started.md) - Quick installation
- [Deployment Guide](deployment-guide.md) - Deploy custom agents
- [Quick Reference](quick-reference.md) - Common commands

---

[← Getting Started](getting-started.md) | [Deployment Guide →](deployment-guide.md)
