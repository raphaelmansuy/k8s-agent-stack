package evaluation

import (
	"context"
	"testing"
	"time"
)

// Mock implementations for testing

type mockMLflowClient struct {
	experimentID string
	runs         map[string]*mockRun
}

func (m *mockMLflowClient) GetOrCreateExperiment(ctx context.Context, name string) (string, error) {
	return m.experimentID, nil
}

func (m *mockMLflowClient) StartRun(ctx context.Context, experimentID, runName string, tags map[string]string) (MLflowRun, error) {
	run := &mockRun{runID: "mock-run-" + runName}
	if m.runs == nil {
		m.runs = make(map[string]*mockRun)
	}
	m.runs[runName] = run
	return run, nil
}

type mockRun struct {
	runID   string
	params  map[string]string
	metrics map[string]float64
	ended   bool
	status  string
}

func (r *mockRun) LogParam(ctx context.Context, key, value string) error {
	if r.params == nil {
		r.params = make(map[string]string)
	}
	r.params[key] = value
	return nil
}

func (r *mockRun) LogMetric(ctx context.Context, key string, value float64, step int64) error {
	if r.metrics == nil {
		r.metrics = make(map[string]float64)
	}
	r.metrics[key] = value
	return nil
}

func (r *mockRun) LogBatch(ctx context.Context, params map[string]string, metrics map[string]float64) error {
	if r.params == nil {
		r.params = make(map[string]string)
	}
	if r.metrics == nil {
		r.metrics = make(map[string]float64)
	}
	for k, v := range params {
		r.params[k] = v
	}
	for k, v := range metrics {
		r.metrics[k] = v
	}
	return nil
}

func (r *mockRun) End(ctx context.Context, status string) error {
	r.ended = true
	r.status = status
	return nil
}

type mockQueue struct {
	messages [][]byte
}

func (q *mockQueue) XAdd(ctx context.Context, stream string, data []byte) error {
	q.messages = append(q.messages, data)
	return nil
}

type mockRepository struct {
	traces   map[string]*Trace
	feedback map[string][]*Feedback
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		traces:   make(map[string]*Trace),
		feedback: make(map[string][]*Feedback),
	}
}

func (r *mockRepository) SaveTrace(ctx context.Context, trace *Trace) error {
	r.traces[trace.ID] = trace
	return nil
}

func (r *mockRepository) GetTrace(ctx context.Context, id string) (*Trace, error) {
	if trace, ok := r.traces[id]; ok {
		return trace, nil
	}
	return nil, nil
}

