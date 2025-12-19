/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rbac

import (
	"context"
	"testing"
	"time"
)

// mockRepository implements Repository for testing.
type mockRepository struct {
	roles    map[string]*Role
	bindings map[string]*RoleBinding
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		roles:    make(map[string]*Role),
		bindings: make(map[string]*RoleBinding),
	}
}

func (m *mockRepository) GetRole(ctx context.Context, id string) (*Role, error) {
	if r, ok := m.roles[id]; ok {
		return r, nil
	}
	return nil, context.DeadlineExceeded // Simulate not found
}

func (m *mockRepository) ListRoles(ctx context.Context, teamID string) ([]Role, error) {
	var roles []Role
	for _, r := range m.roles {
		if r.TeamID == teamID {
			roles = append(roles, *r)
		}
	}
	return roles, nil
}

func (m *mockRepository) CreateRole(ctx context.Context, role *Role) error {
	m.roles[role.ID] = role
	return nil
}

func (m *mockRepository) UpdateRole(ctx context.Context, role *Role) error {
	m.roles[role.ID] = role
	return nil
}

func (m *mockRepository) DeleteRole(ctx context.Context, id string) error {
	delete(m.roles, id)
	return nil
}

func (m *mockRepository) GetRoleBinding(ctx context.Context, id string) (*RoleBinding, error) {
	if b, ok := m.bindings[id]; ok {
		return b, nil
	}
	return nil, context.DeadlineExceeded
}

func (m *mockRepository) GetRoleBindings(ctx context.Context, userID string) ([]RoleBinding, error) {
	var bindings []RoleBinding
	for _, b := range m.bindings {
		if b.UserID == userID {
			bindings = append(bindings, *b)
		}
	}
	return bindings, nil
}

func (m *mockRepository) GetRoleBindingsByScope(ctx context.Context, scope Scope) ([]RoleBinding, error) {
	var bindings []RoleBinding
	for _, b := range m.bindings {
		if b.Scope.Type == scope.Type {
			bindings = append(bindings, *b)
		}
	}
	return bindings, nil
}

func (m *mockRepository) CreateRoleBinding(ctx context.Context, rb *RoleBinding) error {
	m.bindings[rb.ID] = rb
	return nil
}

func (m *mockRepository) DeleteRoleBinding(ctx context.Context, id string) error {
	delete(m.bindings, id)
	return nil
}

func TestCheckPermission(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	// Assign owner role to user
	repo.bindings["rb1"] = &RoleBinding{
		ID:     "rb1",
		UserID: "user1",
		RoleID: "role_owner",
		Scope:  Scope{Type: ScopeTeam, TeamID: "team1"},
	}

	tests := []struct {
		name string
		req  PermissionRequest
		want bool
	}{
		{
			name: "owner can manage agents",
			req: PermissionRequest{
				UserID:    "user1",
				Resource:  ResourceAgent,
				Action:    ActionManage,
				ScopeType: ScopeTeam,
				ScopeID:   "team1",
			},
			want: true,
		},
		{
			name: "owner can create agents",
			req: PermissionRequest{
				UserID:    "user1",
				Resource:  ResourceAgent,
				Action:    ActionCreate,
				ScopeType: ScopeTeam,
				ScopeID:   "team1",
			},
			want: true,
		},
		{
			name: "unknown user has no permissions",
			req: PermissionRequest{
				UserID:    "user2",
				Resource:  ResourceAgent,
				Action:    ActionCreate,
				ScopeType: ScopeTeam,
				ScopeID:   "team1",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.CheckPermission(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Allowed != tt.want {
				t.Errorf("got allowed=%v, want %v", result.Allowed, tt.want)
			}
		})
	}
}

func TestAssignRole(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	binding, err := svc.AssignRole(context.Background(), AssignRoleRequest{
		UserID:    "user1",
		RoleID:    "role_developer",
		Scope:     Scope{Type: ScopeTeam, TeamID: "team1"},
		GrantedBy: "admin1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if binding.UserID != "user1" {
		t.Errorf("got UserID=%s, want user1", binding.UserID)
	}
	if binding.RoleID != "role_developer" {
		t.Errorf("got RoleID=%s, want role_developer", binding.RoleID)
	}
}

func TestRevokeRole(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	// Create a binding first
	repo.bindings["rb1"] = &RoleBinding{
		ID:     "rb1",
		UserID: "user1",
		RoleID: "role_developer",
	}

	err := svc.RevokeRole(context.Background(), "rb1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify binding is deleted
	if _, exists := repo.bindings["rb1"]; exists {
		t.Error("binding should have been deleted")
	}
}

func TestExpiredBinding(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	expired := time.Now().Add(-time.Hour)
	repo.bindings["rb1"] = &RoleBinding{
		ID:        "rb1",
		UserID:    "user1",
		RoleID:    "role_owner",
		Scope:     Scope{Type: ScopeTeam, TeamID: "team1"},
		ExpiresAt: &expired,
	}

	result, err := svc.CheckPermission(context.Background(), PermissionRequest{
		UserID:    "user1",
		Resource:  ResourceAgent,
		Action:    ActionManage,
		ScopeType: ScopeTeam,
		ScopeID:   "team1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expired binding should not grant permissions")
	}
}

func TestSystemRoleImmutability(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	// Try to update system role
	err := svc.UpdateRole(context.Background(), &Role{ID: "role_owner", Name: "Hacked"})
	if err == nil {
		t.Error("should not be able to update system role")
	}

	// Try to delete system role
	err = svc.DeleteRole(context.Background(), "role_owner")
	if err == nil {
		t.Error("should not be able to delete system role")
	}
}

func TestHasPermission(t *testing.T) {
	owner := GetSystemRole("role_owner")
	if owner == nil {
		t.Fatal("owner role should exist")
	}

	if !owner.HasPermission(ResourceAgent, ActionManage) {
		t.Error("owner should have manage permission on agents")
	}
	if !owner.HasPermission(ResourceAgent, ActionCreate) {
		t.Error("owner should have create permission on agents (via manage)")
	}

	viewer := GetSystemRole("role_viewer")
	if viewer == nil {
		t.Fatal("viewer role should exist")
	}

	if viewer.HasPermission(ResourceAgent, ActionCreate) {
		t.Error("viewer should not have create permission on agents")
	}
	if !viewer.HasPermission(ResourceAgent, ActionRead) {
		t.Error("viewer should have read permission on agents")
	}
}
