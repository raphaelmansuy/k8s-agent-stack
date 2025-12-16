# Google ADK Agent - Local Access Guide

## Quick Start

### 1. Agent Service URL
```
http://google-adk-agent.kagent.192.168.139.2.sslip.io
```

### 2. Local Port-Forward (Recommended)

**Start port-forward in one terminal:**
```bash
make port-forward
# OR manually:
kubectl port-forward -n kagent svc/google-adk-agent-00001-private 8080:80
```

**Test in another terminal:**
```bash
# Simple GET request
curl http://localhost:8080

# Expected: HTML "Congratulations" page (from the Cloud Run hello image)
```

### 3. Makefile Targets

```bash
# Deploy agent
make deploy

# Check agent status
make agent-status

# View agent logs
make agent-logs

# Port-forward for local testing
make port-forward

# Test agent (requires port-forward running)
make test-agent

# Describe agent pod details
make agent-describe
```

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│          Your macOS Host                        │
│                                                 │
│  Browser/curl → localhost:8080                  │
│      │                                           │
│      └─ kubectl port-forward ────┐              │
│                                  │              │
└──────────────────────────────────┼──────────────┘
                                   │
                      OrbStack Network Bridge
                                   │
                                   ▼
                    ┌──────────────────────────┐
                    │   OrbStack VM (Linux)    │
                    │                          │
                    │  kagent namespace        │
                    │  ┌────────────────────┐  │
                    │  │ google-adk-agent   │  │
                    │  │ (Knative Service)  │  │
                    │  │                    │  │
                    │  │ ┌──────────────┐   │  │
                    │  │ │ Pod (hello)  │   │  │
                    │  │ │ :8080        │   │  │
                    │  │ └──────────────┘   │  │
                    │  └────────────────────┘  │
                    │                          │
                    └──────────────────────────┘
```

---

## Current Configuration

**Service:** `google-adk-agent`
**Namespace:** `kagent`
**Image:** `us-docker.pkg.dev/cloudrun/container/hello:latest`
**Port:** `8080`
**Environment Variables:** `OPENAI_API_KEY` (via secret)
**Min Replicas:** 1 (warm pod)
**Max Replicas:** 10 (autoscale)

---

## Troubleshooting

### Service not Ready
```bash
# Check pod status
kubectl get pods -n kagent

# Check logs
kubectl logs -n kagent <pod-name> -c user-container

# Describe route conditions
kubectl describe route google-adk-agent -n kagent
```

### Port-Forward Issues
```bash
# Kill existing port-forward
killall kubectl

# Try again
kubectl port-forward -n kagent svc/google-adk-agent-00001-private 8080:80
```

### Access Denied
Ensure the OpenAI API key secret exists:
```bash
kubectl get secrets -n kagent
```

---

## Next Steps

1. **Deploy your own ADK agent:** Modify `kagent-setup.yaml` with your custom image
2. **Add tools/skills:** Configure agent capabilities via kagent CRDs when available
3. **Scale configuration:** Adjust `min-scale` and `max-scale` annotations in kagent-setup.yaml
4. **Monitor:** Use `kubectl logs` and `make agent-logs` to watch agent activity

---

## Links

- [kagent Documentation](https://github.com/kagent-dev/kagent)
- [Knative Serving](https://knative.dev)
- [Google Cloud Run (ADK Agent)](https://cloud.google.com/products/agents)
