# 003 - Agent Lifecycle

> Agent Types, States, and Communication Protocols

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Agent Types

### 1.1 Declarative Agents

Fully managed by kagent. You define behavior in YAML; platform handles runtime.

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: support-agent
  namespace: production
spec:
  type: Declarative
  declarative:
    modelConfig: gpt-4o-config
    systemMessage: |
      You are a customer support agent for Acme Corp.
      - Be helpful and concise
      - Escalate billing issues to humans
      - Never share internal data
    tools:
      - mcpServer:
          name: kagent-tools
          tools:
            - search-knowledge-base
            - create-support-ticket
            - lookup-order
    memory:
      type: conversation
      ttl: 24h
```

**Use When**: Simple agents, rapid prototyping, no custom code needed.

### 1.2 BYO (Bring Your Own) Agents

You provide the container; kagent manages deployment and lifecycle.

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: ml-pipeline-agent
spec:
  type: BYO
  byo:
    deployment:
      image: gcr.io/myproject/ml-agent:v2.1.0
      resources:
        requests:
          cpu: 500m
          memory: 1Gi
          nvidia.com/gpu: 1  # GPU support
        limits:
          cpu: 2
          memory: 4Gi
      env:
        - name: MODEL_PATH
          value: /models/latest
      ports:
        - containerPort: 8080
          name: http
```

**Use When**: Custom frameworks, proprietary logic, GPU workloads.

### 1.3 Comparison Matrix

| Aspect | Declarative | BYO |
|--------|-------------|-----|
| **Code Required** | None | Yes (container) |
| **Framework** | Google ADK (built-in) | Any |
| **Customization** | Limited to config | Unlimited |
| **GPU Support** | ❌ | ✅ |
| **Cold Start** | ~2s | Depends on image |
| **Observability** | Automatic | Manual integration |

---

## 2. Agent State Machine

```text
┌─────────────────────────────────────────────────────────────────┐
│                     Agent State Machine                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                      ┌──────────────┐                           │
│                      │   CREATING   │                           │
│                      └──────┬───────┘                           │
│                             │                                    │
│              ┌──────────────┼──────────────┐                    │
│              │              │              │                    │
│              ▼              ▼              ▼                    │
│       ┌──────────┐   ┌──────────┐   ┌──────────┐               │
│       │  FAILED  │   │ INACTIVE │   │  ACTIVE  │◄──┐           │
│       └──────────┘   └────┬─────┘   └────┬─────┘   │           │
│                           │              │          │           │
│                           │  deploy      │          │           │
│                           ▼              │          │           │
│                    ┌──────────────┐      │          │           │
│                    │  DEPLOYING   │──────┘          │           │
│                    └──────────────┘                 │           │
│                                                     │           │
│                    ┌──────────────┐                 │           │
│                    │  SUSPENDED   │◄────────────────┤           │
│                    └──────┬───────┘                 │           │
│                           │ resume                  │ suspend   │
│                           └─────────────────────────┘           │
│                                                                  │
│                    ┌──────────────┐                              │
│                    │   DELETING   │ ──► (removed)               │
│                    └──────────────┘                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### State Definitions

| State | Description | Actions Available |
|-------|-------------|-------------------|
| `CREATING` | Agent resource created, validating | Wait |
| `INACTIVE` | Valid but not deployed | Deploy |
| `DEPLOYING` | Building/deploying | Cancel, Watch |
| `ACTIVE` | Running, receiving traffic | Update, Suspend, Delete |
| `SUSPENDED` | Paused, no traffic | Resume, Delete |
| `FAILED` | Error state | Retry, Delete |
| `DELETING` | Graceful shutdown | Wait |

---

## 3. Deployment Lifecycle

### 3.1 Deployment Flow

```text
┌─────────────────────────────────────────────────────────────────┐
│                     Deployment Pipeline                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. TRIGGER                                                      │
│  ├── CLI: agentstack deploy                                     │
│  ├── API: POST /v1/agents/{id}/deployments                      │
│  └── GitOps: Commit to main branch                              │
│                                                                  │
│  2. QUEUE                                                        │
│  └── Deployment request queued                                   │
│                                                                  │
│  3. BUILD (if needed)                                            │
│  ├── Pull source code                                            │
│  ├── Build container image                                       │
│  ├── Push to registry                                            │
│  └── Security scan                                               │
│                                                                  │
│  4. DEPLOY                                                       │
│  ├── Create Knative Revision                                     │
│  ├── Wait for pod ready                                          │
│  └── Run health checks                                           │
│                                                                  │
│  5. TRAFFIC                                                      │
│  ├── Update route (0% → 100% or gradual)                        │
│  └── Previous revision scaled down                               │
│                                                                  │
│  6. COMPLETE                                                     │
│  ├── Emit deployment.succeeded event                             │
│  └── Update agent status                                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Revision Management

