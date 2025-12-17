# 020 - Universal Content Model

> Multimodal Content Format with Provider Translation Layer

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Overview

The Universal Content Model (UCM) defines a canonical format for multimodal content that works across all LLM providers. AgentStack uses UCM as the internal representation, with provider adapters translating to/from provider-specific formats.

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    Universal Content Model Architecture                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   Client Request                  AgentStack                  Provider  │
│   (UCM Format)                    Core                        API       │
│                                                                         │
│   ┌─────────────┐              ┌──────────────┐            ┌──────────┐│
│   │ Universal   │─────────────▶│   Provider   │───────────▶│  OpenAI  ││
│   │ Part[]      │              │   Router     │            └──────────┘│
│   └─────────────┘              │              │            ┌──────────┐│
│                                │   ┌──────┐   │───────────▶│ Anthropic││
│                                │   │Adapt.│   │            └──────────┘│
│                                │   └──────┘   │            ┌──────────┐│
│                                │              │───────────▶│  Gemini  ││
│                                └──────────────┘            └──────────┘│
│                                       │                    ┌──────────┐│
│                                       └───────────────────▶│  Ollama  ││
│                                                            └──────────┘│
│                                                                         │
│   Benefits:                                                             │
│   • Model-agnostic agent development                                    │
│   • Consistent multimodal support                                       │
│   • A2A protocol compatibility                                          │
│   • Unified observability                                               │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Core Types

### 2.1 Part Types

| Type | Description | Use Case |
|------|-------------|----------|
| `text` | Plain text content | Messages, prompts |
| `image` | Image data | Vision, generation |
| `audio` | Audio data | Speech, transcription |
| `video` | Video data | Video understanding |
| `document` | Document (PDF, etc.) | Document analysis |
| `file` | Generic file reference | Any file type |
| `function_call` | Tool invocation | Function calling |
| `function_result` | Tool response | Function results |
| `data` | Structured JSON data | Structured outputs |

### 2.2 Part Schema

```yaml
Part:
  type: object
  required:
    - type
  properties:
    type:
      type: string
      enum: [text, image, audio, video, document, file, function_call, function_result, data]
    
    # Text content (type: text)
    text:
      type: string
      description: Plain text content
    
    # Media content (type: image, audio, video, document, file)
    data:
      type: string
      description: Base64-encoded binary data
    mime_type:
      type: string
      description: MIME type (e.g., image/png, audio/wav)
    uri:
      type: string
      description: Remote file URI (https://, gs://, s3://)
    file_id:
      type: string
      description: AgentStack file reference (file_xxx)
    filename:
      type: string
      description: Original filename
    
    # Function call (type: function_call)
    name:
      type: string
      description: Function/tool name
    call_id:
      type: string
      description: Unique call identifier for correlation
    arguments:
      type: object
      description: Function arguments
    
    # Function result (type: function_result)
    result:
      type: any
      description: Function execution result
    error:
      type: string
      description: Error message if function failed
    
    # Structured data (type: data)
    json_data:
      type: object
      description: Arbitrary JSON data
    schema:
      type: string
      description: JSON Schema reference for validation
```

### 2.3 Content Structure

```yaml
Content:
  type: object
  required:
    - role
    - parts
  properties:
    role:
      type: string
      enum: [user, assistant, system, tool]
      description: Message author role
    parts:
      type: array
      items:
        $ref: '#/Part'
      description: Content parts
    metadata:
      type: object
      description: Optional metadata (timestamps, IDs)
```

---

## 3. Part Examples

### 3.1 Text Part

```yaml
# Simple text
{
  "type": "text",
  "text": "What is the capital of France?"
}
```

### 3.2 Image Part

```yaml
# Inline image (base64)
{
  "type": "image",
  "data": "iVBORw0KGgoAAAANSUhEUgAAA...",
  "mime_type": "image/png"
}

# Remote image (URI)
{
  "type": "image",
  "uri": "https://example.com/photo.jpg",
  "mime_type": "image/jpeg"
}

# AgentStack file reference
{
  "type": "image",
  "file_id": "file_abc123",
  "mime_type": "image/png",
  "filename": "screenshot.png"
}
```

### 3.3 Audio Part

```yaml
# Audio content
{
  "type": "audio",
  "data": "UklGRiQAAABXQVZFZm10IBAA...",
  "mime_type": "audio/wav"
}

# URI reference
{
  "type": "audio",
  "uri": "gs://bucket/audio.mp3",
  "mime_type": "audio/mpeg"
}
```

