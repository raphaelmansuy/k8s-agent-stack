# Deployment Guide

This guide covers deploying, updating, and managing agents on k8s-agent-stack.

## Deploy an Agent

### Method 1: kagent Agent CRD (Recommended)

Deploy agents using kagent's Agent Custom Resource:

```bash
# 1. Build your agent image with dev.local prefix
cd kagent-adk-agent
docker build -t dev.local/my-agent:v1 .

# 2. Create Agent CRD manifest
cat <<EOF > my-agent.yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: my-agent
  namespace: kagent
spec:
  type: BYO
  description: "My custom AI agent"
  byo:
    deployment:
      image: dev.local/my-agent:v1
      imagePullPolicy: IfNotPresent
      env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: openai-api-key
              key: api-key
      resources:
        requests:
          cpu: 250m
          memory: 512Mi
        limits:
          cpu: 1000m
          memory: 2Gi
      probes:
        liveness:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
EOF

# 3. Deploy
kubectl apply -f my-agent.yaml

# 4. Wait for ready
kubectl wait --for=condition=Ready agent/my-agent -n kagent --timeout=120s

# 5. Test
kubectl port-forward -n kagent svc/my-agent 8081:8080 &
curl http://localhost:8081/health
```

### Method 2: Pre-built Google ADK Agent

Deploy the included reference agent:

```bash
# Navigate to agent directory
cd kagent-adk-agent

# Deploy to Kubernetes
kubectl apply -f kagent-deployment.yaml

# Wait for ready (30-60 seconds)
kubectl wait --for=condition=ready agent \
  -l app.kubernetes.io/name=google-adk-byo-agent \
  -n kagent --timeout=120s

# Test the agent
kubectl port-forward -n kagent svc/google-adk-byo-agent 8080:8080 &
curl http://localhost:8080/health
```

### Method 3: Knative Service (kn CLI)

```bash
# For remote images
kn service create my-agent \
  --image=gcr.io/your-project/my-agent:v1 \
  --port=8080 \
  --env GOOGLE_API_KEY=your-key \
  --scale-min=0 \
  --scale-max=10 \
  --concurrency-target=10

# For local images (must use dev.local prefix)
docker tag my-agent:v1 dev.local/my-agent:v1
kn service create my-agent \
  --image=dev.local/my-agent:v1 \
  --port=8080 \
  --pull-policy=IfNotPresent
```

### Method 4: From Local Docker Image

```bash
# Build locally with dev.local prefix
cd my-agent
docker build -t dev.local/my-agent:dev .

# Deploy via kubectl
cat <<EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: my-agent
  namespace: kagent
spec:
  template:
    spec:
      containers:
      - image: dev.local/my-agent:dev
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
EOF

# Get URL
kubectl get ksvc my-agent -n kagent -o jsonpath='{.status.url}'
```

> **Important:** Local images must use the `dev.local/` prefix for Knative to skip tag resolution.

---

## Update an Agent

### Update Image Version

```bash
kn service update my-agent --image=gcr.io/your-project/agent:v2
```

### Update Environment Variables

```bash
kn service update my-agent \
  --env MODEL=gpt-4-turbo \
  --env MAX_TOKENS=2000
```

### Update Scaling Parameters

```bash
kn service update my-agent \
  --scale-min=1 \
  --scale-max=10 \
  --concurrency-target=20
```

---

## Traffic Splitting (Canary Deployment)

### Canary Release Pattern

```bash
# Deploy new version
kn service update my-agent --image=gcr.io/your-project/agent:v2

# Split traffic: 90% v1, 10% v2
kn service update my-agent \
  --traffic my-agent-v1=90,@latest=10

# Gradually increase
kn service update my-agent \
  --traffic my-agent-v1=50,@latest=50

# Full rollout
kn service update my-agent \
  --traffic @latest=100
```

### Visualization

```
  Users
    │
    ▼
┌─────────────┐
│   Envoy     │
└──────┬──────┘
       │
       ├─────90%─────▶ Agent v1 (stable)
       │
       └─────10%─────▶ Agent v2 (canary)
```

---

## Monitor and Debug

### Check Status

```bash
# List all services
kn service list
kubectl get ksvc -n kagent

# Describe specific service
kn service describe my-agent
```

