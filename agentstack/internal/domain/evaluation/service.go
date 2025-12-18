// Package evaluation provides the evaluation service for agent interaction tracing and quality assessment.
package evaluation

import (
	"context"
	"encoding/json"
	"time"
)

// Service handles evaluation operations including tracing and feedback.
type Service struct {
	mlflow    MLflowClient
	queue     Queue
	repo      Repository
	idGen     IDGenerator
}

// MLflowClient defines the interface for MLflow operations.
type MLflowClient interface {
	GetOrCreateExperiment(ctx context.Context, name string) (string, error)
	StartRun(ctx context.Context, experimentID, runName string, tags map[string]string) (MLflowRun, error)
}

// MLflowRun represents an active MLflow run.
type MLflowRun interface {
	LogParam(ctx context.Context, key, value string) error
	LogMetric(ctx context.Context, key string, value float64, step int64) error
	LogBatch(ctx context.Context, params map[string]string, metrics map[string]float64) error
	End(ctx context.Context, status string) error
}

// Queue defines the interface for async message queueing.
type Queue interface {
	XAdd(ctx context.Context, stream string, data []byte) error
}

// Repository defines the data access interface for evaluation.
type Repository interface {
	SaveTrace(ctx context.Context, trace *Trace) error
	GetTrace(ctx context.Context, id string) (*Trace, error)
	ListTraces(ctx context.Context, filter TraceFilter) ([]*Trace, error)
	SaveFeedback(ctx context.Context, feedback *Feedback) error
	GetFeedback(ctx context.Context, traceID string) ([]*Feedback, error)
}

// IDGenerator generates unique IDs.
type IDGenerator interface {
	Generate(prefix string) string
}

// NewService creates a new evaluation service.
func NewService(mlflow MLflowClient, queue Queue, repo Repository, idGen IDGenerator) *Service {
	return &Service{
		mlflow: mlflow,
		queue:  queue,
		repo:   repo,
		idGen:  idGen,
	}
}

// TraceInteraction records an agent interaction for evaluation.
func (s *Service) TraceInteraction(ctx context.Context, interaction *Interaction) (*Trace, error) {
	trace := &Trace{
		ID:        s.idGen.Generate("trc"),
		AgentID:   interaction.AgentID,
		SessionID: interaction.SessionID,
		Input:     interaction.Input,
		Output:    interaction.Output,
		Latency:   interaction.Latency,
		TokensIn:  interaction.TokensIn,
		TokensOut: interaction.TokensOut,
		Steps:     interaction.Steps,
		Metadata:  interaction.Metadata,
		CreatedAt: time.Now().UTC(),
	}

	// Save trace to repository
	if err := s.repo.SaveTrace(ctx, trace); err != nil {
		return nil, err
	}

	// Queue for async evaluation (non-blocking)
	go func() {
		_ = s.queueForEvaluation(context.Background(), trace)
	}()

	return trace, nil
}

func (s *Service) queueForEvaluation(ctx context.Context, trace *Trace) error {
	if s.queue == nil {
		return nil
	}
	data, err := json.Marshal(trace)
	if err != nil {
		return err
	}
	return s.queue.XAdd(ctx, "evaluation:queue", data)
}

// RunEvaluation runs evaluation on a trace (called by worker).
func (s *Service) RunEvaluation(ctx context.Context, trace *Trace) (*EvalResult, error) {
	if s.mlflow == nil {
		// Return default result if MLflow not configured
		return &EvalResult{
			TraceID: trace.ID,
			Metrics: s.calculateMetrics(trace),
		}, nil
	}

	// Get or create experiment for the agent
	expID, err := s.mlflow.GetOrCreateExperiment(ctx, "agent-"+trace.AgentID)
	if err != nil {
		return nil, err
	}

	// Start MLflow run
	run, err := s.mlflow.StartRun(ctx, expID, trace.ID, map[string]string{
		"agent_id":   trace.AgentID,
		"session_id": trace.SessionID,
		"trace_id":   trace.ID,
	})
	if err != nil {
		return nil, err
	}

	// Log parameters
	_ = run.LogParam(ctx, "input", truncate(trace.Input, 250))
	_ = run.LogParam(ctx, "output", truncate(trace.Output, 250))

	// Calculate metrics
	metrics := s.calculateMetrics(trace)

	// Log metrics to MLflow
	err = run.LogBatch(ctx, nil, map[string]float64{
		"latency_ms":      float64(trace.Latency.Milliseconds()),
		"tokens_in":       float64(trace.TokensIn),
		"tokens_out":      float64(trace.TokensOut),
		"step_count":      float64(len(trace.Steps)),
		"coherence_score": metrics.CoherenceScore,
		"relevance_score": metrics.RelevanceScore,
		"safety_score":    metrics.SafetyScore,
	})
	if err != nil {
		_ = run.End(ctx, "FAILED")
		return nil, err
	}

	// End run successfully
	_ = run.End(ctx, "FINISHED")

	return &EvalResult{
		TraceID: trace.ID,
		RunID:   "",
		Metrics: metrics,
	}, nil
}

