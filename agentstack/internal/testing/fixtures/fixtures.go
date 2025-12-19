// Package fixtures provides test fixtures and helpers for E2E testing.
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

package fixtures

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
)

// TestEnv represents a complete test environment.
type TestEnv struct {
	Logger    *slog.Logger
	A2AServer *MockA2AServer
	cleanup   []func()
	mu        sync.Mutex
}

// NewTestEnv creates a new test environment.
func NewTestEnv() *TestEnv {
	return &TestEnv{
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		cleanup: make([]func(), 0),
	}
}

// NewTestEnvWithLogging creates a test environment with visible logging.
func NewTestEnvWithLogging() *TestEnv {
	return &TestEnv{
		Logger:  slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
		cleanup: make([]func(), 0),
	}
}

// AddCleanup adds a cleanup function to be called on Close.
func (e *TestEnv) AddCleanup(fn func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cleanup = append(e.cleanup, fn)
}

// Close cleans up all resources in reverse order.
func (e *TestEnv) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := len(e.cleanup) - 1; i >= 0; i-- {
		e.cleanup[i]()
	}
}

// StartMockA2AServer starts a mock A2A-compliant server.
func (e *TestEnv) StartMockA2AServer(opts ...MockA2AOption) *MockA2AServer {
	server := NewMockA2AServer(opts...)
	e.A2AServer = server
	e.AddCleanup(server.Close)
	return server
}

// MockA2AServer is a mock server that implements A2A protocol.
type MockA2AServer struct {
	Server       *httptest.Server
	URL          string
	MessageCount int
	LastRequest  *a2a.Request
	Responses    map[string]interface{}
	mu           sync.RWMutex
	logger       *slog.Logger

	// Behavior configuration
	responseDelay time.Duration
	errorOnSend   bool
	streamChunks  int
}

// MockA2AOption configures the mock server.
type MockA2AOption func(*MockA2AServer)

// WithResponseDelay sets the response delay.
func WithResponseDelay(d time.Duration) MockA2AOption {
	return func(s *MockA2AServer) {
		s.responseDelay = d
	}
}

// WithError makes the server return errors.
func WithError() MockA2AOption {
	return func(s *MockA2AServer) {
		s.errorOnSend = true
	}
}

// WithStreamChunks sets how many chunks to stream.
func WithStreamChunks(n int) MockA2AOption {
	return func(s *MockA2AServer) {
		s.streamChunks = n
	}
}

// WithLogger sets the logger for the mock server.
func WithLogger(l *slog.Logger) MockA2AOption {
	return func(s *MockA2AServer) {
		s.logger = l
	}
}

// NewMockA2AServer creates a new mock A2A server.
func NewMockA2AServer(opts ...MockA2AOption) *MockA2AServer {
	m := &MockA2AServer{
		Responses:    make(map[string]interface{}),
		streamChunks: 3, // default stream chunks
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	for _, opt := range opts {
		opt(m)
	}

	m.Server = httptest.NewServer(http.HandlerFunc(m.handleRequest))
	m.URL = m.Server.URL

	return m
}

// Close stops the mock server.
func (m *MockA2AServer) Close() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// SetResponse sets a custom response for a method.
func (m *MockA2AServer) SetResponse(method string, response interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses[method] = response
}

// GetMessageCount returns the number of messages received.
func (m *MockA2AServer) GetMessageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MessageCount
}

