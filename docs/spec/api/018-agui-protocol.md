# 018 - AG-UI Protocol Integration

> Agent-User Interaction Protocol for Real-Time Streaming

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Overview

AG-UI (Agent-User Interaction) is an open-source protocol for real-time agent-to-frontend communication. AgentStack implements AG-UI to provide:

- **Standardized Streaming**: 16 event types for comprehensive interaction tracking
- **State Synchronization**: Bi-directional state sync between agent and UI
- **Tool Visibility**: Detailed tool call tracking with progressive args
- **Human-in-the-Loop**: Interrupt and resume capabilities
- **Framework Compatibility**: Works with LangGraph, CrewAI, Google ADK, and more

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    AG-UI Event Flow                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   Frontend                              AgentStack                      │
│      │                                      │                           │
│      │  POST /agui/v1/stream                │                           │
│      │─────────────────────────────────────▶│                           │
│      │                                      │                           │
│      │  event: RunStarted                   │                           │
│      │◀─────────────────────────────────────│                           │
│      │                                      │                           │
│      │  event: TextMessageStart             │                           │
│      │◀─────────────────────────────────────│                           │
│      │                                      │                           │
│      │  event: TextMessageContent           │                           │
│      │◀─────────────────────────────────────│ (streaming)               │
│      │                                      │                           │
│      │  event: ToolCallStart                │                           │
│      │◀─────────────────────────────────────│                           │
│      │                                      │                           │
│      │  event: ToolCallEnd                  │                           │
│      │◀─────────────────────────────────────│                           │
│      │                                      │                           │
│      │  event: RunFinished                  │                           │
│      │◀─────────────────────────────────────│                           │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. AG-UI Endpoints

### 2.1 Endpoint Summary

| Operation | Method | Endpoint | Description |
|-----------|--------|----------|-------------|
| Stream Chat | POST | `/agui/v1/stream` | Start AG-UI event stream |
| State Snapshot | GET | `/agui/v1/state/{runId}` | Get current state |
| Send Action | POST | `/agui/v1/actions` | User actions (interrupts) |

### 2.2 Protocol Headers

```yaml
# Required Headers
Accept: text/event-stream
Content-Type: application/json
Authorization: Bearer <token>

# Optional Headers
X-AG-UI-Version: 1.0              # Protocol version
X-AgentStack-SSE-Format: agui     # Force AG-UI format
X-Project-ID: prj_xxx             # Tenant context
```

---

## 3. Event Types Reference

### 3.1 Event Categories

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    AG-UI Event Categories                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   LIFECYCLE                TEXT MESSAGES           TOOL CALLS           │
│   ┌─────────────┐         ┌─────────────┐         ┌─────────────┐      │
│   │ RunStarted  │         │ TextMessage │         │ ToolCall    │      │
│   │ RunFinished │         │   Start     │         │   Start     │      │
│   │ RunError    │         │   Content   │         │   Args      │      │
│   │ StepStarted │         │   End       │         │   End       │      │
│   │ StepFinished│         │   Chunk     │         │   Chunk     │      │
│   └─────────────┘         └─────────────┘         └─────────────┘      │
│                                                                         │
│   STATE                    ACTIVITY                 SPECIAL             │
│   ┌─────────────┐         ┌─────────────┐         ┌─────────────┐      │
│   │ State       │         │ Activity    │         │ Raw         │      │
│   │   Snapshot  │         │   Snapshot  │         │ Custom      │      │
│   │   Delta     │         │   Delta     │         │             │      │
│   │ Messages    │         │             │         │             │      │
│   │   Snapshot  │         │             │         │             │      │
│   └─────────────┘         └─────────────┘         └─────────────┘      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Complete Event Reference

| Event Type | Category | Description |
|------------|----------|-------------|
| `RunStarted` | Lifecycle | Agent run initiated |
| `RunFinished` | Lifecycle | Agent run completed |
| `RunError` | Lifecycle | Error occurred |
| `StepStarted` | Lifecycle | Processing step begun |
| `StepFinished` | Lifecycle | Processing step completed |
| `TextMessageStart` | Text | Message response begun |
| `TextMessageContent` | Text | Text content delta |
| `TextMessageEnd` | Text | Message completed |
| `TextMessageChunk` | Text | Combined start+content+end |
| `ToolCallStart` | Tool | Tool invocation begun |
| `ToolCallArgs` | Tool | Tool arguments (streaming) |
| `ToolCallEnd` | Tool | Tool execution complete |
| `ToolCallResult` | Tool | Tool result (alias) |
| `ToolCallChunk` | Tool | Combined tool event |
| `StateSnapshot` | State | Full state sync |
| `StateDelta` | State | Incremental state update |
| `MessagesSnapshot` | State | Message history sync |
| `ActivitySnapshot` | Activity | Agent activity indicator |
| `ActivityDelta` | Activity | Activity state change |
| `Raw` | Special | Raw data passthrough |
| `Custom` | Special | Extension events |

