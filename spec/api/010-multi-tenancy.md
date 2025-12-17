# 010 - Multi-Tenancy Architecture

> Tenant Isolation, Resource Hierarchy, and Data Partitioning

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Tenant Model

### 1.1 Definition

In AgentStack, a **tenant** is a **Project**. Projects provide:
- Data isolation boundary
- Billing unit
- Quota enforcement scope
- API key scoping

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Tenant Hierarchy                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Organization (org_xxx)                                         │
│  │   • Enterprise billing entity                                │
│  │   • SSO/SAML configuration                                   │
│  │   • Cross-team policies                                      │
│  │                                                               │
│  └── Team (team_xxx)                                            │
│      │   • User management                                      │
│      │   • Role assignments                                     │
│      │   • Shared resources                                     │
│      │                                                           │
│      └── Project (prj_xxx)  ◄── TENANT BOUNDARY                 │
│          │   • Data isolation                                   │
│          │   • Resource quotas                                  │
│          │   • API key scope                                    │
│          │                                                       │
│          ├── Agents                                             │
│          ├── Tools                                              │
│          ├── Secrets                                            │
│          └── Webhooks                                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Tenant Identification

| Method | Use Case | Example |
|--------|----------|---------|
| **Header** | Multi-project API keys | `X-Project-ID: prj_xxx` |
| **Path** | Explicit routing | `/projects/prj_xxx/agents` |
| **Key-scoped** | Single-project keys | API key tied to project |

**Priority**: Key-scoped > Header > Path

---

## 2. Isolation Models

### 2.1 Data Isolation

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Data Isolation Strategy                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  LOGICAL ISOLATION (Default)                                    │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Shared PostgreSQL                                       │    │
│  │  ├── project_id column on all tables                    │    │
│  │  ├── Row-Level Security (RLS) policies                  │    │
│  │  └── Indexed for performance                            │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  SCHEMA ISOLATION (Pro)                                         │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Separate schema per project                            │    │
│  │  ├── prj_abc.agents                                     │    │
│  │  ├── prj_abc.sessions                                   │    │
│  │  └── Automatic schema routing                           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  DATABASE ISOLATION (Enterprise)                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Dedicated database per organization                    │    │
│  │  ├── Full resource isolation                            │    │
│  │  ├── Custom backup schedules                            │    │
│  │  └── Dedicated connection pool                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Compute Isolation

| Level | Description | Use Case |
|-------|-------------|----------|
| **Shared** | Agents on shared Knative cluster | Free/Pro |
| **Namespace** | Dedicated K8s namespace | Pro |
| **Node Pool** | Dedicated node pool | Enterprise |
| **Cluster** | Dedicated K8s cluster | Enterprise+ |

### 2.3 Network Isolation

```yaml
# NetworkPolicy per project namespace
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: project-isolation
  namespace: prj-abc123
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              agentstack.io/system: "true"
        - namespaceSelector:
            matchLabels:
              agentstack.io/project: prj-abc123
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              agentstack.io/system: "true"
    - to: []  # Allow external egress
```

---

## 3. Resource Hierarchy

### 3.1 Ownership Chain

```text
All resources belong to exactly one project:

Agent → Project → Team → Organization
Tool  → Project → Team → Organization
```

### 3.2 Cross-Tenant Access

**Principle**: No cross-tenant access by default.

Exceptions (opt-in):
- Organization-level tool sharing
- Team-level model configs
- Public agent discovery (marketplace)

```yaml
# Tool sharing configuration
apiVersion: agentstack.io/v1
kind: Tool
metadata:
  name: shared-search
  namespace: org-acme
spec:
  sharing:
    scope: organization  # none | team | organization
    allowedProjects:
      - prj_abc
      - prj_xyz
```

---

## 4. Tenant Context Propagation

### 4.1 API Layer

```go
// Middleware extracts and validates tenant context
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Extract from API key (highest priority)
        tenant, err := extractFromAPIKey(r)
        if err != nil {
            // 2. Fallback to header
            tenant, err = extractFromHeader(r, "X-Project-ID")
        }
        if err != nil {
            // 3. Fallback to path
            tenant, err = extractFromPath(r)
        }
        
        if tenant == "" {
            respondError(w, 400, "Tenant context required")
            return
        }
        
        // Validate tenant access
        if !canAccess(r.Context(), tenant) {
            respondError(w, 403, "Access denied to project")
            return
        }
        
        // Inject into context
        ctx := context.WithValue(r.Context(), TenantKey, tenant)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 4.2 Database Layer

```sql
-- Row-Level Security policy
CREATE POLICY tenant_isolation ON agents
    USING (project_id = current_setting('app.project_id'));

