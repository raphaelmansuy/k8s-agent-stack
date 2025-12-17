# 013 - Chat & Sessions API

> Agent Interaction, Streaming, Sessions, and Conversation History

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

> **See Also**: 
> - [AG-UI Protocol](018-agui-protocol.md) for advanced streaming events
> - [A2A Protocol](017-a2a-protocol.md) for agent-to-agent communication
> - [Universal Content Model](020-universal-content-model.md) for multimodal content format
> - [Interactions API](021-interactions-api.md) for server-side state management

---

## 1. Overview

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Chat Architecture                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Client                                                          │
│    │                                                             │
│    ├── POST /chat ────────────▶ Sync Response                   │
│    │                                                             │
│    └── POST /chat/stream ─────▶ SSE Stream                      │
│              │                                                   │
│              ▼                                                   │
│         ┌─────────────────────────────────────────┐             │
│         │  Session (optional)                     │             │
│         │  • Conversation history                 │             │
│         │  • Context persistence                  │             │
│         │  • User identity                        │             │
│         └─────────────────────────────────────────┘             │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Chat Endpoints

### 2.1 Synchronous Chat

```yaml
POST /v1/agents/{agentId}/chat
X-API-Key: as_prj_sk_live_xxx

Request:
{
  "message": "What's the weather in Tokyo?",
  "session_id": "ses_abc123",           # Optional
  "context": {
    "user_id": "usr_xyz",               # Optional
    "metadata": {
      "timezone": "Asia/Tokyo",
      "locale": "ja-JP"
    }
  },
  "tools": {                            # Optional tool constraints
    "allow": ["weather-api"],
    "deny": ["execute-code"]
  },
  "model_override": "gpt-4o",           # Optional
  "response_format": {                  # Optional
    "type": "json_object"
  }
}

Response: 200 OK
{
  "id": "msg_9xLm4nPqRs",
  "session_id": "ses_abc123",
  "message": "The current weather in Tokyo is 22°C and sunny.",
  "tool_calls": [
    {
      "id": "tc_xyz",
      "tool": "weather-api",
      "input": {"city": "Tokyo"},
      "output": {"temp": 22, "condition": "sunny"},
      "duration_ms": 150
    }
  ],
  "usage": {
    "input_tokens": 45,
    "output_tokens": 28,
    "total_tokens": 73
  },
  "model": "gpt-4o",
  "duration_ms": 1250,
  "created_at": "2025-01-15T10:30:00Z"
}
```

### 2.2 Streaming Chat

```yaml
POST /v1/agents/{agentId}/chat/stream
Accept: text/event-stream

Request: (same as sync)
{
  "message": "What's the weather in Tokyo?"
}

Response: 200 OK
Content-Type: text/event-stream

# Stream events:
event: message_start
data: {"message_id": "msg_abc", "role": "assistant"}

event: content_delta
data: {"delta": "Let me check the weather"}

event: tool_start
data: {"id": "tc_xyz", "tool": "weather-api", "input": {"city": "Tokyo"}}

event: tool_result
data: {"id": "tc_xyz", "tool": "weather-api", "output": {"temp": 22, "condition": "sunny"}}

event: content_delta
data: {"delta": " in Tokyo. It's currently 22°C and sunny!"}

event: message_end
data: {"usage": {"input_tokens": 45, "output_tokens": 28}, "duration_ms": 1250}
```

### 2.3 Content-Negotiated Streaming

```yaml
# Request streaming via Accept header
POST /v1/agents/{agentId}/chat
Accept: text/event-stream

# Response automatically streams
Content-Type: text/event-stream
```

---

## 3. Session Management

### 3.1 Create Session

```yaml
POST /v1/agents/{agentId}/sessions

Request:
{
  "user_id": "usr_xyz",                 # Optional
  "metadata": {                         # Optional
    "channel": "web",
    "browser": "Chrome"
  },
  "system_prompt_override": "Custom instructions...",  # Optional
  "ttl": "24h"                          # Default: 24h
}

Response: 201 Created
{
  "id": "ses_5mNn6pQrSt",
  "agent_id": "agt_2xKj9mNpQr",
  "user_id": "usr_xyz",
  "metadata": { ... },
  "message_count": 0,
  "created_at": "2025-01-15T10:30:00Z",
  "expires_at": "2025-01-16T10:30:00Z"
}
```

