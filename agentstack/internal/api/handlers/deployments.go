// Package handlers provides HTTP handlers for the AgentStack API.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
	"github.com/raphaelmansuy/agentstack/internal/domain/deployment"
	"github.com/raphaelmansuy/agentstack/internal/domain/quota"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// Deployment represents an agent deployment.
type Deployment struct {
	ID          string                   `json:"id" doc:"Unique deployment identifier"`
	Name        string                   `json:"name" doc:"Deployment name"`
	Description string                   `json:"description,omitempty" doc:"Deployment description"`
	Image       string                   `json:"image" doc:"Container image"`
	Namespace   string                   `json:"namespace" doc:"Kubernetes namespace"`
	Type        deployment.AgentType     `json:"type" doc:"Deployment type (kagent, knative)"`
	Status      deployment.AgentStatus   `json:"status" doc:"Deployment status"`
	Env         map[string]string        `json:"env,omitempty" doc:"Environment variables"`
	Resources   *deployment.ResourceSpec `json:"resources,omitempty" doc:"Resource requirements"`
	Replicas    int32                    `json:"replicas" doc:"Number of replicas"`
	CreatedAt   time.Time                `json:"created_at" doc:"Creation timestamp"`
	UpdatedAt   time.Time                `json:"updated_at" doc:"Last update timestamp"`
}

// CreateDeploymentInput is the input for creating a deployment.
type CreateDeploymentInput struct {
	Body struct {
		ID          string                   `json:"id" required:"true" doc:"Unique deployment identifier"`
		Name        string                   `json:"name" required:"true" doc:"Deployment name"`
		Description string                   `json:"description,omitempty" doc:"Deployment description"`
		Image       string                   `json:"image" required:"true" doc:"Container image"`
		Namespace   string                   `json:"namespace,omitempty" doc:"Kubernetes namespace"`
		Type        deployment.AgentType     `json:"type,omitempty" doc:"Deployment type (kagent, knative)"`
		Env         map[string]string        `json:"env,omitempty" doc:"Environment variables"`
		Resources   *deployment.ResourceSpec `json:"resources,omitempty" doc:"Resource requirements"`
		Replicas    int32                    `json:"replicas,omitempty" doc:"Number of replicas"`
		Model       *deployment.ModelConfig  `json:"model,omitempty" doc:"Model configuration"`
		Tools       []deployment.ToolSpec    `json:"tools,omitempty" doc:"Tool specifications"`
	}
}

// CreateDeploymentOutput is the output for creating a deployment.
type CreateDeploymentOutput struct {
	Body *deployment.Agent
}

// ListDeploymentsInput is the input for listing deployments.
type ListDeploymentsInput struct {
	Namespace string `query:"namespace" doc:"Filter by namespace"`
}

// ListDeploymentsOutput is the output for listing deployments.
type ListDeploymentsOutput struct {
	Body struct {
		Deployments []*deployment.Agent `json:"deployments" doc:"List of deployments"`
		Total       int                 `json:"total" doc:"Total count"`
	}
}

// GetDeploymentInput is the input for getting a deployment.
type GetDeploymentInput struct {
	ID string `path:"id" doc:"Deployment ID"`
}

// GetDeploymentOutput is the output for getting a deployment.
type GetDeploymentOutput struct {
	Body *deployment.Agent
}

// UpdateDeploymentInput is the input for updating a deployment.
type UpdateDeploymentInput struct {
	ID   string `path:"id" doc:"Deployment ID"`
	Body struct {
		Image       string                   `json:"image,omitempty" doc:"Container image"`
		Description string                   `json:"description,omitempty" doc:"Deployment description"`
		Env         map[string]string        `json:"env,omitempty" doc:"Environment variables"`
		Resources   *deployment.ResourceSpec `json:"resources,omitempty" doc:"Resource requirements"`
	}
}

// UpdateDeploymentOutput is the output for updating a deployment.
type UpdateDeploymentOutput struct {
	Body *deployment.Agent
}

// DeleteDeploymentInput is the input for deleting a deployment.
type DeleteDeploymentInput struct {
	ID string `path:"id" doc:"Deployment ID"`
}

// DeleteDeploymentOutput is the output for deleting a deployment.
type DeleteDeploymentOutput struct {
	Body struct {
		Message string `json:"message" doc:"Confirmation message"`
	}
}

// GetDeploymentStatusInput is the input for getting deployment status.
type GetDeploymentStatusInput struct {
	ID string `path:"id" doc:"Deployment ID"`
}

// GetDeploymentStatusOutput is the output for getting deployment status.
type GetDeploymentStatusOutput struct {
	Body deployment.AgentStatus
}

// WaitForReadyInput is the input for waiting for deployment readiness.
type WaitForReadyInput struct {
	ID   string `path:"id" doc:"Deployment ID"`
	Body struct {
		TimeoutSeconds int `json:"timeoutSeconds,omitempty" doc:"Timeout in seconds"`
	}
}

// WaitForReadyOutput is the output for waiting for deployment readiness.
type WaitForReadyOutput struct {
	Body *deployment.Agent
}

