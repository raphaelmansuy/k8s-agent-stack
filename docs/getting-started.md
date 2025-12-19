# Getting Started with k8s-agent-stack

Deploy sovereign AI agents on Kubernetes in under 10 minutes.

> **New to the platform?** Read the [Architecture Overview](architecture.md) to understand how the components work together.

## Prerequisites

### Required Tools

```bash
# Verify you have these installed
kubectl version --client    # Kubernetes CLI (v1.28+)
helm version                # Helm 3.x
docker --version            # Docker or OrbStack
```

### Install on macOS

```bash
brew install kubectl helm
brew install --cask orbstack  # Recommended for M1/M2/M3 Macs
# OR: brew install --cask docker
```

### Install on Linux

```bash
# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

---

## Quick Start (Local Development)

### Step 1: Clone Repository

```bash
git clone https://github.com/raphaelmansuy/k8s-agent-stack.git
cd k8s-agent-stack
```

### Step 2: Start Kubernetes

**OrbStack (Recommended for macOS):**
- Open OrbStack → Settings → Kubernetes → Enable

**Verify cluster:**
```bash
kubectl cluster-info
```

### Step 3: Install Knative + Contour

```bash
./knative_orbstack.sh
```

This installs:
- **Knative Serving v1.20** - Scale-to-zero, auto-scaling
- **Contour + Envoy** - L7 ingress and routing
- **Magic DNS** - Automatic domain routing via sslip.io

⏱️ Installation time: ~3-5 minutes

### Step 4: Configure Knative for Local Images

```bash
# Allow local Docker images
kubectl patch configmap config-deployment -n knative-serving \
  --type merge \
  -p '{"data":{"registries-skipping-tag-resolving":"kind.local,ko.local,dev.local,docker.io/library"}}'
```

### Step 5: Install kagent (Agent Orchestration)

Kagent manages the lifecycle of your agents on Kubernetes. For more details on how the controller and runtime work, see the [Agent Runtime Architecture](architecture/agent-runtime.md).

```bash
# Set your OpenAI API key
export OPENAI_API_KEY="sk-your-key-here"

# Create namespace and secret
kubectl create namespace kagent --dry-run=client -o yaml | kubectl apply -f -
kubectl create secret generic openai-api-key \
  --from-literal=api-key="$OPENAI_API_KEY" \
  -n kagent --dry-run=client -o yaml | kubectl apply -f -

# Install kagent CRDs
helm install kagent-crds oci://ghcr.io/kagent-dev/kagent/helm/kagent-crds \
  --namespace kagent

# Install kagent controller + demo agents
helm install kagent oci://ghcr.io/kagent-dev/kagent/helm/kagent \
  --namespace kagent \
  --set providers.default=openAI \
  --set providers.openAI.apiKey=$OPENAI_API_KEY
```

### Step 6: Verify Installation

```bash
# Check all components
kubectl get pods -n knative-serving
kubectl get pods -n projectcontour
kubectl get pods -n kagent

# Check agents
kubectl get agents -n kagent
```

Expected output:
```
NAME                             TYPE          READY   ACCEPTED
google-adk-byo-agent             BYO           True    True
k8s-agent                        Declarative   True    True
helm-agent                       Declarative   True    True
... (10+ demo agents)
```

### Step 7: Access kagent UI

The platform includes the official Kagent Web UI for managing agents and chatting.

```bash
# Build the agentctl CLI
make -C agentstack build-cli

# Start the UI with automatic port-forwarding
./agentstack/bin/agentctl ui --port 3000
```

Then open: **`http://localhost:3000`**

---

## Deploy Your First Agent

### Option A: Deploy Pre-built ADK Agent

```bash
# Build the agent image
cd kagent-adk-agent
docker build -t dev.local/kagent-adk-agent:v30 .

# Deploy using kagent CRD
kubectl apply -f kagent-deployment.yaml

# Verify
kubectl get agents -n kagent | grep google-adk
```

### Option B: Deploy via Knative Service

```bash
# Deploy using Knative
kubectl apply -f kagent-setup.yaml

# Get service URL
kubectl get ksvc -n kagent
```

### Test Your Agent

```bash
# Port-forward to the agent
kubectl port-forward -n kagent svc/google-adk-byo-agent 8081:8080 &

# Test health
curl http://localhost:8081/health
# Output: {"status":"healthy"}

# Test A2A endpoint (JSON-RPC)
curl -X POST http://localhost:8081/ \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "id": "1",
    "params": {
      "message": {
        "contextId": "test-session",
        "parts": [{"kind": "text", "text": "What is the weather in San Francisco?"}]
      }
    }
  }'
```

---

## What's Installed

| Component | Namespace | Purpose |
|-----------|-----------|---------|
| **Knative Serving** | knative-serving | Scale-to-zero, auto-scaling |
| **Contour + Envoy** | projectcontour | L7 ingress, load balancing |
| **kagent Controller** | kagent | Agent lifecycle management |
| **kagent UI** | kagent | Web dashboard |
| **Demo Agents** | kagent | k8s-agent, helm-agent, etc. |

---

## Next Steps

- [Architecture Overview](architecture.md) - Understand the 5-layer platform
- [Deployment Guide](deployment-guide.md) - Deploy custom agents
- [Quick Reference](quick-reference.md) - Common commands
- [Troubleshooting](troubleshooting.md) - Debug issues

---

## Production Deployment

For GKE, EKS, AKS, or on-premises clusters:

1. Configure `kubectl` to your production cluster
2. Run the same installation steps
3. Configure a real domain (instead of sslip.io)
4. Set up TLS with cert-manager
5. Configure resource limits and RBAC

See [Deployment Guide](deployment-guide.md#production-deployment) for details.

---

[← Back to README](../README.md) | [Architecture →](architecture.md)
