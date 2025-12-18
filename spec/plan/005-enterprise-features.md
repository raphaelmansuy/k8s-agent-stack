# Phase 5: Enterprise Features

> RBAC, Quotas, Audit Logging, Multi-tenancy

**Duration**: 3 weeks | **Status**: Not Started | **Priority**: Medium  
**Depends On**: Phase 1 (Core API), Phase 2 (Agent Runtime)

---

## Objectives

1. Implement role-based access control (RBAC)
2. Build resource quota system
3. Create comprehensive audit logging
4. Enhance multi-tenancy isolation
5. Implement API rate limiting per tenant

---

## Architecture Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                      Enterprise Security Architecture                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                      API Gateway                                 │   │
│   │                                                                  │   │
│   │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐            │   │
│   │  │ Auth    │  │ RBAC    │  │ Rate    │  │ Audit   │            │   │
│   │  │ Filter  │─▶│ Filter  │─▶│ Limiter │─▶│ Logger  │            │   │
│   │  └─────────┘  └─────────┘  └─────────┘  └─────────┘            │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                              │                                          │
│                              ▼                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                    Business Logic Layer                          │   │
│   │                                                                  │   │
│   │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │   │
│   │  │   Policy    │  │   Quota     │  │   Resource Isolation    │  │   │
│   │  │   Engine    │  │   Manager   │  │   (Team/Project scope)  │  │   │
│   │  └─────────────┘  └─────────────┘  └─────────────────────────┘  │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│   ┌───────────────┐  ┌───────────────┐  ┌───────────────────────────┐   │
│   │   PostgreSQL  │  │    Redis      │  │   Audit Log Storage       │   │
│   │   (Policies)  │  │   (Quotas)    │  │   (Clickhouse/BigQuery)   │   │
│   └───────────────┘  └───────────────┘  └───────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Week 1: RBAC Implementation

### 1.1 Permission Model

**`internal/domain/rbac/model.go`**:
```go
package rbac

import "time"

// Resource represents a protected resource type
type Resource string

const (
	ResourceTeam       Resource = "team"
	ResourceProject    Resource = "project"
	ResourceAgent      Resource = "agent"
	ResourceDeployment Resource = "deployment"
	ResourceAPIKey     Resource = "api_key"
	ResourceMember     Resource = "member"
	ResourceQuota      Resource = "quota"
	ResourceAuditLog   Resource = "audit_log"
)

// Action represents an operation on a resource
type Action string

const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionList   Action = "list"
	ActionInvoke Action = "invoke"
	ActionDeploy Action = "deploy"
	ActionManage Action = "manage" // Full control
)

// Role defines a set of permissions
type Role struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
	IsSystem    bool         `json:"is_system"`
	TeamID      string       `json:"team_id,omitempty"` // Null for system roles
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Permission grants access to perform an action on a resource
type Permission struct {
	Resource   Resource `json:"resource"`
	Action     Action   `json:"action"`
	Conditions []string `json:"conditions,omitempty"` // e.g., "own", "team", "project"
}

// RoleBinding assigns a role to a user for a scope
type RoleBinding struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	RoleID    string    `json:"role_id"`
	Scope     Scope     `json:"scope"`
	GrantedBy string    `json:"granted_by"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Scope defines where a role applies
