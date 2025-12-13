# Cloud Run on Kubernetes (Knative Serving)

> Production-grade serverless containers with scale-to-zero, running anywhere

## 🎯 Why This Exists

**Cloud Run is brilliant** — deploy containers, pay per request, scale to zero. But it locks you to Google Cloud.

**Knative Serving** is Cloud Run's open-source foundation. Same team, same behavior, runs anywhere: your laptop, on-prem, any cloud.

**This project** gets you from zero to a working "Cloud Run clone" in **< 5 minutes**.

---

## ⚡ Quick Start (OrbStack/macOS)

```bash
# Clone this repo
git clone <your-repo>
cd cloudrun-like

# Run the installer (requires OrbStack with Kubernetes enabled)
./knative_orbstack.sh

# Deploy a service
kn service create myapp --image=nginx --port=80

# Test
curl $(kn service describe myapp -o url)
```

**Done!** You now have:
- ✅ Scale-to-zero (<2s)
- ✅ Auto-scaling based on concurrency
- ✅ Automatic routing and DNS
- ✅ Full Cloud Run semantics

---

## 📚 Documentation

| File | Description |
|------|-------------|
| **[knative.md](./knative.md)** | Complete guide: architecture, concepts, production setup |
| **[knative-orbstack.md](./knative-orbstack.md)** | OrbStack-specific quickstart guide |
| **[knative_orbstack.sh](./knative_orbstack.sh)** | Automated installer script |

---

## 🏗️ Architecture

```ascii
┌─────────────────────────────────────────────────────────────┐
│                     YOUR REQUEST                            │
│                          │                                  │
│                          v                                  │
│              ┌──────────────────────┐                       │
│              │  Envoy (Ingress)     │                       │
│              │  LoadBalancer        │                       │
│              └──────────┬───────────┘                       │
│                         │                                   │
│                         v                                   │
│         ┌───────────────────────────────┐                   │
│         │  Knative Serverless Service   │                   │
│         │  (Smart Router)               │                   │
│         └───────┬──────────────┬────────┘                   │
│                 │              │                            │
│    scale=0      v              v  scale>0                   │
│         ┌────────────┐   ┌─────────────┐                   │
│         │ Activator  │   │ Your Pod    │                   │
│         │ (buffer)   │   │ [queue-proxy]                   │
│         └────────────┘   │ [container] │                   │
│                          └─────────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

**Key components:**
- **Envoy** — Entry point (like Cloud Run's Google Front End)
- **Activator** — Buffers requests during scale-from-zero
- **queue-proxy** — Sidecar enforcing concurrency limits
- **Controller** — Reconciles Service → Revision → Deployment

See [knative.md](./knative.md) for detailed architecture diagrams.

---

## 🚀 Features

| Feature | Status | Notes |
|---------|--------|-------|
| Scale-to-zero | ✅ | <2s with activator buffering |
| Concurrency control | ✅ | Per-pod limits, queuing |
| Traffic splitting | ✅ | Blue/green, canary deployments |
| Auto-scaling | ✅ | Concurrency or RPS based |
| Custom domains | ✅ | With cert-manager |
| Auto-TLS | ✅ | Let's Encrypt integration |
| kn CLI | ✅ | Cloud Run-style UX |
| Multi-arch | ✅ | ARM64 + AMD64 |

---

## 📦 What Gets Installed

```bash
./knative_orbstack.sh
```

**Components:**
- **Contour** (v1.33+) — Modern ingress controller
- **Envoy** — L7 proxy (data plane)
- **Knative Serving** (v1.20.0) — Serverless runtime
  - controller, webhook, autoscaler, activator
- **net-contour** — Knative ↔ Contour bridge
- **Magic DNS** — sslip.io for automatic domains

**Optional (flags):**
- `--install-metrics` → metrics-server for HPA
- `--install-metalb` → LoadBalancer IP allocation
- `--prepull` → Pre-pull common images
- `--warm N` → Keep services warm (min N replicas)

---

## 🎓 Learn Concepts

### What is a Knative Service?

A **Service** is the top-level resource (like Cloud Run service). It automatically manages:

```yaml
Service (your definition)
  └→ Configuration (immutable snapshot)
      └→ Revision (versioned deployment)
          └→ Deployment (K8s pods)
              └→ Pods [queue-proxy + your-container]
```

### What is a Revision?

Every update creates a new **Revision** (immutable). You can:
- Route traffic to multiple revisions (blue/green)
- Pin traffic to old revisions (instant rollback)
- Tag revisions for A/B testing

```bash
# Deploy v1
kn service create myapp --image=myapp:v1

# Deploy v2 (creates new revision)
kn service update myapp --image=myapp:v2

# Split traffic: 90% v1, 10% v2
kn service update myapp --traffic @latest=10,myapp-00001=90
```

### How does scale-to-zero work?

```ascii
Traffic arrives → SKS checks replicas
  │
  ├─ IF replicas > 0 → Route directly to pod
  │
  └─ IF replicas = 0 → Route to Activator
                       Activator holds request
                       Autoscaler scales to 1
                       Pod starts (~100-300ms)
                       Activator forwards request
