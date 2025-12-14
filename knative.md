# Cloud Run on Kubernetes: The Complete Guide (2025)

<!--
Copyright 2025 Raphaël MANSUY
Licensed under the Apache License, Version 2.0
https://www.apache.org/licenses/LICENSE-2.0
-->

## Why This Exists

**The Problem**: Cloud Run is brilliant — serverless containers, scale-to-zero, pay-per-request, zero ops. But you're locked to Google Cloud and proprietary APIs.

**The Solution**: Knative Serving is Cloud Run's open-source foundation. Same team, same concepts, same behavior, but runs anywhere: your laptop, on-prem, any cloud.

**This Guide**: Install a production-grade "Cloud Run clone" on OrbStack (local macOS K8s) in <5 minutes. Get identical semantics for local development, then deploy the same manifests to production clusters.

---

## What You Get

| Feature | Cloud Run (GCP) | Knative Serving (this setup) |
|---------|----------------|------------------------------|
| Scale-to-zero | ✓ (<2s) | ✓ (<2s) |
| Cold start (optimized) | ~50-150ms | ~100-250ms (local) |
| Concurrency control | ✓ (up to 1000) | ✓ (configurable) |
| Auto HTTPS/TLS | ✓ (managed certs) | ✓ (cert-manager) |
| Traffic splitting | ✓ (blue/green) | ✓ (same API) |
| CLI tool | `gcloud run` | `kn` (identical UX) |
| Cost (idle) | $0 | $0 |
| Vendor lock-in | Google only | Any K8s |

---

## Architecture: How Requests Become Pods

This diagram shows the **complete data plane + control plane flow** for Knative Serving on OrbStack:

```ascii
┌────────────────────────────────────────────────────────────────────────────┐
│                           YOUR REQUEST JOURNEY                              │
└────────────────────────────────────────────────────────────────────────────┘

  1. Client Request
     curl http://myapp.default.192.168.x.x.sslip.io
         │
         v
  ┌──────────────────────────────────────────────────────────────────────┐
  │  Envoy (Contour Data Plane)                                          │
  │  • LoadBalancer Service (OrbStack assigns real IP)                   │
  │  • Routes by Host header                                             │
  │  • L7 proxy with connection pooling                                  │
  └────────────────┬─────────────────────────────────────────────────────┘
                   │ 2. Route lookup via HTTPProxy
                   v
  ┌──────────────────────────────────────────────────────────────────────┐
  │  Contour Controller (Control Plane)                                  │
  │  • Watches HTTPProxy CRDs                                            │
  │  • Programs Envoy config (xDS protocol)                              │
  │  • Created by net-contour-controller                                 │
  └──────────────────────────────────────────────────────────────────────┘
                   │
                   v
  ┌──────────────────────────────────────────────────────────────────────┐
  │  Knative Serverless Service (SKS)                                    │
  │  • Intelligent routing layer                                         │
  │  • IF replicas > 0 → route to revision pods                          │
  │  • IF replicas = 0 → route to Activator                              │
  └────┬──────────────────────────────────────────────┬──────────────────┘
       │ 3a. Scale=0                                  │ 3b. Scale>0
       v                                              v
  ┌─────────────────────────┐              ┌──────────────────────────────┐
  │  Activator              │              │  Revision Pod                 │
  │  • Buffers request      │              │  ┌────────────────────────┐   │
  │  • Signals autoscaler   │              │  │ queue-proxy (sidecar)  │   │
  │  • Holds ~6s max        │              │  │ • Concurrency metrics  │   │
  │  • Forwards when ready  │──────────────┤  │ • Request queuing      │   │
  └─────────────────────────┘              │  └───────────┬────────────┘   │
                                           │              v                │
                                           │  ┌────────────────────────┐   │
                                           │  │  Your Container        │   │
                                           │  │  (app code)            │   │
                                           │  └────────────────────────┘   │
                                           └──────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────┐
│                          CONTROL PLANE (Reconcilers)                        │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  Knative Controller        Autoscaler              Webhook                │
│  • Watches Service CRD     • Monitors metrics      • Validates/defaults   │
│  • Creates:                • Computes desired      • Mutation on create   │
│    - Configuration           replicas (0..N)       • Version negotiation  │
│    - Route                 • Scales Deployment     • Resource quotas      │
│    - Revision              • Scale-to-zero timer   │                      │
│    - K8s Deployment        • Panic mode (burst)    │                      │
│    - K8s Service           │                       │                      │
│                            │                       │                      │
└────────────────────────────────────────────────────────────────────────────┘

KEY CONCEPTS:
• Envoy = Entry point (like Cloud Run's Google Front End)
• Activator = Cold-start buffer (holds requests during scale-from-zero)
• queue-proxy = Sidecar that enforces concurrency limits (Cloud Run does this internally)
• SKS = Smart router that switches between Activator and direct-to-pod routing
• Revision = Immutable snapshot of your code+config (like Cloud Run revision)
```