### 3.4 Video Part

```yaml
# Video content
{
  "type": "video",
  "uri": "s3://bucket/video.mp4",
  "mime_type": "video/mp4"
}
```

### 3.5 Document Part

```yaml
# PDF document
{
  "type": "document",
  "data": "JVBERi0xLjQKJ...",
  "mime_type": "application/pdf",
  "filename": "report.pdf"
}
```

### 3.6 Function Call Part

```yaml
# Tool invocation
{
  "type": "function_call",
  "call_id": "call_abc123",
  "name": "get_weather",
  "arguments": {
    "location": "Tokyo",
    "units": "celsius"
  }
}
```

### 3.7 Function Result Part

```yaml
# Successful result
{
  "type": "function_result",
  "call_id": "call_abc123",
  "name": "get_weather",
  "result": {
    "temperature": 22,
    "condition": "sunny",
    "humidity": 65
  }
}

# Error result
{
  "type": "function_result",
  "call_id": "call_abc123",
  "name": "get_weather",
  "error": "Location not found"
}
```

### 3.8 Data Part

```yaml
# Structured JSON data
{
  "type": "data",
  "json_data": {
    "order_id": "12345",
    "items": [
      {"product": "Widget", "qty": 2}
    ],
    "total": 99.99
  },
  "schema": "https://schema.example.com/order.json"
}
```

---

## 4. Content Examples

### 4.1 Simple Text Message

```yaml
{
  "role": "user",
  "parts": [
    {"type": "text", "text": "Hello, how are you?"}
  ]
}
```

### 4.2 Multimodal Message (Text + Image)

```yaml
{
  "role": "user",
  "parts": [
    {"type": "text", "text": "What's in this image?"},
    {
      "type": "image",
      "data": "iVBORw0KGgoAAAANSUhEUgAAA...",
      "mime_type": "image/png"
    }
  ]
}
```

### 4.3 Assistant Response with Tool Call

```yaml
{
  "role": "assistant",
  "parts": [
    {"type": "text", "text": "Let me check the weather for you."},
    {
      "type": "function_call",
      "call_id": "call_xyz",
      "name": "get_weather",
      "arguments": {"location": "Paris"}
    }
  ]
}
```

### 4.4 Tool Response

```yaml
{
  "role": "tool",
  "parts": [
    {
      "type": "function_result",
      "call_id": "call_xyz",
      "name": "get_weather",
      "result": {"temperature": 18, "condition": "cloudy"}
    }
  ]
}
```

---

## 5. Provider Translation

### 5.1 Translation Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    Provider Translation Flow                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   AgentStack UCM                                Provider Format         │
│                                                                         │
│   {"type": "text",     ───OpenAI───▶    {"type": "text",               │
│    "text": "Hello"}                      "text": "Hello"}              │
│                                                                         │
│   {"type": "image",    ───OpenAI───▶    {"type": "image_url",          │
│    "data": "base64..", }                 "image_url": {"url":          │
│    "mime_type": "..."}                    "data:image/png;base64,..."}}│
│                                                                         │
│   {"type": "image",    ──Anthropic─▶    {"type": "image",              │
│    "data": "base64..",}                  "source": {"type": "base64",  │
│    "mime_type": "..."}                    "media_type": "...",         │
│                                           "data": "..."}}              │
│                                                                         │
│   {"type": "image",    ───Gemini───▶    {"inlineData": {               │
│    "data": "base64..",}                   "mimeType": "...",           │
│    "mime_type": "..."}                    "data": "..."}}              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 5.2 Adapter Interface

```typescript
interface ProviderAdapter {
  // Provider identification
  name: string;  // "openai", "anthropic", "gemini", "ollama"
  
  // Content translation
  translateToProvider(content: Content[]): ProviderRequest;
  translateFromProvider(response: ProviderResponse): Content[];
  
  // Part-level translation
  translatePart(part: Part): ProviderPart;
  translatePartFromProvider(providerPart: ProviderPart): Part;
  
  // Capability checking
  supportsPartType(type: PartType): boolean;
  supportedMimeTypes(): string[];
  
  // Configuration
  getDefaultConfig(): ProviderConfig;
}
```

### 5.3 OpenAI Adapter

