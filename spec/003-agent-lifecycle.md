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
  
  # ⚠️ Evaluation Configuration (Required for Production)
  evaluation:
    required: true
    dataset:
      ref: datasets/support-agent-v2
      minSamples: 100
    scorers:
      - Safety                    # Built-in: harmful content detection
      - Correctness               # Built-in: factual accuracy
      - RelevanceToQuery          # Built-in: response relevance
      - Guidelines:
          name: brand_voice
          guidelines: "Maintain professional, empathetic tone"
      - Guidelines:
          name: no_pii
          guidelines: "Never expose customer PII in responses"
    minimumScores:
      safety: 1.0                 # 100% pass rate mandatory
      correctness: 0.85           # 85% minimum
      relevance: 0.90             # 90% minimum
    blockOnFailure: true          # Prevent deployment on eval failure
    continuousEvaluation:
      enabled: true
      samplingRate: 0.1           # Evaluate 10% of production traces
      alertThreshold:
        safety: 0.99              # Alert if drops below 99%
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
│  2. BUILD (if needed)                                            │
│  ├── Pull source code                                            │
│  ├── Build container image                                       │
│  ├── Push to registry                                            │
│  └── Security scan                                               │
│                                                                  │
│  3. EVALUATE (MLflow) ⚠️ REQUIRED                                │
│  ├── Load evaluation dataset                                     │
│  ├── Run scorers (Safety, Correctness, Custom)                   │
│  ├── Compare against baseline                                    │
│  └── BLOCK if thresholds not met                                │
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
│  6. MONITOR (Continuous Evaluation)                              │
│  ├── Sample production traces                                    │
│  ├── Run offline evaluation                                      │
│  ├── Alert on quality regression                                 │
│  └── Auto-rollback on safety violations                         │
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
GET /.well-known/agent-card.json
```

```json
{
  "protocolVersion": "1.0",
  "name": "customer-support",
  "description": "Customer Support Agent",
  "version": "1.0.0",
  "supportedInterfaces": [
    {
      "url": "https://customer-support.agentstack.app/a2a/v1",
      "protocolBinding": "HTTP+JSON"
    }
  ],
  "capabilities": {
    "streaming": true,
    "pushNotifications": true
  },
  "skills": [
    {"id": "answer-questions", "name": "Answer Questions"},
    {"id": "create-tickets", "name": "Create Tickets"},
    {"id": "search-kb", "name": "Search Knowledge Base"}
  ],
  "securitySchemes": {
    "bearerAuth": {"type": "http", "scheme": "bearer"},
    "apiKey": {"type": "apiKey", "in": "header", "name": "X-API-Key"}
  }
}
```

> **Note**: This is an abbreviated Agent Card. See [api/017-a2a-protocol.md](api/017-a2a-protocol.md) for the complete schema.

### 4.3 Task Lifecycle

```text
┌─────────────┐                              ┌─────────────┐
│  Requester  │                              │   Provider  │
└──────┬──────┘                              └──────┬──────┘
       │                                            │
       │  POST /a2a/v1/message:send                 │
       │  {                                         │
       │    "message": {                            │
       │      "messageId": "msg-123",               │
       │      "role": "user",                       │
       │      "parts": [{"text": "..."}]            │
       │    },                                      │
       │    "configuration": {...}                  │
       │  }                                         │
       │───────────────────────────────────────────►│
       │                                            │
       │  200 OK                                    │
       │  {"task": {"id": "task-123", "status": ...}}
       │◄───────────────────────────────────────────│
       │                                            │
       │  POST /a2a/v1/message:stream               │
       │───────────────────────────────────────────►│
       │                                            │
       │  SSE: event: statusUpdate                  │
       │       data: {"state": "working"}           │
       │◄───────────────────────────────────────────│
       │                                            │
       │  SSE: event: artifactUpdate                │
       │       data: {"artifact": {...}}            │
       │◄───────────────────────────────────────────│
       │                                            │
       │  SSE: event: statusUpdate                  │
       │       data: {"state": "completed"}         │
       │◄───────────────────────────────────────────│
       │                                            │
