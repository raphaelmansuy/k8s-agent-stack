# 016 - Schemas Reference

> Reusable schemas, error formats, and common patterns

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. ID Format Conventions

All resource IDs follow a typed prefix pattern:

```yaml
ID Format: {prefix}_{random}
Random: 10-character alphanumeric (lowercase + digits)
Example: agt_3kLm4nOpQr
```

| Prefix | Resource |
|--------|----------|
| `agt_` | Agent |
| `dpl_` | Deployment |
| `rev_` | Revision |
| `ses_` | Session |
| `tol_` | Tool |
| `mcp_` | MCP Server |
| `prj_` | Project |
| `team_` | Team |
| `key_` | API Key |
| `dom_` | Domain |
| `whk_` | Webhook |
| `evt_` | Event |
| `del_` | Webhook Delivery |

---

## 2. Error Responses (RFC 7807)

### 2.1 Problem Details

```yaml
ProblemDetails:
  type: object
  required: [type, title, status]
  properties:
    type:
      type: string
      format: uri
      description: URI reference identifying the problem type
      example: "https://agentstack.dev/errors/validation"
    title:
      type: string
      description: Short, human-readable summary
      example: "Validation Error"
    status:
      type: integer
      description: HTTP status code
      example: 400
    detail:
      type: string
      description: Human-readable explanation
      example: "The request body contains invalid fields"
    instance:
      type: string
      description: URI reference to the specific occurrence
      example: "/v1/agents"
    trace_id:
      type: string
      description: Request trace ID for debugging
      example: "abc123xyz"
    errors:
      type: array
      description: Validation errors (if applicable)
      items:
        $ref: '#/components/schemas/ValidationError'
```

### 2.2 Validation Error

```yaml
ValidationError:
  type: object
  required: [field, message]
  properties:
    field:
      type: string
      description: JSON path to invalid field
      example: "config.model"
    message:
      type: string
      description: Human-readable error message
      example: "must be a valid model identifier"
    code:
      type: string
      description: Machine-readable error code
      example: "invalid_model"
```

### 2.3 Standard Error Types

| Type | Status | Description |
|------|--------|-------------|
| `validation` | 400 | Request validation failed |
| `unauthorized` | 401 | Authentication required |
| `forbidden` | 403 | Insufficient permissions |
| `not-found` | 404 | Resource not found |
| `conflict` | 409 | Resource conflict |
| `rate-limited` | 429 | Rate limit exceeded |
| `internal` | 500 | Internal server error |
| `service-unavailable` | 503 | Service temporarily unavailable |

### 2.4 Error Response Examples

```yaml
# 400 - Validation Error
{
  "type": "https://agentstack.dev/errors/validation",
  "title": "Validation Error",
  "status": 400,
  "detail": "Request validation failed",
  "instance": "/v1/agents",
  "trace_id": "abc123",
  "errors": [
    {"field": "name", "message": "is required"},
    {"field": "config.model", "message": "must be a valid model"}
  ]
}

# 401 - Unauthorized
{
  "type": "https://agentstack.dev/errors/unauthorized",
  "title": "Unauthorized",
  "status": 401,
  "detail": "Invalid or expired API key"
}

# 404 - Not Found
{
  "type": "https://agentstack.dev/errors/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "Agent 'agt_abc123' not found"
}

# 429 - Rate Limited
{
  "type": "https://agentstack.dev/errors/rate-limited",
  "title": "Rate Limited",
  "status": 429,
  "detail": "Rate limit exceeded. Check Retry-After header."
}
```

---

## 3. Pagination

### 3.1 Cursor-Based Pagination

```yaml
Pagination:
  type: object
  properties:
    has_more:
      type: boolean
      description: More results available
    next_cursor:
      type: string
      description: Cursor for next page
    prev_cursor:
      type: string
      description: Cursor for previous page
```

### 3.2 Paginated Response

