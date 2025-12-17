# 012 - Agent Endpoints

> Agent CRUD, Deployments, Revisions, and Traffic Management

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Resource Overview

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Agent Resource Hierarchy                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Agent (agt_xxx)                                                │
│  │   • Configuration definition                                 │
│  │   • Current deployment reference                            │
│  │                                                               │
│  ├── Deployment (dpl_xxx)                                       │
│  │   │   • Build + deploy operation                            │
│  │   │   • Status tracking                                     │
│  │   │                                                          │
│  │   └── Revision (rev_xxx)                                    │
│  │       • Immutable snapshot                                   │
│  │       • Addressable version                                  │
│  │                                                               │
│  └── Traffic                                                    │
│      • Route distribution                                       │
│      • Canary configuration                                     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Agent CRUD

### 2.1 List Agents

```yaml
GET /v1/agents
X-Project-ID: prj_abc123

Query Parameters:
  status:   active | inactive | deploying | failed | suspended
  tags:     tag1,tag2  (filter by tags)
  search:   string     (search name/description)
  sort:     created_at | -created_at | name | -name
  limit:    1-100      (default: 20)
  cursor:   string     (pagination)

Response: 200 OK
{
  "data": [
    {
      "id": "agt_2xKj9mNpQr",
      "name": "customer-support",
      "slug": "customer-support",
      "description": "Handles customer inquiries",
      "status": "active",
      "framework": "google-adk",
      "current_deployment": {
        "id": "dpl_8vNm3kLpWx",
        "revision": "rev_4jKm2nPqRs",
        "status": "ready",
        "url": "https://customer-support-acme.agentstack.app"
      },
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-20T14:22:00Z"
    }
  ],
  "pagination": {
    "next_cursor": "eyJpZCI6ImFndF8...",
    "has_more": true
  }
}
```

### 2.2 Create Agent

```yaml
POST /v1/agents
X-Project-ID: prj_abc123
Idempotency-Key: create-agent-12345

# Minimal request
{
  "name": "my-agent",
  "framework": "google-adk"
}

# Full request
{
  "name": "customer-support",
  "description": "Handles customer inquiries with empathy",
  "framework": "google-adk",
  "source": {
    "type": "git",
    "repo": "github.com/acme/support-agent",
    "branch": "main",
    "path": "/"
  },
  "config": {
    "model": "gpt-4o",
    "system_prompt": "You are a helpful customer support agent.",
    "tools": ["search-kb", "create-ticket"],
    "memory": {
      "type": "conversation",
      "ttl": "24h"
    },
    "scaling": {
      "min_instances": 1,
      "max_instances": 10,
      "target_concurrency": 10
    },
    "timeout": "5m",
    "env": {
      "LOG_LEVEL": "info",
      "API_URL": "{{secrets.BACKEND_URL}}"
    }
  },
  "tags": ["production", "customer-facing"],
  "auto_deploy": true
}

Response: 201 Created
Location: /v1/agents/agt_2xKj9mNpQr
{
  "id": "agt_2xKj9mNpQr",
  "name": "customer-support",
  "status": "deploying",
  ...
}
```

### 2.3 Get Agent

```yaml
GET /v1/agents/{agentId}

Response: 200 OK
{
  "id": "agt_2xKj9mNpQr",
  "name": "customer-support",
  "slug": "customer-support",
  "description": "Handles customer inquiries",
  "status": "active",
  "framework": "google-adk",
  "source": { ... },
  "config": { ... },
  "current_deployment": { ... },
  "urls": {
    "chat": "https://customer-support-acme.agentstack.app/chat",
    "api": "https://api.agentstack.io/v1/agents/agt_2xKj9mNpQr"
  },
  "tags": ["production"],
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-20T14:22:00Z"
}
```

### 2.4 Update Agent

```yaml
PATCH /v1/agents/{agentId}

{
  "description": "Updated description",
  "config": {
    "model": "gpt-4o-mini",
    "system_prompt": "New prompt..."
  },
  "auto_deploy": true  # Deploy changes immediately
}

Response: 200 OK
{ ... updated agent ... }
```

