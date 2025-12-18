// Package rbac provides role-based access control functionality.
package rbac

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for RBAC data access.
type Repository interface {
	// Roles
	GetRole(ctx context.Context, id string) (*Role, error)
	ListRoles(ctx context.Context, teamID string) ([]Role, error)
	CreateRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id string) error

	// Role Bindings
	GetRoleBinding(ctx context.Context, id string) (*RoleBinding, error)
	GetRoleBindings(ctx context.Context, userID string) ([]RoleBinding, error)
	GetRoleBindingsByScope(ctx context.Context, scope Scope) ([]RoleBinding, error)
	CreateRoleBinding(ctx context.Context, rb *RoleBinding) error
	DeleteRoleBinding(ctx context.Context, id string) error
}

// Cache defines the interface for caching permissions.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
}

// Service provides RBAC functionality.
type Service struct {
	repo  Repository
	cache Cache
}

// NewService creates a new RBAC service.
func NewService(repo Repository, cache Cache) *Service {
	return &Service{
		repo:  repo,
		cache: cache,
	}
}

// CheckPermission verifies if a user has permission to perform an action.
func (s *Service) CheckPermission(ctx context.Context, req PermissionRequest) (*PermissionResult, error) {
	// Build cache key
	cacheKey := fmt.Sprintf("perm:%s:%s:%s:%s",
		req.UserID, req.Resource, req.Action, req.ScopeID)

	// Check cache first
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
			return &PermissionResult{Allowed: cached == "1"}, nil
		}
	}

	// Get user's role bindings
	bindings, err := s.repo.GetRoleBindings(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role bindings: %w", err)
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

		// Get role (try system roles first)
		role := GetSystemRole(binding.RoleID)
		if role == nil {
			role, err = s.repo.GetRole(ctx, binding.RoleID)
			if err != nil {
				continue
			}
		}

		// Check permissions
		if s.hasPermission(role, req.Resource, req.Action, req.Conditions) {
			allowed = true
			matchedRole = role
			break
		}
	}

	// Cache result (5 minutes)
	if s.cache != nil {
		cacheValue := "0"
		if allowed {
			cacheValue = "1"
		}
		_ = s.cache.Set(ctx, cacheKey, cacheValue, 5*time.Minute)
	}

	result := &PermissionResult{
		Allowed: allowed,
	}
	if matchedRole != nil {
		result.Role = matchedRole.Name
	}
	if !allowed {
		result.Reason = fmt.Sprintf("no permission for %s.%s", req.Resource, req.Action)
	}

	return result, nil
}

func (s *Service) scopeMatches(binding Scope, scopeType ScopeType, scopeID string) bool {
	// Global scope matches everything
	if binding.Type == ScopeGlobal {
		return true
	}

	// Team scope includes all projects in team
	if binding.Type == ScopeTeam && scopeType == ScopeProject {
		return true // Simplified - would need project->team lookup
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
	if context == nil {
		return false
	}
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

// AssignRole assigns a role to a user.
func (s *Service) AssignRole(ctx context.Context, req AssignRoleRequest) (*RoleBinding, error) {
	// Validate role exists
	role := GetSystemRole(req.RoleID)
	if role == nil {
		var err error
		role, err = s.repo.GetRole(ctx, req.RoleID)
		if err != nil {
			return nil, fmt.Errorf("role not found: %w", err)
		}
	}

	binding := &RoleBinding{
		ID:        "rb_" + uuid.New().String()[:8],
		UserID:    req.UserID,
		RoleID:    role.ID,
		Scope:     req.Scope,
		GrantedBy: req.GrantedBy,
		CreatedAt: time.Now(),
		ExpiresAt: req.ExpiresAt,
	}

	if err := s.repo.CreateRoleBinding(ctx, binding); err != nil {
		return nil, fmt.Errorf("failed to create role binding: %w", err)
	}

	// Invalidate cache
	s.invalidateUserPermissions(ctx, req.UserID)

	return binding, nil
}

// RevokeRole removes a role binding.
func (s *Service) RevokeRole(ctx context.Context, bindingID string) error {
	// Get binding to find user for cache invalidation
	binding, err := s.repo.GetRoleBinding(ctx, bindingID)
	if err != nil {
		return fmt.Errorf("role binding not found: %w", err)
	}

	if err := s.repo.DeleteRoleBinding(ctx, bindingID); err != nil {
		return fmt.Errorf("failed to delete role binding: %w", err)
	}

	// Invalidate cache
	s.invalidateUserPermissions(ctx, binding.UserID)

	return nil
}

// GetUserRoles returns all roles assigned to a user.
func (s *Service) GetUserRoles(ctx context.Context, userID string) ([]RoleBinding, error) {
	return s.repo.GetRoleBindings(ctx, userID)
}

// ListRoles returns all roles available for a team.
func (s *Service) ListRoles(ctx context.Context, teamID string) ([]Role, error) {
	// Start with system roles
	roles := make([]Role, len(SystemRoles))
	copy(roles, SystemRoles)

	// Add team-specific roles
	teamRoles, err := s.repo.ListRoles(ctx, teamID)
	if err != nil {
		return nil, err
	}

	return append(roles, teamRoles...), nil
}

// CreateRole creates a custom role for a team.
func (s *Service) CreateRole(ctx context.Context, role *Role) error {
	if role.ID == "" {
		role.ID = "role_" + uuid.New().String()[:8]
	}
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	role.IsSystem = false

	return s.repo.CreateRole(ctx, role)
}

// UpdateRole updates a custom role.
func (s *Service) UpdateRole(ctx context.Context, role *Role) error {
	// Cannot update system roles
	if GetSystemRole(role.ID) != nil {
		return fmt.Errorf("cannot update system role")
	}

	role.UpdatedAt = time.Now()
	return s.repo.UpdateRole(ctx, role)
}

// DeleteRole deletes a custom role.
func (s *Service) DeleteRole(ctx context.Context, roleID string) error {
	// Cannot delete system roles
	if GetSystemRole(roleID) != nil {
		return fmt.Errorf("cannot delete system role")
	}

	return s.repo.DeleteRole(ctx, roleID)
}

func (s *Service) invalidateUserPermissions(ctx context.Context, userID string) {
	if s.cache == nil {
		return
	}
	// Delete all permission cache entries for this user
	pattern := fmt.Sprintf("perm:%s:*", userID)
	keys, _ := s.cache.Keys(ctx, pattern)
	if len(keys) > 0 {
		_ = s.cache.Del(ctx, keys...)
	}
}
