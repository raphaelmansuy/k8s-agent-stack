# 014 - Tools & Models API

> Tool Registry, Model Providers, MCP Integration

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Overview

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Tool & Model Architecture                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Agent                                                           │
│    │                                                             │
│    ├── Model Config ──────────▶ Provider (OpenAI, Anthropic)   │
│    │                                                             │
│    └── Tools ─────────────────▶ Tool Registry                  │
│              │                       │                           │
│              │                       ├── Built-in               │
│              │                       ├── Custom (HTTP, Code)    │
│              │                       └── MCP Servers            │
│              │                                                   │
│              └── At runtime ──▶ Tool Execution                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Tools API

### 2.1 List Tools

```yaml
GET /v1/tools
X-Project-ID: prj_abc123

Query Parameters:
  type:     builtin | custom | mcp
  category: string
  search:   string
  limit:    1-100
  cursor:   string

Response: 200 OK
{
  "data": [
    {
      "id": "tol_7nOp8qRsSt",
      "name": "search-kb",
      "description": "Search the knowledge base",
      "type": "mcp",
      "category": "information",
      "input_schema": {
        "type": "object",
        "properties": {
          "query": {"type": "string", "description": "Search query"}
        },
        "required": ["query"]
      },
      "enabled": true,
      "usage_count": 1500,
      "created_at": "2025-01-10T08:00:00Z"
    },
    {
      "id": "builtin:web-search",
      "name": "web-search",
      "description": "Search the web",
      "type": "builtin",
      "category": "information",
      "enabled": true
    }
  ],
  "pagination": { ... }
}
```

### 2.2 Create Tool

```yaml
POST /v1/tools
X-Project-ID: prj_abc123

# HTTP Tool (calls external API)
{
  "name": "weather-api",
  "description": "Get current weather for a city",
  "type": "http",
  "category": "information",
  "config": {
    "method": "GET",
    "url": "https://api.weather.com/v1/current",
    "headers": {
      "Authorization": "Bearer {{secrets.WEATHER_API_KEY}}"
    },
    "query_params": {
      "city": "{{input.city}}"
    },
    "timeout_ms": 5000
  },
  "input_schema": {
    "type": "object",
    "properties": {
      "city": {"type": "string", "description": "City name"}
    },
    "required": ["city"]
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "temperature": {"type": "number"},
      "condition": {"type": "string"}
    }
  }
}

# Code Tool (runs in sandbox)
{
  "name": "calculate",
  "description": "Evaluate mathematical expressions safely",
  "type": "code",
  "config": {
    "runtime": "python3.11",
    "code": "def execute(expression: str) -> float:\n    import math\n    return eval(expression, {'__builtins__': {}}, {'math': math})",
    "timeout_ms": 5000,
    "memory_mb": 128
  },
  "input_schema": {
    "type": "object",
    "properties": {
      "expression": {"type": "string"}
    },
    "required": ["expression"]
  }
}

Response: 201 Created
{
  "id": "tol_7nOp8qRsSt",
  "name": "weather-api",
  ...
}
```

### 2.3 Get Tool

```yaml
GET /v1/tools/{toolId}

Response: 200 OK
{
  "id": "tol_7nOp8qRsSt",
  "name": "weather-api",
  "description": "Get current weather for a city",
  "type": "http",
  "config": { ... },
  "input_schema": { ... },
  "output_schema": { ... },
  "enabled": true,
  "usage_count": 1500,
  "created_at": "2025-01-10T08:00:00Z",
  "updated_at": "2025-01-15T10:00:00Z"
}
```

### 2.4 Update Tool

```yaml
PATCH /v1/tools/{toolId}

{
  "description": "Updated description",
  "config": { ... },
  "enabled": false
}

Response: 200 OK
{ ... }
```

### 2.5 Delete Tool

```yaml
DELETE /v1/tools/{toolId}

Response: 204 No Content
```

### 2.6 Test Tool

