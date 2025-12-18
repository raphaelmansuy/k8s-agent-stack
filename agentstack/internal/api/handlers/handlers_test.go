package handlers

import (
	"testing"
)

func TestHealthInput(t *testing.T) {
	input := &HealthInput{}
	if input == nil {
		t.Fatal("HealthInput should not be nil")
	}
}

func TestAgent(t *testing.T) {
	agent := Agent{
		ID:        "test-id",
		ProjectID: "project-1",
		Name:      "Test Agent",
		Slug:      "test-agent",
		ModelID:   "gpt-4",
		Status:    "active",
	}

	if agent.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %s", agent.ID)
	}

	if agent.Status != "active" {
		t.Errorf("expected status 'active', got %s", agent.Status)
	}
}

func TestProject(t *testing.T) {
	project := Project{
		ID:     "test-id",
		TeamID: "team-1",
		Name:   "Test Project",
		Slug:   "test-project",
	}

	if project.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %s", project.ID)
	}

	if project.TeamID != "team-1" {
		t.Errorf("expected TeamID 'team-1', got %s", project.TeamID)
	}
}

func TestChatSession(t *testing.T) {
	session := ChatSession{
		ID:      "session-1",
		AgentID: "agent-1",
		Status:  "active",
	}

	if session.ID != "session-1" {
		t.Errorf("expected ID 'session-1', got %s", session.ID)
	}

	if session.Status != "active" {
		t.Errorf("expected status 'active', got %s", session.Status)
	}
}

func TestChatMessage(t *testing.T) {
	msg := ChatMessage{
		ID:        "msg-1",
		SessionID: "session-1",
		Role:      "user",
		Content:   "Hello, world!",
	}

	if msg.Role != "user" {
		t.Errorf("expected role 'user', got %s", msg.Role)
	}

	if msg.Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got %s", msg.Content)
	}
}

func TestToolCall(t *testing.T) {
	tc := ToolCall{
		ID:   "tool-1",
		Name: "search",
		Args: `{"query": "test"}`,
	}

	if tc.Name != "search" {
		t.Errorf("expected name 'search', got %s", tc.Name)
	}
}
