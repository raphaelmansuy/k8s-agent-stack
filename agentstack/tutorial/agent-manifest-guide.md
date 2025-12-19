# AgentStack Manifest Guide

AgentStack uses a declarative approach to manage resources, similar to Kubernetes. This guide explains the structure of the YAML manifests used with `agentctl apply`.

## Resource Structure

Every manifest must include:
- `kind`: The type of resource (`Agent`, `Deployment`, `Project`, `APIKey`).
- `metadata`: Identification information (name, labels).
- `spec`: The desired state of the resource.

## 1. Agent Manifest
Defines the identity and behavior of an AI agent.

```yaml
kind: Agent
metadata:
  name: customer-support-bot
  labels:
    env: production
    team: support
spec:
  description: "Handles tier-1 customer support queries"
  model: "gpt-4o"
  systemPrompt: |
    You are a helpful customer support assistant.
    Always be polite and concise.
  tools:
    - name: search_kb
      description: "Search the knowledge base"
    - name: create_ticket
      description: "Create a support ticket"
```

## 2. Deployment Manifest
Defines how an agent should be run and scaled.

```yaml
kind: Deployment
metadata:
  name: support-bot-v1
spec:
  agentName: customer-support-bot
  replicas: 3
  resources:
    cpu: "500m"
    memory: "512Mi"
  env:
    - name: LOG_LEVEL
      value: "debug"
  strategy:
    type: RollingUpdate
    maxSurge: 1
    maxUnavailable: 0
```

## 3. API Key Manifest
Defines access credentials for external systems.

```yaml
kind: APIKey
metadata:
  name: mobile-app-key
spec:
  description: "Key for the mobile application"
  scopes:
    - "agent:chat"
    - "agent:read"
  expiresAt: "2025-12-31T23:59:59Z"
```

## 4. Multi-Document Manifests
You can combine multiple resources into a single file using `---`.

```yaml
kind: Project
metadata:
  name: support-system
---
kind: Agent
metadata:
  name: support-bot
  project: support-system
spec:
  description: "Support bot"
---
kind: Deployment
metadata:
  name: support-bot-prod
spec:
  agentName: support-bot
  replicas: 2
```

## Applying Manifests

Use the `apply` command to create or update resources:

```bash
# Apply a single file
agentctl apply -f agent.yaml

# Apply all files in a directory
agentctl apply -f ./manifests/

# Dry run (validate without applying)
agentctl apply -f stack.yaml --dry-run
```

## Best Practices
1. **Version Control**: Store your manifests in Git to track changes.
2. **Labels**: Use labels for organization and filtering.
3. **Immutability**: Treat deployments as immutable; update the manifest and re-apply to trigger a rollout.
4. **Secrets**: Do not store sensitive information directly in manifests. Use environment variables or a secret management system.