```yaml
POST /v1/tools/{toolId}/test

{
  "input": {
    "city": "Tokyo"
  }
}

Response: 200 OK
{
  "success": true,
  "output": {
    "temperature": 22,
    "condition": "sunny"
  },
  "duration_ms": 150
}

# On failure
{
  "success": false,
  "error": "Connection timeout",
  "duration_ms": 5000
}
```

---

## 3. Tool Types

### 3.1 Built-in Tools

```yaml
# Platform-provided tools (no configuration needed)
builtin_tools:
  - id: builtin:web-search
    name: web-search
    description: Search the web using DuckDuckGo
    
  - id: builtin:code-interpreter
    name: code-interpreter
    description: Execute Python code in sandbox
    
  - id: builtin:file-read
    name: file-read
    description: Read files from attached storage
    
  - id: builtin:http-request
    name: http-request
    description: Make HTTP requests (with approval)
```

### 3.2 HTTP Tool

```yaml
# External API integration
config:
  method: GET | POST | PUT | PATCH | DELETE
  url: string                      # Supports {{input.xxx}}
  headers: map<string, string>     # Supports {{secrets.XXX}}
  query_params: map<string, string>
  body_template: string            # JSON template
  timeout_ms: 1000-30000
  retry:
    max_attempts: 1-5
    backoff_ms: 100-5000
```

### 3.3 Code Tool

```yaml
# Sandboxed code execution
config:
  runtime: python3.11 | python3.12 | node20
  code: string                     # Function definition
  timeout_ms: 1000-60000
  memory_mb: 64-512
  allowed_packages:                # Whitelist
    - requests
    - pandas
```

### 3.4 MCP Tool

```yaml
# Model Context Protocol server
config:
  server_url: string               # MCP server endpoint
  transport: stdio | sse | http
  auth:
    type: none | api_key | oauth
    credentials: {{secrets.MCP_TOKEN}}
  tools:                           # Optional: filter tools from server
    - search
    - create
```

---

## 4. MCP Servers

### 4.1 List MCP Servers

```yaml
GET /v1/mcp-servers

Response: 200 OK
{
  "data": [
    {
      "id": "mcp_abc123",
      "name": "knowledge-base",
      "url": "https://kb.example.com/mcp",
      "status": "connected",
      "tools_count": 5,
      "last_health_check": "2025-01-15T10:30:00Z"
    }
  ]
}
```

### 4.2 Register MCP Server

```yaml
POST /v1/mcp-servers

{
  "name": "knowledge-base",
  "url": "https://kb.example.com/mcp",
  "transport": "http",
  "auth": {
    "type": "api_key",
    "header": "Authorization",
    "value": "{{secrets.KB_API_KEY}}"
  }
}

Response: 201 Created
{
  "id": "mcp_abc123",
  "name": "knowledge-base",
  "tools": [
    {"name": "search", "description": "Search documents"},
    {"name": "get", "description": "Get document by ID"}
  ]
}
```

### 4.3 Sync MCP Tools

```yaml
POST /v1/mcp-servers/{serverId}/sync

Response: 200 OK
{
  "synced_tools": ["search", "get", "create"],
  "added": ["create"],
  "removed": []
}
```

---

## 5. Models API

### 5.1 List Models

```yaml
GET /v1/models
X-Project-ID: prj_abc123

Query Parameters:
  provider:   openai | anthropic | google | azure | bedrock
  capability: chat | embedding | vision | audio

Response: 200 OK
{
  "data": [
    {
      "id": "gpt-4o",
      "provider": "openai",
      "name": "GPT-4o",
      "capabilities": ["chat", "vision", "function_calling"],
      "context_window": 128000,
      "max_output": 16384,
      "pricing": {
        "input_per_1k": 0.0025,
        "output_per_1k": 0.01
      }
    },
    {
      "id": "claude-3-5-sonnet-20241022",
      "provider": "anthropic",
      "name": "Claude 3.5 Sonnet",
      "capabilities": ["chat", "vision", "function_calling"],
      "context_window": 200000,
      "max_output": 8192,
      "pricing": {
        "input_per_1k": 0.003,
        "output_per_1k": 0.015
      }
    }
  ]
}
```

### 5.2 List Providers