// handleRequest handles incoming A2A requests.
func (m *MockA2AServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	if m.responseDelay > 0 {
		time.Sleep(m.responseDelay)
	}

	var req a2a.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, req.ID, -32700, "Parse error")
		return
	}

	m.mu.Lock()
	m.MessageCount++
	m.LastRequest = &req
	m.mu.Unlock()

	m.logger.Debug("received request", "method", req.Method, "id", req.ID)

	// Check for error mode
	if m.errorOnSend {
		m.writeError(w, req.ID, -32603, "Internal error: mock error mode enabled")
		return
	}

	// Check for custom response
	m.mu.RLock()
	customResp, hasCustom := m.Responses[req.Method]
	m.mu.RUnlock()
	if hasCustom {
		m.writeResponse(w, req.ID, customResp)
		return
	}

	// Handle by method
	switch req.Method {
	case "message/send":
		m.handleMessageSend(w, &req)
	case "message/stream":
		m.handleMessageStream(w, r, &req)
	case "task/get":
		m.handleTaskGet(w, &req)
	case "task/cancel":
		m.handleTaskCancel(w, &req)
	case ".well-known/agent.json":
		m.handleAgentCard(w, &req)
	default:
		m.writeError(w, req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

// handleMessageSend handles message/send requests.
func (m *MockA2AServer) handleMessageSend(w http.ResponseWriter, req *a2a.Request) {
	var params a2a.SendMessageParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		m.writeError(w, req.ID, -32602, "Invalid params")
		return
	}

	// Extract text from message
	inputText := ""
	for _, part := range params.Message.Parts {
		if part.Kind == "text" {
			inputText = part.Text
			break
		}
	}

	// Generate response
	responseText := fmt.Sprintf("Echo: %s", inputText)
	if inputText == "" {
		responseText = "Hello from mock agent!"
	}

	status := a2a.NewTaskStatus(a2a.TaskStateCompleted, a2a.NewMessage(
		"msg-response-1",
		"agent",
		[]a2a.Part{a2a.TextPart(responseText)},
	))

	m.writeResponse(w, req.ID, status)
}

// handleMessageStream handles message/stream requests with SSE.
func (m *MockA2AServer) handleMessageStream(w http.ResponseWriter, r *http.Request, req *a2a.Request) {
	var params a2a.SendMessageParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		m.writeError(w, req.ID, -32602, "Invalid params")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		m.writeError(w, req.ID, -32603, "Streaming not supported")
		return
	}

	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	contextID := params.Message.ContextID
	if contextID == "" {
		contextID = fmt.Sprintf("ctx-%d", time.Now().UnixNano())
	}

	// Send working status
	workingEvent := a2a.NewStatusUpdateEvent(taskID, contextID,
		a2a.NewTaskStatus(a2a.TaskStateWorking, nil), false)
	m.writeSSE(w, flusher, workingEvent)

	// Stream chunks
	for i := 0; i < m.streamChunks; i++ {
		select {
		case <-r.Context().Done():
			return
		default:
			time.Sleep(50 * time.Millisecond)

			chunkText := fmt.Sprintf("Chunk %d of %d", i+1, m.streamChunks)
			artifact := &a2a.Artifact{
				ArtifactID: fmt.Sprintf("artifact-%d", i),
				Parts:      []a2a.Part{a2a.TextPart(chunkText)},
			}

			event := a2a.NewArtifactUpdateEvent(taskID, contextID, artifact, i == m.streamChunks-1)
			m.writeSSE(w, flusher, event)
		}
	}

	// Send completed status
	completedEvent := a2a.NewStatusUpdateEvent(taskID, contextID,
		a2a.NewTaskStatus(a2a.TaskStateCompleted, a2a.NewMessage(
			"msg-final",
			"agent",
			[]a2a.Part{a2a.TextPart("Streaming complete!")},
		)), true)
	m.writeSSE(w, flusher, completedEvent)
}

// handleTaskGet handles task/get requests.
func (m *MockA2AServer) handleTaskGet(w http.ResponseWriter, req *a2a.Request) {
	task := &a2a.Task{
		TaskID:    "task-123",
		ContextID: "ctx-123",
		Status:    a2a.NewTaskStatus(a2a.TaskStateCompleted, nil),
	}
	m.writeResponse(w, req.ID, task)
}

// handleTaskCancel handles task/cancel requests.
func (m *MockA2AServer) handleTaskCancel(w http.ResponseWriter, req *a2a.Request) {
	status := a2a.NewTaskStatus(a2a.TaskStateCancelled, nil)
	m.writeResponse(w, req.ID, status)
}

// handleAgentCard returns the agent card.
func (m *MockA2AServer) handleAgentCard(w http.ResponseWriter, req *a2a.Request) {
	card := &a2a.AgentCard{
		ProtocolVersion: "1.0",
		Name:            "Mock Agent",
		Description:     "A mock agent for testing",
		Version:         "1.0.0",
		SupportedInterfaces: []a2a.Interface{
			{URL: m.URL, ProtocolBinding: "http+json"},
		},
		Capabilities: a2a.Capabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
		Skills: []a2a.Skill{
			{ID: "echo", Name: "Echo", Description: "Echoes back the input"},
		},
	}
	m.writeResponse(w, req.ID, card)
}

