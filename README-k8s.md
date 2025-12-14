# Production Kubernetes on OVH Cloud

**Self-hosted container platform with full control and cloud-native integrations**

## Why This Exists

Cloud platforms like Google Cloud Run, AWS App Runner, and Azure Container Apps offer simplicity but lock you into vendor-specific APIs and pricing. When you need:

- **Portability**: Run anywhere (OVH, AWS, GCP, on-premises)
- **Cost control**: Avoid egress fees and vendor markup
- **Full customization**: Control networking, storage, and security policies
- **Compliance**: Meet data residency and regulatory requirements
- **Kubernetes ecosystem**: Access to 1000+ CNCF tools

...you need Kubernetes. This project provides production-ready patterns for OVH Cloud's managed Kubernetes.

## Quick Start

```bash
# 1. Create OVH Kubernetes cluster
# Go to OVH Cloud Control Panel → Public Cloud → Kubernetes
# Create cluster with 2+ nodes (recommended: b2-7 instances)

# 2. Download kubeconfig
# Control Panel → Kubernetes → Your Cluster → Kubeconfig
# Save to ~/.kube/config or set KUBECONFIG environment variable

# 3. Verify connectivity
kubectl cluster-info
kubectl get nodes

# 4. Install complete stack
./k8s_ovh.sh --install-all

# 5. Get LoadBalancer IP
kubectl get svc -n ingress-nginx ingress-nginx-controller

# 6. Point DNS to LoadBalancer IP
# In your DNS provider, create A record:
# myapp.example.com → 51.89.xxx.xxx

# 7. Deploy sample application
./k8s_ovh.sh --deploy-sample hello

# 8. Create Ingress (replace domain)
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hello
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-staging"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - hello.example.com
    secretName: hello-tls
  rules:
  - host: hello.example.com
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

# 9. Test
curl https://hello.example.com
```

## Architecture Overview

```
                     ┌─────────────────────────────────┐
                     │         Internet                │
                     └────────────┬────────────────────┘
                                  │
                                  ▼
                     ┌─────────────────────────────────┐
                     │   OVH Cloud LoadBalancer        │
                     │   Type: LoadBalancer            │
                     │   External IP: 51.89.xxx.xxx    │
                     └────────────┬────────────────────┘
                                  │
                                  ▼
                     ┌─────────────────────────────────┐
                     │   NGINX Ingress Controller      │
                     │   - TLS termination (443→80)    │
                     │   - Virtual host routing        │
                     │   - Path-based routing          │
                     └────────────┬────────────────────┘
                                  │
              ┌───────────────────┼───────────────────┐
              ▼                   ▼                   ▼
    ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
    │ Service A       │ │ Service B       │ │ Service C       │
    │ ClusterIP       │ │ ClusterIP       │ │ ClusterIP       │
    │ Port: 80        │ │ Port: 80        │ │ Port: 80        │
    └────────┬────────┘ └────────┬────────┘ └────────┬────────┘
             │                   │                   │
    ┌────────┴────────┐ ┌────────┴────────┐ ┌────────┴────────┐
    │ Pods (HPA 2-10) │ │ Pods (HPA 1-5)  │ │ Pods (Static 3) │
    │ CPU: 70%        │ │ Memory: 80%     │ │ No autoscaling  │
    └─────────────────┘ └─────────────────┘ └─────────────────┘
```

### Request Flow

1. **DNS Resolution**: Client resolves `myapp.example.com` → `51.89.xxx.xxx`
2. **LoadBalancer**: OVH Cloud LoadBalancer routes to NGINX Ingress NodePort
3. **Ingress**: NGINX terminates TLS, routes based on `Host` header + path
4. **Service**: ClusterIP load balances to healthy Pod replicas
5. **Pod**: Container handles request, returns response

## Components

| Component | Version | Purpose |
|-----------|---------|---------|
| **NGINX Ingress** | v1.10.0 | HTTP/HTTPS routing, TLS termination |
| **cert-manager** | v1.14.0 | Automated Let's Encrypt certificates |
| **metrics-server** | v0.7.0 | Resource metrics for HPA |
| **Prometheus + Grafana** | latest | Monitoring and observability (optional) |

