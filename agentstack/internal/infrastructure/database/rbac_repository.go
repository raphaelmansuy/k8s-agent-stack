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

package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
)

type RBACRepository struct {
	queries *db.Queries
}

func NewRBACRepository(queries *db.Queries) *RBACRepository {
	return &RBACRepository{
		queries: queries,
	}
}

func (r *RBACRepository) GetRole(ctx context.Context, id string) (*rbac.Role, error) {
	row, err := r.queries.GetRole(ctx, id)
	if err != nil {
		return nil, err
	}

	return &rbac.Role{
		ID:          row.ID,
		TeamID:      row.TeamID.String,
		Name:        row.Name,
		Description: row.Description.String,
		Permissions: r.mapPermissions(row.Permissions),
		IsSystem:    row.IsSystem,
	}, nil
}

func (r *RBACRepository) ListRoles(ctx context.Context, teamID string) ([]rbac.Role, error) {
	rows, err := r.queries.ListRoles(ctx, pgtype.Text{String: teamID, Valid: teamID != ""})
	if err != nil {
		return nil, err
	}

	roles := make([]rbac.Role, len(rows))
	for i, row := range rows {
		roles[i] = rbac.Role{
			ID:          row.ID,
			TeamID:      row.TeamID.String,
			Name:        row.Name,
			Description: row.Description.String,
			Permissions: r.mapPermissions(row.Permissions),
			IsSystem:    row.IsSystem,
		}
	}
	return roles, nil
}

func (r *RBACRepository) CreateRole(ctx context.Context, role *rbac.Role) error {
	_, err := r.queries.CreateRole(ctx, db.CreateRoleParams{
		TeamID:      pgtype.Text{String: role.TeamID, Valid: role.TeamID != ""},
		Name:        role.Name,
		Description: pgtype.Text{String: role.Description, Valid: role.Description != ""},
		Permissions: r.mapPermissionsToDB(role.Permissions),
		IsSystem:    role.IsSystem,
	})
	return err
}

func (r *RBACRepository) UpdateRole(ctx context.Context, role *rbac.Role) error {
	_, err := r.queries.UpdateRole(ctx, db.UpdateRoleParams{
		ID:          role.ID,
		Name:        role.Name,
		Description: pgtype.Text{String: role.Description, Valid: role.Description != ""},
		Permissions: r.mapPermissionsToDB(role.Permissions),
	})
	return err
}

func (r *RBACRepository) DeleteRole(ctx context.Context, id string) error {
	return r.queries.DeleteRole(ctx, id)
}

func (r *RBACRepository) GetRoleBinding(ctx context.Context, id string) (*rbac.RoleBinding, error) {
	row, err := r.queries.GetRoleBinding(ctx, id)
	if err != nil {
		return nil, err
	}

	return &rbac.RoleBinding{
		ID:        row.ID,
		UserID:    row.UserID,
		RoleID:    row.RoleID,
		Scope:     r.mapScope(row.ScopeType, row.ScopeTeamID, row.ScopeProjectID),
		ExpiresAt: r.mapTime(row.ExpiresAt),
	}, nil
}

func (r *RBACRepository) GetRoleBindings(ctx context.Context, userID string) ([]rbac.RoleBinding, error) {
	rows, err := r.queries.ListRoleBindingsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	bindings := make([]rbac.RoleBinding, len(rows))
	for i, row := range rows {
		bindings[i] = rbac.RoleBinding{
			ID:        row.ID,
			UserID:    row.UserID,
			RoleID:    row.RoleID,
			Scope:     r.mapScope(row.ScopeType, row.ScopeTeamID, row.ScopeProjectID),
			ExpiresAt: r.mapTime(row.ExpiresAt),
		}
	}
	return bindings, nil
}

func (r *RBACRepository) GetRoleBindingsByScope(ctx context.Context, scope rbac.Scope) ([]rbac.RoleBinding, error) {
	rows, err := r.queries.ListRoleBindingsByScope(ctx, db.ListRoleBindingsByScopeParams{
		ScopeType:      string(scope.Type),
		ScopeTeamID:    pgtype.Text{String: scope.TeamID, Valid: scope.TeamID != ""},
		ScopeProjectID: pgtype.Text{String: scope.ProjectID, Valid: scope.ProjectID != ""},
	})
	if err != nil {
		return nil, err
	}

	bindings := make([]rbac.RoleBinding, len(rows))
	for i, row := range rows {
		bindings[i] = rbac.RoleBinding{
			ID:        row.ID,
			UserID:    row.UserID,
			RoleID:    row.RoleID,
			Scope:     r.mapScope(row.ScopeType, row.ScopeTeamID, row.ScopeProjectID),
			ExpiresAt: r.mapTime(row.ExpiresAt),
		}
	}
	return bindings, nil
}

func (r *RBACRepository) CreateRoleBinding(ctx context.Context, rb *rbac.RoleBinding) error {
	_, err := r.queries.CreateRoleBinding(ctx, db.CreateRoleBindingParams{
		UserID:         rb.UserID,
		RoleID:         rb.RoleID,
		ScopeType:      string(rb.Scope.Type),
		ScopeTeamID:    pgtype.Text{String: rb.Scope.TeamID, Valid: rb.Scope.TeamID != ""},
		ScopeProjectID: pgtype.Text{String: rb.Scope.ProjectID, Valid: rb.Scope.ProjectID != ""},
		GrantedBy:      pgtype.Text{String: rb.GrantedBy, Valid: rb.GrantedBy != ""},
		ExpiresAt:      pgtype.Timestamptz{Time: r.mapTimeToDB(rb.ExpiresAt), Valid: rb.ExpiresAt != nil},
	})
	return err
}

func (r *RBACRepository) DeleteRoleBinding(ctx context.Context, id string) error {
	return r.queries.DeleteRoleBinding(ctx, id)
}

// Helpers

func (r *RBACRepository) mapPermissions(dbPerms []byte) []rbac.Permission {
	if len(dbPerms) == 0 {
		return nil
	}
	var perms []rbac.Permission
	if err := json.Unmarshal(dbPerms, &perms); err != nil {
		return nil
	}
	return perms
}

func (r *RBACRepository) mapPermissionsToDB(perms []rbac.Permission) []byte {
	if perms == nil {
		return []byte("[]")
	}
	data, err := json.Marshal(perms)
	if err != nil {
		return []byte("[]")
	}
	return data
}

func (r *RBACRepository) mapScope(sType string, teamID, projectID pgtype.Text) rbac.Scope {
	return rbac.Scope{
		Type:      rbac.ScopeType(sType),
		TeamID:    teamID.String,
		ProjectID: projectID.String,
	}
}

func (r *RBACRepository) mapTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func (r *RBACRepository) mapTimeToDB(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
