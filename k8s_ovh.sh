#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════
# Production Kubernetes Stack Installer for OVH Cloud
# ═══════════════════════════════════════════════════════════════════════════
#
# WHY THIS EXISTS
# ───────────────
# Managing Kubernetes applications requires juggling multiple components:
# - Ingress controller (routing HTTP/HTTPS traffic)
# - TLS certificate management (Let's Encrypt automation)
# - Horizontal Pod Autoscaler (scaling based on metrics)
# - Monitoring and observability (Prometheus, Grafana)
# - LoadBalancer integration (cloud provider networking)
#
# This script provides a production-ready foundation for deploying
# containerized applications on OVH Cloud's managed Kubernetes service.
# Unlike platform services (Cloud Run, App Engine), you control the entire
# stack while benefiting from managed control plane, automatic updates, and
# OVH's cloud-native LoadBalancer integration.
#
# ARCHITECTURE OVERVIEW
# ─────────────────────
#
#     Internet
#        │
#        ├─ DNS (A/AAAA record) ──┐
#        │                         │
#        ▼                         ▼
#  ┌──────────────────────────────────────┐
#  │   OVH LoadBalancer (type=LoadBalancer)│
#  │   External IP: xxx.xxx.xxx.xxx       │
#  └──────────────────────────────────────┘
#               │
#               ▼
#  ┌──────────────────────────────────────┐
#  │   NGINX Ingress Controller           │
#  │   - TLS termination (443→80)         │
#  │   - Virtual host routing             │
#  │   - Path-based routing               │
#  └──────────────────────────────────────┘
#               │
#       ┌───────┴────────┬──────────────┐
#       ▼                ▼              ▼
#  ┌─────────┐     ┌─────────┐    ┌─────────┐
#  │ Service │     │ Service │    │ Service │
#  │ ClusterIP│    │ ClusterIP│   │ ClusterIP│
#  └─────────┘     └─────────┘    └─────────┘
#       │                │              │
#  ┌────┴────┐     ┌────┴────┐    ┌────┴────┐
#  │ Pod(s)  │     │ Pod(s)  │    │ Pod(s)  │
#  │ HPA→2-10│     │ HPA→1-5 │    │ Static=3│
#  └─────────┘     └─────────┘    └─────────┘
#
# Request Flow:
#   1. Client → DNS resolves to LoadBalancer external IP
#   2. LoadBalancer → NGINX Ingress (NodePort or LoadBalancer service)
#   3. NGINX → Routes based on Host header + path
#   4. Service → Load balances to Pod replicas
#   5. Pod → Application container handles request
#
# Autoscaling:
#   - HPA (Horizontal Pod Autoscaler) monitors CPU/memory metrics
#   - metrics-server provides real-time resource usage
#   - Scale from min to max replicas based on thresholds
#
# TLS:
#   - cert-manager automates Let's Encrypt certificate issuance
#   - Automatic renewal before expiry
#   - HTTP-01 or DNS-01 challenge support
#
# PREREQUISITES
# ─────────────
# - OVH Cloud account with Kubernetes cluster created
# - kubectl configured to access your cluster (kubeconfig)
# - Cluster with at least 2 worker nodes (recommended)
# - Domain name with DNS managed by OVH or external provider
#
# COMPONENTS INSTALLED
# ────────────────────
# - NGINX Ingress Controller v1.10+ (HTTP/HTTPS routing)
# - cert-manager v1.14+ (automated TLS certificates)
# - metrics-server (resource metrics for HPA)
# - Prometheus + Grafana (optional monitoring stack)
#
# USAGE
# ─────
# Basic installation:
#   ./k8s_ovh.sh --install-all
#
# With monitoring:
#   ./k8s_ovh.sh --install-all --install-monitoring
#
# Deploy sample app:
#   ./k8s_ovh.sh --deploy-sample
#
# Debug failed deployment:
#   ./k8s_ovh.sh --debug --service hello
#
# ═══════════════════════════════════════════════════════════════════════════

set -euo pipefail

# ═══════════════════════════════════════════════════════════════════════════
# CONFIGURATION
# ═══════════════════════════════════════════════════════════════════════════

readonly NGINX_INGRESS_VERSION="v1.10.0"
readonly CERT_MANAGER_VERSION="v1.14.0"
readonly METRICS_SERVER_VERSION="v0.7.0"

readonly NGINX_NAMESPACE="ingress-nginx"
readonly CERT_MANAGER_NAMESPACE="cert-manager"
readonly MONITORING_NAMESPACE="monitoring"