type Scope struct {
	Type      ScopeType `json:"type"`
	TeamID    string    `json:"team_id,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
}

type ScopeType string

const (
	ScopeTeam    ScopeType = "team"
	ScopeProject ScopeType = "project"
)

// Pre-defined system roles
var SystemRoles = []Role{
	{
		ID:          "role_owner",
		Name:        "Owner",
		Description: "Full access to all resources in the team",
		IsSystem:    true,
		Permissions: []Permission{
			{Resource: ResourceTeam, Action: ActionManage},
			{Resource: ResourceProject, Action: ActionManage},
			{Resource: ResourceAgent, Action: ActionManage},
			{Resource: ResourceDeployment, Action: ActionManage},
			{Resource: ResourceAPIKey, Action: ActionManage},
			{Resource: ResourceMember, Action: ActionManage},
			{Resource: ResourceQuota, Action: ActionRead},
			{Resource: ResourceAuditLog, Action: ActionRead},
		},
	},
	{
		ID:          "role_admin",
		Name:        "Admin",
		Description: "Manage projects and agents, invite members",
		IsSystem:    true,
		Permissions: []Permission{
			{Resource: ResourceTeam, Action: ActionRead},
			{Resource: ResourceProject, Action: ActionManage},
			{Resource: ResourceAgent, Action: ActionManage},
			{Resource: ResourceDeployment, Action: ActionManage},
			{Resource: ResourceAPIKey, Action: ActionManage},
			{Resource: ResourceMember, Action: ActionCreate},
			{Resource: ResourceMember, Action: ActionRead},
			{Resource: ResourceQuota, Action: ActionRead},
			{Resource: ResourceAuditLog, Action: ActionRead},
		},
	},
	{
		ID:          "role_developer",
		Name:        "Developer",
		Description: "Create and deploy agents",
		IsSystem:    true,
		Permissions: []Permission{
			{Resource: ResourceTeam, Action: ActionRead},
			{Resource: ResourceProject, Action: ActionRead},
			{Resource: ResourceAgent, Action: ActionCreate},
			{Resource: ResourceAgent, Action: ActionRead},
			{Resource: ResourceAgent, Action: ActionUpdate, Conditions: []string{"own"}},
			{Resource: ResourceDeployment, Action: ActionCreate},
			{Resource: ResourceDeployment, Action: ActionRead},
			{Resource: ResourceAPIKey, Action: ActionCreate, Conditions: []string{"own"}},
			{Resource: ResourceAPIKey, Action: ActionRead, Conditions: []string{"own"}},
		},
	},
	{
		ID:          "role_viewer",
		Name:        "Viewer",
		Description: "Read-only access",
		IsSystem:    true,
		Permissions: []Permission{
			{Resource: ResourceTeam, Action: ActionRead},
			{Resource: ResourceProject, Action: ActionRead},
			{Resource: ResourceAgent, Action: ActionRead},
			{Resource: ResourceDeployment, Action: ActionRead},
		},
	},
}
```

### 1.2 RBAC Service

**`internal/domain/rbac/service.go`**:
```go
package rbac

import (
	"context"
	"fmt"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
)

type Service struct {
	repo  Repository
	cache *cache.RedisClient
}

type Repository interface {
	// Roles
	GetRole(ctx context.Context, id string) (*Role, error)
	ListRoles(ctx context.Context, teamID string) ([]Role, error)
	CreateRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id string) error

	// Role Bindings
	GetRoleBindings(ctx context.Context, userID string) ([]RoleBinding, error)
	CreateRoleBinding(ctx context.Context, rb *RoleBinding) error
	DeleteRoleBinding(ctx context.Context, id string) error
	GetRoleBindingsByScope(ctx context.Context, scope Scope) ([]RoleBinding, error)
}

func NewService(repo Repository, cache *cache.RedisClient) *Service {
	return &Service{
		repo:  repo,
		cache: cache,
	}
}

// CheckPermission verifies if a user has permission to perform an action
func (s *Service) CheckPermission(ctx context.Context, req PermissionRequest) (*PermissionResult, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("perm:%s:%s:%s:%s",
		req.UserID, req.Resource, req.Action, req.ScopeID)

	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		return &PermissionResult{Allowed: cached == "1"}, nil
	}

	// Get user's role bindings
	bindings, err := s.repo.GetRoleBindings(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	// Check each binding
	allowed := false
	var matchedRole *Role

	for _, binding := range bindings {
		// Check scope match
		if !s.scopeMatches(binding.Scope, req.ScopeType, req.ScopeID) {
			continue
		}

		// Check expiration
		if binding.ExpiresAt != nil && binding.ExpiresAt.Before(time.Now()) {
			continue
		}

		// Get role
		role, err := s.repo.GetRole(ctx, binding.RoleID)
		if err != nil {
			continue
		}

		// Check permissions
		if s.hasPermission(role, req.Resource, req.Action, req.Conditions) {
			allowed = true
			matchedRole = role
			break
		}
	}

	// Cache result (5 minutes)
	cacheValue := "0"
	if allowed {
		cacheValue = "1"
	}
	s.cache.SetEX(ctx, cacheKey, cacheValue, 5*time.Minute)

	result := &PermissionResult{
		Allowed: allowed,
	}
	if matchedRole != nil {
		result.Role = matchedRole.Name
	}

	return result, nil
}

func (s *Service) scopeMatches(binding Scope, scopeType ScopeType, scopeID string) bool {
	// Team scope includes all projects in team
	if binding.Type == ScopeTeam && scopeType == ScopeProject {
		// Check if project belongs to team
		// This would need a lookup, simplified here
		return true
	}

	if binding.Type == scopeType {
		if scopeType == ScopeTeam {
			return binding.TeamID == scopeID
		}
		return binding.ProjectID == scopeID
	}

	return false
}

func (s *Service) hasPermission(role *Role, resource Resource, action Action, conditions map[string]string) bool {
	for _, perm := range role.Permissions {
		if perm.Resource != resource {
			continue
		}

		// Check action match
		if perm.Action == ActionManage || perm.Action == action {
			// Check conditions if any
			if len(perm.Conditions) == 0 {
				return true
			}

			for _, cond := range perm.Conditions {
				if s.conditionMet(cond, conditions) {
					return true
				}
			}
		}
	}
	return false
}

func (s *Service) conditionMet(condition string, context map[string]string) bool {
	switch condition {
	case "own":
		return context["owner_id"] == context["user_id"]
	case "team":
		return context["team_id"] != ""
	case "project":
		return context["project_id"] != ""
	default:
		return false
	}
}

// AssignRole assigns a role to a user
func (s *Service) AssignRole(ctx context.Context, req AssignRoleRequest) (*RoleBinding, error) {
	// Validate role exists
	role, err := s.repo.GetRole(ctx, req.RoleID)
	if err != nil {
		return nil, fmt.Errorf("role not found: %w", err)
	}

	// Check if assigner has permission to assign roles
	// ...

	binding := &RoleBinding{
		ID:        generateID("rb"),
		UserID:    req.UserID,
		RoleID:    role.ID,
		Scope:     req.Scope,
		GrantedBy: req.GrantedBy,
		CreatedAt: time.Now(),
		ExpiresAt: req.ExpiresAt,
	}

	if err := s.repo.CreateRoleBinding(ctx, binding); err != nil {
		return nil, err
	}

	// Invalidate cache
	s.invalidateUserPermissions(ctx, req.UserID)

	return binding, nil
}

// RevokeRole removes a role from a user
func (s *Service) RevokeRole(ctx context.Context, bindingID string, revokedBy string) error {
	// Get binding to find user for cache invalidation
	// Then delete
	if err := s.repo.DeleteRoleBinding(ctx, bindingID); err != nil {
		return err
	}

	// Invalidate cache would need user ID from binding
	return nil
}

func (s *Service) invalidateUserPermissions(ctx context.Context, userID string) {
	// Delete all permission cache entries for this user
	pattern := fmt.Sprintf("perm:%s:*", userID)
	keys, _ := s.cache.Keys(ctx, pattern)
	for _, key := range keys {
		s.cache.Del(ctx, key)
	}
}

// Types

type PermissionRequest struct {
	UserID     string
	Resource   Resource
	Action     Action
	ScopeType  ScopeType
	ScopeID    string
	Conditions map[string]string
}

type PermissionResult struct {
	Allowed bool   `json:"allowed"`
	Role    string `json:"role,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type AssignRoleRequest struct {
	UserID    string
	RoleID    string
	Scope     Scope
	GrantedBy string
	ExpiresAt *time.Time
}
```

### 1.3 RBAC Middleware

**`internal/api/middleware/rbac.go`**:
```go
package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

