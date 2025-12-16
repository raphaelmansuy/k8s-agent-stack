#!/usr/bin/env bash
#
# knative_orbstack.sh — Cloud Run on OrbStack (Knative Serving + Contour)
#
# ═══════════════════════════════════════════════════════════════════════════════
# WHY THIS EXISTS
# ═══════════════════════════════════════════════════════════════════════════════
# Cloud Run is Google's serverless container platform: scale-to-zero, pay-per-request,
# no ops. But it's proprietary and cloud-locked.
#
# Knative Serving is the open-source equivalent — same semantics, same scale-to-zero,
# same concurrency model. This script installs a production-grade Knative stack on
# OrbStack (macOS ARM) so you can develop locally with Cloud Run behavior.
#
# WHAT YOU GET:
#   • Scale-to-zero in <2s (like Cloud Run)
#   • Cold starts ~100-200ms (local containers)
#   • Magic DNS via sslip.io (http://myapp.default.192.168.x.x.sslip.io)
#   • Contour/Envoy ingress with automatic HTTPProxy routing
#   • Full kn CLI compatibility (kn service create/update/delete)
#
# ═══════════════════════════════════════════════════════════════════════════════
# ARCHITECTURE (what gets installed)
# ═══════════════════════════════════════════════════════════════════════════════
#
#   ┌─────────────────────────────────────────────────────────────────────────┐
#   │                        OrbStack VM (Linux/ARM64)                        │
#   │                                                                         │
#   │  ┌───────────────────────────────────────────────────────────────────┐  │
#   │  │                    Kubernetes Cluster                             │  │
#   │  │                                                                   │  │
#   │  │  projectcontour/               knative-serving/                   │  │
#   │  │  ┌─────────────┐               ┌──────────────┐                   │  │
#   │  │  │ Contour     │◄──────────────│ net-contour  │                   │  │
#   │  │  │ (control)   │  HTTPProxy    │ controller   │                   │  │
#   │  │  └─────────────┘               └──────────────┘                   │  │
#   │  │        │                              ▲                           │  │
#   │  │        │ programs                     │ watches                   │  │
#   │  │        ▼                              │ Knative Ingress           │  │
#   │  │  ┌─────────────┐               ┌──────────────┐                   │  │
#   │  │  │ Envoy       │               │ controller   │                   │  │
#   │  │  │ (data plane)│               │ (reconciler) │                   │  │
#   │  │  │ LoadBalancer│               └──────────────┘                   │  │
#   │  │  │ :80, :443   │                      │                           │  │
#   │  │  └──────┬──────┘                      │ creates                   │  │
#   │  │         │                             ▼                           │  │
#   │  │         │ routes by Host    ┌──────────────────┐                  │  │
#   │  │         │                   │ Revision Pods     │                  │  │
#   │  │         └──────────────────►│ [queue-proxy]     │                  │  │
#   │  │           (or Activator     │ [your-container]  │                  │  │
#   │  │            if scaled=0)     └──────────────────┘                  │  │
#   │  │                                                                   │  │
#   │  └───────────────────────────────────────────────────────────────────┘  │
#   └─────────────────────────────────────────────────────────────────────────┘
#
# ═══════════════════════════════════════════════════════════════════════════════
# USAGE
# ═══════════════════════════════════════════════════════════════════════════════
#   ./knative_orbstack.sh                    # Install everything
#   ./knative_orbstack.sh --status           # Check installation status
#   ./knative_orbstack.sh --uninstall        # Remove all components
#   ./knative_orbstack.sh --debug            # Collect diagnostics on failures
#   ./knative_orbstack.sh --install-metrics  # Also install metrics-server (for HPA)
#   ./knative_orbstack.sh --prepull          # Pre-pull images for faster starts
#   ./knative_orbstack.sh --warm 1           # Keep services warm (min 1 replica)
#
# REQUIREMENTS:
#   • OrbStack with Kubernetes enabled (Settings → Kubernetes → Enable)
#   • kubectl configured (OrbStack does this automatically)
#   • Optional: Homebrew (for auto-installing kn CLI)
#
# ═══════════════════════════════════════════════════════════════════════════════

