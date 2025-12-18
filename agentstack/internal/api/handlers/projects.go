// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
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
func RegisterProjectRoutes(api huma.API, db *database.Pool) {
	// List projects
	huma.Get(api, "/v1/projects", func(ctx context.Context, input *ListProjectsInput) (*ListProjectsOutput, error) {
		// TODO: Implement database query
		return &ListProjectsOutput{
			Body: struct {
				Projects []Project `json:"projects" doc:"List of projects"`
				Total    int       `json:"total" doc:"Total count"`
			}{
				Projects: []Project{},
				Total:    0,
			},
		}, nil
	})

	// Get project by ID
	huma.Get(api, "/v1/projects/{id}", func(ctx context.Context, input *GetProjectInput) (*GetProjectOutput, error) {
		// TODO: Implement database query
		now := time.Now()
		return &GetProjectOutput{
			Body: Project{
				ID:        input.ID,
				TeamID:    "team-1",
				Name:      "Sample Project",
				Slug:      "sample-project",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}, nil
	})

	// Create project
	huma.Post(api, "/v1/projects", func(ctx context.Context, input *CreateProjectInput) (*CreateProjectOutput, error) {
		now := time.Now()
		project := Project{
			ID:          uuid.New().String(),
			TeamID:      "team-1", // TODO: Get from auth context
			Name:        input.Body.Name,
			Description: input.Body.Description,
			Slug:        input.Body.Slug,
			Settings:    input.Body.Settings,
			Metadata:    input.Body.Metadata,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// TODO: Save to database

		return &CreateProjectOutput{Body: project}, nil
	})

	// Update project
	huma.Patch(api, "/v1/projects/{id}", func(ctx context.Context, input *UpdateProjectInput) (*UpdateProjectOutput, error) {
		now := time.Now()
		// TODO: Fetch and update in database
		project := Project{
			ID:        input.ID,
			TeamID:    "team-1",
			Name:      "Updated Project",
			Slug:      "updated-project",
			CreatedAt: now.Add(-24 * time.Hour),
			UpdatedAt: now,
		}

		if input.Body.Name != nil {
			project.Name = *input.Body.Name
		}

		return &UpdateProjectOutput{Body: project}, nil
	})

	// Delete project
	huma.Delete(api, "/v1/projects/{id}", func(ctx context.Context, input *DeleteProjectInput) (*DeleteProjectOutput, error) {
		// TODO: Delete from database
		return &DeleteProjectOutput{
			Body: struct {
				Message string `json:"message" doc:"Confirmation message"`
			}{
				Message: "Project deleted successfully",
			},
		}, nil
	})
}