```text
Agent: customer-support
│
├── Revision: rev-001 (0% traffic)
│   └── Created: 2025-01-10, Git: abc123
│
├── Revision: rev-002 (10% traffic) ← Canary
│   └── Created: 2025-01-15, Git: def456
│
└── Revision: rev-003 (90% traffic) ← Stable
    └── Created: 2025-01-14, Git: def456
```

### 3.3 Traffic Splitting Strategies

```yaml
# Canary deployment (10% traffic to new version)
traffic:
  - revision: rev-003
    percent: 90
    tag: stable
  - revision: rev-002
    percent: 10
    tag: canary

# Blue-Green (instant switch)
traffic:
  - revision: rev-003
    percent: 100
    tag: live
  - revision: rev-002
    percent: 0
    tag: previous

# A/B Testing (header-based)
traffic:
  - revision: rev-003
    percent: 100
  - revision: rev-002
    percent: 0
    tag: beta  # Access via agent-beta.example.com
```

---

## 4. A2A Protocol (Agent-to-Agent)

> **Full Specification**: See [api/017-a2a-protocol.md](api/017-a2a-protocol.md) for complete A2A implementation details.

### 4.1 Protocol Overview

The A2A protocol enables agents to discover and communicate with each other.

```text
┌─────────────────────────────────────────────────────────────────┐
│                      A2A Protocol Stack                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Application   │  Task messages, context, results               │
│  ──────────────┼────────────────────────────────────────────── │
│  A2A Protocol  │  Discovery, task lifecycle, streaming          │
│  ──────────────┼────────────────────────────────────────────── │
│  Transport     │  HTTP/1.1, HTTP/2, SSE                         │
│  ──────────────┼────────────────────────────────────────────── │
│  Security      │  mTLS, JWT, API Keys                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Discovery

Every agent exposes a discovery endpoint:

```http
GET /.well-known/agent.json
```

```json
{
  "id": "agent:customer-support",
  "name": "Customer Support Agent",
  "version": "1.0.0",
  "capabilities": [
    "answer-questions",
    "create-tickets",
    "search-knowledge-base"
  ],
  "endpoints": {
    "chat": "/a2a/chat",
    "tasks": "/a2a/tasks",
    "health": "/health"
  },
  "authentication": {
    "methods": ["bearer", "api-key"]
  },
  "metadata": {
    "owner": "support-team",
    "sla": "99.9%"
  }
}
```

### 4.3 Task Lifecycle

```text
┌─────────────┐                              ┌─────────────┐
│  Requester  │                              │   Provider  │
└──────┬──────┘                              └──────┬──────┘
       │                                            │
       │  POST /a2a/tasks                           │
       │  {                                         │
       │    "task_id": "task-123",                  │
       │    "type": "answer-question",              │
       │    "input": {                              │
       │      "question": "What is your policy?"    │
       │    },                                      │
       │    "context": {...},                       │
       │    "callback_url": "..."                   │
       │  }                                         │
       │───────────────────────────────────────────►│
       │                                            │
       │  202 Accepted                              │
       │  {"task_id": "task-123", "status": "pending"}
       │◄───────────────────────────────────────────│
       │                                            │
       │  GET /a2a/tasks/task-123/stream            │
       │───────────────────────────────────────────►│
       │                                            │
       │  SSE: event: progress                      │
       │       data: {"step": "searching"}          │
       │◄───────────────────────────────────────────│
       │                                            │
       │  SSE: event: result                        │
       │       data: {"answer": "Our policy..."}    │
       │◄───────────────────────────────────────────│
       │                                            │
       │  SSE: event: done                          │
       │       data: {"status": "completed"}        │
       │◄───────────────────────────────────────────│
       │                                            │
```

### 4.4 Task States

```text
PENDING ──► RUNNING ──► COMPLETED
    │           │
    │           └──► FAILED
    │
    └──► CANCELLED
