# Phase 3: Evaluation & Safety Framework

> MLflow Integration, Safety Gates, Quality Assurance

**Duration**: 2 weeks | **Status**: Not Started | **Priority**: High  
**Depends On**: Phase 2 (Agent Runtime)

---

## Objectives

1. Integrate MLflow 3.x for experiment tracking and evaluation
2. Implement safety gateway with content filtering
3. Build evaluation pipeline (offline and online)
4. Create feedback collection system
5. Implement quality metrics and dashboards

---

## Architecture Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                      Evaluation & Safety Architecture                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   Request Flow                                                          │
│   ┌─────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────┐     │
│   │ Client  │───▶│   Gateway   │───▶│ Safety Gate │───▶│  Agent  │     │
│   │         │    │             │    │  (Pre-exec) │    │         │     │
│   └─────────┘    └─────────────┘    └─────────────┘    └────┬────┘     │
│                                                              │          │
│   Response Flow                                              ▼          │
│   ┌─────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────┐     │
│   │ Client  │◀───│   Gateway   │◀───│ Safety Gate │◀───│Response │     │
│   │         │    │             │    │ (Post-exec) │    │         │     │
│   └─────────┘    └─────────────┘    └─────────────┘    └─────────┘     │
│                                           │                             │
│                                           ▼                             │
│   ┌───────────────────────────────────────────────────────────────┐    │
│   │                     Evaluation Pipeline                        │    │
│   │                                                                │    │
│   │  ┌──────────┐  ┌──────────────┐  ┌───────────┐  ┌──────────┐  │    │
│   │  │  Logger  │─▶│ Async Queue  │─▶│  MLflow   │─▶│ Metrics  │  │    │
│   │  │          │  │ (Redis)      │  │ Evaluator │  │Dashboard │  │    │
│   │  └──────────┘  └──────────────┘  └───────────┘  └──────────┘  │    │
│   │                                                                │    │
│   └───────────────────────────────────────────────────────────────┘    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Week 1: MLflow Integration

### 1.1 MLflow Client (Go)

Since MLflow is Python-based, we'll interact via its REST API:

**`internal/infrastructure/mlflow/client.go`**:
```go
package mlflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps MLflow REST API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates an MLflow client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateExperiment creates a new MLflow experiment
func (c *Client) CreateExperiment(ctx context.Context, name string, tags map[string]string) (string, error) {
	req := map[string]interface{}{
		"name": name,
		"tags": tagsToList(tags),
	}

	resp, err := c.post(ctx, "/api/2.0/mlflow/experiments/create", req)
	if err != nil {
		return "", err
	}

	return resp["experiment_id"].(string), nil
}

// GetOrCreateExperiment gets or creates an experiment
func (c *Client) GetOrCreateExperiment(ctx context.Context, name string) (string, error) {
	// Try to get existing
	resp, err := c.get(ctx, "/api/2.0/mlflow/experiments/get-by-name", map[string]string{
		"experiment_name": name,
	})
	if err == nil {
		exp := resp["experiment"].(map[string]interface{})
		return exp["experiment_id"].(string), nil
	}

	// Create new
	return c.CreateExperiment(ctx, name, nil)
}

// StartRun creates a new run in an experiment
func (c *Client) StartRun(ctx context.Context, experimentID string, runName string, tags map[string]string) (*Run, error) {
	req := map[string]interface{}{
		"experiment_id": experimentID,
		"run_name":      runName,
		"start_time":    time.Now().UnixMilli(),
		"tags":          tagsToList(tags),
	}

	resp, err := c.post(ctx, "/api/2.0/mlflow/runs/create", req)
	if err != nil {
		return nil, err
	}

	runInfo := resp["run"].(map[string]interface{})["info"].(map[string]interface{})
	return &Run{
		RunID:        runInfo["run_id"].(string),
		ExperimentID: experimentID,
		client:       c,
	}, nil
}

// Run represents an MLflow run
type Run struct {
	RunID        string
	ExperimentID string
	client       *Client
}

// LogParam logs a parameter
func (r *Run) LogParam(ctx context.Context, key, value string) error {
	req := map[string]interface{}{
		"run_id": r.RunID,
		"key":    key,
		"value":  value,
	}
	_, err := r.client.post(ctx, "/api/2.0/mlflow/runs/log-parameter", req)
	return err
}

// LogMetric logs a metric
func (r *Run) LogMetric(ctx context.Context, key string, value float64, step int64) error {
	req := map[string]interface{}{
		"run_id":    r.RunID,
		"key":       key,
		"value":     value,
		"timestamp": time.Now().UnixMilli(),
		"step":      step,
	}
	_, err := r.client.post(ctx, "/api/2.0/mlflow/runs/log-metric", req)
	return err
}

// LogBatch logs multiple params and metrics
func (r *Run) LogBatch(ctx context.Context, params map[string]string, metrics map[string]float64) error {
	paramList := make([]map[string]string, 0, len(params))
	for k, v := range params {
		paramList = append(paramList, map[string]string{"key": k, "value": v})
	}

	metricList := make([]map[string]interface{}, 0, len(metrics))
	ts := time.Now().UnixMilli()
	for k, v := range metrics {
		metricList = append(metricList, map[string]interface{}{
			"key":       k,
			"value":     v,
			"timestamp": ts,
			"step":      0,
		})
	}

	req := map[string]interface{}{
		"run_id":  r.RunID,
		"params":  paramList,
		"metrics": metricList,
	}
	_, err := r.client.post(ctx, "/api/2.0/mlflow/runs/log-batch", req)
	return err
}

// End ends the run
func (r *Run) End(ctx context.Context, status string) error {
	req := map[string]interface{}{
		"run_id":   r.RunID,
		"status":   status, // FINISHED, FAILED, KILLED
		"end_time": time.Now().UnixMilli(),
	}
	_, err := r.client.post(ctx, "/api/2.0/mlflow/runs/update", req)
	return err
}

// HTTP helpers
func (c *Client) get(ctx context.Context, path string, params map[string]string) (map[string]interface{}, error) {
	url := c.baseURL + path
	if len(params) > 0 {
		url += "?"
		for k, v := range params {
			url += k + "=" + v + "&"
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

func (c *Client) post(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return c.do(req)
}

func (c *Client) do(req *http.Request) (map[string]interface{}, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MLflow API error: %s - %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func tagsToList(tags map[string]string) []map[string]string {
	list := make([]map[string]string, 0, len(tags))
	for k, v := range tags {
		list = append(list, map[string]string{"key": k, "value": v})
	}
	return list
}
```

