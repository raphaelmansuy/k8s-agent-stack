/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatServiceSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/chat" {
			t.Errorf("expected path /api/v1/chat, got %s", r.URL.Path)
		}

		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.AgentID != "agent-123" {
			t.Errorf("expected agent_id 'agent-123', got %s", req.AgentID)
		}
		if len(req.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(req.Messages))
		}
		if req.Stream {
			t.Error("expected stream to be false")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			ID:        "response-123",
			SessionID: "session-456",
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Hello! How can I help you?",
			},
			Usage: &Usage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	response, err := client.Chat.Send(context.Background(), &ChatRequest{
		AgentID: "agent-123",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.ID != "response-123" {
		t.Errorf("expected ID 'response-123', got %s", response.ID)
	}
	if response.Message.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %s", response.Message.Role)
	}
	if response.Usage.TotalTokens != 18 {
		t.Errorf("expected 18 total tokens, got %d", response.Usage.TotalTokens)
	}
}

func TestChatServiceSendSimple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.AgentID != "agent-123" {
			t.Errorf("expected agent_id 'agent-123', got %s", req.AgentID)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "Hello world" {
			t.Error("expected single user message with 'Hello world'")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			ID: "response-123",
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Hi there!",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	response, err := client.Chat.SendSimple(context.Background(), "agent-123", "Hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Message.Content != "Hi there!" {
		t.Errorf("expected content 'Hi there!', got %s", response.Message.Content)
	}
}

func TestChatServiceStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Error("expected Accept: text/event-stream header")
		}

		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		if !req.Stream {
			t.Error("expected stream to be true")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected response writer to support flushing")
		}

		// Send streaming events
		events := []StreamDelta{
			{ID: "1", Content: "Hello", Done: false},
			{ID: "1", Content: " World", Done: false},
			{ID: "1", Content: "!", Done: true},
		}

		for _, event := range events {
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	streamResp, err := client.Chat.Stream(context.Background(), &ChatRequest{
		AgentID: "agent-123",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var contents []string //nolint:prealloc
	for event := range streamResp.Events {
		contents = append(contents, event.Content)
		if event.Done {
			break
		}
	}

	if len(contents) < 1 {
		t.Error("expected to receive at least one event")
	}
}

func TestChatRequestBuilder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.AgentID != "agent-123" {
			t.Errorf("expected agent_id 'agent-123', got %s", req.AgentID)
		}
		if req.SessionID != "session-456" {
			t.Errorf("expected session_id 'session-456', got %s", req.SessionID)
		}
		if len(req.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Errorf("expected first message role 'system', got %s", req.Messages[0].Role)
		}
		if req.Options == nil || *req.Options.Temperature != 0.7 {
			t.Error("expected temperature 0.7")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			ID: "response-123",
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Response",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	_, err := client.Chat.
		WithSession("agent-123", "session-456").
		AddSystemMessage("You are a helpful assistant").
		AddUserMessage("Hello").
		WithTemperature(0.7).
		Send(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChatServiceStreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: ErrorDetail{
				Code:    "unauthorized",
				Message: "Invalid API key",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "invalid-key")

	_, err := client.Chat.Stream(context.Background(), &ChatRequest{
		AgentID: "agent-123",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr := &APIError{}
	ok := errors.As(err, &apiErr)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
}
