# Scratchpad - k8s-agent-stack Operational Readiness

## Date: 2025-12-16

## Observations

### Initial State
- [x] System structure analyzed
- [x] Current documentation reviewed
- [x] Stack components identified
- [x] Health status checked

### Notes

**Cluster Status (OrbStack Kubernetes)**
- Kubernetes running at 127.0.0.1:26443
- Knative Serving v1.20.0 - RUNNING (5 pods healthy)
- Contour/Envoy - RUNNING (Envoy at 192.168.139.2)
- kagent namespace - EXISTS with 3 pods running

**Key Components Deployed:**
1. `google-adk-agent` - Knative Service (using placeholder hello-world image!)
2. `kagent-portal` - Knative Service (running)
3. `kagent-web` - Deployment (running on port 3000)

**Critical Issue Found:**
- `kagent-setup.yaml` uses `us-docker.pkg.dev/cloudrun/container/hello:latest` (placeholder)
- The actual ADK agent image `kagent-adk-agent:v29` is NOT being used
- kagent CRDs are NOT installed (no `kagent.dev` API resources)
- The stack is pure Knative, not full kagent integration

**ADK Agent Code:**
- Located in `kagent-adk-agent/` directory
- Uses Google ADK with LiteLLM + OpenAI
- Provides tools: get_weather, get_current_time, calculate_math
- Endpoints: `/health`, `/` (A2A JSON-RPC), SSE streaming

---

## Hypotheses

1. The stack setup script doesn't build/push the ADK agent image
2. Users need to build the image locally before deployment
3. kagent-setup.yaml is meant as a quick-start template, not production

---

## Decision Rationale

**Path forward:**
1. Build the actual ADK agent image locally
2. Deploy it as a Knative service with proper image
3. Test the A2A endpoint functionality
4. Document the correct deployment workflow

---

## Documentation Audit Findings

### Root Level (Excessive/Redundant Files)
- `AGENT_ACCESS.md` - Redundant, specific to one-time setup
- `AGENT_DEPLOYMENT_COMPLETE.md` - One-time status, should archive
- `DEPLOYMENT_COMPLETE.md` - One-time status, should archive
- `KAGENT_PORTAL_ACCESS.md` - Merge into main docs
- `PORTAL_ACCESS.md` - Merge into main docs  
- `PORTAL_DEPLOYMENT_COMPLETE.md` - Archive
- `PORTAL_DOCUMENTATION_INDEX.md` - Archive
- `PORTAL_QUICK_ACCESS.md` - Merge into Quick Reference
- `PORTAL_STATUS.md` - Archive
- `QUICK_START.md` - Merge into Getting Started
- `README-k8s.md` - Redundant, archive
- `SETUP.md` - Merge into Getting Started
- `SETUP_COMPLETION.md` - Archive
- `MAKEFILE_GUIDE.md` - Merge into Quick Reference
- `kagent.md` - Good reference but overlaps with docs/
- `knative-orbstack.md` - Merge into Getting Started
- `knative.md` - Merge into Getting Started
- `k8s-ovh.md` - Cloud-specific, archive for now
- `k8s.md` - Merge into Getting Started

### docs/ Directory (Good but needs updates)
- `getting-started.md` - Good structure, needs updates for current state
- `architecture.md` - Excellent, keep and update status
- `deployment-guide.md` - Good, needs current image info
- `troubleshooting.md` - Good, keep
- `glossary.md` - Excellent, keep
- `tool-installation.md` - Good reference
- `building-google-adk-agents-for-kagent.md` - Good, keep
- `kagent-adk-a2a-architecture.md` - Advanced, keep

### Issues Found
1. **Image reference wrong** - Docs say `gcr.io/...` but should be `dev.local/kagent-adk-agent:v30`
2. **kagent CRDs mentioned but not installed** - Need to clarify this is pure Knative
3. **Scattered portal docs** - Need consolidation
4. **No clear known limitations section**
5. **Missing health check verification steps**
