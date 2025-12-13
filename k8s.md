# Kubernetes Deep Dive: Production Container Platform

**Complete guide to deploying and managing containerized applications on Kubernetes**

## Table of Contents

- [Why This Exists](#why-this-exists)
- [Architecture](#architecture)
- [Installation](#installation)
- [Deployment Patterns](#deployment-patterns)
- [Networking](#networking)
- [Autoscaling](#autoscaling)
- [Storage](#storage)
- [Monitoring](#monitoring)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

## Why This Exists

### The Container Platform Problem

Running containerized applications in production requires:

1. **Orchestration**: Scheduling containers across machines
2. **Networking**: Service discovery, load balancing, ingress
3. **Storage**: Persistent data management
4. **Scaling**: Horizontal scaling based on demand
5. **Security**: RBAC, network policies, secrets management
6. **Observability**: Logging, metrics, tracing

Kubernetes solves these problems with a **declarative API**: you describe desired state, Kubernetes makes it happen.

### Why Not Platform Services?

| Requirement | Platform (Cloud Run) | Kubernetes |
|-------------|---------------------|------------|
| **Portability** | Vendor lock-in | Run anywhere |
| **Cost at scale** | High (vendor markup) | Lower (commodity compute) |
| **Customization** | Limited options | Full control |
| **Ecosystem** | Vendor tools only | 1000+ CNCF projects |
| **Data residency** | Limited regions | Any datacenter |
| **Learning curve** | Easy | Steep but transferable |

**Choose Kubernetes when** you need control, portability, or have complex requirements that exceed platform service capabilities.

## Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          CONTROL PLANE                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐│
│  │ API Server   │  │ Scheduler    │  │ Controller   │  │ etcd        ││
│  │ (REST API)   │  │ (Pod→Node)   │  │ Manager      │  │ (State DB)  ││
│  └──────────────┘  └──────────────┘  └──────────────┘  └─────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  │ (kubelet communication)
                                  │
┌─────────────────────────────────┴─────────────────────────────────────┐
│                          WORKER NODES                                  │
│                                                                        │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │  Node 1                                                         │ │
│  │  ┌──────────┐  ┌─────────────────────────────────────────────┐│ │
│  │  │ kubelet  │  │  Pods                                        ││ │
│  │  │          │  │  ┌───────┐  ┌───────┐  ┌───────┐           ││ │
│  │  │          │  │  │ Pod A │  │ Pod B │  │ Pod C │           ││ │
│  │  │          │  │  │ ┌───┐ │  │ ┌───┐ │  │ ┌───┐ │           ││ │
│  │  │          │  │  │ │CNT│ │  │ │CNT│ │  │ │CNT│ │           ││ │
│  │  │          │  │  │ └───┘ │  │ └───┘ │  │ └───┘ │           ││ │
│  │  │          │  │  └───────┘  └───────┘  └───────┘           ││ │
│  │  └──────────┘  └─────────────────────────────────────────────┘│ │
│  │  ┌──────────┐  ┌─────────────────────────────────────────────┐│ │
│  │  │ kube-    │  │  iptables / IPVS (networking)               ││ │
│  │  │ proxy    │  │  - Service load balancing                   ││ │
│  │  └──────────┘  │  - ClusterIP routing                        ││ │
│  │                └─────────────────────────────────────────────┘│ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                        │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │  Node 2 ... Node N (similar structure)                         │ │
│  └─────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────┘
```

### Request Flow (Internet → Pod)

```
Internet Request: https://api.example.com/users/123
        │
        ▼
┌─────────────────────────────────────────┐
│  DNS: api.example.com → 51.89.xxx.xxx  │
│  (LoadBalancer external IP)             │
└────────────────┬────────────────────────┘
                 ▼
┌─────────────────────────────────────────┐
│  OVH Cloud LoadBalancer                 │
│  - Receives traffic on 51.89.xxx.xxx:443│
│  - Forwards to NodePort 30000-32767     │
└────────────────┬────────────────────────┘
                 ▼
┌─────────────────────────────────────────┐
│  NGINX Ingress Controller (Pod)         │
│  Step 1: TLS termination                │
│    - Validates certificate              │
│    - Decrypts HTTPS → HTTP              │
│                                          │
│  Step 2: Routing decision                │
│    - Reads Host: api.example.com        │
│    - Matches Ingress rule               │
│    - Reads Path: /users/123             │
│    - Routes to Service: api-service:80  │
└────────────────┬────────────────────────┘
                 ▼
┌─────────────────────────────────────────┐
│  Service: api-service (ClusterIP)       │
│  - Type: ClusterIP                      │
│  - Selector: app=api                    │
│  - Port: 80 → TargetPort: 8080          │
│  - Endpoints: [10.1.1.5, 10.1.2.8]      │
│                                          │
│  Load balancing algorithm:               │
│    - Round-robin (default)              │
│    - Session affinity (optional)        │
└────────────────┬────────────────────────┘
                 ▼
      ┌──────────┴──────────┐
      ▼                     ▼
┌──────────┐          ┌──────────┐
│ Pod 1    │          │ Pod 2    │
│ IP:      │          │ IP:      │
│ 10.1.1.5 │          │ 10.1.2.8 │
│          │          │          │
│ ┌──────┐ │          │ ┌──────┐ │
│ │ App  │ │          │ │ App  │ │
│ │:8080 │ │          │ │:8080 │ │
│ └──────┘ │          │ └──────┘ │
└──────────┘          └──────────┘
```

**Timing breakdown** (typical):
- t=0ms: Client sends HTTPS request
- t=5ms: DNS resolution
- t=15ms: LoadBalancer receives request
- t=18ms: NGINX Ingress TLS termination
- t=20ms: Service selects backend Pod
- t=22ms: Pod receives HTTP request
- t=25ms: Application processes request
- t=50ms: Response returns to client

**Total latency**: ~50ms (varies by workload, distance, network)

## Installation

### Prerequisites

```bash
# 1. kubectl (Kubernetes CLI)
# macOS
brew install kubectl

# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Verify
kubectl version --client

# 2. Cluster access (kubeconfig)
# OVH: Download from Control Panel → Kubernetes → Kubeconfig
# Save to ~/.kube/config

# Test connectivity
kubectl cluster-info
kubectl get nodes
```

### Component Installation

#### 1. NGINX Ingress Controller

**Why NGINX?**
- Industry standard (used by millions)
- High performance (C-based, async I/O)
- Extensive configuration options
- Built-in rate limiting, auth, rewrites

```bash
# Install via kubectl
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/v1.10.0/deploy/static/provider/cloud/deploy.yaml

# Wait for LoadBalancer IP
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=180s

# Get external IP
kubectl get svc -n ingress-nginx ingress-nginx-controller
# Output: EXTERNAL-IP: 51.89.xxx.xxx
```

**How it works**:
1. Deployment creates NGINX Pods
2. Service type=LoadBalancer provisions OVH LoadBalancer
3. NGINX watches Ingress resources
4. Generates nginx.conf from Ingress rules
5. Reloads configuration on changes

#### 2. cert-manager (TLS Automation)

**Why cert-manager?**
- Automates Let's Encrypt certificate lifecycle
- Supports DNS-01 and HTTP-01 challenges
- Automatic renewal (30 days before expiry)
- Multi-cloud compatible

```bash
# Install CRDs
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.crds.yaml

# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml

# Wait for pods
kubectl wait --namespace cert-manager \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/instance=cert-manager \
  --timeout=120s

# Create Let's Encrypt issuer
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com  # CHANGE THIS
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

**Certificate issuance flow**:
```
1. Ingress created with cert-manager annotation
   │
   ▼
2. cert-manager detects Ingress
   - Creates Certificate resource
   - Creates Order resource
   │
   ▼
3. ACME HTTP-01 challenge
   - cert-manager creates temporary Ingress rule
   - Let's Encrypt queries: http://example.com/.well-known/acme-challenge/<token>
   - cert-manager responds with challenge answer
   │
   ▼
4. Let's Encrypt validates
   - Domain ownership confirmed
   - Issues certificate
   │
   ▼
5. cert-manager stores certificate
   - Creates Secret with TLS cert + key
   - Ingress references Secret for HTTPS
   │
   ▼
6. Auto-renewal (30 days before expiry)
   - Repeats validation process
   - Updates Secret
```

#### 3. metrics-server (Resource Metrics)

**Why metrics-server?**
- Provides CPU/memory metrics to HPA
- Lightweight (single binary)
- Required for `kubectl top` commands

```bash
# Install
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.7.0/components.yaml

# Wait for pod
kubectl wait --namespace kube-system \
  --for=condition=ready pod \
  --selector=k8s-app=metrics-server \
  --timeout=90s

# Test
kubectl top nodes
kubectl top pods -A
```

**Metrics collection flow**:
```
kubelet (on each node)
  │
  ├─ Exposes /metrics/resource endpoint
  │  - CPU usage (nanocores)
  │  - Memory usage (bytes)
  │  - Collection interval: 10s
  │
  ▼
metrics-server
  │
  ├─ Scrapes kubelet every 15s
  ├─ Aggregates cluster-wide metrics
  ├─ Exposes metrics API: /apis/metrics.k8s.io/
  │
  ▼
HPA Controller
  │
  ├─ Queries metrics API every 30s
  ├─ Calculates desired replicas:
  │    desiredReplicas = ceil[currentReplicas × (currentMetricValue / targetMetricValue)]
  │
  ▼
Scale Decision
  │
  ├─ If desiredReplicas > currentReplicas: SCALE UP
  ├─ If desiredReplicas < currentReplicas: SCALE DOWN
  └─ Cooldown: 3min up, 5min down
```

## Deployment Patterns

### Basic Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  labels:
    app: myapp
spec:
  replicas: 3  # Desired number of Pods
  selector:
    matchLabels:
      app: myapp  # Must match template labels
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: app
        image: myregistry.io/myapp:v1.2.3
        ports:
        - containerPort: 8080
          name: http
          protocol: TCP
        
        # Resource management (CRITICAL for production)
        resources:
          requests:
            cpu: 100m       # Minimum guaranteed CPU
            memory: 128Mi   # Minimum guaranteed memory
          limits:
            cpu: 200m       # Maximum CPU (throttled beyond)
            memory: 256Mi   # Maximum memory (OOMKilled beyond)
        
        # Health checks
        livenessProbe:
          httpGet:
            path: /healthz
            port: http
          initialDelaySeconds: 30  # Wait before first check
          periodSeconds: 10        # Check every 10s
          timeoutSeconds: 5        # Timeout after 5s
          failureThreshold: 3      # Restart after 3 failures
        
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
          failureThreshold: 2      # Remove from Service after 2 failures
        
        # Environment variables
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: myapp-secrets
              key: db-url
        - name: LOG_LEVEL
          value: "info"
```

### Rolling Update Strategy

**Zero-downtime deployments**:

```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0      # Keep all old pods until new ones are ready
      maxSurge: 1            # Create 1 extra pod during rollout
```

**Update flow**:
```
Initial state: 3 pods (v1.0)
  [Pod-A] [Pod-B] [Pod-C]

Step 1: Create new pod (v1.1)
  [Pod-A] [Pod-B] [Pod-C] [Pod-D(v1.1)]
  - maxSurge=1 allows 4 pods temporarily

Step 2: Wait for Pod-D readiness
  - readinessProbe must pass
  - Only then does Pod-D join Service

Step 3: Terminate old pod
  [Pod-A] [Pod-B] [Pod-C] → [Pod-B] [Pod-C] [Pod-D(v1.1)]
  - maxUnavailable=0 ensures 3 running

Step 4: Repeat until all updated
  [Pod-B] [Pod-C] [Pod-D(v1.1)]
  [Pod-B] [Pod-C] [Pod-D(v1.1)] [Pod-E(v1.1)]
  [Pod-C] [Pod-D(v1.1)] [Pod-E(v1.1)]
  [Pod-C] [Pod-D(v1.1)] [Pod-E(v1.1)] [Pod-F(v1.1)]
  [Pod-D(v1.1)] [Pod-E(v1.1)] [Pod-F(v1.1)]

Final state: 3 pods (v1.1)
  [Pod-D] [Pod-E] [Pod-F]
```

### Blue-Green Deployment

**Instant switchover, easy rollback**:

```bash
# Deploy green (new version)
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp-green
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
      version: green
  template:
    metadata:
      labels:
        app: myapp
        version: green
    spec:
      containers:
      - name: app
        image: myregistry.io/myapp:v2.0.0
        # ... rest of config
EOF

# Wait for green to be healthy
kubectl wait --for=condition=ready pod -l version=green --timeout=120s

# Test green (port-forward or internal Service)
kubectl port-forward deployment/myapp-green 8080:8080
curl http://localhost:8080/healthz

# Switch traffic (update Service selector)
kubectl patch service myapp -p '{"spec":{"selector":{"version":"green"}}}'

# Verify traffic switched
curl https://myapp.example.com

# If issues, instant rollback:
kubectl patch service myapp -p '{"spec":{"selector":{"version":"blue"}}}'

# Once confident, remove blue
kubectl delete deployment myapp-blue
```

## Networking

### Service Types

#### ClusterIP (Internal Only)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: myapp
spec:
  type: ClusterIP  # Default, internal only
  selector:
    app: myapp
  ports:
  - protocol: TCP
    port: 80        # Service port
    targetPort: 8080  # Pod port
```

**Use case**: Internal microservices, databases

#### LoadBalancer (External Access)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: myapp-external
spec:
  type: LoadBalancer  # OVH provisions external IP
  selector:
    app: myapp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
```

**Use case**: Direct external access (bypassing Ingress)

#### NodePort (Dev/Testing)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: myapp-nodeport
spec:
  type: NodePort
  selector:
    app: myapp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
    nodePort: 30080  # Accessible on any node:30080
```

**Use case**: Development, NGINX Ingress backend

### Ingress (HTTP/HTTPS Routing)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myapp
  annotations:
    # cert-manager
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    
    # NGINX-specific annotations
    nginx.ingress.kubernetes.io/rate-limit: "100"  # 100 req/s per IP
    nginx.ingress.kubernetes.io/ssl-redirect: "true"  # Force HTTPS
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"  # Max upload size
spec:
  ingressClassName: nginx
  
  tls:
  - hosts:
    - myapp.example.com
    secretName: myapp-tls  # cert-manager creates this
  
  rules:
  - host: myapp.example.com
    http:
      paths:
      # API routes
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 80
      
      # Static files
      - path: /static
        pathType: Prefix
        backend:
          service:
            name: static-service
            port:
              number: 80
      
      # Default (frontend)
      - path: /
        pathType: Prefix
        backend:
          service:
            name: frontend-service
            port:
              number: 80
```

### DNS and Service Discovery

**Internal DNS** (automatic):
```bash
# Format: <service-name>.<namespace>.svc.cluster.local

# Same namespace
curl http://api-service

# Different namespace
curl http://api-service.production.svc.cluster.local

# Headless Service (direct Pod IPs)
# Returns: pod-1.api-service.production.svc.cluster.local
```

## Autoscaling

### Horizontal Pod Autoscaler (HPA)

**CPU-based scaling**:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70  # Scale when avg CPU > 70%
```

**Scaling algorithm**:
```
desiredReplicas = ceil[currentReplicas × (currentMetricValue / targetMetricValue)]

Example:
- currentReplicas = 3
- currentCPU = 85% (average across 3 pods)
- targetCPU = 70%

desiredReplicas = ceil[3 × (85 / 70)]
                = ceil[3 × 1.214]
                = ceil[3.642]
                = 4 pods

Next check (30s later):
- currentReplicas = 4
- currentCPU = 68% (below target)
- No scaling (within tolerance)
```

**Multi-metric scaling**:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  
  # Custom metrics (requires Prometheus Adapter)
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: "100"  # Scale at 100 RPS per pod
  
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300  # 5 minutes
      policies:
      - type: Percent
        value: 50  # Max 50% of pods per period
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0  # Immediate
      policies:
      - type: Percent
        value: 100  # Max 100% (double) per period
        periodSeconds: 15
```

## Storage

### Persistent Volumes (PV)

**OVH Block Storage**:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: myapp-data
spec:
  accessModes:
  - ReadWriteOnce  # Single node read-write
  resources:
    requests:
      storage: 20Gi
  storageClassName: csi-cinder-high-speed  # OVH storage class
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 1  # ReadWriteOnce allows only 1 pod
  template:
    spec:
      containers:
      - name: app
        image: myapp:latest
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: myapp-data
```

**StatefulSet** (for databases):

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
spec:
  serviceName: postgres
  replicas: 3
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Gi
```

**Pods get stable names**: `postgres-0`, `postgres-1`, `postgres-2`
**Volumes persist**: Even if pod deleted, PVC remains

## Monitoring

### Prometheus + Grafana Stack

**Install kube-prometheus-stack**:

```bash
# Add Helm repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install with custom values
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace \
  --set prometheus.prometheusSpec.retention=30d \
  --set grafana.adminPassword=admin
```

**Access Grafana**:
```bash
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80

# Open: http://localhost:3000
# Login: admin / admin
```

**Pre-built dashboards**:
- Cluster overview
- Node metrics
- Pod metrics
- Ingress performance

## Security

### RBAC (Role-Based Access Control)

```yaml
# ServiceAccount for app
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp
  namespace: production
---
# Role: permissions within namespace
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: myapp-role
  namespace: production
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list"]
---
# RoleBinding: grant Role to ServiceAccount
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: myapp-binding
  namespace: production
subjects:
- kind: ServiceAccount
  name: myapp
  namespace: production
roleRef:
  kind: Role
  name: myapp-role
  apiGroup: rbac.authorization.k8s.io
---
# Use ServiceAccount in Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  namespace: production
spec:
  template:
    spec:
      serviceAccountName: myapp
      containers:
      - name: app
        image: myapp:latest
```

### Network Policies

**Restrict pod-to-pod traffic**:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-policy
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # Allow from frontend only
  - from:
    - podSelector:
        matchLabels:
          app: frontend
    ports:
    - protocol: TCP
      port: 8080
  egress:
  # Allow to database only
  - to:
    - podSelector:
        matchLabels:
          app: database
    ports:
    - protocol: TCP
      port: 5432
  # Allow DNS
  - to:
    - namespaceSelector: {}
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53
```

## Troubleshooting

### Debugging Pods

```bash
# Describe pod (events, status)
kubectl describe pod <pod-name>

# View logs
kubectl logs <pod-name>
kubectl logs <pod-name> --previous  # Previous container (after crash)
kubectl logs <pod-name> -c <container-name>  # Multi-container pod

# Execute commands in pod
kubectl exec -it <pod-name> -- /bin/bash
kubectl exec -it <pod-name> -- curl http://localhost:8080/healthz

# Copy files to/from pod
kubectl cp <pod-name>:/path/to/file ./local-file
kubectl cp ./local-file <pod-name>:/path/to/file
```

### Debugging Services

```bash
# Check Service endpoints
kubectl get endpoints <service-name>

# Test connectivity from another pod
kubectl run -it --rm debug --image=busybox --restart=Never -- sh
# Inside pod:
wget -O- http://myapp.default.svc.cluster.local

# Check DNS resolution
nslookup myapp.default.svc.cluster.local
```

### Debugging Ingress

```bash
# Describe Ingress
kubectl describe ingress <ingress-name>

# Check NGINX Ingress logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller -f

# View NGINX config
kubectl exec -n ingress-nginx <nginx-pod> -- cat /etc/nginx/nginx.conf

# Test NGINX config syntax
kubectl exec -n ingress-nginx <nginx-pod> -- nginx -T
```

### Common Issues

**Pods stuck in Pending**:
```bash
# Check node resources
kubectl describe node <node-name>

# Check PVC status
kubectl get pvc
kubectl describe pvc <pvc-name>
```

**CrashLoopBackOff**:
```bash
# Check logs
kubectl logs <pod-name> --previous

# Check liveness/readiness probes
kubectl describe pod <pod-name> | grep -A 10 "Liveness\|Readiness"
```

**ImagePullBackOff**:
```bash
# Check image name/tag
kubectl describe pod <pod-name> | grep "Image:"

# Check imagePullSecrets
kubectl get secrets
```

## Production Checklist

- [ ] **Multi-zone cluster**: At least 3 nodes across availability zones
- [ ] **Resource limits**: All containers have CPU/memory limits
- [ ] **Health checks**: Liveness and readiness probes configured
- [ ] **HPA**: Autoscaling enabled for variable workloads
- [ ] **PodDisruptionBudget**: Protect against involuntary disruptions
- [ ] **NetworkPolicy**: Restrict pod-to-pod traffic
- [ ] **RBAC**: Least-privilege ServiceAccounts
- [ ] **Secrets management**: External secrets operator or sealed secrets
- [ ] **TLS**: Production Let's Encrypt certificates
- [ ] **Monitoring**: Prometheus + Grafana + Alertmanager
- [ ] **Logging**: Centralized logging (ELK, Loki)
- [ ] **Backups**: Velero or equivalent for disaster recovery
- [ ] **CI/CD**: Automated deployments with rollback capability
- [ ] **Cost monitoring**: Track resource usage and optimize

## Next Steps

- **OVH-specific guide**: [k8s-ovh.md](k8s-ovh.md)
- **Quick reference**: [README-k8s.md](README-k8s.md)
- **Official docs**: [kubernetes.io/docs](https://kubernetes.io/docs/)
- **CNCF landscape**: [cncf.io/projects](https://www.cncf.io/projects/)

## Contributing

Improvements welcome! Submit issues or pull requests.

## License

MIT License
