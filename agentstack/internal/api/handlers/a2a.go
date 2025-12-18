// Package handlers provides HTTP handlers for the AgentStack API.
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
)

// A2AHandler handles A2A protocol HTTP requests.
type A2AHandler struct {
	service *a2a.Service
	logger  *slog.Logger
}

// NewA2AHandler creates a new A2A handler.
func NewA2AHandler(service *a2a.Service, logger *slog.Logger) *A2AHandler {
	return &A2AHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers A2A routes on the given mux.
func (h *A2AHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /a2a/send", h.SendMessage)
	mux.HandleFunc("POST /a2a/stream", h.StreamMessage)
	mux.HandleFunc("POST /a2a/task/{taskID}", h.GetTask)
	mux.HandleFunc("DELETE /a2a/task/{taskID}", h.CancelTask)
	mux.HandleFunc("GET /a2a/session/{contextID}", h.GetSession)
	mux.HandleFunc("DELETE /a2a/session/{contextID}", h.DeleteSession)
}

// SendMessageRequest is the request body for sending a message.
type SendMessageRequest struct {
	AgentURL  string                 `json:"agentUrl"`
	ContextID string                 `json:"contextId,omitempty"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SendMessageResponse is the response for sending a message.
type SendMessageResponse struct {
	TaskID    string          `json:"taskId"`
	ContextID string          `json:"contextId"`
	Status    *a2a.TaskStatus `json:"status"`
}

// SendMessage handles synchronous message sending to an agent.
func (h *A2AHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AgentURL == "" {
		h.writeError(w, http.StatusBadRequest, "agentUrl is required")
		return
	}

	if req.Content == "" {
		h.writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	params := &a2a.SendMessageParams{
		Message: a2a.MessageInput{
			ContextID: req.ContextID,
			Parts:     []a2a.Part{a2a.TextPart(req.Content)},
		},
	}

	task, err := h.service.SendMessage(r.Context(), req.AgentURL, params)
	if err != nil {
		h.logger.Error("failed to send message", "error", err)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := SendMessageResponse{
		TaskID:    task.TaskID,
		ContextID: task.ContextID,
		Status:    task.Status,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// StreamMessageRequest is the request body for streaming messages.
type StreamMessageRequest struct {
	AgentURL  string                 `json:"agentUrl"`
	ContextID string                 `json:"contextId,omitempty"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// StreamMessage handles streaming message responses via SSE.
func (h *A2AHandler) StreamMessage(w http.ResponseWriter, r *http.Request) {
	var req StreamMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AgentURL == "" {
		h.writeError(w, http.StatusBadRequest, "agentUrl is required")
		return
	}

	if req.Content == "" {
		h.writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	params := &a2a.SendMessageParams{
		Message: a2a.MessageInput{
			ContextID: req.ContextID,
			Parts:     []a2a.Part{a2a.TextPart(req.Content)},
		},
	}

	// Stream events to client
	handler := func(event interface{}) error {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	}

	if err := h.service.StreamMessage(r.Context(), req.AgentURL, params, handler); err != nil {
		// Log error but don't write response since headers already sent
		h.logger.Error("stream error", "error", err)
		// Send error event
		errEvent := map[string]interface{}{
			"kind":  "error",
			"error": err.Error(),
		}
		data, _ := json.Marshal(errEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// GetTask retrieves a task status.
func (h *A2AHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	if taskID == "" {
		h.writeError(w, http.StatusBadRequest, "taskID is required")
		return
	}

	agentURL := r.URL.Query().Get("agentUrl")
	if agentURL == "" {
		h.writeError(w, http.StatusBadRequest, "agentUrl query param is required")
		return
	}

	task, err := h.service.GetTask(r.Context(), agentURL, taskID)
	if err != nil {
		h.logger.Error("failed to get task", "error", err, "taskID", taskID)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

// CancelTask cancels a running task.
func (h *A2AHandler) CancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	if taskID == "" {
		h.writeError(w, http.StatusBadRequest, "taskID is required")
		return
	}

	agentURL := r.URL.Query().Get("agentUrl")
	if agentURL == "" {
		h.writeError(w, http.StatusBadRequest, "agentUrl query param is required")
		return
	}

	if err := h.service.CancelTask(r.Context(), agentURL, taskID); err != nil {
		h.logger.Error("failed to cancel task", "error", err, "taskID", taskID)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// SessionResponse is the response for session operations.
type SessionResponse struct {
	ContextID   string    `json:"contextId"`
	AgentURL    string    `json:"agentUrl"`
	TaskCount   int       `json:"taskCount"`
	CreatedAt   time.Time `json:"createdAt"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// GetSession retrieves session information.
func (h *A2AHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	contextID := r.PathValue("contextID")
	if contextID == "" {
		h.writeError(w, http.StatusBadRequest, "contextID is required")
		return
	}

	session, ok := h.service.GetSession(contextID)
	if !ok {
		h.writeError(w, http.StatusNotFound, "session not found")
		return
	}

	resp := SessionResponse{
		ContextID:   session.ContextID,
		AgentURL:    session.AgentURL,
		TaskCount:   len(session.Tasks),
		CreatedAt:   session.CreatedAt,
		LastUpdated: session.LastUpdated,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// DeleteSession removes a session.
func (h *A2AHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	contextID := r.PathValue("contextID")
	if contextID == "" {
		h.writeError(w, http.StatusBadRequest, "contextID is required")
		return
	}

	_, ok := h.service.GetSession(contextID)
	if !ok {
		h.writeError(w, http.StatusNotFound, "session not found")
		return
	}

	h.service.DeleteSession(contextID)

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// writeJSON writes a JSON response.
func (h *A2AHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func (h *A2AHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