### 1.2 Evaluation Service

**`internal/domain/evaluation/service.go`**:
```go
package evaluation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/mlflow"
	"github.com/raphaelmansuy/agentstack/internal/pkg/id"
)

type Service struct {
	mlflow    *mlflow.Client
	queue     *cache.RedisClient
	repo      Repository
}

type Repository interface {
	SaveTrace(ctx context.Context, trace *Trace) error
	GetTrace(ctx context.Context, id string) (*Trace, error)
	ListTraces(ctx context.Context, filter TraceFilter) ([]*Trace, error)
	SaveFeedback(ctx context.Context, feedback *Feedback) error
}

func NewService(mlflow *mlflow.Client, queue *cache.RedisClient, repo Repository) *Service {
	return &Service{
		mlflow: mlflow,
		queue:  queue,
		repo:   repo,
	}
}

// TraceInteraction records an agent interaction for evaluation
func (s *Service) TraceInteraction(ctx context.Context, interaction *Interaction) (*Trace, error) {
	trace := &Trace{
		ID:          id.Generate("trc"),
		AgentID:     interaction.AgentID,
		SessionID:   interaction.SessionID,
		Input:       interaction.Input,
		Output:      interaction.Output,
		Latency:     interaction.Latency,
		TokensIn:    interaction.TokensIn,
		TokensOut:   interaction.TokensOut,
		Steps:       interaction.Steps,
		CreatedAt:   time.Now(),
	}

	// Save trace
	if err := s.repo.SaveTrace(ctx, trace); err != nil {
		return nil, err
	}

	// Queue for async evaluation
	if err := s.queueForEvaluation(ctx, trace); err != nil {
		// Log but don't fail
	}

	return trace, nil
}

func (s *Service) queueForEvaluation(ctx context.Context, trace *Trace) error {
	data, _ := json.Marshal(trace)
	return s.queue.XAdd(ctx, "evaluation:queue", data)
}

// RunEvaluation runs evaluation on a trace (called by worker)
func (s *Service) RunEvaluation(ctx context.Context, trace *Trace) (*EvalResult, error) {
	// Get or create experiment
	expID, err := s.mlflow.GetOrCreateExperiment(ctx, "agent-"+trace.AgentID)
	if err != nil {
		return nil, err
	}

	// Start run
	run, err := s.mlflow.StartRun(ctx, expID, trace.ID, map[string]string{
		"agent_id":   trace.AgentID,
		"session_id": trace.SessionID,
		"trace_id":   trace.ID,
	})
	if err != nil {
		return nil, err
	}

	// Log parameters
	run.LogParam(ctx, "input", truncate(trace.Input, 250))
	run.LogParam(ctx, "output", truncate(trace.Output, 250))

	// Calculate metrics
	metrics := s.calculateMetrics(trace)

	// Log metrics
	run.LogBatch(ctx, nil, map[string]float64{
		"latency_ms":       float64(trace.Latency.Milliseconds()),
		"tokens_in":        float64(trace.TokensIn),
		"tokens_out":       float64(trace.TokensOut),
		"step_count":       float64(len(trace.Steps)),
		"coherence_score":  metrics.CoherenceScore,
		"relevance_score":  metrics.RelevanceScore,
		"safety_score":     metrics.SafetyScore,
	})

	// End run
	run.End(ctx, "FINISHED")

	return &EvalResult{
		TraceID: trace.ID,
		RunID:   run.RunID,
		Metrics: metrics,
	}, nil
}

func (s *Service) calculateMetrics(trace *Trace) Metrics {
	// Placeholder: In production, use LLM-as-judge or heuristics
	return Metrics{
		CoherenceScore: 0.85,
		RelevanceScore: 0.90,
		SafetyScore:    1.0,
	}
}

// CollectFeedback records user feedback
func (s *Service) CollectFeedback(ctx context.Context, fb *FeedbackRequest) error {
	feedback := &Feedback{
		ID:        id.Generate("fb"),
		TraceID:   fb.TraceID,
		Rating:    fb.Rating,
		Comment:   fb.Comment,
		Tags:      fb.Tags,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveFeedback(ctx, feedback); err != nil {
		return err
	}

	// Update MLflow run with feedback
	go s.updateRunWithFeedback(context.Background(), feedback)

	return nil
}

func (s *Service) updateRunWithFeedback(ctx context.Context, fb *Feedback) {
	// Find run by trace ID and log feedback metrics
	// ...
}

// Types

type Trace struct {
	ID        string
	AgentID   string
	SessionID string
	Input     string
	Output    string
	Latency   time.Duration
	TokensIn  int
	TokensOut int
	Steps     []Step
	CreatedAt time.Time
}

type Step struct {
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Input    interface{}            `json:"input"`
	Output   interface{}            `json:"output"`
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Interaction struct {
	AgentID   string
	SessionID string
	Input     string
	Output    string
	Latency   time.Duration
	TokensIn  int
	TokensOut int
	Steps     []Step
}

type TraceFilter struct {
	AgentID   string
	SessionID string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

type Feedback struct {
	ID        string
	TraceID   string
	Rating    int      // 1-5
	Comment   string
	Tags      []string
	CreatedAt time.Time
}

type FeedbackRequest struct {
	TraceID string   `json:"trace_id"`
	Rating  int      `json:"rating"`
	Comment string   `json:"comment,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

