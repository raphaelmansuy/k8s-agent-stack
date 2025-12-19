// Package kagent provides a mock kagent implementation for E2E testing.
// This simulates a real kagent that can be deployed and communicated with via A2A.
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

package kagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
)

// MockKagent simulates a kagent server for E2E testing.
// It implements the full A2A protocol including agent card discovery.
type MockKagent struct {
	Server    *httptest.Server
	URL       string
	AgentCard *a2a.AgentCard

	// State
	sessions map[string]*Session
	tasks    map[string]*a2a.Task
	mu       sync.RWMutex
	logger   *slog.Logger

	// Configuration
	name           string
	version        string
	skills         []a2a.Skill
	responseDelay  time.Duration
	simulateError  bool
	streamingSpeed time.Duration
}

// Session tracks a conversation session.
type Session struct {
	ContextID string
	History   []*a2a.Message
	CreatedAt time.Time
}

// Config configures the mock kagent.
type Config struct {
	Name           string
	Version        string
	Skills         []a2a.Skill
	ResponseDelay  time.Duration
	StreamingSpeed time.Duration
	Logger         *slog.Logger
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Name:           "Mock Kagent",
		Version:        "1.0.0",
		StreamingSpeed: 50 * time.Millisecond,
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Skills: []a2a.Skill{
			{ID: "echo", Name: "Echo", Description: "Echoes back messages"},
			{ID: "calculate", Name: "Calculate", Description: "Performs basic calculations"},
			{ID: "search", Name: "Search", Description: "Searches for information"},
		},
	}
}

// New creates a new mock kagent.
func New(cfg Config) *MockKagent {
	k := &MockKagent{
		sessions:       make(map[string]*Session),
		tasks:          make(map[string]*a2a.Task),
		name:           cfg.Name,
		version:        cfg.Version,
		skills:         cfg.Skills,
		responseDelay:  cfg.ResponseDelay,
		streamingSpeed: cfg.StreamingSpeed,
		logger:         cfg.Logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/agent.json", k.handleAgentCard)
	mux.HandleFunc("POST /", k.handleJSONRPC)
	mux.HandleFunc("GET /health", k.handleHealth)
	mux.HandleFunc("GET /ready", k.handleReady)

	k.Server = httptest.NewServer(mux)
	k.URL = k.Server.URL

	k.AgentCard = &a2a.AgentCard{
		ProtocolVersion: "1.0",
		Name:            k.name,
		Description:     "A mock kagent for E2E testing",
		Version:         k.version,
		SupportedInterfaces: []a2a.Interface{
			{URL: k.URL, ProtocolBinding: "http+json"},
		},
		Capabilities: a2a.Capabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
		Skills:             k.skills,
	}

	return k
}

// Close shuts down the mock kagent.
func (k *MockKagent) Close() {
	if k.Server != nil {
		k.Server.Close()
	}
}

// SetError enables error simulation.
func (k *MockKagent) SetError(enabled bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.simulateError = enabled
}

// SetResponseDelay sets the response delay.
func (k *MockKagent) SetResponseDelay(d time.Duration) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.responseDelay = d
}

// GetSession returns a session by context ID.
func (k *MockKagent) GetSession(contextID string) (*Session, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	s, ok := k.sessions[contextID]
	return s, ok
}

// GetTask returns a task by ID.
func (k *MockKagent) GetTask(taskID string) (*a2a.Task, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	t, ok := k.tasks[taskID]
	return t, ok
}

// TaskCount returns the number of tasks created.
func (k *MockKagent) TaskCount() int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return len(k.tasks)
}

// handleAgentCard serves the A2A agent card.
func (k *MockKagent) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(k.AgentCard)
}

