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

// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
)

// Project represents a project that contains agents.
type Project struct {
	ID          string            `json:"id" doc:"Unique project identifier"`
	TeamID      string            `json:"team_id" doc:"Owning team ID"`
	Name        string            `json:"name" doc:"Project name"`
	Description string            `json:"description,omitempty" doc:"Project description"`
	Slug        string            `json:"slug" doc:"URL-friendly identifier"`
	Settings    map[string]any    `json:"settings,omitempty" doc:"Project settings"`
	Metadata    map[string]string `json:"metadata,omitempty" doc:"Custom metadata"`
	CreatedAt   time.Time         `json:"created_at" doc:"Creation timestamp"`
	UpdatedAt   time.Time         `json:"updated_at" doc:"Last update timestamp"`
}

// ListProjectsInput is the input for listing projects.
type ListProjectsInput struct {
	Limit  int `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Maximum results"`
	Offset int `query:"offset" default:"0" minimum:"0" doc:"Pagination offset"`
}

// ListProjectsOutput is the output for listing projects.
type ListProjectsOutput struct {
	Body struct {
		Projects []Project `json:"projects" doc:"List of projects"`
		Total    int       `json:"total" doc:"Total count"`
	}
}

// GetProjectInput is the input for getting a project.
type GetProjectInput struct {
	ID string `path:"id" doc:"Project ID"`
}

// GetProjectOutput is the output for getting a project.
type GetProjectOutput struct {
	Body Project
}

// CreateProjectInput is the input for creating a project.
type CreateProjectInput struct {
	Body struct {
		Name        string            `json:"name" required:"true" minLength:"1" maxLength:"255" doc:"Project name"`
		Description string            `json:"description,omitempty" maxLength:"1000" doc:"Project description"`
		Slug        string            `json:"slug" required:"true" pattern:"^[a-z0-9-]+$" doc:"URL-friendly identifier"`
		Settings    map[string]any    `json:"settings,omitempty" doc:"Project settings"`
		Metadata    map[string]string `json:"metadata,omitempty" doc:"Custom metadata"`
	}
}

// CreateProjectOutput is the output for creating a project.
type CreateProjectOutput struct {
	Body Project
}

// UpdateProjectInput is the input for updating a project.
type UpdateProjectInput struct {
	ID   string `path:"id" doc:"Project ID"`
	Body struct {
		Name        *string           `json:"name,omitempty" minLength:"1" maxLength:"255" doc:"Project name"`
		Description *string           `json:"description,omitempty" maxLength:"1000" doc:"Project description"`
		Settings    map[string]any    `json:"settings,omitempty" doc:"Project settings"`
		Metadata    map[string]string `json:"metadata,omitempty" doc:"Custom metadata"`
	}
}

// UpdateProjectOutput is the output for updating a project.
type UpdateProjectOutput struct {
	Body Project
}

// DeleteProjectInput is the input for deleting a project.
type DeleteProjectInput struct {
	ID string `path:"id" doc:"Project ID"`
}

// DeleteProjectOutput is the output for deleting a project.
type DeleteProjectOutput struct {
	Body struct {
		Message string `json:"message" doc:"Confirmation message"`
	}
}

