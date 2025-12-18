// Package handlers provides HTTP handlers for the AgentStack API.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/deployment"
)

// DeploymentHandler handles agent deployment HTTP requests.
type DeploymentHandler struct {
	service *deployment.Service
	logger  *slog.Logger
}

// NewDeploymentHandler creates a new deployment handler.
func NewDeploymentHandler(service *deployment.Service, logger *slog.Logger) *DeploymentHandler {
	return &DeploymentHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers deployment routes on the given mux.
func (h *DeploymentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /agents", h.CreateAgent)
	mux.HandleFunc("GET /agents", h.ListAgents)
	mux.HandleFunc("GET /agents/{agentID}", h.GetAgent)
	mux.HandleFunc("PATCH /agents/{agentID}", h.UpdateAgent)
	mux.HandleFunc("DELETE /agents/{agentID}", h.DeleteAgent)
	mux.HandleFunc("GET /agents/{agentID}/status", h.GetAgentStatus)
	mux.HandleFunc("POST /agents/{agentID}/wait", h.WaitForReady)
}

// CreateAgentRequest is the request body for creating an agent.
type CreateAgentRequest struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Image       string                   `json:"image"`
	Namespace   string                   `json:"namespace,omitempty"`
	Type        deployment.AgentType     `json:"type,omitempty"`
	Env         map[string]string        `json:"env,omitempty"`
	Resources   *deployment.ResourceSpec `json:"resources,omitempty"`
	Replicas    int32                    `json:"replicas,omitempty"`
	Model       *deployment.ModelConfig  `json:"model,omitempty"`
	Tools       []deployment.ToolSpec    `json:"tools,omitempty"`
}

// CreateAgent handles agent creation.
func (h *DeploymentHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var req CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ID == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if req.Image == "" {
		h.writeError(w, http.StatusBadRequest, "image is required")
		return
	}

	agent := &deployment.Agent{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Image:       req.Image,
		Namespace:   req.Namespace,
		Type:        req.Type,
		Env:         req.Env,
		Resources:   req.Resources,
		Replicas:    req.Replicas,
		Model:       req.Model,
		Tools:       req.Tools,
	}

	created, err := h.service.CreateAgent(r.Context(), agent)
	if err != nil {
		h.logger.Error("failed to create agent", "error", err)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusCreated, created)
}

// ListAgents handles listing agents.
func (h *DeploymentHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")

	agents, err := h.service.ListAgents(r.Context(), namespace)
	if err != nil {
		h.logger.Error("failed to list agents", "error", err)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"agents": agents,
		"count":  len(agents),
	})
}

// GetAgent handles retrieving a single agent.
func (h *DeploymentHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentID")
	if agentID == "" {
		h.writeError(w, http.StatusBadRequest, "agentID is required")
		return
	}

	agent, err := h.service.GetAgent(r.Context(), agentID)
	if err != nil {
		h.logger.Error("failed to get agent", "error", err, "agentID", agentID)
		h.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, agent)
}

// UpdateAgentRequest is the request body for updating an agent.
type UpdateAgentRequest struct {
	Image       string                   `json:"image,omitempty"`
	Description string                   `json:"description,omitempty"`
	Env         map[string]string        `json:"env,omitempty"`
	Resources   *deployment.ResourceSpec `json:"resources,omitempty"`
}

// UpdateAgent handles updating an agent.
func (h *DeploymentHandler) UpdateAgent(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentID")
	if agentID == "" {
		h.writeError(w, http.StatusBadRequest, "agentID is required")
		return
	}

	var req UpdateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	update := &deployment.Agent{
		Image:       req.Image,
		Description: req.Description,
		Env:         req.Env,
		Resources:   req.Resources,
	}

	updated, err := h.service.UpdateAgent(r.Context(), agentID, update)
	if err != nil {
		h.logger.Error("failed to update agent", "error", err, "agentID", agentID)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, updated)
}

// DeleteAgent handles deleting an agent.
func (h *DeploymentHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentID")
	if agentID == "" {
		h.writeError(w, http.StatusBadRequest, "agentID is required")
		return
	}

	if err := h.service.DeleteAgent(r.Context(), agentID); err != nil {
		h.logger.Error("failed to delete agent", "error", err, "agentID", agentID)
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetAgentStatus handles retrieving agent status.
func (h *DeploymentHandler) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentID")
	if agentID == "" {
		h.writeError(w, http.StatusBadRequest, "agentID is required")
		return
	}

	agent, err := h.service.GetAgent(r.Context(), agentID)
	if err != nil {
		h.logger.Error("failed to get agent", "error", err, "agentID", agentID)
		h.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, agent.Status)
}

// WaitForReadyRequest is the request body for waiting for agent readiness.
type WaitForReadyRequest struct {
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
}

// WaitForReady handles waiting for an agent to be ready.
func (h *DeploymentHandler) WaitForReady(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentID")
	if agentID == "" {
		h.writeError(w, http.StatusBadRequest, "agentID is required")
		return
	}

	var req WaitForReadyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use default timeout
		req.TimeoutSeconds = 300
	}

	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 300
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second

	if err := h.service.WaitForReady(r.Context(), agentID, timeout); err != nil {
		h.logger.Error("failed waiting for agent", "error", err, "agentID", agentID)
		h.writeError(w, http.StatusGatewayTimeout, err.Error())
		return
	}

	agent, err := h.service.GetAgent(r.Context(), agentID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, agent)
}

// writeJSON writes a JSON response.
func (h *DeploymentHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func (h *DeploymentHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
