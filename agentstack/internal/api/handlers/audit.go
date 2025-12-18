// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// QueryAuditEventsInput is the input for querying audit events.
type QueryAuditEventsInput struct {
	TeamID     string    `path:"id" doc:"Team ID"`
	ProjectID  string    `query:"project_id" doc:"Filter by project ID"`
	ActorID    string    `query:"actor_id" doc:"Filter by actor ID"`
	Resource   string    `query:"resource" doc:"Filter by resource type"`
	ResourceID string    `query:"resource_id" doc:"Filter by resource ID"`
	Result     string    `query:"result" doc:"Filter by result"`
	EventTypes []string  `query:"event_type" doc:"Filter by event types"`
	StartTime  time.Time `query:"start_time" doc:"Filter by start time"`
	EndTime    time.Time `query:"end_time" doc:"Filter by end time"`
	Limit      int       `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Maximum results"`
	Offset     int       `query:"offset" default:"0" minimum:"0" doc:"Pagination offset"`
}

// QueryAuditEventsOutput is the output for querying audit events.
type QueryAuditEventsOutput struct {
	Body struct {
		Events []audit.Event  `json:"events" doc:"List of audit events"`
		Filter map[string]any `json:"filter" doc:"Applied filter"`
	}
}

// GetAuditEventInput is the input for getting an audit event.
type GetAuditEventInput struct {
	ID string `path:"id" doc:"Event ID"`
}

// GetAuditEventOutput is the output for getting an audit event.
type GetAuditEventOutput struct {
	Body audit.Event
}

// GetAuditEventTypesOutput is the output for getting audit event types.
type GetAuditEventTypesOutput struct {
	Body struct {
		EventTypes []map[string]string `json:"event_types" doc:"List of available event types"`
	}
}

// RegisterAuditRoutes registers audit routes.
func RegisterAuditRoutes(api huma.API, service *audit.Service, rbacM *middleware.RBACMiddleware) {
	// Query events
	huma.Register(api, huma.Operation{
		OperationID: "query-audit-events",
		Method:      http.MethodGet,
		Path:        "/v1/teams/{id}/audit",
		Summary:     "Query audit events",
		Tags:        []string{"Audit"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAuditLog, rbac.ActionList),
		},
	}, func(ctx context.Context, input *QueryAuditEventsInput) (*QueryAuditEventsOutput, error) {
		filter := audit.Filter{
			TeamID:     input.TeamID,
			ProjectID:  input.ProjectID,
			ActorID:    input.ActorID,
			Resource:   input.Resource,
			ResourceID: input.ResourceID,
			Result:     input.Result,
			StartTime:  input.StartTime,
			EndTime:    input.EndTime,
			Limit:      input.Limit,
			Offset:     input.Offset,
		}

		for _, et := range input.EventTypes {
			filter.EventTypes = append(filter.EventTypes, audit.EventType(et))
		}

		events, err := service.Query(ctx, filter)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to query audit events", err)
		}

		return &QueryAuditEventsOutput{
			Body: struct {
				Events []audit.Event  `json:"events" doc:"List of audit events"`
				Filter map[string]any `json:"filter" doc:"Applied filter"`
			}{
				Events: events,
				Filter: map[string]any{
					"team_id":    filter.TeamID,
					"project_id": filter.ProjectID,
					"start_time": filter.StartTime,
					"end_time":   filter.EndTime,
					"limit":      filter.Limit,
					"offset":     filter.Offset,
				},
			},
		}, nil
	})

	// Get event
	huma.Register(api, huma.Operation{
		OperationID: "get-audit-event",
		Method:      http.MethodGet,
		Path:        "/v1/audit/{id}",
		Summary:     "Get audit event",
		Tags:        []string{"Audit"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAuditLog, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetAuditEventInput) (*GetAuditEventOutput, error) {
		events, err := service.Query(ctx, audit.Filter{
			ResourceID: input.ID,
			Limit:      1,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get audit event", err)
		}

		if len(events) == 0 {
			return nil, huma.Error404NotFound("Audit event not found")
		}

		return &GetAuditEventOutput{
			Body: events[0],
		}, nil
	})

	// Get event types
	huma.Register(api, huma.Operation{
		OperationID: "get-audit-event-types",
		Method:      http.MethodGet,
		Path:        "/v1/audit/event-types",
		Summary:     "Get audit event types",
		Tags:        []string{"Audit"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAuditLog, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *struct{}) (*GetAuditEventTypesOutput, error) {
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

		return &GetAuditEventTypesOutput{
			Body: struct {
				EventTypes []map[string]string `json:"event_types" doc:"List of available event types"`
			}{
				EventTypes: eventTypes,
			},
		}, nil
	})
}
