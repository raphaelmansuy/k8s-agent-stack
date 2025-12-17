# 006 - Security & Governance

> Authentication, Authorization, Secrets, and Compliance

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Security Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                    Security Architecture                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    PERIMETER                             │    │
│  │  WAF │ DDoS Protection │ TLS 1.3 │ Rate Limiting        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    AUTHENTICATION                        │    │
│  │  JWT │ API Keys │ OAuth 2.0 │ OIDC │ SAML (Enterprise)  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    AUTHORIZATION                         │    │
│  │  RBAC │ Project Scoping │ Resource Policies │ OPA       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    DATA PROTECTION                       │    │
│  │  Encryption at Rest │ Encryption in Transit │ Secrets   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    AUDIT & COMPLIANCE                    │    │
│  │  Audit Logs │ GDPR │ SOC 2 │ Data Residency            │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Authentication

### 2.1 Authentication Methods

| Method | Use Case | Token Lifetime |
|--------|----------|----------------|
| **JWT** | User sessions, UI | 1 hour (+ refresh) |
| **API Key** | Server-to-server, CI/CD | Until revoked |
| **OAuth 2.0** | Third-party integrations | Per provider |
| **OIDC** | Enterprise SSO | Session-based |
| **Service Token** | Internal services | Short-lived |

### 2.2 JWT Structure

```json
{
  "header": {
    "alg": "RS256",
    "typ": "JWT",
    "kid": "key-2025-01"
  },
  "payload": {
    "sub": "usr_abc123",
    "iss": "https://auth.agentstack.io",
    "aud": "https://api.agentstack.io",
    "exp": 1705316400,
    "iat": 1705312800,
    "teams": ["team_xyz"],
    "permissions": ["agents:write", "secrets:read"]
  }
}
```

### 2.3 API Key Format

```text
Format:  ask_{base62_encoded_payload}
Length:  48 characters
Example: ask_7Kx9mNpQr2Lm4jWvXyZ8bC5dF3gH1iJ6kL0aS
                    │
                    └── Contains: key_id, project_id, checksum

Storage: SHA-256 hash in database (never store plaintext)
```

### 2.4 OAuth 2.0 Flow

```text
┌────────┐                              ┌────────────┐
│  User  │                              │ AgentStack │
└───┬────┘                              └─────┬──────┘
    │                                         │
    │  1. Login Request                       │
    │────────────────────────────────────────►│
    │                                         │
    │  2. Redirect to IdP                     │
    │◄────────────────────────────────────────│
    │                                         │
    │         ┌─────────────┐                │
    │         │     IdP     │                │
    │         │ (Google/    │                │
    │         │  GitHub/    │                │
    │         │  Okta)      │                │
    │         └──────┬──────┘                │
    │                │                        │
    │  3. Authenticate                        │
    │───────────────►│                        │
    │                │                        │
    │  4. Auth Code  │                        │
    │◄───────────────│                        │
    │                                         │
    │  5. Exchange Code                       │
    │────────────────────────────────────────►│
    │                                         │
    │  6. JWT + Refresh Token                 │
    │◄────────────────────────────────────────│
```

---

## 3. Authorization (RBAC)

### 3.1 Role Hierarchy

