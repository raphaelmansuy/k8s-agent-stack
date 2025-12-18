// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
)

// Agent represents an AI agent.
type Agent struct {
	ID          string            `json:"id" doc:"Unique agent identifier"`
	ProjectID   string            `json:"project_id" doc:"Parent project ID"`
	Name        string            `json:"name" doc:"Agent name"`
	Description string            `json:"description,omitempty" doc:"Agent description"`
	Slug        string            `json:"slug" doc:"URL-friendly identifier"`
	ModelID     string            `json:"model_id" doc:"LLM model identifier"`
	Config      map[string]any    `json:"config,omitempty" doc:"Agent configuration"`
	Metadata    map[string]string `json:"metadata,omitempty" doc:"Custom metadata"`
	Status      string            `json:"status" doc:"Agent status" enum:"draft,active,archived"`
	CreatedAt   time.Time         `json:"created_at" doc:"Creation timestamp"`
	UpdatedAt   time.Time         `json:"updated_at" doc:"Last update timestamp"`
}

// ListAgentsInput is the input for listing agents.
type ListAgentsInput struct {
	ProjectID string `query:"project_id" required:"true" doc:"Filter by project ID"`
	Limit     int    `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Maximum results"`
	Offset    int    `query:"offset" default:"0" minimum:"0" doc:"Pagination offset"`
	Status    string `query:"status" enum:"draft,active,archived" doc:"Filter by status"`
}

// ListAgentsOutput is the output for listing agents.
type ListAgentsOutput struct {
	Body struct {
		Agents []Agent `json:"agents" doc:"List of agents"`
		Total  int     `json:"total" doc:"Total count"`
	}
}

// GetAgentInput is the input for getting an agent.
type GetAgentInput struct {
	ID string `path:"id" doc:"Agent ID"`
}

// GetAgentOutput is the output for getting an agent.
type GetAgentOutput struct {
	Body Agent
}

// CreateAgentInput is the input for creating an agent.
type CreateAgentInput struct {
	Body struct {
		ProjectID   string            `json:"project_id" required:"true" doc:"Parent project ID"`
		Name        string            `json:"name" required:"true" minLength:"1" maxLength:"255" doc:"Agent name"`
		Description string            `json:"description,omitempty" maxLength:"1000" doc:"Agent description"`
		Slug        string            `json:"slug" required:"true" pattern:"^[a-z0-9-]+$" doc:"URL-friendly identifier"`
		ModelID     string            `json:"model_id" required:"true" doc:"LLM model identifier"`
		Config      map[string]any    `json:"config,omitempty" doc:"Agent configuration"`
		Metadata    map[string]string `json:"metadata,omitempty" doc:"Custom metadata"`
	}
}

// CreateAgentOutput is the output for creating an agent.
type CreateAgentOutput struct {
	Body Agent
}

// UpdateAgentInput is the input for updating an agent.
type UpdateAgentInput struct {
	ID   string `path:"id" doc:"Agent ID"`
	Body struct {
		Name        *string           `json:"name,omitempty" minLength:"1" maxLength:"255" doc:"Agent name"`
		Description *string           `json:"description,omitempty" maxLength:"1000" doc:"Agent description"`
		ModelID     *string           `json:"model_id,omitempty" doc:"LLM model identifier"`
		Config      map[string]any    `json:"config,omitempty" doc:"Agent configuration"`
		Metadata    map[string]string `json:"metadata,omitempty" doc:"Custom metadata"`
		Status      *string           `json:"status,omitempty" enum:"draft,active,archived" doc:"Agent status"`
	}
}

// UpdateAgentOutput is the output for updating an agent.
type UpdateAgentOutput struct {
	Body Agent
}

// DeleteAgentInput is the input for deleting an agent.
type DeleteAgentInput struct {
	ID string `path:"id" doc:"Agent ID"`
}

// DeleteAgentOutput is the output for deleting an agent.
type DeleteAgentOutput struct {
	Body struct {
		Message string `json:"message" doc:"Confirmation message"`
	}
}

// RegisterAgentRoutes registers agent routes.
func RegisterAgentRoutes(api huma.API, db *database.Pool) {
	// List agents
	huma.Get(api, "/v1/agents", func(ctx context.Context, input *ListAgentsInput) (*ListAgentsOutput, error) {
		// TODO: Implement database query
		return &ListAgentsOutput{
			Body: struct {
				Agents []Agent `json:"agents" doc:"List of agents"`
				Total  int     `json:"total" doc:"Total count"`
			}{
				Agents: []Agent{},
				Total:  0,
			},
		}, nil
	})

	// Get agent by ID
	huma.Get(api, "/v1/agents/{id}", func(ctx context.Context, input *GetAgentInput) (*GetAgentOutput, error) {
		// TODO: Implement database query
		now := time.Now()
		return &GetAgentOutput{
			Body: Agent{
				ID:        input.ID,
				ProjectID: "project-1",
				Name:      "Sample Agent",
				Slug:      "sample-agent",
				ModelID:   "gpt-4",
				Status:    "active",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}, nil
	})

	// Create agent
	huma.Post(api, "/v1/agents", func(ctx context.Context, input *CreateAgentInput) (*CreateAgentOutput, error) {
		now := time.Now()
		agent := Agent{
			ID:          uuid.New().String(),
			ProjectID:   input.Body.ProjectID,
			Name:        input.Body.Name,
			Description: input.Body.Description,
			Slug:        input.Body.Slug,
			ModelID:     input.Body.ModelID,
			Config:      input.Body.Config,
			Metadata:    input.Body.Metadata,
			Status:      "draft",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// TODO: Save to database

		return &CreateAgentOutput{Body: agent}, nil
	})

	// Update agent
	huma.Patch(api, "/v1/agents/{id}", func(ctx context.Context, input *UpdateAgentInput) (*UpdateAgentOutput, error) {
		now := time.Now()
		// TODO: Fetch and update in database
		agent := Agent{
			ID:        input.ID,
			ProjectID: "project-1",
			Name:      "Updated Agent",
			Slug:      "updated-agent",
			ModelID:   "gpt-4",
			Status:    "active",
			CreatedAt: now.Add(-24 * time.Hour),
			UpdatedAt: now,
		}

		if input.Body.Name != nil {
			agent.Name = *input.Body.Name
		}
		if input.Body.Status != nil {
			agent.Status = *input.Body.Status
		}

		return &UpdateAgentOutput{Body: agent}, nil
	})

	// Delete agent
	huma.Delete(api, "/v1/agents/{id}", func(ctx context.Context, input *DeleteAgentInput) (*DeleteAgentOutput, error) {
		// TODO: Delete from database
		return &DeleteAgentOutput{
			Body: struct {
				Message string `json:"message" doc:"Confirmation message"`
			}{
				Message: "Agent deleted successfully",
			},
		}, nil
	})
}