### View Logs

```bash
# Stream logs
kubectl logs -f -n kagent deployment/google-adk-agent

# Get logs with label
kubectl logs -n kagent -l app.kubernetes.io/name=google-adk-agent --tail=50

# Previous crash logs
kubectl logs -n kagent <pod-name> --previous
```

### Watch Scaling

```bash
# Watch pods scale
watch 'kubectl get pods -n kagent'

# Check autoscaler
kubectl logs -n knative-serving deploy/autoscaler --tail=30
```

---

## Local Development Loop

```bash
# 1. Make code changes
vim kagent-adk-agent/app/agent.py

# 2. Build locally with dev.local prefix
cd kagent-adk-agent
docker build -t dev.local/my-agent:dev .

# 3. Deploy via Agent CRD
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
      image: dev.local/my-agent:dev
      imagePullPolicy: IfNotPresent
EOF

# 4. Test
kubectl port-forward svc/my-agent -n kagent 8080:8080 &
curl http://localhost:8080/health

# 5. Check logs
kubectl logs -n kagent -l app.kubernetes.io/name=my-agent --tail=20

# 6. Iterate - delete and redeploy
kubectl delete agent my-agent -n kagent
# repeat from step 2
```

---

## Agent Development Workflow

```
┌──────────────┐
│  1. Develop  │  Write agent code locally
│  Locally     │  Test with docker run
└──────┬───────┘
       │
┌──────▼───────┐
│  2. Build    │  docker build -t agent:v1
│  Container   │  
└──────┬───────┘
       │
┌──────▼───────┐
│  3. Deploy   │  kn service create agent --image=agent:v1
│  to K8s      │  or kubectl apply -f deployment.yaml
└──────┬───────┘
       │
┌──────▼───────┐
│  4. Test     │  curl agent-url/endpoint
│  & Monitor   │  kubectl logs, watch pods scale
└──────┬───────┘
       │
┌──────▼───────┐
│  5. Update   │  kn service update agent --image=agent:v2
│  Version     │  Traffic split for canary
└──────────────┘
```

---

## Production Checklist

### Infrastructure

- [ ] metrics-server installed
- [ ] Prometheus + Grafana for monitoring
- [ ] cert-manager for TLS
- [ ] Custom domain configured
- [ ] Network policies for isolation
- [ ] Backup strategy for persistent data

### Agent Configuration

- [ ] Resource limits defined:
  ```yaml
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 2000m
      memory: 2Gi
  ```
- [ ] Health checks implemented (`/health`, `/readiness`)
- [ ] Structured logging (JSON format)
- [ ] Secrets in Kubernetes Secrets

### Autoscaling

- [ ] `scale-min`: 0 (scale-to-zero) or 1+ (warm pods)
- [ ] `scale-max`: Based on expected load
- [ ] `concurrency-target`: 10-50 per pod

### Observability

- [ ] Log aggregation (ELK, Loki)
- [ ] Alerting rules configured
- [ ] Dashboards for metrics

### Security

- [ ] TLS enabled
- [ ] API key rotation scheduled
- [ ] RBAC configured
- [ ] Container images scanned
- [ ] Rate limiting enabled

---

## Best Practices

### Agent Development

| Practice | Why |
|----------|-----|
| Use small base images | Faster cold starts |
| Implement health checks | Reliable deployments |
| Stream long responses | Better UX |
| Log structured data | Easier debugging |
| Make agents stateless | Scalability |

### Deployment

| Practice | Why |
|----------|-----|
| Start with scale-min=0 | Verify scale-to-zero works |
| Set resource limits | Prevent resource exhaustion |
| Use traffic splitting | Safe rollouts |
| Tag images properly | Reproducibility |

### Example: Resource Limits

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: my-agent
spec:
  template:
    spec:
      containers:
      - image: my-agent:v1
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi
```

---

## Next Steps

- [Troubleshooting](troubleshooting.md) - Debug common issues
- [Building ADK Agents](building-google-adk-agents-for-kagent.md) - Custom agent development
- [Architecture](architecture.md) - Understand the platform

---

[← Back to Documentation Index](README.md) • [Getting Started](getting-started.md) • [Troubleshooting](troubleshooting.md) • [Main README](../README.md)