```yaml
# UCM to OpenAI translation rules

text:
  input:  {"type": "text", "text": "..."}
  output: {"type": "text", "text": "..."}

image (inline):
  input:  {"type": "image", "data": "...", "mime_type": "image/png"}
  output: {"type": "image_url", "image_url": {"url": "data:image/png;base64,..."}}

image (uri):
  input:  {"type": "image", "uri": "https://..."}
  output: {"type": "image_url", "image_url": {"url": "https://..."}}

function_call:
  input:  {"type": "function_call", "call_id": "...", "name": "...", "arguments": {...}}
  output: {"type": "function", "function": {"name": "...", "arguments": "{...}"}, "id": "..."}

function_result:
  input:  {"type": "function_result", "call_id": "...", "result": {...}}
  output: (tool message with content as JSON string)

# OpenAI limitations:
# - No native audio in chat (use Whisper for transcription)
# - No native video in chat
# - Documents must be converted to text or images
```

### 5.4 Anthropic Adapter

```yaml
# UCM to Anthropic (Claude) translation rules

text:
  input:  {"type": "text", "text": "..."}
  output: {"type": "text", "text": "..."}

image (inline):
  input:  {"type": "image", "data": "...", "mime_type": "image/png"}
  output: {"type": "image", "source": {"type": "base64", "media_type": "image/png", "data": "..."}}

image (uri):
  input:  {"type": "image", "uri": "https://..."}
  output: {"type": "image", "source": {"type": "url", "url": "https://..."}}

document (PDF):
  input:  {"type": "document", "data": "...", "mime_type": "application/pdf"}
  output: {"type": "document", "source": {"type": "base64", "media_type": "application/pdf", "data": "..."}}

function_call:
  input:  {"type": "function_call", "call_id": "...", "name": "...", "arguments": {...}}
  output: {"type": "tool_use", "id": "...", "name": "...", "input": {...}}

function_result:
  input:  {"type": "function_result", "call_id": "...", "result": {...}}
  output: {"type": "tool_result", "tool_use_id": "...", "content": "..."}
```

### 5.5 Gemini Adapter

```yaml
# UCM to Google Gemini translation rules

text:
  input:  {"type": "text", "text": "..."}
  output: {"text": "..."}

image (inline):
  input:  {"type": "image", "data": "...", "mime_type": "image/png"}
  output: {"inlineData": {"mimeType": "image/png", "data": "..."}}

image (uri - GCS):
  input:  {"type": "image", "uri": "gs://..."}
  output: {"fileData": {"fileUri": "gs://...", "mimeType": "image/png"}}

audio:
  input:  {"type": "audio", "data": "...", "mime_type": "audio/wav"}
  output: {"inlineData": {"mimeType": "audio/wav", "data": "..."}}

video:
  input:  {"type": "video", "uri": "gs://..."}
  output: {"fileData": {"fileUri": "gs://...", "mimeType": "video/mp4"}}

document:
  input:  {"type": "document", "data": "...", "mime_type": "application/pdf"}
  output: {"inlineData": {"mimeType": "application/pdf", "data": "..."}}

function_call:
  input:  {"type": "function_call", "call_id": "...", "name": "...", "arguments": {...}}
  output: {"functionCall": {"name": "...", "args": {...}}}

function_result:
  input:  {"type": "function_result", "call_id": "...", "name": "...", "result": {...}}
  output: {"functionResponse": {"name": "...", "response": {...}}}
```

---

## 6. API Integration

### 6.1 Chat Request with Multimodal Content

```yaml
POST /v1/agents/{agentId}/chat
Content-Type: application/json

Request:
{
  "messages": [
    {
      "role": "user",
      "parts": [
        {"type": "text", "text": "What's in this image?"},
        {
          "type": "image",
          "data": "iVBORw0KGgoAAAANSUhEUgAAA...",
          "mime_type": "image/png"
        }
      ]
    }
  ],
  "session_id": "ses_abc123"
}

Response: 200 OK
{
  "id": "msg_xyz",
  "session_id": "ses_abc123",
  "content": {
    "role": "assistant",
    "parts": [
      {
        "type": "text",
        "text": "This image shows a sunset over the ocean with vibrant orange and purple colors."
      }
    ]
  },
  "usage": {
    "input_tokens": 258,
    "output_tokens": 24
  }
}
```

### 6.2 Streaming with Multimodal Output

```yaml
POST /v1/agents/{agentId}/chat/stream
Accept: text/event-stream

Request:
{
  "messages": [
    {
      "role": "user",
      "parts": [
        {"type": "text", "text": "Generate an image of a cat"}
      ]
    }
  ],
  "response_modalities": ["text", "image"]
}

Response:
event: TextMessageStart
data: {"messageId": "msg_1", "role": "assistant"}

event: TextMessageContent
data: {"messageId": "msg_1", "delta": "Here's a cute cat image:"}

event: TextMessageEnd
data: {"messageId": "msg_1"}

event: ImagePart
data: {"messageId": "msg_1", "part": {"type": "image", "data": "...", "mime_type": "image/png"}}

event: RunFinished
data: {"runId": "run_abc", "usage": {...}}
```

