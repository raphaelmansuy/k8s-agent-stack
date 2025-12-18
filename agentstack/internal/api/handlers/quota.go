// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/raphaelmansuy/agentstack/internal/domain/quota"
)

// QuotaHandler handles quota-related HTTP requests.
type QuotaHandler struct {
	quotaSvc *quota.Service
}

// NewQuotaHandler creates a new quota handler.
func NewQuotaHandler(quotaSvc *quota.Service) *QuotaHandler {
	return &QuotaHandler{quotaSvc: quotaSvc}
}

// GetUsageSummary returns usage summary for a team.
// GET /api/v1/teams/{id}/usage
func (h *QuotaHandler) GetUsageSummary(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("id")
	if teamID == "" {
		writeError(w, http.StatusBadRequest, "team ID is required", nil)
		return
	}

	summary, err := h.quotaSvc.GetUsageSummary(r.Context(), teamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get usage summary", err)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// GetQuotas returns all quotas for a team.
// GET /api/v1/teams/{id}/quotas
func (h *QuotaHandler) GetQuotas(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("id")
	if teamID == "" {
		writeError(w, http.StatusBadRequest, "team ID is required", nil)
		return
	}

	// Get usage summary which includes all quotas
	summary, err := h.quotaSvc.GetUsageSummary(r.Context(), teamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get quotas", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"quotas": summary.Items,
	})
}

// SetQuotaRequest represents a request to set a quota.
type SetQuotaRequest struct {
	Type      quota.QuotaType `json:"type"`
	Limit     int64           `json:"limit"`
	Period    string          `json:"period,omitempty"`
	ProjectID string          `json:"project_id,omitempty"`
}

// SetQuota sets a quota for a team.
// POST /api/v1/teams/{id}/quotas
func (h *QuotaHandler) SetQuota(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("id")
	if teamID == "" {
		writeError(w, http.StatusBadRequest, "team ID is required", nil)
		return
	}

	var req SetQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "quota type is required", nil)
		return
	}

	q := &quota.Quota{
		TeamID:    teamID,
		ProjectID: req.ProjectID,
		Type:      req.Type,
		Limit:     req.Limit,
		Period:    req.Period,
	}

	if err := h.quotaSvc.SetQuota(r.Context(), q); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set quota", err)
		return
	}

	writeJSON(w, http.StatusOK, q)
}

// CheckQuotaRequest represents a request to check a quota.
type CheckQuotaRequest struct {
	Type      quota.QuotaType `json:"type"`
	Amount    int64           `json:"amount"`
	ProjectID string          `json:"project_id,omitempty"`
}

// CheckQuota checks if a quota allows an operation.
// POST /api/v1/teams/{id}/quotas/check
func (h *QuotaHandler) CheckQuota(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("id")
	if teamID == "" {
		writeError(w, http.StatusBadRequest, "team ID is required", nil)
		return
	}

	var req CheckQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "quota type is required", nil)
		return
	}

	if req.Amount == 0 {
		req.Amount = 1
	}

	result, err := h.quotaSvc.CheckQuota(r.Context(), quota.CheckQuotaRequest{
		TeamID:    teamID,
		ProjectID: req.ProjectID,
		Type:      req.Type,
		Amount:    req.Amount,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check quota", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ApplyPlanRequest represents a request to apply a plan.
type ApplyPlanRequest struct {
	PlanID string `json:"plan_id"`
}

// ApplyPlan applies a subscription plan to a team.
// POST /api/v1/teams/{id}/plan
func (h *QuotaHandler) ApplyPlan(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("id")
	if teamID == "" {
		writeError(w, http.StatusBadRequest, "team ID is required", nil)
		return
	}

	var req ApplyPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if req.PlanID == "" {
		writeError(w, http.StatusBadRequest, "plan_id is required", nil)
		return
	}

	plan := quota.GetPlan(req.PlanID)
	if plan == nil {
		writeError(w, http.StatusNotFound, "plan not found", nil)
		return
	}

	if err := h.quotaSvc.ApplyPlan(r.Context(), teamID, req.PlanID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to apply plan", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "plan applied successfully",
		"plan":    plan,
	})
}

// ListPlans lists available subscription plans.
// GET /api/v1/plans
func (h *QuotaHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"plans": quota.DefaultPlans,
	})
}
