# 008 - Deployment & Operations

> CI/CD, GitOps, Scaling, Multi-Region, and Day-2 Operations

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Deployment Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                    Deployment Flow                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Developer                                                      │
│      │                                                          │
│      ▼                                                          │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐        │
│  │   Git Push  │────▶│  CI/CD      │────▶│  Registry   │        │
│  │             │     │  (Build)    │     │  (OCI)      │        │
│  └─────────────┘     └─────────────┘     └──────┬──────┘        │
│                                                  │              │
│                                                  ▼              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                     GitOps                              │    │
│  │  ┌─────────┐      ┌─────────┐      ┌─────────┐          │    │
│  │  │ ArgoCD  │─────▶│ Staging │─────▶│  Prod   │          │    │
│  │  │         │      │ Cluster │      │ Cluster │          │    │
│  │  └─────────┘      └─────────┘      └─────────┘          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Deployment Methods

| Method | Use Case | Trigger |
|--------|----------|---------|
| **GitOps** | Production | Git commit |
| **CLI** | Development | Manual |
| **API** | CI/CD integration | Automated |
| **Declarative YAML** | Kubernetes-native | kubectl apply |

---

## 2. CI/CD Pipeline

### 2.1 Pipeline Stages

```text
┌─────────────────────────────────────────────────────────────────┐
│                      Pipeline Stages                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐      │
│  │  Build   │──▶│  Test    │──▶│  Scan    │──▶│ Publish  │      │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘      │
│       │              │              │              │            │
│       ▼              ▼              ▼              ▼            │
│  • Container     • Unit         • Trivy        • OCI            │
│  • Dependencies  • Integration  • Snyk         • Helm Chart     │
│                  • E2E          • SBOM         • Manifest       │
│                                                                 │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐      │
│  │  Deploy  │──▶│ Validate │──▶│ Promote  │──▶│ Monitor  │      │
│  │ Staging  │   │          │   │   Prod   │   │          │      │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘      │
│       │              │              │              │            │
│       ▼              ▼              ▼              ▼            │
│  • ArgoCD       • Smoke         • Manual/      • Metrics        │
│  • Canary       • E2E           • Auto gate    • Alerts         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 GitHub Actions Example

```yaml
name: Agent CI/CD

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Build Container
        run: docker build -t ${{ env.REGISTRY }}/agent:${{ github.sha }} .
      
      - name: Run Tests
        run: |
          docker run --rm ${{ env.REGISTRY }}/agent:${{ github.sha }} \
            python -m pytest tests/
      
      - name: Security Scan
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: ${{ env.REGISTRY }}/agent:${{ github.sha }}
          exit-code: 1
          severity: 'CRITICAL,HIGH'
      
      - name: Push Image
        run: docker push ${{ env.REGISTRY }}/agent:${{ github.sha }}
  
  deploy-staging:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Update Manifest
        run: |
          yq -i '.spec.template.image = "${{ env.REGISTRY }}/agent:${{ github.sha }}"' \
            environments/staging/agent.yaml
          git commit -am "Deploy ${{ github.sha }} to staging"
          git push
```

---

## 3. GitOps with ArgoCD

### 3.1 Repository Structure

```text
infrastructure/
├── base/                    # Base manifests
│   ├── agents/
│   │   ├── kustomization.yaml
│   │   └── agent-template.yaml
│   └── platform/
│       ├── knative/
│       └── kagent/
├── environments/
│   ├── staging/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   └── production/
│       ├── kustomization.yaml
│       └── patches/
└── projects/
    └── project-abc/
        └── agents/
            └── customer-support.yaml
```

### 3.2 ArgoCD Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: agentstack-prod
  namespace: argocd
spec:
  project: default
  source:
    repoURL: git@github.com:org/agentstack-infra.git
    targetRevision: main
    path: environments/production
  destination:
    server: https://kubernetes.default.svc
    namespace: agentstack
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### 3.3 Promotion Flow

```text
┌───────────────────────────────────────────────────────────────┐
│                    Environment Promotion                      │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│  Development ──▶ Staging ──▶ Production                       │
│       │              │             │                          │
│       ▼              ▼             ▼                          │
│   Auto-sync      Auto-sync    Manual Gate                     │
│   (commit)       (PR merge)   (approval)                      │
│                                                               │
│   Tests:         Tests:       Tests:                          │
│   • Unit         • E2E        • Smoke                         │
│   • Lint         • Perf       • Canary                        │
│                  • Security   • Rollback ready                │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

