package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeploymentsServiceList(t *testing.T) {
	expectedDeployments := []Deployment{
		{ID: "deploy-1", AgentID: "agent-1", Status: "running"},
		{ID: "deploy-2", AgentID: "agent-2", Status: "pending"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/deployments" {
			t.Errorf("expected path /api/v1/deployments, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListDeploymentsResponse{
			Deployments: expectedDeployments,
			Total:       2,
			Page:        1,
			PageSize:    10,
			TotalPages:  1,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	response, err := client.Deployments.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Deployments) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(response.Deployments))
	}
	if response.Deployments[0].Status != "running" {
		t.Errorf("expected status 'running', got %s", response.Deployments[0].Status)
	}
}

func TestDeploymentsServiceGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/deployments/deploy-123" {
			t.Errorf("expected path /api/v1/deployments/deploy-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Deployment{
			ID:       "deploy-123",
			AgentID:  "agent-456",
			Status:   "running",
			Replicas: 3,
			Endpoint: "https://agent.example.com",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	deployment, err := client.Deployments.Get(context.Background(), "deploy-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deployment.ID != "deploy-123" {
		t.Errorf("expected ID 'deploy-123', got %s", deployment.ID)
	}
	if deployment.Replicas != 3 {
		t.Errorf("expected 3 replicas, got %d", deployment.Replicas)
	}
}

func TestDeploymentsServiceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req CreateDeploymentRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.AgentID != "agent-123" {
			t.Errorf("expected agent_id 'agent-123', got %s", req.AgentID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Deployment{
			ID:      "new-deploy-id",
			AgentID: req.AgentID,
			Status:  "pending",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	deployment, err := client.Deployments.Create(context.Background(), &CreateDeploymentRequest{
		AgentID:     "agent-123",
		Environment: "production",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deployment.ID != "new-deploy-id" {
		t.Errorf("expected ID 'new-deploy-id', got %s", deployment.ID)
	}
}

func TestDeploymentsServiceScale(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/deployments/deploy-123/scale" {
			t.Errorf("expected path /api/v1/deployments/deploy-123/scale, got %s", r.URL.Path)
		}

		var req map[string]int
		json.NewDecoder(r.Body).Decode(&req)
		if req["replicas"] != 5 {
			t.Errorf("expected replicas 5, got %d", req["replicas"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Deployment{
			ID:       "deploy-123",
			Replicas: 5,
			Status:   "scaling",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	deployment, err := client.Deployments.Scale(context.Background(), "deploy-123", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deployment.Replicas != 5 {
		t.Errorf("expected 5 replicas, got %d", deployment.Replicas)
	}
}

func TestDeploymentsServiceRestart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/deployments/deploy-123/restart" {
			t.Errorf("expected path /api/v1/deployments/deploy-123/restart, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Deployment{
			ID:     "deploy-123",
			Status: "restarting",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	deployment, err := client.Deployments.Restart(context.Background(), "deploy-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deployment.Status != "restarting" {
		t.Errorf("expected status 'restarting', got %s", deployment.Status)
	}
}

func TestDeploymentsServiceRollback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/deployments/deploy-123/rollback" {
			t.Errorf("expected path /api/v1/deployments/deploy-123/rollback, got %s", r.URL.Path)
		}

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		if req["version"] != "v1.0.0" {
			t.Errorf("expected version 'v1.0.0', got %s", req["version"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Deployment{
			ID:      "deploy-123",
			Version: "v1.0.0",
			Status:  "rolling_back",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	deployment, err := client.Deployments.Rollback(context.Background(), "deploy-123", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deployment.Version != "v1.0.0" {
		t.Errorf("expected version 'v1.0.0', got %s", deployment.Version)
	}
}

func TestDeploymentsServiceDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/deployments/deploy-123" {
			t.Errorf("expected path /api/v1/deployments/deploy-123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	err := client.Deployments.Delete(context.Background(), "deploy-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeploymentsServiceStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Deployment{
			ID:     "deploy-123",
			Status: "running",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	status, err := client.Deployments.Status(context.Background(), "deploy-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status != "running" {
		t.Errorf("expected status 'running', got %s", status)
	}
}