```

**Cold start optimization:**
- Pre-pull images
- Use smaller base images (distroless)
- Keep 1 pod warm: `--scale 1..10`

---

## 🛠️ Common Commands

```bash
# ── Deployment ─────────────────────────────────────────────
kn service create myapp --image=nginx --port=80
kn service update myapp --image=nginx:1.25
kn service delete myapp

# ── Scaling ────────────────────────────────────────────────
kn service update myapp --scale 0..100     # autoscale 0-100
kn service update myapp --scale 2..10      # min 2, max 10

# ── Environment ────────────────────────────────────────────
kn service update myapp --env KEY=value
kn service update myapp --env-from secret:mysecret

# ── Traffic Management ─────────────────────────────────────
kn service update myapp --tag @latest=v2
kn service update myapp --traffic v2=20,v1=80  # canary

# ── Debugging ──────────────────────────────────────────────
kubectl get ksvc                               # List services
kubectl logs -l serving.knative.dev/service=myapp  # Logs
./knative_orbstack.sh --status                 # Installation status
./knative_orbstack.sh --debug                  # Collect diagnostics
```

---

## 🔧 Troubleshooting

### Service not accessible?

```bash
# 1. Check Envoy has external IP
kubectl get svc envoy -n projectcontour

# 2. Verify service is Ready
kubectl get ksvc myapp

# 3. View controller logs
kubectl logs -n knative-serving deploy/controller

# 4. Run diagnostics
./knative_orbstack.sh --debug
```

### Scale-to-zero not working?

```bash
# Check autoscaler config
kubectl get cm config-autoscaler -n knative-serving -o yaml

# View autoscaler logs
kubectl logs -n knative-serving deploy/autoscaler
```

### Slow cold starts?

```bash
# Keep pods warm
kn service update myapp --scale 1..10

# Pre-pull images
docker pull your-image:tag

# Use smaller images
FROM gcr.io/distroless/static:nonroot
```

---

## 📊 Comparison: Cloud Run vs Knative

| Aspect | Cloud Run | Knative (this setup) |
|--------|-----------|---------------------|
| **Setup** | `gcloud run deploy` | `./knative_orbstack.sh` |
| **Cost (idle)** | $0 | $0 |
| **Cost (active)** | $0.40/M requests | ~$0.10/M requests |
| **Cold start** | 50-150ms | 100-300ms |
| **Max scale** | 1000 | Unlimited |
| **Vendor lock** | Google only | Any K8s |
| **Control** | Limited | Full |

**Use Cloud Run when:**
- You want zero ops
- Need global CDN
- Heavy GCP integration

**Use Knative when:**
- Multi-cloud required
- Full control needed
- Cost optimization critical
- Local dev workflow

---

## 🧪 Testing

```bash
# Deploy sample app
kn service create hello --image=us-docker.pkg.dev/cloudrun/container/hello --port=8080

# Get URL
URL=$(kn service describe hello -o url)

# Test basic request
curl $URL

# Test scale-to-zero
# Wait 30s, then:
curl $URL  # Should see ~200ms cold start

# Test concurrent requests
ab -n 1000 -c 50 $URL

# Watch autoscaling
watch kubectl get pods -l serving.knative.dev/service=hello
```

---

## 🎯 Production Checklist

- [ ] Install metrics-server: `--install-metrics`
- [ ] Configure TLS: Install cert-manager
- [ ] Set resource limits: CPU/memory per service
- [ ] Enable observability: Prometheus + Grafana
- [ ] Configure custom domains: DomainMapping
- [ ] Set min-scale for critical services
- [ ] Pre-pull production images
- [ ] Test autoscaling under load
- [ ] Configure network policies
- [ ] Set up backup/disaster recovery

See [knative.md](./knative.md) for detailed production setup.

---

## 📖 Resources

- **[Complete Guide](./knative.md)** — Architecture, concepts, production
- **[OrbStack Guide](./knative-orbstack.md)** — Local development setup
- [Knative Documentation](https://knative.dev/docs/)
- [kn CLI Reference](https://github.com/knative/client/blob/main/docs/cmd/kn.md)
- [Contour Documentation](https://projectcontour.io/docs/)

---

## 🤝 Contributing

Improvements welcome! Areas of focus:
- Performance tuning
- Additional cloud platform guides
- Production deployment patterns
- CI/CD integration examples

---

## 📝 License

MIT

---

## 🙏 Acknowledgments

Built on:
- [Knative](https://knative.dev) — Google's serverless platform
- [Contour](https://projectcontour.io) — VMware's ingress controller
- [Envoy](https://envoyproxy.io) — CNCF's L7 proxy
- [OrbStack](https://orbstack.dev) — Fast containers for macOS

---

**Ready to go serverless anywhere?**

```bash
./knative_orbstack.sh
```

🚀
