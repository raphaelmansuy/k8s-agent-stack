# 004 - API Design

> REST API Specification for AgentStack Platform

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

> **Related**: See [api/](api/) for modular API docs, [api/017-a2a-protocol.md](api/017-a2a-protocol.md) for A2A, [api/018-agui-protocol.md](api/018-agui-protocol.md) for AG-UI streaming

---

## 1. Design Principles

| Principle | Implementation |
|-----------|----------------|
| **RESTful** | Resource-oriented, HTTP verbs, stateless |
| **Versioned** | URL path versioning (`/v1/`) |
| **Consistent** | RFC 7807 errors, cursor pagination |
| **Streaming** | SSE for real-time updates |
| **Idempotent** | Idempotency-Key header support |

---

## 2. Authentication

### Methods

```text
┌─────────────────────────────────────────────────────────────────┐
│                     Authentication Methods                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Bearer Token (JWT)                                          │
│     Authorization: Bearer eyJhbG...                             │
│     → User sessions, UI, OAuth flows                            │
│                                                                 │
│  2. API Key                                                     │
│     X-API-Key: ask_1234567890abcdef                             │
│     → Server-to-server, CI/CD, automation                       │
│                                                                 │
│  3. Service Account (internal)                                  │
│     X-Service-Token: svc_...                                    │
│     → Internal services, cluster-local                          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### API Key Format

```text
Prefix:  ask_           (agentstack key)
Length:  32 characters
Example: ask_1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p
```

---

## 3. API Structure

### Base URL

```
Production:  https://api.agentstack.io/v1
Staging:     https://api.staging.agentstack.io/v1
Local:       http://localhost:8080/v1
```

### Resource Hierarchy

```text
/v1
├── /agents
│   ├── GET           List agents
│   ├── POST          Create agent
│   └── /{agentId}
│       ├── GET       Get agent
│       ├── PATCH     Update agent
│       ├── DELETE    Delete agent
│       ├── /deployments
│       │   ├── GET   List deployments
│       │   ├── POST  Create deployment
│       │   └── /{deploymentId}
│       │       ├── GET              Get deployment
│       │       ├── GET /events      Stream events (SSE)
│       │       └── POST /cancel     Cancel deployment
│       ├── /revisions
│       ├── /traffic
│       ├── /chat
│       │   ├── POST          Sync chat
│       │   └── /stream       SSE chat
│       ├── /sessions
│       └── /logs
├── /tools
├── /models
├── /secrets
├── /projects
├── /teams
├── /domains
├── /webhooks
├── /api-keys
├── /analytics
└── /health
```

---

## 4. Core Endpoints

### 4.1 Agents

```yaml
# Create Agent
POST /v1/agents
Content-Type: application/json
X-Project-ID: prj_abc123

{
  "name": "customer-support",
  "framework": "google-adk",
  "config": {
    "model": "gpt-4o",
    "system_prompt": "You are a helpful assistant.",
    "tools": ["search-kb", "create-ticket"]
  },
  "auto_deploy": true
}

# Response: 201 Created
{
  "id": "agt_2xKj9mNpQr",
  "name": "customer-support",
  "status": "deploying",
  "urls": {
    "chat": "https://customer-support.agentstack.app"
  },
  "created_at": "2025-01-15T10:30:00Z"
}
```

### 4.2 Chat

```yaml
# Streaming Chat
POST /v1/agents/{agentId}/chat/stream
Content-Type: application/json
Accept: text/event-stream

{
  "message": "What's the status of order #12345?",
  "session_id": "ses_abc123"
}

# Response: SSE Stream
event: message_start
data: {"message_id": "msg_xyz", "role": "assistant"}

event: content_delta
data: {"delta": "Let me check that order for you..."}

event: tool_start
data: {"tool": "lookup-order", "input": {"order_id": "12345"}}

event: tool_result
data: {"tool": "lookup-order", "output": {"status": "shipped"}}

event: content_delta
data: {"delta": "Your order #12345 has been shipped!"}

event: message_end
data: {"usage": {"input_tokens": 15, "output_tokens": 42}}
```

### 4.3 Deployments

```yaml
# Create Deployment
POST /v1/agents/{agentId}/deployments
Idempotency-Key: deploy-abc-123

{
  "source": {
    "type": "git",
    "ref": "main"
  }
}

# Response: 202 Accepted
{
  "id": "dpl_8vNm3kLpWx",
  "status": "queued",
  "created_at": "2025-01-15T10:30:00Z"
}

# Stream deployment events
GET /v1/agents/{agentId}/deployments/{deploymentId}/events
Accept: text/event-stream

event: build_start
data: {"step": "dependencies"}

event: build_progress  
data: {"step": "dependencies", "progress": 45}

event: deploy_start
data: {"revision": "rev_abc123"}

