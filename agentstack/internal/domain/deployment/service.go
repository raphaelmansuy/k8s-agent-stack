// Package deployment provides agent deployment services.
package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/infrastructure/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// kagent CRD GVR (Group-Version-Resource)
var kagentGVR = schema.GroupVersionResource{
	Group:    "kagent.dev",
	Version:  "v1alpha2",
	Resource: "agents",
}

// Agent represents an agent definition for deployment.
type Agent struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Image       string            `json:"image"`
	Namespace   string            `json:"namespace"`
	Type        AgentType         `json:"type"`
	Env         map[string]string `json:"env,omitempty"`
	Resources   *ResourceSpec     `json:"resources,omitempty"`
	Replicas    int32             `json:"replicas,omitempty"`
	Model       *ModelConfig      `json:"model,omitempty"`
	Tools       []ToolSpec        `json:"tools,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	Status      AgentStatus       `json:"status"`
}

// AgentType defines the type of agent deployment.
type AgentType string

const (
	// AgentTypeBYO is a Bring Your Own agent (custom container).
	AgentTypeBYO AgentType = "BYO"
	// AgentTypeLLM is an LLM-powered agent with configured model.
	AgentTypeLLM AgentType = "LLM"
	// AgentTypeADK is a Google ADK-based agent.
	AgentTypeADK AgentType = "ADK"
)

// AgentStatus represents the current status of an agent.
type AgentStatus struct {
	Phase      AgentPhase `json:"phase"`
	URL        string     `json:"url,omitempty"`
	Conditions []string   `json:"conditions,omitempty"`
	Message    string     `json:"message,omitempty"`
}

// AgentPhase represents the lifecycle phase of an agent.
type AgentPhase string

const (
	AgentPhasePending  AgentPhase = "Pending"
	AgentPhaseCreating AgentPhase = "Creating"
	AgentPhaseReady    AgentPhase = "Ready"
	AgentPhaseFailed   AgentPhase = "Failed"
	AgentPhaseDeleting AgentPhase = "Deleting"
)

// ResourceSpec defines resource requirements for an agent.
type ResourceSpec struct {
	MemoryRequest string `json:"memoryRequest,omitempty"`
	MemoryLimit   string `json:"memoryLimit,omitempty"`
	CPURequest    string `json:"cpuRequest,omitempty"`
	CPULimit      string `json:"cpuLimit,omitempty"`
}

// ModelConfig defines the LLM configuration for an agent.
type ModelConfig struct {
	Provider    string  `json:"provider"` // openai, gemini, anthropic, litellm
	Model       string  `json:"model"`
	APIKeyRef   string  `json:"apiKeyRef,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"maxTokens,omitempty"`
}

// ToolSpec defines a tool/skill available to an agent.
type ToolSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type"` // function, mcp, a2a
	Config      map[string]interface{} `json:"config,omitempty"`
}

// Service provides agent deployment operations.
type Service struct {
	k8sClients map[string]*k8s.Client // namespace -> client
	dynClient  dynamic.Interface
	agents     map[string]*Agent
	mu         sync.RWMutex
	logger     *slog.Logger
}

// NewService creates a new deployment service.
func NewService(logger *slog.Logger) *Service {
	return &Service{
		k8sClients: make(map[string]*k8s.Client),
		agents:     make(map[string]*Agent),
		logger:     logger,
	}
}

// InitK8sClient initializes a k8s client for a namespace.
func (s *Service) InitK8sClient(kubeconfig, namespace string) error {
	client, err := k8s.NewClient(kubeconfig, namespace)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}
	s.k8sClients[namespace] = client
	return nil
}

// getK8sClient gets or creates a k8s client for the given namespace.
func (s *Service) getK8sClient(namespace string) (*k8s.Client, error) {
	if client, ok := s.k8sClients[namespace]; ok {
		return client, nil
	}

	// Try to create a new client for this namespace
	client, err := k8s.NewClient("", namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client for namespace %s: %w", namespace, err)
	}
	s.k8sClients[namespace] = client
	return client, nil
}