# Colors and symbols for CLI output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m' # No Color

readonly SYMBOL_INFO="▶"
readonly SYMBOL_SUCCESS="✓"
readonly SYMBOL_WARN="⚠"
readonly SYMBOL_ERROR="✗"

# ═══════════════════════════════════════════════════════════════════════════
# OUTPUT FUNCTIONS
# ═══════════════════════════════════════════════════════════════════════════

info() {
    echo -e "${BLUE}${SYMBOL_INFO}${NC} $*" >&2
}

success() {
    echo -e "${GREEN}${SYMBOL_SUCCESS}${NC} $*" >&2
}

warn() {
    echo -e "${YELLOW}${SYMBOL_WARN}${NC} $*" >&2
}

error() {
    echo -e "${RED}${SYMBOL_ERROR}${NC} $*" >&2
}

# ═══════════════════════════════════════════════════════════════════════════
# PREREQUISITE CHECKS
# ═══════════════════════════════════════════════════════════════════════════

require_kubectl() {
    if ! command -v kubectl &>/dev/null; then
        error "kubectl not found"
        info "Install: https://kubernetes.io/docs/tasks/tools/"
        exit 1
    fi
    success "kubectl found: $(kubectl version --client --short 2>/dev/null || kubectl version --client)"
}

check_cluster_connection() {
    info "Checking cluster connection..."
    if ! kubectl cluster-info &>/dev/null; then
        error "Cannot connect to Kubernetes cluster"
        info "Ensure your kubeconfig is configured correctly"
        info "For OVH: Download kubeconfig from OVH Cloud Control Panel"
        exit 1
    fi
    local context=$(kubectl config current-context)
    success "Connected to cluster: ${context}"
}

check_cluster_nodes() {
    info "Checking cluster nodes..."
    local node_count=$(kubectl get nodes --no-headers 2>/dev/null | wc -l)
    if [[ ${node_count} -eq 0 ]]; then
        error "No nodes found in cluster"
        exit 1
    fi
    success "Cluster has ${node_count} node(s)"
    
    if [[ ${node_count} -lt 2 ]]; then
        warn "Single-node cluster detected - production workloads need 2+ nodes"
    fi
}

# ═══════════════════════════════════════════════════════════════════════════
# INSTALLATION FUNCTIONS
# ═══════════════════════════════════════════════════════════════════════════

