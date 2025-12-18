// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
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
func RegisterChatRoutes(api huma.API, db *database.Pool, redis *cache.Client) {
	// Create chat session
	huma.Post(api, "/v1/chat/sessions", func(ctx context.Context, input *CreateChatSessionInput) (*CreateChatSessionOutput, error) {
		now := time.Now()
		session := ChatSession{
			ID:        uuid.New().String(),
			AgentID:   input.Body.AgentID,
			UserID:    "", // TODO: Get from auth context
			Title:     input.Body.Title,
			Metadata:  input.Body.Metadata,
			Status:    "active",
			CreatedAt: now,
			UpdatedAt: now,
		}

		// TODO: Save to database

		return &CreateChatSessionOutput{Body: session}, nil
	})

	// Get chat session
	huma.Get(api, "/v1/chat/sessions/{id}", func(ctx context.Context, input *GetChatSessionInput) (*GetChatSessionOutput, error) {
		now := time.Now()
		// TODO: Fetch from database
		return &GetChatSessionOutput{
			Body: ChatSession{
				ID:        input.ID,
				AgentID:   "agent-1",
				Status:    "active",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}, nil
	})

	// List messages in session
	huma.Get(api, "/v1/chat/sessions/{session_id}/messages", func(ctx context.Context, input *ListChatMessagesInput) (*ListChatMessagesOutput, error) {
		// TODO: Fetch from database
		return &ListChatMessagesOutput{
			Body: struct {
				Messages []ChatMessage `json:"messages" doc:"List of messages"`
				HasMore  bool          `json:"has_more" doc:"Whether more messages exist"`
			}{
				Messages: []ChatMessage{},
				HasMore:  false,
			},
		}, nil
	})

	// Send message (synchronous)
	huma.Post(api, "/v1/chat/sessions/{session_id}/messages", func(ctx context.Context, input *SendMessageInput) (*SendMessageOutput, error) {
		now := time.Now()

		userMsg := ChatMessage{
			ID:        uuid.New().String(),
			SessionID: input.SessionID,
			Role:      "user",
			Content:   input.Body.Content,
			Metadata:  input.Body.Metadata,
			CreatedAt: now,
		}

		// TODO: Call agent and get response
		assistantMsg := ChatMessage{
			ID:        uuid.New().String(),
			SessionID: input.SessionID,
			Role:      "assistant",
			Content:   "This is a placeholder response. Agent integration pending.",
			CreatedAt: now.Add(time.Millisecond * 100),
		}

		// TODO: Save messages to database

		return &SendMessageOutput{
			Body: struct {
				UserMessage      ChatMessage `json:"user_message" doc:"The user's message"`
				AssistantMessage ChatMessage `json:"assistant_message" doc:"The assistant's response"`
			}{
				UserMessage:      userMsg,
				AssistantMessage: assistantMsg,
			},
		}, nil
	})
}