type RBACMiddleware struct {
	rbacSvc *rbac.Service
}

func NewRBACMiddleware(rbacSvc *rbac.Service) *RBACMiddleware {
	return &RBACMiddleware{rbacSvc: rbacSvc}
}

// RequirePermission creates middleware that checks for a specific permission
func (m *RBACMiddleware) RequirePermission(resource rbac.Resource, action rbac.Action) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := GetAuthContext(c)
		if auth == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}

		// Determine scope from request
		scope := m.extractScope(c)

		// Build conditions from request context
		conditions := map[string]string{
			"user_id": auth.UserID,
		}

		// Add resource owner if available
		if ownerID := c.Locals("resource_owner_id"); ownerID != nil {
			conditions["owner_id"] = ownerID.(string)
		}

		result, err := m.rbacSvc.CheckPermission(c.UserContext(), rbac.PermissionRequest{
			UserID:     auth.UserID,
			Resource:   resource,
			Action:     action,
			ScopeType:  scope.Type,
			ScopeID:    scope.ID,
			Conditions: conditions,
		})

		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "permission check failed")
		}

		if !result.Allowed {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}

		// Store result for audit logging
		c.Locals("rbac_result", result)

		return c.Next()
	}
}

func (m *RBACMiddleware) extractScope(c *fiber.Ctx) struct{ Type rbac.ScopeType; ID string } {
	// Try to extract project ID from various sources
	if projectID := c.Params("projectId"); projectID != "" {
		return struct{ Type rbac.ScopeType; ID string }{
			Type: rbac.ScopeProject,
			ID:   projectID,
		}
	}

	if projectID := c.Query("project_id"); projectID != "" {
		return struct{ Type rbac.ScopeType; ID string }{
			Type: rbac.ScopeProject,
			ID:   projectID,
		}
	}

	// Fall back to team scope
	auth := GetAuthContext(c)
	return struct{ Type rbac.ScopeType; ID string }{
		Type: rbac.ScopeTeam,
		ID:   auth.TeamID,
	}
}
```

---

## Week 2: Quota System

### 2.1 Quota Model

**`internal/domain/quota/model.go`**:
```go
package quota

import "time"

// QuotaType defines the type of resource being limited
type QuotaType string

const (
	QuotaAgents           QuotaType = "agents"
	QuotaDeployments      QuotaType = "deployments"
	QuotaAPIRequests      QuotaType = "api_requests"
	QuotaChatMessages     QuotaType = "chat_messages"
	QuotaTokens           QuotaType = "tokens"
	QuotaStorage          QuotaType = "storage_bytes"
	QuotaConcurrentChats  QuotaType = "concurrent_chats"
)

