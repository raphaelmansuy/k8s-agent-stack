// Package a2a implements the Agent-to-Agent (A2A) protocol for agent communication.
// This matches the kagent A2A protocol specification.
package a2a

import (
	"encoding/json"
	"time"
)

// === JSON-RPC 2.0 Types ===

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      string          `json:"id"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      string      `json:"id"`
}

// Error represents a JSON-RPC 2.0 error.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// === A2A Message Types ===

// Part represents a content part in an A2A message.
type Part struct {
	Kind     string                 `json:"kind"` // "text" or "data"
	Text     string                 `json:"text,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TextPart creates a text part.
func TextPart(text string) Part {
	return Part{Kind: "text", Text: text}
}

// DataPart creates a data part.
func DataPart(data map[string]interface{}) Part {
	return Part{Kind: "data", Data: data}
}

// FunctionCallPart creates a function call part.
func FunctionCallPart(id, name string, args map[string]interface{}) Part {
	return Part{
		Kind: "data",
		Data: map[string]interface{}{
			"id":   id,
			"name": name,
			"args": args,
		},
		Metadata: map[string]interface{}{
			"kagent_type": "function_call",
		},
	}
}

// FunctionResponsePart creates a function response part.
func FunctionResponsePart(id, name string, response interface{}) Part {
	return Part{
		Kind: "data",
		Data: map[string]interface{}{
			"id":       id,
			"name":     name,
			"response": response,
		},
		Metadata: map[string]interface{}{
			"kagent_type": "function_response",
		},
	}
}

// Message represents an A2A message.
type Message struct {
	Kind      string                 `json:"kind"`
	MessageID string                 `json:"messageId"`
	Role      string                 `json:"role"` // "user" or "agent"
	Parts     []Part                 `json:"parts"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewMessage creates a new A2A message.
func NewMessage(id, role string, parts []Part) *Message {
	return &Message{
		Kind:      "message",
		MessageID: id,
		Role:      role,
		Parts:     parts,
	}
}

// === Task Types ===

// TaskStatus represents the status of an A2A task.
type TaskStatus struct {
	State     string   `json:"state"` // "submitted", "working", "completed", "failed", "cancelled"
	Timestamp string   `json:"timestamp"`
	Message   *Message `json:"message,omitempty"`
}

// TaskState constants matching A2A spec.
const (
	TaskStateSubmitted TaskState = "submitted"
	TaskStateWorking   TaskState = "working"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
	TaskStateCancelled TaskState = "cancelled"
)

type TaskState string

// NewTaskStatus creates a new task status.
func NewTaskStatus(state TaskState, msg *Message) *TaskStatus {
	return &TaskStatus{
		State:     string(state),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Message:   msg,
	}
}

// Task represents an A2A task.
type Task struct {
	TaskID    string                 `json:"taskId"`
	ContextID string                 `json:"contextId"`
	Status    *TaskStatus            `json:"status"`
	Artifacts []Artifact             `json:"artifacts,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// === Event Types (for SSE streaming) ===

// TaskStatusUpdateEvent represents a streaming status update.
type TaskStatusUpdateEvent struct {
	Kind      string                 `json:"kind"`
	TaskID    string                 `json:"taskId"`
	ContextID string                 `json:"contextId"`
	Status    *TaskStatus            `json:"status"`
	Final     bool                   `json:"final"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewStatusUpdateEvent creates a status update event.
func NewStatusUpdateEvent(taskID, contextID string, status *TaskStatus, final bool) *TaskStatusUpdateEvent {
	return &TaskStatusUpdateEvent{
		Kind:      "status-update",
		TaskID:    taskID,
		ContextID: contextID,
		Status:    status,
		Final:     final,
	}
}

// Artifact represents output data from an agent.
type Artifact struct {
	ArtifactID  string                 `json:"artifactId"`
	Parts       []Part                 `json:"parts"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TaskArtifactUpdateEvent represents a streaming artifact update.
type TaskArtifactUpdateEvent struct {
	Kind      string                 `json:"kind"`
	TaskID    string                 `json:"taskId"`
	ContextID string                 `json:"contextId"`
	Artifact  *Artifact              `json:"artifact"`
	Append    bool                   `json:"append"`
	LastChunk bool                   `json:"lastChunk"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewArtifactUpdateEvent creates an artifact update event.
func NewArtifactUpdateEvent(taskID, contextID string, artifact *Artifact, lastChunk bool) *TaskArtifactUpdateEvent {
	return &TaskArtifactUpdateEvent{
		Kind:      "artifact-update",
		TaskID:    taskID,
		ContextID: contextID,
		Artifact:  artifact,
		Append:    false,
		LastChunk: lastChunk,
	}
}

// === Agent Card Types ===

// AgentCard represents an A2A Agent Card for discovery.
type AgentCard struct {
	ProtocolVersion     string       `json:"protocolVersion"`
	Name                string       `json:"name"`
	Description         string       `json:"description,omitempty"`
	Version             string       `json:"version"`
	SupportedInterfaces []Interface  `json:"supportedInterfaces"`
	Capabilities        Capabilities `json:"capabilities"`
	DefaultInputModes   []string     `json:"defaultInputModes"`
	DefaultOutputModes  []string     `json:"defaultOutputModes"`
	Skills              []Skill      `json:"skills,omitempty"`
}

// Interface represents a supported A2A interface.
type Interface struct {
	URL             string `json:"url"`
	ProtocolBinding string `json:"protocolBinding"`
}

// Capabilities represents agent capabilities.
type Capabilities struct {
	Streaming              bool `json:"streaming"`
	PushNotifications      bool `json:"pushNotifications"`
	StateTransitionHistory bool `json:"stateTransitionHistory"`
}

// Skill represents an agent skill/tool.
type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// === Request/Response Types ===

// SendMessageParams represents params for message/send.
type SendMessageParams struct {
	Message MessageInput `json:"message"`
}

// MessageInput represents the input message format.
type MessageInput struct {
	MessageID string `json:"messageId,omitempty"`
	ContextID string `json:"contextId,omitempty"`
	Role      string `json:"role,omitempty"`
	Parts     []Part `json:"parts"`
}

// GetTaskParams represents params for task/get.
type GetTaskParams struct {
	TaskID string `json:"taskId"`
}

// CancelTaskParams represents params for task/cancel.
type CancelTaskParams struct {
	TaskID string `json:"taskId"`
}
