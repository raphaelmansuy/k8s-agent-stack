package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// LogEntry represents a log entry in the API.
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp" doc:"Log timestamp"`
	Level     string                 `json:"level" doc:"Log level (info, warn, error)"`
	Message   string                 `json:"message" doc:"Log message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" doc:"Optional metadata"`
}

// ListLogsInput is the input for listing logs.
type ListLogsInput struct {
	AgentID      string    `query:"agent_id" doc:"Filter by agent ID"`
	DeploymentID string    `query:"deployment_id" doc:"Filter by deployment ID"`
	Level        string    `query:"level" doc:"Filter by log level"`
	StartTime    time.Time `query:"start_time" doc:"Filter by start time"`
	EndTime      time.Time `query:"end_time" doc:"Filter by end time"`
	Limit        int       `query:"limit" default:"100" minimum:"1" maximum:"1000" doc:"Maximum results"`
}

// ListLogsOutput is the output for listing logs.
type ListLogsOutput struct {
	Body struct {
		Logs       []LogEntry `json:"logs" doc:"List of log entries"`
		NextCursor string     `json:"next_cursor,omitempty" doc:"Cursor for pagination"`
	}
}

// RegisterLogsRoutes registers log-related routes.
func RegisterLogsRoutes(api huma.API, rbacM *middleware.RBACMiddleware, auditM *middleware.AuditMiddleware) {
	huma.Register(api, huma.Operation{
		OperationID: "list-logs",
		Method:      http.MethodGet,
		Path:        "/v1/logs",
		Summary:     "List logs",
		Description: "Retrieve logs for an agent or deployment. In development mode, returns mock logs.",
		Tags:        []string{"Observability"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *ListLogsInput) (*ListLogsOutput, error) {
		// For now, return mock logs to satisfy the CLI/SDK
		// In a real implementation, this would fetch from K8s or a logging service

		now := time.Now()
		logs := []LogEntry{
			{
				Timestamp: now.Add(-5 * time.Minute),
				Level:     "info",
				Message:   "Agent starting up...",
			},
			{
				Timestamp: now.Add(-4 * time.Minute),
				Level:     "info",
				Message:   "Connecting to model provider...",
			},
			{
				Timestamp: now.Add(-3 * time.Minute),
				Level:     "info",
				Message:   "Agent ready and listening for requests.",
			},
		}

		// If specific IDs are provided, customize the messages
		if input.AgentID != "" {
			logs = append(logs, LogEntry{
				Timestamp: now.Add(-1 * time.Minute),
				Level:     "info",
				Message:   "Processing request for agent " + input.AgentID,
			})
		}
		if input.DeploymentID != "" {
			logs = append(logs, LogEntry{
				Timestamp: now.Add(-1 * time.Minute),
				Level:     "info",
				Message:   "Deployment " + input.DeploymentID + " healthy",
			})
		}

		// Limit results
		if len(logs) > input.Limit {
			logs = logs[:input.Limit]
		}

		return &ListLogsOutput{
			Body: struct {
				Logs       []LogEntry `json:"logs" doc:"List of log entries"`
				NextCursor string     `json:"next_cursor,omitempty" doc:"Cursor for pagination"`
			}{
				Logs: logs,
			},
		}, nil
	})
}