// Quota defines limits for a team or project
type Quota struct {
	ID         string    `json:"id"`
	TeamID     string    `json:"team_id"`
	ProjectID  string    `json:"project_id,omitempty"`
	Type       QuotaType `json:"type"`
	Limit      int64     `json:"limit"`
	Period     string    `json:"period,omitempty"` // "hour", "day", "month" for rate limits
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Usage tracks current consumption
type Usage struct {
	QuotaID   string    `json:"quota_id"`
	Current   int64     `json:"current"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Plan defines a set of quotas for a subscription tier
type Plan struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Quotas      map[QuotaType]int64 `json:"quotas"`
	PeriodQuotas map[QuotaType]PeriodQuota `json:"period_quotas"`
	Price       float64          `json:"price"`
	IsActive    bool             `json:"is_active"`
}

type PeriodQuota struct {
	Limit  int64  `json:"limit"`
	Period string `json:"period"`
}

// Pre-defined plans
var DefaultPlans = []Plan{
	{
		ID:          "plan_free",
		Name:        "Free",
		Description: "For personal projects and experimentation",
		Quotas: map[QuotaType]int64{
			QuotaAgents:          3,
			QuotaDeployments:     10,
			QuotaStorage:         100 * 1024 * 1024, // 100MB
			QuotaConcurrentChats: 5,
		},
		PeriodQuotas: map[QuotaType]PeriodQuota{
			QuotaAPIRequests:  {Limit: 1000, Period: "hour"},
			QuotaChatMessages: {Limit: 500, Period: "day"},
			QuotaTokens:       {Limit: 100000, Period: "month"},
		},
		Price:    0,
		IsActive: true,
	},
	{
		ID:          "plan_pro",
		Name:        "Pro",
		Description: "For professional teams",
		Quotas: map[QuotaType]int64{
			QuotaAgents:          25,
			QuotaDeployments:     100,
			QuotaStorage:         10 * 1024 * 1024 * 1024, // 10GB
			QuotaConcurrentChats: 50,
		},
		PeriodQuotas: map[QuotaType]PeriodQuota{
			QuotaAPIRequests:  {Limit: 50000, Period: "hour"},
			QuotaChatMessages: {Limit: 10000, Period: "day"},
			QuotaTokens:       {Limit: 5000000, Period: "month"},
		},
		Price:    99,
		IsActive: true,
	},
	{
		ID:          "plan_enterprise",
		Name:        "Enterprise",
		Description: "For large organizations",
		Quotas: map[QuotaType]int64{
			QuotaAgents:          -1, // Unlimited
			QuotaDeployments:     -1,
			QuotaStorage:         -1,
			QuotaConcurrentChats: -1,
		},
		PeriodQuotas: map[QuotaType]PeriodQuota{
			QuotaAPIRequests:  {Limit: -1, Period: "hour"},
			QuotaChatMessages: {Limit: -1, Period: "day"},
			QuotaTokens:       {Limit: -1, Period: "month"},
		},
		Price:    0, // Custom pricing
		IsActive: true,
	},
}
```

### 2.2 Quota Service

**`internal/domain/quota/service.go`**:
```go
package quota

import (
	"context"
	"fmt"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
)

type Service struct {
	repo  Repository
	cache *cache.RedisClient
}

type Repository interface {
	GetQuota(ctx context.Context, teamID string, quotaType QuotaType) (*Quota, error)
	GetQuotas(ctx context.Context, teamID string) ([]Quota, error)
	SetQuota(ctx context.Context, quota *Quota) error
	GetUsage(ctx context.Context, quotaID string) (*Usage, error)
}

func NewService(repo Repository, cache *cache.RedisClient) *Service {
	return &Service{
		repo:  repo,
		cache: cache,
	}
}

// CheckQuota verifies if an action would exceed quota
func (s *Service) CheckQuota(ctx context.Context, req CheckQuotaRequest) (*CheckQuotaResult, error) {
	// Get quota definition
	quota, err := s.repo.GetQuota(ctx, req.TeamID, req.Type)
	if err != nil {
		// No quota defined = unlimited
		return &CheckQuotaResult{Allowed: true}, nil
	}

	// Unlimited quota
	if quota.Limit == -1 {
		return &CheckQuotaResult{Allowed: true}, nil
	}

	// Get current usage from cache (faster) or DB
	var current int64
	if quota.Period != "" {
		// Rate-limited quota - use Redis
		current, _ = s.getRateLimitUsage(ctx, quota)
	} else {
		// Absolute quota - use DB
		usage, err := s.repo.GetUsage(ctx, quota.ID)
		if err == nil {
			current = usage.Current
		}
	}

	allowed := current+req.Amount <= quota.Limit

	return &CheckQuotaResult{
		Allowed:   allowed,
		Current:   current,
		Limit:     quota.Limit,
		Remaining: quota.Limit - current,
	}, nil
}

// IncrementUsage increases quota usage
func (s *Service) IncrementUsage(ctx context.Context, req IncrementUsageRequest) error {
	quota, err := s.repo.GetQuota(ctx, req.TeamID, req.Type)
	if err != nil {
		return nil // No quota = no tracking needed
	}

	if quota.Period != "" {
		// Rate-limited quota - use Redis with TTL
		return s.incrementRateLimitUsage(ctx, quota, req.Amount)
	}

	// Absolute quota - use DB
	return s.incrementAbsoluteUsage(ctx, quota, req.Amount)
}

func (s *Service) getRateLimitUsage(ctx context.Context, quota *Quota) (int64, error) {
	key := s.rateLimitKey(quota)
	return s.cache.IncrBy(ctx, key, 0) // Get without increment
}

func (s *Service) incrementRateLimitUsage(ctx context.Context, quota *Quota, amount int64) error {
	key := s.rateLimitKey(quota)

	// Increment
	_, err := s.cache.IncrBy(ctx, key, amount)
	if err != nil {
		return err
	}

	// Set TTL if not set
	ttl, _ := s.cache.TTL(ctx, key)
	if ttl < 0 {
		s.cache.Expire(ctx, key, s.periodToDuration(quota.Period))
	}

	return nil
}

func (s *Service) incrementAbsoluteUsage(ctx context.Context, quota *Quota, amount int64) error {
	// Use database transaction to increment
	// Implementation depends on DB
	return nil
}

func (s *Service) rateLimitKey(quota *Quota) string {
	// Include period start time in key for automatic reset
	periodStart := s.periodStart(quota.Period)
	return fmt.Sprintf("quota:%s:%s:%d", quota.ID, quota.Type, periodStart.Unix())
}

func (s *Service) periodStart(period string) time.Time {
	now := time.Now().UTC()
	switch period {
	case "hour":
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	case "day":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	case "month":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return now
	}
}

func (s *Service) periodToDuration(period string) time.Duration {
	switch period {
	case "hour":
		return time.Hour
	case "day":
		return 24 * time.Hour
	case "month":
		return 30 * 24 * time.Hour
	default:
		return time.Hour
	}
}

// GetUsageSummary returns quota usage for a team
func (s *Service) GetUsageSummary(ctx context.Context, teamID string) (*UsageSummary, error) {
	quotas, err := s.repo.GetQuotas(ctx, teamID)
	if err != nil {
		return nil, err
	}

	items := make([]UsageItem, len(quotas))
	for i, quota := range quotas {
		var current int64
		if quota.Period != "" {
			current, _ = s.getRateLimitUsage(ctx, &quota)
		} else {
			if usage, err := s.repo.GetUsage(ctx, quota.ID); err == nil {
				current = usage.Current
			}
		}

		items[i] = UsageItem{
			Type:      quota.Type,
			Current:   current,
			Limit:     quota.Limit,
			Period:    quota.Period,
			Remaining: quota.Limit - current,
		}

		if quota.Limit > 0 {
			items[i].Percentage = float64(current) / float64(quota.Limit) * 100
		}
	}

	return &UsageSummary{
		TeamID: teamID,
		Items:  items,
	}, nil
}

// Types

type CheckQuotaRequest struct {
	TeamID    string
	ProjectID string
	Type      QuotaType
	Amount    int64
}

type CheckQuotaResult struct {
	Allowed   bool  `json:"allowed"`
	Current   int64 `json:"current"`
	Limit     int64 `json:"limit"`
	Remaining int64 `json:"remaining"`
}

type IncrementUsageRequest struct {
	TeamID    string
	ProjectID string
	Type      QuotaType
	Amount    int64
}

type UsageSummary struct {
	TeamID string      `json:"team_id"`
	Items  []UsageItem `json:"items"`
}

type UsageItem struct {
	Type       QuotaType `json:"type"`
	Current    int64     `json:"current"`
	Limit      int64     `json:"limit"`
	Period     string    `json:"period,omitempty"`
	Remaining  int64     `json:"remaining"`
	Percentage float64   `json:"percentage"`
}
```

### 2.3 Quota Middleware

**`internal/api/middleware/quota.go`**:
```go
package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/raphaelmansuy/agentstack/internal/domain/quota"
)

type QuotaMiddleware struct {
	quotaSvc *quota.Service
}

func NewQuotaMiddleware(quotaSvc *quota.Service) *QuotaMiddleware {
	return &QuotaMiddleware{quotaSvc: quotaSvc}
}

// CheckQuota middleware verifies quota before allowing request
func (m *QuotaMiddleware) CheckQuota(quotaType quota.QuotaType, amount int64) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := GetAuthContext(c)
		if auth == nil {
			return c.Next() // Let auth middleware handle this
		}

		result, err := m.quotaSvc.CheckQuota(c.UserContext(), quota.CheckQuotaRequest{
			TeamID:    auth.TeamID,
			ProjectID: auth.ProjectID,
			Type:      quotaType,
			Amount:    amount,
		})

		if err != nil {
			// Log but don't fail
			return c.Next()
		}

		if !result.Allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":     "quota_exceeded",
				"message":   "Quota exceeded for " + string(quotaType),
				"current":   result.Current,
				"limit":     result.Limit,
				"remaining": result.Remaining,
			})
		}

		// Store for post-request increment
		c.Locals("quota_request", quota.IncrementUsageRequest{
			TeamID:    auth.TeamID,
			ProjectID: auth.ProjectID,
			Type:      quotaType,
			Amount:    amount,
		})

		return c.Next()
	}
}

// IncrementUsageAfter increments quota usage after successful request
func (m *QuotaMiddleware) IncrementUsageAfter() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Run after handler
		err := c.Next()

		// Only increment on success
		if c.Response().StatusCode() >= 200 && c.Response().StatusCode() < 300 {
			if req, ok := c.Locals("quota_request").(quota.IncrementUsageRequest); ok {
				go m.quotaSvc.IncrementUsage(c.UserContext(), req)
			}
		}

		return err
	}
}
```

---

## Week 3: Audit Logging

### 3.1 Audit Model

**`internal/domain/audit/model.go`**:
```go
package audit

import (
	"encoding/json"
	"time"
)

// EventType categorizes audit events
type EventType string

const (
	// Authentication events
	EventLogin            EventType = "auth.login"
	EventLogout           EventType = "auth.logout"
	EventAPIKeyCreated    EventType = "auth.api_key_created"
	EventAPIKeyRevoked    EventType = "auth.api_key_revoked"

	// Resource events
	EventAgentCreated     EventType = "agent.created"
	EventAgentUpdated     EventType = "agent.updated"
	EventAgentDeleted     EventType = "agent.deleted"
	EventAgentDeployed    EventType = "agent.deployed"
	EventAgentInvoked     EventType = "agent.invoked"

	EventProjectCreated   EventType = "project.created"
	EventProjectUpdated   EventType = "project.updated"
	EventProjectDeleted   EventType = "project.deleted"

	// Access events
	EventMemberInvited    EventType = "member.invited"
	EventMemberRemoved    EventType = "member.removed"
	EventRoleAssigned     EventType = "role.assigned"
	EventRoleRevoked      EventType = "role.revoked"
	EventPermissionDenied EventType = "permission.denied"

	// System events
	EventQuotaExceeded    EventType = "quota.exceeded"
	EventRateLimited      EventType = "rate.limited"
)

// Event represents an audit log entry
type Event struct {
	ID         string          `json:"id"`
	Type       EventType       `json:"type"`
	Timestamp  time.Time       `json:"timestamp"`
	TeamID     string          `json:"team_id"`
	ProjectID  string          `json:"project_id,omitempty"`
	ActorID    string          `json:"actor_id"`
	ActorType  string          `json:"actor_type"` // user, api_key, system
	ActorEmail string          `json:"actor_email,omitempty"`
	ResourceID string          `json:"resource_id,omitempty"`
	Resource   string          `json:"resource,omitempty"`
	Action     string          `json:"action"`
	Result     string          `json:"result"` // success, failure, denied
	IPAddress  string          `json:"ip_address,omitempty"`
	UserAgent  string          `json:"user_agent,omitempty"`
	RequestID  string          `json:"request_id,omitempty"`
	Details    json.RawMessage `json:"details,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// EventDetails holds additional event-specific information
type EventDetails struct {
	// For resource events
	Changes map[string]Change `json:"changes,omitempty"`

	// For auth events
	Method      string `json:"method,omitempty"`
	FailReason  string `json:"fail_reason,omitempty"`

	// For permission events
	Permission string `json:"permission,omitempty"`
	Scope      string `json:"scope,omitempty"`

	// For rate/quota events
	Limit   int64 `json:"limit,omitempty"`
	Current int64 `json:"current,omitempty"`
}

type Change struct {
	From interface{} `json:"from,omitempty"`
	To   interface{} `json:"to,omitempty"`
}
```

### 3.2 Audit Service

**`internal/domain/audit/service.go`**:
```go
package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/pkg/id"
)

type Service struct {
	writer Writer
}

type Writer interface {
	Write(ctx context.Context, event *Event) error
	Query(ctx context.Context, filter Filter) ([]Event, error)
}

func NewService(writer Writer) *Service {
	return &Service{writer: writer}
}

// Log records an audit event
func (s *Service) Log(ctx context.Context, event *Event) error {
	if event.ID == "" {
		event.ID = id.Generate("evt")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	return s.writer.Write(ctx, event)
}

// LogAction is a convenience method for common actions
func (s *Service) LogAction(ctx context.Context, params LogParams) error {
	details, _ := json.Marshal(params.Details)

	event := &Event{
		ID:         id.Generate("evt"),
		Type:       params.Type,
		Timestamp:  time.Now().UTC(),
		TeamID:     params.TeamID,
		ProjectID:  params.ProjectID,
		ActorID:    params.ActorID,
		ActorType:  params.ActorType,
		ActorEmail: params.ActorEmail,
		ResourceID: params.ResourceID,
		Resource:   params.Resource,
		Action:     params.Action,
		Result:     params.Result,
		IPAddress:  params.IPAddress,
		UserAgent:  params.UserAgent,
		RequestID:  params.RequestID,
		Details:    details,
	}

	return s.writer.Write(ctx, event)
}

// Query retrieves audit events
func (s *Service) Query(ctx context.Context, filter Filter) ([]Event, error) {
	return s.writer.Query(ctx, filter)
}

// Types

type LogParams struct {
	Type       EventType
	TeamID     string
	ProjectID  string
	ActorID    string
	ActorType  string
	ActorEmail string
	ResourceID string
	Resource   string
	Action     string
	Result     string
	IPAddress  string
	UserAgent  string
	RequestID  string
	Details    interface{}
}

type Filter struct {
	TeamID     string
	ProjectID  string
	ActorID    string
	EventTypes []EventType
	StartTime  time.Time
	EndTime    time.Time
	Resource   string
	ResourceID string
	Result     string
	Limit      int
	Offset     int
}
```

### 3.3 Audit Middleware

**`internal/api/middleware/audit.go`**:
```go
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
)

type AuditMiddleware struct {
	auditSvc *audit.Service
}

func NewAuditMiddleware(auditSvc *audit.Service) *AuditMiddleware {
	return &AuditMiddleware{auditSvc: auditSvc}
}

// RequestLogger logs all API requests for audit
func (m *AuditMiddleware) RequestLogger(c *fiber.Ctx) error {
	start := time.Now()

	// Generate request ID
	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
		c.Set("X-Request-ID", requestID)
	}
	c.Locals("request_id", requestID)

	// Execute handler
	err := c.Next()

	// Log after request
	go m.logRequest(c, start, requestID)

	return err
}

