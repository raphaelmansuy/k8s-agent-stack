// Package handlers provides HTTP handlers for the AgentStack API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// A2ASendMessageInput is the input for sending a message.
type A2ASendMessageInput struct {
	Body struct {
		AgentURL  string                 `json:"agentUrl" required:"true" doc:"Agent URL"`
		ContextID string                 `json:"contextId,omitempty" doc:"Context ID"`
		Content   string                 `json:"content" required:"true" doc:"Message content"`
		Metadata  map[string]interface{} `json:"metadata,omitempty" doc:"Custom metadata"`
	}
}

// A2ASendMessageOutput is the output for sending a message.
type A2ASendMessageOutput struct {
	Body struct {
		TaskID    string          `json:"taskId" doc:"Task ID"`
		ContextID string          `json:"contextId" doc:"Context ID"`
		Status    *a2a.TaskStatus `json:"status" doc:"Task status"`
	}
}

// StreamMessageInput is the input for streaming messages.
type StreamMessageInput struct {
	Body struct {
		AgentURL  string                 `json:"agentUrl" required:"true" doc:"Agent URL"`
		ContextID string                 `json:"contextId,omitempty" doc:"Context ID"`
		Content   string                 `json:"content" required:"true" doc:"Message content"`
		Metadata  map[string]interface{} `json:"metadata,omitempty" doc:"Custom metadata"`
	}
}

// GetTaskInput is the input for getting a task.
type GetTaskInput struct {
	TaskID   string `path:"taskID" doc:"Task ID"`
	AgentURL string `query:"agentUrl" required:"true" doc:"Agent URL"`
}

// GetTaskOutput is the output for getting a task.
type GetTaskOutput struct {
	Body *a2a.Task
}

// CancelTaskInput is the input for cancelling a task.
type CancelTaskInput struct {
	TaskID   string `path:"taskID" doc:"Task ID"`
	AgentURL string `query:"agentUrl" required:"true" doc:"Agent URL"`
}

// CancelTaskOutput is the output for cancelling a task.
type CancelTaskOutput struct {
	Body struct {
		Status string `json:"status" doc:"Cancellation status"`
	}
}

// GetSessionInput is the input for getting a session.
type GetSessionInput struct {
	ContextID string `path:"contextID" doc:"Context ID"`
}

// GetSessionOutput is the output for getting a session.
type GetSessionOutput struct {
	Body struct {
		ContextID   string    `json:"contextId" doc:"Context ID"`
		AgentURL    string    `json:"agentUrl" doc:"Agent URL"`
		TaskCount   int       `json:"taskCount" doc:"Number of tasks"`
		CreatedAt   time.Time `json:"createdAt" doc:"Creation timestamp"`
		LastUpdated time.Time `json:"lastUpdated" doc:"Last update timestamp"`
	}
}

// DeleteSessionInput is the input for deleting a session.
type DeleteSessionInput struct {
	ContextID string `path:"contextID" doc:"Context ID"`
}

// DeleteSessionOutput is the output for deleting a session.
type DeleteSessionOutput struct {
	Body struct {
		Status string `json:"status" doc:"Deletion status"`
	}
}

