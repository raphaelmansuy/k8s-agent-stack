# 015 - Admin Endpoints

> Teams, Projects, Secrets, Domains, Webhooks, API Keys, Analytics

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Teams

### 1.1 List Teams

```yaml
GET /v1/teams

Response: 200 OK
{
  "data": [
    {
      "id": "team_6pQr7sStuV",
      "name": "Acme Corp",
      "slug": "acme-corp",
      "avatar_url": "https://...",
      "member_count": 12,
      "project_count": 5,
      "billing": {
        "plan": "pro",
        "status": "active"
      },
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

### 1.2 Create Team

```yaml
POST /v1/teams

{
  "name": "Acme Corp",
  "slug": "acme-corp"
}

Response: 201 Created
{
  "id": "team_6pQr7sStuV",
  "name": "Acme Corp",
  ...
}
```

### 1.3 Get Team

```yaml
GET /v1/teams/{teamId}

Response: 200 OK
{ ... }
```

### 1.4 Update Team

```yaml
PATCH /v1/teams/{teamId}

{
  "name": "Acme Corporation",
  "avatar_url": "https://..."
}

Response: 200 OK
```

### 1.5 Team Members

```yaml
# List members
GET /v1/teams/{teamId}/members

Response: 200 OK
{
  "data": [
    {
      "user_id": "usr_abc",
      "email": "admin@acme.com",
      "name": "Admin User",
      "role": "owner",
      "joined_at": "2025-01-01T00:00:00Z"
    }
  ]
}

# Add member
POST /v1/teams/{teamId}/members
{
  "email": "developer@acme.com",
  "role": "member"
}

# Update role
PATCH /v1/teams/{teamId}/members/{userId}
{
  "role": "admin"
}

# Remove member
DELETE /v1/teams/{teamId}/members/{userId}
```

---

## 2. Projects

### 2.1 List Projects

```yaml
GET /v1/projects

