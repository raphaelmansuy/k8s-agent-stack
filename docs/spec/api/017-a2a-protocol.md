# 017 - A2A Protocol Integration

> Agent-to-Agent Protocol Implementation for AgentStack

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

> **See Also**: [Universal Content Model](020-universal-content-model.md) for UCM ↔ A2A Part mapping

---

## 1. Overview

The Agent-to-Agent (A2A) Protocol is an open standard from the Linux Foundation that enables interoperability between AI agents built on different frameworks. AgentStack implements A2A to enable:

- **Agent Discovery**: Agents publish capabilities via Agent Cards
- **Cross-Platform Collaboration**: Agents communicate regardless of framework
- **Task Management**: Standardized task lifecycle and state tracking
- **Enterprise Features**: Authentication, streaming, push notifications

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    A2A Protocol Architecture                            │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   External Agent                    AgentStack Agent                    │
│   (Any Framework)                   (ADK/LangGraph/CrewAI)             │
│        │                                    │                           │
│        │  A2A SendMessage                   │                           │
│        │───────────────────────────────────▶│                           │
│        │                                    │                           │
│        │  A2A Task/Streaming Response       │                           │
│        │◀───────────────────────────────────│                           │
│        │                                    │                           │
│        │  Agent Card Discovery              │                           │
│        │───────────────▶ /.well-known/agent-card.json                   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. A2A Endpoints

### 2.1 Endpoint Summary

| Operation | Method | Endpoint | Description |
|-----------|--------|----------|-------------|
| Agent Discovery | GET | `/.well-known/agent-card.json` | Get public Agent Card |
| Extended Card | GET | `/a2a/v1/extendedAgentCard` | Get authenticated Agent Card |
| Send Message | POST | `/a2a/v1/message:send` | Send message to agent |
| Stream Message | POST | `/a2a/v1/message:stream` | Stream message response |
| Get Task | GET | `/a2a/v1/tasks/{taskId}` | Get task status |
| List Tasks | GET | `/a2a/v1/tasks` | List tasks |
| Cancel Task | POST | `/a2a/v1/tasks/{taskId}:cancel` | Cancel running task |
| Subscribe | POST | `/a2a/v1/tasks/{taskId}:subscribe` | Subscribe to task updates |
| Push Config | POST | `/a2a/v1/tasks/{taskId}/pushNotificationConfigs` | Set webhook |

### 2.2 Protocol Headers

```yaml
# Required Headers
Content-Type: application/a2a+json    # A2A media type
Authorization: Bearer <token>          # Or X-API-Key

# Optional Headers
A2A-Version: 1.0                       # Protocol version
A2A-Extensions: uri1,uri2              # Extension URIs
X-Project-ID: prj_xxx                  # Tenant context (AgentStack)
```

---

## 3. Agent Card

### 3.1 Discovery Endpoint

```yaml
GET /.well-known/agent-card.json

# No authentication required for public card
# Returns: application/a2a+json
```

### 3.2 Agent Card Schema

```json
{
  "protocolVersion": "1.0",
  "name": "customer-support",
  "description": "Customer support agent for order inquiries and ticket creation",
  
  "supportedInterfaces": [
    {
      "url": "https://customer-support.agentstack.app/a2a/v1",
      "protocolBinding": "HTTP+JSON"
    }
  ],
  
  "provider": {
    "organization": "Acme Corp",
    "url": "https://acme.com"
  },
  
  "version": "1.0.0",
  "iconUrl": "https://customer-support.agentstack.app/icon.png",
  "documentationUrl": "https://docs.acme.com/support-agent",
  
  "capabilities": {
    "streaming": true,
    "pushNotifications": true,
    "stateTransitionHistory": false,
    "extensions": []
  },
  
  "securitySchemes": {
    "bearerAuth": {
      "type": "http",
      "scheme": "bearer"
    },
    "apiKey": {
      "type": "apiKey",
      "in": "header",
      "name": "X-API-Key"
    }
  },
  
  "security": [
    {"bearerAuth": []},
    {"apiKey": []}
  ],
  
  "defaultInputModes": ["text/plain", "application/json"],
  "defaultOutputModes": ["text/plain", "application/json", "text/html"],
  
  "skills": [
    {
      "id": "order-lookup",
      "name": "Order Status Lookup",
      "description": "Check the status of customer orders by order number",
      "tags": ["orders", "status", "tracking", "shipping"],
      "examples": [
        "What's the status of order #12345?",
        "Where is my package?",
        "Track order ORD-2025-001"
      ],
      "inputModes": ["text/plain"],
      "outputModes": ["text/plain", "application/json"]
    },
    {
      "id": "ticket-creation",
      "name": "Support Ticket Creation",
      "description": "Create support tickets for customer issues",
      "tags": ["support", "tickets", "issues", "help"],
      "examples": [
        "I have a problem with my order",
        "Create a ticket for a refund request"
      ]
    }
  ],
  
  "supportsExtendedAgentCard": true
}
```

