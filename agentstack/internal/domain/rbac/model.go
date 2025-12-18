// Package rbac provides role-based access control functionality.
package rbac

import "time"

// Resource represents a protected resource type.
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

// Action represents an operation on a resource.
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

// Role defines a set of permissions.
type Role struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
	IsSystem    bool         `json:"is_system"`
	TeamID      string       `json:"team_id,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Permission grants access to perform an action on a resource.
type Permission struct {
	Resource   Resource `json:"resource"`
	Action     Action   `json:"action"`
	Conditions []string `json:"conditions,omitempty"` // e.g., "own", "team", "project"
}

// RoleBinding assigns a role to a user for a scope.
type RoleBinding struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	RoleID    string     `json:"role_id"`
	Scope     Scope      `json:"scope"`
	GrantedBy string     `json:"granted_by"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Scope defines where a role applies.
type Scope struct {
	Type      ScopeType `json:"type"`
	TeamID    string    `json:"team_id,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
}

// ScopeType defines the scope level.
type ScopeType string

const (
	ScopeGlobal  ScopeType = "global"
	ScopeTeam    ScopeType = "team"
	ScopeProject ScopeType = "project"
)

// PermissionRequest represents a permission check request.
type PermissionRequest struct {
	UserID     string
	Resource   Resource
	Action     Action
	ScopeType  ScopeType
	ScopeID    string
	Conditions map[string]string
}

// PermissionResult contains the result of a permission check.
type PermissionResult struct {
	Allowed bool   `json:"allowed"`
	Role    string `json:"role,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// AssignRoleRequest represents a request to assign a role.
type AssignRoleRequest struct {
	UserID    string
	RoleID    string
	Scope     Scope
	GrantedBy string
	ExpiresAt *time.Time
}

// SystemRoles are the pre-defined roles available to all teams.
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

// GetSystemRole returns a system role by ID.
func GetSystemRole(id string) *Role {
	for _, r := range SystemRoles {
		if r.ID == id {
			role := r
			return &role
		}
	}
	return nil
}

// HasPermission checks if a role has a specific permission.
func (r *Role) HasPermission(resource Resource, action Action) bool {
	for _, p := range r.Permissions {
		if p.Resource == resource {
			if p.Action == ActionManage || p.Action == action {
				return true
			}
		}
	}
	return false
}
