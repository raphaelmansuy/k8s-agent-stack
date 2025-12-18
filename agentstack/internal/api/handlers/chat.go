// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
)

// ChatSession represents a chat session with an agent.
type ChatSession struct {
	ID        string            `json:"id" doc:"Unique session identifier"`
	AgentID   string            `json:"agent_id" doc:"Associated agent ID"`
	UserID    string            `json:"user_id,omitempty" doc:"User who initiated the session"`
	Title     string            `json:"title,omitempty" doc:"Session title"`
	Metadata  map[string]string `json:"metadata,omitempty" doc:"Custom metadata"`
	Status    string            `json:"status" doc:"Session status" enum:"active,closed"`
	CreatedAt time.Time         `json:"created_at" doc:"Creation timestamp"`
	UpdatedAt time.Time         `json:"updated_at" doc:"Last update timestamp"`
}

// ChatMessage represents a message in a chat session.
type ChatMessage struct {
	ID        string         `json:"id" doc:"Unique message identifier"`
	SessionID string         `json:"session_id" doc:"Parent session ID"`
	Role      string         `json:"role" doc:"Message role" enum:"user,assistant,system,tool"`
	Content   string         `json:"content" doc:"Message content"`
	ToolCalls []ToolCall     `json:"tool_calls,omitempty" doc:"Tool calls made by assistant"`
	Metadata  map[string]any `json:"metadata,omitempty" doc:"Message metadata"`
	CreatedAt time.Time      `json:"created_at" doc:"Creation timestamp"`
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	ID       string `json:"id" doc:"Tool call ID"`
	Name     string `json:"name" doc:"Tool name"`
	Args     string `json:"args" doc:"Tool arguments as JSON"`
	Response string `json:"response,omitempty" doc:"Tool response"`
}

// CreateChatSessionInput is the input for creating a chat session.
type CreateChatSessionInput struct {
	Body struct {
		AgentID  string            `json:"agent_id" required:"true" doc:"Agent to chat with"`
		Title    string            `json:"title,omitempty" maxLength:"255" doc:"Optional session title"`
		Metadata map[string]string `json:"metadata,omitempty" doc:"Custom metadata"`
	}
}

// CreateChatSessionOutput is the output for creating a chat session.
type CreateChatSessionOutput struct {
	Body ChatSession
}

// GetChatSessionInput is the input for getting a chat session.
type GetChatSessionInput struct {
	ID string `path:"id" doc:"Session ID"`
}

// GetChatSessionOutput is the output for getting a chat session.
type GetChatSessionOutput struct {
	Body ChatSession
}

// ListChatMessagesInput is the input for listing messages in a session.
type ListChatMessagesInput struct {
	SessionID string `path:"session_id" doc:"Session ID"`
	Limit     int    `query:"limit" default:"50" minimum:"1" maximum:"100" doc:"Maximum messages"`
	Before    string `query:"before" doc:"Cursor for pagination (message ID)"`
}

// ListChatMessagesOutput is the output for listing messages.
type ListChatMessagesOutput struct {
	Body struct {
		Messages []ChatMessage `json:"messages" doc:"List of messages"`
		HasMore  bool          `json:"has_more" doc:"Whether more messages exist"`
	}
}

// SendMessageInput is the input for sending a message.
type SendMessageInput struct {
	SessionID string `path:"session_id" doc:"Session ID"`
	Body      struct {
		Content  string         `json:"content" required:"true" minLength:"1" doc:"Message content"`
		Metadata map[string]any `json:"metadata,omitempty" doc:"Message metadata"`
	}
}

// SendMessageOutput is the output for sending a message.
type SendMessageOutput struct {
	Body struct {
		UserMessage      ChatMessage `json:"user_message" doc:"The user's message"`
		AssistantMessage ChatMessage `json:"assistant_message" doc:"The assistant's response"`
	}
}

