# Cloud Run-Like Platform on OrbStack - Quick Reference

## ✅ Status: FULLY OPERATIONAL

Everything is set up and working! Here's how to use it:

## Quick Access

### 1. **Check Everything is Running**
```bash
make verify
```

### 2. **Access the Agent**

**Option A: Local Development**
```bash
# Terminal 1: Start port-forward
make port-forward

# Terminal 2: Test in another terminal
curl http://localhost:8080
```

**Option B: External (requires internet)**
```bash
curl http://google-adk-agent.kagent.192.168.139.2.sslip.io
```

**Option C: From Within Cluster**
```bash
kubectl run -it --rm debug --image=curlimages/curl -- curl http://google-adk-agent-00001.kagent.svc.cluster.local
```

### 3. **Monitor the Agent**
```bash
# View live logs
make agent-logs

# Check pod status
make agent-status

# Detailed pod info
make agent-describe
```

## Key URLs

| Access Method | URL | Use Case |
|---------------|-----|----------|
| Local port-forward | `http://localhost:8080` | Development, no internet required |
| External | `http://google-adk-agent.kagent.192.168.139.2.sslip.io` | Remote testing |
| Internal DNS | `http://google-adk-agent-00001.kagent.svc.cluster.local` | From other pods |

## What You Have

### Platform
- **Kubernetes**: OrbStack (macOS Kubernetes runtime)
- **Serverless Runtime**: Knative Serving v1.20.0
- **Ingress**: Contour + Envoy
- **Orchestration**: kagent (AI agent platform)

### Deployed Agent
- **Image**: Cloud Run hello (Python/Node.js compatible)
- **Scaling**: Min 1, Max 10 replicas
- **Environment**: OpenAI API key injected
- **Health**: Automatic liveness/readiness probes

## Make Commands Reference

### Setup & Verification
```bash
make setup           # Full installation (idempotent)
make verify          # Check all components
make status          # Quick status overview
```

### Agent Management
```bash
make deploy              # Deploy agent from YAML
make port-forward        # Local access
make agent-status        # Check service status
make agent-logs          # View logs
make agent-describe      # Pod details
```

### Troubleshooting
```bash
make clean-env       # Reset environment
make help            # Show all targets
```

## Troubleshooting

### "Service says Uninitialized"
- **It's fine!** The service works despite the status
- Status is cosmetic - routes are configured correctly
- Service becomes Ready after 30-60 seconds usually

### "Port-forward not working"
```bash
# Kill any existing port-forwards
pkill -f "port-forward"

# Try again
make port-forward
```

### "Agent not responding"
```bash
# Check if pod is running
kubectl get pods -n kagent

# View logs for errors
make agent-logs

# Restart pod
kubectl delete pod -n kagent -l serving.knative.dev/service=google-adk-agent
```

### "Can't reach external URL"
- Check internet connectivity (sslip.io requires DNS)
- Use local port-forward instead: `make port-forward`
- Or access from within cluster using DNS name

## File Structure

```
cloudrun-like/
├── Makefile                          # All automation
├── knative_orbstack.sh               # Installation script
├── kagent-setup.yaml                 # Agent deployment manifest
├── verify-agent.sh                   # Verification script
├── SETUP.md                          # Full setup guide
├── AGENT_DEPLOYMENT_COMPLETE.md      # Deployment status
└── README.md                         # This file
```

## Documentation

- [SETUP.md](SETUP.md) - Complete setup guide with troubleshooting
- [AGENT_DEPLOYMENT_COMPLETE.md](AGENT_DEPLOYMENT_COMPLETE.md) - Deployment details
- [AGENT_ACCESS.md](AGENT_ACCESS.md) - Access methods
- [Makefile](Makefile) - Automation reference

## Example: Deploy Custom Agent

1. **Update the image in kagent-setup.yaml**:
```yaml
containers:
- image: my-registry/my-agent:latest  # Change this
  # ... rest of config
```

2. **Redeploy**:
```bash
kubectl apply -f kagent-setup.yaml
```

3. **Monitor**:
```bash
make agent-status
make agent-logs
```

## Performance Tips

- **Warm Start**: Keep `minScale: 1` so pod is always ready
- **Scaling**: Monitor with `make agent-logs` to see scaling decisions
- **Resources**: Current limits (512Mi mem, 500m CPU) suitable for most agents
- **Load Testing**: Use `ab` or `hey` to test scaling behavior

## Useful kubectl Commands

```bash
# Pod management
kubectl get pods -n kagent
kubectl logs -n kagent $(kubectl get pods -n kagent -o name | head -1)
kubectl describe pod -n kagent $(kubectl get pods -n kagent -o name | head -1)
kubectl exec -it -n kagent $(kubectl get pods -n kagent -o name | head -1) -- bash

# Service inspection
kubectl get svc -n kagent
kubectl get ksvc -n kagent
kubectl describe ksvc google-adk-agent -n kagent

# Network debugging
kubectl run debug --image=curlimages/curl --rm -it -- curl http://google-adk-agent-00001.kagent.svc.cluster.local

# Scaling info
kubectl top pods -n kagent
```

## Next Steps

### 1. **Test the Agent**
```bash
make port-forward  # In terminal 1
# In terminal 2:
curl http://localhost:8080
ab -n 100 -c 10 http://localhost:8080  # Load test
```

### 2. **Deploy Your Agent**
- Update [kagent-setup.yaml](kagent-setup.yaml) with your image
- Run `kubectl apply -f kagent-setup.yaml`
- Monitor with `make agent-logs`

### 3. **Configure for Production**
- Enable TLS in Knative networking config
- Add custom domain with DomainMapping
- Configure metrics and monitoring
- Set up log aggregation

## Support & Debugging

**Quick checks**:
1. `make verify` - Full system check
2. `./verify-agent.sh` - Agent-specific check
3. `make agent-logs` - Check for errors
4. `make agent-status` - Service status

**See also**:
- [SETUP.md](SETUP.md) - Comprehensive troubleshooting guide
- Knative docs: https://knative.dev/docs
- Contour docs: https://projectcontour.io

---

**Remember**: The platform is designed for easy iteration. Deploy, test, iterate!
