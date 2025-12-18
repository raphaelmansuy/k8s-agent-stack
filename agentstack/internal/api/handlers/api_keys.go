package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
	"github.com/raphaelmansuy/agentstack/internal/domain/auth"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// APIKeyResponse represents an API key in responses.
type APIKeyResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateAPIKeyInput is the input for creating an API key.
type CreateAPIKeyInput struct {
	Body struct {
		Name      string     `json:"name" required:"true" minLength:"1" maxLength:"255" doc:"Key name"`
		ProjectID string     `json:"project_id,omitempty" doc:"Optional project ID"`
		Scopes    []string   `json:"scopes" doc:"Optional scopes"`
		ExpiresAt *time.Time `json:"expires_at,omitempty" doc:"Optional expiration timestamp"`
	}
}

// CreateAPIKeyOutput is the output for creating an API key.
type CreateAPIKeyOutput struct {
	Body struct {
		APIKeyResponse
		RawKey string `json:"api_key" doc:"The full API key (only shown once)"`
	}
}

// ListAPIKeysOutput is the output for listing API keys.
type ListAPIKeysOutput struct {
	Body struct {
		Keys []APIKeyResponse `json:"keys"`
	}
}

// DeleteAPIKeyInput is the input for deleting an API key.
type DeleteAPIKeyInput struct {
	ID string `path:"id" doc:"API Key ID"`
}

// RegisterAPIKeyRoutes registers API key management routes.
func RegisterAPIKeyRoutes(api huma.API, authService *auth.Service, rbacM *middleware.RBACMiddleware, auditM *middleware.AuditMiddleware) {
	// List API keys
	huma.Register(api, huma.Operation{
		OperationID: "list-api-keys",
		Method:      http.MethodGet,
		Path:        "/v1/api-keys",
		Summary:     "List API keys",
		Tags:        []string{"Authentication"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAPIKey, rbac.ActionList),
		},
	}, func(ctx context.Context, input *struct{}) (*ListAPIKeysOutput, error) {
		teamID := middleware.GetTeamID(ctx)
		if teamID == "" {
			return nil, huma.Error401Unauthorized("Authentication required")
		}

		keys, err := authService.ListKeys(ctx, teamID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list API keys", err)
		}

		res := make([]APIKeyResponse, len(keys))
		for i, k := range keys {
			res[i] = APIKeyResponse{
				ID:         k.ID,
				Name:       k.Name,
				KeyPrefix:  k.KeyPrefix,
				Scopes:     k.Scopes,
				LastUsedAt: k.LastUsedAt,
				ExpiresAt:  k.ExpiresAt,
				CreatedAt:  k.CreatedAt,
			}
		}

		return &ListAPIKeysOutput{
			Body: struct {
				Keys []APIKeyResponse `json:"keys"`
			}{Keys: res},
		}, nil
	})

	// Create API key
	huma.Register(api, huma.Operation{
		OperationID: "create-api-key",
		Method:      http.MethodPost,
		Path:        "/v1/api-keys",
		Summary:     "Create API key",
		Tags:        []string{"Authentication"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAPIKey, rbac.ActionCreate),
			auditM.HumaLogAction(audit.EventAPIKeyCreated, string(rbac.ResourceAPIKey)),
		},
	}, func(ctx context.Context, input *CreateAPIKeyInput) (*CreateAPIKeyOutput, error) {
		teamID := middleware.GetTeamID(ctx)
		if teamID == "" {
			return nil, huma.Error401Unauthorized("Authentication required")
		}

		generated, err := authService.CreateKey(ctx, &auth.APIKeyCreate{
			TeamID:    teamID,
			ProjectID: input.Body.ProjectID,
			Name:      input.Body.Name,
			Scopes:    input.Body.Scopes,
			ExpiresAt: input.Body.ExpiresAt,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create API key", err)
		}

		return &CreateAPIKeyOutput{
			Body: struct {
				APIKeyResponse
				RawKey string `json:"api_key" doc:"The full API key (only shown once)"`
			}{
				APIKeyResponse: APIKeyResponse{
					ID:        generated.ID,
					Name:      generated.Name,
					KeyPrefix: generated.KeyPrefix,
					Scopes:    generated.Scopes,
					ExpiresAt: generated.ExpiresAt,
					CreatedAt: generated.CreatedAt,
				},
				RawKey: generated.RawKey,
			},
		}, nil
	})

	// Delete API key
	huma.Register(api, huma.Operation{
		OperationID: "delete-api-key",
		Method:      http.MethodDelete,
		Path:        "/v1/api-keys/{id}",
		Summary:     "Delete API key",
		Tags:        []string{"Authentication"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAPIKey, rbac.ActionDelete),
			auditM.HumaLogAction(audit.EventAPIKeyRevoked, string(rbac.ResourceAPIKey)),
		},
	}, func(ctx context.Context, input *DeleteAPIKeyInput) (*struct {
		Body struct {
			Message string `json:"message"`
		}
	}, error) {
		err := authService.DeleteKey(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to delete API key", err)
		}

		return &struct {
			Body struct {
				Message string `json:"message"`
			}
		}{
			Body: struct {
				Message string `json:"message"`
			}{Message: "API key deleted successfully"},
		}, nil
	})
}
