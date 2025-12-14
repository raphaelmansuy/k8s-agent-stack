# Troubleshooting Guide

Quick solutions for common issues with k8s-agent-stack.

## Quick Diagnostics

Run the built-in diagnostics:

```bash
./knative_orbstack.sh --debug
```

Expected output:
```
✓ Knative Serving: Ready
✓ Contour: Ready
✓ Envoy: Ready
✓ DNS: Configured
```

---

## Common Issues

| Issue | Symptoms | Solution |
|-------|----------|----------|
| Agent not responding | HTTP 503 errors | Check pod status |
| Cold start too slow | First request >10s | Reduce image size or set `scale-min=1` |
| Ingress not working | Can't reach agent URL | Check Envoy external IP |
| Scaling not working | Pods don't scale | Check autoscaler logs |
| Agent crashes | CrashLoopBackOff | Check previous pod logs |
| Image pull errors | ImagePullBackOff | Verify image exists |
| DNS resolution fails | nslookup errors | Check CoreDNS |

---

## Debug Workflow

```
Problem: Agent not responding
         │
    ┌────▼────┐
    │ 1. Check│  kn service list
    │ Service │
    └────┬────┘
         │
    ┌────▼────┐
    │ 2. Check│  kubectl get pods -n kagent
    │ Pods    │
    └────┬────┘
         │
    ┌────▼────┐
    │ 3. Check│  kubectl logs <pod> -n kagent
    │ Logs    │
    └────┬────┘
         │
    ┌────▼────┐
    │ 4. Check│  kubectl describe pod <pod>
    │ Events  │
    └────┬────┘
         │
    ┌────▼────┐
    │ 5. Fix  │  kn service update my-agent ...
    │ Redeploy│
    └─────────┘
```

---

## Detailed Commands

### Service Health

```bash
# List all Knative services
kn service list
kubectl get ksvc -n kagent

# Describe specific service
kn service describe my-agent
kubectl describe ksvc my-agent -n kagent
```

### Pod Status

```bash
# Get pods
kubectl get pods -n kagent

# Describe pod for events
kubectl describe pod <pod-name> -n kagent

# Check resource usage
kubectl top pods -n kagent
```

### Logs

```bash
# Stream logs
kubectl logs -f -n kagent deployment/google-adk-agent

# Get logs with label
kubectl logs -n kagent -l app.kubernetes.io/name=google-adk-agent --tail=100

# Previous crash logs
kubectl logs -n kagent <pod-name> --previous

# Export to file
kubectl logs -n kagent deployment/google-adk-agent --tail=500 > agent-logs.txt
```

### Knative Components

```bash
# Check Knative pods
kubectl get pods -n knative-serving

# Controller logs
kubectl logs -n knative-serving deploy/controller --tail=50

# Autoscaler logs
kubectl logs -n knative-serving deploy/autoscaler --tail=50

# Activator logs
kubectl logs -n knative-serving deploy/activator --tail=50
```

### Ingress

```bash
# Check Contour services
kubectl get svc -n projectcontour

# Contour logs
kubectl logs -n projectcontour deploy/contour --tail=50

# Get external IP
kubectl get svc envoy -n projectcontour
```

### Connectivity

```bash
# Test from inside cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- sh

# Inside pod:
curl http://my-agent.kagent.svc.cluster.local
```

### Events

```bash
# Recent events in namespace
kubectl get events -n kagent --sort-by='.lastTimestamp'

# All events
kubectl get events --all-namespaces --sort-by='.lastTimestamp' | head -20
```

---

## Specific Issues

### Agent Not Responding (503)

1. **Check if pods are running:**
   ```bash
   kubectl get pods -n kagent
   ```

2. **If no pods, check service:**
   ```bash
   kn service describe my-agent
   ```

3. **If pods are pending, check events:**
   ```bash
   kubectl describe pod <pod-name> -n kagent
   ```

4. **Check for resource constraints:**
   ```bash
   kubectl top nodes
   kubectl describe nodes | grep -A 5 "Allocated resources"
   ```

### Cold Start Too Slow

1. **Check cold start time:**
   ```bash
   time curl $(kn service describe my-agent -o url)/health
   ```

2. **Solutions:**
   - Reduce image size (use `python:3.12-slim` or `distroless`)
   - Keep warm pods: `kn service update my-agent --scale-min=1`
   - Optimize imports in your code

3. **Check image size:**
   ```bash
   docker images my-agent
   ```

### CrashLoopBackOff

1. **Get crash logs:**
   ```bash
   kubectl logs -n kagent <pod-name> --previous
   ```

2. **Check events:**
   ```bash
   kubectl describe pod <pod-name> -n kagent | grep -A 10 Events
   ```

3. **Common causes:**
   - Missing environment variables
   - Invalid configuration
   - Out of memory (check resource limits)
   - Port mismatch

### ImagePullBackOff

1. **Verify image exists:**
   ```bash
   docker pull <image>
   ```

2. **For private registries, check secrets:**
   ```bash
   kubectl get secrets -n kagent
   kubectl describe pod <pod-name> -n kagent | grep -A 5 "Image"
   ```

3. **Create pull secret if needed:**
   ```bash
   kubectl create secret docker-registry regcred \
     --docker-server=<registry> \
     --docker-username=<username> \
     --docker-password=<password> \
     -n kagent
   ```

### Scaling Issues

1. **Check autoscaler:**
   ```bash
   kubectl logs -n knative-serving deploy/autoscaler --tail=50
   ```

2. **Check metrics:**
   ```bash
   kubectl get pods -n kube-system | grep metrics-server
   kubectl top pods -n kagent
   ```

3. **If metrics-server missing:**
   ```bash
   kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
   ```

### DNS Resolution Fails

1. **Check CoreDNS:**
   ```bash
   kubectl get pods -n kube-system -l k8s-app=kube-dns
   kubectl logs -n kube-system -l k8s-app=kube-dns --tail=20
   ```

2. **Test DNS from inside cluster:**
   ```bash
   kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup my-agent.kagent.svc.cluster.local
   ```

---

## Performance Optimization

### Monitor Scaling

```bash
# Watch pods scale
watch 'kubectl get pods -n kagent -o wide'

# Check HPA (if using HPA)
kubectl get hpa -n kagent
```

### Adjust Autoscaling

```bash
kn service update my-agent \
  --scale-min=1 \              # Keep 1 warm
  --scale-max=10 \             # Max 10 replicas
  --concurrency-target=10 \    # 10 requests per pod
  --concurrency-limit=50       # Max 50 requests per pod
```

### Resource Optimization

```bash
# Check current resource usage
kubectl top pod -n kagent

# Update resources
kubectl patch deployment my-agent -n kagent --type='json' -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value": "2Gi"}
]'
```

---

## Getting Help

If you can't resolve an issue:

1. **Run full diagnostics:**
   ```bash
   ./knative_orbstack.sh --debug > diagnostics.txt 2>&1
   ```

2. **Collect logs:**
   ```bash
   kubectl logs -n kagent -l app.kubernetes.io/name=google-adk-agent --tail=500 > agent-logs.txt
   kubectl get events -n kagent --sort-by='.lastTimestamp' > events.txt
   ```

3. **Open an issue** with the diagnostic output:
   - [GitHub Issues](https://github.com/raphaelmansuy/k8s-agent-stack/issues)

---

## Related Docs

- [Getting Started](getting-started.md)
- [Deployment Guide](deployment-guide.md)
- [Glossary](glossary.md)

---

[← Back to Documentation Index](README.md) • [Architecture](architecture.md) • [Getting Started](getting-started.md) • [Main README](../README.md)
