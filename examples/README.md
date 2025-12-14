# Agent Examples

This directory contains example agent configurations and deployment manifests for k8s-agent-stack.

## Available Examples

### 1. Kubernetes Helper Agent

**File**: [k8s-helper-agent.yaml](k8s-helper-agent.yaml)

A declarative agent that helps users interact with their Kubernetes cluster through the Kagent web UI.

**Features**:
- List Kubernetes resources (pods, services, deployments)
- Describe specific resources
- Get pod logs
- View cluster events

**Deploy**:
```bash
kubectl apply -f examples/k8s-helper-agent.yaml
```

**Usage**: Access through Kagent web UI and ask:
- "Show me all pods in the default namespace"
- "What deployments are running?"
- "Get logs for pod xyz"

### 2. Model Configuration

**File**: [model-config.yaml](model-config.yaml)

Example `ModelConfig` resource for configuring LLM providers.

**Supported Providers**:
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude)
- Google (Gemini)
- Local models (Ollama, LM Studio)

**Deploy**:
```bash
# Create API key secret first
kubectl create secret generic openai-api-key \
  --from-literal=api-key=sk-... \
  -n kagent

# Deploy model config
kubectl apply -f examples/model-config.yaml
```

**Reference in Agent**:
```yaml
spec:
  declarative:
    modelConfig: gpt-4o-mini  # References the ModelConfig name
```

## Creating Your Own Agent

### Declarative Agent Template

```yaml
# Copyright 2025 Your Name
# Licensed under the Apache License, Version 2.0

apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: my-custom-agent
  namespace: kagent
spec:
  declarative:
    modelConfig: gpt-4o-mini
    systemMessage: |
      You are a helpful assistant that...
      [Define your agent's role and capabilities]
    tools:
      - type: McpServer
        mcpServer:
          apiGroup: kagent.dev
          kind: RemoteMCPServer
          name: kagent-tool-server
          toolNames:
            - tool1
            - tool2
```

### Custom Agent with Container

For agents built with frameworks like Google ADK, LangGraph, or CrewAI:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: my-adk-agent
  namespace: kagent
spec:
  custom:
    image: gcr.io/your-project/my-agent:v1
    port: 8080
    env:
      - name: GOOGLE_API_KEY
        valueFrom:
          secretKeyRef:
            name: google-api-key
            key: api-key
```

See [Building Google ADK Agents](../docs/building-google-adk-agents-for-kagent.md) for complete guide.

## Example Structure

When contributing new examples, use this structure:

```
examples/
  my-agent/
    README.md              # Agent description and usage
    agent.yaml             # Kagent Agent manifest
    model-config.yaml      # ModelConfig (if needed)
    deployment.yaml        # Kubernetes deployment (for custom agents)
    Dockerfile             # Container build (for custom agents)
    requirements.txt       # Dependencies (for Python agents)
```

## Testing Examples

```bash
# 1. Deploy the agent
kubectl apply -f examples/k8s-helper-agent.yaml

# 2. Verify agent is registered
kubectl get agents -n kagent

# 3. Check agent status
kubectl describe agent k8s-helper-agent -n kagent

# 4. Access through Kagent UI
kubectl port-forward -n kagent svc/kagent-web 3000:3000

# Open http://localhost:3000 and select your agent
```

## Available Tools

Kagent provides built-in tools through `kagent-tool-server`:

| Tool | Description |
|------|-------------|
| `k8s_get_resources` | List Kubernetes resources |
| `k8s_describe_resource` | Get resource details |
| `k8s_get_pod_logs` | Retrieve pod logs |
| `k8s_get_events` | View cluster events |
| `http_request` | Make HTTP requests |
| `search_web` | Search the internet |
| `read_file` | Read file contents |
| `write_file` | Write to files |

See [Kagent documentation](https://kagent.dev/docs/tools/) for complete tool reference.

## Common Patterns

### Multi-Tool Agent

```yaml
spec:
  declarative:
    tools:
      - type: McpServer
        mcpServer:
          name: kagent-tool-server
          toolNames:
            - k8s_get_resources
            - http_request
            - search_web
```

### Agent with Custom Memory

```yaml
spec:
  declarative:
    memory:
      type: redis
      config:
        host: redis-service
        port: 6379
```

### Agent with Specific Model

```yaml
spec:
  declarative:
    modelConfig: claude-3-sonnet  # Custom ModelConfig
    temperature: 0.7
    maxTokens: 2000
```

## Contributing Examples

We welcome new agent examples! Please:

1. Test thoroughly before submitting
2. Include clear documentation
3. Add appropriate comments
4. Follow the example structure above
5. Submit a pull request

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## Resources

- **[Kagent Documentation](https://kagent.dev/docs/)** - Official Kagent docs
- **[Building ADK Agents](../docs/building-google-adk-agents-for-kagent.md)** - Google ADK guide
- **[Deployment Guide](../docs/deployment-guide.md)** - Deployment strategies
- **[Architecture](../docs/architecture.md)** - Platform architecture

## License

All examples are licensed under Apache License 2.0. See [../LICENSE](../LICENSE).

---

[Back to README](../README.md) • [Documentation](../docs/) • [Getting Started](../docs/getting-started.md)