---

## 4. Scaling Strategy

### 4.1 Horizontal Pod Autoscaling

```yaml
# Knative autoscaling configuration
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: customer-support
spec:
  template:
    metadata:
      annotations:
        # Scale based on concurrent requests
        autoscaling.knative.dev/class: kpa.autoscaling.knative.dev
        autoscaling.knative.dev/metric: concurrency
        autoscaling.knative.dev/target: "10"
        
        # Scale bounds
        autoscaling.knative.dev/min-scale: "1"
        autoscaling.knative.dev/max-scale: "100"
        
        # Scale-to-zero configuration
        autoscaling.knative.dev/scale-down-delay: "1m"
```

### 4.2 Scaling Dimensions

| Dimension | Trigger | Range | Tool |
|-----------|---------|-------|------|
| **Pods** | Requests/concurrency | 0-100 | KPA |
| **Nodes** | Resource utilization | 3-50 | Cluster Autoscaler |
| **Regions** | Latency/availability | 1-N | Multi-cluster |

### 4.3 Warm Pool

```yaml
# Keep warm instances for critical agents
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: critical-agent
spec:
  template:
    metadata:
      annotations:
        # Never scale to zero
        autoscaling.knative.dev/min-scale: "2"
        
        # Pre-warm additional instances
        autoscaling.knative.dev/initial-scale: "3"
```

---

## 5. Multi-Region Deployment

### 5.1 Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Multi-Region Architecture                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Global DNS                           │    │
│  │           (Route 53 / Cloud DNS / Cloudflare)           │    │
│  └──────────────────────┬──────────────────────────────────┘    │
│                         │                                       │
│         ┌───────────────┼───────────────┐                       │
│         │               │               │                       │
│         ▼               ▼               ▼                       │
│  ┌───────────┐   ┌───────────┐   ┌───────────┐                  │
│  │ EU-West   │   │ US-East   │   │ AP-South  │                  │
│  │           │   │           │   │           │                  │
│  │ ┌───────┐ │   │ ┌───────┐ │   │ ┌───────┐ │                  │
│  │ │Cluster│ │   │ │Cluster│ │   │ │Cluster│ │                  │
│  │ │       │ │   │ │       │ │   │ │       │ │                  │
│  │ │Agents │ │   │ │Agents │ │   │ │Agents │ │                  │
│  │ └───────┘ │   │ └───────┘ │   │ └───────┘ │                  │
│  │           │   │           │   │           │                  │
│  │ ┌───────┐ │   │ ┌───────┐ │   │ ┌───────┐ │                  │
│  │ │  DB   │◀┼───┼▶│  DB   │◀┼───┼▶│  DB   │ │                  │
│  │ │Replica│ │   │ │Primary│ │   │ │Replica│ │                  │
│  │ └───────┘ │   │ └───────┘ │   │ └───────┘ │                  │
│  └───────────┘   └───────────┘   └───────────┘                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Data Replication

| Data Type | Strategy | Latency |
|-----------|----------|---------|
| **Agent Config** | Async replication | < 1s |
| **Sessions** | Regional (Redis Cluster) | - |
| **Conversations** | Cross-region async | < 5s |
| **Secrets** | Per-region (external-secrets) | - |

### 5.3 Failover

```yaml
# Global load balancing
apiVersion: networking.gke.io/v1
kind: MultiClusterIngress
metadata:
  name: agentstack-global
spec:
  template:
    spec:
      backend:
        serviceName: agentstack-mcs
        servicePort: 443
---
# Health check
apiVersion: networking.gke.io/v1
kind: MultiClusterService
metadata:
  name: agentstack-mcs
spec:
  template:
    spec:
      ports:
        - port: 443
          protocol: TCP
  clusters:
    - link: "europe-west1/agentstack-eu"
    - link: "us-east1/agentstack-us"
```