event: ready
data: {"url": "https://my-agent.agentstack.app"}
```

---

## 5. Response Formats

### Success Response

```json
{
  "id": "agt_2xKj9mNpQr",
  "name": "my-agent",
  "status": "active",
  "created_at": "2025-01-15T10:30:00Z"
}
```

### List Response (Cursor Pagination)

```json
{
  "data": [
    {"id": "agt_1", "name": "agent-1"},
    {"id": "agt_2", "name": "agent-2"}
  ],
  "pagination": {
    "next_cursor": "eyJpZCI6ImFndF8yIn0",
    "has_more": true
  }
}
```

### Error Response (RFC 7807)

```json
{
  "type": "https://agentstack.io/errors/validation",
  "title": "Validation Error",
  "status": 400,
  "detail": "Request body contains invalid fields",
  "instance": "/v1/agents",
  "trace_id": "abc123xyz",
  "errors": [
    {"field": "name", "message": "must be 3-64 characters"},
    {"field": "config.model", "message": "unknown model"}
  ]
}
```

---

## 6. Rate Limiting

### Limits by Plan

| Plan | Requests/min | Burst | Tokens/day |
|------|--------------|-------|------------|
| Free | 60 | 100 | 100K |
| Pro | 600 | 1000 | 10M |
| Enterprise | 6000 | 10000 | Unlimited |

### Response Headers

```http
X-RateLimit-Limit: 600
X-RateLimit-Remaining: 599
X-RateLimit-Reset: 1705312800
Retry-After: 45  # Only on 429
```

---

## 7. Common Patterns

### 7.1 Idempotency

```http
POST /v1/agents
Idempotency-Key: create-agent-abc-123
Content-Type: application/json

{"name": "my-agent", ...}
```

Same key within 24h returns cached response.

### 7.2 Filtering & Search

```http
GET /v1/agents?status=active&tags=production,critical&search=support
```

### 7.3 Sorting

```http
GET /v1/agents?sort=-created_at  # Descending
GET /v1/agents?sort=name         # Ascending
```

### 7.4 Field Selection

```http
GET /v1/agents?fields=id,name,status
```

---

## 8. Webhook Events

### Event Types

| Event | Description |
|-------|-------------|
| `agent.created` | Agent created |
| `agent.deleted` | Agent deleted |
| `deployment.started` | Deployment began |
| `deployment.succeeded` | Deployment completed |
| `deployment.failed` | Deployment failed |
| `chat.message` | Message sent |
| `alert.triggered` | Alert fired |

### Payload Format

```json
{
  "id": "evt_abc123",
  "type": "deployment.succeeded",
  "created_at": "2025-01-15T10:30:00Z",
  "data": {
    "agent_id": "agt_xyz",
    "deployment_id": "dpl_123",
    "url": "https://agent.agentstack.app"
  }
}
```

### Signature Verification

```http
X-Webhook-Signature: sha256=abc123...
X-Webhook-Timestamp: 1705312800
```

```python
import hmac
expected = hmac.new(
    secret.encode(),
    f"{timestamp}.{body}".encode(),
    "sha256"
).hexdigest()
assert hmac.compare_digest(expected, signature)
```

---

## 9. SDK Examples

### Python

```python
from agentstack import AgentStack

client = AgentStack(api_key="ask_...")

# Create agent
agent = client.agents.create(
    name="support",
    framework="google-adk",
    config={"model": "gpt-4o"}
)

# Chat with streaming
for event in client.chat.stream(agent.id, "Hello!"):
    if event.type == "content_delta":
        print(event.delta, end="")
```

### TypeScript

```typescript
import { AgentStack } from '@agentstack/sdk';

const client = new AgentStack({ apiKey: 'ask_...' });

// Create agent
const agent = await client.agents.create({
  name: 'support',
  framework: 'google-adk',
  config: { model: 'gpt-4o' }
});

// Chat with streaming
const stream = await client.chat.stream(agent.id, 'Hello!');
for await (const event of stream) {
  if (event.type === 'content_delta') {
    process.stdout.write(event.delta);
  }
}
```

### Go

```go
import "github.com/agentstack/sdk-go"

client := agentstack.New("ask_...")

// Create agent
agent, _ := client.Agents.Create(ctx, &agentstack.AgentCreate{
    Name:      "support",
    Framework: "google-adk",
    Config:    &agentstack.Config{Model: "gpt-4o"},
})

// Chat with streaming
stream, _ := client.Chat.Stream(ctx, agent.ID, "Hello!")
for event := range stream.Events() {
    if event.Type == "content_delta" {
        fmt.Print(event.Delta)
    }
}
```

---

## 10. OpenAPI Specification

The OpenAPI 3.1 specification is **auto-generated** by [Huma](https://huma.rocks/) from Go handler definitions. This ensures documentation never drifts from implementation.

**Access the spec at runtime:**
```text
Development:  http://localhost:8080/openapi.json
Production:   https://api.agentstack.io/openapi.json
```

**Generate SDK clients:**

```bash
# Fetch the spec from a running server
curl -o openapi.yaml http://localhost:8080/openapi.yaml

# Go
openapi-generator generate -i openapi.yaml -g go -o sdk/go

# Python
openapi-generator generate -i openapi.yaml -g python -o sdk/python

# TypeScript
openapi-generator generate -i openapi.yaml -g typescript-fetch -o sdk/ts
```

> **See Also**: [tech_stack/001-api-tech-stack.md](tech_stack/001-api-tech-stack.md) for Huma implementation details.

---

## 11. Implementation Notes

### Go Handler Structure

```text
internal/api/
├── handler/          # HTTP handlers
│   ├── agents.go
│   ├── chat.go
│   └── deployments.go
├── middleware/       # Auth, rate limit, logging
├── router/           # Route definitions
└── response/         # RFC 7807 helpers
```

### Key Libraries

| Library | Purpose |
|---------|---------|
| `huma` | API framework with OpenAPI generation |
| `pgx` | PostgreSQL driver |
| `sqlc` | Type-safe SQL codegen |
| `zap` | Structured logging |
| `otel` | OpenTelemetry |

> **Note**: See [tech_stack/README.md](../tech_stack/README.md) for detailed stack documentation.

---

**Previous**: [003-agent-lifecycle.md](003-agent-lifecycle.md)  
**Next**: [005-data-architecture.md](005-data-architecture.md)
