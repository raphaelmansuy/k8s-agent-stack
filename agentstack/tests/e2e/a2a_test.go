// Package e2e provides end-to-end tests for the AgentStack platform.
// These tests verify the complete flow from API to agent communication.
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

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
	"github.com/raphaelmansuy/agentstack/internal/testing/kagent"
)

// TestA2AMessageSend tests synchronous message sending via A2A.
func TestA2AMessageSend(t *testing.T) {
	ctx := context.Background()

	// Start mock kagent
	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	// Wait for ready
	if err := k.WaitForReady(ctx, 5*time.Second); err != nil {
		t.Fatalf("kagent not ready: %v", err)
	}

	// Create A2A service
	a2aService := a2a.NewService()

	// Send message
	params := &a2a.SendMessageParams{
		Message: a2a.MessageInput{
			Parts: []a2a.Part{a2a.TextPart("Hello, kagent!")},
		},
	}

	task, err := a2aService.SendMessage(ctx, k.URL, params)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	if task.TaskID == "" {
		t.Error("expected taskID to be set")
	}

	if task.Status == nil {
		t.Error("expected status to be set")
	}

	// Verify task was recorded in kagent
	if k.TaskCount() < 1 {
		t.Error("expected kagent to record at least one task")
	}
}

// TestA2AMessageStream tests streaming message via A2A.
func TestA2AMessageStream(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start mock kagent
	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	// Wait for ready
	if err := k.WaitForReady(ctx, 5*time.Second); err != nil {
		t.Fatalf("kagent not ready: %v", err)
	}

	// Create A2A service
	a2aService := a2a.NewService()

	// Track received events
	var events []interface{}

	params := &a2a.SendMessageParams{
		Message: a2a.MessageInput{
			Parts: []a2a.Part{a2a.TextPart("Stream test message")},
		},
	}

	handler := func(event interface{}) error {
		events = append(events, event)
		return nil
	}

	err := a2aService.StreamMessage(ctx, k.URL, params, handler)
	if err != nil {
		t.Fatalf("failed to stream message: %v", err)
	}

	if len(events) == 0 {
		t.Error("expected to receive streaming events")
	}
}

// TestA2ASessionManagement tests session creation and retrieval.
func TestA2ASessionManagement(t *testing.T) {
	a2aService := a2a.NewService()

	// Create session
	session, err := a2aService.NewSession("http://test-agent.example.com")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session.ContextID == "" {
		t.Error("expected contextID to be set")
	}

	// Retrieve session
	retrieved, ok := a2aService.GetSession(session.ContextID)
	if !ok {
		t.Error("expected to retrieve session")
	}

	if retrieved.ContextID != session.ContextID {
		t.Error("retrieved session has different contextID")
	}

	// Delete session
	a2aService.DeleteSession(session.ContextID)

	_, ok = a2aService.GetSession(session.ContextID)
	if ok {
		t.Error("expected session to be deleted")
	}
}

// TestA2AErrorHandling tests error responses from agents.
func TestA2AErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Start mock kagent with error mode
	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	k.SetError(true)

	a2aService := a2a.NewService()

	params := &a2a.SendMessageParams{
		Message: a2a.MessageInput{
			Parts: []a2a.Part{a2a.TextPart("This should fail")},
		},
	}

	_, err := a2aService.SendMessage(ctx, k.URL, params)
	if err == nil {
		t.Error("expected error from error-mode kagent")
	}
}