```

---

## 5. MCP Integration (Model Context Protocol)

### 5.1 Tool Server Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                     MCP Tool Servers                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────┐  ┌─────────────────┐  ┌────────────────┐   │
│  │  kagent-tools   │  │  grafana-mcp    │  │  custom-tools  │   │
│  │                 │  │                 │  │                │   │
│  │  • kubectl      │  │  • query        │  │  • your-tool   │   │
│  │  • helm         │  │  • dashboards   │  │  • another     │   │
│  │  • prometheus   │  │  • alerts       │  │                │   │
│  │  • istio        │  │                 │  │                │   │
│  └────────┬────────┘  └────────┬────────┘  └───────┬────────┘   │
│           │                    │                   │             │
│           └────────────────────┼───────────────────┘             │
│                                │                                 │
│                                ▼                                 │
│                    ┌──────────────────────┐                      │
│                    │   KMCP Controller    │                      │
│                    │   (Tool Registry)    │                      │
│                    └──────────────────────┘                      │
│                                │                                 │
│                                ▼                                 │
│                    ┌──────────────────────┐                      │
│                    │      Agents          │                      │
│                    │  (Tool Consumers)    │                      │
│                    └──────────────────────┘                      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 ToolServer CRD

```yaml
apiVersion: kagent.dev/v1alpha2
kind: ToolServer
metadata:
  name: custom-tools
spec:
  type: builtin  # or external
  builtin:
    tools:
      - name: search-database
        description: Search the product database
        inputSchema:
          type: object
          properties:
            query:
              type: string
              description: Search query
          required: [query]
        handler:
          type: http
          url: http://search-api.internal/search
          method: POST
```

---

## 6. Memory & State

### 6.1 Memory Types

| Type | Scope | Duration | Use Case |
|------|-------|----------|----------|
| **None** | - | - | Stateless agents |
| **Conversation** | Session | TTL (e.g., 24h) | Chat context |
| **Persistent** | Agent | Permanent | Long-term memory |
| **Shared** | Multi-agent | Configurable | Team knowledge |

### 6.2 Memory Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                     Memory Architecture                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Short-Term Memory                     │    │
│  │                    (Redis / In-Process)                  │    │
│  │  • Conversation context                                  │    │
│  │  • Tool call results                                     │    │
│  │  • Session state                                         │    │
│  │  TTL: minutes to hours                                   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Long-Term Memory                      │    │
│  │                    (PostgreSQL + pgvector)               │    │
│  │  • Conversation summaries                                │    │
│  │  • User preferences                                      │    │
│  │  • Learned patterns                                      │    │
│  │  TTL: permanent                                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Vector Store                          │    │
│  │                    (pgvector / Qdrant)                   │    │
│  │  • Semantic search                                       │    │
│  │  • Similar conversation retrieval                        │    │
│  │  • Knowledge base embeddings                             │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 7. Health & Readiness

### 7.1 Probe Configuration

```yaml
# Automatic probes for all agents
healthChecks:
  startup:
    path: /health/startup
    initialDelaySeconds: 5
    periodSeconds: 5
    failureThreshold: 30  # Allow 150s for cold start
    
  readiness:
    path: /health/ready
    periodSeconds: 10
    failureThreshold: 3
    
  liveness:
    path: /health/live
    periodSeconds: 30
    failureThreshold: 3
```

### 7.2 Health Response

```json
{
  "status": "healthy",
  "checks": {
    "model": "ok",
    "tools": "ok",
    "memory": "ok"
  },
  "version": "1.2.3",
  "uptime_seconds": 3600
}
```

---

## 8. Implementation Checklist

### Phase 1: Core Lifecycle
- [ ] Agent CRD validation
- [ ] State machine implementation
- [ ] Basic deployment flow
- [ ] Health check integration

### Phase 2: Communication
- [ ] A2A discovery endpoint
- [ ] Task submission/streaming
- [ ] MCP tool integration
- [ ] Agent registry

### Phase 3: Advanced
- [ ] Traffic splitting
- [ ] Memory backends
- [ ] Multi-agent workflows
- [ ] Cross-cluster A2A

---

## 9. References

- [kagent Agent CRD](https://kagent.dev/docs/agent-crd)
- [Knative Revisions](https://knative.dev/docs/serving/revisions/)
- [A2A Protocol Draft](https://github.com/a2a-protocol/spec)
- [MCP Specification](https://modelcontextprotocol.io/)

---

**Previous**: [002-architecture-layers.md](002-architecture-layers.md)  
**Next**: [004-api-design.md](004-api-design.md)
