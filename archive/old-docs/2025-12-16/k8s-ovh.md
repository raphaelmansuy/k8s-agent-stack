# Kubernetes on OVH Cloud: 5-Minute Setup

**Fast-track guide for deploying containerized apps on OVH's managed Kubernetes**

## Overview

OVH Cloud provides managed Kubernetes with:
- **Free control plane** (unlike GKE/EKS)
- **EU data residency** (GDPR-compliant)
- **Competitive pricing** (~€0.04/hour per worker node)
- **Free egress** within OVH network
- **Automatic LoadBalancer** provisioning

## Step 1: Create OVH Kubernetes Cluster (5 minutes)

### Via OVH Control Panel

1. Log in to [OVH Cloud Control Panel](https://www.ovh.com/manager/)
2. Navigate to: **Public Cloud** → **Kubernetes**
3. Click **Create a cluster**
4. Configure:
   - **Name**: `production-k8s`
   - **Region**: Choose closest to users (e.g., `GRA7` for France, `UK1` for UK)
   - **Kubernetes version**: Latest stable (1.28+)
   - **Node pool**:
     - **Flavor**: `b2-7` (2 vCPU, 7GB RAM) or higher
     - **Count**: 3 nodes (recommended for HA)
     - **Auto-scaling**: Enable (min: 2, max: 10)
5. Click **Create**

**Provisioning time**: 3-5 minutes

### Via OVH API / Terraform

```hcl
# terraform/main.tf
resource "ovh_cloud_project_kube" "k8s_cluster" {
  service_name = var.service_name
  name         = "production-k8s"
  region       = "GRA7"
  version      = "1.28"
}

resource "ovh_cloud_project_kube_nodepool" "node_pool" {
  service_name  = var.service_name
  kube_id       = ovh_cloud_project_kube.k8s_cluster.id
  name          = "default-pool"
  flavor_name   = "b2-7"
  desired_nodes = 3
  min_nodes     = 2
  max_nodes     = 10
  autoscale     = true
}
```

```bash
terraform init
terraform apply
```

## Step 2: Download Kubeconfig (30 seconds)

### Via Control Panel

1. In OVH Control Panel: **Kubernetes** → **Your Cluster**
2. Click **Download kubeconfig**
3. Save to `~/.kube/config`:

```bash
# Backup existing config
mv ~/.kube/config ~/.kube/config.backup

# Move OVH kubeconfig
mv ~/Downloads/kubeconfig.yaml ~/.kube/config

# Or merge with existing config
KUBECONFIG=~/.kube/config:~/Downloads/kubeconfig.yaml kubectl config view --flatten > ~/.kube/merged_config
mv ~/.kube/merged_config ~/.kube/config

# Verify
kubectl cluster-info
kubectl get nodes
```

### Via OVH CLI

```bash
# Install OVH CLI
pip install ovh

# Configure credentials
ovh-eu  # Follow prompts for API keys

# Download kubeconfig
ovh-eu cloud-project-kube get-kubeconfig \
  --service-name <your-service-name> \
  --kube-id <your-kube-id> \
  > ~/.kube/ovh-config.yaml

# Use
export KUBECONFIG=~/.kube/ovh-config.yaml
kubectl get nodes
```

## Step 3: Install Core Components (3 minutes)

```bash
# Clone or download k8s_ovh.sh
curl -O https://raw.githubusercontent.com/your-repo/cloudrun-like/main/k8s_ovh.sh
chmod +x k8s_ovh.sh

# Install everything
./k8s_ovh.sh --install-all

# Output:
# ✓ kubectl found
# ✓ Connected to cluster: ovh_production-k8s
# ✓ Cluster has 3 node(s)
# ▶ Installing NGINX Ingress Controller v1.10.0...
# ✓ NGINX Ingress Controller is ready
# ✓ LoadBalancer external IP: 51.89.160.25
# ▶ Installing cert-manager v1.14.0...
# ✓ cert-manager is ready
# ✓ ClusterIssuers created
# ▶ Installing metrics-server v0.7.0...
# ✓ metrics-server is ready
```

## Step 4: Configure DNS (2 minutes)

Get LoadBalancer IP:

```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller
# EXTERNAL-IP: 51.89.160.25
```

### Option A: OVH DNS

1. Go to: **Web Cloud** → **Domain names** → **Your domain**
2. Click **DNS zone**
3. Add **A record**:
   - **Subdomain**: `@` (root) or `myapp`
   - **Target**: `51.89.160.25`
   - **TTL**: 300
4. Click **Confirm**

**Propagation**: 5-15 minutes

### Option B: External DNS (Cloudflare, etc.)

```
Type: A
Name: myapp.example.com
Value: 51.89.160.25
TTL: Auto
Proxy: Off (disable Cloudflare proxy for cert-manager)
```

### Option C: Wildcard (for multiple apps)

```
Type: A
Name: *.k8s
Value: 51.89.160.25
```

Access apps: `app1.k8s.example.com`, `app2.k8s.example.com`

## Step 5: Deploy Application (2 minutes)

### Deploy Sample App

```bash
./k8s_ovh.sh --deploy-sample hello

# Output:
# ▶ Deploying sample application: hello
# ✓ Application deployed: hello
# ✓ Application pods are ready
```

### Create Ingress with TLS

**Update email and domain**:

```bash
# Update cert-manager email (required for Let's Encrypt)
kubectl edit clusterissuer letsencrypt-prod
# Change: email: admin@example.com → your-email@example.com

# Create Ingress
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hello
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - hello.example.com  # CHANGE THIS
    secretName: hello-tls
  rules:
  - host: hello.example.com  # CHANGE THIS
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hello
            port:
              number: 80
EOF
```

### Wait for Certificate

```bash
# Watch certificate issuance (30-60 seconds)
kubectl get certificate -w

# Output:
# NAME        READY   SECRET       AGE
# hello-tls   False   hello-tls    5s
# hello-tls   True    hello-tls    45s  ← Certificate ready

# Check details
kubectl describe certificate hello-tls
```

### Test

```bash
# HTTP redirects to HTTPS
curl -I http://hello.example.com
# Location: https://hello.example.com

# HTTPS works
curl https://hello.example.com
# Hello from Kubernetes!
```

## Step 6: Deploy Your Own App

### 1. Create Deployment

```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: app
        image: your-registry.io/myapp:v1.0.0  # CHANGE THIS
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: myapp
spec:
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
---
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
        averageUtilization: 70
EOF
```

### 2. Create Ingress

```bash
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myapp
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - myapp.example.com  # CHANGE THIS
    secretName: myapp-tls
  rules:
  - host: myapp.example.com  # CHANGE THIS
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: myapp
            port:
              number: 80
EOF
```

### 3. Verify

```bash
# Watch deployment
kubectl get pods -w

# Check HPA
kubectl get hpa

# Check certificate
kubectl get certificate

# Test endpoint
curl https://myapp.example.com
```

## OVH-Specific Features

### LoadBalancer Annotations

```yaml
apiVersion: v1
kind: Service
metadata:
  name: myapp-lb
  annotations:
    # OVH LoadBalancer flavor
    service.beta.kubernetes.io/ovh-loadbalancer-flavor: "small"  # small, medium, large
    
    # Enable proxy protocol (get real client IP)
    service.beta.kubernetes.io/ovh-loadbalancer-proxy-protocol: "v2"
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
```

### Storage Classes

```bash
# List available storage classes
kubectl get storageclass

# OVH provides:
# - csi-cinder-high-speed (SSD)
# - csi-cinder-classic (HDD)
```

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: myapp-data
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
  storageClassName: csi-cinder-high-speed  # Use SSD
```

### Auto-Scaling Nodes

Enable cluster autoscaler:

```bash
# Via control panel: Kubernetes → Your Cluster → Node Pools → Edit
# Enable: "Auto-scaling"
# Min: 2
# Max: 10

# Nodes scale automatically based on pod resource requests
```

## Monitoring and Debugging

### View Logs

```bash
# Application logs
kubectl logs -f deployment/myapp

# NGINX Ingress logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller -f

# cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager -f
```

### Check Resources

```bash
# Node resources
kubectl top nodes

# Pod resources
kubectl top pods --all-namespaces

# HPA status
kubectl get hpa
```

### Debug Failed Deployment

```bash
# Collect diagnostics
./k8s_ovh.sh --debug --service myapp

# Output: /tmp/k8s-diag-20251213-143045-myapp/
# - deployment.yaml
# - pods.yaml
# - service.yaml
# - ingress.yaml
# - events.txt
# - logs/
```

## Common Issues

### Certificate Not Issued

```bash
# Check certificate status
kubectl describe certificate myapp-tls

# Common causes:
# 1. DNS not propagated
nslookup myapp.example.com

# 2. Firewall blocking port 80 (HTTP-01 challenge)
curl http://myapp.example.com/.well-known/acme-challenge/test

# 3. Wrong email in ClusterIssuer
kubectl edit clusterissuer letsencrypt-prod

# 4. Rate limit (use letsencrypt-staging for testing)
kubectl patch ingress myapp -p '{"metadata":{"annotations":{"cert-manager.io/cluster-issuer":"letsencrypt-staging"}}}'
```

### LoadBalancer Pending

```bash
# Check Service status
kubectl describe svc -n ingress-nginx ingress-nginx-controller

# If stuck in "Pending":
# 1. Check OVH quota (Public Cloud → Quota)
# 2. Verify cluster has public IP quota
# 3. Check OVH API status
```

### Pods Not Scaling

```bash
# Check HPA
kubectl describe hpa myapp

# Common causes:
# 1. metrics-server not running
kubectl get pods -n kube-system -l k8s-app=metrics-server

# 2. No resource requests defined
kubectl get deployment myapp -o jsonpath='{.spec.template.spec.containers[0].resources}'

# 3. Already at max replicas
kubectl get hpa myapp -o jsonpath='{.status.currentReplicas} / {.spec.maxReplicas}'
```

## Cost Management

### Current Usage

```bash
# Node count
kubectl get nodes | wc -l

# LoadBalancer count
kubectl get svc --all-namespaces -o json | jq '[.items[] | select(.spec.type=="LoadBalancer")] | length'

# PVC usage
kubectl get pvc --all-namespaces -o json | jq '[.items[].spec.resources.requests.storage] | join(", ")'
```

### Estimated Monthly Cost

```
Example production setup:
- 3 × b2-7 nodes: 3 × €28.80 = €86.40
- 1 × LoadBalancer: €5.76
- 100GB SSD storage: €10.00
Total: ~€102/month
```

### Optimization

```bash
# 1. Right-size nodes
kubectl top nodes

# 2. Remove unused LoadBalancers
kubectl get svc --all-namespaces --field-selector spec.type=LoadBalancer

# 3. Clean up old PVCs
kubectl get pvc --all-namespaces

# 4. Use cluster autoscaler for dev/staging
# Scales down to min nodes when idle
```

## Next Steps

- **Comprehensive guide**: [k8s.md](k8s.md) - Deep dive into Kubernetes concepts
- **Production patterns**: [README-k8s.md](README-k8s.md) - Best practices and architecture
- **OVH documentation**: [OVH Kubernetes Guide](https://docs.ovh.com/gb/en/kubernetes/)

## Quick Commands Reference

```bash
# Get cluster info
kubectl cluster-info
kubectl get nodes

# Deploy app
kubectl apply -f deployment.yaml

# Check status
kubectl get pods
kubectl get svc
kubectl get ingress
kubectl get certificate

# View logs
kubectl logs -f <pod-name>

# Debug
kubectl describe pod <pod-name>
kubectl exec -it <pod-name> -- /bin/bash

# Scale
kubectl scale deployment myapp --replicas=5

# Update image
kubectl set image deployment/myapp app=myregistry.io/myapp:v2.0.0

# Rollback
kubectl rollout undo deployment/myapp

# Port forward
kubectl port-forward svc/myapp 8080:80
```

## Support

- **OVH Support**: [OVH Help Center](https://help.ovhcloud.com/)
- **Kubernetes Slack**: [slack.k8s.io](https://slack.k8s.io/)
- **CNCF Slack**: [slack.cncf.io](https://slack.cncf.io/)