```text
┌─────────────────────────────────────────────────────────────────┐
│                      RBAC Hierarchy                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Platform Level                                                  │
│  └── platform:admin    → Full platform access                   │
│                                                                  │
│  Team Level                                                      │
│  ├── team:owner        → Full team control, billing             │
│  ├── team:admin        → Manage members, all projects           │
│  ├── team:member       → Access granted projects                │
│  └── team:viewer       → Read-only access                       │
│                                                                  │
│  Project Level                                                   │
│  ├── project:admin     → Full project control                   │
│  ├── project:developer → Deploy, manage agents                  │
│  └── project:viewer    → Read-only                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Permission Matrix

| Permission | Owner | Admin | Member | Viewer |
|------------|:-----:|:-----:|:------:|:------:|
| `agents:read` | ✅ | ✅ | ✅ | ✅ |
| `agents:write` | ✅ | ✅ | ✅ | ❌ |
| `agents:delete` | ✅ | ✅ | ❌ | ❌ |
| `secrets:read` | ✅ | ✅ | ✅ | ❌ |
| `secrets:write` | ✅ | ✅ | ❌ | ❌ |
| `deployments:create` | ✅ | ✅ | ✅ | ❌ |
| `deployments:rollback` | ✅ | ✅ | ✅ | ❌ |
| `team:manage` | ✅ | ✅ | ❌ | ❌ |
| `billing:manage` | ✅ | ❌ | ❌ | ❌ |

### 3.3 Policy Enforcement

```go
// Middleware implementation
func Authorize(permission string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := UserFromContext(r.Context())
            project := ProjectFromContext(r.Context())
            
            if !user.HasPermission(project, permission) {
                response.Forbidden(w, r, "insufficient permissions")
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 4. Secrets Management

### 4.1 Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                    Secrets Management                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    API / CLI                             │    │
│  │  agentstack secrets set OPENAI_KEY=sk-...               │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                 Secrets Controller                       │    │
│  │  • Encrypt with KMS                                      │    │
│  │  • Store encrypted blob                                  │    │
│  │  • Create K8s Secret                                     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│           ┌──────────────────┼──────────────────┐               │
│           ▼                  ▼                  ▼               │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │  Database   │    │    KMS      │    │ K8s Secret  │         │
│  │ (encrypted  │    │ (key mgmt)  │    │ (runtime)   │         │
│  │   blob)     │    │             │    │             │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Encryption Scheme

```yaml
encryption:
  algorithm: AES-256-GCM
  key_management: 
    provider: aws-kms  # or gcp-kms, vault, local
    key_rotation: 90d
  envelope_encryption: true
  
# Envelope encryption flow:
# 1. Generate DEK (Data Encryption Key) per secret
# 2. Encrypt secret with DEK
# 3. Encrypt DEK with KEK (from KMS)
# 4. Store: encrypted_secret + encrypted_DEK
```

### 4.3 Secret Reference in Agent Config

```yaml
# Agent configuration
apiVersion: kagent.dev/v1alpha2
kind: Agent
spec:
  config:
    env:
      - name: OPENAI_API_KEY
        valueFrom:
          secretKeyRef:
            name: openai-credentials
            key: api-key
      - name: DATABASE_URL
        value: "{{secrets.DATABASE_URL}}"  # Template syntax
```

### 4.4 External Secrets Operator (Enterprise)

```yaml
# Integration with external secret stores
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: openai-secret
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: openai-credentials
  data:
    - secretKey: api-key
      remoteRef:
        key: secret/data/openai
        property: api_key
```

---

## 5. Network Security

### 5.1 Network Policies

```yaml
# Default deny all ingress
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-ingress
  namespace: agents
spec:
  podSelector: {}
  policyTypes:
    - Ingress

---
# Allow only from Knative ingress
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-knative-ingress
  namespace: agents
spec:
  podSelector:
    matchLabels:
      serving.knative.dev/service: "*"
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              app.kubernetes.io/name: knative-serving
```

### 5.2 TLS Configuration

```yaml
# Minimum TLS 1.2, prefer 1.3
tls:
  min_version: "1.2"
  cipher_suites:
    - TLS_AES_256_GCM_SHA384
    - TLS_CHACHA20_POLY1305_SHA256
    - TLS_AES_128_GCM_SHA256
  certificate_management:
    provider: cert-manager
    issuer: letsencrypt-prod
    auto_renewal: true
    renewal_before: 30d
```

### 5.3 mTLS (Optional)

```yaml
# For A2A communication between agents
mtls:
  enabled: true
  certificate_authority: internal-ca
  client_verification: required
```

---

## 6. Audit Logging

### 6.1 Audit Events

```json
{
  "event_id": "evt_abc123",
  "timestamp": "2025-01-15T10:30:00.123Z",
  "actor": {
    "type": "user",
    "id": "usr_xyz",
    "email": "user@example.com",
    "ip": "192.168.1.1",
    "user_agent": "AgentStack-CLI/1.0"
  },
  "action": "agent.deploy",
  "resource": {
    "type": "agent",
    "id": "agt_abc",
    "name": "customer-support"
  },
  "context": {
    "project_id": "prj_123",
    "team_id": "team_456"
  },
  "request": {
    "method": "POST",
    "path": "/v1/agents/agt_abc/deployments",
    "request_id": "req_789"
  },
  "result": {
    "status": "success",
    "status_code": 202
  },
  "changes": {
    "before": {"status": "active", "revision": "rev_old"},
    "after": {"status": "deploying", "revision": "rev_new"}
  }
}
```

### 6.2 Audited Actions

| Category | Actions |
|----------|---------|
| **Auth** | login, logout, token_refresh, api_key_create |
| **Agents** | create, update, delete, deploy, rollback |
| **Secrets** | create, update, delete, access |
| **Team** | member_add, member_remove, role_change |
| **Billing** | plan_change, payment_method_update |

### 6.3 Audit Log Storage

```yaml
audit:
  storage:
    primary: postgresql  # Real-time queries
    archive: s3          # Long-term retention
  retention:
    hot: 90d             # In PostgreSQL
    cold: 7y             # In S3 (compliance)
  immutability: true     # Append-only, no deletes
```

---

## 7. Compliance

### 7.1 GDPR Compliance

| Requirement | Implementation |
|-------------|----------------|
| **Data Minimization** | Collect only necessary data |
| **Right to Access** | Export API for user data |
| **Right to Deletion** | Anonymization/deletion workflow |
| **Data Portability** | Standard export formats (JSON) |
| **Consent** | Explicit consent tracking |
| **Data Residency** | Region-specific deployments |

### 7.2 Data Residency

```yaml
data_residency:
  regions:
    eu:
      database: eu-west-1
      storage: eu-west-1
      processing: eu-west-1
    us:
      database: us-east-1
      storage: us-east-1
      processing: us-east-1
  enforcement:
    project_setting: true
    network_policies: true
    cross_region_transfer: blocked
```

### 7.3 SOC 2 Controls

| Control | Implementation |
|---------|----------------|
| **CC6.1** | Logical access controls (RBAC) |
| **CC6.2** | User authentication (MFA support) |
| **CC6.3** | Authorization enforcement |
| **CC7.1** | System monitoring (audit logs) |
| **CC7.2** | Anomaly detection |

---

## 8. Quotas & Rate Limiting

### 8.1 Quota Types

```yaml
quotas:
  team_level:
    projects: 10
    members: 50
    
  project_level:
    agents: 100
    secrets: 500
    deployments_per_day: 1000
    
  agent_level:
    requests_per_minute: 600
    tokens_per_day: 10_000_000
    concurrent_sessions: 1000
```

### 8.2 Rate Limiting Implementation

```text
┌─────────────────────────────────────────────────────────────────┐
│                     Rate Limiting                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Algorithm: Token Bucket (via Redis)                            │
│                                                                  │
│  Key Format: ratelimit:{project_id}:{endpoint}:{window}         │
│                                                                  │
│  Layers:                                                         │
│  1. Global    → Protect infrastructure (10K/min)                │
│  2. Per-User  → Fair usage (varies by plan)                     │
│  3. Per-Agent → Prevent runaway costs                           │
│                                                                  │
│  Bypass:                                                         │
│  • Internal services (service tokens)                           │
│  • Health checks                                                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 9. Security Checklist

### Pre-Production

- [ ] All secrets encrypted at rest
- [ ] TLS 1.2+ enforced everywhere
- [ ] RBAC policies implemented
- [ ] Audit logging enabled
- [ ] Network policies applied
- [ ] Container images scanned
- [ ] Dependencies vulnerability scan
- [ ] Penetration testing completed

### Ongoing

- [ ] Key rotation (90-day cycle)
- [ ] Access reviews (quarterly)
- [ ] Security patches (< 7 days critical)
- [ ] Audit log review (weekly)
- [ ] Incident response drills (quarterly)

---

## 10. References

- [OWASP API Security Top 10](https://owasp.org/API-Security/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [External Secrets Operator](https://external-secrets.io/)

---

**Previous**: [005-data-architecture.md](005-data-architecture.md)  
**Next**: [007-observability.md](007-observability.md)