type EvalResult struct {
	TraceID string
	RunID   string
	Metrics Metrics
}

type Metrics struct {
	CoherenceScore float64
	RelevanceScore float64
	SafetyScore    float64
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
```

### 1.3 Evaluation Worker

**`cmd/eval-worker/main.go`**:
```go
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/mlflow"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/db"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize dependencies
	redisClient := cache.NewRedisClient(os.Getenv("REDIS_URL"))
	defer redisClient.Close()

	database := db.Connect(os.Getenv("DATABASE_URL"))
	defer database.Close()

	mlflowClient := mlflow.NewClient(os.Getenv("MLFLOW_URL"))

	evalRepo := evaluation.NewPostgresRepository(database)
	evalSvc := evaluation.NewService(mlflowClient, redisClient, evalRepo)

	// Start consumer
	go consumeEvalQueue(ctx, redisClient, evalSvc)

	// Wait for shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down evaluation worker...")
	cancel()
}

func consumeEvalQueue(ctx context.Context, redis *cache.RedisClient, evalSvc *evaluation.Service) {
	consumer := "eval-worker-" + os.Getenv("HOSTNAME")
	group := "eval-workers"

	// Create consumer group if not exists
	redis.XGroupCreateMkStream(ctx, "evaluation:queue", group, "0")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read from stream
			messages, err := redis.XReadGroup(ctx, group, consumer, "evaluation:queue", ">", 10)
			if err != nil {
				log.Printf("Error reading from queue: %v", err)
				continue
			}

			for _, msg := range messages {
				var trace evaluation.Trace
				if err := json.Unmarshal([]byte(msg.Values["data"].(string)), &trace); err != nil {
					log.Printf("Error unmarshaling trace: %v", err)
					redis.XAck(ctx, "evaluation:queue", group, msg.ID)
					continue
				}

				// Run evaluation
				result, err := evalSvc.RunEvaluation(ctx, &trace)
				if err != nil {
					log.Printf("Error evaluating trace %s: %v", trace.ID, err)
				} else {
					log.Printf("Evaluated trace %s, run %s", trace.ID, result.RunID)
				}

				// Acknowledge message
				redis.XAck(ctx, "evaluation:queue", group, msg.ID)
			}
		}
	}
}
```

---

## Week 2: Safety Gates

### 2.1 Safety Service

**`internal/domain/safety/service.go`**:
```go
package safety

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type Service struct {
	config     Config
	moderator  Moderator
}