func (r *mockRepository) ListTraces(ctx context.Context, filter TraceFilter) ([]*Trace, error) {
	var result []*Trace
	for _, trace := range r.traces {
		if filter.AgentID != "" && trace.AgentID != filter.AgentID {
			continue
		}
		result = append(result, trace)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	return result, nil
}

func (r *mockRepository) SaveFeedback(ctx context.Context, feedback *Feedback) error {
	r.feedback[feedback.TraceID] = append(r.feedback[feedback.TraceID], feedback)
	return nil
}

func (r *mockRepository) GetFeedback(ctx context.Context, traceID string) ([]*Feedback, error) {
	return r.feedback[traceID], nil
}

type mockIDGenerator struct {
	counter int
}

func (g *mockIDGenerator) Generate(prefix string) string {
	g.counter++
	return prefix + "-test-" + string(rune('0'+g.counter))
}

// Tests

func TestNewService(t *testing.T) {
	mlflow := &mockMLflowClient{experimentID: "exp-123"}
	queue := &mockQueue{}
	repo := newMockRepository()
	idGen := &mockIDGenerator{}

	svc := NewService(mlflow, queue, repo, idGen)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestTraceInteraction(t *testing.T) {
	mlflow := &mockMLflowClient{experimentID: "exp-123"}
	queue := &mockQueue{}
	repo := newMockRepository()
	idGen := &mockIDGenerator{}

	svc := NewService(mlflow, queue, repo, idGen)

	interaction := &Interaction{
		AgentID:   "agent-1",
		SessionID: "session-1",
		Input:     "Hello, world!",
		Output:    "Hi there!",
		Latency:   100 * time.Millisecond,
		TokensIn:  10,
		TokensOut: 5,
	}

	trace, err := svc.TraceInteraction(context.Background(), interaction)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if trace.ID == "" {
		t.Error("expected non-empty trace ID")
	}
	if trace.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", trace.AgentID)
	}
	if trace.Input != "Hello, world!" {
		t.Errorf("expected input 'Hello, world!', got '%s'", trace.Input)
	}

	// Verify trace was saved
	savedTrace, _ := repo.GetTrace(context.Background(), trace.ID)
	if savedTrace == nil {
		t.Error("expected trace to be saved in repository")
	}
}

func TestRunEvaluation(t *testing.T) {
	mlflow := &mockMLflowClient{experimentID: "exp-123"}
	queue := &mockQueue{}
	repo := newMockRepository()
	idGen := &mockIDGenerator{}

	svc := NewService(mlflow, queue, repo, idGen)

	trace := &Trace{
		ID:        "trc-test-1",
		AgentID:   "agent-1",
		SessionID: "session-1",
		Input:     "Hello, world!",
		Output:    "Hi there!",
		Latency:   100 * time.Millisecond,
		TokensIn:  10,
		TokensOut: 5,
		CreatedAt: time.Now(),
	}

	result, err := svc.RunEvaluation(context.Background(), trace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TraceID != "trc-test-1" {
		t.Errorf("expected trace ID 'trc-test-1', got '%s'", result.TraceID)
	}
	if result.Metrics.CoherenceScore == 0 {
		t.Error("expected non-zero coherence score")
	}
}

func TestRunEvaluation_NoMLflow(t *testing.T) {
	queue := &mockQueue{}
	repo := newMockRepository()
	idGen := &mockIDGenerator{}

	// Create service without MLflow client
	svc := NewService(nil, queue, repo, idGen)

	trace := &Trace{
		ID:        "trc-test-1",
		AgentID:   "agent-1",
		SessionID: "session-1",
		Input:     "Hello",
		Output:    "Hi",
		Latency:   50 * time.Millisecond,
	}

	result, err := svc.RunEvaluation(context.Background(), trace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TraceID != "trc-test-1" {
		t.Errorf("expected trace ID 'trc-test-1', got '%s'", result.TraceID)
	}
}

func TestCollectFeedback(t *testing.T) {
	mlflow := &mockMLflowClient{experimentID: "exp-123"}
	queue := &mockQueue{}
	repo := newMockRepository()
	idGen := &mockIDGenerator{}

	svc := NewService(mlflow, queue, repo, idGen)

	req := &FeedbackRequest{
		TraceID: "trc-123",
		Rating:  5,
		Comment: "Great response!",
		Tags:    []string{"helpful", "accurate"},
	}

	err := svc.CollectFeedback(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify feedback was saved
	feedback, _ := repo.GetFeedback(context.Background(), "trc-123")
	if len(feedback) != 1 {
		t.Errorf("expected 1 feedback, got %d", len(feedback))
	}
	if feedback[0].Rating != 5 {
		t.Errorf("expected rating 5, got %d", feedback[0].Rating)
	}
}

func TestGetTrace(t *testing.T) {
	mlflow := &mockMLflowClient{experimentID: "exp-123"}
	queue := &mockQueue{}
	repo := newMockRepository()
	idGen := &mockIDGenerator{}

	svc := NewService(mlflow, queue, repo, idGen)

	// Save a trace first
	trace := &Trace{
		ID:        "trc-test-get",
		AgentID:   "agent-1",
		SessionID: "session-1",
		Input:     "Test input",
		Output:    "Test output",
	}
	repo.SaveTrace(context.Background(), trace)

	// Get the trace
	retrieved, err := svc.GetTrace(context.Background(), "trc-test-get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected non-nil trace")
	}
	if retrieved.ID != "trc-test-get" {
		t.Errorf("expected trace ID 'trc-test-get', got '%s'", retrieved.ID)
	}
}

func TestListTraces(t *testing.T) {
	mlflow := &mockMLflowClient{experimentID: "exp-123"}
	queue := &mockQueue{}
	repo := newMockRepository()
	idGen := &mockIDGenerator{}

	svc := NewService(mlflow, queue, repo, idGen)

	// Save multiple traces
	for i := 0; i < 5; i++ {
		trace := &Trace{
			ID:        "trc-list-" + string(rune('0'+i)),
			AgentID:   "agent-1",
			SessionID: "session-1",
		}
		repo.SaveTrace(context.Background(), trace)
	}

	// List traces
	traces, err := svc.ListTraces(context.Background(), TraceFilter{
		AgentID: "agent-1",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(traces) != 5 {
		t.Errorf("expected 5 traces, got %d", len(traces))
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is a ..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.max)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.max, result, tt.expected)
		}
	}
}

func TestCalculateMetrics_EmptyOutput(t *testing.T) {
	svc := NewService(nil, nil, newMockRepository(), &mockIDGenerator{})

	trace := &Trace{
		ID:     "test",
		Output: "",
	}

	metrics := svc.calculateMetrics(trace)
	if metrics.CoherenceScore != 0 {
		t.Errorf("expected coherence score 0 for empty output, got %f", metrics.CoherenceScore)
	}
	if metrics.RelevanceScore != 0 {
		t.Errorf("expected relevance score 0 for empty output, got %f", metrics.RelevanceScore)
	}
}

func TestCalculateMetrics_HighLatency(t *testing.T) {
	svc := NewService(nil, nil, newMockRepository(), &mockIDGenerator{})

	trace := &Trace{
		ID:      "test",
		Output:  "Some output",
		Latency: 60 * time.Second, // High latency
	}

	metrics := svc.calculateMetrics(trace)
	// Coherence should be penalized for high latency
	if metrics.CoherenceScore >= 0.85 {
		t.Errorf("expected penalized coherence score, got %f", metrics.CoherenceScore)
	}
}