---

## 6. Day-2 Operations

### 6.1 Runbook: Rollback

```bash
# 1. Identify current and previous revision
kubectl get revisions -l serving.knative.dev/service=my-agent

# 2. Route traffic to previous revision
kubectl patch service my-agent \
  --type merge \
  -p '{"spec":{"traffic":[{"revisionName":"my-agent-00001","percent":100}]}}'

# 3. Verify rollback
kubectl get ksvc my-agent -o jsonpath='{.status.traffic}'
```

### 6.2 Runbook: Scaling Emergency

```bash
# Disable scale-to-zero and set minimum
kubectl patch ksvc my-agent --type merge -p '
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/min-scale: "10"
        autoscaling.knative.dev/max-scale: "200"
'

# Force immediate scale-up
kubectl scale deployment my-agent-00001-deployment --replicas=10
```

### 6.3 Runbook: Database Migration

```bash
# 1. Enable maintenance mode
kubectl patch agent my-agent --type merge -p '{"spec":{"maintenance":true}}'

# 2. Run migration
kubectl exec -it postgres-0 -- psql -c "SELECT run_migration('v1.2.0')"

# 3. Verify migration
kubectl exec -it postgres-0 -- psql -c "SELECT version FROM schema_migrations"

# 4. Disable maintenance mode
kubectl patch agent my-agent --type merge -p '{"spec":{"maintenance":false}}'
```

---

## 7. Disaster Recovery

### 7.1 Backup Strategy

```text
┌─────────────────────────────────────────────────────────────────┐
│                      Backup Strategy                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Component         Frequency      Retention    RTO     RPO      │
│  ─────────────────────────────────────────────────────────────  │
│  PostgreSQL        Continuous     30 days      15m     0        │
│  Redis (RDB)       Hourly         7 days       5m      1h       │
│  Secrets           On-change      90 days      5m      0        │
│  Agent Configs     Git            Forever      1m      0        │
│  Logs              N/A            90 days      -       -        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 Recovery Procedure

```text
1. Assess damage and identify affected components
2. Notify stakeholders (status page update)
3. Restore from backup:
   - PostgreSQL: Point-in-time recovery
   - Secrets: Restore from vault backup
   - Configs: Git checkout
4. Validate data integrity
5. Restore traffic (gradual)
6. Post-mortem
```

---

## 8. Maintenance Windows

### 8.1 Types

| Type | Duration | Impact | Frequency |
|------|----------|--------|-----------|
| **Rolling Update** | 0 downtime | None | Weekly |
| **Canary Deploy** | 0 downtime | <1% traffic | Per release |
| **Full Maintenance** | 15-60 min | Full | Quarterly |

### 8.2 Maintenance Mode

```yaml
apiVersion: v1
kind: Service
metadata:
  name: maintenance-page
spec:
  selector:
    app: maintenance
---
# Route all traffic to maintenance page
kubectl patch virtualservice agentstack \
  --type merge \
  -p '{"spec":{"http":[{"route":[{"destination":{"host":"maintenance-page"}}]}]}}'
```

---

## 9. Implementation Checklist

### Phase 1: Basic CI/CD
- [ ] Container registry setup
- [ ] GitHub Actions pipeline
- [ ] Basic security scanning
- [ ] Staging environment

### Phase 2: GitOps
- [ ] ArgoCD deployment
- [ ] Environment separation
- [ ] Promotion workflow
- [ ] Rollback procedures

### Phase 3: Production Ready
- [ ] Multi-region deployment
- [ ] Disaster recovery
- [ ] Runbooks documented
- [ ] On-call rotation

---

## 10. References

- [ArgoCD](https://argo-cd.readthedocs.io/)
- [Knative Autoscaling](https://knative.dev/docs/serving/autoscaling/)
- [Kubernetes Multi-cluster](https://kubernetes.io/docs/concepts/cluster-administration/multi-cluster/)
- [GitOps Principles](https://opengitops.dev/)

---

**Previous**: [007-observability.md](007-observability.md)  
**Next**: [009-developer-experience.md](009-developer-experience.md)