type Moderator interface {
	Classify(ctx context.Context, text string) (*Classification, error)
}

type Config struct {
	Enabled              bool
	BlockedPatterns      []string
	MaxInputLength       int
	MaxOutputLength      int
	RequireModeration    bool
	ModerationThresholds Thresholds
}

type Thresholds struct {
	Hate          float64
	Violence      float64
	Sexual        float64
	SelfHarm      float64
	HateWithThreat float64
	ViolenceGraphic float64
}

func NewService(config Config, moderator Moderator) *Service {
	return &Service{
		config:    config,
		moderator: moderator,
	}
}

// CheckInput validates user input before sending to agent
func (s *Service) CheckInput(ctx context.Context, input string) (*CheckResult, error) {
	if !s.config.Enabled {
		return &CheckResult{Allowed: true}, nil
	}

	result := &CheckResult{Allowed: true, Checks: []Check{}}

	// Length check
	if len(input) > s.config.MaxInputLength {
		result.Allowed = false
		result.Checks = append(result.Checks, Check{
			Name:    "length",
			Passed:  false,
			Reason:  fmt.Sprintf("Input exceeds maximum length of %d", s.config.MaxInputLength),
		})
		return result, nil
	}
	result.Checks = append(result.Checks, Check{Name: "length", Passed: true})

	// Pattern check
	for _, pattern := range s.config.BlockedPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(input) {
			result.Allowed = false
			result.Checks = append(result.Checks, Check{
				Name:   "pattern",
				Passed: false,
				Reason: "Input contains blocked pattern",
			})
			return result, nil
		}
	}
	result.Checks = append(result.Checks, Check{Name: "pattern", Passed: true})

	// Moderation check (if enabled)
	if s.config.RequireModeration && s.moderator != nil {
		classification, err := s.moderator.Classify(ctx, input)
		if err != nil {
			// Log error but don't block on moderation failure
			result.Checks = append(result.Checks, Check{
				Name:   "moderation",
				Passed: true,
				Reason: "Moderation service unavailable, allowing by default",
			})
		} else {
			check := s.evaluateClassification(classification)
			result.Checks = append(result.Checks, check)
			if !check.Passed {
				result.Allowed = false
			}
		}
	}

	return result, nil
}

// CheckOutput validates agent output before sending to user
func (s *Service) CheckOutput(ctx context.Context, output string) (*CheckResult, error) {
	if !s.config.Enabled {
		return &CheckResult{Allowed: true}, nil
	}

	result := &CheckResult{Allowed: true, Checks: []Check{}}

	// Length check
	if len(output) > s.config.MaxOutputLength {
		// Truncate instead of blocking
		result.Checks = append(result.Checks, Check{
			Name:    "length",
			Passed:  true,
			Reason:  "Output truncated to max length",
		})
	}
	result.Checks = append(result.Checks, Check{Name: "length", Passed: true})

	// Moderation check for output
	if s.config.RequireModeration && s.moderator != nil {
		classification, err := s.moderator.Classify(ctx, output)
		if err != nil {
			result.Checks = append(result.Checks, Check{
				Name:   "moderation",
				Passed: true,
				Reason: "Moderation service unavailable",
			})
		} else {
			check := s.evaluateClassification(classification)
			result.Checks = append(result.Checks, check)
			if !check.Passed {
				result.Allowed = false
			}
		}
	}

	return result, nil
}

