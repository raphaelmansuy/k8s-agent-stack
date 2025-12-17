# 009 - Developer Experience

> CLI, SDKs, Local Development, Testing, and Workflows

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Developer Workflow

```text
┌─────────────────────────────────────────────────────────────────┐
│                    Developer Workflow                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐         │
│  │  Init   │──▶│  Code   │──▶│  Test   │──▶│ Deploy  │         │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘         │
│       │             │             │             │               │
│       ▼             ▼             ▼             ▼               │
│   agentctl     VSCode/IDE    Local mode     git push            │
│   init         + Copilot     + mocks        (GitOps)            │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   Feedback Loop                          │    │
│  │    Logs ◀── Metrics ◀── Traces ◀── Alerts               │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. CLI (agentctl)

### 2.1 Core Commands

```bash
# Authentication
agentctl auth login                    # Browser-based login
agentctl auth login --token <token>    # API token
agentctl auth whoami                   # Current identity

# Project management
agentctl project create <name>
agentctl project list
agentctl project switch <name>

# Agent operations
agentctl agent init <name>             # Scaffold new agent
agentctl agent deploy <path>           # Deploy agent
agentctl agent list                    # List agents
agentctl agent logs <name> --follow    # Stream logs
agentctl agent delete <name>           # Remove agent

# Development
agentctl dev                           # Start local dev server
agentctl test                          # Run agent tests
agentctl chat <agent>                  # Interactive chat
```

### 2.2 Agent Scaffolding

```bash
$ agentctl agent init customer-support --template google-adk

Creating agent: customer-support
✓ Created directory structure
✓ Generated agent.yaml
✓ Generated Dockerfile
✓ Created tests/
✓ Added .gitignore

Next steps:
  cd customer-support
  agentctl dev          # Start local development
  agentctl deploy       # Deploy to platform
```

Generated structure:
```text
customer-support/
├── agent.yaml          # Agent configuration
├── Dockerfile          # Container definition
├── requirements.txt    # Python dependencies
├── src/
│   └── agent.py        # Agent code
└── tests/
    └── test_agent.py   # Unit tests
```

### 2.3 Configuration File

```yaml
# ~/.agentctl/config.yaml
current-context: production
contexts:
  - name: production
    endpoint: https://api.agentstack.io
    project: prj_abc123
    
  - name: staging
    endpoint: https://staging.agentstack.io
    project: prj_staging
    
  - name: local
    endpoint: http://localhost:8080
    project: local
```

---

## 3. SDKs

### 3.1 Python SDK

```python
from agentstack import Client, Agent

# Initialize client
client = Client(api_key="as_...")

# Create agent from YAML
agent = client.agents.create_from_file("agent.yaml")

# Or programmatically
agent = client.agents.create(
    name="my-agent",
    model="gpt-4o",
    system_prompt="You are a helpful assistant.",
    tools=["search-kb", "create-ticket"]
)

# Chat with agent
response = client.agents.chat(
    agent_id=agent.id,
    message="Hello, how can you help?"
)

# Streaming
for chunk in client.agents.chat_stream(agent_id=agent.id, message="..."):
    print(chunk.content, end="")
```

### 3.2 TypeScript/JavaScript SDK

```typescript
import { AgentStack } from '@agentstack/sdk';

const client = new AgentStack({ apiKey: 'as_...' });

// Create agent
const agent = await client.agents.create({
  name: 'my-agent',
  model: 'gpt-4o',
  systemPrompt: 'You are a helpful assistant.'
});

// Chat with streaming
const stream = await client.agents.chat(agent.id, {
  message: 'Hello!',
  stream: true
});

for await (const chunk of stream) {
  process.stdout.write(chunk.content);
}
```

### 3.3 Go SDK

```go
package main

import (
    "context"
    "fmt"
    agentstack "github.com/agentstack/sdk-go"
)

func main() {
    client := agentstack.NewClient("as_...")
    
    // Create agent
    agent, _ := client.Agents.Create(context.Background(), &agentstack.CreateAgentRequest{
        Name:         "my-agent",
        Model:        "gpt-4o",
        SystemPrompt: "You are a helpful assistant.",
    })
    
    // Chat
    resp, _ := client.Agents.Chat(context.Background(), agent.ID, &agentstack.ChatRequest{
        Message: "Hello!",
    })
    
    fmt.Println(resp.Content)
}
```

---

## 4. Local Development

### 4.1 Local Dev Server

```bash
$ agentctl dev

Starting local development server...
✓ Loading agent.yaml
✓ Building container (docker build)
✓ Starting dependencies (Redis, mock LLM)
✓ Agent running at http://localhost:8080

Endpoints:
  Chat:    POST http://localhost:8080/chat
  SSE:     POST http://localhost:8080/chat/stream
  Health:  GET  http://localhost:8080/health

Watching for changes... (Ctrl+C to stop)
```

### 4.2 Docker Compose

```yaml
# docker-compose.yaml
version: '3.8'
services:
  agent:
    build: .
    ports:
      - "8080:8080"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - REDIS_URL=redis://redis:6379
    volumes:
      - ./src:/app/src    # Hot reload
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  # Mock LLM for testing
  mock-llm:
    image: agentstack/mock-llm:latest
    ports:
      - "8081:8080"
```

### 4.3 IDE Integration

```text
┌─────────────────────────────────────────────────────────────────┐
│                   VSCode Extension                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Features:                                                       │
│  • YAML schema validation for agent.yaml                        │
│  • IntelliSense for configuration                               │
│  • Integrated terminal for agentctl                             │
│  • Log streaming sidebar                                        │
│  • Debug configuration for local agents                         │
│                                                                  │
│  Commands (Cmd+Shift+P):                                        │
│  • AgentStack: Create New Agent                                 │
│  • AgentStack: Start Dev Server                                 │
│  • AgentStack: Deploy Agent                                     │
│  • AgentStack: View Logs                                        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Testing

