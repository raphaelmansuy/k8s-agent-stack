// Package a2a provides the Agent-to-Agent protocol service.
package a2a

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventHandler handles A2A streaming events.
type EventHandler func(event interface{}) error

// Service provides A2A protocol operations for agent communication.
type Service struct {
	httpClient *http.Client
	sessions   map[string]*Session
	mu         sync.RWMutex
}

// Session represents an active A2A conversation session.
type Session struct {
	ContextID   string
	AgentURL    string
	Tasks       map[string]*Task
	History     []*Message
	CreatedAt   time.Time
	LastUpdated time.Time
	mu          sync.RWMutex
}

// NewService creates a new A2A service.
func NewService() *Service {
	return &Service{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		sessions: make(map[string]*Session),
	}
}

// NewSession creates a new conversation session with an agent.
func (s *Service) NewSession(agentURL string) (*Session, error) {
	contextID := uuid.NewString()

	session := &Session{
		ContextID:   contextID,
		AgentURL:    agentURL,
		Tasks:       make(map[string]*Task),
		History:     make([]*Message, 0),
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	s.mu.Lock()
	s.sessions[contextID] = session
	s.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by context ID.
func (s *Service) GetSession(contextID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[contextID]
	return session, ok
}

// DeleteSession removes a session.
func (s *Service) DeleteSession(contextID string) {
	s.mu.Lock()
	delete(s.sessions, contextID)
	s.mu.Unlock()
}

// SendMessage sends a message to an agent and returns the task.
// This is a synchronous call that waits for completion.
func (s *Service) SendMessage(ctx context.Context, agentURL string, params *SendMessageParams) (*Task, error) {
	// Mock response for development
	if strings.Contains(agentURL, ".mock.svc.cluster.local") {
		return &Task{
			TaskID: uuid.NewString(),
			Status: &TaskStatus{
				State: string(TaskStateCompleted),
				Message: &Message{
					Role: "assistant",
					Parts: []Part{
						{Kind: "text", Text: "This is a mock response from the agent."},
					},
				},
			},
		}, nil
	}

	// Create JSON-RPC request
	reqID := uuid.NewString()
	contextID := params.Message.ContextID
	if contextID == "" {
		contextID = uuid.NewString()
	}

	req := Request{
		JSONRPC: "2.0",
		Method:  "message/send",
		ID:      reqID,
		Params:  mustMarshal(params),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", agentURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	var jsonResp Response
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if jsonResp.Error != nil {
		return nil, fmt.Errorf("agent error: %s (code %d)", jsonResp.Error.Message, jsonResp.Error.Code)
	}

	// Parse result as task
	var task Task
	resultBytes, _ := json.Marshal(jsonResp.Result)
	if err := json.Unmarshal(resultBytes, &task); err != nil {
		// Fallback: maybe it's just a TaskStatus?
		var taskStatus TaskStatus
		if err := json.Unmarshal(resultBytes, &taskStatus); err == nil {
			task.Status = &taskStatus
		} else {
			return nil, fmt.Errorf("failed to parse result as Task or TaskStatus: %w", err)
		}
	}

	// Ensure TaskID and ContextID are set if they were in the result but with different names
	if task.TaskID == "" {
		// Try to get "id" from the raw result if TaskID is empty
		var rawResult map[string]interface{}
		json.Unmarshal(resultBytes, &rawResult)
		if id, ok := rawResult["id"].(string); ok {
			task.TaskID = id
		}
	}

	return &task, nil
}

// StreamMessage sends a message and streams the response via SSE.
func (s *Service) StreamMessage(ctx context.Context, agentURL string, params *SendMessageParams, handler EventHandler) error {
	// Create JSON-RPC request for streaming
	reqID := uuid.NewString()
	contextID := params.Message.ContextID
	if contextID == "" {
		contextID = uuid.NewString()
	}

	req := Request{
		JSONRPC: "2.0",
		Method:  "message/stream",
		ID:      reqID,
		Params:  mustMarshal(params),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", agentURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	// Parse SSE stream
	return s.parseSSEStream(ctx, resp.Body, handler)
}

// parseSSEStream reads SSE events and dispatches to handler.
func (s *Service) parseSSEStream(ctx context.Context, r io.Reader, handler EventHandler) error {
	scanner := bufio.NewScanner(r)

	var dataBuffer bytes.Buffer

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		// Empty line signals end of event
		if line == "" {
			if dataBuffer.Len() > 0 {
				if err := s.handleSSEData(dataBuffer.Bytes(), handler); err != nil {
					return err
				}
				dataBuffer.Reset()
			}
			continue
		}

		// Parse SSE format
		if len(line) > 5 && line[:5] == "data:" {
			data := line[5:]
			if len(data) > 0 && data[0] == ' ' {
				data = data[1:]
			}
			dataBuffer.WriteString(data)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	// Handle any remaining data
	if dataBuffer.Len() > 0 {
		if err := s.handleSSEData(dataBuffer.Bytes(), handler); err != nil {
			return err
		}
	}

	return nil
}

// handleSSEData parses a single SSE data payload and dispatches to handler.
func (s *Service) handleSSEData(data []byte, handler EventHandler) error {
	// First try to parse as JSON-RPC response
	var jsonRPC Response
	if err := json.Unmarshal(data, &jsonRPC); err == nil && jsonRPC.JSONRPC == "2.0" {
		if jsonRPC.Error != nil {
			return fmt.Errorf("agent error: %s", jsonRPC.Error.Message)
		}

		// Extract the result and determine event type
		if jsonRPC.Result != nil {
			resultBytes, _ := json.Marshal(jsonRPC.Result)
			return s.dispatchEvent(resultBytes, handler)
		}
		return nil
	}

	// Try to parse directly as an event
	return s.dispatchEvent(data, handler)
}

// dispatchEvent determines event type and dispatches to handler.
func (s *Service) dispatchEvent(data []byte, handler EventHandler) error {
	// Check the "kind" field to determine type
	var kindCheck struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &kindCheck); err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	switch kindCheck.Kind {
	case "status-update":
		var event TaskStatusUpdateEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to parse status event: %w", err)
		}
		return handler(&event)

	case "artifact-update":
		var event TaskArtifactUpdateEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return fmt.Errorf("failed to parse artifact event: %w", err)
		}
		return handler(&event)

	case "message":
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("failed to parse message: %w", err)
		}
		return handler(&msg)

	default:
		// Unknown event type, pass raw data
		return handler(data)
	}
}

// GetTask retrieves a task status from an agent.
func (s *Service) GetTask(ctx context.Context, agentURL string, taskID string) (*Task, error) {
	reqID := uuid.NewString()

	req := Request{
		JSONRPC: "2.0",
		Method:  "task/get",
		ID:      reqID,
		Params:  mustMarshal(GetTaskParams{TaskID: taskID}),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", agentURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var jsonResp Response
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if jsonResp.Error != nil {
		return nil, fmt.Errorf("agent error: %s", jsonResp.Error.Message)
	}

	// Parse result as task
	resultBytes, _ := json.Marshal(jsonResp.Result)
	var task Task
	if err := json.Unmarshal(resultBytes, &task); err != nil {
		return nil, fmt.Errorf("failed to parse task: %w", err)
	}

	return &task, nil
}

// CancelTask cancels a running task.
func (s *Service) CancelTask(ctx context.Context, agentURL string, taskID string) error {
	reqID := uuid.NewString()

	req := Request{
		JSONRPC: "2.0",
		Method:  "task/cancel",
		ID:      reqID,
		Params:  mustMarshal(CancelTaskParams{TaskID: taskID}),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", agentURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var jsonResp Response
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if jsonResp.Error != nil {
		return fmt.Errorf("agent error: %s", jsonResp.Error.Message)
	}

	return nil
}

// GetAgentCard retrieves the Agent Card from an A2A agent.
func (s *Service) GetAgentCard(ctx context.Context, agentURL string) (*AgentCard, error) {
	// Agent cards are typically at /.well-known/agent.json
	cardURL := agentURL + "/.well-known/agent.json"

	httpReq, err := http.NewRequestWithContext(ctx, "GET", cardURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent card returned status %d", resp.StatusCode)
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("failed to decode agent card: %w", err)
	}

	return &card, nil
}

// mustMarshal marshals to JSON or panics.
func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal: %v", err))
	}
	return data
}