---

## Installation: Two Methods

### Method 1: Automated Script (Recommended)

```bash
# Clone this repo
git clone <your-repo>
cd cloudrun-like

# Run the installer
./knative_orbstack.sh

# With optional features
./knative_orbstack.sh --install-metrics --prepull --warm 1
```

**What it does**:
1. Installs Contour (ingress controller)
2. Installs Knative Serving v1.20.0
3. Patches Envoy for OrbStack (removes hostPort conflicts)
4. Configures sslip.io magic DNS
5. Creates sample services (hello + nginx)
6. Tests connectivity

**Duration**: ~3-5 minutes

---

### Method 2: Manual Step-by-Step

Use this if you want to understand each component or customize the setup.

#### Prerequisites

```bash
# Verify OrbStack is running with Kubernetes
kubectl cluster-info

# Install kn CLI (optional but recommended)
brew install knative/client/kn
```

#### Step 1: Install Contour (Ingress Controller)

**Why Contour?** It's the 2025 default for Knative. Faster than Kourier, more lightweight than Istio, production-ready.

```bash
# Install Contour + Envoy
kubectl apply -f https://projectcontour.io/quickstart/contour.yaml

# Wait for pods
kubectl wait --for=condition=Ready pods --all -n projectcontour --timeout=120s
```

**OrbStack-specific patch** (Envoy uses hostPort by default, which conflicts on single-node):

```bash
# Remove hostPort from Envoy DaemonSet
kubectl patch daemonset envoy -n projectcontour --type='json' \
  -p='[{"op":"remove","path":"/spec/template/spec/containers/1/ports/0/hostPort"},
       {"op":"remove","path":"/spec/template/spec/containers/1/ports/1/hostPort"},
       {"op":"remove","path":"/spec/template/spec/containers/1/ports/2/hostPort"}]'

# Restart Envoy pods
kubectl delete pods -n projectcontour -l app=envoy
```

---

#### Step 2: Install Knative Serving

```bash
# Install CRDs (Custom Resource Definitions)
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.20.0/serving-crds.yaml

# Install core Knative components (controller, webhook, autoscaler, activator)
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.20.0/serving-core.yaml

# Wait for all pods to be ready
kubectl wait --for=condition=Ready pods --all -n knative-serving --timeout=180s
```

**What you get:**
- `controller` — reconciles Service → Configuration → Revision → Deployment
- `webhook` — validates and defaults Knative resources
- `autoscaler` — computes desired replicas based on concurrency/RPS
- `activator` — buffers requests during scale-from-zero

---

#### Step 3: Install net-contour (Knative ↔ Contour Bridge)

```bash
# Install the integration layer
kubectl apply -f https://github.com/knative-extensions/net-contour/releases/download/knative-v1.20.0/net-contour.yaml

# Tell Knative to use Contour as the ingress
kubectl patch configmap/config-network -n knative-serving --type merge \
  -p '{"data":{"ingress.class":"contour.ingress.networking.knative.dev"}}'
```

**OrbStack-specific configuration** (Contour uses `projectcontour` namespace):

```bash
# Point Knative to the correct Envoy service
kubectl patch configmap/config-contour -n knative-serving --type merge -p '{
  "data": {
    "visibility": "ExternalIP:\n  class: contour\n  service: projectcontour/envoy\nClusterLocal:\n  class: contour-internal\n  service: projectcontour/envoy\n"
  }
}'
```

---

#### Step 4: Configure Magic DNS (sslip.io)