```yaml
GET /v1/models/providers

Response: 200 OK
{
  "data": [
    {
      "id": "prov_openai",
      "provider": "openai",
      "status": "active",
      "models_available": 8,
      "created_at": "2025-01-01T00:00:00Z"
    },
    {
      "id": "prov_anthropic",
      "provider": "anthropic",
      "status": "active",
      "models_available": 4,
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

### 5.3 Add Provider

```yaml
POST /v1/models/providers

{
  "provider": "openai",
  "config": {
    "api_key": "sk-...",
    "organization": "org-..."
  }
}

# Azure OpenAI
{
  "provider": "azure",
  "config": {
    "api_key": "...",
    "endpoint": "https://myresource.openai.azure.com",
    "api_version": "2024-02-01",
    "deployments": {
      "gpt-4o": "my-gpt4o-deployment"
    }
  }
}

Response: 201 Created
{
  "id": "prov_openai",
  "provider": "openai",
  "status": "active",
  "models_available": 8
}
```

### 5.4 Update Provider

```yaml
PATCH /v1/models/providers/{providerId}

{
  "config": {
    "api_key": "sk-new-key..."
  }
}

Response: 200 OK
```

### 5.5 Delete Provider

```yaml
DELETE /v1/models/providers/{providerId}

Response: 204 No Content
```

---

## 6. Model Configuration

### 6.1 Per-Agent Model Config

```yaml
# In agent configuration
{
  "config": {
    "model": "gpt-4o",
    "model_config": {
      "temperature": 0.7,
      "max_tokens": 4096,
      "top_p": 0.9,
      "frequency_penalty": 0,
      "presence_penalty": 0
    }
  }
}
```

### 6.2 Fallback Models

```yaml
# Configure fallback chain
{
  "config": {
    "model": "gpt-4o",
    "fallback_models": [
      "gpt-4o-mini",
      "claude-3-5-sonnet-20241022"
    ]
  }
}
```

---

## 7. Provider Configuration Reference

### OpenAI

```yaml
provider: openai
config:
  api_key: required
  organization: optional
  base_url: optional (for proxies)
```

### Anthropic

```yaml
provider: anthropic
config:
  api_key: required
```

### Google (Vertex AI)

```yaml
provider: google
config:
  project_id: required
  location: required (e.g., us-central1)
  credentials: service account JSON
```

### Azure OpenAI

```yaml
provider: azure
config:
  api_key: required
  endpoint: required
  api_version: required
  deployments: map<model_id, deployment_name>
```

### AWS Bedrock

```yaml
provider: bedrock
config:
  region: required
  access_key_id: optional (uses IAM if not provided)
  secret_access_key: optional
```

---

## 8. Multi-Tenant Considerations

| Aspect | Implementation |
|--------|----------------|
| **Tool isolation** | Tools scoped to project |
| **Provider keys** | Per-project or per-org secrets |
| **MCP servers** | Per-project registration |
| **Model access** | Plan-based model availability |
| **Usage tracking** | Per-project token metering |

---

## 9. Error Responses

```yaml
# Tool not found
404 Not Found
{
  "type": "https://api.agentstack.io/errors/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "Tool tol_notexist was not found"
}

# Provider error
503 Service Unavailable
{
  "type": "https://api.agentstack.io/errors/provider-error",
  "title": "Provider Error",
  "status": 503,
  "detail": "OpenAI API returned error: rate limit exceeded"
}

# Invalid tool config
400 Bad Request
{
  "type": "https://api.agentstack.io/errors/validation",
  "title": "Validation Error",
  "status": 400,
  "errors": [
    {"field": "config.url", "message": "Invalid URL format"}
  ]
}
```

---

## 10. Implementation Checklist

- [ ] Tool CRUD endpoints
- [ ] HTTP tool execution
- [ ] Code tool sandboxing
- [ ] MCP server integration
- [ ] Model provider management
- [ ] Per-tenant tool isolation
- [ ] Usage metering
- [ ] Tool testing endpoint

---

**Previous**: [013-chat-sessions.md](013-chat-sessions.md)  
**Next**: [015-admin-endpoints.md](015-admin-endpoints.md)