// CreateAgent creates a new agent deployment.
func (s *Service) CreateAgent(ctx context.Context, agent *Agent) (*Agent, error) {
	if agent.ID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	if agent.Name == "" {
		agent.Name = agent.ID
	}

	if agent.Namespace == "" {
		agent.Namespace = "default"
	}

	if agent.Type == "" {
		agent.Type = AgentTypeBYO
	}

	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()
	agent.Status = AgentStatus{Phase: AgentPhasePending}

	// Get k8s client
	k8sClient, err := s.getK8sClient(agent.Namespace)
	if err != nil {
		return nil, err
	}

	// Check if kagent CRD exists
	useKagent := s.kagentCRDExists(ctx, agent.Namespace)

	if useKagent {
		// Deploy using kagent CRD
		if err := s.deployKagentAgent(ctx, agent); err != nil {
			return nil, fmt.Errorf("failed to deploy kagent agent: %w", err)
		}
	} else {
		// Deploy using Knative Service
		if err := s.deployKnativeAgent(ctx, k8sClient, agent); err != nil {
			return nil, fmt.Errorf("failed to deploy knative agent: %w", err)
		}
	}

	s.mu.Lock()
	s.agents[agent.ID] = agent
	s.mu.Unlock()

	return agent, nil
}

// kagentCRDExists checks if kagent CRD is available in the cluster.
func (s *Service) kagentCRDExists(ctx context.Context, namespace string) bool {
	if s.dynClient == nil {
		return false
	}
	_, err := s.dynClient.Resource(kagentGVR).Namespace(namespace).List(ctx, metav1.ListOptions{Limit: 1})
	return err == nil
}

// deployKagentAgent deploys an agent using kagent CRD.
func (s *Service) deployKagentAgent(ctx context.Context, agent *Agent) error {
	if s.dynClient == nil {
		return fmt.Errorf("dynamic client not initialized")
	}

	// Build kagent Agent CRD spec
	spec := map[string]interface{}{
		"type":        string(agent.Type),
		"description": agent.Description,
	}

	// BYO agent configuration
	if agent.Type == AgentTypeBYO {
		deployment := map[string]interface{}{
			"image": agent.Image,
		}

		if agent.Resources != nil {
			resources := map[string]interface{}{}
			if agent.Resources.MemoryRequest != "" || agent.Resources.MemoryLimit != "" {
				memory := map[string]interface{}{}
				if agent.Resources.MemoryRequest != "" {
					memory["request"] = agent.Resources.MemoryRequest
				}
				if agent.Resources.MemoryLimit != "" {
					memory["limit"] = agent.Resources.MemoryLimit
				}
				resources["memory"] = memory
			}
			if agent.Resources.CPURequest != "" || agent.Resources.CPULimit != "" {
				cpu := map[string]interface{}{}
				if agent.Resources.CPURequest != "" {
					cpu["request"] = agent.Resources.CPURequest
				}
				if agent.Resources.CPULimit != "" {
					cpu["limit"] = agent.Resources.CPULimit
				}
				resources["cpu"] = cpu
			}
			if len(resources) > 0 {
				deployment["resources"] = resources
			}
		}

		if len(agent.Env) > 0 {
			env := make([]map[string]interface{}, 0, len(agent.Env))
			for k, v := range agent.Env {
				env = append(env, map[string]interface{}{
					"name":  k,
					"value": v,
				})
			}
			deployment["env"] = env
		}

		spec["byo"] = map[string]interface{}{
			"deployment": deployment,
		}
	}

	// Model configuration for LLM agents
	if agent.Model != nil && agent.Type == AgentTypeLLM {
		modelSpec := map[string]interface{}{
			"provider": agent.Model.Provider,
			"model":    agent.Model.Model,
		}
		if agent.Model.APIKeyRef != "" {
			modelSpec["apiKeyRef"] = agent.Model.APIKeyRef
		}
		spec["model"] = modelSpec
	}

	// Tools configuration
	if len(agent.Tools) > 0 {
		tools := make([]map[string]interface{}, 0, len(agent.Tools))
		for _, t := range agent.Tools {
			tool := map[string]interface{}{
				"name": t.Name,
				"type": t.Type,
			}
			if t.Description != "" {
				tool["description"] = t.Description
			}
			if len(t.Config) > 0 {
				tool["config"] = t.Config
			}
			tools = append(tools, tool)
		}
		spec["tools"] = tools
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kagent.dev/v1alpha2",
			"kind":       "Agent",
			"metadata": map[string]interface{}{
				"name":      agent.Name,
				"namespace": agent.Namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "agentstack",
					"agentstack.io/agent-id":       agent.ID,
				},
			},
			"spec": spec,
		},
	}

	_, err := s.dynClient.Resource(kagentGVR).Namespace(agent.Namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create kagent agent: %w", err)
	}

	agent.Status.Phase = AgentPhaseCreating

	return nil
}

