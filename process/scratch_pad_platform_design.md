# Platform Design Scratchpad

> Comprehensive analysis of A2A, AG-UI, and A2UI protocols for AgentStack integration

**Author**: AI Design Assistant  
**Date**: 2025-12-17  
**Status**: Working Document

---

## 1. Protocol Analysis Summary

### 1.1 Protocol Stack Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    COMPLETE AGENT PROTOCOL STACK                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ┌──────────────────────────────────────────────────────────────────┐  │
│   │  LAYER 5: A2UI (Agent-to-UI)                                     │  │
│   │  Declarative UI components, security-first rendering             │  │
│   │  Apache 2.0 | Google | Framework-agnostic                        │  │
│   └──────────────────────────────────────────────────────────────────┘  │
│                               │ UI Specification                        │
│   ┌──────────────────────────────────────────────────────────────────┐  │
│   │  LAYER 4: AG-UI (Agent-User Interaction)                         │  │
│   │  Real-time streaming, state sync, human-in-the-loop              │  │
│   │  MIT | 10.7k stars | LangGraph, CrewAI, ADK integrations         │  │
│   └──────────────────────────────────────────────────────────────────┘  │
│                               │ Frontend Transport                      │
│   ┌──────────────────────────────────────────────────────────────────┐  │
│   │  LAYER 3: A2A (Agent-to-Agent)                                   │  │
│   │  Agent discovery, task management, multi-agent collaboration     │  │
│   │  Apache 2.0 | Linux Foundation | JSON-RPC/gRPC/REST              │  │
│   └──────────────────────────────────────────────────────────────────┘  │
│                               │ Agent-to-Agent                          │
│   ┌──────────────────────────────────────────────────────────────────┐  │
│   │  LAYER 2: MCP (Model Context Protocol)                           │  │
│   │  Tool integration, resource access, structured I/O               │  │
│   │  Anthropic | Already integrated in kagent                        │  │
│   └──────────────────────────────────────────────────────────────────┘  │
│                               │ Tool Access                             │
│   ┌──────────────────────────────────────────────────────────────────┐  │
│   │  LAYER 1: LLM / Agent Framework                                  │  │
│   │  Google ADK, LangGraph, CrewAI, AutoGen                          │  │
│   │  Multiple frameworks supported                                    │  │
│   └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. A2A Protocol Deep Analysis

### 2.1 Core Concepts

| Concept | Description | AgentStack Mapping |
|---------|-------------|-------------------|
| **Agent Card** | JSON manifest with capabilities, skills, security | Agent CRD extension |
| **Task** | Unit of work with lifecycle states | Chat session + job tracking |
| **Message** | Communication turn with Parts | Chat message with attachments |
| **Artifact** | Task output (documents, images, data) | Response artifacts |
| **Streaming** | Real-time SSE updates | Already have SSE |
| **Push Notifications** | Webhook callbacks for async tasks | New capability needed |

### 2.2 A2A Task State Machine

```text
┌─────────────────────────────────────────────────────────────────┐
│                    A2A Task State Machine                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                       ┌──────────────┐                          │
│    SendMessage ──────▶│  SUBMITTED   │                          │
│                       └──────┬───────┘                          │
│                              │                                   │
│                              ▼                                   │
│                       ┌──────────────┐                          │
│                       │   WORKING    │◄──────────────┐          │
│                       └──────┬───────┘               │          │
│                              │                       │          │
│              ┌───────────────┼───────────────┐       │          │
│              │               │               │       │          │
│              ▼               ▼               ▼       │          │
│       ┌──────────┐   ┌──────────────┐  ┌────────────┴┐         │
│       │COMPLETED │   │INPUT_REQUIRED│  │AUTH_REQUIRED│         │
│       └──────────┘   └──────────────┘  └─────────────┘         │
│                                                                  │
│    Terminal States: COMPLETED, FAILED, CANCELLED, REJECTED      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.3 A2A Data Model Mapping

```yaml
# Current AgentStack Chat API
POST /v1/agents/{agentId}/chat
{
  "message": "...",
  "session_id": "ses_xxx"
}

# Proposed A2A-Compatible API
POST /v1/agents/{agentId}/a2a/message:send
{
  "message": {
    "messageId": "msg_uuid",
    "role": "user",
    "parts": [
      {"text": "What's the weather in Tokyo?"},
      {"file": {"mediaType": "image/png", "fileWithBytes": "..."}}
    ],
    "contextId": "ctx_abc123"   # Maps to session_id
  },
  "configuration": {
    "acceptedOutputModes": ["text/plain", "application/json"],
    "historyLength": 10
  }
}

