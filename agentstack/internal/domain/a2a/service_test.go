package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatal("expected service, got nil")
	}
	if service.sessions == nil {
		t.Error("expected sessions map initialized")
	}
}

func TestNewSession(t *testing.T) {
	service := NewService()

	session, err := service.NewSession("http://agent.example.com")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session.ContextID == "" {
		t.Error("expected contextID to be set")
	}
	if session.AgentURL != "http://agent.example.com" {
		t.Errorf("expected agentURL 'http://agent.example.com', got '%s'", session.AgentURL)
	}

	// Verify session is stored
	retrieved, ok := service.GetSession(session.ContextID)
	if !ok {
		t.Error("expected session to be retrievable")
	}
	if retrieved.ContextID != session.ContextID {
		t.Error("retrieved session has different contextID")
	}
}

func TestDeleteSession(t *testing.T) {
	service := NewService()

	session, _ := service.NewSession("http://agent.example.com")

	service.DeleteSession(session.ContextID)

	_, ok := service.GetSession(session.ContextID)
	if ok {
		t.Error("expected session to be deleted")
	}
}

func TestSendMessageSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json")
		}

		// Parse request
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.JSONRPC != "2.0" {
			t.Errorf("expected jsonrpc 2.0, got %s", req.JSONRPC)
		}
		if req.Method != "message/send" {
			t.Errorf("expected method message/send, got %s", req.Method)
		}

		// Return response
		resp := Response{
			JSONRPC: "2.0",
			Result: map[string]interface{}{
				"taskId":    "test-task-id",
				"contextId": "test-context-id",
				"state":     "completed",
				"timestamp": "2024-01-01T00:00:00Z",
			},
			ID: req.ID,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	service := NewService()

	params := &SendMessageParams{
		Message: MessageInput{
			Parts: []Part{TextPart("Hello, agent!")},
		},
	}

	task, err := service.SendMessage(context.Background(), server.URL, params)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	if task.TaskID == "" {
		t.Error("expected taskID to be set")
	}
}

func TestSendMessageError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)

		resp := Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    -32600,
				Message: "Invalid Request",
			},
			ID: req.ID,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	service := NewService()

	params := &SendMessageParams{
		Message: MessageInput{
			Parts: []Part{TextPart("Hello!")},
		},
	}

	_, err := service.SendMessage(context.Background(), server.URL, params)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestSendMessageHTTPError(t *testing.T) {
	// Create mock server that returns HTTP error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	service := NewService()

	params := &SendMessageParams{
		Message: MessageInput{
			Parts: []Part{TextPart("Hello!")},
		},
	}

	_, err := service.SendMessage(context.Background(), server.URL, params)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetTask(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "task/get" {
			t.Errorf("expected method task/get, got %s", req.Method)
		}

		resp := Response{
			JSONRPC: "2.0",
			Result: map[string]interface{}{
				"taskId":    "task-123",
				"contextId": "ctx-456",
				"status": map[string]interface{}{
					"state":     "completed",
					"timestamp": "2024-01-01T00:00:00Z",
				},
			},
			ID: req.ID,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	service := NewService()

	task, err := service.GetTask(context.Background(), server.URL, "task-123")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}

	if task.TaskID != "task-123" {
		t.Errorf("expected taskId 'task-123', got '%s'", task.TaskID)
	}
}

func TestCancelTask(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "task/cancel" {
			t.Errorf("expected method task/cancel, got %s", req.Method)
		}

		resp := Response{
			JSONRPC: "2.0",
			Result:  map[string]interface{}{"status": "cancelled"},
			ID:      req.ID,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	service := NewService()

	err := service.CancelTask(context.Background(), server.URL, "task-123")
	if err != nil {
		t.Fatalf("failed to cancel task: %v", err)
	}
}

func TestGetAgentCard(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/agent.json" {
			t.Errorf("expected path /.well-known/agent.json, got %s", r.URL.Path)
		}

		card := AgentCard{
			ProtocolVersion: "1.0",
			Name:            "test-agent",
			Version:         "0.1.0",
			Capabilities: Capabilities{
				Streaming: true,
			},
		}
		json.NewEncoder(w).Encode(card)
	}))
	defer server.Close()

	service := NewService()

	card, err := service.GetAgentCard(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("failed to get agent card: %v", err)
	}

	if card.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got '%s'", card.Name)
	}
	if !card.Capabilities.Streaming {
		t.Error("expected streaming capability")
	}
}

func TestStreamMessage(t *testing.T) {
	// Create mock SSE server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// Send status update events
		event1 := TaskStatusUpdateEvent{
			Kind:      "status-update",
			TaskID:    "task-123",
			ContextID: "ctx-456",
			Status:    NewTaskStatus(TaskStateWorking, nil),
			Final:     false,
		}
		data1, _ := json.Marshal(event1)
		w.Write([]byte("data: "))
		w.Write(data1)
		w.Write([]byte("\n\n"))
		flusher.Flush()

		event2 := TaskStatusUpdateEvent{
			Kind:      "status-update",
			TaskID:    "task-123",
			ContextID: "ctx-456",
			Status:    NewTaskStatus(TaskStateCompleted, nil),
			Final:     true,
		}
		data2, _ := json.Marshal(event2)
		w.Write([]byte("data: "))
		w.Write(data2)
		w.Write([]byte("\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	service := NewService()

	params := &SendMessageParams{
		Message: MessageInput{
			Parts: []Part{TextPart("Hello!")},
		},
	}

	eventCount := 0
	var lastEvent *TaskStatusUpdateEvent

	err := service.StreamMessage(context.Background(), server.URL, params, func(event interface{}) error {
		eventCount++
		if statusEvent, ok := event.(*TaskStatusUpdateEvent); ok {
			lastEvent = statusEvent
		}
		return nil
	})

	if err != nil {
		t.Fatalf("failed to stream message: %v", err)
	}

	if eventCount != 2 {
		t.Errorf("expected 2 events, got %d", eventCount)
	}

	if lastEvent == nil {
		t.Fatal("expected last event")
	}
	if lastEvent.Status.State != "completed" {
		t.Errorf("expected final state 'completed', got '%s'", lastEvent.Status.State)
	}
}

func TestParseSSEWithJSONRPC(t *testing.T) {
	// Create mock SSE server with JSON-RPC wrapped events
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// Send JSON-RPC wrapped event
		event := TaskStatusUpdateEvent{
			Kind:      "status-update",
			TaskID:    "task-123",
			ContextID: "ctx-456",
			Status:    NewTaskStatus(TaskStateCompleted, nil),
			Final:     true,
		}

		jsonRPC := Response{
			JSONRPC: "2.0",
			Result:  event,
			ID:      "req-123",
		}
		data, _ := json.Marshal(jsonRPC)
		w.Write([]byte("data: "))
		w.Write(data)
		w.Write([]byte("\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	service := NewService()

	params := &SendMessageParams{
		Message: MessageInput{
			Parts: []Part{TextPart("Hello!")},
		},
	}

	received := false

	err := service.StreamMessage(context.Background(), server.URL, params, func(event interface{}) error {
		received = true
		return nil
	})

	if err != nil {
		t.Fatalf("failed to stream message: %v", err)
	}

	if !received {
		t.Error("expected to receive event")
	}
}

func TestMustMarshal(t *testing.T) {
	// Valid input should work
	result := mustMarshal(map[string]string{"key": "value"})
	if result == nil {
		t.Error("expected result")
	}

	// Verify it's valid JSON
	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Errorf("failed to unmarshal: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("expected 'value', got '%s'", parsed["key"])
	}
}
