# Complete Setup Guide: Knative Serving + kagent on OrbStack

**Last Updated:** December 16, 2025  
**Target Environment:** OrbStack (macOS ARM64)  
**Tested Version:** Knative v1.20.0, kagent CRDs

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start (5 minutes)](#quick-start-5-minutes)
3. [Troubleshooting](#troubleshooting)
4. [Advanced Configuration](#advanced-configuration)
5. [Architecture Overview](#architecture-overview)

---

## Prerequisites

Before starting, ensure you have:

- **OrbStack** installed and running with Kubernetes enabled
  - Download: https://orbstack.dev
  - Ensure `kubectl` is connected: `kubectl cluster-info`
- **Homebrew** for package management
- **kn CLI** (will be installed automatically during setup)
- **helm** (will be installed automatically during setup)

**Verify prerequisites:**

```bash
# Check Kubernetes connection
kubectl cluster-info

# Check if OrbStack node is ready
kubectl get nodes
```

---

## Quick Start (5 minutes)

### 1. Clone and Navigate

```bash
cd /path/to/cloudrun-like
```

### 2. Run Setup

```bash
make setup
```

This single command will:
- ✅ Verify Kubernetes cluster
- ✅ Install `kn` CLI if needed
- ✅ Install Knative Serving (Contour ingress, autoscaler, etc.)
- ✅ Install kagent CRDs
- ✅ Deploy sample services (`hello`, `nginx`)
- ✅ Verify everything is working

### 3. Verify Installation

```bash
make verify
```

Expected output:
```
Knative Serving:
  ✓ Namespace exists
  ✓ Pods running

Contour/Envoy:
  ✓ Namespace exists
  ✓ Envoy IP assigned

kagent:
  ✓ Namespace exists
  ✓ CRDs installed

Knative Services:
  ✓ 2 service(s) ready
```

### 4. Test the Sample Service

**From inside your OrbStack VM or via port-forward:**

```bash
# Get the Envoy IP
ENVOY_IP=$(kubectl get svc envoy -n projectcontour -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test the hello service
curl http://hello.default.${ENVOY_IP}.sslip.io

# Expected: HTML response with "Congratulations" message
```

**From macOS host (via port-forward):**

If the sslip.io URL is not reachable from your Mac, use port-forward:

```bash
# In one terminal, start port-forward
kubectl port-forward -n default service/hello-00001-private 8080:80

# In another terminal, test
curl http://localhost:8080

# Expected: HTML response with "Congratulations" message
```

---

## Troubleshooting

### Issue 1: `Envoy IP pending` or LoadBalancer not getting an IP

**Symptom:**
```
⚠ Envoy IP pending (may take a moment)
```

**Solution:**

1. **Check Envoy pods are running:**
   ```bash
   kubectl get pods -n projectcontour -l app=envoy
   ```
   Expected: 1 pod in `Running` state

2. **If pod is Pending, check node capacity:**
   ```bash
   kubectl describe pod -n projectcontour -l app=envoy
   ```
   Look for port conflicts or insufficient resources.

3. **Force cleanup and restart:**
   ```bash
   # Force-delete old Envoy pods
   kubectl delete pod -n projectcontour -l app=envoy --force --grace-period=0
   
   # Wait for new pod
   kubectl wait --for=condition=Ready pod -l app=envoy -n projectcontour --timeout=60s
   
   # Re-run setup
   make setup
   ```

### Issue 2: Knative services not becoming Ready

**Symptom:**
```
hello: ⚠ Service 'hello' may not be ready
```

**Solution:**

1. **Check Envoy status first** (Issue 1 above)

2. **Check Knative controller logs:**
   ```bash
   kubectl logs -n knative-serving deploy/controller --tail=50
   ```

3. **Describe the service:**
   ```bash
   kubectl describe ksvc hello -n default
   ```
   Look for `Conditions` → `Ready` reason.

4. **Re-run setup:**
   ```bash
   make clean
   make setup
   ```

### Issue 3: Services not reachable from macOS host

**Symptom:**
```
curl http://hello.default.192.168.139.2.sslip.io
# Timeout or connection refused
```

**Root Cause:**
OrbStack's networking may require port-forward from host or SSH tunnel for external access.

**Workarounds:**

**Option A: Use port-forward (simplest)**
```bash
kubectl port-forward -n default service/hello-00001-private 8080:80 &
curl http://localhost:8080
```

**Option B: SSH into OrbStack VM and test from inside**
```bash
# Find OrbStack VM SSH credentials in OrbStack app settings
ssh user@orbstack_ip
curl http://hello.default.192.168.139.2.sslip.io
```

**Option C: Configure OrbStack network for external routing**
- See "Advanced Configuration" section below

### Issue 4: kagent CRDs not found

**Symptom:**
```
✗ CRDs not found
```

**Solution:**

1. **Manually install kagent CRDs:**
   ```bash
   # Add Helm repo
   helm repo add kagent oci://ghcr.io/kagent-dev/kagent/helm
   helm repo update
   
   # Install CRDs
   helm install kagent-crds kagent/kagent-crds -n kagent --create-namespace
   ```

2. **Verify:**
   ```bash
   kubectl get crds | grep kagent.dev
   ```

---

## Advanced Configuration

### Scaling and Autoscaler Tuning

The setup configures autoscaling with Cloud Run-like behavior:
- **Scale-to-zero grace period:** 6 seconds (fast cold starts)
- **Container concurrency target:** 100 requests per pod
- **Panic window:** 6 seconds (quick scale-up on traffic spike)

**To modify scaling:**

```bash
# Edit autoscaler config
kubectl edit cm config-autoscaler -n knative-serving

# Key fields:
# - enable-scale-to-zero: "true"          (allow scaling to zero)
# - scale-to-zero-grace-period: "6s"      (time to wait before scaling)
# - container-concurrency-target-default: "100"
# - target-burst-capacity: "200"
# - panic-window: "6s"
```

### Installing Full kagent Controller (with API credentials)

The basic setup installs kagent CRDs only. To use kagent for agent management, install the full controller:

```bash
# Set your OpenAI API key
export OPENAI_API_KEY="sk-..."

# Add and update Helm repo
helm repo add kagent oci://ghcr.io/kagent-dev/kagent/helm
helm repo update

# Install controller
helm install kagent kagent/kagent -n kagent \
  --set providers.default=openai \
  --set providers.openai.apiKey=$OPENAI_API_KEY
```

Then deploy an agent:

```bash
kubectl apply -f kagent-adk-agent-declarative.yaml
```

### Warm Pods (No Scale-to-Zero)

To prevent cold starts, keep pods warm (always running):

```bash
# Use dedicated setup target
make setup-warm
```

Or manually:
```bash
kn service update hello --min-scale=1 --max-scale=10
```

### Metrics and Monitoring

Install Prometheus and Grafana for monitoring:

```bash
# Install via Helm
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/prometheus -n monitoring --create-namespace
```

---

## Architecture Overview

### Component Layout

```
┌─────────────────────────────────────────────────────────────┐
│                   OrbStack VM (Linux/ARM64)                 │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Kubernetes Cluster                      │  │
│  │                                                      │  │
│  │  projectcontour/          knative-serving/          │  │
│  │  ┌─────────────┐          ┌──────────────────┐      │  │
│  │  │ Contour     │◄─────────│ net-contour      │      │  │
│  │  │ (control)   │ HTTPProxy│ controller       │      │  │
│  │  └─────┬───────┘          └──────────────────┘      │  │
│  │        │                                             │  │
│  │        │ programs              kagent/              │  │
│  │        ▼                        ┌────────┐          │  │
│  │  ┌─────────────┐               │ CRDs   │          │  │
│  │  │ Envoy       │               │ (Agent │          │  │
│  │  │ (data plane)│               │  types)│          │  │
│  │  │ LB:80,443   │               └────────┘          │  │
│  │  └─────┬───────┘                                    │  │
│  │        │ routes by Host                            │  │
│  │        ├─────────────────────────────┬─────────────┤  │
│  │        ▼                             ▼             │  │
│  │  ┌──────────────────────────────────────────────┐  │  │
│  │  │    Knative Revision Pods                     │  │  │
│  │  │  [queue-proxy] ──► [user-container]          │  │  │
│  │  │                                              │  │  │
│  │  │  Metrics & concurrency handled by proxy      │  │  │
│  │  └──────────────────────────────────────────────┘  │  │
│  │                                                      │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
         ▲
         │ Native bridge network
         │ (OrbStack handles port mapping)
         │
    [macOS Host]
```

### Data Flow

1. **Request arrives** at Envoy (LoadBalancer IP)
2. **Host header routing** via HTTPProxy (Contour) → routes to Knative Service
3. **Activator** wakes up pods if scaled to zero
4. **Queue Proxy** sidecar handles metrics, concurrency limits, request queuing
5. **Your container** receives request on port 8080 (standard Knative)
6. **Response** flows back through reverse path

---

## Verification Checklist

After setup, verify each component:

```bash
# 1. Kubernetes connectivity
kubectl get nodes

# 2. Knative system
kubectl get ns knative-serving
kubectl get pods -n knative-serving

# 3. Contour/Envoy
kubectl get ns projectcontour
kubectl get pods -n projectcontour
kubectl get svc envoy -n projectcontour

# 4. kagent
kubectl get ns kagent
kubectl get crds | grep kagent.dev

# 5. Sample services
kubectl get ksvc
kn service list

# 6. Test connectivity
ENVOY_IP=$(kubectl get svc envoy -n projectcontour -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Envoy IP: $ENVOY_IP"
curl http://hello.default.${ENVOY_IP}.sslip.io || echo "(Try port-forward if this fails)"
```

---

## Common Commands

```bash
# Deploy a service
kn service create myapp --image=nginx:latest --port=80

# Update a service
kn service update myapp --image=nginx:1.25 --env VAR=value

# Scale configuration
kn service update myapp --min-scale=0 --max-scale=100

# Delete a service
kn service delete myapp

# View logs
kubectl logs -n knative-serving deploy/autoscaler
kubectl logs -n projectcontour -l app=envoy

# Port-forward for testing
kubectl port-forward -n default service/myapp 8080:80

# Watch autoscaling
watch -n 1 'kubectl get pods -n default -o wide'
```

---

## Support & Documentation

- **Knative Docs:** https://knative.dev/docs
- **Contour Docs:** https://projectcontour.io
- **OrbStack:** https://orbstack.dev
- **kagent Docs:** https://github.com/kagent-dev/kagent

---

## License

Licensed under the Apache License, Version 2.0.  
See [LICENSE](LICENSE) file for details.