```yaml
PaginatedResponse:
  type: object
  required: [data, pagination]
  properties:
    data:
      type: array
      items: {}  # Actual resource type
    pagination:
      $ref: '#/components/schemas/Pagination'
```

### 3.3 Usage

```yaml
# Request
GET /v1/agents?limit=20&cursor=eyJpZCI6ImFndF8xMjMifQ

# Response
{
  "data": [...],
  "pagination": {
    "has_more": true,
    "next_cursor": "eyJpZCI6ImFndF80NTYifQ"
  }
}
```

---

## 4. Agent Schemas

### 4.1 Agent

```yaml
Agent:
  type: object
  required: [id, name, status, framework, created_at]
  properties:
    id:
      type: string
      example: "agt_3kLm4nOpQr"
    name:
      type: string
      minLength: 3
      maxLength: 64
    slug:
      type: string
      pattern: "^[a-z][a-z0-9-]*$"
    description:
      type: string
      maxLength: 500
    status:
      type: string
      enum: [active, inactive, deploying, failed, suspended, deleting]
    framework:
      type: string
      enum: [google-adk, langchain, crewai, autogen, custom]
    source:
      $ref: '#/components/schemas/AgentSource'
    config:
      $ref: '#/components/schemas/AgentConfig'
    current_deployment:
      $ref: '#/components/schemas/DeploymentSummary'
    urls:
      $ref: '#/components/schemas/AgentURLs'
    tags:
      type: array
      items:
        type: string
      maxItems: 10
    created_at:
      type: string
      format: date-time
    updated_at:
      type: string
      format: date-time
```

### 4.2 Agent Source

```yaml
AgentSource:
  type: object
  required: [type]
  properties:
    type:
      type: string
      enum: [git, image, inline]
    repo:
      type: string
      description: Git repository URL
    branch:
      type: string
      default: "main"
    path:
      type: string
      default: "."
    image:
      type: string
      description: Container image URL
    runtime:
      type: string
      enum: [python3.11, python3.12, node20]
    entrypoint:
      type: string
      description: Entry point file
    files:
      type: object
      additionalProperties:
        type: string
      description: Inline files (path → content)
```

### 4.3 Agent Config

```yaml
AgentConfig:
  type: object
  properties:
    model:
      type: string
      example: "gpt-4o"
    system_prompt:
      type: string
    tools:
      type: array
      items:
        type: string
    memory:
      $ref: '#/components/schemas/MemoryConfig'
    scaling:
      $ref: '#/components/schemas/ScalingConfig'
    timeout:
      type: string
      pattern: "^\\d+[smh]$"
      default: "60s"
    env:
      type: object
      additionalProperties:
        type: string
```

### 4.4 Memory Config

```yaml
MemoryConfig:
  type: object
  properties:
    enabled:
      type: boolean
      default: true
    type:
      type: string
      enum: [session, persistent, vector]
    max_tokens:
      type: integer
      default: 4000
    ttl:
      type: string
      pattern: "^\\d+[dhm]$"
```

### 4.5 Scaling Config

```yaml
ScalingConfig:
  type: object
  properties:
    min_instances:
      type: integer
      minimum: 0
      default: 0
    max_instances:
      type: integer
      minimum: 1
      default: 10
    target_concurrency:
      type: integer
      default: 80
```

### 4.6 Agent URLs

```yaml
AgentURLs:
  type: object
  properties:
    chat:
      type: string
      format: uri
      example: "https://agent.agentstack.app/chat"
    stream:
      type: string
      format: uri
      example: "https://agent.agentstack.app/chat/stream"
    dashboard:
      type: string
      format: uri
```

---

## 5. Deployment Schemas

### 5.1 Deployment

```yaml
Deployment:
  type: object
  required: [id, agent_id, status, created_at]
  properties:
    id:
      type: string
      example: "dpl_4mNoPqRsT"
    agent_id:
      type: string
    revision_id:
      type: string
    status:
      type: string
      enum: [queued, building, deploying, ready, failed, cancelled]
    source:
      $ref: '#/components/schemas/AgentSource'
    url:
      type: string
      format: uri
    error:
      $ref: '#/components/schemas/DeploymentError'
    meta:
      type: object
      properties:
        commit:
          type: string
        branch:
          type: string
        author:
          type: string
    timing:
      $ref: '#/components/schemas/DeploymentTiming'
    created_at:
      type: string
      format: date-time
```