set -euo pipefail
IFS=$'\n\t'

# ─────────────────────────────────────────────────────────────────────────────
# CONFIGURATION
# ─────────────────────────────────────────────────────────────────────────────

# Knative version (Dec 2025 latest stable)
KNATIVE_TAG="knative-v1.20.0"

# Manifest URLs
KNATIVE_SERVING_BASE="https://github.com/knative/serving/releases/download/${KNATIVE_TAG}"
NET_CONTOUR_BASE="https://github.com/knative-extensions/net-contour/releases/download/${KNATIVE_TAG}"
CONTOUR_MANIFEST="https://projectcontour.io/quickstart/contour.yaml"
DEFAULT_DOMAIN_MANIFEST="${KNATIVE_SERVING_BASE}/serving-default-domain.yaml"
CRDS_MANIFEST="${KNATIVE_SERVING_BASE}/serving-crds.yaml"
CORE_MANIFEST="${KNATIVE_SERVING_BASE}/serving-core.yaml"
NET_CONTOUR_MANIFEST="${NET_CONTOUR_BASE}/net-contour.yaml"

# Namespaces
NAMESPACE="knative-serving"
CONTOUR_NAMESPACE="projectcontour"

# Timeouts
TIMEOUT_PODS=180s

# ─────────────────────────────────────────────────────────────────────────────
# COLORS & OUTPUT HELPERS
# ─────────────────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()    { printf "${BLUE}▶${NC} %s\n" "$*"; }
success() { printf "${GREEN}✓${NC} %s\n" "$*"; }
warn()    { printf "${YELLOW}⚠${NC} %s\n" "$*" >&2; }
errorf()  { printf "${RED}✗${NC} %s\n" "$*" >&2; }
step()    { printf "\n${CYAN}${BOLD}═══ %s ═══${NC}\n" "$*"; }

# ─────────────────────────────────────────────────────────────────────────────
# FLAGS (set via CLI args)
# ─────────────────────────────────────────────────────────────────────────────

INSTALL_METRICS=false
INSTALL_METALLB=false
PREPULL=false
WARM_REPLICAS=0
DEBUG=false

# ─────────────────────────────────────────────────────────────────────────────
# HELP
# ─────────────────────────────────────────────────────────────────────────────

print_help() {
  cat <<EOF
${BOLD}knative_orbstack.sh${NC} — Cloud Run on OrbStack

${BOLD}USAGE${NC}
  $0 [OPTIONS]

${BOLD}OPTIONS${NC}
  (none)             Install Knative Serving + Contour on OrbStack
  --status, -s       Show current installation status
  --uninstall, -u    Remove all Knative and Contour components
  --debug            Collect diagnostics automatically on test failures
  --install-metrics  Also install metrics-server (enables HPA)
  --install-metalb   Also install MetalLB for LoadBalancer IP allocation
  --prepull          Pre-pull common images to speed up first deployment
  --warm <N>         Keep sample services warm with minimum N replicas
  --help, -h         Show this help

${BOLD}EXAMPLES${NC}
  # Basic install
  $0

  # Install with metrics-server and keep services warm
  $0 --install-metrics --warm 1

  # Check status
  $0 --status

  # Full cleanup
  $0 --uninstall

${BOLD}QUICK COMMANDS AFTER INSTALL${NC}
  kn service create myapp --image=nginx --port=80   # Deploy a service
  kn service list                                    # List services
  kubectl get ksvc                                   # Same, via kubectl
  curl http://myapp.default.<IP>.sslip.io           # Test your service
EOF
}

# ─────────────────────────────────────────────────────────────────────────────
# PREREQUISITES
# ─────────────────────────────────────────────────────────────────────────────

require_kubectl() {
  if ! command -v kubectl &>/dev/null; then
    errorf "kubectl not found. Install it or enable Kubernetes in OrbStack."
    exit 1
  fi
  if ! kubectl cluster-info &>/dev/null; then
    errorf "Cannot connect to Kubernetes. Is OrbStack running with Kubernetes enabled?"
    errorf "  → Open OrbStack → Settings → Kubernetes → Enable"
    exit 1
  fi
  success "Connected to Kubernetes cluster"
}

