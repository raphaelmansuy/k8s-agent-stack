// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
	"github.com/raphaelmansuy/agentstack/internal/domain/quota"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
)

// Agent represents an AI agent.
type Agent struct {
	ID          string            `json:"id" doc:"Unique agent identifier"`
	ProjectID   string            `json:"project_id" doc:"Parent project ID"`
	Name        string            `json:"name" doc:"Agent name"`
	Description string            `json:"description,omitempty" doc:"Agent description"`
	Slug        string            `json:"slug" doc:"URL-friendly identifier"`
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
		Framework   string            `json:"framework" default:"custom" doc:"Agent framework"`
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
func RegisterAgentRoutes(api huma.API, pool *database.Pool, rbacM *middleware.RBACMiddleware, quotaM *middleware.QuotaMiddleware, auditM *middleware.AuditMiddleware) {
	queries := db.New(pool.Pool)

	// List agents
	huma.Register(api, huma.Operation{
		OperationID: "list-agents",
		Method:      http.MethodGet,
		Path:        "/v1/agents",
		Summary:     "List agents",
		Tags:        []string{"Agents"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionList),
		},
	}, func(ctx context.Context, input *ListAgentsInput) (*ListAgentsOutput, error) {
		agents, err := queries.ListAgents(ctx, db.ListAgentsParams{
			ProjectID: input.ProjectID,
			Limit:     int32(input.Limit),
			Offset:    int32(input.Offset),
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list agents", err)
		}

		res := make([]Agent, len(agents))
		for i, a := range agents {
			var config map[string]any
			if len(a.Config) > 0 {
				json.Unmarshal(a.Config, &config)
			}
			res[i] = Agent{
				ID:        a.ID,
				ProjectID: a.ProjectID,
				Name:      a.Name,
				Slug:      a.Slug,

				Config:    config,
				Status:    a.Status,
				CreatedAt: a.CreatedAt,
				UpdatedAt: a.UpdatedAt,
			}
		}

		return &ListAgentsOutput{
			Body: struct {
				Agents []Agent `json:"agents" doc:"List of agents"`
				Total  int     `json:"total" doc:"Total count"`
			}{
				Agents: res,
				Total:  len(res),
			},
		}, nil
	})

	// Get agent by ID
	huma.Register(api, huma.Operation{
		OperationID: "get-agent",
		Method:      http.MethodGet,
		Path:        "/v1/agents/{id}",
		Summary:     "Get agent",
		Tags:        []string{"Agents"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetAgentInput) (*GetAgentOutput, error) {
		a, err := queries.GetAgent(ctx, input.ID)
		if err != nil {
			return nil, huma.Error404NotFound("Agent not found")
		}

		var config map[string]any
		if len(a.Config) > 0 {
			json.Unmarshal(a.Config, &config)
		}

		return &GetAgentOutput{
			Body: Agent{
				ID:        a.ID,
				ProjectID: a.ProjectID,
				Name:      a.Name,
				Slug:      a.Slug,

				Config:    config,
				Status:    a.Status,
				CreatedAt: a.CreatedAt,
				UpdatedAt: a.UpdatedAt,
			},
		}, nil
	})

	// Create agent
	huma.Register(api, huma.Operation{
		OperationID: "create-agent",
		Method:      http.MethodPost,
		Path:        "/v1/agents",
		Summary:     "Create agent",
		Tags:        []string{"Agents"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionCreate),
			quotaM.HumaCheckQuota(quota.QuotaAgents),
			quotaM.HumaIncrementAfter(quota.QuotaAgents),
			auditM.HumaLogAction(audit.EventAgentCreated, string(rbac.ResourceAgent)),
		},
	}, func(ctx context.Context, input *CreateAgentInput) (*CreateAgentOutput, error) {
		config, _ := json.Marshal(input.Body.Config)

		a, err := queries.CreateAgent(ctx, db.CreateAgentParams{
			ProjectID:   input.Body.ProjectID,
			Name:        input.Body.Name,
			Slug:        input.Body.Slug,
			Description: pgtype.Text{String: input.Body.Description, Valid: input.Body.Description != ""},
			Framework:   input.Body.Framework,
			Config:      config,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create agent", err)
		}

		var resConfig map[string]any
		if len(a.Config) > 0 {
			json.Unmarshal(a.Config, &resConfig)
		}

		return &CreateAgentOutput{
			Body: Agent{
				ID:        a.ID,
				ProjectID: a.ProjectID,
				Name:      a.Name,
				Slug:      a.Slug,
				Config:    resConfig,
				Status:    a.Status,
				CreatedAt: a.CreatedAt,
				UpdatedAt: a.UpdatedAt,
			},
		}, nil
	})

	// Update agent
	huma.Register(api, huma.Operation{
		OperationID: "update-agent",
		Method:      http.MethodPatch,
		Path:        "/v1/agents/{id}",
		Summary:     "Update agent",
		Tags:        []string{"Agents"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionUpdate),
		},
	}, func(ctx context.Context, input *UpdateAgentInput) (*UpdateAgentOutput, error) {
		var config []byte
		if input.Body.Config != nil {
			config, _ = json.Marshal(input.Body.Config)
		}

		var name string
		if input.Body.Name != nil {
			name = *input.Body.Name
		}

		a, err := queries.UpdateAgent(ctx, db.UpdateAgentParams{
			ID:     input.ID,
			Name:   name,
			Config: config,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to update agent", err)
		}

		var resConfig map[string]any
		if len(a.Config) > 0 {
			json.Unmarshal(a.Config, &resConfig)
		}

		return &UpdateAgentOutput{
			Body: Agent{
				ID:        a.ID,
				ProjectID: a.ProjectID,
				Name:      a.Name,
				Slug:      a.Slug,
				Config:    resConfig,
				Status:    a.Status,
				CreatedAt: a.CreatedAt,
				UpdatedAt: a.UpdatedAt,
			},
		}, nil
	})

	// Delete agent
	huma.Register(api, huma.Operation{
		OperationID: "delete-agent",
		Method:      http.MethodDelete,
		Path:        "/v1/agents/{id}",
		Summary:     "Delete agent",
		Tags:        []string{"Agents"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionDelete),
			auditM.HumaLogAction(audit.EventAgentDeleted, string(rbac.ResourceAgent)),
		},
	}, func(ctx context.Context, input *DeleteAgentInput) (*DeleteAgentOutput, error) {
		err := queries.DeleteAgent(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to delete agent", err)
		}

		return &DeleteAgentOutput{
			Body: struct {
				Message string `json:"message" doc:"Confirmation message"`
			}{
				Message: "Agent deleted successfully",
			},
		}, nil
	})
}
