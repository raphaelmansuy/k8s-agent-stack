// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// ListRolesOutput is the output for listing roles.
type ListRolesOutput struct {
	Body struct {
		Roles []rbac.Role `json:"roles" doc:"List of roles"`
	}
}

// GetRoleInput is the input for getting a role.
type GetRoleInput struct {
	ID string `path:"id" doc:"Role ID"`
}

// GetRoleOutput is the output for getting a role.
type GetRoleOutput struct {
	Body *rbac.Role
}

// CreateRoleInput is the input for creating a role.
type CreateRoleInput struct {
	Body struct {
		Name        string            `json:"name" required:"true" doc:"Role name"`
		Description string            `json:"description,omitempty" doc:"Role description"`
		Permissions []rbac.Permission `json:"permissions" required:"true" doc:"List of permissions"`
	}
}

// CreateRoleOutput is the output for creating a role.
type CreateRoleOutput struct {
	Body *rbac.Role
}

// UpdateRoleInput is the input for updating a role.
type UpdateRoleInput struct {
	ID   string `path:"id" doc:"Role ID"`
	Body struct {
		Name        string            `json:"name" required:"true" doc:"Role name"`
		Description string            `json:"description,omitempty" doc:"Role description"`
		Permissions []rbac.Permission `json:"permissions" required:"true" doc:"List of permissions"`
	}
}

// UpdateRoleOutput is the output for updating a role.
type UpdateRoleOutput struct {
	Body *rbac.Role
}

// DeleteRoleInput is the input for deleting a role.
type DeleteRoleInput struct {
	ID string `path:"id" doc:"Role ID"`
}

// AssignRoleInput is the input for assigning a role.
type AssignRoleInput struct {
	Body struct {
		UserID    string     `json:"user_id" required:"true" doc:"User ID"`
		RoleID    string     `json:"role_id" required:"true" doc:"Role ID"`
		Scope     rbac.Scope `json:"scope" required:"true" doc:"Scope"`
		ExpiresAt *time.Time `json:"expires_at,omitempty" doc:"Expiration timestamp"`
	}
}

// AssignRoleOutput is the output for assigning a role.
type AssignRoleOutput struct {
	Body *rbac.RoleBinding
}

// RevokeRoleInput is the input for revoking a role.
type RevokeRoleInput struct {
	ID string `path:"id" doc:"Binding ID"`
}

// GetUserRolesInput is the input for getting user roles.
type GetUserRolesInput struct {
	ID string `path:"id" doc:"User ID"`
}

// GetUserRolesOutput is the output for getting user roles.
type GetUserRolesOutput struct {
	Body struct {
		Bindings []rbac.RoleBinding `json:"bindings" doc:"List of role bindings"`
	}
}

// CheckPermissionInput is the input for checking permission.
type CheckPermissionInput struct {
	Body struct {
		UserID     string            `json:"user_id" required:"true" doc:"User ID"`
		Resource   rbac.Resource     `json:"resource" required:"true" doc:"Resource type"`
		Action     rbac.Action       `json:"action" required:"true" doc:"Action type"`
		ScopeType  rbac.ScopeType    `json:"scope_type" required:"true" doc:"Scope type"`
		ScopeID    string            `json:"scope_id" required:"true" doc:"Scope ID"`
		Conditions map[string]string `json:"conditions,omitempty" doc:"Custom conditions"`
	}
}

// CheckPermissionOutput is the output for checking permission.
type CheckPermissionOutput struct {
	Body *rbac.PermissionResult
}