func (m *AuditMiddleware) logRequest(c *fiber.Ctx, start time.Time, requestID string) {
	auth := GetAuthContext(c)
	if auth == nil {
		return // Don't log unauthenticated requests
	}

	// Determine event type from path and method
	eventType := m.determineEventType(c.Method(), c.Path())
	if eventType == "" {
		return // Not an auditable action
	}

	result := "success"
	if c.Response().StatusCode() >= 400 {
		result = "failure"
	}
	if c.Response().StatusCode() == 403 {
		result = "denied"
	}

	m.auditSvc.LogAction(c.UserContext(), audit.LogParams{
		Type:       eventType,
		TeamID:     auth.TeamID,
		ProjectID:  auth.ProjectID,
		ActorID:    auth.UserID,
		ActorType:  auth.Type,
		ResourceID: c.Params("agentId", c.Params("projectId", "")),
		Resource:   m.extractResource(c.Path()),
		Action:     c.Method(),
		Result:     result,
		IPAddress:  c.IP(),
		UserAgent:  c.Get("User-Agent"),
		RequestID:  requestID,
		Details: map[string]interface{}{
			"path":     c.Path(),
			"status":   c.Response().StatusCode(),
			"duration": time.Since(start).Milliseconds(),
		},
	})
}

func (m *AuditMiddleware) determineEventType(method, path string) audit.EventType {
	// Map routes to event types
	// This is simplified - in production use a route registry
	switch {
	case method == "POST" && contains(path, "/agents"):
		return audit.EventAgentCreated
	case method == "PATCH" && contains(path, "/agents/"):
		return audit.EventAgentUpdated
	case method == "DELETE" && contains(path, "/agents/"):
		return audit.EventAgentDeleted
	case method == "POST" && contains(path, "/deploy"):
		return audit.EventAgentDeployed
	case method == "POST" && contains(path, "/chat"):
		return audit.EventAgentInvoked
	// Add more mappings...
	default:
		return ""
	}
}