// RegisterDeploymentRoutes registers deployment routes.
func RegisterDeploymentRoutes(api huma.API, service *deployment.Service, rbacM *middleware.RBACMiddleware, quotaM *middleware.QuotaMiddleware, auditM *middleware.AuditMiddleware) {
	// List deployments
	huma.Register(api, huma.Operation{
		OperationID: "list-deployments",
		Method:      http.MethodGet,
		Path:        "/v1/deployments",
		Summary:     "List deployments",
		Tags:        []string{"Deployments"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceDeployment, rbac.ActionList),
		},
	}, func(ctx context.Context, input *ListDeploymentsInput) (*ListDeploymentsOutput, error) {
		agents, err := service.ListAgents(ctx, input.Namespace)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list deployments", err)
		}

		return &ListDeploymentsOutput{
			Body: struct {
				Deployments []*deployment.Agent `json:"deployments" doc:"List of deployments"`
				Total       int                 `json:"total" doc:"Total count"`
			}{
				Deployments: agents,
				Total:       len(agents),
			},
		}, nil
	})

	// Get deployment by ID
	huma.Register(api, huma.Operation{
		OperationID: "get-deployment",
		Method:      http.MethodGet,
		Path:        "/v1/deployments/{id}",
		Summary:     "Get deployment",
		Tags:        []string{"Deployments"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceDeployment, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetDeploymentInput) (*GetDeploymentOutput, error) {
		agent, err := service.GetAgent(ctx, input.ID)
		if err != nil {
			return nil, huma.Error404NotFound("Deployment not found", err)
		}

		return &GetDeploymentOutput{
			Body: agent,
		}, nil
	})

	// Create deployment
	huma.Register(api, huma.Operation{
		OperationID: "create-deployment",
		Method:      http.MethodPost,
		Path:        "/v1/deployments",
		Summary:     "Create deployment",
		Tags:        []string{"Deployments"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceDeployment, rbac.ActionCreate),
			quotaM.HumaCheckQuota(quota.QuotaDeployments),
			quotaM.HumaIncrementAfter(quota.QuotaDeployments),
			auditM.HumaLogAction(audit.EventAgentDeployed, string(rbac.ResourceDeployment)),
		},
	}, func(ctx context.Context, input *CreateDeploymentInput) (*CreateDeploymentOutput, error) {
		agent := &deployment.Agent{
			ID:          input.Body.ID,
			Name:        input.Body.Name,
			Description: input.Body.Description,
			Image:       input.Body.Image,
			Namespace:   input.Body.Namespace,
			Type:        input.Body.Type,
			Env:         input.Body.Env,
			Resources:   input.Body.Resources,
			Replicas:    input.Body.Replicas,
			Model:       input.Body.Model,
			Tools:       input.Body.Tools,
		}

		created, err := service.CreateAgent(ctx, agent)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create deployment", err)
		}

		return &CreateDeploymentOutput{
			Body: created,
		}, nil
	})

	// Update deployment
	huma.Register(api, huma.Operation{
		OperationID: "update-deployment",
		Method:      http.MethodPatch,
		Path:        "/v1/deployments/{id}",
		Summary:     "Update deployment",
		Tags:        []string{"Deployments"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceDeployment, rbac.ActionUpdate),
		},
	}, func(ctx context.Context, input *UpdateDeploymentInput) (*UpdateDeploymentOutput, error) {
		update := &deployment.Agent{
			Image:       input.Body.Image,
			Description: input.Body.Description,
			Env:         input.Body.Env,
			Resources:   input.Body.Resources,
		}

		updated, err := service.UpdateAgent(ctx, input.ID, update)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to update deployment", err)
		}

		return &UpdateDeploymentOutput{
			Body: updated,
		}, nil
	})

	// Delete deployment
	huma.Register(api, huma.Operation{
		OperationID: "delete-deployment",
		Method:      http.MethodDelete,
		Path:        "/v1/deployments/{id}",
		Summary:     "Delete deployment",
		Tags:        []string{"Deployments"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceDeployment, rbac.ActionDelete),
			auditM.HumaLogAction(audit.EventAgentDeleted, string(rbac.ResourceDeployment)),
		},
	}, func(ctx context.Context, input *DeleteDeploymentInput) (*DeleteDeploymentOutput, error) {
		err := service.DeleteAgent(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to delete deployment", err)
		}

		return &DeleteDeploymentOutput{
			Body: struct {
				Message string `json:"message" doc:"Confirmation message"`
			}{
				Message: "Deployment deleted successfully",
			},
		}, nil
	})

	// Get deployment status
	huma.Register(api, huma.Operation{
		OperationID: "get-deployment-status",
		Method:      http.MethodGet,
		Path:        "/v1/deployments/{id}/status",
		Summary:     "Get deployment status",
		Tags:        []string{"Deployments"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceDeployment, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetDeploymentStatusInput) (*GetDeploymentStatusOutput, error) {
		agent, err := service.GetAgent(ctx, input.ID)
		if err != nil {
			return nil, huma.Error404NotFound("Deployment not found", err)
		}

		return &GetDeploymentStatusOutput{
			Body: agent.Status,
		}, nil
	})

	// Wait for deployment readiness
	huma.Register(api, huma.Operation{
		OperationID: "wait-for-deployment",
		Method:      http.MethodPost,
		Path:        "/v1/deployments/{id}/wait",
		Summary:     "Wait for deployment readiness",
		Tags:        []string{"Deployments"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceDeployment, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *WaitForReadyInput) (*WaitForReadyOutput, error) {
		timeoutSeconds := input.Body.TimeoutSeconds
		if timeoutSeconds <= 0 {
			timeoutSeconds = 300
		}
		timeout := time.Duration(timeoutSeconds) * time.Second

		if err := service.WaitForReady(ctx, input.ID, timeout); err != nil {
			return nil, huma.Error504GatewayTimeout("Timeout waiting for deployment readiness", err)
		}

		agent, err := service.GetAgent(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get deployment after wait", err)
		}

		return &WaitForReadyOutput{
			Body: agent,
		}, nil
	})
}