// RegisterRBACRoutes registers RBAC routes.
func RegisterRBACRoutes(api huma.API, service *rbac.Service, rbacM *middleware.RBACMiddleware, auditM *middleware.AuditMiddleware) {
	// List roles
	huma.Register(api, huma.Operation{
		OperationID: "list-roles",
		Method:      http.MethodGet,
		Path:        "/v1/roles",
		Summary:     "List roles",
		Tags:        []string{"RBAC"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceMember, rbac.ActionList),
		},
	}, func(ctx context.Context, input *struct{}) (*ListRolesOutput, error) {
		auth := middleware.GetAuthFromContext(ctx)
		teamID := ""
		if auth != nil {
			teamID = auth.TeamID
		}

		roles, err := service.ListRoles(ctx, teamID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list roles", err)
		}

		return &ListRolesOutput{
			Body: struct {
				Roles []rbac.Role `json:"roles" doc:"List of roles"`
			}{
				Roles: roles,
			},
		}, nil
	})

	// Get role
	huma.Register(api, huma.Operation{
		OperationID: "get-role",
		Method:      http.MethodGet,
		Path:        "/v1/roles/{id}",
		Summary:     "Get role",
		Tags:        []string{"RBAC"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceMember, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetRoleInput) (*GetRoleOutput, error) {
		role := rbac.GetSystemRole(input.ID)
		if role == nil {
			return nil, huma.Error404NotFound("Role not found")
		}

		return &GetRoleOutput{
			Body: role,
		}, nil
	})

	// Create role
	huma.Register(api, huma.Operation{
		OperationID: "create-role",
		Method:      http.MethodPost,
		Path:        "/v1/roles",
		Summary:     "Create role",
		Tags:        []string{"RBAC"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceMember, rbac.ActionManage),
			auditM.HumaLogAction(audit.EventRoleCreated, string(rbac.ResourceMember)),
		},
	}, func(ctx context.Context, input *CreateRoleInput) (*CreateRoleOutput, error) {
		role := &rbac.Role{
			Name:        input.Body.Name,
			Description: input.Body.Description,
			Permissions: input.Body.Permissions,
		}

		if err := service.CreateRole(ctx, role); err != nil {
			return nil, huma.Error500InternalServerError("Failed to create role", err)
		}

		return &CreateRoleOutput{
			Body: role,
		}, nil
	})

	// Update role
	huma.Register(api, huma.Operation{
		OperationID: "update-role",
		Method:      http.MethodPut,
		Path:        "/v1/roles/{id}",
		Summary:     "Update role",
		Tags:        []string{"RBAC"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceMember, rbac.ActionManage),
		},
	}, func(ctx context.Context, input *UpdateRoleInput) (*UpdateRoleOutput, error) {
		role := &rbac.Role{
			ID:          input.ID,
			Name:        input.Body.Name,
			Description: input.Body.Description,
			Permissions: input.Body.Permissions,
		}

		if err := service.UpdateRole(ctx, role); err != nil {
			return nil, huma.Error400BadRequest("Failed to update role", err)
		}

		return &UpdateRoleOutput{
			Body: role,
		}, nil
	})

	// Delete role
	huma.Register(api, huma.Operation{
		OperationID: "delete-role",
		Method:      http.MethodDelete,
		Path:        "/v1/roles/{id}",
		Summary:     "Delete role",
		Tags:        []string{"RBAC"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceMember, rbac.ActionManage),
		},
	}, func(ctx context.Context, input *DeleteRoleInput) (*struct{}, error) {
		if err := service.DeleteRole(ctx, input.ID); err != nil {
			return nil, huma.Error400BadRequest("Failed to delete role", err)
		}

		return nil, nil
	})

	// Assign role
	huma.Register(api, huma.Operation{
		OperationID: "assign-role",
		Method:      http.MethodPost,
		Path:        "/v1/role-bindings",
		Summary:     "Assign role",
		Tags:        []string{"RBAC"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceMember, rbac.ActionManage),
		},
	}, func(ctx context.Context, input *AssignRoleInput) (*AssignRoleOutput, error) {
		auth := middleware.GetAuthFromContext(ctx)
		grantedBy := ""
		if auth != nil {
			grantedBy = auth.UserID
		}

		binding, err := service.AssignRole(ctx, rbac.AssignRoleRequest{
			UserID:    input.Body.UserID,
			RoleID:    input.Body.RoleID,
			Scope:     input.Body.Scope,
			GrantedBy: grantedBy,
			ExpiresAt: input.Body.ExpiresAt,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to assign role", err)
		}

		return &AssignRoleOutput{
			Body: binding,
		}, nil
	})

	// Revoke role
	huma.Register(api, huma.Operation{
		OperationID: "revoke-role",
		Method:      http.MethodDelete,
		Path:        "/v1/role-bindings/{id}",
		Summary:     "Revoke role",
		Tags:        []string{"RBAC"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceMember, rbac.ActionManage),
		},
	}, func(ctx context.Context, input *RevokeRoleInput) (*struct{}, error) {
		if err := service.RevokeRole(ctx, input.ID); err != nil {
			return nil, huma.Error500InternalServerError("Failed to revoke role", err)
		}

		return nil, nil
	})

	// Get user roles
	huma.Register(api, huma.Operation{
		OperationID: "get-user-roles",
		Method:      http.MethodGet,
		Path:        "/v1/users/{id}/roles",
		Summary:     "Get user roles",
		Tags:        []string{"RBAC"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceMember, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetUserRolesInput) (*GetUserRolesOutput, error) {
		bindings, err := service.GetUserRoles(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user roles", err)
		}

		return &GetUserRolesOutput{
			Body: struct {
				Bindings []rbac.RoleBinding `json:"bindings" doc:"List of role bindings"`
			}{
				Bindings: bindings,
			},
		}, nil
	})

	// Check permission
	huma.Register(api, huma.Operation{
		OperationID: "check-permission",
		Method:      http.MethodPost,
		Path:        "/v1/permissions/check",
		Summary:     "Check permission",
		Tags:        []string{"RBAC"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceMember, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *CheckPermissionInput) (*CheckPermissionOutput, error) {
		result, err := service.CheckPermission(ctx, rbac.PermissionRequest{
			UserID:     input.Body.UserID,
			Resource:   input.Body.Resource,
			Action:     input.Body.Action,
			ScopeType:  input.Body.ScopeType,
			ScopeID:    input.Body.ScopeID,
			Conditions: input.Body.Conditions,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to check permission", err)
		}

		return &CheckPermissionOutput{
			Body: result,
		}, nil
	})
}
