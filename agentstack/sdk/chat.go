// Package sdk provides the AgentStack Go SDK for programmatic access to the API.
package sdk

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChatService handles chat-related operations.
type ChatService struct {
	client *Client
}

// Send sends a chat message and returns the response.
func (s *ChatService) Send(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	req.Stream = false // Ensure non-streaming
	var response ChatResponse
	if err := s.client.Post(ctx, "/api/v1/chat", req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// StreamResponse represents a streaming chat response.
type StreamResponse struct {
	Events <-chan StreamDelta
	Errors <-chan error
	Done   <-chan struct{}
}

// Stream sends a chat message and returns a streaming response.
func (s *ChatService) Stream(ctx context.Context, req *ChatRequest) (*StreamResponse, error) {
	req.Stream = true

	httpReq, err := s.client.Request(ctx, http.MethodPost, "/api/v1/chat", req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(body),
			}
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       errResp.Error.Code,
			Message:    errResp.Error.Message,
		}
	}

	events := make(chan StreamDelta, 100)
	errors := make(chan error, 1)
	done := make(chan struct{})

	go s.readSSEStream(ctx, resp.Body, events, errors, done)

	return &StreamResponse{
		Events: events,
		Errors: errors,
		Done:   done,
	}, nil
}

// readSSEStream reads the SSE stream and sends events to the channels.
func (s *ChatService) readSSEStream(ctx context.Context, body io.ReadCloser, events chan<- StreamDelta, errors chan<- error, done chan<- struct{}) {
	defer close(events)
	defer close(errors)
	defer close(done)
	defer body.Close()

	reader := bufio.NewReader(body)

	for {
		select {
		case <-ctx.Done():
			errors <- ctx.Err()
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			errors <- fmt.Errorf("error reading stream: %w", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle SSE format: "data: {...}"
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for end of stream marker
			if data == "[DONE]" {
				return
			}

			var delta StreamDelta
			if err := json.Unmarshal([]byte(data), &delta); err != nil {
				// Try to parse as a simple content string
				delta = StreamDelta{Content: data}
			}

			select {
			case events <- delta:
			case <-ctx.Done():
				errors <- ctx.Err()
				return
			}

			if delta.Done {
				return
			}
		}

		// Handle "event:" lines (optional metadata)
		if strings.HasPrefix(line, "event:") {
			// Event type, can be used for metadata but we focus on data
			continue
		}
	}
}

// SendSimple sends a simple text message to an agent.
func (s *ChatService) SendSimple(ctx context.Context, agentID, message string) (*ChatResponse, error) {
	req := &ChatRequest{
		AgentID: agentID,
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: message,
			},
		},
	}
	return s.Send(ctx, req)
}

// StreamSimple streams a simple text message to an agent.
func (s *ChatService) StreamSimple(ctx context.Context, agentID, message string) (*StreamResponse, error) {
	req := &ChatRequest{
		AgentID: agentID,
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: message,
			},
		},
	}
	return s.Stream(ctx, req)
}

// WithSession creates a chat request with a session ID for conversation continuity.
func (s *ChatService) WithSession(agentID, sessionID string) *ChatRequestBuilder {
	return &ChatRequestBuilder{
		service: s,
		request: &ChatRequest{
			AgentID:   agentID,
			SessionID: sessionID,
			Messages:  []ChatMessage{},
		},
	}
}

// ChatRequestBuilder builds a chat request.
type ChatRequestBuilder struct {
	service *ChatService
	request *ChatRequest
}

// AddMessage adds a message to the request.
func (b *ChatRequestBuilder) AddMessage(role, content string) *ChatRequestBuilder {
	b.request.Messages = append(b.request.Messages, ChatMessage{
		Role:    role,
		Content: content,
	})
	return b
}

// AddUserMessage adds a user message to the request.
func (b *ChatRequestBuilder) AddUserMessage(content string) *ChatRequestBuilder {
	return b.AddMessage("user", content)
}

// AddAssistantMessage adds an assistant message to the request.
func (b *ChatRequestBuilder) AddAssistantMessage(content string) *ChatRequestBuilder {
	return b.AddMessage("assistant", content)
}

// AddSystemMessage adds a system message to the request.
func (b *ChatRequestBuilder) AddSystemMessage(content string) *ChatRequestBuilder {
	return b.AddMessage("system", content)
}

// WithOptions sets chat options.
func (b *ChatRequestBuilder) WithOptions(opts *ChatOptions) *ChatRequestBuilder {
	b.request.Options = opts
	return b
}

// WithTemperature sets the temperature option.
func (b *ChatRequestBuilder) WithTemperature(temp float64) *ChatRequestBuilder {
	if b.request.Options == nil {
		b.request.Options = &ChatOptions{}
	}
	b.request.Options.Temperature = &temp
	return b
}

// WithMaxTokens sets the max tokens option.
func (b *ChatRequestBuilder) WithMaxTokens(maxTokens int) *ChatRequestBuilder {
	if b.request.Options == nil {
		b.request.Options = &ChatOptions{}
	}
	b.request.Options.MaxTokens = &maxTokens
	return b
}

// Send sends the chat request.
func (b *ChatRequestBuilder) Send(ctx context.Context) (*ChatResponse, error) {
	return b.service.Send(ctx, b.request)
}

// Stream sends the chat request and returns a streaming response.
func (b *ChatRequestBuilder) Stream(ctx context.Context) (*StreamResponse, error) {
	return b.service.Stream(ctx, b.request)
}