### 5.2 Deployment Timing

```yaml
DeploymentTiming:
  type: object
  properties:
    queued_at:
      type: string
      format: date-time
    build_started_at:
      type: string
      format: date-time
    build_finished_at:
      type: string
      format: date-time
    deployed_at:
      type: string
      format: date-time
```

### 5.3 Deployment Error

```yaml
DeploymentError:
  type: object
  properties:
    code:
      type: string
      enum: [build_failed, timeout, resource_limit, config_error, 
             runtime_error, dependency_error]
    message:
      type: string
    details:
      type: string
```

### 5.4 Revision

```yaml
Revision:
  type: object
  required: [id, agent_id, deployment_id, status, created_at]
  properties:
    id:
      type: string
      example: "rev_5nOpQrStU"
    agent_id:
      type: string
    deployment_id:
      type: string
    status:
      type: string
      enum: [active, inactive, draining]
    traffic_percent:
      type: integer
      minimum: 0
      maximum: 100
    config:
      $ref: '#/components/schemas/AgentConfig'
    meta:
      type: object
      properties:
        commit:
          type: string
        image_digest:
          type: string
    created_at:
      type: string
      format: date-time
```

---

## 6. Chat Schemas

### 6.1 Chat Message

```yaml
ChatMessage:
  type: object
  required: [role, content]
  properties:
    role:
      type: string
      enum: [system, user, assistant, tool]
    content:
      type: string
    name:
      type: string
      description: Tool name (for role=tool)
    tool_call_id:
      type: string
      description: Tool call ID (for role=tool)
```

### 6.2 Chat Request

```yaml
ChatRequest:
  type: object
  required: [messages]
  properties:
    messages:
      type: array
      items:
        $ref: '#/components/schemas/ChatMessage'
      minItems: 1
    session_id:
      type: string
    stream:
      type: boolean
      default: false
    model:
      type: string
    temperature:
      type: number
      minimum: 0
      maximum: 2
    max_tokens:
      type: integer
    tools:
      type: array
      items:
        type: string
    user:
      type: string
      description: End-user identifier
    metadata:
      type: object
```

### 6.3 Chat Response

```yaml
ChatResponse:
  type: object
  required: [id, message, usage]
  properties:
    id:
      type: string
    message:
      $ref: '#/components/schemas/ChatMessage'
    session_id:
      type: string
    tool_calls:
      type: array
      items:
        $ref: '#/components/schemas/ToolCall'
    usage:
      $ref: '#/components/schemas/Usage'
    model:
      type: string
    finish_reason:
      type: string
      enum: [stop, tool_calls, length, error]
```

### 6.4 Tool Call

```yaml
ToolCall:
  type: object
  required: [id, type, function]
  properties:
    id:
      type: string
    type:
      type: string
      enum: [function]
    function:
      type: object
      required: [name, arguments]
      properties:
        name:
          type: string
        arguments:
          type: string
          description: JSON-encoded arguments
```

### 6.5 Usage

```yaml
Usage:
  type: object
  properties:
    prompt_tokens:
      type: integer
    completion_tokens:
      type: integer
    total_tokens:
      type: integer
```

### 6.6 Session

```yaml
Session:
  type: object
  required: [id, agent_id, created_at]
  properties:
    id:
      type: string
      example: "ses_6oPqRsTuV"
    agent_id:
      type: string
    user_id:
      type: string
    metadata:
      type: object
    message_count:
      type: integer
    last_message_at:
      type: string
      format: date-time
    expires_at:
      type: string
      format: date-time
    created_at:
      type: string
      format: date-time
```

---

## 7. Tool Schemas

