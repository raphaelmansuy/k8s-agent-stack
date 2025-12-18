// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// ListTracesInput is the input for listing traces.
type ListTracesInput struct {
	AgentID   string `query:"agent_id" doc:"Filter by agent ID"`
	SessionID string `query:"session_id" doc:"Filter by session ID"`
	Limit     int    `query:"limit" default:"50" doc:"Maximum number of traces to return"`
}

// ListTracesOutput is the output for listing traces.
type ListTracesOutput struct {
	Body struct {
		Traces []*evaluation.Trace `json:"traces" doc:"List of traces"`
		Total  int                 `json:"total" doc:"Total count"`
	}
}

// GetTraceInput is the input for getting a trace.
type GetTraceInput struct {
	TraceID string `path:"traceId" doc:"Trace ID"`
}

// GetTraceOutput is the output for getting a trace.
type GetTraceOutput struct {
	Body *evaluation.Trace
}

// SubmitFeedbackInput is the input for submitting feedback.
type SubmitFeedbackInput struct {
	Body struct {
		TraceID string   `json:"trace_id" required:"true" doc:"Trace ID to provide feedback for"`
		Rating  int      `json:"rating" required:"true" minimum:"1" maximum:"5" doc:"Rating from 1-5"`
		Comment string   `json:"comment,omitempty" doc:"Optional feedback comment"`
		Tags    []string `json:"tags,omitempty" doc:"Optional feedback tags"`
	}
}

// SubmitFeedbackOutput is the output for submitting feedback.
type SubmitFeedbackOutput struct {
	Body struct {
		Success bool   `json:"success" doc:"Success status"`
		Message string `json:"message" doc:"Status message"`
	}
}

// GetFeedbackInput is the input for getting feedback.
type GetFeedbackInput struct {
	TraceID string `path:"traceId" doc:"Trace ID"`
}

// GetFeedbackOutput is the output for getting feedback.
type GetFeedbackOutput struct {
	Body struct {
		Feedback []*evaluation.Feedback `json:"feedback" doc:"List of feedback"`
	}
}

// RegisterEvaluationRoutes registers evaluation routes.
func RegisterEvaluationRoutes(api huma.API, service *evaluation.Service, rbacM *middleware.RBACMiddleware, auditM *middleware.AuditMiddleware) {
	// List traces
	huma.Register(api, huma.Operation{
		OperationID: "list-traces",
		Method:      http.MethodGet,
		Path:        "/v1/evaluation/traces",
		Summary:     "List evaluation traces",
		Tags:        []string{"Evaluation"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionList),
		},
	}, func(ctx context.Context, input *ListTracesInput) (*ListTracesOutput, error) {
		filter := evaluation.TraceFilter{
			AgentID:   input.AgentID,
			SessionID: input.SessionID,
			Limit:     input.Limit,
		}

		traces, err := service.ListTraces(ctx, filter)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list traces", err)
		}

		return &ListTracesOutput{
			Body: struct {
				Traces []*evaluation.Trace `json:"traces" doc:"List of traces"`
				Total  int                 `json:"total" doc:"Total count"`
			}{
				Traces: traces,
				Total:  len(traces),
			},
		}, nil
	})

	// Get trace
	huma.Register(api, huma.Operation{
		OperationID: "get-trace",
		Method:      http.MethodGet,
		Path:        "/v1/evaluation/traces/{traceId}",
		Summary:     "Get evaluation trace",
		Tags:        []string{"Evaluation"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetTraceInput) (*GetTraceOutput, error) {
		trace, err := service.GetTrace(ctx, input.TraceID)
		if err != nil {
			return nil, huma.Error404NotFound("Trace not found", err)
		}

		return &GetTraceOutput{
			Body: trace,
		}, nil
	})

	// Submit feedback
	huma.Register(api, huma.Operation{
		OperationID: "submit-feedback",
		Method:      http.MethodPost,
		Path:        "/v1/evaluation/feedback",
		Summary:     "Submit feedback",
		Tags:        []string{"Evaluation"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionInvoke),
			auditM.HumaLogAction(audit.EventFeedbackSubmitted, string(rbac.ResourceAgent)),
		},
	}, func(ctx context.Context, input *SubmitFeedbackInput) (*SubmitFeedbackOutput, error) {
		err := service.CollectFeedback(ctx, &evaluation.FeedbackRequest{
			TraceID: input.Body.TraceID,
			Rating:  input.Body.Rating,
			Comment: input.Body.Comment,
			Tags:    input.Body.Tags,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to submit feedback", err)
		}

		return &SubmitFeedbackOutput{
			Body: struct {
				Success bool   `json:"success" doc:"Success status"`
				Message string `json:"message" doc:"Status message"`
			}{
				Success: true,
				Message: "Feedback submitted successfully",
			},
		}, nil
	})

	// Get feedback
	huma.Register(api, huma.Operation{
		OperationID: "get-feedback",
		Method:      http.MethodGet,
		Path:        "/v1/evaluation/traces/{traceId}/feedback",
		Summary:     "Get feedback",
		Tags:        []string{"Evaluation"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetFeedbackInput) (*GetFeedbackOutput, error) {
		feedback, err := service.GetFeedback(ctx, input.TraceID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get feedback", err)
		}

		return &GetFeedbackOutput{
			Body: struct {
				Feedback []*evaluation.Feedback `json:"feedback" doc:"List of feedback"`
			}{
				Feedback: feedback,
			},
		}, nil
	})
}
