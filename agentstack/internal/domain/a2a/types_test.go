package a2a

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTextPart(t *testing.T) {
	part := TextPart("Hello, world!")

	if part.Kind != "text" {
		t.Errorf("expected kind 'text', got '%s'", part.Kind)
	}
	if part.Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got '%s'", part.Text)
	}
}

func TestDataPart(t *testing.T) {
	data := map[string]interface{}{
		"key":   "value",
		"count": 42,
	}
	part := DataPart(data)

	if part.Kind != "data" {
		t.Errorf("expected kind 'data', got '%s'", part.Kind)
	}
	if part.Data["key"] != "value" {
		t.Errorf("expected data key 'value', got '%v'", part.Data["key"])
	}
}

func TestFunctionCallPart(t *testing.T) {
	args := map[string]interface{}{
		"city": "Paris",
	}
	part := FunctionCallPart("call-123", "get_weather", args)

	if part.Kind != "data" {
		t.Errorf("expected kind 'data', got '%s'", part.Kind)
	}
	if part.Data["name"] != "get_weather" {
		t.Errorf("expected name 'get_weather', got '%v'", part.Data["name"])
	}
	if part.Metadata["kagent_type"] != "function_call" {
		t.Errorf("expected kagent_type 'function_call', got '%v'", part.Metadata["kagent_type"])
	}
}

func TestFunctionResponsePart(t *testing.T) {
	response := map[string]interface{}{
		"temperature": 20,
		"unit":        "celsius",
	}
	part := FunctionResponsePart("call-123", "get_weather", response)

	if part.Kind != "data" {
		t.Errorf("expected kind 'data', got '%s'", part.Kind)
	}
	if part.Metadata["kagent_type"] != "function_response" {
		t.Errorf("expected kagent_type 'function_response', got '%v'", part.Metadata["kagent_type"])
	}
}

func TestNewMessage(t *testing.T) {
	parts := []Part{TextPart("Hello!")}
	msg := NewMessage("msg-123", "user", parts)

	if msg.Kind != "message" {
		t.Errorf("expected kind 'message', got '%s'", msg.Kind)
	}
	if msg.MessageID != "msg-123" {
		t.Errorf("expected messageId 'msg-123', got '%s'", msg.MessageID)
	}
	if msg.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", msg.Role)
	}
	if len(msg.Parts) != 1 {
		t.Errorf("expected 1 part, got %d", len(msg.Parts))
	}
}

func TestNewTaskStatus(t *testing.T) {
	msg := NewMessage("msg-123", "agent", []Part{TextPart("Done!")})
	status := NewTaskStatus(TaskStateCompleted, msg)

	if status.State != "completed" {
		t.Errorf("expected state 'completed', got '%s'", status.State)
	}
	if status.Message == nil {
		t.Error("expected message, got nil")
	}

	// Verify timestamp is recent
	ts, err := time.Parse(time.RFC3339, status.Timestamp)
	if err != nil {
		t.Errorf("failed to parse timestamp: %v", err)
	}
	if time.Since(ts) > time.Minute {
		t.Error("timestamp too old")
	}
}

func TestTaskStatusUpdateEvent(t *testing.T) {
	status := NewTaskStatus(TaskStateWorking, nil)
	event := NewStatusUpdateEvent("task-123", "ctx-456", status, false)

	if event.Kind != "status-update" {
		t.Errorf("expected kind 'status-update', got '%s'", event.Kind)
	}
	if event.TaskID != "task-123" {
		t.Errorf("expected taskId 'task-123', got '%s'", event.TaskID)
	}
	if event.ContextID != "ctx-456" {
		t.Errorf("expected contextId 'ctx-456', got '%s'", event.ContextID)
	}
	if event.Final {
		t.Error("expected final=false")
	}
}

func TestArtifactUpdateEvent(t *testing.T) {
	artifact := &Artifact{
		ArtifactID:  "art-123",
		Parts:       []Part{TextPart("Generated content")},
		Name:        "output.txt",
		Description: "Generated output file",
	}
	event := NewArtifactUpdateEvent("task-123", "ctx-456", artifact, true)

	if event.Kind != "artifact-update" {
		t.Errorf("expected kind 'artifact-update', got '%s'", event.Kind)
	}
	if !event.LastChunk {
		t.Error("expected lastChunk=true")
	}
}

func TestAgentCard(t *testing.T) {
	card := AgentCard{
		ProtocolVersion: "1.0",
		Name:            "test-agent",
		Description:     "A test agent",
		Version:         "0.1.0",
		SupportedInterfaces: []Interface{
			{URL: "http://localhost:8080", ProtocolBinding: "http+json"},
		},
		Capabilities: Capabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
		Skills: []Skill{
			{ID: "weather", Name: "Get Weather", Description: "Get weather info"},
		},
	}

	// Serialize and deserialize
	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("failed to marshal agent card: %v", err)
	}

	var parsed AgentCard
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal agent card: %v", err)
	}

	if parsed.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got '%s'", parsed.Name)
	}
	if !parsed.Capabilities.Streaming {
		t.Error("expected streaming=true")
	}
	if len(parsed.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(parsed.Skills))
	}
}

func TestJSONRPCRequest(t *testing.T) {
	params := SendMessageParams{
		Message: MessageInput{
			ContextID: "ctx-123",
			Parts:     []Part{TextPart("Hello!")},
		},
	}
	paramsData, _ := json.Marshal(params)

	req := Request{
		JSONRPC: "2.0",
		Method:  "message/send",
		Params:  paramsData,
		ID:      "req-123",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	var parsed Request
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if parsed.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got '%s'", parsed.JSONRPC)
	}
	if parsed.Method != "message/send" {
		t.Errorf("expected method 'message/send', got '%s'", parsed.Method)
	}
}

func TestJSONRPCResponse(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		Result:  map[string]interface{}{"taskId": "task-123"},
		ID:      "req-123",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var parsed Response
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if parsed.Error != nil {
		t.Error("expected no error")
	}
}

func TestJSONRPCError(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		Error: &Error{
			Code:    -32600,
			Message: "Invalid Request",
			Data:    map[string]string{"field": "message"},
		},
		ID: "req-123",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var parsed Response
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if parsed.Error == nil {
		t.Error("expected error")
	}
	if parsed.Error.Code != -32600 {
		t.Errorf("expected code -32600, got %d", parsed.Error.Code)
	}
}

func TestTaskStates(t *testing.T) {
	states := []TaskState{
		TaskStateSubmitted,
		TaskStateWorking,
		TaskStateCompleted,
		TaskStateFailed,
		TaskStateCancelled,
	}

	expected := []string{
		"submitted",
		"working",
		"completed",
		"failed",
		"cancelled",
	}

	for i, state := range states {
		if string(state) != expected[i] {
			t.Errorf("expected state '%s', got '%s'", expected[i], state)
		}
	}
}