---

## 4. Event Payloads

### 4.1 Lifecycle Events

```yaml
# RunStarted
event: RunStarted
data: {
  "type": "RunStarted",
  "runId": "run_abc123",
  "agentId": "agt_xyz",
  "sessionId": "ses_def456",
  "timestamp": "2025-01-15T10:30:00.000Z"
}

# RunFinished
event: RunFinished
data: {
  "type": "RunFinished",
  "runId": "run_abc123",
  "finishReason": "completed",    # completed | cancelled | error | interrupted
  "timestamp": "2025-01-15T10:30:05.000Z"
}

# RunError
event: RunError
data: {
  "type": "RunError",
  "runId": "run_abc123",
  "error": {
    "code": "TOOL_EXECUTION_ERROR",
    "message": "Failed to execute weather-api tool",
    "recoverable": true
  },
  "timestamp": "2025-01-15T10:30:03.000Z"
}

# StepStarted / StepFinished
event: StepStarted
data: {
  "type": "StepStarted",
  "runId": "run_abc123",
  "stepId": "step_1",
  "stepName": "planning",
  "timestamp": "2025-01-15T10:30:01.000Z"
}
```

### 4.2 Text Message Events

```yaml
# TextMessageStart
event: TextMessageStart
data: {
  "type": "TextMessageStart",
  "messageId": "msg_xyz",
  "role": "assistant",
  "runId": "run_abc123"
}

# TextMessageContent
event: TextMessageContent
data: {
  "type": "TextMessageContent",
  "messageId": "msg_xyz",
  "delta": "Let me check the weather for you..."
}

# TextMessageEnd
event: TextMessageEnd
data: {
  "type": "TextMessageEnd",
  "messageId": "msg_xyz"
}
```

### 4.3 Tool Call Events

```yaml
# ToolCallStart
event: ToolCallStart
data: {
  "type": "ToolCallStart",
  "toolCallId": "tc_abc",
  "toolName": "weather-api",
  "runId": "run_abc123",
  "messageId": "msg_xyz"
}

# ToolCallArgs (streaming arguments)
event: ToolCallArgs
data: {
  "type": "ToolCallArgs",
  "toolCallId": "tc_abc",
  "delta": "{\"city\": \"Tokyo\""
}

event: ToolCallArgs
data: {
  "type": "ToolCallArgs",
  "toolCallId": "tc_abc",
  "delta": ", \"units\": \"celsius\"}"
}

# ToolCallEnd
event: ToolCallEnd
data: {
  "type": "ToolCallEnd",
  "toolCallId": "tc_abc",
  "result": {
    "temperature": 22,
    "condition": "sunny",
    "humidity": 65
  },
  "duration_ms": 150
}
```

### 4.4 State Events

```yaml
# StateSnapshot (full state sync)
event: StateSnapshot
data: {
  "type": "StateSnapshot",
  "runId": "run_abc123",
  "state": {
    "messages": [
      {"id": "msg_1", "role": "user", "content": "What's the weather?"},
      {"id": "msg_2", "role": "assistant", "content": "Checking..."}
    ],
    "toolCalls": [
      {"id": "tc_abc", "tool": "weather-api", "status": "running"}
    ],
    "metadata": {
      "agentId": "agt_xyz",
      "sessionId": "ses_def"
    }
  }
}

# StateDelta (incremental update)
event: StateDelta
data: {
  "type": "StateDelta",
  "runId": "run_abc123",
  "delta": {
    "op": "add",
    "path": "/messages/-",
    "value": {"id": "msg_3", "role": "assistant", "content": "Done!"}
  }
}
```

---

## 5. Stream Request Format

### 5.1 Start Stream

```yaml
POST /agui/v1/stream
Content-Type: application/json
Accept: text/event-stream
Authorization: Bearer <token>
X-Project-ID: prj_abc123

Request:
{
  "agentId": "agt_xyz",
  "message": "What's the weather in Tokyo?",
  "sessionId": "ses_def456",           # Optional
  "context": {
    "userId": "usr_123",
    "metadata": {
      "timezone": "Asia/Tokyo"
    }
  },
  "config": {
    "includeState": true,              # Include StateSnapshot events
    "includeToolArgs": true,           # Stream tool args progressively
    "includeSteps": false              # Include step events
  }
}
```