// TestA2AAgentCard tests agent card discovery.
func TestA2AAgentCard(t *testing.T) {
	// Start mock kagent
	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	// Fetch agent card
	resp, err := httpGetWithContext(k.URL + "/.well-known/agent.json")
	if err != nil {
		t.Fatalf("failed to fetch agent card: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var card a2a.AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatalf("failed to decode agent card: %v", err)
	}

	if card.Name != "Mock Kagent" {
		t.Errorf("expected name 'Mock Kagent', got %q", card.Name)
	}

	if card.ProtocolVersion != "1.0" {
		t.Errorf("expected protocol version '1.0', got %q", card.ProtocolVersion)
	}

	if !card.Capabilities.Streaming {
		t.Error("expected streaming capability to be true")
	}

	if len(card.Skills) == 0 {
		t.Error("expected at least one skill")
	}
}

// TestA2AHealthEndpoints tests kagent health endpoints.
func TestA2AHealthEndpoints(t *testing.T) {
	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	tests := []struct {
		name     string
		endpoint string
		wantKey  string
	}{
		{"health", "/health", "status"},
		{"ready", "/ready", "status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := httpGetWithContext(k.URL + tt.endpoint)
			if err != nil {
				t.Fatalf("failed to fetch %s: %v", tt.endpoint, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}

			var result map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if _, ok := result[tt.wantKey]; !ok {
				t.Errorf("expected %q key in response", tt.wantKey)
			}
		})
	}
}

// TestA2AConversationFlow tests a complete conversation flow.
func TestA2AConversationFlow(t *testing.T) {
	ctx := context.Background()

	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	if err := k.WaitForReady(ctx, 5*time.Second); err != nil {
		t.Fatalf("kagent not ready: %v", err)
	}

	a2aService := a2a.NewService()

	// First message
	params1 := &a2a.SendMessageParams{
		Message: a2a.MessageInput{
			Parts: []a2a.Part{a2a.TextPart("Hello!")},
		},
	}

	task1, err := a2aService.SendMessage(ctx, k.URL, params1)
	if err != nil {
		t.Fatalf("first message failed: %v", err)
	}

	if task1.TaskID == "" {
		t.Error("expected taskID in first response")
	}

	if task1.ContextID == "" {
		t.Error("expected contextID in first response")
	}

	// Second message - verify we can continue conversation
	params2 := &a2a.SendMessageParams{
		Message: a2a.MessageInput{
			ContextID: task1.ContextID,
			Parts:     []a2a.Part{a2a.TextPart("How are you?")},
		},
	}

	task2, err := a2aService.SendMessage(ctx, k.URL, params2)
	if err != nil {
		t.Fatalf("second message failed: %v", err)
	}

	// Verify both tasks have valid IDs
	if task2.TaskID == "" {
		t.Error("expected taskID in second response")
	}

	// Verify kagent received both messages
	if k.TaskCount() < 2 {
		t.Errorf("expected at least 2 tasks, got %d", k.TaskCount())
	}
}

// TestA2AEchoSkill tests the echo skill.
func TestA2AEchoSkill(t *testing.T) {
	ctx := context.Background()

	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	a2aService := a2a.NewService()

	// Note: The mock kagent lowercases input, so we use lowercase for comparison
	testMessage := "this is a test echo message"
	params := &a2a.SendMessageParams{
		Message: a2a.MessageInput{
			Parts: []a2a.Part{a2a.TextPart("echo " + testMessage)},
		},
	}

	task, err := a2aService.SendMessage(ctx, k.URL, params)
	if err != nil {
		t.Fatalf("echo failed: %v", err)
	}

	if task.Status == nil || task.Status.Message == nil {
		t.Fatal("expected status with message")
	}

	// Check response contains echoed text (mock returns lowercased text)
	for _, part := range task.Status.Message.Parts {
		if part.Kind == "text" && strings.Contains(strings.ToLower(part.Text), testMessage) {
			return // Test passed
		}
	}

	// Log actual response for debugging
	t.Logf("actual response parts: %+v", task.Status.Message.Parts)
	t.Error("expected echoed message in response")
}

// TestA2AConcurrentMessages tests concurrent message sending.
func TestA2AConcurrentMessages(t *testing.T) {
	ctx := context.Background()

	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	if err := k.WaitForReady(ctx, 5*time.Second); err != nil {
		t.Fatalf("kagent not ready: %v", err)
	}

	a2aService := a2a.NewService()

	// Send multiple messages concurrently
	numMessages := 10
	results := make(chan error, numMessages)

	for i := range numMessages {
		go func(idx int) {
			params := &a2a.SendMessageParams{
				Message: a2a.MessageInput{
					Parts: []a2a.Part{a2a.TextPart(fmt.Sprintf("Concurrent message %d", idx))},
				},
			}

			_, err := a2aService.SendMessage(ctx, k.URL, params)
			results <- err
		}(i)
	}

	// Collect results
	var errors []error
	for range numMessages {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Errorf("got %d errors in concurrent messages: %v", len(errors), errors)
	}

	// Verify all tasks were created
	if k.TaskCount() < numMessages {
		t.Errorf("expected %d tasks, got %d", numMessages, k.TaskCount())
	}
}

// TestA2ATimeout tests request timeout handling.
func TestA2ATimeout(t *testing.T) {
	// Create slow server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	// Use short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	a2aService := a2a.NewService()

	params := &a2a.SendMessageParams{
		Message: a2a.MessageInput{
			Parts: []a2a.Part{a2a.TextPart("Should timeout")},
		},
	}

	_, err := a2aService.SendMessage(ctx, slowServer.URL, params)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func httpGetWithContext(url string) (*http.Response, error) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	return http.DefaultClient.Do(req)
}