**Why?** Knative generates URLs like `myapp-default.example.com`. sslip.io gives you real DNS without configuration: `myapp.default.192.168.1.100.sslip.io` → resolves to `192.168.1.100`.

```bash
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.20.0/serving-default-domain.yaml
```

---

#### Step 5: Tune for Cloud Run Behavior

```bash
# Enable fast scale-to-zero
kubectl patch configmap/config-autoscaler -n knative-serving --type merge -p '{
  "data": {
    "enable-scale-to-zero": "true",
    "scale-to-zero-grace-period": "6s",
    "container-concurrency-target-default": "100",
    "target-burst-capacity": "200",
    "stable-window": "60s",
    "panic-window": "6s"
  }
}'
```

**Explanation:**
- `scale-to-zero-grace-period: 6s` — Wait 6s after last request before scaling to zero
- `container-concurrency-target-default: 100` — Target 100 concurrent requests per pod
- `panic-window: 6s` — React to traffic spikes within 6s (vs 60s stable window)

---

## Deploy Your First Service

### Using kn CLI (Cloud Run-style)

```bash
# Deploy a service
kn service create myapp \
  --image=nginx \
  --port=80 \
  --scale=0..10 \
  --concurrency-limit=80

# Get URL
kn service describe myapp -o url

# Test
curl $(kn service describe myapp -o url)

# Update with env vars
kn service update myapp \
  --env DATABASE_URL=postgres://... \
  --env CACHE_TTL=300

# Traffic splitting (blue/green)
kn service update myapp --tag=@latest=stable
kn service update myapp --traffic stable=90,@latest=10  # 90% stable, 10% canary
```

---

### Using kubectl (YAML)

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: myapp
  namespace: default
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/min-scale: "0"
        autoscaling.knative.dev/max-scale: "10"
        autoscaling.knative.dev/target: "100"
    spec:
      containers:
      - image: nginx
        ports:
        - containerPort: 80
        env:
        - name: DATABASE_URL
          value: postgres://...
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
```

```bash
kubectl apply -f service.yaml
kubectl get ksvc myapp  # Check status
```

---

## Key Concepts Explained

### Revision (Immutable Snapshot)

Every time you update a Knative Service, a new **Revision** is created:

```bash
# Deploy v1
kn service create myapp --image=myapp:v1

# Update to v2 (creates revision myapp-00002)
kn service update myapp --image=myapp:v2

# List revisions
kn revision list