### 3.3 Agent Card Generation

Agent Cards are auto-generated from the kagent Agent CRD:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: customer-support
  namespace: production
  annotations:
    # A2A metadata
    a2a.agentstack.io/description: "Customer support agent"
    a2a.agentstack.io/documentation-url: "https://docs.acme.com"
    a2a.agentstack.io/icon-url: "https://example.com/icon.png"
spec:
  type: Declarative
  declarative:
    modelConfig: gpt-4o-config
    systemMessage: |
      You are a customer support agent.
    tools:
      - mcpServer:
          name: support-tools
          tools:
            - lookup-order      # Becomes skill: order-lookup
            - create-ticket     # Becomes skill: ticket-creation
  
  # A2A-specific configuration
  a2a:
    enabled: true
    skills:
      - id: order-lookup
        name: Order Status Lookup
        description: Check order status
        tags: ["orders", "tracking"]
        examples:
          - "What's the status of order #12345?"
      - id: ticket-creation
        name: Support Ticket Creation
        description: Create support tickets
        tags: ["support", "tickets"]
```

---

## 4. Message Operations

### 4.1 Send Message (Synchronous)

```yaml
POST /a2a/v1/message:send
Content-Type: application/a2a+json
Authorization: Bearer <token>
X-Project-ID: prj_abc123

Request:
{
  "message": {
    "messageId": "msg_uuid_123",
    "role": "user",
    "parts": [
      {"text": "What's the status of order #12345?"}
    ],
    "contextId": "ctx_existing_conversation",  # Optional
    "taskId": "task_existing_task"             # Optional, for follow-ups
  },
  "configuration": {
    "acceptedOutputModes": ["text/plain", "application/json"],
    "historyLength": 10,
    "blocking": true        # Wait for completion
  },
  "metadata": {
    "user_id": "usr_xyz",
    "channel": "web"
  }
}

Response: 200 OK
{
  "task": {
    "id": "task_abc123",
    "contextId": "ctx_def456",
    "status": {
      "state": "completed",
      "timestamp": "2025-01-15T10:30:00Z"
    },
    "artifacts": [
      {
        "artifactId": "art_1",
        "name": "Order Status",
        "parts": [
          {
            "text": "Order #12345 was shipped on January 14th and is expected to arrive by January 17th."
          }
        ]
      }
    ],
    "history": [
      {
        "messageId": "msg_uuid_123",
        "role": "user",
        "parts": [{"text": "What's the status of order #12345?"}]
      }
    ]
  }
}
```

### 4.2 Send Streaming Message

```yaml
POST /a2a/v1/message:stream
Content-Type: application/a2a+json
Accept: text/event-stream
Authorization: Bearer <token>

Request: (same as sync)

Response: 200 OK
Content-Type: text/event-stream

# Initial task
data: {"task": {"id": "task_abc", "contextId": "ctx_def", "status": {"state": "working"}}}

# Status update
data: {"statusUpdate": {"taskId": "task_abc", "contextId": "ctx_def", "status": {"state": "working"}, "final": false}}

# Artifact streaming (with append)
data: {"artifactUpdate": {"taskId": "task_abc", "contextId": "ctx_def", "artifact": {"artifactId": "art_1", "parts": [{"text": "Order #12345 "}]}, "append": true, "lastChunk": false}}