### 5.1 Unit Testing

```python
# tests/test_agent.py
import pytest
from agentstack.testing import MockLLM, AgentTestClient

@pytest.fixture
def client():
    return AgentTestClient("agent.yaml")

@pytest.fixture
def mock_llm():
    return MockLLM(responses=[
        "I can help you with customer support.",
        "Let me look that up for you."
    ])

def test_greeting(client, mock_llm):
    response = client.chat("Hello")
    assert "help" in response.content.lower()
    assert response.status == "success"

def test_tool_call(client, mock_llm):
    mock_llm.add_tool_response("search-kb", {"results": ["FAQ #1"]})
    
    response = client.chat("Search for password reset")
    
    assert len(response.tool_calls) == 1
    assert response.tool_calls[0].name == "search-kb"
```

### 5.2 Integration Testing

```python
# tests/integration/test_e2e.py
import pytest
from agentstack import Client

@pytest.fixture(scope="module")
def live_client():
    return Client(
        endpoint="https://staging.agentstack.io",
        api_key=os.environ["STAGING_API_KEY"]
    )

@pytest.mark.integration
def test_full_conversation(live_client):
    # Start session
    session = live_client.sessions.create()
    
    # First message
    r1 = live_client.chat(
        agent_id="customer-support",
        session_id=session.id,
        message="I need help with my order"
    )
    assert r1.status == "success"
    
    # Follow-up
    r2 = live_client.chat(
        agent_id="customer-support",
        session_id=session.id,
        message="Order #12345"
    )
    assert "12345" in r2.content
```

### 5.3 Load Testing

```python
# tests/load/locustfile.py
from locust import HttpUser, task, between

class AgentUser(HttpUser):
    wait_time = between(1, 3)
    
    @task
    def chat(self):
        self.client.post("/agents/customer-support/chat", json={
            "message": "Hello, I need help"
        }, headers={
            "Authorization": f"Bearer {self.api_key}"
        })
```

```bash
# Run load test
locust -f tests/load/locustfile.py \
  --host=https://staging.agentstack.io \
  --users=100 \
  --spawn-rate=10 \
  --run-time=5m
```

---

## 6. Documentation

### 6.1 Structure

```text
docs/
├── getting-started/
│   ├── quickstart.md
│   ├── installation.md
│   └── first-agent.md
├── guides/
│   ├── tools.md
│   ├── memory.md
│   └── multi-agent.md
├── api-reference/
│   ├── rest-api.md
│   ├── python-sdk.md
│   └── cli.md
├── tutorials/
│   ├── customer-support-bot.md
│   └── code-assistant.md
└── concepts/
    ├── agents.md
    ├── a2a-protocol.md
    └── mcp.md
```

### 6.2 API Reference Generation

```yaml
# docs/openapi.yaml (generated from code)
openapi: 3.1.0
info:
  title: AgentStack API
  version: 1.0.0

# Auto-generated from:
# - Go struct tags
# - Python type hints
# - TypeScript interfaces
```

---

## 7. Templates & Examples

### 7.1 Agent Templates

| Template | Framework | Use Case |
|----------|-----------|----------|
| `google-adk` | Google ADK | General purpose |
| `langgraph` | LangGraph | Complex workflows |
| `crewai` | CrewAI | Multi-agent |
| `minimal` | None | Simple HTTP |

### 7.2 Example Agents

```text
examples/
├── customer-support/      # Ticket handling, FAQ
├── code-assistant/        # Code review, generation
├── data-analyst/          # SQL queries, visualization
├── onboarding-bot/        # User onboarding flow
└── multi-agent-research/  # CrewAI collaboration
```

### 7.3 Quickstart Template

```yaml
# agent.yaml (minimal)
apiVersion: agentstack.io/v1alpha1
kind: Agent
metadata:
  name: my-first-agent
spec:
  model: gpt-4o
  systemPrompt: |
    You are a helpful assistant.
  
  # Optional: Add tools
  tools:
    - name: search
      type: mcp
      server: search-kb
```

---

## 8. Developer Portal

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Developer Portal                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Dashboard                                               │    │
│  │  ├── Projects                                           │    │
│  │  ├── Agents                                             │    │
│  │  ├── API Keys                                           │    │
│  │  └── Usage & Billing                                    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Playground                                              │    │
│  │  • Interactive chat with any agent                      │    │
│  │  • Tool testing                                         │    │
│  │  • Prompt engineering                                   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Docs                                                    │    │
│  │  • API Reference                                        │    │
│  │  • Guides                                               │    │
│  │  • Tutorials                                            │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 9. Implementation Checklist

### Phase 1: Core Tools
- [ ] CLI scaffolding (Go/Cobra)
- [ ] Python SDK
- [ ] Local dev server
- [ ] Basic documentation

### Phase 2: Testing & DX
- [ ] Testing framework
- [ ] VSCode extension
- [ ] TypeScript SDK
- [ ] Example agents

### Phase 3: Portal
- [ ] Developer portal
- [ ] API playground
- [ ] Usage dashboard

---

## 10. References

- [Cobra CLI Framework](https://cobra.dev/)
- [Google ADK](https://github.com/google/adk-python)
- [LangGraph](https://langchain-ai.github.io/langgraph/)
- [OpenAPI Generator](https://openapi-generator.tech/)

---

**Previous**: [008-deployment-operations.md](008-deployment-operations.md)  
**Index**: [README.md](README.md)