// RegisterChatRoutes registers chat routes.
func RegisterChatRoutes(api huma.API, pool *database.Pool, redis *cache.Client, a2aService *a2a.Service) {
	queries := db.New(pool.Pool)

	// Create chat session
	huma.Post(api, "/v1/chat/sessions", func(ctx context.Context, input *CreateChatSessionInput) (*CreateChatSessionOutput, error) {
		metadata, _ := json.Marshal(input.Body.Metadata)

		session, err := queries.CreateChatSession(ctx, db.CreateChatSessionParams{
			AgentID:  input.Body.AgentID,
			Status:   "active",
			Metadata: metadata,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to create chat session", err)
		}

		return &CreateChatSessionOutput{
			Body: ChatSession{
				ID:        session.ID,
				AgentID:   session.AgentID,
				Status:    session.Status,
				Metadata:  input.Body.Metadata,
				CreatedAt: session.CreatedAt,
				UpdatedAt: session.UpdatedAt,
			},
		}, nil
	})

	// Get chat session
	huma.Get(api, "/v1/chat/sessions/{id}", func(ctx context.Context, input *GetChatSessionInput) (*GetChatSessionOutput, error) {
		session, err := queries.GetChatSession(ctx, input.ID)
		if err != nil {
			return nil, huma.Error404NotFound("Chat session not found")
		}

		var metadata map[string]string
		if len(session.Metadata) > 0 {
			json.Unmarshal(session.Metadata, &metadata)
		}

		return &GetChatSessionOutput{
			Body: ChatSession{
				ID:        session.ID,
				AgentID:   session.AgentID,
				Status:    session.Status,
				Metadata:  metadata,
				CreatedAt: session.CreatedAt,
				UpdatedAt: session.UpdatedAt,
			},
		}, nil
	})

	// List messages in session
	huma.Get(api, "/v1/chat/sessions/{session_id}/messages", func(ctx context.Context, input *ListChatMessagesInput) (*ListChatMessagesOutput, error) {
		messages, err := queries.ListChatMessages(ctx, input.SessionID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list messages", err)
		}

		chatMessages := make([]ChatMessage, len(messages))
		for i, m := range messages {
			var toolCalls []ToolCall
			if len(m.ToolCalls) > 0 {
				json.Unmarshal(m.ToolCalls, &toolCalls)
			}
			var metadata map[string]any
			if len(m.Metadata) > 0 {
				json.Unmarshal(m.Metadata, &metadata)
			}

			chatMessages[i] = ChatMessage{
				ID:        m.ID,
				SessionID: m.SessionID,
				Role:      m.Role,
				Content:   m.Content,
				ToolCalls: toolCalls,
				Metadata:  metadata,
				CreatedAt: m.CreatedAt,
			}
		}

		return &ListChatMessagesOutput{
			Body: struct {
				Messages []ChatMessage `json:"messages" doc:"List of messages"`
				HasMore  bool          `json:"has_more" doc:"Whether more messages exist"`
			}{
				Messages: chatMessages,
				HasMore:  false,
			},
		}, nil
	})

	// Send message (synchronous)
	huma.Post(api, "/v1/chat/sessions/{session_id}/messages", func(ctx context.Context, input *SendMessageInput) (*SendMessageOutput, error) {
		// 1. Get session to find agent ID
		session, err := queries.GetChatSession(ctx, input.SessionID)
		if err != nil {
			return nil, huma.Error404NotFound("Chat session not found")
		}

		// 2. Save user message
		userMetadata, _ := json.Marshal(input.Body.Metadata)
		userMsg, err := queries.CreateChatMessage(ctx, db.CreateChatMessageParams{
			SessionID: input.SessionID,
			Role:      "user",
			Content:   input.Body.Content,
			Metadata:  userMetadata,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to save user message", err)
		}

		// 3. Call agent via A2A
		// For now, we use a convention for the agent endpoint
		// In a real system, this would be looked up in the agent registry/database
		agentEndpoint := fmt.Sprintf("http://%s.kagent.svc.cluster.local:8080", session.AgentID)

		// If we are running in the same namespace or using full DNS
		// The agent we deployed is google-adk-byo-agent

		task, err := a2aService.SendMessage(ctx, agentEndpoint, &a2a.SendMessageParams{
			Message: a2a.MessageInput{
				MessageID: userMsg.ID,
				ContextID: input.SessionID,
				Role:      "user",
				Parts: []a2a.Part{
					a2a.TextPart(input.Body.Content),
				},
			},
		})
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("Failed to call agent at %s", agentEndpoint), err)
		}

		// Extract response content
		var assistantContent string
		if task.Status != nil && task.Status.Message != nil {
			for _, part := range task.Status.Message.Parts {
				if part.Kind == "text" {
					assistantContent += part.Text
				}
			}
		}

		// If no content in message, check artifacts (common in some ADK agents)
		if assistantContent == "" && len(task.Artifacts) > 0 {
			for _, artifact := range task.Artifacts {
				for _, part := range artifact.Parts {
					if part.Kind == "text" {
						assistantContent += part.Text
					}
				}
			}
		}

		// 4. Save assistant message
		assistantMetadata, _ := json.Marshal(task.Metadata)
		assistantMsg, err := queries.CreateChatMessage(ctx, db.CreateChatMessageParams{
			SessionID: input.SessionID,
			Role:      "assistant",
			Content:   assistantContent,
			Metadata:  assistantMetadata,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to save assistant message", err)
		}

		return &SendMessageOutput{
			Body: struct {
				UserMessage      ChatMessage `json:"user_message" doc:"The user's message"`
				AssistantMessage ChatMessage `json:"assistant_message" doc:"The assistant's response"`
			}{
				UserMessage: ChatMessage{
					ID:        userMsg.ID,
					SessionID: userMsg.SessionID,
					Role:      userMsg.Role,
					Content:   userMsg.Content,
					CreatedAt: userMsg.CreatedAt,
				},
				AssistantMessage: ChatMessage{
					ID:        assistantMsg.ID,
					SessionID: assistantMsg.SessionID,
					Role:      assistantMsg.Role,
					Content:   assistantMsg.Content,
					CreatedAt: assistantMsg.CreatedAt,
				},
			},
		}, nil
	})
}
