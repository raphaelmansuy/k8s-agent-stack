// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
)

// AuditHandler handles audit-related HTTP requests.
type AuditHandler struct {
	auditSvc *audit.Service
}

// NewAuditHandler creates a new audit handler.
func NewAuditHandler(auditSvc *audit.Service) *AuditHandler {
	return &AuditHandler{auditSvc: auditSvc}
}

// QueryEvents queries audit events.
// GET /api/v1/teams/{id}/audit
func (h *AuditHandler) QueryEvents(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("id")
	if teamID == "" {
		writeError(w, http.StatusBadRequest, "team ID is required", nil)
		return
	}

	filter := audit.Filter{
		TeamID: teamID,
	}

	// Parse optional query parameters
	if projectID := r.URL.Query().Get("project_id"); projectID != "" {
		filter.ProjectID = projectID
	}
	if actorID := r.URL.Query().Get("actor_id"); actorID != "" {
		filter.ActorID = actorID
	}
	if resource := r.URL.Query().Get("resource"); resource != "" {
		filter.Resource = resource
	}
	if resourceID := r.URL.Query().Get("resource_id"); resourceID != "" {
		filter.ResourceID = resourceID
	}
	if result := r.URL.Query().Get("result"); result != "" {
		filter.Result = result
	}

	// Parse event types
	if eventTypes := r.URL.Query()["event_type"]; len(eventTypes) > 0 {
		for _, et := range eventTypes {
			filter.EventTypes = append(filter.EventTypes, audit.EventType(et))
		}
	}

	// Parse time range
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = startTime
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = endTime
		}
	}

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	events, err := h.auditSvc.Query(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query events", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"events": events,
		"filter": map[string]any{
			"team_id":    filter.TeamID,
			"project_id": filter.ProjectID,
			"start_time": filter.StartTime,
			"end_time":   filter.EndTime,
			"limit":      filter.Limit,
			"offset":     filter.Offset,
		},
	})
}

// GetEvent returns a specific audit event.
// GET /api/v1/audit/{id}
func (h *AuditHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	if eventID == "" {
		writeError(w, http.StatusBadRequest, "event ID is required", nil)
		return
	}

	// Query for the specific event
	events, err := h.auditSvc.Query(r.Context(), audit.Filter{
		ResourceID: eventID,
		Limit:      1,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get event", err)
		return
	}

	if len(events) == 0 {
		writeError(w, http.StatusNotFound, "event not found", nil)
		return
	}

	writeJSON(w, http.StatusOK, events[0])
}

// GetEventTypes returns available audit event types.
// GET /api/v1/audit/event-types
func (h *AuditHandler) GetEventTypes(w http.ResponseWriter, r *http.Request) {
	eventTypes := []map[string]string{
		{"type": string(audit.EventLogin), "category": "auth", "description": "User login"},
		{"type": string(audit.EventLogout), "category": "auth", "description": "User logout"},
		{"type": string(audit.EventAPIKeyCreated), "category": "auth", "description": "API key created"},
		{"type": string(audit.EventAPIKeyRevoked), "category": "auth", "description": "API key revoked"},
		{"type": string(audit.EventAgentCreated), "category": "agent", "description": "Agent created"},
		{"type": string(audit.EventAgentUpdated), "category": "agent", "description": "Agent updated"},
		{"type": string(audit.EventAgentDeleted), "category": "agent", "description": "Agent deleted"},
		{"type": string(audit.EventAgentDeployed), "category": "agent", "description": "Agent deployed"},
		{"type": string(audit.EventAgentInvoked), "category": "agent", "description": "Agent invoked"},
		{"type": string(audit.EventProjectCreated), "category": "project", "description": "Project created"},
		{"type": string(audit.EventProjectUpdated), "category": "project", "description": "Project updated"},
		{"type": string(audit.EventProjectDeleted), "category": "project", "description": "Project deleted"},
		{"type": string(audit.EventDeploymentCreated), "category": "deployment", "description": "Deployment created"},
		{"type": string(audit.EventDeploymentUpdated), "category": "deployment", "description": "Deployment updated"},
		{"type": string(audit.EventDeploymentDeleted), "category": "deployment", "description": "Deployment deleted"},
		{"type": string(audit.EventDeploymentScaled), "category": "deployment", "description": "Deployment scaled"},
		{"type": string(audit.EventDeploymentRestart), "category": "deployment", "description": "Deployment restarted"},
		{"type": string(audit.EventMemberInvited), "category": "team", "description": "Team member invited"},
		{"type": string(audit.EventMemberRemoved), "category": "team", "description": "Team member removed"},
		{"type": string(audit.EventRoleAssigned), "category": "rbac", "description": "Role assigned"},
		{"type": string(audit.EventRoleRevoked), "category": "rbac", "description": "Role revoked"},
		{"type": string(audit.EventPermissionDenied), "category": "access", "description": "Permission denied"},
		{"type": string(audit.EventQuotaExceeded), "category": "quota", "description": "Quota exceeded"},
		{"type": string(audit.EventRateLimited), "category": "quota", "description": "Rate limited"},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"event_types": eventTypes,
	})
}