Response: 200 OK
{
  "data": [
    {
      "id": "prj_3kLm4nOpQr",
      "name": "Production",
      "slug": "production",
      "team_id": "team_xxx",
      "settings": {
        "default_model": "gpt-4o",
        "log_retention": "30d"
      },
      "stats": {
        "agent_count": 5,
        "request_count_30d": 125000
      },
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

### 2.2 Create Project

```yaml
POST /v1/projects

{
  "name": "Production",
  "description": "Production agents",
  "team_id": "team_xxx"
}

Response: 201 Created
{
  "id": "prj_3kLm4nOpQr",
  ...
}
```

### 2.3 Update Project

```yaml
PATCH /v1/projects/{projectId}

{
  "name": "Production v2",
  "settings": {
    "default_model": "gpt-4o-mini",
    "log_retention": "90d"
  }
}

Response: 200 OK
```

### 2.4 Delete Project

```yaml
DELETE /v1/projects/{projectId}

Response: 202 Accepted
{
  "message": "Project deletion initiated. All agents and data will be removed."
}
```

---

## 3. Secrets

### 3.1 List Secrets

```yaml
GET /v1/secrets
X-Project-ID: prj_abc123

Response: 200 OK
{
  "data": [
    {
      "name": "OPENAI_API_KEY",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-15T10:00:00Z",
      "used_by": ["agt_xxx", "agt_yyy"]
    }
  ]
}
```

### 3.2 Create Secret

```yaml
POST /v1/secrets
X-Project-ID: prj_abc123

{
  "name": "OPENAI_API_KEY",
  "value": "sk-..."
}

Response: 201 Created
{
  "name": "OPENAI_API_KEY",
  "created_at": "2025-01-15T10:00:00Z"
}
```

### 3.3 Update Secret

```yaml
PUT /v1/secrets/{secretName}
X-Project-ID: prj_abc123

{
  "value": "sk-new-key..."
}

Response: 200 OK
{
  "name": "OPENAI_API_KEY",
  "updated_at": "2025-01-15T10:00:00Z"
}
```

### 3.4 Delete Secret

```yaml
DELETE /v1/secrets/{secretName}

Response: 204 No Content

# If in use
Response: 409 Conflict
{
  "type": "https://api.agentstack.io/errors/conflict",
  "title": "Secret In Use",
  "detail": "Secret is referenced by agents: agt_xxx, agt_yyy"
}
```

---

## 4. Domains

### 4.1 List Domains

```yaml
GET /v1/domains
X-Project-ID: prj_abc123

Response: 200 OK
{
  "data": [
    {
      "id": "dom_8rSt9uVwXy",
      "domain": "agents.example.com",
      "status": "active",
      "verification": {
        "type": "cname",
        "name": "_agentstack.agents.example.com",
        "value": "verify.agentstack.io"
      },
      "ssl": {
        "status": "active",
        "expires_at": "2026-01-15T00:00:00Z"
      },
      "agent_id": "agt_xxx",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

### 4.2 Add Domain

```yaml
POST /v1/domains
X-Project-ID: prj_abc123

{
  "domain": "agents.example.com",
  "agent_id": "agt_xxx"
}

Response: 201 Created
{
  "id": "dom_8rSt9uVwXy",
  "domain": "agents.example.com",
  "status": "pending",
  "verification": {
    "type": "cname",
    "name": "_agentstack.agents.example.com",
    "value": "verify.agentstack.io"
  },
  "instructions": "Add a CNAME record to verify domain ownership."
}
```

### 4.3 Verify Domain

```yaml
POST /v1/domains/{domainId}/verify

Response: 200 OK
{
  "verified": true,
  "message": "Domain verified. SSL certificate provisioning..."
}

# Not yet verified
{
  "verified": false,
  "message": "DNS record not found. Please add the CNAME record."
}
```

### 4.4 Delete Domain

```yaml
DELETE /v1/domains/{domainId}

Response: 204 No Content
```

---

## 5. Webhooks

### 5.1 List Webhooks

```yaml
GET /v1/webhooks
X-Project-ID: prj_abc123

Response: 200 OK
{
  "data": [
    {
      "id": "whk_9sTu0vWxYz",
      "url": "https://example.com/webhook",
      "events": ["deployment.succeeded", "deployment.failed"],
      "status": "active",
      "failure_count": 0,
      "last_delivery": {
        "status": "delivered",
        "timestamp": "2025-01-15T10:30:00Z"
      },
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

### 5.2 Create Webhook

```yaml
POST /v1/webhooks
X-Project-ID: prj_abc123

{
  "url": "https://example.com/webhook",
  "events": [
    "agent.created",
    "agent.deleted",
    "deployment.started",
    "deployment.succeeded",
    "deployment.failed"
  ]
}

Response: 201 Created
{
  "id": "whk_9sTu0vWxYz",
  "url": "https://example.com/webhook",
  "events": [...],
  "secret": "whsec_abc123xyz...",  # Only shown on create
  "created_at": "2025-01-15T10:00:00Z"
}
```

### 5.3 Webhook Events

| Event | Description |
|-------|-------------|
| `agent.created` | New agent created |
| `agent.updated` | Agent configuration changed |
| `agent.deleted` | Agent deleted |
| `deployment.started` | Deployment began |
| `deployment.succeeded` | Deployment completed |
| `deployment.failed` | Deployment failed |
| `chat.message` | User message received |
| `chat.response` | Agent response sent |
| `alert.triggered` | Alert condition met |

### 5.4 Webhook Payload

```json
{
  "id": "evt_abc123",
  "type": "deployment.succeeded",
  "created_at": "2025-01-15T10:30:00Z",
  "project_id": "prj_abc123",
  "data": {
    "agent_id": "agt_xyz",
    "deployment_id": "dpl_123",
    "revision_id": "rev_456",
    "url": "https://agent.agentstack.app"
  }
}
```

### 5.5 Webhook Signature

```http
POST /webhook HTTP/1.1
X-AgentStack-Signature: sha256=abc123...
X-AgentStack-Timestamp: 1705312800
```

```python
# Verify signature
import hmac
import hashlib

def verify_signature(payload, signature, secret, timestamp):
    expected = hmac.new(
        secret.encode(),
        f"{timestamp}.{payload}".encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature)
```

### 5.6 List Deliveries

```yaml
GET /v1/webhooks/{webhookId}/deliveries

Response: 200 OK
{
  "data": [
    {
      "id": "del_abc",
      "event_id": "evt_xyz",
      "event_type": "deployment.succeeded",
      "status": "delivered",
      "response": {
        "status_code": 200,
        "duration_ms": 150
      },
      "attempts": 1,
      "created_at": "2025-01-15T10:30:00Z"
    }
  ]
}
```

### 5.7 Test Webhook

```yaml
POST /v1/webhooks/{webhookId}/test

{
  "event_type": "deployment.succeeded"
}

Response: 200 OK
{
  "delivery_id": "del_test_abc",
  "status": "delivered",
  "response": {
    "status_code": 200,
    "duration_ms": 120
  }
}
```

---

## 6. API Keys

### 6.1 List API Keys

```yaml
GET /v1/api-keys
X-Project-ID: prj_abc123

Response: 200 OK
{
  "data": [
    {
      "id": "key_1gTs8vXxZa",
      "name": "Production Backend",
      "prefix": "as_prj_sk_live_abc1",
      "scope": "project",
      "permissions": ["agents:read", "agents:chat"],
      "last_used_at": "2025-01-15T10:30:00Z",
      "expires_at": null,
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

### 6.2 Create API Key

```yaml
POST /v1/api-keys
X-Project-ID: prj_abc123

{
  "name": "Production Backend",
  "scope": "project",
  "permissions": ["agents:read", "agents:chat"],
  "expires_at": "2026-01-01T00:00:00Z"
}

Response: 201 Created
{
  "id": "key_1gTs8vXxZa",
  "key": "as_prj_sk_live_abc123xyz...",  # Only shown ONCE
  "name": "Production Backend",
  ...
}
```

### 6.3 Revoke API Key

```yaml
DELETE /v1/api-keys/{keyId}

Response: 204 No Content
```

---

## 7. Logs

### 7.1 Get Agent Logs

```yaml
GET /v1/agents/{agentId}/logs
  ?start=2025-01-15T10:00:00Z
  &end=2025-01-15T11:00:00Z
  &level=info
  &search=error
  &limit=100

Response: 200 OK
{
  "data": [
    {
      "timestamp": "2025-01-15T10:30:00.123Z",
      "level": "info",
      "message": "Request completed",
      "request_id": "req_abc",
      "session_id": "ses_xyz",
      "metadata": {
        "duration_ms": 1250,
        "tokens": 150
      }
    }
  ],
  "pagination": { ... }
}
```

### 7.2 Stream Logs

```yaml
GET /v1/agents/{agentId}/logs/stream
  ?level=info
  &since=2025-01-15T10:00:00Z
Accept: text/event-stream

Response: 200 OK
Content-Type: text/event-stream

event: log
data: {"timestamp": "2025-01-15T10:30:00Z", "level": "info", "message": "..."}

event: log
data: {"timestamp": "2025-01-15T10:30:01Z", "level": "warn", "message": "..."}
```

---

## 8. Analytics

### 8.1 Usage Metrics

```yaml
GET /v1/analytics/usage
  ?start=2025-01-01T00:00:00Z
  &end=2025-01-31T23:59:59Z
  &granularity=day
  &agent_id=agt_xxx  # Optional filter
X-Project-ID: prj_abc123

Response: 200 OK
{
  "period": {
    "start": "2025-01-01T00:00:00Z",
    "end": "2025-01-31T23:59:59Z"
  },
  "totals": {
    "requests": 125000,
    "tokens_input": 15000000,
    "tokens_output": 8000000,
    "tool_calls": 45000,
    "errors": 250,
    "cost_usd": 456.78
  },
  "series": [
    {
      "timestamp": "2025-01-01T00:00:00Z",
      "requests": 4000,
      "tokens": 750000,
      "errors": 8
    },
    ...
  ]
}
```

### 8.2 Agent Analytics

```yaml
GET /v1/analytics/agents/{agentId}
  ?start=2025-01-01T00:00:00Z
  &end=2025-01-31T23:59:59Z

Response: 200 OK
{
  "agent_id": "agt_xxx",
  "period": { ... },
  "summary": {
    "total_requests": 50000,
    "unique_users": 1200,
    "avg_response_time_ms": 1250,
    "p95_response_time_ms": 2800,
    "error_rate": 0.002,
    "satisfaction_score": 4.2
  },
  "top_tools": [
    {"tool": "search-kb", "count": 15000},
    {"tool": "create-ticket", "count": 8000}
  ],
  "top_errors": [
    {"error": "context_length", "count": 50},
    {"error": "tool_timeout", "count": 30}
  ]
}
```

---

## 9. Health & Status

### 9.1 Health Check

```yaml
GET /v1/health

# No authentication required
Response: 200 OK
{
  "status": "healthy",
  "version": "1.0.0"
}
```

### 9.2 System Status

```yaml
GET /v1/status

Response: 200 OK
{
  "status": "operational",
  "components": {
    "api": "operational",
    "deployments": "operational",
    "chat": "operational",
    "logging": "operational"
  }
}
```

---

## 10. Multi-Tenant Considerations

| Endpoint | Tenant Scope |
|----------|--------------|
| Teams | User-accessible teams |
| Projects | Team membership |
| Secrets | Project |
| Domains | Project |
| Webhooks | Project |
| API Keys | Project |
| Logs | Project |
| Analytics | Project |

---

**Previous**: [014-tools-models.md](014-tools-models.md)  
**Next**: [016-schemas.md](016-schemas.md)
