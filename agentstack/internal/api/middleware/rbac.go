// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// RBACMiddleware provides RBAC checking middleware.
type RBACMiddleware struct {
	api     huma.API
	rbacSvc *rbac.Service
}

// NewRBACMiddleware creates a new RBAC middleware.
func NewRBACMiddleware(api huma.API, rbacSvc *rbac.Service) *RBACMiddleware {
	return &RBACMiddleware{api: api, rbacSvc: rbacSvc}
}

// RequirePermission creates middleware that checks for a specific permission.
func (m *RBACMiddleware) RequirePermission(resource rbac.Resource, action rbac.Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := GetAuthFromContext(r.Context())
			if auth == nil {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			// Determine scope from request
			scopeType, scopeID := m.extractScope(r)

			// Build conditions from request context
			conditions := map[string]string{
				"user_id": auth.UserID,
			}

			// Add resource owner if available
			if ownerID, ok := r.Context().Value(ContextKeyResourceOwner).(string); ok {
				conditions["owner_id"] = ownerID
			}
			if auth.TeamID != "" {
				conditions["team_id"] = auth.TeamID
			}
			if auth.ProjectID != "" {
				conditions["project_id"] = auth.ProjectID
			}

			result, err := m.rbacSvc.CheckPermission(r.Context(), rbac.PermissionRequest{
				UserID:     auth.UserID,
				Resource:   resource,
				Action:     action,
				ScopeType:  scopeType,
				ScopeID:    scopeID,
				Conditions: conditions,
			})

			if err != nil {
				http.Error(w, "permission check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				http.Error(w, "insufficient permissions", http.StatusForbidden)
				return
			}

			// Store result for audit logging
			ctx := SetRBACResultInContext(r.Context(), result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// HumaRequirePermission creates a Huma-compatible middleware that checks for a specific permission.
func (m *RBACMiddleware) HumaRequirePermission(resource rbac.Resource, action rbac.Action) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		auth := GetAuthFromContext(ctx.Context())
		if auth == nil {
			huma.WriteErr(m.api, ctx, http.StatusUnauthorized, "authentication required")
			return
		}

		// For Huma, we use global scope for now as extracting from path is complex without the router context
		scopeType := rbac.ScopeGlobal
		scopeID := ""

		conditions := map[string]string{
			"user_id": auth.UserID,
		}

		if auth.TeamID != "" {
			conditions["team_id"] = auth.TeamID
		}
		if auth.ProjectID != "" {
			conditions["project_id"] = auth.ProjectID
		}

		result, err := m.rbacSvc.CheckPermission(ctx.Context(), rbac.PermissionRequest{
			UserID:     auth.UserID,
			Resource:   resource,
			Action:     action,
			ScopeType:  scopeType,
			ScopeID:    scopeID,
			Conditions: conditions,
		})

		if err != nil {
			huma.WriteErr(m.api, ctx, http.StatusInternalServerError, "permission check failed", err)
			return
		}

		if !result.Allowed {
			huma.WriteErr(m.api, ctx, http.StatusForbidden, "insufficient permissions")
			return
		}

		next(ctx)
	}
}

// RequireRole creates middleware that requires a specific role.
func (m *RBACMiddleware) RequireRole(roleIDs ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := GetAuthFromContext(r.Context())
			if auth == nil {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			bindings, err := m.rbacSvc.GetUserRoles(r.Context(), auth.UserID)
			if err != nil {
				http.Error(w, "failed to get user roles", http.StatusInternalServerError)
				return
			}

			hasRole := false
			for _, binding := range bindings {
				for _, roleID := range roleIDs {
					if binding.RoleID == roleID {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				http.Error(w, "required role not found", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *RBACMiddleware) extractScope(r *http.Request) (rbac.ScopeType, string) {
	// Try to extract project ID from path or query
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		projectID = r.PathValue("projectId")
	}
	if projectID != "" {
		return rbac.ScopeProject, projectID
	}

	// Fall back to team scope
	auth := GetAuthFromContext(r.Context())
	if auth != nil && auth.TeamID != "" {
		return rbac.ScopeTeam, auth.TeamID
	}

	return rbac.ScopeGlobal, ""
}