### 5.2 Complete Stream Example

```yaml
# Full stream for: "What's the weather in Tokyo?"

event: RunStarted
data: {"type": "RunStarted", "runId": "run_abc", "agentId": "agt_xyz", "timestamp": "2025-01-15T10:30:00.000Z"}

event: TextMessageStart
data: {"type": "TextMessageStart", "messageId": "msg_1", "role": "assistant", "runId": "run_abc"}

event: TextMessageContent
data: {"type": "TextMessageContent", "messageId": "msg_1", "delta": "Let me check "}

event: TextMessageContent
data: {"type": "TextMessageContent", "messageId": "msg_1", "delta": "the weather in Tokyo."}

event: TextMessageEnd
data: {"type": "TextMessageEnd", "messageId": "msg_1"}

event: ToolCallStart
data: {"type": "ToolCallStart", "toolCallId": "tc_1", "toolName": "weather-api", "runId": "run_abc"}

event: ToolCallArgs
data: {"type": "ToolCallArgs", "toolCallId": "tc_1", "delta": "{\"city\": \"Tokyo\"}"}

event: ToolCallEnd
data: {"type": "ToolCallEnd", "toolCallId": "tc_1", "result": {"temp": 22, "condition": "sunny"}, "duration_ms": 150}

event: TextMessageStart
data: {"type": "TextMessageStart", "messageId": "msg_2", "role": "assistant", "runId": "run_abc"}

event: TextMessageContent
data: {"type": "TextMessageContent", "messageId": "msg_2", "delta": "The weather in Tokyo is 22°C and sunny!"}

event: TextMessageEnd
data: {"type": "TextMessageEnd", "messageId": "msg_2"}

event: RunFinished
data: {"type": "RunFinished", "runId": "run_abc", "finishReason": "completed", "timestamp": "2025-01-15T10:30:05.000Z"}
```

---

## 6. Migration from Legacy SSE

### 6.1 Event Name Mapping

| Legacy Event | AG-UI Event | Notes |
|--------------|-------------|-------|
| `message_start` | `TextMessageStart` | Role moved to data |
| `content_delta` | `TextMessageContent` | Delta is same |
| `message_end` | `TextMessageEnd` | Usage in RunFinished |
| `tool_start` | `ToolCallStart` | Added toolCallId |
| `tool_result` | `ToolCallEnd` | Result in data |
| `error` | `RunError` | Structured error |
| `ping` | `ping` | Unchanged |
| - | `RunStarted` | New event |
| - | `RunFinished` | New event |
| - | `ToolCallArgs` | New event |
| - | `StateSnapshot` | New event |

### 6.2 Format Selection Header

```yaml
# Request legacy format (default for backward compatibility)
X-AgentStack-SSE-Format: legacy
→ Uses: message_start, content_delta, message_end, tool_start, tool_result

# Request AG-UI format
X-AgentStack-SSE-Format: agui
→ Uses: RunStarted, TextMessageStart, TextMessageContent, ToolCallStart, etc.

# Or use dedicated AG-UI endpoint (always AG-UI format)
POST /agui/v1/stream
→ Always uses AG-UI event format
```

### 6.3 Dual Format Response

AgentStack can emit both formats on the existing chat endpoint:

```yaml
POST /v1/agents/{agentId}/chat/stream
Accept: text/event-stream
X-AgentStack-SSE-Format: agui

# Response uses AG-UI event types
event: RunStarted
data: {...}

event: TextMessageStart
data: {...}
```

---

## 7. State Management

### 7.1 State Snapshot Request

```yaml
GET /agui/v1/state/{runId}
Authorization: Bearer <token>

Response: 200 OK
{
  "runId": "run_abc123",
  "status": "running",
  "messages": [
    {
      "id": "msg_1",
      "role": "user",
      "content": "What's the weather?",
      "timestamp": "2025-01-15T10:30:00.000Z"
    },
    {
      "id": "msg_2",
      "role": "assistant",
      "content": "Let me check...",
      "timestamp": "2025-01-15T10:30:01.000Z"
    }
  ],
  "toolCalls": [
    {
      "id": "tc_1",
      "tool": "weather-api",
      "status": "completed",
      "args": {"city": "Tokyo"},
      "result": {"temp": 22}
    }
  ],
  "metadata": {
    "agentId": "agt_xyz",
    "sessionId": "ses_def",
    "startedAt": "2025-01-15T10:30:00.000Z"
  }
}
```