### 6.3 Legacy Compatibility

For backward compatibility, simple string messages are auto-converted:

```yaml
# Legacy format (still supported)
{
  "message": "Hello, world!"
}

# Automatically converted to UCM:
{
  "messages": [
    {
      "role": "user",
      "parts": [
        {"type": "text", "text": "Hello, world!"}
      ]
    }
  ]
}
```

---

## 7. A2A Protocol Alignment

### 7.1 UCM ↔ A2A Part Mapping

| UCM Part Type | A2A Part Type | Notes |
|---------------|---------------|-------|
| `text` | `TextPart` | Direct mapping |
| `image` | `FilePart` / `DataPart` | Based on inline vs URI |
| `audio` | `FilePart` / `DataPart` | Based on inline vs URI |
| `video` | `FilePart` | Usually URI reference |
| `document` | `FilePart` / `DataPart` | PDF, etc. |
| `file` | `FilePart` | Generic file |
| `function_call` | (Task request) | Maps to A2A task |
| `function_result` | (Artifact) | Maps to A2A artifact |
| `data` | `DataPart` | JSON data |

### 7.2 A2A Message Integration

```yaml
# UCM Content → A2A Message
{
  "role": "user",
  "parts": [
    {"type": "text", "text": "Analyze this image"},
    {"type": "image", "data": "...", "mime_type": "image/png"}
  ]
}

# Translates to A2A Message:
{
  "messageId": "msg_uuid",
  "role": "user",
  "parts": [
    {"text": "Analyze this image"},
    {
      "data": {
        "mediaType": "image/png",
        "data": "..."
      }
    }
  ]
}
```

---

## 8. File Management

### 8.1 File Upload

```yaml
POST /v1/files
Content-Type: multipart/form-data

Request:
  file: (binary)
  purpose: "chat"
  metadata: {"description": "Product photo"}

Response: 201 Created
{
  "id": "file_abc123",
  "filename": "product.jpg",
  "mime_type": "image/jpeg",
  "size_bytes": 245632,
  "purpose": "chat",
  "status": "ready",
  "created_at": "2025-01-15T10:30:00Z",
  "expires_at": "2025-01-22T10:30:00Z"
}
```

### 8.2 Using File References

```yaml
# Reference uploaded file in message
{
  "role": "user",
  "parts": [
    {"type": "text", "text": "Describe this product"},
    {
      "type": "image",
      "file_id": "file_abc123"
    }
  ]
}

# AgentStack resolves file_id to actual content before sending to provider
```

---

## 9. Capability Matrix

### 9.1 Provider Capabilities

| Capability | OpenAI | Anthropic | Gemini | Ollama |
|------------|--------|-----------|--------|--------|
| Text | ✅ | ✅ | ✅ | ✅ |
| Image Input | ✅ (GPT-4V) | ✅ | ✅ | ✅ (LLaVA) |
| Image Output | ✅ (DALL-E) | ❌ | ✅ (Imagen) | ❌ |
| Audio Input | ✅ (Whisper) | ❌ | ✅ | ❌ |
| Audio Output | ✅ (TTS) | ❌ | ❌ | ❌ |
| Video Input | ❌ | ❌ | ✅ | ❌ |
| PDF Input | ❌ | ✅ | ✅ | ❌ |
| Function Calling | ✅ | ✅ | ✅ | ✅ |
| Streaming | ✅ | ✅ | ✅ | ✅ |

### 9.2 Automatic Fallbacks

When a part type is not supported by a provider:

```yaml
# Unsupported audio → transcribe first
{
  "type": "audio",
  "data": "...",
  "mime_type": "audio/wav"
}

# For providers without audio support, AgentStack can:
# 1. Transcribe via Whisper → convert to text part
# 2. Return error with supported alternatives
# 3. Use alternative provider for that content type
```

---

## 10. References

- [Google GenAI SDK](https://github.com/googleapis/python-genai)
- [Google Interactions API](https://ai.google.dev/gemini-api/docs/interactions)
- [A2A Protocol](017-a2a-protocol.md)
- [AG-UI Protocol](018-agui-protocol.md)
- [Chat Sessions API](013-chat-sessions.md)

---

*End of Universal Content Model Specification*