### 7.1 Tool

```yaml
Tool:
  type: object
  required: [id, name, type, created_at]
  properties:
    id:
      type: string
      example: "tol_7pQrStUvW"
    name:
      type: string
      pattern: "^[a-z][a-z0-9_-]*$"
    description:
      type: string
    type:
      type: string
      enum: [builtin, http, code, mcp]
    category:
      type: string
    config:
      type: object
      description: Type-specific configuration
    input_schema:
      type: object
      description: JSON Schema for input
    output_schema:
      type: object
      description: JSON Schema for output
    enabled:
      type: boolean
      default: true
    usage_count:
      type: integer
    created_at:
      type: string
      format: date-time
    updated_at:
      type: string
      format: date-time
```

### 7.2 HTTP Tool Config

```yaml
HTTPToolConfig:
  type: object
  required: [method, url]
  properties:
    method:
      type: string
      enum: [GET, POST, PUT, PATCH, DELETE]
    url:
      type: string
      format: uri-template
    headers:
      type: object
      additionalProperties:
        type: string
    timeout:
      type: string
      default: "30s"
    retry:
      type: object
      properties:
        attempts:
          type: integer
          default: 3
        backoff:
          type: string
          default: "exponential"
```

### 7.3 MCP Server

```yaml
MCPServer:
  type: object
  required: [id, name, url, created_at]
  properties:
    id:
      type: string
      example: "mcp_8qRsTuVwX"
    name:
      type: string
    url:
      type: string
      format: uri
    transport:
      type: string
      enum: [stdio, sse]
    status:
      type: string
      enum: [connected, disconnected, error]
    tools:
      type: array
      items:
        type: string
      description: List of tool names
    created_at:
      type: string
      format: date-time
```

---

## 8. Organization Schemas

### 8.1 Team

```yaml
Team:
  type: object
  required: [id, name, created_at]
  properties:
    id:
      type: string
      example: "team_6pQr7sStuV"
    name:
      type: string
      minLength: 2
      maxLength: 64
    slug:
      type: string
      pattern: "^[a-z][a-z0-9-]*$"
    avatar_url:
      type: string
      format: uri
    member_count:
      type: integer
    project_count:
      type: integer
    billing:
      $ref: '#/components/schemas/BillingInfo'
    created_at:
      type: string
      format: date-time
```

### 8.2 Team Member

```yaml
TeamMember:
  type: object
  required: [user_id, email, role]
  properties:
    user_id:
      type: string
    email:
      type: string
      format: email
    name:
      type: string
    avatar_url:
      type: string
      format: uri
    role:
      type: string
      enum: [owner, admin, member, viewer]
    joined_at:
      type: string
      format: date-time
```

### 8.3 Project

```yaml
Project:
  type: object
  required: [id, name, created_at]
  properties:
    id:
      type: string
      example: "prj_3kLm4nOpQr"
    name:
      type: string
    slug:
      type: string
    description:
      type: string
    team_id:
      type: string
    settings:
      type: object
      properties:
        default_model:
          type: string
        build_timeout:
          type: string
        log_retention:
          type: string
    stats:
      type: object
      properties:
        agent_count:
          type: integer
        deployment_count:
          type: integer
        request_count_30d:
          type: integer
    created_at:
      type: string
      format: date-time
    updated_at:
      type: string
      format: date-time
```

### 8.4 Billing Info

```yaml
BillingInfo:
  type: object
  properties:
    plan:
      type: string
      enum: [free, pro, enterprise]
    status:
      type: string
      enum: [active, past_due, cancelled]
```

---

## 9. Infrastructure Schemas

### 9.1 Domain

```yaml
Domain:
  type: object
  required: [id, domain, status, created_at]
  properties:
    id:
      type: string
      example: "dom_8rSt9uVwXy"
    domain:
      type: string
      pattern: "^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$"
    status:
      type: string
      enum: [pending, verified, active, error]
    verification:
      type: object
      properties:
        type:
          type: string
          enum: [cname, txt]
        name:
          type: string
        value:
          type: string
    ssl:
      type: object
      properties:
        status:
          type: string
          enum: [pending, active, error]
        expires_at:
          type: string
          format: date-time
    agent_id:
      type: string
    created_at:
      type: string
      format: date-time
```

