# AgentStack CLI (`agentctl`) Quickstart Guide

This guide provides concise, actionable examples for managing your AI agents and deployments using `agentctl`.

## 1. Getting Started

### Authentication
Authenticate with the AgentStack API.
```bash
# Interactive login
agentctl login --endpoint http://api.agentstack.com

# Login with an existing API key
agentctl login --api-key your-secret-key --profile prod
```

### Configuration
Manage CLI profiles and settings.
```bash
# View current config
agentctl config view

# List profiles
agentctl config profiles

# Switch profile
agentctl config use-profile prod

# Set default project for current profile
agentctl config set default-project my-project-id
```

## 2. Managing Projects
Projects are the top-level containers for agents.
```bash
# List projects
agentctl project list

# Create a new project
agentctl project create --name "Customer Support" --description "Support agents"

# Get project details
agentctl project get customer-support
```

## 3. Managing Agents
Agents are the core AI entities.
```bash
# List agents in current project
agentctl agent list

# Create an agent
agentctl agent create --name "Echo Agent" --project-id my-project

# Get agent details
agentctl agent get echo-agent
```

## 4. Declarative Management (`apply`)
Manage resources using YAML manifests (recommended for production).
```bash
# Apply a manifest (creates or updates)
agentctl apply -f agent.yaml

# Apply a full stack (multi-document YAML)
agentctl apply -f stack.yaml
```

**Example `stack.yaml`:**
```yaml
kind: Agent
metadata:
  name: support-bot
spec:
  description: "Helpful support bot"
---
kind: Deployment
metadata:
  name: support-bot-prod
spec:
  agentName: support-bot
  replicas: 3
  image: agentstack/support-bot:latest
```

## 5. Deployments & Scaling
Run your agents at scale.
```bash
# List deployments
agentctl deploy list

# Create a deployment for an agent
agentctl deploy create --agent echo-agent --version v1.0.0

# Scale a deployment
agentctl deploy scale my-deployment-id --replicas 5

# Check deployment status
agentctl deploy status my-deployment-id
```

## 6. Observability (Logs)
Monitor your agents in real-time.
```bash
# View last 100 lines of logs
agentctl logs echo-agent

# Follow logs in real-time (tail -f)
agentctl logs echo-agent -f

# View logs for a specific deployment with timestamps
agentctl logs deploy/my-deployment-id --timestamps
```

## 7. Interactive Chat
Test your agents directly from the terminal.
```bash
# Start an interactive chat session
agentctl chat echo-agent

# Chat commands (inside session):
# /help    - Show help
# /clear   - Clear history
# /quit    - Exit chat
```

## 8. Security (API Keys)
Manage access keys for your applications.
```bash
# List API keys
agentctl keys list

# Create a new key with specific scopes
agentctl keys create --name "CI/CD Key" --scopes "agent:read,deploy:create"

# Rotate a key (invalidates old, generates new)
agentctl keys rotate my-key-id --force

# Delete a key
agentctl keys delete my-key-id
```

## Global Flags
These flags can be used with any command:
- `-o, --output`: Output format (`table`, `json`, `yaml`, `wide`)
- `-p, --project`: Override the default project ID
- `-v, --verbose`: Enable verbose logging
- `--api-key`: Override the configured API key
