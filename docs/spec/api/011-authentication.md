# 011 - Authentication & Authorization

> API Keys, JWT, OAuth, Rate Limiting, and Security

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Authentication Methods

```text
┌─────────────────────────────────────────────────────────────────┐
│                   Authentication Flow                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Client Request                                                 │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐        │
│  │  API Key?   │─No─▶│    JWT?     │─No─▶│   OAuth?    │        │
│  └──────┬──────┘     └──────┬──────┘     └──────┬──────┘        │
│         │Yes               │Yes               │Yes              │
│         ▼                  ▼                  ▼                 │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐        │
│  │  Validate   │     │   Verify    │     │  Token      │        │
│  │  Key + Scope│     │   Signature │     │  Exchange   │        │
│  └──────┬──────┘     └──────┬──────┘     └──────┬──────┘        │
│         │                  │                  │                 │
│         └──────────────────┼──────────────────┘                 │
│                            ▼                                    │
│                    ┌─────────────┐                              │
│                    │  Authorize  │                              │
│                    │   Request   │                              │
│                    └─────────────┘                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.1 API Keys

Primary method for programmatic access.

```yaml
# API Key Types
types:
  - name: Project Key
    prefix: as_prj_
    scope: Single project
    permissions: Full project access
    
  - name: Agent Key
    prefix: as_agt_
    scope: Single agent
    permissions: Chat only
    
  - name: Admin Key
    prefix: as_adm_
    scope: Organization
    permissions: Full admin access
```

**Request Example**:
```http
POST /v1/agents/agt_xxx/chat HTTP/1.1
Host: api.agentstack.io
X-API-Key: as_prj_sk_live_abc123xyz...
Content-Type: application/json
```

### 1.2 JWT (Bearer Token)

For user sessions and web applications.

```go
// JWT Claims
type Claims struct {
    jwt.RegisteredClaims
    UserID    string   `json:"uid"`
    Email     string   `json:"email"`
    OrgID     string   `json:"org_id"`
    TeamIDs   []string `json:"team_ids"`
    Roles     []string `json:"roles"`
    Scope     string   `json:"scope"`
}
```

**Request Example**:
```http
GET /v1/agents HTTP/1.1
Host: api.agentstack.io
Authorization: Bearer eyJhbGciOiJSUzI1NiIs...
X-Project-ID: prj_abc123
```

### 1.3 OAuth 2.0 / OIDC

For third-party integrations.

```text
Supported Flows:
• Authorization Code + PKCE (recommended)
• Client Credentials (M2M)

Supported Providers:
• GitHub
• Google
• Microsoft Entra ID
• Custom OIDC
```

---

## 2. API Key Management

### 2.1 Key Structure

```text
as_prj_sk_live_2xKj9mNpQrStUvWxYz0123456789abcdef
│  │   │  │    └─────────────────────────────────┘
│  │   │  │              Random entropy (32 bytes)
│  │   │  └── Environment (live/test)
│  │   └── Key type (sk=secret, pk=publishable)
│  └── Scope (prj=project, agt=agent, adm=admin)
└── Prefix (agentstack)
```

### 2.2 Key Endpoints

```yaml
# Create API Key
POST /v1/api-keys
{
  "name": "Production Backend",
  "scope": "project",
  "project_id": "prj_abc123",
  "permissions": ["agents:read", "agents:chat"],
  "expires_at": "2026-01-01T00:00:00Z"
}

# Response (key shown ONCE)
{
  "id": "key_xxx",
  "key": "as_prj_sk_live_abc123...",  # Only in create response
  "name": "Production Backend",
  "prefix": "as_prj_sk_live_abc1",    # For identification
  "created_at": "2025-01-15T10:00:00Z"
}

# List API Keys (values hidden)
GET /v1/api-keys
{
  "data": [
    {
      "id": "key_xxx",
      "name": "Production Backend",
      "prefix": "as_prj_sk_live_abc1",
      "last_used_at": "2025-01-15T10:30:00Z"
    }
  ]
}

# Revoke API Key
DELETE /v1/api-keys/{keyId}
```

### 2.3 Key Rotation

```text
Best Practice: 90-day rotation

1. Create new key
2. Update applications
3. Revoke old key (grace period: 7 days)
```

---

## 3. Permissions Model

### 3.1 Permission Format

```text
resource:action[:scope]