# Response
{
  "task": {
    "id": "task_uuid",
    "contextId": "ctx_abc123",
    "status": {
      "state": "completed",
      "timestamp": "2025-01-15T10:30:00Z"
    },
    "artifacts": [{
      "artifactId": "art_1",
      "name": "Weather Report",
      "parts": [{"text": "The weather in Tokyo is 22°C and sunny"}]
    }]
  }
}
```

### 2.4 Agent Card Integration

```yaml
# AgentStack Agent Card (A2A compatible)
# Published at: /.well-known/agent-card.json

{
  "protocolVersion": "1.0",
  "name": "customer-support",
  "description": "Customer support agent for Acme Corp",
  
  "supportedInterfaces": [
    {"url": "https://customer-support.agentstack.app/a2a/v1", "protocolBinding": "HTTP+JSON"},
    {"url": "https://customer-support.agentstack.app/a2a/grpc", "protocolBinding": "GRPC"}
  ],
  
  "provider": {
    "organization": "Acme Corp",
    "url": "https://acme.com"
  },
  
  "capabilities": {
    "streaming": true,
    "pushNotifications": true,
    "extensions": []
  },
  
  "defaultInputModes": ["text/plain", "application/json"],
  "defaultOutputModes": ["text/plain", "application/json", "text/html"],
  
  "skills": [
    {
      "id": "order-lookup",
      "name": "Order Status Lookup",
      "description": "Check the status of customer orders",
      "tags": ["orders", "status", "tracking"],
      "examples": ["What's the status of order #12345?"]
    },
    {
      "id": "ticket-creation",
      "name": "Support Ticket Creation",
      "description": "Create support tickets for customer issues",
      "tags": ["support", "tickets", "issues"]
    }
  ],
  
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
  }
}
```

---

## 3. AG-UI Protocol Deep Analysis

### 3.1 Event Types Mapping

| AG-UI Event | Current SSE Event | Action |
|-------------|-------------------|--------|
| `RunStarted` | - | Add (new) |
| `RunFinished` | - | Add (new) |
| `RunError` | `error` | Already compatible |
| `StepStarted` | - | Add (new) |
| `StepFinished` | - | Add (new) |
| `TextMessageStart` | `message_start` | Rename for compatibility |
| `TextMessageContent` | `content_delta` | Rename for compatibility |
| `TextMessageEnd` | `message_end` | Rename for compatibility |
| `ToolCallStart` | `tool_start` | Rename for compatibility |
| `ToolCallArgs` | - | Add (new) |
| `ToolCallEnd` | `tool_result` | Rename for compatibility |
| `StateSnapshot` | - | Add (for UI sync) |
| `StateDelta` | - | Add (for UI sync) |

### 3.2 AG-UI Event Stream Format

```text
# Current AgentStack SSE Format
event: message_start
data: {"message_id": "msg_abc", "role": "assistant"}

event: content_delta
data: {"delta": "Let me check..."}

event: message_end
data: {"usage": {...}}

# Proposed AG-UI Compatible Format
event: RunStarted
data: {"runId": "run_abc", "timestamp": "2025-01-15T10:30:00Z"}

event: TextMessageStart
data: {"messageId": "msg_abc", "role": "assistant"}

event: TextMessageContent
data: {"messageId": "msg_abc", "delta": "Let me check..."}

event: ToolCallStart
data: {"toolCallId": "tc_1", "toolName": "weather-api"}

event: ToolCallArgs
data: {"toolCallId": "tc_1", "delta": "{\"city\": \"Tokyo\"}"}

event: ToolCallEnd
data: {"toolCallId": "tc_1", "result": {"temp": 22}}

event: TextMessageEnd
data: {"messageId": "msg_abc"}

event: RunFinished
data: {"runId": "run_abc", "finishReason": "completed"}
```

### 3.3 State Synchronization

```yaml
# State Snapshot for frontend sync
event: StateSnapshot
data: {
  "runId": "run_abc",
  "messages": [
    {"id": "msg_1", "role": "user", "content": "Hello"},
    {"id": "msg_2", "role": "assistant", "content": "Hi there!"}
  ],
  "toolCalls": [
    {"id": "tc_1", "status": "completed", "tool": "weather-api"}
  ],
  "metadata": {
    "agentId": "agt_xxx",
    "sessionId": "ses_yyy"
  }
}
```

---

## 4. A2UI Protocol Deep Analysis

### 4.1 Key Design Principles

1. **Security First**: Declarative data format, not executable code
2. **Component Catalog**: LLM generates from approved components only
3. **Framework Agnostic**: Works with React, Angular, Lit, Flutter
4. **Incremental Updates**: Supports partial UI updates via diffs

### 4.2 A2UI Integration Pattern

```text
┌─────────────────────────────────────────────────────────────────┐
│                    A2UI Rendering Flow                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Agent                    A2UI                    Frontend      │
│     │                        │                         │         │
│     │  Generate UI Spec      │                         │         │
│     │───────────────────────▶│                         │         │
│     │                        │                         │         │
│     │                        │   Send via AG-UI/A2A    │         │
│     │                        │────────────────────────▶│         │
│     │                        │                         │         │
│     │                        │                    Render from    │
│     │                        │                    Component      │
│     │                        │                    Catalog        │
│     │                        │                         │         │
│     │                        │                         ▼         │
│     │                        │                    [Rendered UI]  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 A2UI Component Example

