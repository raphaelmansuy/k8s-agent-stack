# Kagent: The Cloud-Native AI Agent Platform

## 1. WHY Kagent? (The Value Proposition)

Most AI agents today exist as fragile Python scripts running on local machines. Moving them to production requires solving **lifecycle management**, **scaling**, **secrets**, and **networking**.

**Kagent** solves this by treating AI Agents as **Infrastructure**.

* **For Ops:** If you know Kubernetes, you know Kagent. Agents are just Custom Resources (CRDs).
* **For Devs:** You stop writing boilerplate for HTTP servers and API keys. You just define "Brain" (LLM) and "Tools" (MCP).
* **The Shift:** Stop thinking "Script." Start thinking "Deployment."

-----

## 2. The Mental Model: "Agents as Resources"

To understand Kagent, map it to concepts you already know:

| Traditional AI Dev | Kagent / Kubernetes Model |
| :--- | :--- |
| `client = OpenAI(...)` | `ModelConfig` (CRD) |
| `def get_weather(): ...` | `RemoteMCPServer` (CRD) |
| `while True: chat()` | `Agent` (CRD) |
| `python agent.py` | `kubectl apply -f agent.yaml` |

**The Control Loop:**
Just as a K8s Controller watches a `Deployment` and ensures Pods are running, the **Kagent Controller** watches an `Agent` CRD and ensures the necessary compute, networking, and tool connections are active.

-----

## 3. Architecture Overview

Kagent relies on the **Model Context Protocol (MCP)** to standardize how agents talk to tools.

```ascii
      THE BRAIN                   THE BODY                  THE HANDS
+-------------------+       +------------------+       +------------------+
|   ModelConfig     |       |      Agent       |       | RemoteMCPServer  |
| (CRD: Credentials)|       | (CRD: The Logic) |       | (CRD: The Tools) |
+---------+---------+       +--------+---------+       +--------+---------+
          |                          |                          |
          v                          v                          v
  +---------------+          +---------------+          +---------------+
  |  LLM Provider |<-------->|  Agent Pod    |<-------->|  Tool Service |
  | (OpenAI, etc) |   API    | (Stateless)   |   MCP    | (DB, APIs)    |
  +---------------+          +---------------+          +---------------+
                                     ^
                                     | HTTP/SSE
                             +-------+-------+
                             |   User / App  |
                             +---------------+
```

-----

## 4. Actionable Tutorial: Zero to Agent

### Phase 1: Installation

We assume you have a cluster (`kind`, `minikube`, or cloud).

```bash
# 1. Add Repo
helm repo add kagent oci://ghcr.io/kagent-dev/kagent/helm

# 2. Install CRDs (The Definitions)
helm install kagent-crds kagent/kagent-crds -n kagent --create-namespace

# 3. Install Controller (The Manager)
# Replace 'sk-...' with your actual key
export OPENAI_API_KEY="sk-..."
helm install kagent kagent/kagent -n kagent \
  --set providers.default=openai \
  --set providers.openai.apiKey=$OPENAI_API_KEY
```

### Phase 2: Define the "Brain" (ModelConfig)

Decouple your prompt logic from your wallet. This config handles the auth.

```yaml
# model.yaml
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: gpt-4o-mini
  namespace: kagent
spec:
  provider: OpenAI
  model: gpt-4o-mini
  credentials:
    apiKeySecret:
      name: openai-api-key
      key: api-key
```

### Phase 3: Define the "Hands" (Tools via MCP)

We will use a standard "Filesystem" tool as an example.

```yaml
# tools.yaml
apiVersion: kagent.dev/v1alpha2
kind: RemoteMCPServer
metadata:
  name: fs-tools
  namespace: kagent
spec:
  type: McpServer
  protocol: STREAMABLE_HTTP
  # This URL points to a service running an MCP-compliant server
  url: http://filesystem-mcp.default.svc.cluster.local/sse
```

### Phase 4: The Agent (The Logic)

This binds the brain and hands together.

```yaml
# agent.yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: system-admin
  namespace: kagent
spec:
  # OPTION A: Declarative (No code, just prompt)
  declarative:
    modelConfig: gpt4-turbo
    systemMessage: "You are a sysadmin. Use tools to manage files."
    tools:
      - mcpServer:
          name: fs-tools
          toolNames: ["list_files", "read_file"]
```

**Deploy:**

```bash
kubectl apply -f model.yaml -f tools.yaml -f agent.yaml
```

-----

## 5. Advanced Concepts: Lifecycle, Memory & ADK

[Image of AI agent memory architecture diagram]

This is where Kagent moves beyond simple scripts.

### A. Lifecycle & Scaling

* **Scenario:** Your agent crashes due to an OOM error.
* **Kagent's Reaction:** Kubernetes detects the Pod death. The ReplicaSet recreates the Pod.
* **The Difference:** The *Context* (chat history) is not in the Pod. It is held by the Client or the Session Manager. The new Pod picks up the next request seamlessly.

### B. Memory (The "Stateful Pocket")

Do not store state in the agent. Use **MCP Memory Servers**.

1. Deploy a Vector Database (e.g., Qdrant/Weaviate) in your cluster.
2. Deploy an MCP Server that wraps that DB.
3. Give the `Agent` access to that MCP Server.

Now, your agent has "Long Term Memory" that survives restarts.

### C. Integrating Google ADK (BYO Logic)

If you need complex logic (loops, chains) that YAML can't provide, use the **Bring Your Own (BYO)** pattern with Google ADK.

1. **Build:** Create a Docker container running your Google ADK code.
2. **Expose:** Ensure it listens on port 8080.
3. **Config:**

```yaml
# adk-agent.yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: my-complex-researcher
spec:
  type: BYO  # <--- Tells Kagent "I have my own container"
  byo:
    deployment:
      image: my-registry/adk-agent:v1
```

-----

## 6. Interaction

Once running, the Agent exposes a service. You can interact via the CLI or HTTP.

**CLI One-Liner:**

```bash
# Start a chat session
kagent run -n kagent --agent system-admin "List the files in /tmp"
```

**Next Step for You:**
Check your current cluster resources. If you have `helm` installed, run the **Phase 1** commands above to get the controller running. If successful, you should see the pod running: `kubectl get pods -n kagent`.