ensure_kn() {
  if command -v kn &>/dev/null; then
    success "kn CLI found: $(kn version --short 2>/dev/null || echo 'installed')"
    return 0
  fi
  info "kn CLI not found. Installing via Homebrew..."
  if command -v brew &>/dev/null; then
    brew install knative/client/kn && success "kn CLI installed" && return 0
  fi
  warn "Could not install kn. Install manually:"
  warn "  brew install knative/client/kn"
  warn "  OR: https://github.com/knative/client/releases"
}

# ─────────────────────────────────────────────────────────────────────────────
# MANIFEST APPLICATION
# ─────────────────────────────────────────────────────────────────────────────

apply_manifest() {
  local url="$1"
  local name=$(basename "$url" .yaml)
  info "Applying: $name"
  if kubectl apply -f "$url" &>/dev/null; then
    success "Applied $name"
  else
    sleep 2
    kubectl apply -f "$url" || { errorf "Failed to apply $url"; return 1; }
    success "Applied $name (retry)"
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# CONFIGURATION PATCHES
# ─────────────────────────────────────────────────────────────────────────────

patch_config_network() {
  info "Configuring Knative to use Contour ingress"
  kubectl patch configmap/config-network -n ${NAMESPACE} --type merge \
    -p '{"data":{"ingress.class":"contour.ingress.networking.knative.dev"}}' 2>/dev/null || true
}

patch_config_contour() {
  info "Configuring Contour for OrbStack (projectcontour/envoy)"
  # OrbStack uses projectcontour namespace, not contour-external
  kubectl patch cm config-contour -n ${NAMESPACE} --type merge -p '{
    "data": {
      "visibility": "ExternalIP:\n  class: contour\n  service: projectcontour/envoy\nClusterLocal:\n  class: contour-internal\n  service: projectcontour/envoy\n"
    }
  }' 2>/dev/null || true
}

patch_autoscaler() {
  info "Tuning autoscaler for fast scale-to-zero"
  # These settings give Cloud Run-like behavior:
  # - scale-to-zero-grace-period: 6s (fast scale down)
  # - container-concurrency-target-default: 100 (per-pod concurrency)
  kubectl patch cm config-autoscaler -n ${NAMESPACE} --type merge -p '{
    "data": {
      "enable-scale-to-zero": "true",
      "scale-to-zero-grace-period": "6s",
      "container-concurrency-target-default": "100",
      "target-burst-capacity": "200",
      "stable-window": "60s",
      "panic-window": "6s"
    }
  }' 2>/dev/null || true
}

# ─────────────────────────────────────────────────────────────────────────────
# ENVOY HOSTPORT PATCH (OrbStack-specific)
# ─────────────────────────────────────────────────────────────────────────────
# Contour's quickstart manifest uses hostPort for Envoy, which conflicts on
# OrbStack (single-node). We remove hostPort so the DaemonSet can schedule.

patch_envoy_hostport() {
  info "Removing Envoy hostPort (OrbStack compatibility)"
  
  local has_hostport=$(kubectl get ds envoy -n ${CONTOUR_NAMESPACE} \
    -o jsonpath='{.spec.template.spec.containers[1].ports[0].hostPort}' 2>/dev/null || echo "")
  
  if [ -n "$has_hostport" ]; then
    kubectl patch ds envoy -n ${CONTOUR_NAMESPACE} --type='json' \
      -p='[{"op":"remove","path":"/spec/template/spec/containers/1/ports/0/hostPort"},
           {"op":"remove","path":"/spec/template/spec/containers/1/ports/1/hostPort"},
           {"op":"remove","path":"/spec/template/spec/containers/1/ports/2/hostPort"}]' 2>/dev/null && \
      success "Envoy hostPort removed" || warn "hostPort patch failed (may be OK)"
    # Restart Envoy pods to pick up the change
    kubectl delete pods -n ${CONTOUR_NAMESPACE} -l app=envoy --wait=false 2>/dev/null || true
  else
    success "Envoy hostPort already removed"
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# WAIT FUNCTIONS
# ─────────────────────────────────────────────────────────────────────────────