func (s *Service) evaluateClassification(c *Classification) Check {
	check := Check{Name: "moderation", Passed: true}

	// Check each category against thresholds
	if c.Hate >= s.config.ModerationThresholds.Hate {
		check.Passed = false
		check.Reason = "Content flagged for hate"
	} else if c.Violence >= s.config.ModerationThresholds.Violence {
		check.Passed = false
		check.Reason = "Content flagged for violence"
	} else if c.Sexual >= s.config.ModerationThresholds.Sexual {
		check.Passed = false
		check.Reason = "Content flagged for sexual content"
	} else if c.SelfHarm >= s.config.ModerationThresholds.SelfHarm {
		check.Passed = false
		check.Reason = "Content flagged for self-harm"
	}

	return check
}

// Types

type CheckResult struct {
	Allowed bool    `json:"allowed"`
	Checks  []Check `json:"checks"`
}

type Check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Reason string `json:"reason,omitempty"`
}

type Classification struct {
	Hate            float64 `json:"hate"`
	Violence        float64 `json:"violence"`
	Sexual          float64 `json:"sexual"`
	SelfHarm        float64 `json:"self_harm"`
	HateWithThreat  float64 `json:"hate_threatening"`
	ViolenceGraphic float64 `json:"violence_graphic"`
}
```

### 2.2 OpenAI Moderation Client

**`internal/infrastructure/moderation/openai.go`**:
```go
package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/safety"
)

type OpenAIModerator struct {
	apiKey     string
	httpClient *http.Client
}

func NewOpenAIModerator(apiKey string) *OpenAIModerator {
	return &OpenAIModerator{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (m *OpenAIModerator) Classify(ctx context.Context, text string) (*safety.Classification, error) {
	req := map[string]interface{}{
		"input": text,
	}

	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://api.openai.com/v1/moderations",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("moderation API returned %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			CategoryScores struct {
				Hate            float64 `json:"hate"`
				HateThreatening float64 `json:"hate/threatening"`
				SelfHarm        float64 `json:"self-harm"`
				Sexual          float64 `json:"sexual"`
				SexualMinors    float64 `json:"sexual/minors"`
				Violence        float64 `json:"violence"`
				ViolenceGraphic float64 `json:"violence/graphic"`
			} `json:"category_scores"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no moderation results")
	}

	scores := result.Results[0].CategoryScores

	return &safety.Classification{
		Hate:            scores.Hate,
		HateWithThreat:  scores.HateThreatening,
		SelfHarm:        scores.SelfHarm,
		Sexual:          scores.Sexual,
		Violence:        scores.Violence,
		ViolenceGraphic: scores.ViolenceGraphic,
	}, nil
}
```

### 2.3 Safety Middleware

**`internal/api/middleware/safety.go`**:
```go
package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/raphaelmansuy/agentstack/internal/domain/safety"
	"github.com/raphaelmansuy/agentstack/internal/pkg/logger"
)

type SafetyMiddleware struct {
	safetySvc *safety.Service
	log       *logger.Logger
}

func NewSafetyMiddleware(safetySvc *safety.Service, log *logger.Logger) *SafetyMiddleware {
	return &SafetyMiddleware{
		safetySvc: safetySvc,
		log:       log,
	}
}

// PreExecution checks input before agent execution
func (m *SafetyMiddleware) PreExecution(c *fiber.Ctx) error {
	// Extract input from request
	var body struct {
		Message interface{} `json:"message"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Next()
	}

	// Convert message to string for safety check
	input := extractTextFromMessage(body.Message)
	if input == "" {
		return c.Next()
	}

	// Check input
	result, err := m.safetySvc.CheckInput(c.UserContext(), input)
	if err != nil {
		m.log.Error("Safety check failed", "error", err)
		return c.Next() // Fail open
	}

	if !result.Allowed {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "content_blocked",
			"reason": result.Checks[len(result.Checks)-1].Reason,
		})
	}

	// Store result for post-processing
	c.Locals("safety_pre_check", result)

	return c.Next()
}

// PostExecution checks output after agent execution
// This is more complex as we need to intercept the response
// For streaming, use the SafetyTransformer instead

func extractTextFromMessage(msg interface{}) string {
	switch v := msg.(type) {
	case string:
		return v
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok {
			return text
		}
		if content, ok := v["content"].(string); ok {
			return content
		}
	}
	return ""
}
```

### 2.4 Evaluation API Endpoints

**`internal/api/handlers/evaluation.go`**:
```go
package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
)