Examples:
• agents:read          - Read all agents
• agents:write         - Create/update agents
• agents:chat          - Chat with agents
• agents:delete        - Delete agents
• deployments:create   - Deploy agents
• secrets:read         - Read secret metadata
• secrets:write        - Create/update secrets
• admin:*              - Full admin access
```

### 3.2 RBAC Roles

| Role | Permissions |
|------|-------------|
| **Viewer** | `*:read` |
| **Developer** | `agents:*`, `tools:*`, `sessions:*` |
| **Admin** | `*:*` except `billing:*` |
| **Owner** | `*:*` |

### 3.3 Permission Check

```go
// Authorization middleware
func Authorize(required ...Permission) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            auth := AuthFromContext(r.Context())
            
            for _, perm := range required {
                if !auth.HasPermission(perm) {
                    respondError(w, 403, "Missing permission: "+perm.String())
                    return
                }
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// Usage
router.POST("/agents", 
    Authorize(Permission("agents:write")),
    createAgentHandler,
)
```

---

## 4. Rate Limiting

### 4.1 Limits by Plan

| Plan | Requests/min | Burst | Concurrent |
|------|--------------|-------|------------|
| **Free** | 60 | 100 | 5 |
| **Pro** | 600 | 1,000 | 100 |
| **Enterprise** | 6,000 | 10,000 | 1,000 |

### 4.2 Rate Limit Headers

```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 600
X-RateLimit-Remaining: 542
X-RateLimit-Reset: 1705312800
X-RateLimit-Policy: 600;w=60
```

### 4.3 Rate Limit Response

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 45
Content-Type: application/json

{
  "type": "https://api.agentstack.io/errors/rate-limited",
  "title": "Rate Limited",
  "status": 429,
  "detail": "Rate limit exceeded. Retry after 45 seconds."
}
```

### 4.4 Implementation

```go
// Redis-based rate limiter
func RateLimiter(limit int, window time.Duration) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := rateLimitKey(r)  // project_id + endpoint
            
            count, reset, err := redis.Incr(key, window)
            if err != nil {
                // Fail open on Redis errors
                next.ServeHTTP(w, r)
                return
            }
            
            w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
            w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, limit-count)))
            w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
            
            if count > limit {
                w.Header().Set("Retry-After", strconv.Itoa(int(reset.Sub(time.Now()).Seconds())))
                respondError(w, 429, "Rate limit exceeded")
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 5. Request Security

### 5.1 Request Signing (Optional)

For high-security integrations:

```http
POST /v1/agents HTTP/1.1
X-API-Key: as_prj_sk_live_xxx
X-Timestamp: 1705312800
X-Signature: sha256=abc123...
```

```go
// Signature calculation
signature := hmac.New(sha256.New, []byte(apiSecret))
signature.Write([]byte(timestamp + "." + body))
expected := hex.EncodeToString(signature.Sum(nil))
```

### 5.2 Idempotency

```http
POST /v1/agents HTTP/1.1
Idempotency-Key: user-request-12345
```

```go
// Idempotency handling
func Idempotent(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        key := r.Header.Get("Idempotency-Key")
        if key == "" {
            next.ServeHTTP(w, r)
            return
        }
        
        // Check cache
        if cached, ok := redis.Get("idempotency:" + key); ok {
            w.Header().Set("Idempotency-Replayed", "true")
            w.Write(cached)
            return
        }
        
        // Capture response
        rec := httptest.NewRecorder()
        next.ServeHTTP(rec, r)
        
        // Cache for 24 hours
        redis.Set("idempotency:"+key, rec.Body.Bytes(), 24*time.Hour)
        
        // Copy to actual response
        for k, v := range rec.Header() {
            w.Header()[k] = v
        }
        w.WriteHeader(rec.Code)
        w.Write(rec.Body.Bytes())
    })
}
```

### 5.3 Request Validation

```go
// Input validation
type CreateAgentRequest struct {
    Name        string   `json:"name" validate:"required,min=3,max=64,slug"`
    Description string   `json:"description" validate:"max=500"`
    Framework   string   `json:"framework" validate:"required,oneof=google-adk langchain crewai"`
    Tags        []string `json:"tags" validate:"max=10,dive,max=32"`
}
```

---

## 6. Security Headers

### 6.1 Response Headers

```http
HTTP/1.1 200 OK
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'none'
Cache-Control: no-store
```

### 6.2 CORS Configuration

```yaml
cors:
  allowed_origins:
    - https://app.agentstack.io
    - https://*.agentstack.app
  allowed_methods:
    - GET
    - POST
    - PATCH
    - DELETE
  allowed_headers:
    - Authorization
    - X-API-Key
    - X-Project-ID
    - Content-Type
    - Idempotency-Key
  max_age: 86400
```

---

## 7. Audit Logging

### 7.1 Authentication Events

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "event": "auth.success",
  "method": "api_key",
  "key_id": "key_xxx",
  "key_prefix": "as_prj_sk_live_abc1",
  "project_id": "prj_abc123",
  "ip": "203.0.113.1",
  "user_agent": "agentctl/1.0"
}
```

### 7.2 Security Events

| Event | Description |
|-------|-------------|
| `auth.success` | Successful authentication |
| `auth.failure` | Failed authentication |
| `auth.rate_limited` | Rate limit triggered |
| `key.created` | API key created |
| `key.revoked` | API key revoked |
| `permission.denied` | Authorization failure |

---

## 8. API Endpoints

### 8.1 Authentication

```yaml
# Login (returns JWT)
POST /v1/auth/login
{
  "email": "user@example.com",
  "password": "..."
}

# OAuth callback
GET /v1/auth/oauth/{provider}/callback?code=xxx

# Refresh token
POST /v1/auth/refresh
{
  "refresh_token": "..."
}

# Logout
POST /v1/auth/logout
```

### 8.2 API Keys

```yaml
# List keys
GET /v1/api-keys

# Create key
POST /v1/api-keys
{
  "name": "Backend API",
  "scope": "project",
  "permissions": ["agents:read", "agents:chat"]
}

# Get key metadata
GET /v1/api-keys/{keyId}

# Update key
PATCH /v1/api-keys/{keyId}
{
  "name": "Updated Name"
}

# Revoke key
DELETE /v1/api-keys/{keyId}
```

---

## 9. Implementation Checklist

- [ ] API key generation and validation
- [ ] JWT signing and verification
- [ ] OAuth provider integration
- [ ] Rate limiting with Redis
- [ ] Permission checking middleware
- [ ] Audit logging
- [ ] Key rotation workflow
- [ ] Security headers

---

## 10. References

- [OAuth 2.0](https://oauth.net/2/)
- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)
- [API Key Design](https://cloud.google.com/endpoints/docs/openapi/when-why-api-key)
- [Rate Limiting Patterns](https://stripe.com/docs/rate-limits)

---

**Previous**: [010-multi-tenancy.md](010-multi-tenancy.md)  
**Next**: [012-agents-endpoints.md](012-agents-endpoints.md)