```

> **Full Protocol Reference**: See [api/017-a2a-protocol.md](api/017-a2a-protocol.md) for complete endpoint documentation.

### 4.4 Task States

```text
SUBMITTED ──► WORKING ──► COMPLETED
    │            │
    │            ├──► FAILED
    │            │
    │            └──► INPUT-REQUIRED
    │
    └──► CANCELLED
```

| State | Description | Terminal |
|-------|-------------|----------|
| `submitted` | Task created, queued | No |
| `working` | Processing in progress | No |
| `input-required` | Waiting for user input | No |
| `completed` | Successfully finished | Yes |
| `failed` | Error occurred | Yes |
| `cancelled` | User cancelled | Yes |

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

## 8. Evaluation & Quality Gates

> **Safety Critical**: No agent can be deployed without passing evaluation.

### 8.1 Pre-Deployment Evaluation

```text
┌─────────────────────────────────────────────────────────────────┐
│                 Pre-Deployment Evaluation Gate                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Agent Ready for Deploy                                          │
│       │                                                          │
│       ▼                                                          │
│  ┌─────────────────┐                                            │
│  │ Load Dataset    │ ◄── datasets/agent-name-v2                 │
│  └────────┬────────┘                                            │
│           │                                                      │
│           ▼                                                      │
│  ┌─────────────────┐                                            │
│  │ Run Scorers     │ ◄── Safety, Correctness, Custom            │
│  │ (MLflow)        │                                            │
│  └────────┬────────┘                                            │
│           │                                                      │
│     ┌─────┴─────┐                                               │
│     │           │                                               │
│   PASS        FAIL                                               │
│     │           │                                               │
│     ▼           ▼                                               │
│ Continue    Block Deploy                                         │
│ Pipeline    Alert Team                                           │
│             Log Failures                                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 8.2 Evaluation Scorers

| Scorer | Purpose | Required |
|--------|---------|----------|
| **Safety** | Detect harmful/toxic content | ✅ All agents |
| **Correctness** | Validate factual accuracy | ✅ Info agents |
| **RetrievalGroundedness** | Ensure grounded in retrieval | ✅ RAG agents |
| **Guidelines** | Custom policy compliance | Configurable |
| **ToolSafety** | Validate tool usage scope | ✅ Tool agents |

### 8.3 Canary Evaluation

During gradual rollout, evaluate canary traffic in real-time:

```yaml
traffic:
  - revision: rev-003
    percent: 90
    tag: stable
  - revision: rev-002
    percent: 10
    tag: canary
    evaluation:
      enabled: true
      scorers: [Safety, Correctness]
      autoRollback:
        onSafetyViolation: true
        onScoreRegression: 0.05  # 5% degradation triggers rollback
```

### 8.4 Continuous Production Evaluation

```yaml
continuousEvaluation:
  enabled: true
  sampling:
    rate: 0.1               # 10% of traces
    errorRate: 1.0          # 100% of errors
    slowRate: 0.5           # 50% of slow requests
  schedule:
    batch: "*/30 * * * *"   # Every 30 minutes
  alerting:
    safetyThreshold: 0.99   # Alert below 99%
    correctnessThreshold: 0.80
    channels:
      - slack: "#agent-quality"
      - pagerduty: critical  # For safety violations
```

---

## 9. Implementation Checklist

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

### Phase 3: Evaluation & Safety
- [ ] MLflow integration
- [ ] Pre-deployment evaluation gate
- [ ] Canary evaluation
- [ ] Continuous production eval
- [ ] Human feedback collection

### Phase 4: Advanced
- [ ] Traffic splitting
- [ ] Memory backends
- [ ] Multi-agent workflows
- [ ] Cross-cluster A2A

---

## 10. References

- [kagent Agent CRD](https://kagent.dev/docs/agent-crd)
- [Knative Revisions](https://knative.dev/docs/serving/revisions/)
- [A2A Protocol Draft](https://github.com/a2a-protocol/spec)
- [MCP Specification](https://modelcontextprotocol.io/)
- [MLflow GenAI Evaluation](https://mlflow.org/docs/latest/genai/eval-monitor/)
- [MLflow Agent Evaluation](https://mlflow.org/docs/latest/genai/eval-monitor/running-evaluation/agents/)

---

**Previous**: [002-architecture-layers.md](002-architecture-layers.md)  
**Next**: [004-api-design.md](004-api-design.md)
