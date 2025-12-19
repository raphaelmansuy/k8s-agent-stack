# Kagent Quick Start Tutorial

This tutorial will guide you through setting up and using the **k8s-agent-stack** to deploy and interact with AI agents on Kubernetes.

## 📋 Prerequisites

- **Kubernetes Cluster**: OrbStack (recommended), Docker Desktop, or Minikube.
- **OpenAI API Key**: Required for the agents to process natural language.
- **Tools**: `kubectl`, `docker`, and `make` installed.

---

## 🚀 Step 1: Installation

First, set your OpenAI API key and initialize the stack.

```bash
# 1. Set your API key
export OPENAI_API_KEY="your-openai-api-key"

# 2. Install the stack (CLI, Controller, and UI)
make start
```

*This command installs the `kagent` CLI, sets up the controller in your cluster, and configures your API key as a Kubernetes secret.*

---

## 🤖 Step 2: Deploy Your First Agent

We provide a reference agent built with the Google ADK (Agent Development Kit).

```bash
# Build and deploy the ADK agent
make adk-agent

# Check the status
make adk-agent-status
```

*Wait until the status shows `READY: True`.*

---

## 💬 Step 3: Interact via CLI

You can chat with your agent directly from the terminal using the `kagent` CLI.

```bash
# Ask a simple question
kagent invoke --namespace kagent --agent google-adk-byo-agent --task "Hello, what can you do?"

# Test its tools (Math)
kagent invoke --namespace kagent --agent google-adk-byo-agent --task "What is 123 * 456?"

# Test its tools (Weather)
kagent invoke --namespace kagent --agent google-adk-byo-agent --task "What is the weather in Paris?"
```

---

## 🌐 Step 4: Use the Web UI

Kagent comes with a built-in dashboard for managing and chatting with agents.

```bash
# Start the UI port-forward
make ui
```

1. Open [http://localhost:8080](http://localhost:8080) in your browser.
2. Navigate to the **Agents** section.
3. Select `google-adk-byo-agent` to start a chat session.

---

## 🧹 Step 5: Cleanup

When you are done, you can remove the kagent components from your cluster.

```bash
# Remove kagent and agents
make clean

# To remove everything including the namespace
make clean-all
```

---

## 💡 Next Steps

- **Build your own agent**: Check the [kagent-adk-agent/](kagent-adk-agent/) directory to see how the reference agent is implemented.
- **Explore Documentation**: Read the [docs/](../docs/) directory for architecture details and advanced deployment guides.
