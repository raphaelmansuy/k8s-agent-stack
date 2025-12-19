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
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentsServiceList(t *testing.T) {
	expectedAgents := []Agent{
		{ID: "agent-1", Name: "Test Agent 1", Status: "active"},
		{ID: "agent-2", Name: "Test Agent 2", Status: "inactive"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/agents" {
			t.Errorf("expected path /api/v1/agents, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListAgentsResponse{
			Agents:     expectedAgents,
			Total:      2,
			Page:       1,
			PageSize:   10,
			TotalPages: 1,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	response, err := client.Agents.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(response.Agents))
	}
	if response.Agents[0].ID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got %s", response.Agents[0].ID)
	}
}

func TestAgentsServiceGet(t *testing.T) {
	expectedAgent := Agent{
		ID:     "agent-123",
		Name:   "Test Agent",
		Status: "active",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/agents/agent-123" {
			t.Errorf("expected path /api/v1/agents/agent-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedAgent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	agent, err := client.Agents.Get(context.Background(), "agent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent.ID != "agent-123" {
		t.Errorf("expected agent ID 'agent-123', got %s", agent.ID)
	}
	if agent.Name != "Test Agent" {
		t.Errorf("expected agent name 'Test Agent', got %s", agent.Name)
	}
}

func TestAgentsServiceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req CreateAgentRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name != "New Agent" {
			t.Errorf("expected name 'New Agent', got %s", req.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Agent{
			ID:        "new-agent-id",
			Name:      req.Name,
			ProjectID: req.ProjectID,
			Status:    "pending",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	agent, err := client.Agents.Create(context.Background(), &CreateAgentRequest{
		Name:      "New Agent",
		ProjectID: "project-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent.ID != "new-agent-id" {
		t.Errorf("expected agent ID 'new-agent-id', got %s", agent.ID)
	}
}

func TestAgentsServiceUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/agents/agent-123" {
			t.Errorf("expected path /api/v1/agents/agent-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Agent{
			ID:     "agent-123",
			Name:   "Updated Agent",
			Status: "active",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	name := "Updated Agent"
	agent, err := client.Agents.Update(context.Background(), "agent-123", &UpdateAgentRequest{
		Name: &name,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent.Name != "Updated Agent" {
		t.Errorf("expected agent name 'Updated Agent', got %s", agent.Name)
	}
}

func TestAgentsServiceDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/agents/agent-123" {
			t.Errorf("expected path /api/v1/agents/agent-123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	err := client.Agents.Delete(context.Background(), "agent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentsServiceListWithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		pageSize := r.URL.Query().Get("page_size")

		if page != "2" {
			t.Errorf("expected page '2', got %s", page)
		}
		if pageSize != "10" {
			t.Errorf("expected page_size '10', got %s", pageSize)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListAgentsResponse{
			Agents:     []Agent{},
			Total:      20,
			Page:       2,
			PageSize:   10,
			TotalPages: 2,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")

	_, err := client.Agents.List(context.Background(), &ListOptions{
		Page:     2,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