func (m *AuditMiddleware) extractResource(path string) string {
	// Extract resource type from path
	// /v1/agents/xxx -> agent
	// /v1/projects/xxx -> project
	return ""
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(s) > len(substr) && (s[:len(substr)+1] == substr+"/" || s[len(s)-len(substr)-1:] == "/"+substr))
}
```

### 3.4 ClickHouse Audit Writer

**`internal/infrastructure/audit/clickhouse.go`**:
```go
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
)

type ClickHouseWriter struct {
	db *sql.DB
}

func NewClickHouseWriter(dsn string) (*ClickHouseWriter, error) {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, err
	}

	// Create table if not exists
	if err := createTable(db); err != nil {
		return nil, err
	}

	return &ClickHouseWriter{db: db}, nil
}

func createTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS audit_events (
			id String,
			type String,
			timestamp DateTime64(3, 'UTC'),
			team_id String,
			project_id String,
			actor_id String,
			actor_type String,
			actor_email String,
			resource_id String,
			resource String,
			action String,
			result String,
			ip_address String,
			user_agent String,
			request_id String,
			details String,
			metadata String
		) ENGINE = MergeTree()
		ORDER BY (team_id, timestamp)
		PARTITION BY toYYYYMM(timestamp)
		TTL toDateTime(timestamp) + INTERVAL 2 YEAR
	`
	_, err := db.Exec(query)
	return err
}

func (w *ClickHouseWriter) Write(ctx context.Context, event *audit.Event) error {
	query := `
		INSERT INTO audit_events (
			id, type, timestamp, team_id, project_id,
			actor_id, actor_type, actor_email,
			resource_id, resource, action, result,
			ip_address, user_agent, request_id,
			details, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := w.db.ExecContext(ctx, query,
		event.ID,
		string(event.Type),
		event.Timestamp,
		event.TeamID,
		event.ProjectID,
		event.ActorID,
		event.ActorType,
		event.ActorEmail,
		event.ResourceID,
		event.Resource,
		event.Action,
		event.Result,
		event.IPAddress,
		event.UserAgent,
		event.RequestID,
		string(event.Details),
		string(event.Metadata),
	)

	return err
}

