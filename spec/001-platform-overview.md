# 001 - Platform Overview

> AgentStack: Sovereign GenAI Agent Platform

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Vision & Mission

### Mission Statement
Build the **Sovereign AGI Platform** for Europe and free nations—enabling organizations to deploy, orchestrate, and scale AI agents on their own infrastructure with full data sovereignty.

### Strategic Goals

| Goal | Description | Success Metric |
|------|-------------|----------------|
| **Sovereignty** | Zero vendor lock-in, run anywhere | 100% on-prem capable |
| **Simplicity** | Vercel-like DX for agents | Deploy in < 2 minutes |
| **Scale** | Enterprise-grade reliability | 99.9% SLA, 10K+ agents |
| **Open** | Apache 2.0, CNCF ecosystem | Community-driven roadmap |

---

## 2. Platform Positioning

```text
┌──────────────────────────────────────────────────────────────────┐
│                     AGENT PLATFORMS LANDSCAPE                    │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Proprietary ◄─────────────────────────────────► Open Source   │
│                                                                  │
│   ┌─────────────┐                        ┌─────────────────┐    │
│   │ Vertex AI   │                        │   AgentStack    │    │
│   │ Agent       │                        │   (This)        │    │
│   └─────────────┘                        └─────────────────┘    │
│   ┌─────────────┐                        ┌─────────────────┐    │
│   │ Azure AI    │                        │   kagent        │    │
│   │ Agents      │                        │   (Component)   │    │
│   └─────────────┘                        └─────────────────┘    │
│   ┌─────────────┐                        ┌─────────────────┐    │
│   │ AWS         │                        │   LangServe     │    │
│   │ Bedrock     │                        │                 │    │
│   └─────────────┘                        └─────────────────┘    │
│                                                                  │
│   Simple ◄───────────────────────────────────────► Full Control │
└──────────────────────────────────────────────────────────────────┘
```

### Competitive Differentiation

| Capability | AgentStack | Cloud Vendors | OSS Frameworks |
|------------|------------|---------------|----------------|
| Run Anywhere | ✅ | ❌ Locked | ✅ |
| Scale-to-Zero | ✅ | ⚠️ Varies | ❌ DIY |
| A2A Protocol | ✅ Native | ❌ Proprietary | ⚠️ Limited |
| Multi-Framework | ✅ ADK, LangGraph, CrewAI | ❌ Limited | ✅ |
| Production UI | ✅ kagent UI | ✅ | ❌ |
| Enterprise RBAC | ✅ | ✅ | ❌ DIY |

---

## 3. Core Technology Stack

### Component Selection Criteria

All components must satisfy:
1. **License**: Apache 2.0 or compatible permissive license
2. **Maturity**: CNCF Graduated/Incubating or production-proven
3. **Community**: Active maintainers, >1000 GitHub stars
4. **Extensibility**: Plugin/extension architecture

### Selected Components

```text
┌─────────────────────────────────────────────────────────────────┐
│                        TECHNOLOGY STACK                          │
├──────────────────────┬──────────────────────────────────────────┤
│ LAYER                │ COMPONENTS                               │
├──────────────────────┼──────────────────────────────────────────┤
│ Agent Frameworks     │ Google ADK, LangGraph, CrewAI, AutoGen   │
├──────────────────────┼──────────────────────────────────────────┤
│ Agent Orchestration  │ kagent (CNCF Sandbox)                    │
├──────────────────────┼──────────────────────────────────────────┤
│ Evaluation & Safety  │ MLflow 3.x (Tracing, Scorers, Judges)    │
├──────────────────────┼──────────────────────────────────────────┤
│ Serverless Runtime   │ Knative Serving 1.20+                    │
├──────────────────────┼──────────────────────────────────────────┤
│ Ingress/Gateway      │ Contour + Envoy (Gateway API)            │
├──────────────────────┼──────────────────────────────────────────┤
│ Certificate Mgmt     │ cert-manager                             │
├──────────────────────┼──────────────────────────────────────────┤
│ Observability        │ OpenTelemetry, Prometheus, Grafana       │
├──────────────────────┼──────────────────────────────────────────┤
│ Data                 │ PostgreSQL, Redis, S3-compatible         │
├──────────────────────┼──────────────────────────────────────────┤
│ Container Runtime    │ containerd                               │
├──────────────────────┼──────────────────────────────────────────┤
│ Kubernetes           │ 1.28+ (any distribution)                 │
└──────────────────────┴──────────────────────────────────────────┘
```

---

## 4. Design Principles

### 4.1 Sovereignty First

```yaml
# Every deployment option must support:
deployment_modes:
  - local:      # Developer laptop (OrbStack, kind, minikube)
  - on_prem:    # Enterprise data center
  - private_cloud: # VPC in any cloud
  - air_gapped: # No internet required
  - hybrid:     # Mix of above
```

**Trade-off**: Accepting slightly more operational complexity in exchange for complete data control.

### 4.2 Progressive Disclosure

```text
Level 1: agentstack deploy              # Just works
Level 2: agentstack deploy --config     # Customize behavior  
Level 3: kubectl apply -f agent.yaml    # Full control
Level 4: Helm chart with 100+ values    # Enterprise tuning
```

### 4.3 Observable by Default

Every agent automatically gets:
- Distributed tracing (OpenTelemetry + MLflow)
- Metrics (Prometheus format)
- Structured logging (JSON)
- Cost tracking (token usage)
- **Safety evaluation (MLflow scorers)**

### 4.4 Evaluation-First Safety

> **Agents cannot be deployed without passing evaluation gates.**