-- Set tenant context before queries
SET LOCAL app.project_id = 'prj_abc123';
SELECT * FROM agents;  -- Only returns prj_abc123 agents
```

### 4.3 Trace Context

```go
// Propagate tenant in distributed traces
span.SetAttributes(
    attribute.String("tenant.project_id", projectID),
    attribute.String("tenant.team_id", teamID),
    attribute.String("tenant.org_id", orgID),
)
```

---

## 5. Quota Management

### 5.1 Quota Types

| Quota | Free | Pro | Enterprise |
|-------|------|-----|------------|
| Agents per project | 3 | 50 | Unlimited |
| Deployments/day | 10 | 100 | Unlimited |
| Requests/min | 60 | 600 | 6,000 |
| Token budget/month | 1M | 50M | Custom |
| Storage (GB) | 1 | 50 | Custom |
| Team members | 3 | 25 | Unlimited |

### 5.2 Quota Enforcement

```go
// Check quota before resource creation
func (s *AgentService) Create(ctx context.Context, req CreateAgentRequest) (*Agent, error) {
    tenant := TenantFromContext(ctx)
    
    // Check agent count quota
    count, _ := s.repo.CountAgents(ctx, tenant.ProjectID)
    if count >= tenant.Quotas.MaxAgents {
        return nil, QuotaExceededError{
            Resource: "agents",
            Limit:    tenant.Quotas.MaxAgents,
            Current:  count,
        }
    }
    
    return s.repo.CreateAgent(ctx, req)
}
```

### 5.3 Quota Headers

```http
HTTP/1.1 200 OK
X-Quota-Limit-Agents: 50
X-Quota-Remaining-Agents: 23
X-Quota-Limit-Requests: 600
X-Quota-Remaining-Requests: 542
X-Quota-Reset: 2025-01-15T11:00:00Z
```

---

## 6. Noisy Neighbor Prevention

### 6.1 Rate Limiting

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Rate Limiting Strategy                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Layer 1: Global (Platform Protection)                          │
│  └── 100,000 req/s total platform capacity                     │
│                                                                  │
│  Layer 2: Organization                                          │
│  └── Enterprise SLA-based limits                               │
│                                                                  │
│  Layer 3: Project (Tenant)                                      │
│  └── Plan-based limits (60/600/6000 req/min)                   │
│                                                                  │
│  Layer 4: Agent                                                 │
│  └── Per-agent concurrency (10/100/1000)                       │
│                                                                  │
│  Layer 5: User (optional)                                       │
│  └── Per-user limits within tenant                             │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2 Resource Limits

```yaml
# Default resource limits per agent
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "1Gi"
    cpu: "1000m"

# Enterprise: Custom limits per project
projectOverrides:
  prj_enterprise:
    resources:
      limits:
        memory: "8Gi"
        cpu: "4000m"
```

### 6.3 Queue Isolation

```text
Each project gets isolated queues:

prj_abc123:
  └── agent-tasks-prj_abc123
  └── deployment-jobs-prj_abc123
  └── webhook-deliveries-prj_abc123
```

---

## 7. Tenant Lifecycle

### 7.1 Provisioning

```text
1. User creates project
   └── Generate project_id (prj_xxx)
   └── Create database records
   └── Initialize quotas
   └── Generate default API key

2. First agent deployment
   └── Create K8s namespace (if namespace isolation)
   └── Apply NetworkPolicies
   └── Configure resource quotas
```

### 7.2 Migration

```go
// Move project between teams
func MoveProject(ctx context.Context, projectID, newTeamID string) error {
    // 1. Validate permissions
    // 2. Update ownership
    // 3. Migrate billing
    // 4. Update audit logs
    return nil
}
```

### 7.3 Deletion

```text
Soft Delete (30 days retention):
1. Mark project as deleted
2. Suspend all agents
3. Revoke API keys
4. Stop billing

Hard Delete (after retention):
1. Delete all agents
2. Purge database records
3. Delete K8s namespace
4. Remove from backups
```

---

## 8. Audit & Compliance

### 8.1 Audit Log Schema

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "event_type": "agent.created",
  "actor": {
    "type": "user",
    "id": "usr_abc",
    "email": "user@example.com"
  },
  "tenant": {
    "project_id": "prj_abc123",
    "team_id": "team_xyz",
    "org_id": "org_acme"
  },
  "resource": {
    "type": "agent",
    "id": "agt_123",
    "name": "customer-support"
  },
  "changes": {
    "status": {"old": null, "new": "active"}
  },
  "request": {
    "ip": "203.0.113.1",
    "user_agent": "agentctl/1.0"
  }
}
```

### 8.2 Data Residency

```yaml
# Project-level data residency configuration
apiVersion: agentstack.io/v1
kind: Project
metadata:
  name: eu-project
spec:
  dataResidency:
    region: eu-west-1
    compliance:
      - gdpr
      - iso27001
    encryption:
      keyProvider: customer-managed
      keyId: arn:aws:kms:eu-west-1:xxx
```

---

## 9. Best Practices Summary

| Practice | Implementation |
|----------|----------------|
| **Always scope queries** | Include project_id in all queries |
| **Validate tenant context** | Middleware on every request |
| **Use RLS** | Database-level isolation |
| **Log tenant context** | All logs include project_id |
| **Separate queues** | Per-tenant job queues |
| **Monitor per-tenant** | Metrics labeled by tenant |
| **Quota before action** | Check limits pre-mutation |
| **Soft delete first** | Retention period before purge |

---

## 10. References

- [Azure Multi-tenant Architecture](https://learn.microsoft.com/en-us/azure/architecture/guide/multitenant/overview)
- [Stripe API Design](https://stripe.com/docs/api)
- [PostgreSQL RLS](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [Kubernetes Multi-tenancy](https://kubernetes.io/docs/concepts/security/multi-tenancy/)

---

**Next**: [011-authentication.md](011-authentication.md)