### 7.2 Client State Recovery

When a client reconnects:

```yaml
# 1. Request state snapshot
GET /agui/v1/state/{runId}

# 2. If run still active, subscribe to remaining events
POST /agui/v1/stream
{
  "resumeRunId": "run_abc123",    # Resume existing run
  "fromEventId": "evt_last_seen"  # Start from last event
}
```

---

## 8. User Actions

### 8.1 Interrupt Action

```yaml
POST /agui/v1/actions
Authorization: Bearer <token>

Request:
{
  "runId": "run_abc123",
  "action": "interrupt",
  "reason": "User requested stop"
}

Response: 200 OK
{
  "acknowledged": true,
  "runId": "run_abc123",
  "newStatus": "interrupted"
}

# Stream will receive:
event: RunFinished
data: {"type": "RunFinished", "runId": "run_abc123", "finishReason": "interrupted"}
```

### 8.2 Provide Input Action

```yaml
POST /agui/v1/actions
Authorization: Bearer <token>

Request:
{
  "runId": "run_abc123",
  "action": "input",
  "data": {
    "field": "confirmation",
    "value": "yes"
  }
}

Response: 200 OK
{
  "acknowledged": true,
  "runId": "run_abc123"
}

# Run continues processing with provided input
```

---

## 9. Integration with Chat API

### 9.1 Relationship to Chat Endpoints

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    Streaming Endpoint Options                            │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   Option 1: Traditional Chat API (simpler)                              │
│   POST /v1/agents/{agentId}/chat/stream                                 │
│   • Legacy SSE events by default                                        │
│   • AG-UI events with header                                            │
│   • Session-based                                                       │
│                                                                         │
│   Option 2: AG-UI Dedicated Endpoint (richer)                           │
│   POST /agui/v1/stream                                                  │
│   • Always AG-UI events                                                 │
│   • State management                                                    │
│   • User actions                                                        │
│   • Resume capability                                                   │
│                                                                         │
│   Option 3: A2A Streaming (interop)                                     │
│   POST /a2a/v1/message:stream                                           │
│   • A2A event format                                                    │
│   • Task lifecycle                                                      │
│   • Cross-platform                                                      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 9.2 Recommended Usage

| Use Case | Endpoint | Format |
|----------|----------|--------|
| Simple chat widget | `/v1/agents/{id}/chat/stream` | Legacy |
| Rich frontend app | `/agui/v1/stream` | AG-UI |
| Agent-to-agent | `/a2a/v1/message:stream` | A2A |
| SDK integration | Any + header | Configurable |

---

## 10. Error Handling

### 10.1 Stream Errors

```yaml
# Error event in stream
event: RunError
data: {
  "type": "RunError",
  "runId": "run_abc123",
  "error": {
    "code": "MODEL_RATE_LIMITED",
    "message": "Rate limit exceeded for model gpt-4o",
    "recoverable": true,
    "retryAfter": 5000
  }
}

# Fatal error closes stream
event: RunError
data: {
  "type": "RunError",
  "runId": "run_abc123",
  "error": {
    "code": "AGENT_NOT_FOUND",
    "message": "Agent agt_xyz does not exist",
    "recoverable": false
  }
}
```

### 10.2 Common Error Codes

| Code | Description | Recoverable |
|------|-------------|-------------|
| `MODEL_RATE_LIMITED` | LLM rate limit | Yes |
| `TOOL_EXECUTION_ERROR` | Tool failed | Sometimes |
| `CONTEXT_LENGTH_EXCEEDED` | Too many tokens | No |
| `AGENT_NOT_FOUND` | Invalid agent | No |
| `SESSION_EXPIRED` | Session timeout | No |
| `INTERNAL_ERROR` | System error | No |

---

## 11. Security Considerations

### 11.1 Authentication

All AG-UI endpoints require authentication:
- Bearer tokens (JWT)
- API Keys
- Same auth as REST API

### 11.2 Stream Security

- SSE connections use same TLS as HTTP
- Tokens validated on connection
- Session scoping enforced
- No cross-tenant data leakage

### 11.3 Client Validation

```yaml
# Only accept connections from allowed origins
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Credentials: true
```

---

## 12. References

- [AG-UI Protocol](https://docs.ag-ui.com)
- [AG-UI GitHub](https://github.com/ag-ui-protocol/ag-ui)
- [AgentStack Chat API](013-chat-sessions.md)
- [A2A Protocol Integration](017-a2a-protocol.md)

---

*End of AG-UI Protocol Specification*