```json
{
  "type": "a2ui/card",
  "props": {
    "title": "Order Status",
    "subtitle": "Order #12345"
  },
  "children": [
    {
      "type": "a2ui/status-badge",
      "props": {
        "status": "shipped",
        "label": "Shipped"
      }
    },
    {
      "type": "a2ui/progress",
      "props": {
        "value": 75,
        "max": 100,
        "label": "Delivery Progress"
      }
    },
    {
      "type": "a2ui/button",
      "props": {
        "action": "track",
        "label": "Track Package",
        "variant": "primary"
      }
    }
  ]
}
```

---

## 5. Unified Protocol Architecture

### 5.1 Proposed AgentStack Protocol Stack

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    AgentStack Protocol Architecture                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   EXTERNAL INTERFACES                                                   │
│   ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐          │
│   │  REST API       │ │  A2A Endpoint   │ │  AG-UI Stream   │          │
│   │  /v1/agents/... │ │  /a2a/v1/...    │ │  /agui/stream   │          │
│   └────────┬────────┘ └────────┬────────┘ └────────┬────────┘          │
│            │                   │                   │                    │
│            └───────────────────┼───────────────────┘                    │
│                                │                                        │
│   PROTOCOL ADAPTER LAYER       ▼                                        │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                    Protocol Router                               │   │
│   │  • Request normalization                                         │   │
│   │  • Protocol detection (A2A, AG-UI, REST)                        │   │
│   │  • Response transformation                                       │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                │                                        │
│   CORE PLATFORM                ▼                                        │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                    Agent Runtime                                 │   │
│   │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐               │   │
│   │  │   kagent    │ │   Session   │ │   Task      │               │   │
│   │  │   CRD       │ │   Manager   │ │   Manager   │               │   │
│   │  └─────────────┘ └─────────────┘ └─────────────┘               │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                │                                        │
│   AGENT FRAMEWORKS             ▼                                        │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  Google ADK │ LangGraph │ CrewAI │ AutoGen │ Custom             │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 5.2 Endpoint Design

```yaml
# API Endpoints Structure

# Traditional REST API (backward compatible)
/v1/agents/{agentId}/chat              # Sync chat
/v1/agents/{agentId}/chat/stream       # SSE stream
/v1/agents/{agentId}/sessions          # Session management

# A2A Protocol Endpoints (NEW)
/a2a/v1/message:send                   # A2A SendMessage
/a2a/v1/message:stream                 # A2A SendStreamingMessage
/a2a/v1/tasks/{taskId}                 # A2A GetTask
/a2a/v1/tasks                          # A2A ListTasks
/a2a/v1/tasks/{taskId}:cancel          # A2A CancelTask
/a2a/v1/tasks/{taskId}:subscribe       # A2A SubscribeToTask
/.well-known/agent-card.json           # A2A Agent Discovery

# AG-UI Protocol Endpoints (NEW)
/agui/v1/stream                        # AG-UI event stream
/agui/v1/state                         # State snapshot
/agui/v1/actions                       # User actions
```

### 5.3 Protocol Header Detection

```yaml
# Auto-detect protocol based on headers

# A2A Request Detection
Content-Type: application/a2a+json
A2A-Version: 1.0
A2A-Extensions: ...

# AG-UI Request Detection
Accept: text/event-stream
X-AG-UI-Version: 1.0

# Standard REST (default)
Content-Type: application/json
```

---

## 6. Migration Strategy

### 6.1 Backward Compatibility

```text
PHASE 1: Add A2A/AG-UI as separate endpoints
         Keep existing /v1/... API unchanged
         
PHASE 2: Add protocol adapter layer
         Route requests based on headers
         
PHASE 3: Deprecate old SSE event names
         Provide automatic transformation
         
PHASE 4: Full protocol compliance
         A2A certification ready
```

### 6.2 SSE Event Migration

