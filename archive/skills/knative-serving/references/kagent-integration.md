# kagent Integration

Integrating Knative Services with kagent orchestrator.

## Overview

kagent is a CNCF Sandbox project that provides:
- Agent-to-Agent (A2A) protocol support
- Multi-framework agent orchestration
- Agent discovery and routing
- Unified management UI

## kagent Agent CRD

Define agents as Kubernetes custom resources:

```yaml
apiVersion: kagent.dev/v1alpha1
kind: Agent
metadata:
  name: customer-support-agent
  namespace: agents
spec:
  # Agent metadata
  displayName: "Customer Support Agent"
  description: "Handles customer inquiries and support tickets"
  version: "1.0.0"
  
  # Framework configuration
  framework: google-adk
  
  # Runtime configuration
  runtime:
    type: knative
    knative:
      serviceRef:
        name: customer-support-agent
        namespace: agents
  
  # A2A protocol configuration
  a2a:
    enabled: true
    capabilities:
      - name: query
        description: "Answer customer questions"
      - name: create_ticket
        description: "Create support tickets"
    
  # Model configuration
  model:
    provider: openai
    model: gpt-4o
    configRef:
      name: openai-config
      namespace: agents
  
  # Tool bindings
  tools:
    - name: search-knowledge-base
      type: mcp
      mcpRef:
        name: kb-search-server
        namespace: tools
    - name: create-ticket
      type: http
      httpRef:
        url: "http://ticketing-api.internal/api/tickets"
        method: POST
```

## Linking Knative Service to kagent

### Option 1: Annotate Knative Service

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: customer-support-agent
  namespace: agents
  annotations:
    kagent.dev/agent-name: "customer-support-agent"
    kagent.dev/framework: "google-adk"
    kagent.dev/a2a-enabled: "true"
spec:
  # ... service spec
```

### Option 2: Separate Agent CRD

```yaml
apiVersion: kagent.dev/v1alpha1
kind: Agent
metadata:
  name: customer-support-agent
spec:
  runtime:
    type: knative
    knative:
      serviceRef:
        name: customer-support-agent
        namespace: agents
```

## Agent Discovery

kagent provides DNS-based discovery:

```
# Internal service URL
http://<agent-name>.agents.svc.cluster.local

# A2A discovery endpoint
http://<agent-name>.agents.svc.cluster.local/.well-known/agent.json
```

### Agent Card Response

```json
{
  "name": "customer-support-agent",
  "description": "Handles customer inquiries",
  "version": "1.0.0",
  "capabilities": [
    {
      "name": "query",
      "description": "Answer customer questions",
      "inputSchema": {...}
    }
  ],
  "endpoint": "http://customer-support-agent.agents.svc.cluster.local"
}
```

## Multi-Agent Orchestration

### Agent Composition

```yaml
apiVersion: kagent.dev/v1alpha1
kind: AgentComposition
metadata:
  name: enterprise-assistant
spec:
  orchestrator:
    type: supervisor
    agentRef:
      name: supervisor-agent
  
  workers:
    - name: customer-support
      agentRef:
        name: customer-support-agent
      capabilities: [query, create_ticket]
    
    - name: billing
      agentRef:
        name: billing-agent
      capabilities: [check_balance, process_payment]
    
    - name: technical
      agentRef:
        name: technical-agent
      capabilities: [troubleshoot, escalate]
```

### Routing Rules

```yaml
apiVersion: kagent.dev/v1alpha1
kind: AgentRoute
metadata:
  name: support-routing
spec:
  selector:
    matchLabels:
      department: support
  
  rules:
    - match:
        intent: billing
      target:
        agentRef:
          name: billing-agent
    
    - match:
        intent: technical
      target:
        agentRef:
          name: technical-agent
    
    - match:
        default: true
      target:
        agentRef:
          name: customer-support-agent
```

## Model Configuration

```yaml
apiVersion: kagent.dev/v1alpha1
kind: ModelConfig
metadata:
  name: openai-config
  namespace: agents
spec:
  provider: openai
  model: gpt-4o
  parameters:
    temperature: 0.7
    maxTokens: 4096
  credentials:
    secretRef:
      name: openai-credentials
      key: api_key
```

## Observability Integration

kagent integrates with platform observability:

```yaml
apiVersion: kagent.dev/v1alpha1
kind: Agent
metadata:
  name: customer-support-agent
spec:
  observability:
    tracing:
      enabled: true
      samplingRate: 0.1
    metrics:
      enabled: true
      port: 9090
    logging:
      level: info
      format: json
```

## Accessing kagent UI

```bash
# Port forward to kagent UI
kubectl port-forward -n kagent svc/kagent-ui 8080:80

# Access at http://localhost:8080
```

## API Integration

### List Agents

```bash
curl http://kagent.agentstack.svc.cluster.local/api/v1/agents \
  -H "Authorization: Bearer $TOKEN"
```

### Invoke Agent

```bash
curl http://kagent.agentstack.svc.cluster.local/api/v1/agents/customer-support-agent/invoke \
  -H "Content-Type: application/json" \
  -d '{"message": "How do I reset my password?"}'
```

### Stream Response

```bash
curl http://kagent.agentstack.svc.cluster.local/api/v1/agents/customer-support-agent/stream \
  -H "Accept: text/event-stream" \
  -d '{"message": "How do I reset my password?"}'
```