// RegisterA2ARoutes registers A2A routes.
func RegisterA2ARoutes(api huma.API, service *a2a.Service, rbacM *middleware.RBACMiddleware, auditM *middleware.AuditMiddleware) {
	// Send message
	huma.Register(api, huma.Operation{
		OperationID: "a2a-send-message",
		Method:      http.MethodPost,
		Path:        "/v1/a2a/send",
		Summary:     "Send A2A message",
		Tags:        []string{"A2A"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionInvoke),
			auditM.HumaLogAction(audit.EventAgentInvoked, string(rbac.ResourceAgent)),
		},
	}, func(ctx context.Context, input *A2ASendMessageInput) (*A2ASendMessageOutput, error) {
		params := &a2a.SendMessageParams{
			Message: a2a.MessageInput{
				ContextID: input.Body.ContextID,
				Parts:     []a2a.Part{a2a.TextPart(input.Body.Content)},
			},
		}

		task, err := service.SendMessage(ctx, input.Body.AgentURL, params)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to send message", err)
		}

		return &A2ASendMessageOutput{
			Body: struct {
				TaskID    string          `json:"taskId" doc:"Task ID"`
				ContextID string          `json:"contextId" doc:"Context ID"`
				Status    *a2a.TaskStatus `json:"status" doc:"Task status"`
			}{
				TaskID:    task.TaskID,
				ContextID: task.ContextID,
				Status:    task.Status,
			},
		}, nil
	})

	// Stream message
	huma.Register(api, huma.Operation{
		OperationID: "a2a-stream-message",
		Method:      http.MethodPost,
		Path:        "/v1/a2a/stream",
		Summary:     "Stream A2A message",
		Tags:        []string{"A2A"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionInvoke),
		},
	}, func(ctx context.Context, input *StreamMessageInput) (*huma.StreamResponse, error) {
		return &huma.StreamResponse{
			Body: func(w huma.Context) {
				rw := w.BodyWriter()

				// Set SSE headers
				w.SetHeader("Content-Type", "text/event-stream")
				w.SetHeader("Cache-Control", "no-cache")
				w.SetHeader("Connection", "keep-alive")
				w.SetHeader("X-Accel-Buffering", "no")

				flusher, ok := rw.(http.Flusher)
				if !ok {
					return
				}

				params := &a2a.SendMessageParams{
					Message: a2a.MessageInput{
						ContextID: input.Body.ContextID,
						Parts:     []a2a.Part{a2a.TextPart(input.Body.Content)},
					},
				}

				handler := func(event interface{}) error {
					data, err := json.Marshal(event)
					if err != nil {
						return err
					}

					fmt.Fprintf(rw, "data: %s\n\n", data)
					flusher.Flush()
					return nil
				}

				if err := service.StreamMessage(ctx, input.Body.AgentURL, params, handler); err != nil {
					errEvent := map[string]interface{}{
						"kind":  "error",
						"error": err.Error(),
					}
					data, _ := json.Marshal(errEvent)
					fmt.Fprintf(rw, "data: %s\n\n", data)
					flusher.Flush()
				}
			},
		}, nil
	})

	// Get task
	huma.Register(api, huma.Operation{
		OperationID: "a2a-get-task",
		Method:      http.MethodGet,
		Path:        "/v1/a2a/task/{taskID}",
		Summary:     "Get A2A task",
		Tags:        []string{"A2A"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetTaskInput) (*GetTaskOutput, error) {
		task, err := service.GetTask(ctx, input.AgentURL, input.TaskID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get task", err)
		}

		return &GetTaskOutput{
			Body: task,
		}, nil
	})

	// Cancel task
	huma.Register(api, huma.Operation{
		OperationID: "a2a-cancel-task",
		Method:      http.MethodDelete,
		Path:        "/v1/a2a/task/{taskID}",
		Summary:     "Cancel A2A task",
		Tags:        []string{"A2A"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionUpdate),
		},
	}, func(ctx context.Context, input *CancelTaskInput) (*CancelTaskOutput, error) {
		if err := service.CancelTask(ctx, input.AgentURL, input.TaskID); err != nil {
			return nil, huma.Error500InternalServerError("Failed to cancel task", err)
		}

		return &CancelTaskOutput{
			Body: struct {
				Status string `json:"status" doc:"Cancellation status"`
			}{
				Status: "cancelled",
			},
		}, nil
	})

	// Get session
	huma.Register(api, huma.Operation{
		OperationID: "a2a-get-session",
		Method:      http.MethodGet,
		Path:        "/v1/a2a/session/{contextID}",
		Summary:     "Get A2A session",
		Tags:        []string{"A2A"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionRead),
		},
	}, func(ctx context.Context, input *GetSessionInput) (*GetSessionOutput, error) {
		session, ok := service.GetSession(input.ContextID)
		if !ok {
			return nil, huma.Error404NotFound("Session not found")
		}

		return &GetSessionOutput{
			Body: struct {
				ContextID   string    `json:"contextId" doc:"Context ID"`
				AgentURL    string    `json:"agentUrl" doc:"Agent URL"`
				TaskCount   int       `json:"taskCount" doc:"Number of tasks"`
				CreatedAt   time.Time `json:"createdAt" doc:"Creation timestamp"`
				LastUpdated time.Time `json:"lastUpdated" doc:"Last update timestamp"`
			}{
				ContextID:   session.ContextID,
				AgentURL:    session.AgentURL,
				TaskCount:   len(session.Tasks),
				CreatedAt:   session.CreatedAt,
				LastUpdated: session.LastUpdated,
			},
		}, nil
	})

	// Delete session
	huma.Register(api, huma.Operation{
		OperationID: "a2a-delete-session",
		Method:      http.MethodDelete,
		Path:        "/v1/a2a/session/{contextID}",
		Summary:     "Delete A2A session",
		Tags:        []string{"A2A"},
		Middlewares: huma.Middlewares{
			rbacM.HumaRequirePermission(rbac.ResourceAgent, rbac.ActionDelete),
		},
	}, func(ctx context.Context, input *DeleteSessionInput) (*DeleteSessionOutput, error) {
		if _, ok := service.GetSession(input.ContextID); !ok {
			return nil, huma.Error404NotFound("Session not found")
		}

		service.DeleteSession(input.ContextID)

		return &DeleteSessionOutput{
			Body: struct {
				Status string `json:"status" doc:"Deletion status"`
			}{
				Status: "deleted",
			},
		}, nil
	})
}