```yaml
# Event Name Mapping
old_events:
  message_start:  TextMessageStart
  content_delta:  TextMessageContent
  message_end:    TextMessageEnd
  tool_start:     ToolCallStart
  tool_result:    ToolCallEnd
  error:          RunError
  ping:           ping  # Keep as-is

# Migration Mode Header
X-AgentStack-SSE-Format: agui  # Use AG-UI event names
X-AgentStack-SSE-Format: legacy  # Use old event names (default)
```

---

## 7. Key Design Decisions

### 7.1 Agent Communication: A2A vs Custom

| Aspect | A2A Protocol | Current Custom |
|--------|--------------|----------------|
| **Agent Discovery** | Agent Card standard | Ad-hoc |
| **Multi-agent** | Native support | Limited |
| **Interoperability** | Cross-platform | AgentStack only |
| **Complexity** | Higher | Lower |
| **Ecosystem** | Growing (Google, LF) | Internal |

**Decision**: Adopt A2A for agent-to-agent communication while keeping simplified REST API for client applications.

### 7.2 Streaming: AG-UI vs SSE

| Aspect | AG-UI Protocol | Current SSE |
|--------|----------------|-------------|
| **Event Types** | 16 standardized | 6 custom |
| **State Sync** | Built-in | Manual |
| **Tool Calls** | Detailed tracking | Basic |
| **Frontend SDK** | Available | Custom |
| **Adoption** | 10.7k stars | Internal |

**Decision**: Adopt AG-UI event types for streaming, maintain backward compat layer for existing clients.

### 7.3 Generative UI: A2UI

| Aspect | A2UI | Current |
|--------|------|---------|
| **UI Rendering** | Declarative spec | N/A |
| **Security** | Component catalog | N/A |
| **Rich UI** | Cards, forms, charts | Text only |

**Decision**: Add A2UI support as optional capability for rich UI rendering in compatible clients.

---

## 8. Implementation Priority

### 8.1 High Priority (Phase 1)

1. **Agent Card endpoint** - `/.well-known/agent-card.json`
2. **A2A Message endpoints** - `/a2a/v1/message:send`, `/a2a/v1/message:stream`
3. **AG-UI event types** - Rename existing SSE events
4. **Protocol detection** - Header-based routing

### 8.2 Medium Priority (Phase 2)

1. **A2A Task management** - Full task lifecycle
2. **Push notifications** - Webhook support
3. **State synchronization** - AG-UI StateSnapshot
4. **Multi-agent routing** - Agent-to-agent calls

### 8.3 Lower Priority (Phase 3)

1. **A2UI support** - Generative UI rendering
2. **A2A Extensions** - Custom protocol extensions
3. **gRPC binding** - Full A2A gRPC support
4. **Agent Card signing** - JWS signatures

---

## 9. Specification Updates Required

### 9.1 Documents to Update

| Document | Required Changes |
|----------|------------------|
| [004-api-design.md](../spec/004-api-design.md) | Add A2A/AG-UI endpoints |
| [013-chat-sessions.md](../spec/api/013-chat-sessions.md) | AG-UI event types |
| [003-agent-lifecycle.md](../spec/003-agent-lifecycle.md) | Agent Card integration |
| [002-architecture-layers.md](../spec/002-architecture-layers.md) | Protocol layer diagram |

### 9.2 New Documents Needed

| Document | Purpose |
|----------|---------|
| `017-a2a-protocol.md` | A2A implementation spec |
| `018-agui-protocol.md` | AG-UI streaming spec |
| `019-agent-discovery.md` | Agent Card specification |

---

## 10. Questions & Open Items

### 10.1 Architecture Questions

1. Should A2A be the primary API or a secondary protocol?
   - **Recommendation**: Secondary - REST for clients, A2A for agents

2. How to handle A2A Task vs Session mapping?
   - **Recommendation**: Task = short conversation, Session = persistent context

3. Should we support all 3 A2A bindings (JSON-RPC, gRPC, HTTP+JSON)?
   - **Recommendation**: Start with HTTP+JSON, add gRPC later

### 10.2 Implementation Questions

1. Where to store Agent Cards?
   - **Recommendation**: Generate from Agent CRD metadata

2. How to handle A2A authentication schemes?
   - **Recommendation**: Map to existing auth (Bearer, API Key)

3. Push notification infrastructure?
   - **Recommendation**: Use existing webhook system

---

## 11. Next Steps

```markdown
- [ ] Create detailed work plan with phases
- [ ] Clean up leftover spec files
- [ ] Update 004-api-design.md with A2A endpoints
- [ ] Create 017-a2a-protocol.md specification
- [ ] Update SSE event names in 013-chat-sessions.md
- [ ] Add Agent Card to Agent CRD spec
- [ ] Create protocol compatibility matrix
```

---

*End of Scratchpad*