### 2.5 Delete Agent

```yaml
DELETE /v1/agents/{agentId}?force=false

Response: 202 Accepted
{
  "id": "agt_2xKj9mNpQr",
  "status": "deleting",
  "message": "Agent deletion initiated. Will complete within 30 seconds."
}
```

---

## 3. Deployments

### 3.1 List Deployments

```yaml
GET /v1/agents/{agentId}/deployments
  ?status=ready|building|failed
  &limit=20
  &cursor=xxx

Response: 200 OK
{
  "data": [
    {
      "id": "dpl_8vNm3kLpWx",
      "agent_id": "agt_2xKj9mNpQr",
      "revision_id": "rev_4jKm2nPqRs",
      "status": "ready",
      "url": "https://customer-support-acme.agentstack.app",
      "meta": {
        "git_sha": "a1b2c3d",
        "git_message": "Fix bug",
        "git_author": "developer@example.com"
      },
      "timing": {
        "queued_at": "2025-01-15T10:30:00Z",
        "build_started_at": "2025-01-15T10:30:05Z",
        "build_finished_at": "2025-01-15T10:31:00Z",
        "deployed_at": "2025-01-15T10:31:30Z",
        "duration_ms": 90000
      },
      "created_at": "2025-01-15T10:30:00Z"
    }
  ]
}
```

### 3.2 Create Deployment

```yaml
POST /v1/agents/{agentId}/deployments
Idempotency-Key: deploy-12345

# Deploy from Git
{
  "source": {
    "type": "git",
    "ref": "main"
  }
}

# Deploy from Image
{
  "source": {
    "type": "image",
    "image": "gcr.io/my-project/agent:v1.2.3"
  }
}

# Deploy with config overrides
{
  "source": { "type": "git", "ref": "feature-branch" },
  "config_overrides": {
    "scaling": { "max_instances": 5 }
  },
  "alias": "canary"
}

Response: 202 Accepted
Location: /v1/agents/agt_xxx/deployments/dpl_yyy
{
  "id": "dpl_8vNm3kLpWx",
  "status": "queued",
  "created_at": "2025-01-15T10:30:00Z"
}
```

### 3.3 Get Deployment

```yaml
GET /v1/agents/{agentId}/deployments/{deploymentId}

Response: 200 OK
{
  "id": "dpl_8vNm3kLpWx",
  "status": "building",
  "build_logs_url": "/v1/agents/agt_xxx/deployments/dpl_yyy/logs",
  ...
}
```

### 3.4 Stream Deployment Events

```yaml
GET /v1/agents/{agentId}/deployments/{deploymentId}/events
Accept: text/event-stream

Response: 200 OK
Content-Type: text/event-stream

event: build_start
data: {"step": "dependencies", "message": "Installing dependencies..."}

event: build_progress
data: {"step": "dependencies", "progress": 45}

event: build_complete
data: {"step": "dependencies", "duration_ms": 12340}

event: deploy_start
data: {"revision": "rev_abc123"}

event: ready
data: {"url": "https://agent.agentstack.app", "duration_ms": 45000}
```

### 3.5 Cancel Deployment

```yaml
POST /v1/agents/{agentId}/deployments/{deploymentId}/cancel

Response: 200 OK
{
  "id": "dpl_8vNm3kLpWx",
  "status": "cancelled"
}
```

---

## 4. Revisions

### 4.1 List Revisions

```yaml
GET /v1/agents/{agentId}/revisions
  ?limit=20
  &cursor=xxx

Response: 200 OK
{
  "data": [
    {
      "id": "rev_4jKm2nPqRs",
      "agent_id": "agt_2xKj9mNpQr",
      "deployment_id": "dpl_8vNm3kLpWx",
      "status": "active",
      "traffic_percent": 100,
      "url": "https://rev-4jKm2nPqRs.customer-support.agentstack.app",
      "meta": {
        "git_sha": "a1b2c3d",
        "image_digest": "sha256:..."
      },
      "created_at": "2025-01-15T10:31:30Z"
    }
  ]
}
```