// handleHealth handles health checks.
func (k *MockKagent) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// handleReady handles readiness checks.
func (k *MockKagent) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// handleJSONRPC handles JSON-RPC 2.0 requests.
func (k *MockKagent) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	// Apply response delay
	k.mu.RLock()
	delay := k.responseDelay
	simulateError := k.simulateError
	k.mu.RUnlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	var req a2a.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		k.writeError(w, "", -32700, "Parse error")
		return
	}

	k.logger.Debug("received request", "method", req.Method, "id", req.ID)

	if simulateError {
		k.writeError(w, req.ID, -32603, "Simulated error")
		return
	}

	// Route by method
	switch req.Method {
	case "message/send":
		k.handleMessageSend(w, &req)
	case "message/stream":
		k.handleMessageStream(w, r, &req)
	case "task/get":
		k.handleTaskGet(w, &req)
	case "task/cancel":
		k.handleTaskCancel(w, &req)
	default:
		k.writeError(w, req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

// handleMessageSend processes synchronous message requests.
func (k *MockKagent) handleMessageSend(w http.ResponseWriter, req *a2a.Request) {
	var params a2a.SendMessageParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		k.writeError(w, req.ID, -32602, "Invalid params")
		return
	}

	// Get or create session
	contextID := params.Message.ContextID
	if contextID == "" {
		contextID = fmt.Sprintf("ctx-%d", time.Now().UnixNano())
	}

	k.mu.Lock()
	session, exists := k.sessions[contextID]
	if !exists {
		session = &Session{
			ContextID: contextID,
			History:   make([]*a2a.Message, 0),
			CreatedAt: time.Now(),
		}
		k.sessions[contextID] = session
	}
	k.mu.Unlock()

	// Extract input
	inputText := k.extractText(params.Message.Parts)

	// Store user message
	userMsg := a2a.NewMessage(fmt.Sprintf("msg-%d", time.Now().UnixNano()), "user", params.Message.Parts)
	session.History = append(session.History, userMsg)

	// Generate response based on input
	responseText := k.generateResponse(inputText)

	// Create response message
	agentMsg := a2a.NewMessage(
		fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		"agent",
		[]a2a.Part{a2a.TextPart(responseText)},
	)
	session.History = append(session.History, agentMsg)

	// Create task
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	task := &a2a.Task{
		TaskID:    taskID,
		ContextID: contextID,
		Status:    a2a.NewTaskStatus(a2a.TaskStateCompleted, agentMsg),
	}

	k.mu.Lock()
	k.tasks[taskID] = task
	k.mu.Unlock()

	k.writeResponse(w, req.ID, task)
}

// handleMessageStream handles streaming message requests.
func (k *MockKagent) handleMessageStream(w http.ResponseWriter, r *http.Request, req *a2a.Request) {
	var params a2a.SendMessageParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		k.writeError(w, req.ID, -32602, "Invalid params")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	contextID := params.Message.ContextID
	if contextID == "" {
		contextID = fmt.Sprintf("ctx-%d", time.Now().UnixNano())
	}

	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	inputText := k.extractText(params.Message.Parts)
	responseText := k.generateResponse(inputText)

	// Create task in working state
	task := &a2a.Task{
		TaskID:    taskID,
		ContextID: contextID,
		Status:    a2a.NewTaskStatus(a2a.TaskStateWorking, nil),
	}

	k.mu.Lock()
	k.tasks[taskID] = task
	k.mu.Unlock()

	// Send working status
	k.sendSSE(w, flusher, a2a.NewStatusUpdateEvent(taskID, contextID,
		a2a.NewTaskStatus(a2a.TaskStateWorking, nil), false))

	// Stream response in chunks
	words := strings.Fields(responseText)
	chunkSize := 3
	for i := 0; i < len(words); i += chunkSize {
		select {
		case <-r.Context().Done():
			return
		default:
			end := i + chunkSize
			if end > len(words) {
				end = len(words)
			}
			chunk := strings.Join(words[i:end], " ")

			artifact := &a2a.Artifact{
				ArtifactID: fmt.Sprintf("artifact-%d", i/chunkSize),
				Parts:      []a2a.Part{a2a.TextPart(chunk)},
			}

			k.sendSSE(w, flusher, a2a.NewArtifactUpdateEvent(
				taskID, contextID, artifact, end >= len(words)))

			time.Sleep(k.streamingSpeed)
		}
	}

	// Send completed status
	agentMsg := a2a.NewMessage(
		fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		"agent",
		[]a2a.Part{a2a.TextPart(responseText)},
	)

	task.Status = a2a.NewTaskStatus(a2a.TaskStateCompleted, agentMsg)

	k.sendSSE(w, flusher, a2a.NewStatusUpdateEvent(taskID, contextID, task.Status, true))
}

// handleTaskGet retrieves task status.
func (k *MockKagent) handleTaskGet(w http.ResponseWriter, req *a2a.Request) {
	var params a2a.GetTaskParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		k.writeError(w, req.ID, -32602, "Invalid params")
		return
	}

	k.mu.RLock()
	task, exists := k.tasks[params.TaskID]
	k.mu.RUnlock()

	if !exists {
		k.writeError(w, req.ID, -32002, "Task not found")
		return
	}

	k.writeResponse(w, req.ID, task)
}