### 3.2 List Sessions

```yaml
GET /v1/agents/{agentId}/sessions
  ?user_id=usr_xyz
  &limit=20
  &cursor=xxx

Response: 200 OK
{
  "data": [
    {
      "id": "ses_5mNn6pQrSt",
      "user_id": "usr_xyz",
      "message_count": 15,
      "last_message_at": "2025-01-15T10:45:00Z",
      "created_at": "2025-01-15T10:30:00Z"
    }
  ],
  "pagination": { ... }
}
```

### 3.3 Get Session (with history)

```yaml
GET /v1/agents/{agentId}/sessions/{sessionId}

Response: 200 OK
{
  "id": "ses_5mNn6pQrSt",
  "agent_id": "agt_2xKj9mNpQr",
  "user_id": "usr_xyz",
  "metadata": { ... },
  "message_count": 3,
  "messages": [
    {
      "id": "msg_1",
      "role": "user",
      "content": "Hello",
      "created_at": "2025-01-15T10:30:00Z"
    },
    {
      "id": "msg_2",
      "role": "assistant",
      "content": "Hello! How can I help?",
      "tool_calls": [],
      "created_at": "2025-01-15T10:30:01Z"
    },
    {
      "id": "msg_3",
      "role": "user",
      "content": "What's the weather?",
      "created_at": "2025-01-15T10:30:30Z"
    }
  ],
  "created_at": "2025-01-15T10:30:00Z",
  "expires_at": "2025-01-16T10:30:00Z"
}
```

### 3.4 Delete Session

```yaml
DELETE /v1/agents/{agentId}/sessions/{sessionId}

Response: 204 No Content
```

---

## 4. SSE Event Reference

### 4.1 Event Types

| Event | Description | Data |
|-------|-------------|------|
| `message_start` | Response begun | `{message_id, role}` |
| `content_delta` | Text chunk | `{delta}` |
| `tool_start` | Tool invoked | `{id, tool, input}` |
| `tool_result` | Tool completed | `{id, tool, output}` |
| `message_end` | Response complete | `{usage, duration_ms}` |
| `error` | Error occurred | `{code, message}` |
| `ping` | Keep-alive | `{}` |

### 4.2 Full Stream Example

```text
: Connection established

event: message_start
data: {"message_id": "msg_abc", "role": "assistant"}

event: content_delta
data: {"delta": "I'll help you "}

event: content_delta
data: {"delta": "find information "}

event: content_delta
data: {"delta": "about that."}

event: tool_start
data: {"id": "tc_1", "tool": "search-kb", "input": {"query": "pricing"}}

: ping

event: tool_result
data: {"id": "tc_1", "tool": "search-kb", "output": {"results": [...]}}

event: content_delta
data: {"delta": "\n\nBased on my search, "}

event: content_delta
data: {"delta": "here's what I found: ..."}

event: message_end
data: {"usage": {"input_tokens": 150, "output_tokens": 200}, "duration_ms": 3500}
```

### 4.3 Error Handling

```text
event: error
data: {"code": "rate_limited", "message": "Too many requests"}

event: error
data: {"code": "context_length", "message": "Conversation too long"}

event: error
data: {"code": "tool_failed", "message": "Tool execution failed", "tool": "search-kb"}
```

---

## 5. Message Context

### 5.1 User Context

```yaml
# Pass user info for personalization
{
  "message": "Show my recent orders",
  "context": {
    "user_id": "usr_xyz",
    "metadata": {
      "name": "John Doe",
      "email": "john@example.com",
      "plan": "pro",
      "timezone": "America/New_York"
    }
  }
}
```

### 5.2 Tool Constraints

```yaml
# Restrict which tools can be used
{
  "message": "Search for pricing info",
  "tools": {
    "allow": ["search-kb", "lookup-pricing"],  # Only these tools
    "deny": []
  }
}

# Or blacklist specific tools
{
  "message": "Help me with this task",
  "tools": {
    "allow": [],                               # Empty = all allowed
    "deny": ["execute-code", "send-email"]     # Except these
  }
}
```

