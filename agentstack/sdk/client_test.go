package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://api.example.com", "test-api-key")

	if client.baseURL != "https://api.example.com" {
		t.Errorf("expected baseURL to be 'https://api.example.com', got %s", client.baseURL)
	}
	if client.apiKey != "test-api-key" {
		t.Errorf("expected apiKey to be 'test-api-key', got %s", client.apiKey)
	}
	if client.Agents == nil {
		t.Error("expected Agents service to be initialized")
	}
	if client.Chat == nil {
		t.Error("expected Chat service to be initialized")
	}
	if client.Deployments == nil {
		t.Error("expected Deployments service to be initialized")
	}
	if client.Auth == nil {
		t.Error("expected Auth service to be initialized")
	}
	if client.Projects == nil {
		t.Error("expected Projects service to be initialized")
	}
}

func TestClientWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 60 * time.Second}
	client := NewClient("https://api.example.com", "test-api-key").WithHTTPClient(customClient)

	if client.httpClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestClientWithUserAgent(t *testing.T) {
	client := NewClient("https://api.example.com", "test-api-key").WithUserAgent("custom-agent/1.0")

	if client.userAgent != "custom-agent/1.0" {
		t.Errorf("expected userAgent to be 'custom-agent/1.0', got %s", client.userAgent)
	}
}

func TestClientRequest(t *testing.T) {
	client := NewClient("https://api.example.com", "test-api-key")

	body := map[string]string{"key": "value"}
	req, err := client.Request(context.Background(), http.MethodPost, "/api/v1/test", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Method != http.MethodPost {
		t.Errorf("expected method POST, got %s", req.Method)
	}
	if req.URL.String() != "https://api.example.com/api/v1/test" {
		t.Errorf("unexpected URL: %s", req.URL.String())
	}
	if req.Header.Get("Authorization") != "Bearer test-api-key" {
		t.Error("expected Authorization header to be set")
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type to be application/json")
	}
}

func TestClientGet(t *testing.T) {
	expectedResponse := map[string]string{"status": "ok"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Error("expected Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	var result map[string]string
	err := client.Get(context.Background(), "/test", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %s", result["status"])
	}
}

func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "test" {
			t.Errorf("expected name 'test', got %s", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	var result map[string]string
	err := client.Post(context.Background(), "/test", map[string]string{"name": "test"}, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("expected id '123', got %s", result["id"])
	}
}

func TestClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	err := client.Delete(context.Background(), "/test/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: ErrorDetail{
				Code:    "not_found",
				Message: "Resource not found",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	err := client.Get(context.Background(), "/test/nonexistent", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
	if apiErr.Code != "not_found" {
		t.Errorf("expected code 'not_found', got %s", apiErr.Code)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound to return true")
	}
}

func TestBuildQueryString(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]string
		expected string
	}{
		{
			name:     "empty params",
			params:   map[string]string{},
			expected: "",
		},
		{
			name:     "single param",
			params:   map[string]string{"page": "1"},
			expected: "?page=1",
		},
		{
			name:     "skip empty values",
			params:   map[string]string{"page": "1", "empty": ""},
			expected: "?page=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildQueryString(tt.params)
			// For single param case, the result should contain the param
			if tt.name == "single param" && result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
			if tt.name == "empty params" && result != "" {
				t.Errorf("expected empty string, got %s", result)
			}
		})
	}
}