// deployKnativeAgent deploys an agent using Knative Service.
func (s *Service) deployKnativeAgent(ctx context.Context, k8sClient *k8s.Client, agent *Agent) error {
	spec := &k8s.KnativeServiceSpec{
		Name:    agent.Name,
		AgentID: agent.ID,
		Image:   agent.Image,
		Env:     agent.Env,
	}

	if agent.Resources != nil {
		spec.Resources = k8s.ResourceSpec{
			MemoryRequest: agent.Resources.MemoryRequest,
			MemoryLimit:   agent.Resources.MemoryLimit,
			CPURequest:    agent.Resources.CPURequest,
			CPULimit:      agent.Resources.CPULimit,
		}
	}

	if err := k8sClient.CreateKnativeService(ctx, spec); err != nil {
		return fmt.Errorf("failed to create knative service: %w", err)
	}

	agent.Status.Phase = AgentPhaseCreating

	return nil
}

// GetAgent retrieves an agent by ID.
func (s *Service) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	s.mu.RLock()
	agent, ok := s.agents[agentID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// Refresh status from cluster
	if err := s.refreshAgentStatus(ctx, agent); err != nil {
		s.logger.Warn("failed to refresh agent status", "agentID", agentID, "error", err)
	}

	return agent, nil
}

// refreshAgentStatus fetches current status from the cluster.
func (s *Service) refreshAgentStatus(ctx context.Context, agent *Agent) error {
	// Try kagent first
	if s.dynClient != nil {
		obj, err := s.dynClient.Resource(kagentGVR).Namespace(agent.Namespace).Get(ctx, agent.Name, metav1.GetOptions{})
		if err == nil {
			return s.parseKagentStatus(obj, agent)
		}
	}

	// Fall back to Knative
	k8sClient, err := s.getK8sClient(agent.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	status, err := k8sClient.GetKnativeServiceStatus(ctx, agent.Name)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	agent.Status.URL = status.URL
	if status.Ready {
		agent.Status.Phase = AgentPhaseReady
	} else {
		agent.Status.Phase = AgentPhaseCreating
	}

	return nil
}

// parseKagentStatus parses status from kagent Agent CRD.
func (s *Service) parseKagentStatus(obj *unstructured.Unstructured, agent *Agent) error {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return nil
	}

	phase, _, _ := unstructured.NestedString(status, "phase")
	agent.Status.Phase = AgentPhase(phase)

	url, _, _ := unstructured.NestedString(status, "url")
	agent.Status.URL = url

	message, _, _ := unstructured.NestedString(status, "message")
	agent.Status.Message = message

	return nil
}

