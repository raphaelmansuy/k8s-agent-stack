// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// RoleHandler handles RBAC-related HTTP requests.
type RoleHandler struct {
	rbacSvc *rbac.Service
}

// NewRoleHandler creates a new role handler.
func NewRoleHandler(rbacSvc *rbac.Service) *RoleHandler {
	return &RoleHandler{rbacSvc: rbacSvc}
}

// ListRoles lists all roles.
// GET /api/v1/roles
func (h *RoleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	// Get team ID from auth context
	auth := middleware.GetAuthFromContext(r.Context())
	teamID := ""
	if auth != nil {
		teamID = auth.TeamID
	}

	roles, err := h.rbacSvc.ListRoles(r.Context(), teamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list roles", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"roles": roles,
	})
}

// GetRole returns a specific role.
// GET /api/v1/roles/{id}
func (h *RoleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "role ID is required", nil)
		return
	}

	// First check system roles
	role := rbac.GetSystemRole(id)
	if role == nil {
		writeError(w, http.StatusNotFound, "role not found", nil)
		return
	}

	writeJSON(w, http.StatusOK, role)
}

// CreateRoleRequest represents a request to create a role.
type CreateRoleRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Permissions []rbac.Permission `json:"permissions"`
}

// CreateRole creates a new custom role.
// POST /api/v1/roles
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required", nil)
		return
	}

	role := &rbac.Role{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	}

	if err := h.rbacSvc.CreateRole(r.Context(), role); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create role", err)
		return
	}

	writeJSON(w, http.StatusCreated, role)
}

// UpdateRoleRequest represents a request to update a role.
type UpdateRoleRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Permissions []rbac.Permission `json:"permissions"`
}

// UpdateRole updates an existing role.
// PUT /api/v1/roles/{id}
func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "role ID is required", nil)
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	role := &rbac.Role{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	}

	if err := h.rbacSvc.UpdateRole(r.Context(), role); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	writeJSON(w, http.StatusOK, role)
}

// DeleteRole deletes a role.
// DELETE /api/v1/roles/{id}
func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "role ID is required", nil)
		return
	}

	if err := h.rbacSvc.DeleteRole(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AssignRoleRequest represents a request to assign a role.
type AssignRoleRequest struct {
	UserID    string     `json:"user_id"`
	RoleID    string     `json:"role_id"`
	Scope     rbac.Scope `json:"scope"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// AssignRole assigns a role to a user.
// POST /api/v1/role-bindings
func (h *RoleHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.UserID == "" || req.RoleID == "" {
		writeError(w, http.StatusBadRequest, "user_id and role_id are required", nil)
		return
	}

	// Get grantor from auth context
	auth := middleware.GetAuthFromContext(r.Context())
	grantedBy := ""
	if auth != nil {
		grantedBy = auth.UserID
	}

	binding, err := h.rbacSvc.AssignRole(r.Context(), rbac.AssignRoleRequest{
		UserID:    req.UserID,
		RoleID:    req.RoleID,
		Scope:     req.Scope,
		GrantedBy: grantedBy,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to assign role", err)
		return
	}

	writeJSON(w, http.StatusCreated, binding)
}

// RevokeRole revokes a role from a user.
// DELETE /api/v1/role-bindings/{id}
func (h *RoleHandler) RevokeRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "binding ID is required", nil)
		return
	}

	if err := h.rbacSvc.RevokeRole(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke role", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUserRoles returns roles assigned to a user.
// GET /api/v1/users/{id}/roles
func (h *RoleHandler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user ID is required", nil)
		return
	}

	bindings, err := h.rbacSvc.GetUserRoles(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user roles", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"bindings": bindings,
	})
}

// CheckPermissionRequest represents a permission check request.
type CheckPermissionRequest struct {
	UserID     string            `json:"user_id"`
	Resource   rbac.Resource     `json:"resource"`
	Action     rbac.Action       `json:"action"`
	ScopeType  rbac.ScopeType    `json:"scope_type"`
	ScopeID    string            `json:"scope_id"`
	Conditions map[string]string `json:"conditions"`
}

// CheckPermission checks if a user has a permission.
// POST /api/v1/permissions/check
func (h *RoleHandler) CheckPermission(w http.ResponseWriter, r *http.Request) {
	var req CheckPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	result, err := h.rbacSvc.CheckPermission(r.Context(), rbac.PermissionRequest{
		UserID:     req.UserID,
		Resource:   req.Resource,
		Action:     req.Action,
		ScopeType:  req.ScopeType,
		ScopeID:    req.ScopeID,
		Conditions: req.Conditions,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check permission", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
