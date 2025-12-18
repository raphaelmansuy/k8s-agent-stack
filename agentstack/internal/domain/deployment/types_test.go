package deployment

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAgentTypes(t *testing.T) {
	types := []AgentType{
		AgentTypeBYO,
		AgentTypeLLM,
		AgentTypeADK,
	}

	expected := []string{"BYO", "LLM", "ADK"}

	for i, at := range types {
		if string(at) != expected[i] {
			t.Errorf("expected type '%s', got '%s'", expected[i], at)
		}
	}
}

func TestAgentPhases(t *testing.T) {
	phases := []AgentPhase{
		AgentPhasePending,
		AgentPhaseCreating,
		AgentPhaseReady,
		AgentPhaseFailed,
		AgentPhaseDeleting,
	}

	expected := []string{"Pending", "Creating", "Ready", "Failed", "Deleting"}

	for i, phase := range phases {
		if string(phase) != expected[i] {
			t.Errorf("expected phase '%s', got '%s'", expected[i], phase)
		}
	}
}

func TestAgentSerialization(t *testing.T) {
	agent := &Agent{
		ID:          "agent-123",
		Name:        "test-agent",
		Description: "A test agent",
		Image:       "ghcr.io/test/agent:v1.0",
		Namespace:   "default",
		Type:        AgentTypeBYO,
		Env: map[string]string{
			"API_KEY": "secret",
		},
		Resources: &ResourceSpec{
			MemoryRequest: "256Mi",
			MemoryLimit:   "512Mi",
			CPURequest:    "100m",
			CPULimit:      "500m",
		},
		Replicas: 2,
		Model: &ModelConfig{
			Provider:    "openai",
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   1000,
		},
		Tools: []ToolSpec{
			{
				Name:        "get_weather",
				Description: "Get weather info",
				Type:        "function",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status: AgentStatus{
			Phase: AgentPhaseReady,
			URL:   "https://agent.example.com",
		},
	}

	// Serialize
	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal agent: %v", err)
	}

	// Deserialize
	var parsed Agent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal agent: %v", err)
	}

	// Verify fields
	if parsed.ID != "agent-123" {
		t.Errorf("expected ID 'agent-123', got '%s'", parsed.ID)
	}
	if parsed.Name != "test-agent" {
		t.Errorf("expected Name 'test-agent', got '%s'", parsed.Name)
	}
	if parsed.Type != AgentTypeBYO {
		t.Errorf("expected Type 'BYO', got '%s'", parsed.Type)
	}
	if parsed.Resources.MemoryLimit != "512Mi" {
		t.Errorf("expected MemoryLimit '512Mi', got '%s'", parsed.Resources.MemoryLimit)
	}
	if parsed.Model.Provider != "openai" {
		t.Errorf("expected Provider 'openai', got '%s'", parsed.Model.Provider)
	}
	if len(parsed.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(parsed.Tools))
	}
	if parsed.Status.Phase != AgentPhaseReady {
		t.Errorf("expected Phase 'Ready', got '%s'", parsed.Status.Phase)
	}
}

func TestResourceSpec(t *testing.T) {
	spec := &ResourceSpec{
		MemoryRequest: "128Mi",
		MemoryLimit:   "256Mi",
		CPURequest:    "50m",
		CPULimit:      "200m",
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ResourceSpec
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.MemoryRequest != "128Mi" {
		t.Errorf("expected MemoryRequest '128Mi', got '%s'", parsed.MemoryRequest)
	}
}

func TestModelConfig(t *testing.T) {
	config := &ModelConfig{
		Provider:    "anthropic",
		Model:       "claude-3-sonnet",
		APIKeyRef:   "secret/anthropic-key",
		Temperature: 0.5,
		MaxTokens:   2000,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ModelConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Provider != "anthropic" {
		t.Errorf("expected Provider 'anthropic', got '%s'", parsed.Provider)
	}
	if parsed.Temperature != 0.5 {
		t.Errorf("expected Temperature 0.5, got %f", parsed.Temperature)
	}
}

func TestToolSpec(t *testing.T) {
	tool := &ToolSpec{
		Name:        "search_web",
		Description: "Search the web",
		Type:        "mcp",
		Config: map[string]interface{}{
			"endpoint": "http://mcp-server:8080",
			"timeout":  30,
		},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ToolSpec
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Name != "search_web" {
		t.Errorf("expected Name 'search_web', got '%s'", parsed.Name)
	}
	if parsed.Type != "mcp" {
		t.Errorf("expected Type 'mcp', got '%s'", parsed.Type)
	}
	if parsed.Config["endpoint"] != "http://mcp-server:8080" {
		t.Errorf("expected endpoint, got %v", parsed.Config["endpoint"])
	}
}

func TestAgentStatus(t *testing.T) {
	status := AgentStatus{
		Phase:      AgentPhaseReady,
		URL:        "https://agent.example.com",
		Conditions: []string{"Ready", "Available"},
		Message:    "Agent is healthy",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed AgentStatus
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Phase != AgentPhaseReady {
		t.Errorf("expected Phase 'Ready', got '%s'", parsed.Phase)
	}
	if parsed.URL != "https://agent.example.com" {
		t.Errorf("expected URL, got '%s'", parsed.URL)
	}
	if len(parsed.Conditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(parsed.Conditions))
	}
}

func TestMinimalAgent(t *testing.T) {
	// Test with minimal required fields
	agent := &Agent{
		ID:    "minimal-agent",
		Image: "nginx:latest",
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed Agent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.ID != "minimal-agent" {
		t.Errorf("expected ID 'minimal-agent', got '%s'", parsed.ID)
	}
	if parsed.Image != "nginx:latest" {
		t.Errorf("expected Image 'nginx:latest', got '%s'", parsed.Image)
	}
}

func TestAgentDefaultValues(t *testing.T) {
	agent := &Agent{
		ID:    "test",
		Image: "test:latest",
	}

	// Check default values
	if agent.Namespace != "" {
		t.Errorf("expected empty namespace by default")
	}
	if agent.Type != "" {
		t.Errorf("expected empty type by default")
	}
	if agent.Replicas != 0 {
		t.Errorf("expected 0 replicas by default")
	}
}

func TestAgentEnvMap(t *testing.T) {
	agent := &Agent{
		ID:    "env-test",
		Image: "test:latest",
		Env: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
		},
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed Agent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.Env) != 3 {
		t.Errorf("expected 3 env vars, got %d", len(parsed.Env))
	}
	if parsed.Env["VAR1"] != "value1" {
		t.Errorf("expected VAR1='value1', got '%s'", parsed.Env["VAR1"])
	}
}

func TestMultipleTools(t *testing.T) {
	agent := &Agent{
		ID:    "multi-tool",
		Image: "test:latest",
		Tools: []ToolSpec{
			{Name: "tool1", Type: "function"},
			{Name: "tool2", Type: "mcp"},
			{Name: "tool3", Type: "a2a"},
		},
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed Agent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(parsed.Tools))
	}

	toolTypes := map[string]bool{}
	for _, tool := range parsed.Tools {
		toolTypes[tool.Type] = true
	}

	if !toolTypes["function"] || !toolTypes["mcp"] || !toolTypes["a2a"] {
		t.Error("expected all tool types to be present")
	}
}