data: {"artifactUpdate": {"taskId": "task_abc", "contextId": "ctx_def", "artifact": {"artifactId": "art_1", "parts": [{"text": "was shipped on January 14th..."}]}, "append": true, "lastChunk": true}}

# Final status
data: {"statusUpdate": {"taskId": "task_abc", "contextId": "ctx_def", "status": {"state": "completed"}, "final": true}}
```

### 4.3 Message Parts

A2A supports multiple content types in message parts:

```yaml
# Text Part
{"text": "Hello, how can I help?"}

# File Part (inline bytes)
{
  "file": {
    "mediaType": "image/png",
    "name": "screenshot.png",
    "fileWithBytes": "iVBORw0KGgoAAAANSUhEUgAAAAUA..."  # Base64
  }
}

# File Part (URI reference)
{
  "file": {
    "mediaType": "application/pdf",
    "name": "invoice.pdf",
    "fileWithUri": "https://storage.example.com/files/invoice.pdf"
  }
}

# Data Part (structured JSON)
{
  "data": {
    "data": {
      "order_id": "12345",
      "status": "shipped",
      "tracking": "1Z999AA10123456784"
    }
  }
}
```

---

## 5. Task Management

### 5.1 Task States

| State | Description | Terminal |
|-------|-------------|----------|
| `submitted` | Task created, queued | No |
| `working` | Processing in progress | No |
| `input-required` | Waiting for user input | No (interrupted) |
| `auth-required` | Additional auth needed | No |
| `completed` | Successfully finished | Yes |
| `failed` | Error occurred | Yes |
| `cancelled` | User cancelled | Yes |
| `rejected` | Agent refused task | Yes |

### 5.2 Get Task

```yaml
GET /a2a/v1/tasks/{taskId}?historyLength=10
Authorization: Bearer <token>

Response: 200 OK
{
  "id": "task_abc123",
  "contextId": "ctx_def456",
  "status": {
    "state": "completed",
    "message": {
      "role": "agent",
      "parts": [{"text": "Task completed successfully"}]
    },
    "timestamp": "2025-01-15T10:30:00Z"
  },
  "artifacts": [...],
  "history": [...],
  "metadata": {...}
}
```

### 5.3 List Tasks

```yaml
GET /a2a/v1/tasks
  ?contextId=ctx_def456
  &status=working
  &pageSize=20
  &pageToken=xxx
  &lastUpdatedAfter=1705312200000
Authorization: Bearer <token>

Response: 200 OK
{
  "tasks": [
    {
      "id": "task_1",
      "contextId": "ctx_def456",
      "status": {"state": "working", "timestamp": "..."}
    }
  ],
  "totalSize": 42,
  "pageSize": 20,
  "nextPageToken": "yyy"
}
```

### 5.4 Cancel Task

```yaml
POST /a2a/v1/tasks/{taskId}:cancel
Authorization: Bearer <token>

Response: 200 OK
{
  "id": "task_abc123",
  "status": {
    "state": "cancelled",
    "timestamp": "2025-01-15T10:31:00Z"
  }
}
```

### 5.5 Subscribe to Task

```yaml
POST /a2a/v1/tasks/{taskId}:subscribe
Accept: text/event-stream
Authorization: Bearer <token>

Response: 200 OK
Content-Type: text/event-stream

# Returns current task state, then streams updates
data: {"task": {"id": "task_abc", "status": {"state": "working"}}}

data: {"statusUpdate": {"taskId": "task_abc", "status": {"state": "completed"}, "final": true}}
```

---

## 6. Push Notifications

### 6.1 Configure Webhook

```yaml
POST /a2a/v1/tasks/{taskId}/pushNotificationConfigs
Authorization: Bearer <token>

Request:
{
  "url": "https://myapp.example.com/webhooks/a2a",
  "token": "webhook-secret-token",
  "authentication": {
    "schemes": ["Bearer"],
    "credentials": "my-webhook-auth-token"
  }
}

Response: 201 Created
{
  "id": "pnc_xyz",
  "url": "https://myapp.example.com/webhooks/a2a",
  "token": "webhook-secret-token"
}
```

### 6.2 Webhook Payload

When task updates occur, AgentStack sends:

```yaml
POST https://myapp.example.com/webhooks/a2a
Content-Type: application/a2a+json
Authorization: Bearer my-webhook-auth-token
X-A2A-Notification-Token: webhook-secret-token