### 4.2 Get Revision

```yaml
GET /v1/agents/{agentId}/revisions/{revisionId}

Response: 200 OK
{
  "id": "rev_4jKm2nPqRs",
  "config": {
    "model": "gpt-4o",
    "system_prompt": "...",
    ...
  },
  ...
}
```

---

## 5. Traffic Management

### 5.1 Get Traffic Configuration

```yaml
GET /v1/agents/{agentId}/traffic

Response: 200 OK
{
  "routes": [
    {
      "revision": "rev_current",
      "percent": 100,
      "tag": null
    }
  ]
}
```

### 5.2 Update Traffic Split

```yaml
PUT /v1/agents/{agentId}/traffic

# Canary deployment (10% to new revision)
{
  "routes": [
    { "revision": "rev_current", "percent": 90 },
    { "revision": "rev_canary", "percent": 10, "tag": "canary" }
  ]
}

# Blue-green switch
{
  "routes": [
    { "revision": "rev_new", "percent": 100 }
  ]
}

Response: 200 OK
{
  "routes": [ ... ]
}
```

### 5.3 Rollback

```yaml
# Quick rollback to previous revision
POST /v1/agents/{agentId}/rollback

{
  "revision": "rev_previous"  # or "previous" for auto-detect
}

Response: 200 OK
{
  "routes": [
    { "revision": "rev_previous", "percent": 100 }
  ]
}
```

---

## 6. Agent Configuration Schema

```yaml
AgentConfig:
  model: string                    # Default LLM model
  system_prompt: string            # System instruction (max 32KB)
  tools: string[]                  # Tool IDs or names
  memory:
    type: none | conversation | persistent
    ttl: duration                  # e.g., "24h", "7d"
  scaling:
    min_instances: 0-100           # Scale-to-zero capable
    max_instances: 1-1000
    target_concurrency: 1-1000     # Requests per instance
  timeout: duration                # Max request time
  env: map<string, string>         # Environment variables

AgentSource:
  type: git | image | inline
  # For git:
  repo: string                     # Repository URL
  branch: string                   # Branch name
  path: string                     # Path to agent code
  # For image:
  image: string                    # Container image
  # For inline:
  runtime: python3.11 | python3.12 | node20
  entrypoint: string               # Entry file
  files: map<string, string>       # File contents
```

---

## 7. Status Values

| Status | Description |
|--------|-------------|
| `active` | Agent running, receiving traffic |
| `inactive` | Agent stopped, no instances |
| `deploying` | Deployment in progress |
| `failed` | Last deployment failed |
| `suspended` | Paused by admin |
| `deleting` | Deletion in progress |

| Deployment Status | Description |
|-------------------|-------------|
| `queued` | Waiting to build |
| `building` | Build in progress |
| `deploying` | Deploying to cluster |
| `ready` | Successfully deployed |
| `failed` | Deployment failed |
| `cancelled` | Cancelled by user |

---

## 8. Error Responses

```yaml
# Agent not found
404 Not Found
{
  "type": "https://api.agentstack.io/errors/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "Agent agt_notexist was not found"
}

# Deployment conflict
409 Conflict
{
  "type": "https://api.agentstack.io/errors/conflict",
  "title": "Conflict",
  "status": 409,
  "detail": "A deployment is already in progress for this agent"
}

# Validation error
400 Bad Request
{
  "type": "https://api.agentstack.io/errors/validation",
  "title": "Validation Error",
  "status": 400,
  "errors": [
    {"field": "name", "message": "must be 3-64 characters"},
    {"field": "config.model", "message": "unknown model 'gpt-5'"}
  ]
}
```

---

## 9. Multi-Tenant Considerations

| Aspect | Implementation |
|--------|----------------|
| **Scoping** | All agents scoped to project |
| **Naming** | Unique within project |
| **URLs** | Include project slug |
| **Quotas** | Max agents per project |
| **Isolation** | Namespace per project (Pro+) |

---

**Previous**: [011-authentication.md](011-authentication.md)  
**Next**: [013-chat-sessions.md](013-chat-sessions.md)