# Rollback (pin traffic to old revision)
kn service update myapp --traffic myapp-00001=100
```

**Cloud Run equivalent**: `gcloud run deploy` creates a new revision each time.

---

### Autoscaling: How It Works

```ascii
┌─────────────────────────────────────────────────────────────────────┐
│                     AUTOSCALING ALGORITHM                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. queue-proxy reports metrics every 1s:                          │
│     • concurrentRequests (in-flight)                               │
│     • requestCount (total)                                         │
│                                                                     │
│  2. Autoscaler aggregates across all pods:                         │
│     avgConcurrency = sum(concurrentRequests) / numPods            │
│                                                                     │
│  3. Compute desired replicas:                                      │
│     desired = ceil(avgConcurrency / target)                        │
│     where target = 100 (default)                                   │
│                                                                     │
│  4. Apply bounds:                                                  │
│     desired = max(minScale, min(desired, maxScale))               │
│                                                                     │
│  5. Panic mode (burst protection):                                 │
│     IF avgConcurrency > target * 2 in 6s window                   │
│     THEN scale immediately (don't wait for stable window)         │
│                                                                     │
│  6. Scale-to-zero:                                                 │
│     IF avgConcurrency = 0 for grace-period (6s)                   │
│     THEN scale to 0, route traffic to Activator                   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Configure:**

```bash
# Via annotation
kn service update myapp \
  --annotation autoscaling.knative.dev/target=50 \
  --annotation autoscaling.knative.dev/metric=rps  # or 'concurrency'

# Via ConfigMap (global default)
kubectl patch cm config-autoscaler -n knative-serving --type merge -p '{
  "data": {"container-concurrency-target-default": "50"}
}'
```

---

### Scale-to-Zero: Cold Start Flow

```ascii
┌────────────────────────────────────────────────────────────────────┐
│                  COLD START SEQUENCE (scale=0 → 1)                 │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  t=0ms    Client → Envoy → SKS (sees scale=0) → Activator         │
│           Activator accepts connection, signals autoscaler         │
│                                                                    │
│  t=10ms   Autoscaler → K8s API: patch Deployment replicas=1       │
│                                                                    │
│  t=50ms   Kubelet pulls image (if not cached)                     │
│                                                                    │
│  t=120ms  Container starts, queue-proxy reports ready             │
│                                                                    │
│  t=130ms  SKS updates endpoints, routes traffic directly          │
│           Activator forwards buffered request                     │
│                                                                    │
│  t=150ms  Response returned to client                             │
│                                                                    │
│  TOTAL: ~150ms (local), ~300-500ms (cloud with image pull)        │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

**Optimize cold starts:**

```yaml
# Pre-pull images
imagePullPolicy: Always  # or IfNotPresent if pre-pulled

# Keep 1 pod warm
autoscaling.knative.dev/min-scale: "1"

# Use smaller base images
FROM gcr.io/distroless/static:nonroot  # ~2MB vs 100MB+
```

---

## Production Hardening

### 1. Add TLS with cert-manager

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml

# Install Knative cert-manager integration
kubectl apply -f https://github.com/knative/net-certmanager/releases/download/knative-v1.20.0/release.yaml

# Enable auto-TLS
kubectl patch configmap/config-network -n knative-serving --type merge -p '{
  "data": {
    "auto-tls": "Enabled",
    "http-protocol": "Redirected"
  }
}'

# Create ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: contour
EOF
```

Now all services get automatic HTTPS with Let's Encrypt certificates.

---

### 2. Custom Domains

```bash
# Add custom domain annotation
kn service update myapp \
  --annotation serving.knative.dev/custom-domains=myapp.example.com

# Or via YAML
apiVersion: serving.knative.dev/v1
kind: DomainMapping
metadata:
  name: myapp.example.com
  namespace: default
spec:
  ref:
    name: myapp
    kind: Service
    apiVersion: serving.knative.dev/v1
```

Point DNS: `myapp.example.com` → `<ENVOY_EXTERNAL_IP>`

---

### 3. Install metrics-server (for HPA)

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Now you can use CPU-based autoscaling
kn service update myapp \
  --annotation autoscaling.knative.dev/metric=cpu \
  --annotation autoscaling.knative.dev/target=80  # 80% CPU
```

---

### 4. Observability

```bash
# Prometheus metrics are exposed by default
kubectl port-forward -n knative-serving deploy/controller 9090:9090

# View metrics
curl localhost:9090/metrics | grep knative

# Add Grafana dashboard
# Import dashboard ID: 19142 (Knative Serving)
```

---

## Common Commands Cheat Sheet

```bash
# ── Service Management ─────────────────────────────────────────────
kn service create myapp --image=nginx --port=80
kn service update myapp --image=nginx:1.25
kn service delete myapp
kn service list
kn service describe myapp

# ── Scaling ────────────────────────────────────────────────────────
kn service update myapp --scale 0..100        # autoscale 0-100
kn service update myapp --scale 2..10         # min 2, max 10
kn service update myapp --scale 1             # fixed 1 replica

# ── Environment Variables ──────────────────────────────────────────
kn service update myapp --env KEY=value
kn service update myapp --env-from secret:mysecret
kn service update myapp --env-from configmap:myconfig

# ── Traffic Splitting ──────────────────────────────────────────────
kn service update myapp --tag @latest=v2 --tag myapp-00001=v1
kn service update myapp --traffic v1=80,v2=20  # canary 20%
kn service update myapp --traffic @latest=100  # promote v2

# ── Revisions ──────────────────────────────────────────────────────
kn revision list
kn revision describe myapp-00002
kn revision delete myapp-00001

# ── Debugging ──────────────────────────────────────────────────────
kubectl get ksvc                    # List services
kubectl describe ksvc myapp         # Show details
kubectl get revision                # List revisions
kubectl get pods                    # See running pods
kubectl logs -l serving.knative.dev/service=myapp  # App logs

# View Knative controller logs
kubectl logs -n knative-serving deploy/controller
kubectl logs -n knative-serving deploy/autoscaler
kubectl logs -n knative-serving deploy/activator

# ── Status Checks ──────────────────────────────────────────────────
./knative_orbstack.sh --status      # Script status
kubectl get pods -n knative-serving # Knative pods
kubectl get pods -n projectcontour  # Contour/Envoy
kubectl get svc -n projectcontour envoy  # LoadBalancer IP
```

---

## Troubleshooting

### Service not accessible

```bash
# 1. Check Envoy has external IP
kubectl get svc envoy -n projectcontour
# Should show EXTERNAL-IP (e.g., 192.168.139.2)

# 2. Check service is Ready
kubectl get ksvc myapp
# STATUS should be "Ready", URL should be populated

# 3. Test with curl
curl -v http://myapp.default.<IP>.sslip.io

# 4. Check Envoy routing
kubectl get httpproxy -A
# Should show HTTPProxy for your service

# 5. View net-contour-controller logs
kubectl logs -n knative-serving deploy/net-contour-controller --tail=100
```

---

### Scale-to-zero not working

```bash
# Check autoscaler config
kubectl get cm config-autoscaler -n knative-serving -o yaml

# Should have:
# enable-scale-to-zero: "true"
# scale-to-zero-grace-period: "6s"

# Check if there's active traffic
kubectl logs -l serving.knative.dev/service=myapp -c queue-proxy
```

---

### Cold starts too slow

```bash
# 1. Pre-pull images
docker pull your-image:tag

# 2. Keep 1 pod warm
kn service update myapp --scale 1..10

# 3. Reduce image size
# Use distroless or alpine base images

# 4. Check metrics-server is installed
kubectl get deployment metrics-server -n kube-system
```

---

## Run Diagnostics

```bash
# Automated diagnostics collection
./knative_orbstack.sh --debug

# Collects:
# - Service YAML and status
# - Revision details
# - Pod logs
# - HTTPProxy configuration
# - Envoy/Contour logs
# - Events
# Saves to: /tmp/knative-diag-<timestamp>-<service>
```

---

## Comparison: Cloud Run vs Knative

### Advantages of Knative

✓ **Run anywhere** — Your laptop, on-prem, any cloud  
✓ **No vendor lock-in** — Open source, CNCF project  
✓ **Lower cost** — Use spot/preemptible instances, no markup  
✓ **Full control** — Tune every setting, debug internally  
✓ **Multi-cloud** — Same manifests on GCP, AWS, Azure  

### Advantages of Cloud Run

✓ **Zero ops** — Fully managed, no cluster to maintain  
✓ **Global CDN** — Built-in load balancing, edge network  
✓ **IAM integration** — Native GCP identity, VPC, secrets  
✓ **Faster cold starts** — Optimized infrastructure (< 100ms typical)  
✓ **Simpler billing** — Per-100ms billing, no cluster costs  

### When to use which?

| Scenario | Use |
|----------|-----|
| Local dev, integration tests | Knative (OrbStack) |
| Multi-cloud strategy required | Knative |
| Cost-sensitive at scale | Knative (spot instances) |
| Need full infrastructure control | Knative |
| Want zero ops, fastest setup | Cloud Run |
| Heavy GCP integration | Cloud Run |
| Global deployment, edge compute | Cloud Run |

---

## Next Steps

1. **Deploy your app**: Replace sample services with real applications
2. **Add monitoring**: Install Prometheus + Grafana
3. **Enable TLS**: Configure cert-manager for production domains
4. **Learn more**: 
   - [Knative Docs](https://knative.dev/docs/)
   - [kn CLI Reference](https://github.com/knative/client/blob/main/docs/cmd/kn.md)
   - [Contour Documentation](https://projectcontour.io/docs/)

---

## Summary Commands

```bash
# Install everything
./knative_orbstack.sh

# Deploy a service
kn service create myapp --image=nginx --port=80

# Check status
./knative_orbstack.sh --status

# View logs
kubectl logs -l serving.knative.dev/service=myapp

# Cleanup
./knative_orbstack.sh --uninstall
```

You now have a complete Cloud Run equivalent running locally! 🚀