// ListAgents returns all managed agents.
func (s *Service) ListAgents(ctx context.Context, namespace string) ([]*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*Agent, 0)
	for _, agent := range s.agents {
		if namespace == "" || agent.Namespace == namespace {
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

// UpdateAgent updates an agent deployment.
func (s *Service) UpdateAgent(ctx context.Context, agentID string, update *Agent) (*Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, ok := s.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// Apply updates
	if update.Image != "" {
		agent.Image = update.Image
	}
	if update.Description != "" {
		agent.Description = update.Description
	}
	if update.Resources != nil {
		agent.Resources = update.Resources
	}
	if len(update.Env) > 0 {
		if agent.Env == nil {
			agent.Env = make(map[string]string)
		}
		for k, v := range update.Env {
			agent.Env[k] = v
		}
	}

	agent.UpdatedAt = time.Now()

	// Update in cluster
	k8sClient, err := s.getK8sClient(agent.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	useKagent := s.kagentCRDExists(ctx, agent.Namespace)
	if useKagent {
		if err := s.updateKagentAgent(ctx, agent); err != nil {
			return nil, fmt.Errorf("failed to update kagent agent: %w", err)
		}
	} else {
		if err := s.updateKnativeAgent(ctx, k8sClient, agent); err != nil {
			return nil, fmt.Errorf("failed to update knative agent: %w", err)
		}
	}

	return agent, nil
}

// updateKagentAgent updates an existing kagent Agent.
func (s *Service) updateKagentAgent(ctx context.Context, agent *Agent) error {
	if s.dynClient == nil {
		return fmt.Errorf("dynamic client not initialized")
	}

	obj, err := s.dynClient.Resource(kagentGVR).Namespace(agent.Namespace).Get(ctx, agent.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Update image
	if agent.Type == AgentTypeBYO {
		unstructured.SetNestedField(obj.Object, agent.Image, "spec", "byo", "deployment", "image")
	}

	_, err = s.dynClient.Resource(kagentGVR).Namespace(agent.Namespace).Update(ctx, obj, metav1.UpdateOptions{})
	return err
}

// updateKnativeAgent updates an existing Knative Service.
func (s *Service) updateKnativeAgent(ctx context.Context, k8sClient *k8s.Client, agent *Agent) error {
	spec := &k8s.KnativeServiceSpec{
		Name:    agent.Name,
		AgentID: agent.ID,
		Image:   agent.Image,
		Env:     agent.Env,
	}

	if agent.Resources != nil {
		spec.Resources = k8s.ResourceSpec{
			MemoryRequest: agent.Resources.MemoryRequest,
			MemoryLimit:   agent.Resources.MemoryLimit,
			CPURequest:    agent.Resources.CPURequest,
			CPULimit:      agent.Resources.CPULimit,
		}
	}

	return k8sClient.UpdateKnativeService(ctx, spec)
}

// DeleteAgent removes an agent deployment.
func (s *Service) DeleteAgent(ctx context.Context, agentID string) error {
	s.mu.Lock()
	agent, ok := s.agents[agentID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("agent not found: %s", agentID)
	}
	delete(s.agents, agentID)
	s.mu.Unlock()

	// Delete from cluster
	k8sClient, err := s.getK8sClient(agent.Namespace)
	if err != nil {
		s.logger.Warn("failed to get k8s client for delete", "namespace", agent.Namespace, "error", err)
		return nil
	}

	useKagent := s.kagentCRDExists(ctx, agent.Namespace)
	if useKagent && s.dynClient != nil {
		err := s.dynClient.Resource(kagentGVR).Namespace(agent.Namespace).Delete(ctx, agent.Name, metav1.DeleteOptions{})
		if err != nil {
			s.logger.Warn("failed to delete kagent agent", "name", agent.Name, "error", err)
		}
	} else {
		if err := k8sClient.DeleteKnativeService(ctx, agent.Name); err != nil {
			s.logger.Warn("failed to delete knative service", "name", agent.Name, "error", err)
		}
	}

	return nil
}

// WaitForReady blocks until an agent is ready or times out.
func (s *Service) WaitForReady(ctx context.Context, agentID string, timeout time.Duration) error {
	s.mu.RLock()
	agent, ok := s.agents[agentID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	k8sClient, err := s.getK8sClient(agent.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	useKagent := s.kagentCRDExists(ctx, agent.Namespace)
	if !useKagent {
		_, err := k8sClient.WaitForReady(ctx, agent.Name, timeout)
		return err
	}

	// Wait for kagent agent
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := s.refreshAgentStatus(ctx, agent); err != nil {
			return err
		}

		if agent.Status.Phase == AgentPhaseReady {
			return nil
		}

		if agent.Status.Phase == AgentPhaseFailed {
			return fmt.Errorf("agent failed: %s", agent.Status.Message)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return fmt.Errorf("timeout waiting for agent to be ready")
}

// ExportConfig exports agent configuration as JSON.
func (s *Service) ExportConfig(agent *Agent) ([]byte, error) {
	return json.MarshalIndent(agent, "", "  ")
}

// ImportConfig imports agent configuration from JSON.
func (s *Service) ImportConfig(data []byte) (*Agent, error) {
	var agent Agent
	if err := json.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &agent, nil
}