// handleTaskCancel cancels a task.
func (k *MockKagent) handleTaskCancel(w http.ResponseWriter, req *a2a.Request) {
	var params a2a.CancelTaskParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		k.writeError(w, req.ID, -32602, "Invalid params")
		return
	}

	k.mu.Lock()
	task, exists := k.tasks[params.TaskID]
	if exists {
		task.Status = a2a.NewTaskStatus(a2a.TaskStateCancelled, nil)
	}
	k.mu.Unlock()

	if !exists {
		k.writeError(w, req.ID, -32002, "Task not found")
		return
	}

	k.writeResponse(w, req.ID, task.Status)
}

// extractText extracts text from message parts.
func (k *MockKagent) extractText(parts []a2a.Part) string {
	var texts []string
	for _, part := range parts {
		if part.Kind == "text" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, " ")
}

// generateResponse generates a response based on input.
func (k *MockKagent) generateResponse(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))

	// Handle different input patterns
	switch {
	case input == "":
		return "Hello! I'm the mock kagent. How can I help you?"
	case strings.HasPrefix(input, "echo "):
		return strings.TrimPrefix(input, "echo ")
	case strings.Contains(input, "hello") || strings.Contains(input, "hi"):
		return "Hello! Nice to meet you. I'm ready to help."
	case strings.Contains(input, "calculate") || strings.Contains(input, "math"):
		return "I can help with calculations. What would you like me to compute?"
	case strings.Contains(input, "search") || strings.Contains(input, "find"):
		return "I've searched and found relevant information for your query."
	case strings.Contains(input, "help"):
		return "I'm a mock kagent with echo, calculate, and search skills. Try asking me something!"
	case strings.Contains(input, "error"):
		return "Error simulation requested but not enabled on this request."
	default:
		return fmt.Sprintf("I received your message: %q. Processing complete.", input)
	}
}

// writeResponse sends a JSON-RPC response.
func (k *MockKagent) writeResponse(w http.ResponseWriter, id string, result interface{}) {
	resp := a2a.Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// writeError sends a JSON-RPC error response.
func (k *MockKagent) writeError(w http.ResponseWriter, id string, code int, message string) {
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

// sendSSE sends an SSE event.
func (k *MockKagent) sendSSE(w http.ResponseWriter, flusher http.Flusher, event interface{}) {
	data, _ := json.Marshal(event)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// WaitForReady waits for the kagent to be ready.
func (k *MockKagent) WaitForReady(ctx context.Context, timeout time.Duration) error {
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, k.URL+"/ready", http.NoBody)
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				_ = resp.Body.Close()
				return nil
			}
			if resp != nil {
				_ = resp.Body.Close()
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	return fmt.Errorf("kagent not ready after %v", timeout)
}
