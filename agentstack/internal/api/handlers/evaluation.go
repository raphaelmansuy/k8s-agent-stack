package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
)

// EvaluationHandler handles evaluation-related API endpoints.
type EvaluationHandler struct {
	evalSvc EvaluationService
}

// EvaluationService defines the interface for evaluation operations.
type EvaluationService interface {
	TraceInteraction(ctx context.Context, interaction *evaluation.Interaction) (*evaluation.Trace, error)
	GetTrace(ctx context.Context, id string) (*evaluation.Trace, error)
	ListTraces(ctx context.Context, filter evaluation.TraceFilter) ([]*evaluation.Trace, error)
	CollectFeedback(ctx context.Context, req *evaluation.FeedbackRequest) error
	GetFeedback(ctx context.Context, traceID string) ([]*evaluation.Feedback, error)
}

// NewEvaluationHandler creates a new evaluation handler.
func NewEvaluationHandler(evalSvc EvaluationService) *EvaluationHandler {
	return &EvaluationHandler{evalSvc: evalSvc}
}

// Register registers evaluation routes with the Huma API.
func (h *EvaluationHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-traces",
		Method:      http.MethodGet,
		Path:        "/v1/evaluation/traces",
		Summary:     "List traces",
		Description: "List agent interaction traces with optional filtering",
		Tags:        []string{"evaluation"},
	}, h.ListTraces)

	huma.Register(api, huma.Operation{
		OperationID: "get-trace",
		Method:      http.MethodGet,
		Path:        "/v1/evaluation/traces/{traceId}",
		Summary:     "Get trace",
		Description: "Get a specific agent interaction trace",
		Tags:        []string{"evaluation"},
	}, h.GetTrace)

	huma.Register(api, huma.Operation{
		OperationID: "submit-feedback",
		Method:      http.MethodPost,
		Path:        "/v1/evaluation/feedback",
		Summary:     "Submit feedback",
		Description: "Submit user feedback for an agent interaction",
		Tags:        []string{"evaluation"},
	}, h.SubmitFeedback)

	huma.Register(api, huma.Operation{
		OperationID: "get-feedback",
		Method:      http.MethodGet,
		Path:        "/v1/evaluation/traces/{traceId}/feedback",
		Summary:     "Get feedback",
		Description: "Get feedback for a specific trace",
		Tags:        []string{"evaluation"},
	}, h.GetFeedback)
}

// --- Request/Response Types ---

// ListTracesRequest defines the request for listing traces.
type ListTracesRequest struct {
	AgentID   string `query:"agent_id" doc:"Filter by agent ID"`
	SessionID string `query:"session_id" doc:"Filter by session ID"`
	Limit     int    `query:"limit" default:"50" doc:"Maximum number of traces to return"`
}

// ListTracesResponse defines the response for listing traces.
type ListTracesResponse struct {
	Body struct {
		Traces []*evaluation.Trace `json:"traces"`
		Total  int                 `json:"total"`
	}
}

// ListTraces handles GET /v1/evaluation/traces
func (h *EvaluationHandler) ListTraces(ctx context.Context, req *ListTracesRequest) (*ListTracesResponse, error) {
	filter := evaluation.TraceFilter{
		AgentID:   req.AgentID,
		SessionID: req.SessionID,
		Limit:     req.Limit,
	}

	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	traces, err := h.evalSvc.ListTraces(ctx, filter)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list traces")
	}

	resp := &ListTracesResponse{}
	resp.Body.Traces = traces
	resp.Body.Total = len(traces)
	return resp, nil
}

// GetTraceRequest defines the request for getting a trace.
type GetTraceRequest struct {
	TraceID string `path:"traceId" doc:"Trace ID"`
}

// GetTraceResponse defines the response for getting a trace.
type GetTraceResponse struct {
	Body *evaluation.Trace
}

// GetTrace handles GET /v1/evaluation/traces/:traceId
func (h *EvaluationHandler) GetTrace(ctx context.Context, req *GetTraceRequest) (*GetTraceResponse, error) {
	trace, err := h.evalSvc.GetTrace(ctx, req.TraceID)
	if err != nil {
		return nil, huma.Error404NotFound("trace not found")
	}

	return &GetTraceResponse{Body: trace}, nil
}

// SubmitFeedbackRequest defines the request for submitting feedback.
type SubmitFeedbackRequest struct {
	Body struct {
		TraceID string   `json:"trace_id" required:"true" doc:"Trace ID to provide feedback for"`
		Rating  int      `json:"rating" required:"true" minimum:"1" maximum:"5" doc:"Rating from 1-5"`
		Comment string   `json:"comment,omitempty" doc:"Optional feedback comment"`
		Tags    []string `json:"tags,omitempty" doc:"Optional feedback tags"`
	}
}

// SubmitFeedbackResponse defines the response for submitting feedback.
type SubmitFeedbackResponse struct {
	Body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
}

// SubmitFeedback handles POST /v1/evaluation/feedback
func (h *EvaluationHandler) SubmitFeedback(ctx context.Context, req *SubmitFeedbackRequest) (*SubmitFeedbackResponse, error) {
	if req.Body.Rating < 1 || req.Body.Rating > 5 {
		return nil, huma.Error400BadRequest("rating must be between 1 and 5")
	}

	feedbackReq := &evaluation.FeedbackRequest{
		TraceID: req.Body.TraceID,
		Rating:  req.Body.Rating,
		Comment: req.Body.Comment,
		Tags:    req.Body.Tags,
	}

	if err := h.evalSvc.CollectFeedback(ctx, feedbackReq); err != nil {
		return nil, huma.Error500InternalServerError("failed to submit feedback")
	}

	resp := &SubmitFeedbackResponse{}
	resp.Body.Success = true
	resp.Body.Message = "Feedback submitted successfully"
	return resp, nil
}

// GetFeedbackRequest defines the request for getting feedback.
type GetFeedbackRequest struct {
	TraceID string `path:"traceId" doc:"Trace ID"`
}

// GetFeedbackResponse defines the response for getting feedback.
type GetFeedbackResponse struct {
	Body struct {
		Feedback []*evaluation.Feedback `json:"feedback"`
	}
}

// GetFeedback handles GET /v1/evaluation/traces/:traceId/feedback
func (h *EvaluationHandler) GetFeedback(ctx context.Context, req *GetFeedbackRequest) (*GetFeedbackResponse, error) {
	feedback, err := h.evalSvc.GetFeedback(ctx, req.TraceID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get feedback")
	}

	resp := &GetFeedbackResponse{}
	resp.Body.Feedback = feedback
	return resp, nil
}
