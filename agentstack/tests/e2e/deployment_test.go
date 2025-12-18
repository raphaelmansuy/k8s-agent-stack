// Package e2e provides end-to-end tests for the AgentStack platform.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
	"github.com/raphaelmansuy/agentstack/internal/testing/kagent"
)

type SendMessageResponse struct {
	TaskID string `json:"taskId"`
}

func registerA2AMock(mux *http.ServeMux, service *a2a.Service) {
	mux.HandleFunc("/a2a/send", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			AgentURL  string `json:"agentUrl"`
			Content   string `json:"content"`
			ContextID string `json:"contextId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if body.AgentURL == "" || body.Content == "" {
			http.Error(w, "missing required fields", http.StatusBadRequest)
			return
		}

		params := &a2a.SendMessageParams{
			Message: a2a.MessageInput{
				Parts:     []a2a.Part{a2a.TextPart(body.Content)},
				ContextID: body.ContextID,
			},
		}

		task, err := service.SendMessage(r.Context(), body.AgentURL, params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"taskId":    task.TaskID,
			"contextId": task.ContextID,
		})
	})
}

// TestE2EAgentDeploymentFlow tests the complete agent deployment flow.
func TestE2EAgentDeploymentFlow(t *testing.T) {
	ctx := context.Background()

	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	if err := k.WaitForReady(ctx, 5*time.Second); err != nil {
		t.Fatalf("kagent not ready: %v", err)
	}

	a2aService := a2a.NewService()
	

	mux := http.NewServeMux()
	registerA2AMock(mux, a2aService)

	server := httptest.NewServer(mux)
	defer server.Close()

	reqBody := map[string]interface{}{
		"agentUrl": k.URL,
		"content":  "Hello from E2E test!",
	}
	reqJSON, _ := json.Marshal(reqBody)

	resp, err := http.Post(server.URL+"/a2a/send", "application/json", bytes.NewReader(reqJSON))
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var sendResp SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&sendResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if sendResp.TaskID == "" {
		t.Error("expected taskID in response")
	}

	if k.TaskCount() < 1 {
		t.Error("expected kagent to record message")
	}
}

// TestE2EAPIChain tests the full API chain.
func TestE2EAPIChain(t *testing.T) {
	ctx := context.Background()

	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	if err := k.WaitForReady(ctx, 5*time.Second); err != nil {
		t.Fatalf("kagent not ready: %v", err)
	}

	a2aService := a2a.NewService()
	

	mux := http.NewServeMux()
	registerA2AMock(mux, a2aService)

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name    string
		content string
	}{
		{"greeting", "Hello there!"},
		{"echo", "echo this is echoed"},
		{"question", "What can you do?"},
		{"help", "I need help with something"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"agentUrl": k.URL,
				"content":  tt.content,
			}
			reqJSON, _ := json.Marshal(reqBody)

			resp, err := http.Post(server.URL+"/a2a/send", "application/json", bytes.NewReader(reqJSON))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			var result SendMessageResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode: %v", err)
			}

			if result.TaskID == "" {
				t.Error("expected taskID")
			}
		})
	}
}

// TestE2EValidation tests input validation across the API.
func TestE2EValidation(t *testing.T) {

	a2aService := a2a.NewService()
	

	mux := http.NewServeMux()
	registerA2AMock(mux, a2aService)

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "missing agentUrl",
			body:       map[string]interface{}{"content": "test"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing content",
			body:       map[string]interface{}{"agentUrl": "http://example.com"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqJSON, _ := json.Marshal(tt.body)
			resp, err := http.Post(server.URL+"/a2a/send", "application/json", bytes.NewReader(reqJSON))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("expected %d, got %d", tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

// TestE2EAgentDiscovery tests agent card discovery flow.
func TestE2EAgentDiscovery(t *testing.T) {
	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	resp, err := http.Get(k.URL + "/.well-known/agent.json")
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var card a2a.AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatalf("failed to decode card: %v", err)
	}

	if card.ProtocolVersion == "" {
		t.Error("expected protocolVersion")
	}

	if card.Name == "" {
		t.Error("expected name")
	}

	if len(card.SupportedInterfaces) == 0 {
		t.Error("expected at least one supported interface")
	}
}

// TestE2EErrorRecovery tests error handling and recovery.
func TestE2EErrorRecovery(t *testing.T) {
	ctx := context.Background()

	k := kagent.New(kagent.DefaultConfig())
	defer k.Close()

	a2aService := a2a.NewService()
	

	mux := http.NewServeMux()
	registerA2AMock(mux, a2aService)

	server := httptest.NewServer(mux)
	defer server.Close()

	k.SetError(true)

	req1 := map[string]interface{}{
		"agentUrl": k.URL,
		"content":  "Should fail",
	}
	reqJSON1, _ := json.Marshal(req1)

	resp1, _ := http.Post(server.URL+"/a2a/send", "application/json", bytes.NewReader(reqJSON1))
	resp1.Body.Close()

	if resp1.StatusCode == http.StatusOK {
		t.Error("expected error when error mode enabled")
	}

	k.SetError(false)

	if err := k.WaitForReady(ctx, 5*time.Second); err != nil {
		t.Fatalf("kagent not ready after recovery: %v", err)
	}

	req2 := map[string]interface{}{
		"agentUrl": k.URL,
		"content":  "Should succeed",
	}
	reqJSON2, _ := json.Marshal(req2)

	resp2, err := http.Post(server.URL+"/a2a/send", "application/json", bytes.NewReader(reqJSON2))
	if err != nil {
		t.Fatalf("recovery request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after recovery, got %d", resp2.StatusCode)
	}
}
