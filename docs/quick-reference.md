# Quick Reference

Common commands and workflows for k8s-agent-stack.

## Essential Commands

### Installation & Setup

```bash
# Install Knative + Contour
./knative_orbstack.sh

# Install kagent
helm install kagent-crds oci://ghcr.io/kagent-dev/kagent/helm/kagent-crds -n kagent
helm install kagent oci://ghcr.io/kagent-dev/kagent/helm/kagent -n kagent \
  --set providers.default=openAI \
  --set providers.openAI.apiKey=$OPENAI_API_KEY

# Verify installation
make verify
```

### Agent Management

```bash
# List all agents
kubectl get agents -n kagent

# Get agent details
kubectl describe agent <agent-name> -n kagent

# View agent logs
kubectl logs -n kagent -l app.kubernetes.io/name=<agent-name> -f

# Delete an agent
kubectl delete agent <agent-name> -n kagent
```

### Port Forwarding

```bash
# AgentStack UI (Recommended)
agentctl ui

# Manual kagent UI (web dashboard)
kubectl port-forward -n agentstack svc/agentstack-ui 3000:3000 8080:8080 8083:8083 8081:8081

# Agent endpoint
kubectl port-forward -n kagent svc/<agent-name> 8081:8080
```

### Health Checks

```bash
# Check all pods
kubectl get pods -n kagent

# Check Knative services
kubectl get ksvc --all-namespaces

# Run diagnostics
./knative_orbstack.sh --debug

# Check agent readiness
kubectl get agents -n kagent -o wide
```

---

## Makefile Targets

```bash
# Installation
make setup              # Install Knative + kagent
make install            # Install required CLI tools
make verify             # Verify installation status

# Agent Operations
make deploy             # Deploy example agent
make undeploy           # Remove example agent
make list-agents        # List all deployed agents
make agent-status       # Check agent status
make agent-logs         # View agent logs

# Development
make build              # Build agent Docker image
make build-dev          # Build development image
make test               # Run all tests

# Portal Access
make portal-access      # Port-forward kagent UI

# Cleanup
make clean              # Remove all agents
make clean-all          # Full cleanup
```

---

## Agent Deployment

### Deploy BYO Agent

```bash
# 1. Build image
cd kagent-adk-agent
docker build -t dev.local/my-agent:v1 .

# 2. Create Agent CRD
cat <<EOF | kubectl apply -f -
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: my-agent
  namespace: kagent
spec:
  type: BYO
  byo:
    deployment:
      image: dev.local/my-agent:v1
      imagePullPolicy: IfNotPresent
EOF

# 3. Verify
kubectl get agents -n kagent
```

### Deploy Knative Service

```bash
# Create Knative service
kubectl apply -f kagent-setup.yaml

# Check status
kubectl get ksvc -n kagent

# Get URL
kubectl get ksvc <service-name> -n kagent -o jsonpath='{.status.url}'
```

---

## Agent Testing

```bash
# Health check
curl http://localhost:8081/health

# A2A message (SSE streaming)
curl -X POST http://localhost:8081/ \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "id": "1",
    "params": {
      "message": {
        "contextId": "test",
        "parts": [{"kind": "text", "text": "Hello, agent!"}]
      }
    }
  }'
```

---

## Scaling Operations

```bash
# Keep agent warm (no scale-to-zero)
kubectl annotate ksvc <service-name> -n kagent \
  autoscaling.knative.dev/min-scale="1" --overwrite

# Allow scale-to-zero
kubectl annotate ksvc <service-name> -n kagent \
  autoscaling.knative.dev/min-scale="0" --overwrite

# Set max scale
kubectl annotate ksvc <service-name> -n kagent \
  autoscaling.knative.dev/max-scale="10" --overwrite

# Watch pods scale
watch 'kubectl get pods -n kagent'
```

---

## Troubleshooting

```bash
# Pod issues
kubectl describe pod <pod-name> -n kagent
kubectl logs <pod-name> -n kagent --previous

# Knative issues
kubectl logs -n knative-serving deploy/controller --tail=50
kubectl logs -n knative-serving deploy/autoscaler --tail=50

# Ingress issues
kubectl logs -n projectcontour deploy/contour --tail=50
kubectl get svc envoy -n projectcontour

# Events
kubectl get events -n kagent --sort-by='.lastTimestamp'
```

---

## Configuration

### OpenAI API Key

```bash
# Update API key
kubectl create secret generic openai-api-key \
  --from-literal=api-key="$OPENAI_API_KEY" \
  -n kagent --dry-run=client -o yaml | kubectl apply -f -
```

### ModelConfig

```bash
# View current config
kubectl get modelconfigs -n kagent

# View details
kubectl describe modelconfig default-model-config -n kagent
```

---

## Access Points

| Service | Command | URL |
|---------|---------|-----|
| **kagent UI** | `kubectl port-forward -n kagent svc/kagent-ui 8080:8080` | http://localhost:8080 |
| **Controller API** | `kubectl port-forward -n kagent svc/kagent-controller 8083:8083` | http://localhost:8083/api |
| **Agent (example)** | `kubectl port-forward -n kagent svc/google-adk-byo-agent 8081:8080` | http://localhost:8081 |

---

## Resource Queries

```bash
# All kagent resources
kubectl get agents,modelconfigs,toolservers,memories -n kagent

# Knative services
kubectl get ksvc,revisions,configurations -n kagent

# All pods with resource usage
kubectl top pods -n kagent
```

---

[← Getting Started](getting-started.md) | [Troubleshooting →](troubleshooting.md)
