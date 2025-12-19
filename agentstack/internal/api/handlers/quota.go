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
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/quota"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// GetUsageSummaryInput is the input for getting usage summary.
type GetUsageSummaryInput struct {
	TeamID string `path:"id" doc:"Team ID"`
}

// GetUsageSummaryOutput is the output for getting usage summary.
type GetUsageSummaryOutput struct {
	Body *quota.UsageSummary
}

// GetQuotasInput is the input for getting quotas.
type GetQuotasInput struct {
	TeamID string `path:"id" doc:"Team ID"`
}

// GetQuotasOutput is the output for getting quotas.
type GetQuotasOutput struct {
	Body struct {
		Quotas []quota.UsageItem `json:"quotas" doc:"List of quotas"`
	}
}

// SetQuotaInput is the input for setting a quota.
type SetQuotaInput struct {
	TeamID string `path:"id" doc:"Team ID"`
	Body   struct {
		Type      quota.QuotaType `json:"type" required:"true" doc:"Quota type"`
		Limit     int64           `json:"limit" required:"true" doc:"Quota limit"`
		Period    string          `json:"period,omitempty" doc:"Quota period"`
		ProjectID string          `json:"project_id,omitempty" doc:"Project ID"`
	}
}

// SetQuotaOutput is the output for setting a quota.
type SetQuotaOutput struct {
	Body *quota.Quota
}

// CheckQuotaInput is the input for checking a quota.
type CheckQuotaInput struct {
	TeamID string `path:"id" doc:"Team ID"`
	Body   struct {
		Type      quota.QuotaType `json:"type" required:"true" doc:"Quota type"`
		Amount    int64           `json:"amount" default:"1" doc:"Amount to check"`
		ProjectID string          `json:"project_id,omitempty" doc:"Project ID"`
	}
}

// CheckQuotaOutput is the output for checking a quota.
type CheckQuotaOutput struct {
	Body *quota.CheckQuotaResult
}

// ApplyPlanInput is the input for applying a plan.
type ApplyPlanInput struct {
	TeamID string `path:"id" doc:"Team ID"`
	Body   struct {
		PlanID string `json:"plan_id" required:"true" doc:"Plan ID"`
	}
}

// ApplyPlanOutput is the output for applying a plan.
type ApplyPlanOutput struct {
	Body struct {
		Message string      `json:"message" doc:"Confirmation message"`
		Plan    *quota.Plan `json:"plan" doc:"Applied plan"`
	}
}

// ListPlansOutput is the output for listing plans.
type ListPlansOutput struct {
	Body struct {
		Plans []quota.Plan `json:"plans" doc:"List of available plans"`
	}
}

// RegisterQuotaRoutes registers quota routes.
func RegisterQuotaRoutes(api huma.API, service *quota.Service, rbacM *middleware.RBACMiddleware) {
	// Get usage summary
	huma.Register(api, huma.Operation{
		OperationID: "get-usage-summary",
		Method:      http.MethodGet,
		Path:        "/v1/teams/{id}/usage",
		Summary:     "Get usage summary",
		Tags:        []string{"Quota"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceQuota, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetUsageSummaryInput) (*GetUsageSummaryOutput, error) {
		summary, err := service.GetUsageSummary(ctx, input.TeamID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get usage summary", err)
		}

		return &GetUsageSummaryOutput{
			Body: summary,
		}, nil
	})

	// Get quotas
	huma.Register(api, huma.Operation{
		OperationID: "get-quotas",
		Method:      http.MethodGet,
		Path:        "/v1/teams/{id}/quotas",
		Summary:     "Get quotas",
		Tags:        []string{"Quota"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceQuota, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetQuotasInput) (*GetQuotasOutput, error) {
		summary, err := service.GetUsageSummary(ctx, input.TeamID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get quotas", err)
		}

		return &GetQuotasOutput{
			Body: struct {
				Quotas []quota.UsageItem `json:"quotas" doc:"List of quotas"`
			}{
				Quotas: summary.Items,
			},
		}, nil
	})

	// Set quota
	huma.Register(api, huma.Operation{
		OperationID: "set-quota",
		Method:      http.MethodPost,
		Path:        "/v1/teams/{id}/quotas",
		Summary:     "Set quota",
		Tags:        []string{"Quota"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceQuota, rbac.ActionManage),
		},
	}, func(ctx context.Context, input *SetQuotaInput) (*SetQuotaOutput, error) {
		q := &quota.Quota{
			TeamID:    input.TeamID,
			ProjectID: input.Body.ProjectID,
			Type:      input.Body.Type,
			Limit:     input.Body.Limit,
			Period:    input.Body.Period,
		}

		if err := service.SetQuota(ctx, q); err != nil {
			return nil, huma.Error500InternalServerError("Failed to set quota", err)
		}

		return &SetQuotaOutput{
			Body: q,
		}, nil
	})

	// Check quota
	huma.Register(api, huma.Operation{
		OperationID: "check-quota",
		Method:      http.MethodPost,
		Path:        "/v1/teams/{id}/quotas/check",
		Summary:     "Check quota",
		Tags:        []string{"Quota"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceQuota, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *CheckQuotaInput) (*CheckQuotaOutput, error) {
		result, err := service.CheckQuota(ctx, quota.CheckQuotaRequest{
			TeamID:    input.TeamID,
			ProjectID: input.Body.ProjectID,
			Type:      input.Body.Type,
			Amount:    input.Body.Amount,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to check quota", err)
		}

		return &CheckQuotaOutput{
			Body: result,
		}, nil
	})

	// Apply plan
	huma.Register(api, huma.Operation{
		OperationID: "apply-plan",
		Method:      http.MethodPost,
		Path:        "/v1/teams/{id}/plan",
		Summary:     "Apply plan",
		Tags:        []string{"Quota"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceQuota, rbac.ActionManage),
		},
	}, func(ctx context.Context, input *ApplyPlanInput) (*ApplyPlanOutput, error) {
		plan := quota.GetPlan(input.Body.PlanID)
		if plan == nil {
			return nil, huma.Error404NotFound("Plan not found")
		}

		if err := service.ApplyPlan(ctx, input.TeamID, input.Body.PlanID); err != nil {
			return nil, huma.Error500InternalServerError("Failed to apply plan", err)
		}

		return &ApplyPlanOutput{
			Body: struct {
				Message string      `json:"message" doc:"Confirmation message"`
				Plan    *quota.Plan `json:"plan" doc:"Applied plan"`
			}{
				Message: "Plan applied successfully",
				Plan:    plan,
			},
		}, nil
	})

	// List plans
	huma.Register(api, huma.Operation{
		OperationID: "list-plans",
		Method:      http.MethodGet,
		Path:        "/v1/plans",
		Summary:     "List plans",
		Tags:        []string{"Quota"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceQuota, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *struct{}) (*ListPlansOutput, error) {
		return &ListPlansOutput{
			Body: struct {
				Plans []quota.Plan `json:"plans" doc:"List of available plans"`
			}{
				Plans: quota.DefaultPlans,
			},
		}, nil
	})
}