install_nginx_ingress() {
    info "Installing NGINX Ingress Controller ${NGINX_INGRESS_VERSION}..."
    
    # Create namespace
    kubectl create namespace ${NGINX_NAMESPACE} --dry-run=client -o yaml | kubectl apply -f - >/dev/null
    
    # Install via Helm (official method) or kubectl apply
    # Using kubectl apply for simplicity and reproducibility
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/${NGINX_INGRESS_VERSION}/deploy/static/provider/cloud/deploy.yaml >/dev/null 2>&1 || {
        error "Failed to install NGINX Ingress Controller"
        return 1
    }
    
    success "NGINX Ingress Controller manifest applied"
    
    # Wait for deployment
    info "Waiting for NGINX Ingress Controller to be ready (max 180s)..."
    if kubectl wait --namespace ${NGINX_NAMESPACE} \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=180s >/dev/null 2>&1; then
        success "NGINX Ingress Controller is ready"
    else
        warn "NGINX Ingress Controller pods may still be starting"
    fi
    
    # Wait for LoadBalancer external IP
    info "Waiting for LoadBalancer external IP (max 120s)..."
    local elapsed=0
    local max_wait=120
    while [[ ${elapsed} -lt ${max_wait} ]]; do
        local external_ip=$(kubectl get svc -n ${NGINX_NAMESPACE} ingress-nginx-controller \
            -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
        
        if [[ -n "${external_ip}" ]]; then
            success "LoadBalancer external IP: ${external_ip}"
            info "Point your DNS A record to this IP"
            return 0
        fi
        
        sleep 5
        elapsed=$((elapsed + 5))
    done
    
    warn "LoadBalancer IP not yet assigned - check OVH Cloud console"
}

install_cert_manager() {
    info "Installing cert-manager ${CERT_MANAGER_VERSION}..."
    
    # Create namespace
    kubectl create namespace ${CERT_MANAGER_NAMESPACE} --dry-run=client -o yaml | kubectl apply -f - >/dev/null
    
    # Install cert-manager CRDs
    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.crds.yaml >/dev/null 2>&1 || {
        error "Failed to install cert-manager CRDs"
        return 1
    }
    
    # Install cert-manager
    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml >/dev/null 2>&1 || {
        error "Failed to install cert-manager"
        return 1
    }
    
    success "cert-manager manifests applied"
    
    # Wait for cert-manager to be ready
    info "Waiting for cert-manager to be ready (max 120s)..."
    if kubectl wait --namespace ${CERT_MANAGER_NAMESPACE} \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/instance=cert-manager \
        --timeout=120s >/dev/null 2>&1; then
        success "cert-manager is ready"
    else
        warn "cert-manager pods may still be starting"
    fi
    
    # Create Let's Encrypt ClusterIssuer (staging for testing)
    info "Creating Let's Encrypt ClusterIssuer..."
    kubectl apply -f - <<'EOF' >/dev/null
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: admin@example.com  # CHANGE THIS
    privateKeySecretRef:
      name: letsencrypt-staging
    solvers:
    - http01:
        ingress:
          class: nginx
---
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
    success "ClusterIssuers created (update email in production)"
}

install_metrics_server() {
    info "Installing metrics-server ${METRICS_SERVER_VERSION}..."
    
    # Install metrics-server
    kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/download/${METRICS_SERVER_VERSION}/components.yaml >/dev/null 2>&1 || {
        error "Failed to install metrics-server"
        return 1
    }
    
    success "metrics-server manifest applied"
    
    # Wait for metrics-server to be ready
    info "Waiting for metrics-server to be ready (max 90s)..."
    if kubectl wait --namespace kube-system \
        --for=condition=ready pod \
        --selector=k8s-app=metrics-server \
        --timeout=90s >/dev/null 2>&1; then
        success "metrics-server is ready"
    else
        warn "metrics-server pods may still be starting"
    fi
}

install_monitoring() {
    info "Installing Prometheus + Grafana monitoring stack..."
    
    # Create namespace
    kubectl create namespace ${MONITORING_NAMESPACE} --dry-run=client -o yaml | kubectl apply -f - >/dev/null
    
    # Install kube-prometheus-stack via manifests
    # Note: Production environments should use Helm for easier upgrades
    info "Using kube-prometheus-stack (this may take a few minutes)..."
    
    # Clone and apply manifests (simplified for demo)
    kubectl apply --server-side -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/main/manifests/setup/0namespace-namespace.yaml >/dev/null 2>&1 || true
    kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/main/manifests/setup/ >/dev/null 2>&1 || true
    
    # Wait for CRDs to be established
    sleep 10
    
    kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/main/manifests/ >/dev/null 2>&1 || {
        warn "Monitoring stack installation encountered issues - check manually"
        return 0
    }
    
    success "Monitoring stack applied"
    info "Access Grafana: kubectl port-forward -n monitoring svc/grafana 3000:3000"
    info "Default credentials: admin / admin"
}

# ═══════════════════════════════════════════════════════════════════════════
# DEPLOYMENT FUNCTIONS
# ═══════════════════════════════════════════════════════════════════════════

deploy_sample_app() {
    local app_name="${1:-hello}"
    local namespace="${2:-default}"
    local replicas="${3:-2}"
    
    info "Deploying sample application: ${app_name}"
    
    # Create deployment
    kubectl apply -f - <<EOF >/dev/null
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${app_name}
  namespace: ${namespace}
  labels:
    app: ${app_name}
spec:
  replicas: ${replicas}
  selector:
    matchLabels:
      app: ${app_name}
  template:
    metadata:
      labels:
        app: ${app_name}
    spec:
      containers:
      - name: app
        image: us-docker.pkg.dev/cloudrun/container/hello:latest
        ports:
        - containerPort: 8080
          name: http
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
        livenessProbe:
          httpGet:
            path: /
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: ${app_name}
  namespace: ${namespace}
spec:
  selector:
    app: ${app_name}
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ${app_name}
  namespace: ${namespace}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ${app_name}
  minReplicas: 1
  maxReplicas: 10
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
EOF
    
    success "Application deployed: ${app_name}"
    
    # Wait for deployment
    info "Waiting for pods to be ready..."
    if kubectl wait --namespace ${namespace} \
        --for=condition=ready pod \
        --selector=app=${app_name} \
        --timeout=120s >/dev/null 2>&1; then
        success "Application pods are ready"
    else
        warn "Pods may still be starting - check with: kubectl get pods -n ${namespace}"
    fi
    
    # Get LoadBalancer IP for Ingress instruction
    local lb_ip=$(kubectl get svc -n ${NGINX_NAMESPACE} ingress-nginx-controller \
        -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    
    info "To expose via Ingress, create an Ingress resource:"
    cat <<EOF

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ${app_name}
  namespace: ${namespace}
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-staging"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - ${app_name}.example.com  # CHANGE THIS
    secretName: ${app_name}-tls
  rules:
  - host: ${app_name}.example.com  # CHANGE THIS
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: ${app_name}
            port:
              number: 80

Then point your DNS: ${app_name}.example.com → ${lb_ip}
EOF
}

# ═══════════════════════════════════════════════════════════════════════════
# DIAGNOSTICS
# ═══════════════════════════════════════════════════════════════════════════

collect_diagnostics() {
    local service_name="${1}"
    local namespace="${2:-default}"
    local diag_dir="/tmp/k8s-diag-$(date +%Y%m%d-%H%M%S)-${service_name}"
    
    info "Collecting diagnostics for service: ${service_name}"
    mkdir -p "${diag_dir}"
    
    # Deployment
    kubectl get deployment ${service_name} -n ${namespace} -o yaml > "${diag_dir}/deployment.yaml" 2>/dev/null || true
    
    # Pods
    kubectl get pods -n ${namespace} -l app=${service_name} -o wide > "${diag_dir}/pods.txt" 2>/dev/null || true
    kubectl get pods -n ${namespace} -l app=${service_name} -o yaml > "${diag_dir}/pods.yaml" 2>/dev/null || true
    
    # Service
    kubectl get svc ${service_name} -n ${namespace} -o yaml > "${diag_dir}/service.yaml" 2>/dev/null || true
    
    # HPA
    kubectl get hpa ${service_name} -n ${namespace} -o yaml > "${diag_dir}/hpa.yaml" 2>/dev/null || true
    
    # Ingress
    kubectl get ingress -n ${namespace} -o yaml > "${diag_dir}/ingress.yaml" 2>/dev/null || true
    
    # Events
    kubectl get events -n ${namespace} --sort-by='.lastTimestamp' > "${diag_dir}/events.txt" 2>/dev/null || true
    
    # Pod logs
    for pod in $(kubectl get pods -n ${namespace} -l app=${service_name} -o name 2>/dev/null); do
        local pod_name=$(basename ${pod})
        kubectl logs -n ${namespace} ${pod_name} --tail=500 > "${diag_dir}/${pod_name}.log" 2>/dev/null || true
    done
    
    # NGINX Ingress Controller logs
    kubectl logs -n ${NGINX_NAMESPACE} -l app.kubernetes.io/component=controller --tail=200 > "${diag_dir}/nginx-ingress.log" 2>/dev/null || true
    
    success "Diagnostics saved to: ${diag_dir}"
    info "Review files:"
    ls -lh "${diag_dir}"
}

# ═══════════════════════════════════════════════════════════════════════════
# MAIN SCRIPT
# ═══════════════════════════════════════════════════════════════════════════

print_usage() {
    cat <<EOF
${CYAN}Production Kubernetes Stack Installer for OVH Cloud${NC}

${YELLOW}USAGE:${NC}
  $0 [OPTIONS]

${YELLOW}OPTIONS:${NC}
  --install-all           Install complete stack (NGINX, cert-manager, metrics-server)
  --install-nginx         Install only NGINX Ingress Controller
  --install-cert-manager  Install only cert-manager
  --install-metrics       Install only metrics-server
  --install-monitoring    Install Prometheus + Grafana monitoring
  
  --deploy-sample [NAME]  Deploy sample application (default: hello)
  --debug --service NAME  Collect diagnostics for a service
  
  --help                  Show this help message

${YELLOW}EXAMPLES:${NC}
  # Complete installation
  $0 --install-all

  # Install with monitoring
  $0 --install-all --install-monitoring

  # Deploy sample app
  $0 --deploy-sample myapp

  # Debug deployment
  $0 --debug --service myapp

${YELLOW}PREREQUISITES:${NC}
  - OVH Cloud Kubernetes cluster (active and accessible)
  - kubectl configured with cluster credentials
  - At least 2 worker nodes (recommended for HA)

${YELLOW}COMPONENTS:${NC}
  - NGINX Ingress Controller ${NGINX_INGRESS_VERSION}
  - cert-manager ${CERT_MANAGER_VERSION}
  - metrics-server ${METRICS_SERVER_VERSION}

${YELLOW}LEARN MORE:${NC}
  - Read k8s.md for comprehensive guide
  - Read k8s-ovh.md for OVH-specific setup
  - OVH Kubernetes: https://www.ovhcloud.com/en/public-cloud/kubernetes/

EOF
}

main() {
    local do_install_all=false
    local do_install_nginx=false
    local do_install_cert_manager=false
    local do_install_metrics=false
    local do_install_monitoring=false
    local do_deploy_sample=false
    local do_diagnostics=false
    local sample_name="hello"
    local debug_service=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --install-all)
                do_install_all=true
                shift
                ;;
            --install-nginx)
                do_install_nginx=true
                shift
                ;;
            --install-cert-manager)
                do_install_cert_manager=true
                shift
                ;;
            --install-metrics)
                do_install_metrics=true
                shift
                ;;
            --install-monitoring)
                do_install_monitoring=true
                shift
                ;;
            --deploy-sample)
                do_deploy_sample=true
                if [[ -n "${2:-}" ]] && [[ ! "${2}" =~ ^-- ]]; then
                    sample_name="$2"
                    shift
                fi
                shift
                ;;
            --debug)
                do_diagnostics=true
                shift
                ;;
            --service)
                debug_service="${2:-}"
                shift 2
                ;;
            --help)
                print_usage
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # Show usage if no options
    if [[ "${do_install_all}" == "false" ]] && \
       [[ "${do_install_nginx}" == "false" ]] && \
       [[ "${do_install_cert_manager}" == "false" ]] && \
       [[ "${do_install_metrics}" == "false" ]] && \
       [[ "${do_install_monitoring}" == "false" ]] && \
       [[ "${do_deploy_sample}" == "false" ]] && \
       [[ "${do_diagnostics}" == "false" ]]; then
        print_usage
        exit 0
    fi
    
    # Prerequisites
    require_kubectl
    check_cluster_connection
    check_cluster_nodes
    
    echo ""
    info "Starting installation..."
    echo ""
    
    # Install components
    if [[ "${do_install_all}" == "true" ]] || [[ "${do_install_nginx}" == "true" ]]; then
        install_nginx_ingress
        echo ""
    fi
    
    if [[ "${do_install_all}" == "true" ]] || [[ "${do_install_cert_manager}" == "true" ]]; then
        install_cert_manager
        echo ""
    fi
    
    if [[ "${do_install_all}" == "true" ]] || [[ "${do_install_metrics}" == "true" ]]; then
        install_metrics_server
        echo ""
    fi
    
    if [[ "${do_install_monitoring}" == "true" ]]; then
        install_monitoring
        echo ""
    fi
    
    # Deploy sample
    if [[ "${do_deploy_sample}" == "true" ]]; then
        deploy_sample_app "${sample_name}"
        echo ""
    fi
    
    # Diagnostics
    if [[ "${do_diagnostics}" == "true" ]]; then
        if [[ -z "${debug_service}" ]]; then
            error "Must specify --service NAME with --debug"
            exit 1
        fi
        collect_diagnostics "${debug_service}"
        echo ""
    fi
    
    # Print summary
    print_summary
}