### 5.3 Response Format

```yaml
# JSON output
{
  "message": "Extract entities from: 'Meeting with John at 3pm'",
  "response_format": {
    "type": "json_object"
  }
}

# Structured JSON with schema
{
  "message": "Parse this order",
  "response_format": {
    "type": "json_schema",
    "schema": {
      "type": "object",
      "properties": {
        "items": { "type": "array" },
        "total": { "type": "number" }
      },
      "required": ["items", "total"]
    }
  }
}
```

---

## 6. Rate Limits

### 6.1 Chat-Specific Limits

| Plan | Messages/min | Concurrent | Max Tokens/msg |
|------|--------------|------------|----------------|
| **Free** | 20 | 5 | 4,000 |
| **Pro** | 200 | 50 | 32,000 |
| **Enterprise** | 2,000 | 500 | 128,000 |

### 6.2 Session Limits

| Plan | Max Sessions | History Length | TTL |
|------|--------------|----------------|-----|
| **Free** | 100 | 50 messages | 24h |
| **Pro** | 10,000 | 200 messages | 7d |
| **Enterprise** | Unlimited | 1,000 messages | 30d |

---

## 7. Webhooks

### 7.1 Chat Events

```yaml
# Message received
{
  "id": "evt_abc123",
  "type": "chat.message",
  "created_at": "2025-01-15T10:30:00Z",
  "data": {
    "agent_id": "agt_xxx",
    "session_id": "ses_yyy",
    "message_id": "msg_zzz",
    "role": "user",
    "content": "Hello"
  }
}

# Response sent
{
  "id": "evt_abc124",
  "type": "chat.response",
  "created_at": "2025-01-15T10:30:01Z",
  "data": {
    "agent_id": "agt_xxx",
    "session_id": "ses_yyy",
    "message_id": "msg_aaa",
    "role": "assistant",
    "content": "Hello! How can I help?",
    "usage": { ... }
  }
}
```

---

## 8. Error Responses

```yaml
# Agent unavailable
503 Service Unavailable
{
  "type": "https://api.agentstack.io/errors/unavailable",
  "title": "Agent Unavailable",
  "status": 503,
  "detail": "Agent is currently scaling up. Retry in a moment.",
  "retry_after": 5
}

# Session expired
410 Gone
{
  "type": "https://api.agentstack.io/errors/session-expired",
  "title": "Session Expired",
  "status": 410,
  "detail": "Session ses_xxx has expired"
}

# Context too long
400 Bad Request
{
  "type": "https://api.agentstack.io/errors/context-length",
  "title": "Context Too Long",
  "status": 400,
  "detail": "Conversation exceeds maximum context length"
}
```

---

## 9. Multi-Tenant Considerations

| Aspect | Implementation |
|--------|----------------|
| **Session isolation** | Sessions scoped to project |
| **History storage** | Per-project database partition |
| **User context** | Tenant-controlled user_id |
| **Quotas** | Per-project message limits |
| **Data retention** | Tenant-configurable TTL |

---

## 10. SDK Examples

### Python

```python
from agentstack import Client

client = Client(api_key="as_prj_sk_live_xxx")

# Sync chat
response = client.agents.chat(
    agent_id="agt_xxx",
    message="Hello!",
    session_id="ses_yyy"
)
print(response.message)

# Streaming
for chunk in client.agents.chat_stream(
    agent_id="agt_xxx",
    message="Tell me a story"
):
    if chunk.type == "content_delta":
        print(chunk.delta, end="")
```

### TypeScript

```typescript
import { AgentStack } from '@agentstack/sdk';

const client = new AgentStack({ apiKey: 'as_prj_sk_live_xxx' });

// Sync chat
const response = await client.agents.chat('agt_xxx', {
  message: 'Hello!',
  sessionId: 'ses_yyy'
});

// Streaming
const stream = await client.agents.chatStream('agt_xxx', {
  message: 'Tell me a story'
});

for await (const chunk of stream) {
  if (chunk.type === 'content_delta') {
    process.stdout.write(chunk.delta);
  }
}
```

---

**Previous**: [012-agents-endpoints.md](012-agents-endpoints.md)  
**Next**: [014-tools-models.md](014-tools-models.md)