## Features

### ✓ Horizontal Pod Autoscaling (HPA)

Scale pods automatically based on CPU/memory:

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
        averageUtilization: 70  # Scale at 70% CPU
```

**How it works**:
- metrics-server collects resource usage every 15s
- HPA controller checks metrics every 30s
- Scale up: current > target → add pods (up to max)
- Scale down: current < target → remove pods (down to min)
- Cooldown: 3 minutes between scale-up, 5 minutes for scale-down

### ✓ Automated TLS Certificates

cert-manager handles Let's Encrypt certificates:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myapp
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - myapp.example.com
    secretName: myapp-tls  # cert-manager creates this
```

**Certificate lifecycle**:
1. Ingress created with `cert-manager.io/cluster-issuer` annotation
2. cert-manager detects and creates Certificate resource
3. HTTP-01 challenge: cert-manager creates temporary Ingress route
4. Let's Encrypt validates domain ownership
5. Certificate issued and stored in Secret `myapp-tls`
6. Auto-renewal 30 days before expiry

### ✓ OVH LoadBalancer Integration

OVH Cloud Controller Manager automatically provisions LoadBalancers:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-loadbalancer
spec:
  type: LoadBalancer  # OVH provisions external IP
  ports:
  - port: 80
    targetPort: 8080
```

**Features**:
- Automatic public IP assignment
- Health checks to backend nodes
- DDoS protection included
- IPv4 and IPv6 support

## Common Patterns

### Multi-Environment Setup

Use namespaces for environment isolation:

```bash
# Create environments
kubectl create namespace staging
kubectl create namespace production

# Deploy to staging
kubectl apply -f deployment.yaml -n staging

# Deploy to production
kubectl apply -f deployment.yaml -n production
```

### Blue-Green Deployment

Zero-downtime deployments:

```bash
# Deploy green (new version)
kubectl apply -f deployment-green.yaml

# Wait for green to be ready
kubectl wait --for=condition=ready pod -l version=green --timeout=120s

# Switch traffic (update Service selector)
kubectl patch service myapp -p '{"spec":{"selector":{"version":"green"}}}'

# Remove blue (old version)
kubectl delete deployment myapp-blue
```

### Health Checks

Define liveness and readiness probes:

```yaml
spec:
  containers:
  - name: app
    livenessProbe:
      httpGet:
        path: /healthz
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
      failureThreshold: 3  # Restart after 3 failures
    
    readinessProbe:
      httpGet:
        path: /ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
      failureThreshold: 2  # Remove from Service after 2 failures
```

## Monitoring

### Install Prometheus + Grafana

```bash
./k8s_ovh.sh --install-monitoring

# Access Grafana
kubectl port-forward -n monitoring svc/grafana 3000:3000

# Open: http://localhost:3000
# Credentials: admin / admin
```

### View Metrics

```bash
# Node metrics
kubectl top nodes

# Pod metrics
kubectl top pods --all-namespaces

# HPA status
kubectl get hpa -w  # Watch autoscaling
```

## Troubleshooting

### Collect Diagnostics

```bash
./k8s_ovh.sh --debug --service myapp
```

Saves to `/tmp/k8s-diag-<timestamp>-myapp/`:
- deployment.yaml
- pods.yaml, pods.txt
- service.yaml
- hpa.yaml
- ingress.yaml
- events.txt
- Pod logs
- NGINX Ingress logs

### Common Issues

**Pods not starting**:
```bash
kubectl describe pod <pod-name>
kubectl logs <pod-name>
```

**Service unreachable**:
```bash
# Check Service endpoints
kubectl get endpoints <service-name>

# Test from another Pod
kubectl run -it --rm debug --image=busybox --restart=Never -- wget -O- http://myapp.default.svc.cluster.local
```

**Ingress not working**:
```bash
# Check Ingress status
kubectl describe ingress myapp

# View NGINX logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller

# Test NGINX config
kubectl exec -n ingress-nginx -it <nginx-pod> -- nginx -T
```

**Certificate not issued**:
```bash
# Check Certificate status
kubectl get certificate
kubectl describe certificate myapp-tls

# Check CertificateRequest
kubectl get certificaterequest
kubectl describe certificaterequest <name>

# Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager
```

## Cost Optimization

### OVH Pricing (as of Dec 2025)

| Resource | Price | Notes |
|----------|-------|-------|
| Control Plane | Free | Managed by OVH |
| Worker Node (b2-7) | ~€0.04/hour | 2 vCPU, 7GB RAM |
| LoadBalancer | ~€0.008/hour | Per LoadBalancer |
| Storage (Block) | ~€0.10/GB/month | Persistent volumes |
| Bandwidth | Free egress | Within OVH network |

**Example monthly cost** (3-node cluster):
- 3 × b2-7 nodes: €86.40
- 1 × LoadBalancer: €5.76
- 100GB storage: €10.00
- **Total**: ~€102/month

### Optimization Tips

1. **Right-size nodes**: Use `kubectl top nodes` to check utilization
2. **Scale down dev/staging**: Use cluster autoscaler or manual scaling
3. **Use PVC sparingly**: Block storage adds up quickly
4. **Leverage OVH Object Storage**: Cheaper than PVCs for large datasets
5. **Monitor idle resources**: Remove unused Services, PVCs, LoadBalancers

## Security Checklist

- [ ] Enable RBAC (Role-Based Access Control)
- [ ] Use NetworkPolicies to restrict pod-to-pod traffic
- [ ] Scan images for vulnerabilities (Trivy, Snyk)
- [ ] Rotate secrets regularly
- [ ] Use PodSecurityPolicies or Pod Security Standards
- [ ] Enable audit logging
- [ ] Restrict API server access (IP whitelist)
- [ ] Use private node pools for sensitive workloads
- [ ] Implement backup strategy (Velero)
- [ ] Monitor with Prometheus Alertmanager

## OVH Cloud vs Other Providers

| Feature | OVH Cloud | GKE | EKS | AKS |
|---------|-----------|-----|-----|-----|
| **Control Plane** | Free | Paid ($0.10/h) | Paid ($0.10/h) | Free |
| **Node Pricing** | Competitive | Higher | Higher | Competitive |
| **Egress Bandwidth** | Free (OVH network) | Paid | Paid | Paid |
| **Regions** | EU-focused | Global | Global | Global |
| **Data Residency** | EU GDPR-compliant | Multi-region | Multi-region | Multi-region |
| **Support** | 24/7 | Enterprise | Enterprise | Enterprise |

**Choose OVH when**:
- EU data residency required (GDPR, regulations)
- Cost-sensitive (free egress, competitive pricing)
- Already using OVH infrastructure (vRack, Object Storage)

## Production Checklist

Before going live:

- [ ] **Multi-node cluster**: At least 3 nodes across availability zones
- [ ] **Resource limits**: Set CPU/memory limits on all containers
- [ ] **Health checks**: Define liveness and readiness probes
- [ ] **HPA configured**: Enable autoscaling for variable workloads
- [ ] **TLS enabled**: Use `letsencrypt-prod` (not staging)
- [ ] **Monitoring**: Install Prometheus + Grafana
- [ ] **Backups**: Set up Velero or equivalent
- [ ] **CI/CD**: Automate deployments (GitLab CI, GitHub Actions)
- [ ] **Secrets management**: Use external secrets operator or sealed secrets
- [ ] **Logging**: Centralized logging (ELK, Loki, or OVH Logs Data Platform)

## Next Steps

- Read [k8s.md](k8s.md) for comprehensive technical guide
- Read [k8s-ovh.md](k8s-ovh.md) for OVH-specific setup
- Explore [Kubernetes documentation](https://kubernetes.io/docs/)
- Join [CNCF Slack](https://slack.cncf.io/) for community support

## Contributing

Found an issue or have improvements? Contributions welcome!

## License

Apache License 2.0 - see LICENSE file for details