| Principle | Implementation |
|-----------|----------------|
| **Pre-deploy Eval** | All agents must pass safety/quality scorers |
| **Continuous Eval** | Production traces evaluated in real-time |
| **Quality Gates** | Configurable thresholds block unsafe deploys |
| **Human Feedback** | Expert review loop improves scorers |

```yaml
# Every deployment requires evaluation
evaluation:
  required: true
  minimumScores:
    safety: 1.0        # 100% pass rate mandatory
    correctness: 0.85  # 85% minimum
  blockOnFailure: true
```

### 4.5 Fail-Safe Defaults

| Setting | Default | Rationale |
|---------|---------|-----------|
| `minInstances` | 0 | Cost optimization (scale-to-zero) |
| `maxInstances` | 10 | Prevent runaway scaling |
| `timeout` | 5m | Reasonable for LLM calls |
| `memory` | 512Mi | Sufficient for most agents |
| `concurrency` | 10 | Balance throughput/latency |

---

## 5. Platform Capabilities

### 5.1 Agent Management

| Capability | Description |
|------------|-------------|
| **Declarative Agents** | YAML-defined, kagent manages runtime |
| **BYO Agents** | Bring your container, platform manages lifecycle |
| **Multi-Framework** | ADK, LangGraph, CrewAI, custom |
| **Versioning** | Immutable revisions, rollback support |
| **Traffic Splitting** | Canary, blue-green, A/B testing |

### 5.2 Agent Communication

| Protocol | Use Case |
|----------|----------|
| **A2A (Agent-to-Agent)** | Inter-agent communication |
| **MCP (Model Context Protocol)** | Tool integration |
| **HTTP/REST** | External API integration |
| **CloudEvents** | Event-driven workflows |

### 5.3 Agent Evaluation & Safety

| Capability | Implementation |
|------------|----------------|
| **Tracing** | MLflow + OpenTelemetry auto-instrumentation |
| **LLM-as-Judge** | Safety, Correctness, Hallucination scorers |
| **Quality Gates** | Pre-deploy and canary evaluation |
| **Datasets** | Versioned evaluation test sets |
| **Human Feedback** | Expert annotation for scorer alignment |
| **Continuous Eval** | Production trace monitoring |

### 5.4 Operations

| Capability | Implementation |
|------------|----------------|
| **Auto-scaling** | Knative KPA (0→N→0) |
| **Load Balancing** | Envoy L7 |
| **Health Checks** | Kubernetes probes |
| **Secrets** | Kubernetes Secrets + external-secrets |
| **CI/CD** | GitOps (Argo CD, Flux) |

---

## 6. Target Users

### Primary Personas

```text
┌─────────────────────────────────────────────────────────────────┐
│                         USER PERSONAS                            │
├─────────────────┬───────────────────────────────────────────────┤
│ PERSONA         │ NEEDS                                         │
├─────────────────┼───────────────────────────────────────────────┤
│ AI Engineer     │ Build agents, test locally, deploy fast       │
│                 │ → Uses: CLI, SDK, local dev                   │
├─────────────────┼───────────────────────────────────────────────┤
│ Platform Eng    │ Operate platform, manage quotas, security     │
│                 │ → Uses: Helm, GitOps, dashboards              │
├─────────────────┼───────────────────────────────────────────────┤
│ App Developer   │ Integrate agents into applications            │
│                 │ → Uses: REST API, SDKs                        │
├─────────────────┼───────────────────────────────────────────────┤
│ Enterprise Ops  │ Compliance, auditing, cost control            │
│                 │ → Uses: Admin UI, reports, RBAC               │
└─────────────────┴───────────────────────────────────────────────┘
```

---

## 7. Success Metrics

### Platform KPIs

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Cold Start** | < 3s | P95 time from 0→1 pod |
| **Deployment** | < 60s | Code push to live |
| **Availability** | 99.9% | Platform uptime |
| **Agent Density** | 100/node | Agents per K8s node |
| **Cost Efficiency** | < $0.001/request | For scale-to-zero agents |

### Safety & Evaluation KPIs

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Safety Score** | ≥ 99.9% | % passing safety evaluation |
| **Correctness Score** | ≥ 85% | % factually correct responses |
| **Eval Coverage** | 100% | Agents with pre-deploy eval |
| **Eval Latency** | < 30s | P95 evaluation pipeline time |
| **Scorer Alignment** | ≥ 90% | Agreement with human judges |

### Developer Experience KPIs

| Metric | Target |
|--------|--------|
| Time to first agent | < 5 minutes |
| Lines of YAML for basic agent | < 20 |
| CLI commands to deploy | 1 (`agentstack deploy`) |

---

## 8. References

### Official Documentation
- [Knative Serving](https://knative.dev/docs/serving/)
- [kagent](https://kagent.dev/docs/)
- [Kubernetes](https://kubernetes.io/docs/)
- [Contour](https://projectcontour.io/docs/)
- [OpenTelemetry](https://opentelemetry.io/docs/)
- [MLflow GenAI](https://mlflow.org/docs/latest/genai/)
- [MLflow Evaluation](https://mlflow.org/docs/latest/genai/eval-monitor/)
- [MLflow Tracing](https://mlflow.org/docs/latest/genai/tracing/)

### Standards
- [CloudEvents 1.0](https://cloudevents.io/)
- [OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0)
- [RFC 7807 Problem Details](https://datatracker.ietf.org/doc/html/rfc7807)

---

**Next**: [002-architecture-layers.md](002-architecture-layers.md) - Detailed 5-layer architecture