func (s *Service) calculateMetrics(trace *Trace) Metrics {
	// Basic heuristic metrics (can be enhanced with LLM-as-judge)
	metrics := Metrics{
		CoherenceScore: 0.85,
		RelevanceScore: 0.90,
		SafetyScore:    1.0,
	}

	// Adjust based on trace characteristics
	if trace.Output == "" {
		metrics.CoherenceScore = 0.0
		metrics.RelevanceScore = 0.0
	}

	// Penalize very long latencies
	if trace.Latency > 30*time.Second {
		metrics.CoherenceScore *= 0.8
	}

	return metrics
}

// GetTrace retrieves a trace by ID.
func (s *Service) GetTrace(ctx context.Context, id string) (*Trace, error) {
	return s.repo.GetTrace(ctx, id)
}

// ListTraces lists traces matching the filter.
func (s *Service) ListTraces(ctx context.Context, filter TraceFilter) ([]*Trace, error) {
	return s.repo.ListTraces(ctx, filter)
}

// CollectFeedback records user feedback for a trace.
func (s *Service) CollectFeedback(ctx context.Context, req *FeedbackRequest) error {
	feedback := &Feedback{
		ID:        s.idGen.Generate("fb"),
		TraceID:   req.TraceID,
		Rating:    req.Rating,
		Comment:   req.Comment,
		Tags:      req.Tags,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.SaveFeedback(ctx, feedback); err != nil {
		return err
	}

	// Update MLflow run with feedback asynchronously
	go s.updateRunWithFeedback(context.Background(), feedback)

	return nil
}

func (s *Service) updateRunWithFeedback(ctx context.Context, fb *Feedback) {
	// In production, find the MLflow run by trace ID and log feedback
	// This is a placeholder for async feedback logging
}

// GetFeedback retrieves feedback for a trace.
func (s *Service) GetFeedback(ctx context.Context, traceID string) ([]*Feedback, error) {
	return s.repo.GetFeedback(ctx, traceID)
}

// --- Types ---

// Trace represents a traced agent interaction.
type Trace struct {
	ID        string                 `json:"id"`
	AgentID   string                 `json:"agent_id"`
	SessionID string                 `json:"session_id"`
	Input     string                 `json:"input"`
	Output    string                 `json:"output"`
	Latency   time.Duration          `json:"latency"`
	TokensIn  int                    `json:"tokens_in"`
	TokensOut int                    `json:"tokens_out"`
	Steps     []Step                 `json:"steps,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// Step represents a step in an agent's execution.
type Step struct {
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Input    interface{}            `json:"input,omitempty"`
	Output   interface{}            `json:"output,omitempty"`
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Interaction represents input for tracing an agent interaction.
type Interaction struct {
	AgentID   string
	SessionID string
	Input     string
	Output    string
	Latency   time.Duration
	TokensIn  int
	TokensOut int
	Steps     []Step
	Metadata  map[string]interface{}
}

// TraceFilter defines filters for listing traces.
type TraceFilter struct {
	AgentID   string
	SessionID string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// Feedback represents user feedback on an interaction.
type Feedback struct {
	ID        string    `json:"id"`
	TraceID   string    `json:"trace_id"`
	Rating    int       `json:"rating"` // 1-5
	Comment   string    `json:"comment,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// FeedbackRequest represents a request to submit feedback.
type FeedbackRequest struct {
	TraceID string   `json:"trace_id" validate:"required"`
	Rating  int      `json:"rating" validate:"required,min=1,max=5"`
	Comment string   `json:"comment,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

// EvalResult represents the result of an evaluation.
type EvalResult struct {
	TraceID string  `json:"trace_id"`
	RunID   string  `json:"run_id,omitempty"`
	Metrics Metrics `json:"metrics"`
}

// Metrics represents evaluation metrics.
type Metrics struct {
	CoherenceScore float64 `json:"coherence_score"`
	RelevanceScore float64 `json:"relevance_score"`
	SafetyScore    float64 `json:"safety_score"`
}

// Helper functions

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