### 9.2 Webhook

```yaml
Webhook:
  type: object
  required: [id, url, events, created_at]
  properties:
    id:
      type: string
      example: "whk_9sTu0vWxYz"
    url:
      type: string
      format: uri
    events:
      type: array
      items:
        type: string
        enum:
          - agent.created
          - agent.updated
          - agent.deleted
          - deployment.started
          - deployment.succeeded
          - deployment.failed
          - chat.message
          - chat.response
          - alert.triggered
    secret:
      type: string
      description: Only shown on create
    status:
      type: string
      enum: [active, paused, failing]
    failure_count:
      type: integer
    last_delivery:
      type: object
      properties:
        status:
          type: string
        timestamp:
          type: string
          format: date-time
    created_at:
      type: string
      format: date-time
```

### 9.3 API Key

```yaml
ApiKey:
  type: object
  required: [id, name, created_at]
  properties:
    id:
      type: string
      example: "key_1gTs8vXxZa"
    name:
      type: string
      maxLength: 64
    prefix:
      type: string
      description: First 8 characters for identification
    scope:
      type: string
      enum: [project, team]
    permissions:
      type: array
      items:
        type: string
        enum:
          - agents:read
          - agents:write
          - agents:chat
          - deployments:read
          - deployments:write
          - tools:read
          - tools:write
          - secrets:read
          - secrets:write
          - projects:read
          - projects:write
          - admin
    last_used_at:
      type: string
      format: date-time
    expires_at:
      type: string
      format: date-time
    created_at:
      type: string
      format: date-time
```

---

## 10. Analytics Schemas

### 10.1 Usage Metrics

```yaml
UsageMetrics:
  type: object
  properties:
    period:
      type: object
      properties:
        start:
          type: string
          format: date-time
        end:
          type: string
          format: date-time
    totals:
      type: object
      properties:
        requests:
          type: integer
        tokens_input:
          type: integer
        tokens_output:
          type: integer
        tool_calls:
          type: integer
        errors:
          type: integer
        cost_usd:
          type: number
    series:
      type: array
      items:
        type: object
        properties:
          timestamp:
            type: string
            format: date-time
          requests:
            type: integer
          tokens:
            type: integer
          errors:
            type: integer
```

### 10.2 Log Entry

```yaml
LogEntry:
  type: object
  properties:
    timestamp:
      type: string
      format: date-time
    level:
      type: string
      enum: [debug, info, warn, error]
    message:
      type: string
    request_id:
      type: string
    session_id:
      type: string
    metadata:
      type: object
```

---

## 11. Common Patterns

### 11.1 Timestamp Fields

All timestamps use ISO 8601 format with timezone:

```yaml
created_at: "2025-01-15T10:30:00Z"
updated_at: "2025-01-15T10:30:00Z"
```

### 11.2 Duration Strings

Durations use Go-style format:

```yaml
# Format: \d+[smhd]
timeout: "30s"      # 30 seconds
ttl: "24h"          # 24 hours
retention: "30d"    # 30 days
```

### 11.3 Optional Fields

Null values are omitted from responses (not sent as `null`):

```yaml
# Good: field omitted
{"id": "agt_xxx", "name": "My Agent"}

# Avoid: null values
{"id": "agt_xxx", "name": "My Agent", "description": null}
```

### 11.4 Audit Fields

Resources track creation and modification:

```yaml
created_at: "2025-01-15T10:00:00Z"
updated_at: "2025-01-15T10:30:00Z"
created_by: "usr_abc"      # When applicable
```

---

**Previous**: [015-admin-endpoints.md](015-admin-endpoints.md)  
**Back to**: [README.md](README.md)