// RegisterProjectRoutes registers project routes.
func RegisterProjectRoutes(api huma.API, pool *database.Pool, rbacM *middleware.RBACMiddleware, auditM *middleware.AuditMiddleware) {
	queries := db.New(pool.Pool)

	// List projects
	huma.Register(api, huma.Operation{
		OperationID: "list-projects",
		Method:      http.MethodGet,
		Path:        "/v1/projects",
		Summary:     "List projects",
		Tags:        []string{"Projects"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceProject, rbac.ActionList),
		},
	}, func(ctx context.Context, input *ListProjectsInput) (*ListProjectsOutput, error) {
		auth := middleware.GetAuthFromContext(ctx)
		teamID := "default-team"
		if auth != nil {
			teamID = auth.TeamID
		}

		projects, err := queries.ListProjects(ctx, db.ListProjectsParams{
			TeamID: teamID,
			Limit:  int32(input.Limit),  //nolint:gosec // Pagination limit is safe
			Offset: int32(input.Offset), //nolint:gosec // Pagination offset is safe
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list projects", err)
		}

		res := make([]Project, len(projects))
		for i, p := range projects {
			var settings map[string]any
			if len(p.Settings) > 0 {
				_ = json.Unmarshal(p.Settings, &settings)
			}
			res[i] = Project{
				ID:        p.ID,
				TeamID:    p.TeamID,
				Name:      p.Name,
				Slug:      p.Slug,
				Settings:  settings,
				CreatedAt: p.CreatedAt,
				UpdatedAt: p.UpdatedAt,
			}
		}

		return &ListProjectsOutput{
			Body: struct {
				Projects []Project `json:"projects" doc:"List of projects"`
				Total    int       `json:"total" doc:"Total count"`
			}{
				Projects: res,
				Total:    len(res),
			},
		}, nil
	})

	// Get project by ID
	huma.Register(api, huma.Operation{
		OperationID: "get-project",
		Method:      http.MethodGet,
		Path:        "/v1/projects/{id}",
		Summary:     "Get project",
		Tags:        []string{"Projects"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceProject, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetProjectInput) (*GetProjectOutput, error) {
		p, err := queries.GetProject(ctx, input.ID)
		if err != nil {
			return nil, huma.Error404NotFound("Project not found")
		}

		var settings map[string]any
		if len(p.Settings) > 0 {
			_ = json.Unmarshal(p.Settings, &settings)
		}

		return &GetProjectOutput{
			Body: Project{
				ID:        p.ID,
				TeamID:    p.TeamID,
				Name:      p.Name,
				Slug:      p.Slug,
				Settings:  settings,
				CreatedAt: p.CreatedAt,
				UpdatedAt: p.UpdatedAt,
			},
		}, nil
	})

	// Create project
	huma.Register(api, huma.Operation{
		OperationID: "create-project",
		Method:      http.MethodPost,
		Path:        "/v1/projects",
		Summary:     "Create project",
		Tags:        []string{"Projects"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceProject, rbac.ActionCreate),
			auditM.HumaLogAction(audit.EventProjectCreated, string(rbac.ResourceProject)),
		},
	}, func(ctx context.Context, input *CreateProjectInput) (*CreateProjectOutput, error) {
		auth := middleware.GetAuthFromContext(ctx)
		teamID := "default-team"
		if auth != nil {
			teamID = auth.TeamID
		}

		settings, _ := json.Marshal(input.Body.Settings)

		p, err := queries.CreateProject(ctx, db.CreateProjectParams{
			TeamID:   teamID,
			Name:     input.Body.Name,
			Slug:     input.Body.Slug,
			Settings: settings,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create project", err)
		}

		var resSettings map[string]any
		if len(p.Settings) > 0 {
			_ = json.Unmarshal(p.Settings, &resSettings)
		}

		return &CreateProjectOutput{
			Body: Project{
				ID:        p.ID,
				TeamID:    p.TeamID,
				Name:      p.Name,
				Slug:      p.Slug,
				Settings:  resSettings,
				CreatedAt: p.CreatedAt,
				UpdatedAt: p.UpdatedAt,
			},
		}, nil
	})

	// Update project
	huma.Register(api, huma.Operation{
		OperationID: "update-project",
		Method:      http.MethodPatch,
		Path:        "/v1/projects/{id}",
		Summary:     "Update project",
		Tags:        []string{"Projects"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceProject, rbac.ActionUpdate),
		},
	}, func(ctx context.Context, input *UpdateProjectInput) (*UpdateProjectOutput, error) {
		var settings []byte
		if input.Body.Settings != nil {
			settings, _ = json.Marshal(input.Body.Settings)
		}

		var name string
		if input.Body.Name != nil {
			name = *input.Body.Name
		}

		p, err := queries.UpdateProject(ctx, db.UpdateProjectParams{
			ID:       input.ID,
			Name:     name,
			Settings: settings,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to update project", err)
		}

		var resSettings map[string]any
		if len(p.Settings) > 0 {
			_ = json.Unmarshal(p.Settings, &resSettings)
		}

		return &UpdateProjectOutput{
			Body: Project{
				ID:        p.ID,
				TeamID:    p.TeamID,
				Name:      p.Name,
				Slug:      p.Slug,
				Settings:  resSettings,
				CreatedAt: p.CreatedAt,
				UpdatedAt: p.UpdatedAt,
			},
		}, nil
	})

	// Delete project
	huma.Register(api, huma.Operation{
		OperationID: "delete-project",
		Method:      http.MethodDelete,
		Path:        "/v1/projects/{id}",
		Summary:     "Delete project",
		Tags:        []string{"Projects"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceProject, rbac.ActionDelete),
			auditM.HumaLogAction(audit.EventProjectDeleted, string(rbac.ResourceProject)),
		},
	}, func(ctx context.Context, input *DeleteProjectInput) (*DeleteProjectOutput, error) {
		err := queries.DeleteProject(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to delete project", err)
		}

		return &DeleteProjectOutput{
			Body: struct {
				Message string `json:"message" doc:"Confirmation message"`
			}{
				Message: "Project deleted successfully",
			},
		}, nil
	})
}