type EvaluationHandler struct {
	evalSvc *evaluation.Service
}

func NewEvaluationHandler(evalSvc *evaluation.Service) *EvaluationHandler {
	return &EvaluationHandler{evalSvc: evalSvc}
}

// ListTraces returns traces for an agent
// GET /v1/evaluation/traces
func (h *EvaluationHandler) ListTraces(c *fiber.Ctx) error {
	auth := middleware.GetAuthContext(c)

	filter := evaluation.TraceFilter{
		AgentID: c.Query("agent_id"),
		Limit:   c.QueryInt("limit", 50),
	}

	// Validate agent belongs to project
	// ...

	traces, err := h.evalSvc.ListTraces(c.UserContext(), filter)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"traces": traces,
	})
}

// GetTrace returns a single trace
// GET /v1/evaluation/traces/:traceId
func (h *EvaluationHandler) GetTrace(c *fiber.Ctx) error {
	traceID := c.Params("traceId")

	trace, err := h.evalSvc.GetTrace(c.UserContext(), traceID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "trace not found")
	}

	return c.JSON(trace)
}

// SubmitFeedback records user feedback
// POST /v1/evaluation/feedback
func (h *EvaluationHandler) SubmitFeedback(c *fiber.Ctx) error {
	var req evaluation.FeedbackRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}

	if req.Rating < 1 || req.Rating > 5 {
		return fiber.NewError(fiber.StatusBadRequest, "rating must be 1-5")
	}

	if err := h.evalSvc.CollectFeedback(c.UserContext(), &req); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}
```

---

## Deployment Configuration

### MLflow Kubernetes Deployment

**`deploy/mlflow/deployment.yaml`**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mlflow
  namespace: agentstack
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mlflow
  template:
    metadata:
      labels:
        app: mlflow
    spec:
      containers:
        - name: mlflow
          image: ghcr.io/mlflow/mlflow:v2.18.0
          args:
            - server
            - --host=0.0.0.0
            - --port=5000
            - --backend-store-uri=postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/mlflow
            - --default-artifact-root=s3://agentstack-mlflow/artifacts
          env:
            - name: POSTGRES_USER
              valueFrom:
                secretKeyRef:
                  name: mlflow-secrets
                  key: postgres-user
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: mlflow-secrets
                  key: postgres-password
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: mlflow-secrets
                  key: aws-access-key
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: mlflow-secrets
                  key: aws-secret-key
          ports:
            - containerPort: 5000
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi
---
apiVersion: v1
kind: Service
metadata:
  name: mlflow
  namespace: agentstack
spec:
  selector:
    app: mlflow
  ports:
    - port: 5000
      targetPort: 5000
```

---

## Deliverables Checklist

### Week 1
- [ ] MLflow Go client implementation
- [ ] Evaluation service with trace logging
- [ ] Evaluation worker for async processing
- [ ] Feedback collection API
- [ ] MLflow deployment in Kubernetes

### Week 2
- [ ] Safety service with moderation
- [ ] OpenAI moderation integration
- [ ] Safety middleware for pre/post checks
- [ ] Evaluation API endpoints
- [ ] Integration tests for safety gates

---

## Definition of Done

- [ ] Traces logged to MLflow with metrics
- [ ] Feedback can be collected and linked to traces
- [ ] Safety gate blocks harmful content
- [ ] Evaluation worker processes traces async
- [ ] Safety checks don't add >50ms latency

---

## Sage AI Guidance

### Key Design Decisions

1. **MLflow via REST API**: Since MLflow is Python, use REST API from Go. No need for gRPC.

2. **Async Evaluation**: Use Redis Streams for async evaluation. Don't block the request path.

3. **Fail Open for Safety**: If moderation service is down, allow requests but log. Don't block users.

4. **OpenAI Moderation**: Start with OpenAI's moderation API. It's fast and accurate. Swap later if needed.

### Metrics to Track

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Safety gate latency | <50ms p95 | >100ms |
| Moderation API success rate | >99.9% | <99% |
| Evaluation worker lag | <1min | >5min |
| Feedback submission rate | >1% of interactions | - |

### Testing Strategy

```text
1. Unit test safety patterns
2. Integration test MLflow client
3. Load test safety middleware (10k req/s)
4. E2E test feedback loop
5. Chaos test: moderation service down
```

---

**Next Phase**: [004-developer-experience.md](004-developer-experience.md) - CLI, SDK, Documentation