{
  "statusUpdate": {
    "taskId": "task_abc123",
    "contextId": "ctx_def456",
    "status": {
      "state": "completed",
      "timestamp": "2025-01-15T10:30:00Z"
    },
    "final": true
  }
}
```

---

## 7. Multi-Turn Conversations

### 7.1 Context Management

A2A uses `contextId` to group related tasks:

```text
Context: ctx_abc123
├── Task: task_001 (completed)
│   └── User: "What's the weather?"
├── Task: task_002 (completed)  
│   └── User: "What about tomorrow?"
└── Task: task_003 (working)
    └── User: "Should I bring an umbrella?"
```

### 7.2 Input Required State

When the agent needs more information:

```yaml
# Response with input-required state
{
  "task": {
    "id": "task_abc",
    "status": {
      "state": "input-required",
      "message": {
        "role": "agent",
        "parts": [{"text": "Which order would you like me to check? Please provide the order number."}]
      }
    }
  }
}

# Client sends follow-up
POST /a2a/v1/message:send
{
  "message": {
    "messageId": "msg_2",
    "role": "user",
    "taskId": "task_abc",           # Continue existing task
    "contextId": "ctx_def",
    "parts": [{"text": "Order #12345"}]
  }
}
```

---

## 8. Error Handling

### 8.1 A2A Error Types

| Error | HTTP Code | Description |
|-------|-----------|-------------|
| `TaskNotFoundError` | 404 | Task ID doesn't exist |
| `TaskNotCancelableError` | 409 | Task in terminal state |
| `ContentTypeNotSupportedError` | 415 | Unsupported media type |
| `UnsupportedOperationError` | 400 | Operation not supported |
| `PushNotificationNotSupportedError` | 400 | Push not enabled |
| `VersionNotSupportedError` | 400 | A2A version not supported |

### 8.2 Error Response Format

```json
{
  "type": "https://a2a-protocol.org/errors/task-not-found",
  "title": "Task Not Found",
  "status": 404,
  "detail": "The specified task ID does not exist",
  "taskId": "task_nonexistent",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

---

## 9. Integration with AgentStack

### 9.1 REST API Mapping

| A2A Concept | AgentStack Mapping |
|-------------|-------------------|
| `contextId` | Session ID (`ses_xxx`) |
| `taskId` | Message exchange ID |
| `artifact` | Response with attachments |
| `Agent Card` | Agent metadata from CRD |
| `skill` | Tool capabilities |

### 9.2 Dual API Support

AgentStack provides both APIs:

```text
# Traditional REST (for client apps)
POST /v1/agents/{agentId}/chat/stream
→ Simple, opinionated, AgentStack-specific

# A2A Protocol (for agent interop)
POST /a2a/v1/message:stream
→ Standard, cross-platform, verbose
```

### 9.3 Protocol Detection

```yaml
# Automatic protocol routing based on headers/path

# A2A requests (prefix + content-type)
POST /a2a/v1/message:send
Content-Type: application/a2a+json
→ Routes to A2A handler

# REST requests (traditional)
POST /v1/agents/{agentId}/chat
Content-Type: application/json
→ Routes to REST handler
```

---

## 10. Security Considerations

### 10.1 Authentication

A2A requests use the same auth as REST API:

- Bearer tokens (JWT)
- API Keys (`X-API-Key`)
- Service accounts (internal)

### 10.2 Authorization Scoping

All A2A operations respect AgentStack tenant boundaries:

- Tasks scoped to authenticated project
- Agent Cards filtered by permissions
- Cross-tenant access denied

### 10.3 Agent Card Security

- Public cards at `/.well-known/` omit sensitive details
- Extended cards require authentication
- Optional JWS signatures for integrity

---

## 11. References

- [A2A Protocol Specification](https://a2a-protocol.org/latest/specification/)
- [A2A GitHub Repository](https://github.com/a2aproject/A2A)
- [AgentStack Chat API](013-chat-sessions.md)
- [AgentStack Authentication](011-authentication.md)

---

*End of A2A Protocol Specification*
