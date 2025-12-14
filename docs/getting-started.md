# Getting Started

This guide covers installing k8s-agent-stack for local development and production environments.

## Prerequisites

Before installing, ensure you have:

```bash
# Required
kubectl version --client    # Kubernetes CLI
kn version                  # Knative CLI
docker --version            # Container runtime

# Optional
helm version                # Package manager
```

### Installing Prerequisites

**macOS:**
```bash
brew install kubectl
brew install knative/client/kn
brew install --cask orbstack  # Recommended for M1/M2
# OR
brew install --cask docker
```

**Linux:**
```bash
# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# kn CLI
curl -L https://github.com/knative/client/releases/download/knative-v1.20.0/kn-linux-amd64 \
  -o /usr/local/bin/kn
chmod +x /usr/local/bin/kn
```

📖 See [Tool Installation Guide](tool-installation.md) for detailed instructions.

---

## Local Development (OrbStack/kind)

Best for development and testing on your local machine.

### Step 1: Clone Repository

```bash
git clone https://github.com/raphaelmansuy/k8s-agent-stack.git
cd k8s-agent-stack
```

### Step 2: Run Installer

```bash
./knative_orbstack.sh
```

This installs:
- Knative Serving (v1.12+)
- Contour ingress controller
- Envoy proxy
- metrics-server (optional)

**Installation time**: ~3-5 minutes ⏱️

### Step 3: Verify Installation

```bash
# Check all components
kubectl get pods -n knative-serving
kubectl get pods -n projectcontour

# Run diagnostics
./knative_orbstack.sh --debug
```

Expected output:
```
✓ Knative Serving: Ready
✓ Contour: Ready
✓ Envoy: Ready
✓ DNS: Configured
```

### Step 4: Deploy Test Agent

```bash
# Create a simple test service
kn service create hello \
  --image=gcr.io/knative-samples/helloworld-go \
  --port=8080

# Test it
curl $(kn service describe hello -o url)
```

---

## Production Deployment

For GKE, EKS, AKS, or on-premises Kubernetes clusters.

### Step 1: Configure kubectl

Ensure kubectl is connected to your production cluster:

```bash
kubectl cluster-info
kubectl get nodes
```

### Step 2: Run Installer with Production Settings

```bash
./knative_orbstack.sh --install-metrics --production
```

### Step 3: Configure Domain

```bash
# Edit the config-domain ConfigMap
kubectl edit configmap config-domain -n knative-serving

# Add your domain:
# data:
#   example.com: ""
```

### Step 4: Install TLS (cert-manager)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

### Step 5: Configure Ingress

For cloud providers, the LoadBalancer is typically auto-provisioned. Check the external IP:

```bash
kubectl get svc envoy -n projectcontour
```

### Production Checklist

- [ ] Domain configured in config-domain
- [ ] TLS certificates via cert-manager
- [ ] Prometheus + Grafana for monitoring
- [ ] Resource limits on agents
- [ ] RBAC policies configured
- [ ] Secrets management (External Secrets or Sealed Secrets)

📖 See full [Production Checklist](deployment-guide.md#production-checklist)

---

## Kubernetes Platforms

### Google GKE

```bash
# Create cluster
gcloud container clusters create k8s-agent-stack \
  --num-nodes=3 \
  --machine-type=e2-standard-4

# Get credentials
gcloud container clusters get-credentials k8s-agent-stack

# Install
./knative_orbstack.sh --install-metrics --production
```

### Amazon EKS

```bash
# Create cluster (using eksctl)
eksctl create cluster --name k8s-agent-stack --nodes 3

# Install
./knative_orbstack.sh --install-metrics --production
```

### Azure AKS

```bash
# Create cluster
az aks create -g myResourceGroup -n k8s-agent-stack --node-count 3

# Get credentials
az aks get-credentials -g myResourceGroup -n k8s-agent-stack

# Install
./knative_orbstack.sh --install-metrics --production
```

### On-Premises

For bare-metal or on-prem clusters, you may need MetalLB for load balancing:

```bash
# Install MetalLB
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.12/config/manifests/metallb-native.yaml

# Configure IP address pool (edit as needed)
cat <<EOF | kubectl apply -f -
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: default
  namespace: metallb-system
spec:
  addresses:
  - 192.168.1.240-192.168.1.250
EOF
```

---

## Next Steps

After installation:

1. **Deploy your first agent**: [Deployment Guide](deployment-guide.md)
2. **Build custom agents**: [Building ADK Agents](building-google-adk-agents-for-kagent.md)
3. **Understand the architecture**: [Architecture](architecture.md)

## Troubleshooting

If installation fails, run diagnostics:

```bash
./knative_orbstack.sh --debug
```

Common issues:

| Issue | Solution |
|-------|----------|
| Pods not starting | Check node resources: `kubectl top nodes` |
| Ingress not accessible | Verify LoadBalancer IP: `kubectl get svc -n projectcontour` |
| DNS not resolving | Check CoreDNS: `kubectl get pods -n kube-system -l k8s-app=kube-dns` |

📖 See [Troubleshooting Guide](troubleshooting.md) for more solutions.

---

[← Back to Documentation Index](README.md) • [Architecture](architecture.md) • [Deployment Guide](deployment-guide.md) • [Main README](../README.md)