// writeResponse writes a successful JSON-RPC response.
func (m *MockA2AServer) writeResponse(w http.ResponseWriter, id string, result interface{}) {
	resp := a2a.Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// writeError writes a JSON-RPC error response.
func (m *MockA2AServer) writeError(w http.ResponseWriter, id string, code int, message string) {
	resp := a2a.Response{
		JSONRPC: "2.0",
		Error: &a2a.Error{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// writeSSE writes an SSE event.
func (m *MockA2AServer) writeSSE(w http.ResponseWriter, flusher http.Flusher, event interface{}) {
	data, _ := json.Marshal(event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// === Test Data Builders ===

// AgentBuilder builds test agent configurations.
type AgentBuilder struct {
	id        string
	name      string
	image     string
	namespace string
	env       map[string]string
	replicas  int32
}

// NewAgentBuilder creates a new agent builder with defaults.
func NewAgentBuilder() *AgentBuilder {
	return &AgentBuilder{
		id:        "test-agent",
		name:      "Test Agent",
		image:     "ghcr.io/test/agent:latest",
		namespace: "default",
		env:       make(map[string]string),
		replicas:  1,
	}
}

// WithID sets the agent ID.
func (b *AgentBuilder) WithID(id string) *AgentBuilder {
	b.id = id
	return b
}

// WithName sets the agent name.
func (b *AgentBuilder) WithName(name string) *AgentBuilder {
	b.name = name
	return b
}

// WithImage sets the container image.
func (b *AgentBuilder) WithImage(image string) *AgentBuilder {
	b.image = image
	return b
}

// WithNamespace sets the namespace.
func (b *AgentBuilder) WithNamespace(ns string) *AgentBuilder {
	b.namespace = ns
	return b
}

// WithEnv adds environment variables.
func (b *AgentBuilder) WithEnv(key, value string) *AgentBuilder {
	b.env[key] = value
	return b
}

// WithReplicas sets the replica count.
func (b *AgentBuilder) WithReplicas(n int32) *AgentBuilder {
	b.replicas = n
	return b
}

// Build returns the agent configuration as a map.
func (b *AgentBuilder) Build() map[string]interface{} {
	return map[string]interface{}{
		"id":        b.id,
		"name":      b.name,
		"image":     b.image,
		"namespace": b.namespace,
		"env":       b.env,
		"replicas":  b.replicas,
	}
}

// MessageBuilder builds test A2A messages.
type MessageBuilder struct {
	contextID string
	content   string
	metadata  map[string]interface{}
}

// NewMessageBuilder creates a new message builder.
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		metadata: make(map[string]interface{}),
	}
}

// WithContextID sets the context ID.
func (b *MessageBuilder) WithContextID(id string) *MessageBuilder {
	b.contextID = id
	return b
}

// WithContent sets the message content.
func (b *MessageBuilder) WithContent(content string) *MessageBuilder {
	b.content = content
	return b
}

// WithMetadata adds metadata.
func (b *MessageBuilder) WithMetadata(key string, value interface{}) *MessageBuilder {
	b.metadata[key] = value
	return b
}

// Build returns the message params.
func (b *MessageBuilder) Build() *a2a.SendMessageParams {
	return &a2a.SendMessageParams{
		Message: a2a.MessageInput{
			ContextID: b.contextID,
			Parts:     []a2a.Part{a2a.TextPart(b.content)},
		},
	}
}

// === Assertion Helpers ===

// Eventually retries a check function until it succeeds or times out.
func Eventually(ctx context.Context, check func() error, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := check(); err == nil {
				return nil
			} else {
				lastErr = err
			}
			time.Sleep(interval)
		}
	}

	return fmt.Errorf("timeout waiting for condition: %w", lastErr)
}

// RequireNoError panics if err is not nil.
func RequireNoError(err error, msg string) {
	if err != nil {
		panic(fmt.Sprintf("%s: %v", msg, err))
	}
}

// HTTPClient returns a configured HTTP client for testing.
func HTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}