func (w *ClickHouseWriter) Query(ctx context.Context, filter audit.Filter) ([]audit.Event, error) {
	query := `
		SELECT 
			id, type, timestamp, team_id, project_id,
			actor_id, actor_type, actor_email,
			resource_id, resource, action, result,
			ip_address, user_agent, request_id,
			details, metadata
		FROM audit_events
		WHERE team_id = ?
	`

	args := []interface{}{filter.TeamID}

	if filter.ProjectID != "" {
		query += " AND project_id = ?"
		args = append(args, filter.ProjectID)
	}

	if filter.ActorID != "" {
		query += " AND actor_id = ?"
		args = append(args, filter.ActorID)
	}

	if !filter.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.StartTime)
	}

	if !filter.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.EndTime)
	}

	if len(filter.EventTypes) > 0 {
		query += " AND type IN (?)"
		args = append(args, filter.EventTypes)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := w.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []audit.Event
	for rows.Next() {
		var e audit.Event
		var details, metadata string

		err := rows.Scan(
			&e.ID, &e.Type, &e.Timestamp, &e.TeamID, &e.ProjectID,
			&e.ActorID, &e.ActorType, &e.ActorEmail,
			&e.ResourceID, &e.Resource, &e.Action, &e.Result,
			&e.IPAddress, &e.UserAgent, &e.RequestID,
			&details, &metadata,
		)
		if err != nil {
			return nil, err
		}

		e.Details = json.RawMessage(details)
		e.Metadata = json.RawMessage(metadata)
		events = append(events, e)
	}

	return events, nil
}
```

---

## Deliverables Checklist

### Week 1 - RBAC
- [ ] Permission model (resources, actions)
- [ ] Role definitions (system roles)
- [ ] Role binding service
- [ ] Permission checking with caching
- [ ] RBAC middleware
- [ ] API endpoints for role management

### Week 2 - Quotas
- [ ] Quota model (types, plans)
- [ ] Rate limiting with Redis
- [ ] Absolute quota tracking
- [ ] Quota middleware
- [ ] Usage summary API
- [ ] Quota alerts

### Week 3 - Audit Logging
- [ ] Audit event model
- [ ] ClickHouse writer
- [ ] Audit middleware
- [ ] Query API with filters
- [ ] Retention policies
- [ ] Export functionality

---

## Definition of Done

- [ ] RBAC blocks unauthorized actions
- [ ] Quotas enforce limits correctly
- [ ] All mutations are audit logged
- [ ] Audit logs are queryable
- [ ] Permission cache invalidates correctly
- [ ] Rate limits use sliding window

---

## Sage AI Guidance

### Security Considerations

1. **Principle of Least Privilege**: Default to no access, explicitly grant.
2. **Cache Invalidation**: Always invalidate on role change.
3. **Audit Immutability**: Never modify or delete audit logs.
4. **Rate Limit by API Key**: Not just by IP.

### Performance Tips

| Component | Strategy |
|-----------|----------|
| RBAC | Cache permissions for 5 min |
| Quotas | Redis for rate limits, DB for absolute |
| Audit | Async write, batch inserts |
| Queries | Pre-aggregate common queries |

### Testing Checklist

```text
1. Test role hierarchy (owner > admin > developer > viewer)
2. Test quota enforcement at boundaries
3. Test audit log completeness
4. Test permission cache invalidation
5. Load test rate limiting (10k req/s)
```

---

**Next Phase**: [006-production-hardening.md](006-production-hardening.md) - Performance, Security, Multi-region