print_summary() {
    cat <<EOF
${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}
${GREEN}${SYMBOL_SUCCESS} Installation Complete${NC}
${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}

${CYAN}Next Steps:${NC}

1. Get LoadBalancer external IP:
   ${YELLOW}kubectl get svc -n ingress-nginx ingress-nginx-controller${NC}

2. Point your DNS A record to the LoadBalancer IP

3. Deploy your application:
   ${YELLOW}kubectl apply -f deployment.yaml${NC}

4. Create Ingress with TLS:
   ${YELLOW}kubectl apply -f ingress.yaml${NC}

5. Monitor certificate issuance:
   ${YELLOW}kubectl get certificate --all-namespaces -w${NC}

${CYAN}Useful Commands:${NC}

  # Check pods across all namespaces
  ${YELLOW}kubectl get pods -A${NC}

  # View NGINX Ingress logs
  ${YELLOW}kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller${NC}

  # Check HPA status
  ${YELLOW}kubectl get hpa${NC}

  # View metrics
  ${YELLOW}kubectl top nodes && kubectl top pods${NC}

  # Access Grafana (if monitoring installed)
  ${YELLOW}kubectl port-forward -n monitoring svc/grafana 3000:3000${NC}

${CYAN}Documentation:${NC}
  - Comprehensive guide: k8s.md
  - OVH-specific setup: k8s-ovh.md
  - README: README-k8s.md

${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}
EOF
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