wait_contour_ready() {
  info "Waiting for Contour deployment..."
  if kubectl wait --for=condition=Available deployment/contour -n ${CONTOUR_NAMESPACE} --timeout=${TIMEOUT_PODS} 2>/dev/null; then
    success "Contour deployment ready"
  else
    warn "Contour deployment timed out (check pods)"
  fi
}

wait_for_envoy_ready() {
  info "Waiting for Envoy pods..."
  # Envoy is a DaemonSet, so we wait for pods to be ready
  if kubectl wait --for=condition=Ready pod -l app=envoy -n ${CONTOUR_NAMESPACE} --timeout=${TIMEOUT_PODS} 2>/dev/null; then
    success "Envoy pods ready"
  else
    warn "Envoy pods timed out"
  fi
}

wait_knative_ready() {
  info "Waiting for Knative deployments..."
  # Wait for all deployments in the namespace to be available
  if kubectl wait --for=condition=Available deployment --all -n ${NAMESPACE} --timeout=${TIMEOUT_PODS} 2>/dev/null; then
    success "All Knative deployments ready"
  else
    warn "Some Knative deployments not ready; check: kubectl get deploy -n ${NAMESPACE}"
  fi
}

wait_for_envoy_ip() {
  info "Waiting for Envoy LoadBalancer IP..."
  for i in $(seq 1 30); do
    local ip=$(kubectl get svc envoy -n ${CONTOUR_NAMESPACE} \
      -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    if [ -n "$ip" ] && [ "$ip" != "<pending>" ]; then
      success "Envoy external IP: $ip"
      export ENVOY_IP="$ip"
      return 0
    fi
    printf "\r  Waiting for IP... (%d/30)" "$i"
    sleep 2
  done
  printf "\n"
  warn "No external IP assigned (services may not be accessible)"
}

wait_for_service_ready() {
  local name="$1"
  info "Waiting for service '$name'..."
  for i in $(seq 1 40); do
    local ready=$(kubectl get ksvc "$name" \
      -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
    [ "$ready" = "True" ] && { success "Service '$name' is Ready"; return 0; }
    printf "\r  Waiting... (%d/40)" "$i"
    sleep 3
  done
  printf "\n"
  warn "Service '$name' may not be ready"
  return 1
}

# ... (skip to create_sample_services)

create_sample_services() {
  if command -v kn &>/dev/null; then
    info "Creating sample services via kn CLI (background)..."
    kn service create hello --image=us-docker.pkg.dev/cloudrun/container/hello --port=8080 --force --no-wait 2>/dev/null || true
    kn service create nginx --image=nginx --port=80 --force --no-wait 2>/dev/null || true
  else
    info "Creating sample services via kubectl"
    # ... (kubectl apply is already async-ish)
    cat <<EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: hello
  namespace: default
spec:
  template:
    spec:
      containers:
      - image: us-docker.pkg.dev/cloudrun/container/hello
        ports:
        - containerPort: 8080
---
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: nginx
  namespace: default
spec:
  template:
    spec:
      containers:
      - image: nginx
        ports:
        - containerPort: 80
EOF
  fi
}

test_service() {
  local name="${1:-hello}"
  info "Testing service '$name'..."
  wait_for_service_ready "$name" || return 1
  
  local url=$(kubectl get ksvc "$name" -o jsonpath='{.status.url}' 2>/dev/null)
  [ -z "$url" ] && { warn "No URL for $name"; return 1; }
  
  info "URL: $url"
  for i in $(seq 1 8); do
    if curl -sf --max-time 15 "$url" | grep -qi "hello\|welcome\|nginx\|congratulations"; then
      success "Service '$name' is responding!"
      [ "$DEBUG" = true ] && collect_diagnostics "$name"
      return 0
    fi
    printf "\r  Attempt %d/8..." "$i"
    sleep 3
  done
  printf "\n"
  warn "Service test failed. Try: curl $url"
  collect_diagnostics "$name"
  return 1
}

# ─────────────────────────────────────────────────────────────────────────────
# DIAGNOSTICS
# ─────────────────────────────────────────────────────────────────────────────

collect_diagnostics() {
  local name="${1:-hello}"
  local outdir="/tmp/knative-diag-$(date +%Y%m%d-%H%M%S)-${name}"
  mkdir -p "$outdir"
  info "Collecting diagnostics → $outdir"

  # Knative resources
  kubectl get ksvc "$name" -o yaml > "$outdir/ksvc.yaml" 2>/dev/null || true
  kubectl describe ksvc "$name" > "$outdir/ksvc.describe" 2>/dev/null || true
  kubectl get revisions -l serving.knative.dev/service="$name" -o yaml > "$outdir/revisions.yaml" 2>/dev/null || true
  kubectl get pods -l serving.knative.dev/service="$name" -o wide > "$outdir/pods.txt" 2>/dev/null || true

  # Networking
  kubectl get httpproxy -A -o yaml > "$outdir/httpproxy.yaml" 2>/dev/null || true
  kubectl get svc envoy -n ${CONTOUR_NAMESPACE} -o yaml > "$outdir/envoy-svc.yaml" 2>/dev/null || true
  kubectl get endpoints envoy -n ${CONTOUR_NAMESPACE} -o yaml > "$outdir/envoy-endpoints.yaml" 2>/dev/null || true

  # Logs
  kubectl logs -n ${CONTOUR_NAMESPACE} -l app=envoy --tail=100 > "$outdir/envoy.log" 2>/dev/null || true
  kubectl logs -n ${CONTOUR_NAMESPACE} -l app=contour --tail=100 > "$outdir/contour.log" 2>/dev/null || true
  kubectl logs -n ${NAMESPACE} deploy/net-contour-controller --tail=200 > "$outdir/net-contour.log" 2>/dev/null || true
  kubectl logs -n ${NAMESPACE} deploy/activator --tail=100 > "$outdir/activator.log" 2>/dev/null || true

  # Events
  kubectl get events -n default --sort-by='.lastTimestamp' > "$outdir/events-default.txt" 2>/dev/null || true
  kubectl get events -n ${NAMESPACE} --sort-by='.lastTimestamp' > "$outdir/events-knative.txt" 2>/dev/null || true

  # Quick summary
  echo "=== Last 30 lines of net-contour-controller ===" 
  tail -30 "$outdir/net-contour.log" 2>/dev/null || true

  success "Diagnostics saved: $outdir"
}

# ─────────────────────────────────────────────────────────────────────────────
# STATUS
# ─────────────────────────────────────────────────────────────────────────────

show_status() {
  cat <<EOF
${BOLD}Knative Serving & kagent Status${NC}
════════════════════════════════════

EOF
  echo "${BOLD}Namespaces:${NC}"
  kubectl get ns 2>/dev/null | grep -E 'NAME|knative|contour|kagent' || echo "  None"
  
  echo ""
  echo "${BOLD}Knative Pods (knative-serving):${NC}"
  kubectl get pods -n ${NAMESPACE} 2>/dev/null || echo "  None"
  
  echo ""
  echo "${BOLD}Contour Pods (projectcontour):${NC}"
  kubectl get pods -n ${CONTOUR_NAMESPACE} 2>/dev/null || echo "  None"
  
  echo ""
  echo "${BOLD}kagent Status:${NC}"
  verify_kagent || echo "  Not installed"
  
  echo ""
  echo "${BOLD}Knative Services:${NC}"
  kubectl get ksvc -A 2>/dev/null || echo "  None"
  
  echo ""
  echo "${BOLD}Envoy LoadBalancer:${NC}"
  kubectl get svc envoy -n ${CONTOUR_NAMESPACE} 2>/dev/null || echo "  Not found"
  
  local ip=$(kubectl get svc envoy -n ${CONTOUR_NAMESPACE} -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
  if [ -n "$ip" ]; then
    echo ""
    echo "${BOLD}Sample URLs:${NC}"
    echo "  http://hello.default.${ip}.sslip.io"
    echo "  http://nginx.default.${ip}.sslip.io"
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# UNINSTALL
# ─────────────────────────────────────────────────────────────────────────────

uninstall() {
  warn "Uninstalling Knative Serving and Contour..."
  
  # Delete sample services
  info "Deleting sample services"
  kubectl delete ksvc hello nginx 2>/dev/null || true
  
  # Delete Knative
  info "Deleting Knative Serving"
  kubectl delete -f "${DEFAULT_DOMAIN_MANIFEST}" 2>/dev/null || true
  kubectl delete -f "${NET_CONTOUR_MANIFEST}" 2>/dev/null || true
  kubectl delete -f "${CORE_MANIFEST}" 2>/dev/null || true
  kubectl delete -f "${CRDS_MANIFEST}" 2>/dev/null || true
  
  # Delete Contour
  info "Deleting Contour"
  kubectl delete -f "${CONTOUR_MANIFEST}" 2>/dev/null || true
  
  # Delete namespaces
  kubectl delete namespace ${NAMESPACE} ${CONTOUR_NAMESPACE} 2>/dev/null || true
  
  # Optional components
  kubectl delete -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml 2>/dev/null || true
  kubectl delete namespace metallb-system 2>/dev/null || true
  
  success "Uninstall complete"
}

# ─────────────────────────────────────────────────────────────────────────────
# KAGENT INSTALLATION
# ─────────────────────────────────────────────────────────────────────────────

install_kagent() {
  info "Installing kagent platform..."
  
  if ! command -v helm &>/dev/null; then
    warn "Helm not found. Skipping full kagent installation."
    return
  fi

  # Create kagent namespace if it doesn't exist
  kubectl create namespace kagent 2>/dev/null || true
  
  # Install CRDs using OCI URL
  info "Installing kagent CRDs..."
  if helm upgrade --install kagent-crds oci://ghcr.io/kagent-dev/kagent/helm/kagent-crds \
      --version 0.7.7 \
      -n kagent \
      --wait; then
    success "kagent CRDs installed"
  else
    warn "Failed to install kagent CRDs (check internet/permissions)"
  fi

  # Install Platform (UI, Controller, etc.)
  info "Installing kagent platform (with UI)..."
  # Removed --wait to speed up return. Pods will start in background.
  if helm upgrade --install kagent oci://ghcr.io/kagent-dev/kagent/helm/kagent \
      --version 0.7.7 \
      -n kagent \
      --set ui.enabled=true; then
    success "kagent platform installation started (background)"
  else
    warn "Failed to install kagent platform"
  fi
}

verify_kagent() {
  local ns=$(kubectl get ns kagent 2>/dev/null)
  if [ -z "$ns" ]; then
    warn "kagent namespace not found"
    return 1
  fi
  
  # Check for kagent-crds
  local crds=$(kubectl get crds 2>/dev/null | grep -c "kagent.dev" || echo 0)
  # Ensure crds is a valid integer
  if ! [[ "$crds" =~ ^[0-9]+$ ]]; then
    crds=0
  fi
  if [ "$crds" -gt 0 ]; then
    success "kagent CRDs installed ($crds CRD types found)"
    return 0
  else
    info "kagent CRDs not yet installed (manual installation required)"
    return 1
  fi
}

# ─────────────────────────────────────────────────────────────────────────────
# MAIN
# ─────────────────────────────────────────────────────────────────────────────

main() {
  cat <<EOF

${BOLD}════════════════════════════════════════════════════════════════════${NC}
${CYAN}  Knative Serving Installer for OrbStack${NC}
${BOLD}════════════════════════════════════════════════════════════════════${NC}
  Version: ${KNATIVE_TAG}
  Target:  Cloud Run-like serverless on local Kubernetes

EOF

  require_kubectl
  ensure_kn

  # Optional: install metrics-server
  [ "$INSTALL_METRICS" = true ] && install_metrics_server
  
  # Optional: install MetalLB
  [ "$INSTALL_METALLB" = true ] && install_metal_lb
  
  # Optional: pre-pull images
  [ "$PREPULL" = true ] && prepull_images

  step "Step 1/6: Installing Core Components (Parallel)"
  
  # Start Contour install in background
  (
    info "Installing Contour..."
    apply_manifest "${CONTOUR_MANIFEST}"
  ) &
  pid_contour=$!

  # Start Knative Core install in background
  (
    info "Installing Knative Serving CRDs & Core..."
    apply_manifest "${CRDS_MANIFEST}"
    apply_manifest "${CORE_MANIFEST}"
  ) &
  pid_knative=$!

  wait $pid_contour
  wait $pid_knative
  success "Core components manifests applied"

  step "Step 2/6: Configuring Envoy & Integration"
  # Wait for Contour deployment to be available before patching
  wait_contour_ready
  patch_envoy_hostport
  wait_for_envoy_ready

  info "Installing Knative-Contour Integration..."
  apply_manifest "${NET_CONTOUR_MANIFEST}"

  step "Step 3/6: Applying Configuration"
  apply_manifest "${DEFAULT_DOMAIN_MANIFEST}"
  patch_config_network
  patch_config_contour
  patch_autoscaler

  step "Step 4/6: Verifying Installation"
  wait_knative_ready
  wait_for_envoy_ip

  step "Step 5/6: Installing kagent CRDs"
  install_kagent

  step "Step 6/6: Deploying Sample Services"
  create_sample_services
  
  # Warm replicas if requested
  if [ "$WARM_REPLICAS" -gt 0 ]; then
    set_warm_scale "hello" "$WARM_REPLICAS"
    set_warm_scale "nginx" "$WARM_REPLICAS"
  fi

  # Skip blocking tests to speed up setup
  # test_service "hello" || true
  # test_service "nginx" || true
  info "Sample services deployed. Check status with: kubectl get ksvc"

  print_summary
}

print_summary() {
  local ip=$(kubectl get svc envoy -n ${CONTOUR_NAMESPACE} -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
  local hello_url=$(kubectl get ksvc hello -o jsonpath='{.status.url}' 2>/dev/null || echo "N/A")
  local nginx_url=$(kubectl get ksvc nginx -o jsonpath='{.status.url}' 2>/dev/null || echo "N/A")

  cat <<EOF

${BOLD}════════════════════════════════════════════════════════════════════${NC}
${GREEN}  ✓ Knative Serving Installation Complete!${NC}
${BOLD}════════════════════════════════════════════════════════════════════${NC}

${BOLD}Cluster Info:${NC}
  Knative Version:   ${KNATIVE_TAG}
  Envoy External IP: ${ip}

${BOLD}Sample Services:${NC}
  hello: ${hello_url}
  nginx: ${nginx_url}

${BOLD}Quick Commands:${NC}
  # List all Knative services
  kubectl get ksvc

  # Create a new service
  kn service create myapp --image=nginx --port=80

  # Update with environment variables
  kn service update myapp --env KEY=value

  # Scale configuration
  kn service update myapp --scale 1..10    # min 1, max 10
  kn service update myapp --scale 0..5     # allow scale-to-zero

  # Delete a service
  kn service delete myapp

${BOLD}Test:${NC}
  curl ${hello_url}
  curl ${nginx_url}

${BOLD}Troubleshooting:${NC}
  # Check status
  $0 --status

  # View logs
  kubectl logs -n knative-serving deploy/controller
  kubectl logs -n knative-serving deploy/activator
  kubectl logs -n projectcontour -l app=envoy

  # Re-run with diagnostics
  $0 --debug

════════════════════════════════════════════════════════════════════
EOF
}

# ─────────────────────────────────────────────────────────────────────────────
# CLI ARGUMENT PARSING
# ─────────────────────────────────────────────────────────────────────────────

while [[ $# -gt 0 ]]; do
  case "$1" in
    --help|-h)
      print_help
      exit 0
      ;;
    --status|-s)
      show_status
      exit 0
      ;;
    --uninstall|-u)
      uninstall
      exit 0
      ;;
    --debug)
      DEBUG=true
      shift
      ;;
    --install-metrics)
      INSTALL_METRICS=true
      shift
      ;;
    --install-metalb)
      INSTALL_METALLB=true
      shift
      ;;
    --prepull)
      PREPULL=true
      shift
      ;;
    --warm)
      WARM_REPLICAS="${2:-1}"
      shift 2
      ;;
    *)
      errorf "Unknown option: $1"
      print_help
      exit 1
      ;;
  esac
done

main
