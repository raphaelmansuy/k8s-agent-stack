# 021 - Interactions API

> Unified Interface for Model and Agent Interactions with Server-Side State

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Overview

The Interactions API provides a unified interface for interacting with both LLM models and agents. Inspired by [Google's Interactions API](https://ai.google.dev/gemini-api/docs/interactions), it simplifies state management, tool orchestration, and long-running tasks.

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    Interactions API Architecture                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   Traditional Chat API              Interactions API                    │
│   ──────────────────────           ─────────────────                   │
│   • Client-side history             • Server-side state                 │
│   • Resend full context             • Reference by ID                   │
│   • Single model type               • Models + Agents unified           │
│   • Sync/Stream only                • + Background execution            │
│                                                                         │
│   ┌──────────────────────────────────────────────────────────────────┐  │
│   │                    Interaction Object                             │  │
│   │  ┌─────────────────────────────────────────────────────────────┐ │  │
│   │  │ id: "int_abc123"                                            │ │  │
│   │  │ agent_id: "agt_xyz" | model_id: "gpt-4o"                   │ │  │
│   │  │ status: "completed" | "in_progress" | "requires_action"    │ │  │
│   │  │ inputs: [Content...]                                        │ │  │
│   │  │ outputs: [Content...]                                       │ │  │
│   │  │ tool_calls: [...]                                           │ │  │
│   │  │ previous_interaction_id: "int_prev"                         │ │  │
│   │  │ usage: {input_tokens, output_tokens}                        │ │  │
│   │  └─────────────────────────────────────────────────────────────┘ │  │
│   └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Core Concepts

### 2.1 Interaction Object

An Interaction represents a complete turn in a conversation or task:

```yaml
Interaction:
  type: object
  properties:
    id:
      type: string
      description: Unique interaction ID (int_xxx)
    
    # Agent or Model (mutually exclusive)
    agent_id:
      type: string
      description: Agent to use (agt_xxx)
    model_id:
      type: string
      description: Model to use (gpt-4o, gemini-2.5-flash, etc.)
    
    # Input/Output
    inputs:
      type: array
      items:
        $ref: '#/Content'
      description: Input content (user message, tool results)
    outputs:
      type: array
      items:
        $ref: '#/Content'
      description: Model/agent outputs
    
    # State
    status:
      type: string
      enum: [pending, in_progress, completed, requires_action, failed, cancelled]
    previous_interaction_id:
      type: string
      description: ID of previous turn for context
    
    # Tool handling
    tools:
      type: array
      items:
        $ref: '#/Tool'
    tool_calls:
      type: array
      items:
        $ref: '#/ToolCall'
    
    # Execution mode
    background:
      type: boolean
      description: Run asynchronously
    store:
      type: boolean
      default: true
      description: Persist interaction for state management
    
    # Metadata
    usage:
      $ref: '#/Usage'
    created_at:
      type: string
      format: date-time
    completed_at:
      type: string
      format: date-time
```

### 2.2 Interaction Status

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    Interaction Status Flow                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│           create()                                                      │
│              │                                                          │
│              ▼                                                          │
│         ┌─────────┐                                                     │
│         │ pending │                                                     │
│         └────┬────┘                                                     │
│              │                                                          │
│              ▼                                                          │
│      ┌─────────────┐                                                    │
│      │ in_progress │◄───────────────────┐                              │
│      └──────┬──────┘                    │                              │
│             │                           │                              │
│    ┌────────┼────────┐                  │                              │
│    │        │        │                  │                              │
│    ▼        ▼        ▼                  │                              │
│ ┌─────┐ ┌────────┐ ┌─────────────────┐ │                              │
│ │done │ │requires│ │     failed      │ │                              │
│ │     │ │ action │ └─────────────────┘ │                              │
│ └─────┘ └───┬────┘                      │                              │
│             │                           │                              │
│             │  provide_action()         │                              │
│             └───────────────────────────┘                              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

| Status | Description |
|--------|-------------|
| `pending` | Created, not yet processing |
| `in_progress` | Currently executing |
| `completed` | Successfully finished |
| `requires_action` | Waiting for user input (tool confirmation, etc.) |
| `failed` | Error occurred |
| `cancelled` | User cancelled |

---

## 3. API Endpoints

### 3.1 Endpoint Summary

| Operation | Method | Endpoint | Description |
|-----------|--------|----------|-------------|
| Create | POST | `/v1/interactions` | Create and execute interaction |
| Get | GET | `/v1/interactions/{id}` | Retrieve interaction |
| List | GET | `/v1/interactions` | List interactions |
| Cancel | POST | `/v1/interactions/{id}/cancel` | Cancel running interaction |
| Delete | DELETE | `/v1/interactions/{id}` | Delete stored interaction |

### 3.2 Create Interaction

```yaml
POST /v1/interactions
Content-Type: application/json

Request:
{
  # Target (one of agent_id or model_id required)
  "agent_id": "agt_customer_support",
  # OR
  "model_id": "gpt-4o",
  
  # Input content (required)
  "input": [
    {
      "role": "user",
      "parts": [
        {"type": "text", "text": "What's the weather in Tokyo?"}
      ]
    }
  ],
  
  # Optional: Continue conversation
  "previous_interaction_id": "int_abc123",
  
  # Optional: Tools
  "tools": [
    {
      "type": "function",
      "name": "get_weather",
      "description": "Get weather for a location",
      "parameters": {
        "type": "object",
        "properties": {
          "location": {"type": "string"}
        },
        "required": ["location"]
      }
    }
  ],
  
  # Optional: Built-in tools
  "builtin_tools": ["google_search", "code_execution"],
  
  # Optional: Configuration
  "config": {
    "temperature": 0.7,
    "max_output_tokens": 1000,
    "response_modalities": ["text", "image"]
  },
  
  # Optional: Execution mode
  "background": false,
  "store": true,
  "stream": false
}

Response: 200 OK
{
  "id": "int_xyz789",
  "agent_id": "agt_customer_support",
  "status": "completed",
  "inputs": [...],
  "outputs": [
    {
      "role": "assistant",
      "parts": [
        {"type": "text", "text": "The weather in Tokyo is 22°C and sunny."}
      ]
    }
  ],
  "tool_calls": [
    {
      "id": "tc_abc",
      "name": "get_weather",
      "arguments": {"location": "Tokyo"},
      "result": {"temperature": 22, "condition": "sunny"}
    }
  ],
  "usage": {
    "input_tokens": 45,
    "output_tokens": 28
  },
  "created_at": "2025-01-15T10:30:00Z",
  "completed_at": "2025-01-15T10:30:02Z"
}
```

### 3.3 Simplified Input Formats

```yaml
# Format 1: Simple string
{
  "agent_id": "agt_xxx",
  "input": "What's the weather?"
}

# Format 2: Array of parts
{
  "agent_id": "agt_xxx",
  "input": [
    {"type": "text", "text": "Describe this image"},
    {"type": "image", "data": "...", "mime_type": "image/png"}
  ]
}

# Format 3: Full content structure
{
  "agent_id": "agt_xxx",
  "input": [
    {
      "role": "user",
      "parts": [...]
    }
  ]
}
```

---

## 4. Stateful Conversations

### 4.1 Server-Side State

Using `previous_interaction_id` chains conversations without resending history:

```yaml
# Turn 1: Initial question
POST /v1/interactions
{
  "agent_id": "agt_support",
  "input": "Hi, my name is Alice."
}

Response:
{
  "id": "int_001",
  "outputs": [{"role": "assistant", "parts": [{"type": "text", "text": "Hello Alice! How can I help you today?"}]}],
  ...
}

# Turn 2: Follow-up (reference previous)
POST /v1/interactions
{
  "agent_id": "agt_support",
  "input": "What is my name?",
  "previous_interaction_id": "int_001"
}

Response:
{
  "id": "int_002",
  "previous_interaction_id": "int_001",
  "outputs": [{"role": "assistant", "parts": [{"type": "text", "text": "Your name is Alice."}]}],
  ...
}
```

### 4.2 Retrieve Conversation History

```yaml
GET /v1/interactions/{id}?include_history=true

Response:
{
  "id": "int_002",
  "history": [
    {
      "id": "int_001",
      "inputs": [...],
      "outputs": [...]
    },
    {
      "id": "int_002",
      "inputs": [...],
      "outputs": [...]
    }
  ]
}
```

### 4.3 Client-Side State (Stateless Mode)

If you prefer to manage state client-side:

```yaml
POST /v1/interactions
{
  "agent_id": "agt_support",
  "input": [
    {"role": "user", "parts": [{"type": "text", "text": "Hi, I'm Alice."}]},
    {"role": "assistant", "parts": [{"type": "text", "text": "Hello Alice!"}]},
    {"role": "user", "parts": [{"type": "text", "text": "What is my name?"}]}
  ],
  "store": false  # Don't persist this interaction
}
```

---

## 5. Background Execution

### 5.1 Long-Running Tasks

For tasks that take time (research, complex reasoning):

```yaml
POST /v1/interactions
{
  "agent_id": "agt_researcher",
  "input": "Research the history of quantum computing",
  "background": true
}

Response: 202 Accepted
{
  "id": "int_research_001",
  "status": "in_progress",
  "estimated_duration_seconds": 120
}

# Poll for results
GET /v1/interactions/int_research_001

Response (in progress):
{
  "id": "int_research_001",
  "status": "in_progress",
  "progress": {
    "step": "Gathering sources",
    "percent": 35
  }
}

Response (completed):
{
  "id": "int_research_001",
  "status": "completed",
  "outputs": [...],
  "completed_at": "2025-01-15T10:32:00Z"
}
```

### 5.2 Webhooks for Background Tasks

```yaml
POST /v1/interactions
{
  "agent_id": "agt_researcher",
  "input": "Research quantum computing",
  "background": true,
  "webhook_url": "https://myapp.com/webhooks/interaction"
}

# Webhook payload on completion
POST https://myapp.com/webhooks/interaction
{
  "event": "interaction.completed",
  "interaction": {
    "id": "int_research_001",
    "status": "completed",
    "outputs": [...]
  }
}
```

---

## 6. Function Calling

### 6.1 Automatic Tool Execution

AgentStack can automatically execute tools:

```yaml
POST /v1/interactions
{
  "agent_id": "agt_assistant",
  "input": "What's the weather in Paris?",
  "tools": [
    {
      "type": "function",
      "name": "get_weather",
      "description": "Get current weather",
      "parameters": {...}
    }
  ],
  "auto_execute_tools": true  # Default
}

Response:
{
  "id": "int_xxx",
  "status": "completed",
  "outputs": [{"parts": [{"type": "text", "text": "The weather in Paris is 18°C and cloudy."}]}],
  "tool_calls": [
    {
      "id": "tc_001",
      "name": "get_weather",
      "arguments": {"location": "Paris"},
      "result": {"temp": 18, "condition": "cloudy"},
      "duration_ms": 150
    }
  ]
}
```

### 6.2 Manual Tool Handling

For tools requiring user confirmation or custom execution:

```yaml
POST /v1/interactions
{
  "agent_id": "agt_assistant",
  "input": "Book a flight to Tokyo",
  "tools": [{"type": "function", "name": "book_flight", ...}],
  "auto_execute_tools": false
}

Response:
{
  "id": "int_xxx",
  "status": "requires_action",
  "required_action": {
    "type": "tool_calls",
    "tool_calls": [
      {
        "id": "tc_001",
        "name": "book_flight",
        "arguments": {"destination": "Tokyo", "date": "2025-02-01"}
      }
    ]
  }
}

# Client executes tool and provides result
POST /v1/interactions
{
  "previous_interaction_id": "int_xxx",
  "input": [
    {
      "role": "tool",
      "parts": [
        {
          "type": "function_result",
          "call_id": "tc_001",
          "name": "book_flight",
          "result": {"confirmation": "FL123", "status": "booked"}
        }
      ]
    }
  ]
}
```

### 6.3 Built-in Tools

```yaml
POST /v1/interactions
{
  "model_id": "gemini-2.5-flash",
  "input": "Who won the Super Bowl in 2024?",
  "builtin_tools": ["google_search"]
}

POST /v1/interactions
{
  "model_id": "gemini-2.5-flash",
  "input": "Calculate the 50th Fibonacci number",
  "builtin_tools": ["code_execution"]
}
```

| Built-in Tool | Description |
|---------------|-------------|
| `google_search` | Web search grounding |
| `code_execution` | Safe code execution |
| `url_context` | Fetch and analyze URLs |

---

## 7. Streaming

### 7.1 Stream Response

```yaml
POST /v1/interactions
Content-Type: application/json
Accept: text/event-stream

{
  "agent_id": "agt_writer",
  "input": "Write a short story",
  "stream": true
}

Response:
event: interaction.started
data: {"id": "int_xxx", "status": "in_progress"}

event: content.delta
data: {"type": "text", "delta": "Once upon a time"}

event: content.delta
data: {"type": "text", "delta": ", in a faraway land"}

event: content.delta
data: {"type": "text", "delta": "..."}

event: tool_call.start
data: {"id": "tc_001", "name": "search_lore"}

event: tool_call.end
data: {"id": "tc_001", "result": {...}}

event: interaction.completed
data: {"id": "int_xxx", "status": "completed", "usage": {...}}
```

### 7.2 Streaming Event Types

| Event | Description |
|-------|-------------|
| `interaction.started` | Interaction began processing |
| `content.delta` | Incremental content (text, thought) |
| `content.part` | Complete part (image, data) |
| `tool_call.start` | Tool invocation started |
| `tool_call.args` | Streaming tool arguments |
| `tool_call.end` | Tool completed |
| `interaction.completed` | Interaction finished |
| `interaction.error` | Error occurred |

---

## 8. Multimodal Interactions

### 8.1 Image Understanding

```yaml
POST /v1/interactions
{
  "model_id": "gpt-4o",
  "input": [
    {"type": "text", "text": "What's in this image?"},
    {"type": "image", "data": "...", "mime_type": "image/png"}
  ]
}
```

### 8.2 Image Generation

```yaml
POST /v1/interactions
{
  "model_id": "gemini-2.5-flash",
  "input": "Generate an image of a sunset over mountains",
  "config": {
    "response_modalities": ["image"]
  }
}

Response:
{
  "outputs": [
    {
      "role": "assistant",
      "parts": [
        {"type": "image", "data": "...", "mime_type": "image/png"}
      ]
    }
  ]
}
```

### 8.3 Audio/Video Understanding

```yaml
POST /v1/interactions
{
  "model_id": "gemini-2.5-flash",
  "input": [
    {"type": "text", "text": "Transcribe this audio"},
    {"type": "audio", "uri": "gs://bucket/audio.mp3", "mime_type": "audio/mpeg"}
  ]
}

POST /v1/interactions
{
  "model_id": "gemini-2.5-flash",
  "input": [
    {"type": "text", "text": "Summarize this video"},
    {"type": "video", "uri": "gs://bucket/video.mp4", "mime_type": "video/mp4"}
  ]
}
```

---

## 9. Structured Output

### 9.1 JSON Schema Response

```yaml
POST /v1/interactions
{
  "model_id": "gpt-4o",
  "input": "Extract: John Smith, age 30, works at Acme Corp",
  "config": {
    "response_format": {
      "type": "json_schema",
      "json_schema": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "age": {"type": "integer"},
          "company": {"type": "string"}
        },
        "required": ["name", "age", "company"]
      }
    }
  }
}

Response:
{
  "outputs": [
    {
      "parts": [
        {
          "type": "data",
          "json_data": {
            "name": "John Smith",
            "age": 30,
            "company": "Acme Corp"
          }
        }
      ]
    }
  ]
}
```

---

## 10. Error Handling

### 10.1 Error Response

```yaml
Response: 400 Bad Request
{
  "error": {
    "code": "invalid_request",
    "message": "Either agent_id or model_id is required",
    "details": {
      "missing_field": "agent_id or model_id"
    }
  }
}

Response: 404 Not Found
{
  "error": {
    "code": "interaction_not_found",
    "message": "Interaction int_xxx not found"
  }
}

Response: 429 Too Many Requests
{
  "error": {
    "code": "rate_limited",
    "message": "Rate limit exceeded",
    "retry_after": 30
  }
}
```

### 10.2 Interaction Errors

```yaml
# Failed interaction
{
  "id": "int_xxx",
  "status": "failed",
  "error": {
    "code": "model_overloaded",
    "message": "The model is currently overloaded",
    "recoverable": true
  }
}
```

---

## 11. Relationship to Other APIs

### 11.1 Comparison

| Feature | Chat API | Interactions API | A2A API |
|---------|----------|------------------|---------|
| State | Client-side | Server-side | Server-side |
| Multi-turn | Session ID | Previous ID | Context ID |
| Background | ❌ | ✅ | ✅ |
| Tool handling | Auto only | Auto + Manual | Auto + Manual |
| Streaming | SSE | SSE | SSE |
| Model-agnostic | ✅ | ✅ | ✅ |
| Agent interop | ❌ | ❌ | ✅ |

### 11.2 When to Use Each

| Use Case | Recommended API |
|----------|-----------------|
| Simple chatbot | Chat API |
| Complex multi-turn with state | Interactions API |
| Long-running research tasks | Interactions API (background) |
| Agent-to-agent communication | A2A API |
| Third-party agent integration | A2A API |

---

## 12. References

- [Google Interactions API](https://ai.google.dev/gemini-api/docs/interactions)
- [Universal Content Model](020-universal-content-model.md)
- [Chat Sessions API](013-chat-sessions.md)
- [A2A Protocol](017-a2a-protocol.md)

---

*End of Interactions API Specification*
