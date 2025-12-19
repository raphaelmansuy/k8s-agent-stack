# Phase 2: Agent Runtime & Lifecycle

> Agent Deployment, A2A Protocol, Streaming, Kubernetes Integration

**Duration**: 3 weeks | **Status**: Not Started | **Priority**: Critical  
**Depends On**: Phase 1 (Core API)

---

## Objectives

1. Implement agent deployment to Knative
2. Implement A2A protocol endpoints (discovery, message, stream)
3. Implement Server-Sent Events (SSE) streaming
4. Implement Universal Content Model (UCM) translation
5. Integrate with existing kagent-adk-agent

---

## Architecture Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                      Agent Runtime Architecture                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   API Gateway                     Agent Runtime                          │
│   ┌─────────────┐                 ┌─────────────────────────────────┐   │
│   │             │    Kubernetes   │                                 │   │
│   │  /v1/agents │    Client       │         Knative Service         │   │
│   │  /v1/chat   │────────────────▶│                                 │   │
│   │  /a2a/v1/*  │                 │  ┌─────────┐  ┌─────────────┐  │   │
│   │             │                 │  │ Revision│  │ Revision    │  │   │
│   └─────────────┘                 │  │ v1 (90%)│  │ v2 (10%)    │  │   │
│         │                         │  └─────────┘  └─────────────┘  │   │
│         │                         └─────────────────────────────────┘   │
│         │                                        │                      │
│         │  SSE Stream                           │                      │
│         ▼                                       ▼                      │
│   ┌─────────────┐                 ┌─────────────────────────────────┐   │
│   │   Client    │◀────────────────│       Agent Container           │   │
│   │             │   A2A JSON-RPC  │  (ADK / LangGraph / Custom)     │   │
│   └─────────────┘                 └─────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Week 1: Kubernetes Integration

### 1.1 Kubernetes Client Setup

**`internal/infrastructure/k8s/client.go`**:
```go
package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes API access
type Client struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	namespace     string
}

// Knative Service GVR
var knativeServiceGVR = schema.GroupVersionResource{
	Group:    "serving.knative.dev",
	Version:  "v1",
	Resource: "services",
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfig string, namespace string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		// In-cluster config
		config, err = rest.InClusterConfig()
	} else {
		// Out-of-cluster config
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build k8s config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		namespace:     namespace,
	}, nil
}

// CreateKnativeService creates a Knative Service for an agent
func (c *Client) CreateKnativeService(ctx context.Context, spec *KnativeServiceSpec) error {
	service := c.buildKnativeService(spec)

	_, err := c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Create(ctx, service, metav1.CreateOptions{})

	if err != nil {
		return fmt.Errorf("failed to create knative service: %w", err)
	}

	return nil
}

// UpdateKnativeService updates an existing Knative Service
func (c *Client) UpdateKnativeService(ctx context.Context, spec *KnativeServiceSpec) error {
	// Get existing service first
	existing, err := c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing service: %w", err)
	}

	// Build updated service
	service := c.buildKnativeService(spec)

	// Preserve resourceVersion for optimistic locking
	service.SetResourceVersion(existing.GetResourceVersion())

	_, err = c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Update(ctx, service, metav1.UpdateOptions{})

	if err != nil {
		return fmt.Errorf("failed to update knative service: %w", err)
	}

	return nil
}

// DeleteKnativeService deletes a Knative Service
func (c *Client) DeleteKnativeService(ctx context.Context, name string) error {
	err := c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Delete(ctx, name, metav1.DeleteOptions{})

	if err != nil {
		return fmt.Errorf("failed to delete knative service: %w", err)
	}

	return nil
}

// GetKnativeServiceStatus gets the status of a Knative Service
func (c *Client) GetKnativeServiceStatus(ctx context.Context, name string) (*ServiceStatus, error) {
	service, err := c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return parseServiceStatus(service)
}

// WatchKnativeService watches for changes to a Knative Service
func (c *Client) WatchKnativeService(ctx context.Context, name string) (<-chan *ServiceStatus, error) {
	// Implementation using watch.Interface
	// ...
	return nil, nil
}

func (c *Client) buildKnativeService(spec *KnativeServiceSpec) *unstructured.Unstructured {
	// Build container spec
	container := map[string]interface{}{
		"image": spec.Image,
		"ports": []map[string]interface{}{
			{
				"containerPort": 8080,
				"protocol":      "TCP",
			},
		},
		"env": buildEnvVars(spec.Env),
		"resources": map[string]interface{}{
			"requests": map[string]interface{}{
				"cpu":    spec.Resources.CPURequest,
				"memory": spec.Resources.MemoryRequest,
			},
			"limits": map[string]interface{}{
				"cpu":    spec.Resources.CPULimit,
				"memory": spec.Resources.MemoryLimit,
			},
		},
	}

	// Build annotations for autoscaling
	annotations := map[string]interface{}{
		"autoscaling.knative.dev/min-scale":         fmt.Sprintf("%d", spec.MinScale),
		"autoscaling.knative.dev/max-scale":         fmt.Sprintf("%d", spec.MaxScale),
		"autoscaling.knative.dev/target":            fmt.Sprintf("%d", spec.ConcurrencyTarget),
		"autoscaling.knative.dev/scale-down-delay":  spec.ScaleDownDelay,
	}

	// Build labels
	labels := map[string]interface{}{
		"app.kubernetes.io/name":       spec.Name,
		"app.kubernetes.io/managed-by": "agentstack",
		"agentstack.io/agent-id":       spec.AgentID,
		"agentstack.io/project-id":     spec.ProjectID,
	}

	service := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "serving.knative.dev/v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      spec.Name,
				"namespace": c.namespace,
				"labels":    labels,
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": annotations,
						"labels":      labels,
					},
					"spec": map[string]interface{}{
						"containers":                    []interface{}{container},
						"containerConcurrency":          spec.ContainerConcurrency,
						"timeoutSeconds":                spec.TimeoutSeconds,
						"serviceAccountName":            spec.ServiceAccountName,
					},
				},
			},
		},
	}

	return service
}

// Types

type KnativeServiceSpec struct {
	Name                 string
	AgentID              string
	ProjectID            string
	Image                string
	Env                  map[string]string
	Resources            ResourceSpec
	MinScale             int
	MaxScale             int
	ConcurrencyTarget    int
	ContainerConcurrency int64
	TimeoutSeconds       int64
	ScaleDownDelay       string
	ServiceAccountName   string
}

type ResourceSpec struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
}

type ServiceStatus struct {
	Ready          bool
	URL            string
	LatestRevision string
	Traffic        []TrafficTarget
	Conditions     []Condition
}

type TrafficTarget struct {
	Revision string
	Percent  int64
	Tag      string
}

type Condition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}
```

### 1.2 Deployment Service

**`internal/domain/deployment/service.go`**:
```go
package deployment

import (
	"context"
	"fmt"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/agent"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/k8s"
	"github.com/raphaelmansuy/agentstack/internal/pkg/id"
)

type Service struct {
	repo      Repository
	agentRepo agent.Repository
	k8sClient *k8s.Client
	registry  string
}

type Repository interface {
	Create(ctx context.Context, d *Deployment) error
	GetByID(ctx context.Context, id string) (*Deployment, error)
	UpdateStatus(ctx context.Context, id string, status Status, message string) error
	ListByAgent(ctx context.Context, agentID string) ([]*Deployment, error)
}

func NewService(
	repo Repository,
	agentRepo agent.Repository,
	k8sClient *k8s.Client,
	registry string,
) *Service {
	return &Service{
		repo:      repo,
		agentRepo: agentRepo,
		k8sClient: k8sClient,
		registry:  registry,
	}
}

// Deploy creates a new deployment for an agent
func (s *Service) Deploy(ctx context.Context, agentID string, req DeployRequest) (*Deployment, error) {
	// Get agent
	ag, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	// Create deployment record
	deployment := &Deployment{
		ID:        id.Generate("dpl"),
		AgentID:   agentID,
		Status:    StatusPending,
		Source:    req.Source,
		Config:    ag.Config,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Start async deployment
	go s.executeDeployment(context.Background(), deployment, ag)

	return deployment, nil
}

func (s *Service) executeDeployment(ctx context.Context, d *Deployment, ag *agent.Agent) {
	// Update status to building
	s.repo.UpdateStatus(ctx, d.ID, StatusBuilding, "Building container image")

	// Determine image
	var image string
	if ag.Source != nil && ag.Source.Image != "" {
		image = ag.Source.Image
	} else {
		// Build image from source
		// TODO: Integrate with build system
		image = fmt.Sprintf("%s/%s:%s", s.registry, ag.Slug, d.ID)
	}

	// Update status to deploying
	s.repo.UpdateStatus(ctx, d.ID, StatusDeploying, "Creating Knative service")

	// Build Knative service spec
	spec := &k8s.KnativeServiceSpec{
		Name:      ag.Slug,
		AgentID:   ag.ID,
		ProjectID: ag.ProjectID,
		Image:     image,
		Env: map[string]string{
			"AGENT_ID":      ag.ID,
			"AGENT_NAME":    ag.Name,
			"AGENT_VERSION": d.ID,
		},
		Resources: k8s.ResourceSpec{
			CPURequest:    "100m",
			CPULimit:      "1000m",
			MemoryRequest: "256Mi",
			MemoryLimit:   "1Gi",
		},
		MinScale:             0,
		MaxScale:             10,
		ConcurrencyTarget:    10,
		ContainerConcurrency: 0, // unlimited
		TimeoutSeconds:       300,
		ScaleDownDelay:       "30s",
		ServiceAccountName:   "agentstack-agent",
	}

	// Create or update Knative service
	err := s.k8sClient.CreateKnativeService(ctx, spec)
	if err != nil {
		// Try update if create fails
		err = s.k8sClient.UpdateKnativeService(ctx, spec)
	}

	if err != nil {
		s.repo.UpdateStatus(ctx, d.ID, StatusFailed, err.Error())
		return
	}

	// Wait for ready
	s.waitForReady(ctx, d, ag.Slug)
}

func (s *Service) waitForReady(ctx context.Context, d *Deployment, serviceName string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-timeout:
			s.repo.UpdateStatus(ctx, d.ID, StatusFailed, "Timeout waiting for service to be ready")
			return
		case <-ticker.C:
			status, err := s.k8sClient.GetKnativeServiceStatus(ctx, serviceName)
			if err != nil {
				continue
			}

			if status.Ready {
				d.URL = status.URL
				d.Revision = status.LatestRevision
				s.repo.UpdateStatus(ctx, d.ID, StatusReady, "Service is ready")

				// Update agent status
				s.agentRepo.UpdateStatus(ctx, d.AgentID, agent.StatusActive)
				return
			}
		}
	}
}

// GetStatus returns the current status of a deployment
func (s *Service) GetStatus(ctx context.Context, deploymentID string) (*Deployment, error) {
	return s.repo.GetByID(ctx, deploymentID)
}

// Cancel cancels a running deployment
func (s *Service) Cancel(ctx context.Context, deploymentID string) error {
	d, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return err
	}

	if d.Status != StatusPending && d.Status != StatusBuilding && d.Status != StatusDeploying {
		return fmt.Errorf("deployment cannot be cancelled in status: %s", d.Status)
	}

	return s.repo.UpdateStatus(ctx, deploymentID, StatusCancelled, "Cancelled by user")
}

// Types

type Deployment struct {
	ID        string
	AgentID   string
	Status    Status
	Source    Source
	Config    agent.Config
	URL       string
	Revision  string
	Logs      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Status string

const (
	StatusPending   Status = "pending"
	StatusBuilding  Status = "building"
	StatusDeploying Status = "deploying"
	StatusReady     Status = "ready"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Source struct {
	Type string `json:"type"` // git, image
	Ref  string `json:"ref"`
}

type DeployRequest struct {
	Source Source `json:"source"`
}
```

---

## Week 2: A2A Protocol Implementation

### 2.1 A2A Endpoints

**`internal/api/handlers/a2a.go`**:
```go
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	"github.com/raphaelmansuy/agentstack/internal/domain/agent"
	"github.com/raphaelmansuy/agentstack/internal/domain/a2a"
	"github.com/raphaelmansuy/agentstack/internal/pkg/logger"
)

type A2AHandler struct {
	agentSvc  *agent.Service
	a2aSvc    *a2a.Service
	log       *logger.Logger
}

func NewA2AHandler(agentSvc *agent.Service, a2aSvc *a2a.Service, log *logger.Logger) *A2AHandler {
	return &A2AHandler{
		agentSvc: agentSvc,
		a2aSvc:   a2aSvc,
		log:      log,
	}
}

// AgentCard returns the public Agent Card for discovery
// GET /.well-known/agent-card.json
func (h *A2AHandler) AgentCard(c *fiber.Ctx) error {
	agentSlug := c.Params("agentSlug")

	ag, err := h.agentSvc.GetBySlug(c.UserContext(), agentSlug)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "agent not found")
	}

	card := h.buildAgentCard(ag, c.BaseURL())

	c.Set("Content-Type", "application/a2a+json")
	return c.JSON(card)
}

// SendMessage handles A2A message:send
// POST /a2a/v1/message:send
func (h *A2AHandler) SendMessage(c *fiber.Ctx) error {
	var req a2a.SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Create task
	task, err := h.a2aSvc.SendMessage(c.UserContext(), req)
	if err != nil {
		return err
	}

	c.Set("Content-Type", "application/a2a+json")
	return c.JSON(task)
}

// StreamMessage handles A2A message:stream with SSE
// POST /a2a/v1/message:stream
func (h *A2AHandler) StreamMessage(c *fiber.Ctx) error {
	var req a2a.SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // Disable nginx buffering

	c.Context().SetBodyStreamWriter(func(w *fasthttp.StreamWriter) {
		eventCh, err := h.a2aSvc.StreamMessage(c.UserContext(), req)
		if err != nil {
			writeSSEError(w, err)
			return
		}

		for event := range eventCh {
			writeSSEEvent(w, event)
		}
	})

	return nil
}

// GetTask retrieves a task by ID
// GET /a2a/v1/tasks/{taskId}
func (h *A2AHandler) GetTask(c *fiber.Ctx) error {
	taskID := c.Params("taskId")

	task, err := h.a2aSvc.GetTask(c.UserContext(), taskID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}

	c.Set("Content-Type", "application/a2a+json")
	return c.JSON(task)
}

// CancelTask cancels a running task
// POST /a2a/v1/tasks/{taskId}:cancel
func (h *A2AHandler) CancelTask(c *fiber.Ctx) error {
	taskID := c.Params("taskId")

	if err := h.a2aSvc.CancelTask(c.UserContext(), taskID); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Helper functions

func (h *A2AHandler) buildAgentCard(ag *agent.Agent, baseURL string) *a2a.AgentCard {
	return &a2a.AgentCard{
		ProtocolVersion: "1.0",
		Name:            ag.Name,
		Description:     ag.Description,
		Version:         "1.0.0",
		SupportedInterfaces: []a2a.Interface{
			{
				URL:             fmt.Sprintf("%s/a2a/v1", baseURL),
				ProtocolBinding: "HTTP+JSON",
			},
		},
		Capabilities: a2a.Capabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: false,
		},
		DefaultInputModes:  []string{"text/plain", "application/json"},
		DefaultOutputModes: []string{"text/plain", "application/json"},
		Skills:             buildSkillsFromTools(ag.Config.Tools),
	}
}

func writeSSEEvent(w *fasthttp.StreamWriter, event *a2a.StreamEvent) {
	data, _ := json.Marshal(event.Data)
	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", data)
	w.Flush()
}

func writeSSEError(w *fasthttp.StreamWriter, err error) {
	fmt.Fprintf(w, "event: error\n")
	fmt.Fprintf(w, "data: {\"error\": \"%s\"}\n\n", err.Error())
	w.Flush()
}

func buildSkillsFromTools(tools []string) []a2a.Skill {
	skills := make([]a2a.Skill, len(tools))
	for i, tool := range tools {
		skills[i] = a2a.Skill{
			ID:   tool,
			Name: tool,
		}
	}
	return skills
}
```

### 2.2 A2A Service

**`internal/domain/a2a/service.go`**:
```go
package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/agent"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/pkg/id"
)

type Service struct {
	agentRepo   agent.Repository
	taskRepo    TaskRepository
	cache       *cache.RedisClient
	httpClient  *http.Client
}

type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id string) (*Task, error)
	Update(ctx context.Context, task *Task) error
}

func NewService(
	agentRepo agent.Repository,
	taskRepo TaskRepository,
	cache *cache.RedisClient,
) *Service {
	return &Service{
		agentRepo: agentRepo,
		taskRepo:  taskRepo,
		cache:     cache,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// SendMessage sends a message to an agent and returns a task
func (s *Service) SendMessage(ctx context.Context, req SendMessageRequest) (*Task, error) {
	// Get agent
	ag, err := s.agentRepo.GetByID(ctx, req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	if ag.Status != agent.StatusActive {
		return nil, fmt.Errorf("agent is not active")
	}

	// Create task
	task := &Task{
		ID:        id.Generate("tsk"),
		AgentID:   req.AgentID,
		ContextID: req.ContextID,
		Status:    StatusPending,
		Input:     req.Message,
		CreatedAt: time.Now(),
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	// Forward to agent (async for non-streaming)
	go s.executeTask(context.Background(), task, ag, req)

	return task, nil
}

// StreamMessage streams messages from an agent
func (s *Service) StreamMessage(ctx context.Context, req SendMessageRequest) (<-chan *StreamEvent, error) {
	// Get agent
	ag, err := s.agentRepo.GetByID(ctx, req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	eventCh := make(chan *StreamEvent, 100)

	go s.streamFromAgent(ctx, ag, req, eventCh)

	return eventCh, nil
}

func (s *Service) streamFromAgent(ctx context.Context, ag *agent.Agent, req SendMessageRequest, eventCh chan<- *StreamEvent) {
	defer close(eventCh)

	// Build A2A request
	a2aReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "message/send",
		"id":      id.Generate("req"),
		"params": map[string]interface{}{
			"message": req.Message,
		},
	}

	body, _ := json.Marshal(a2aReq)

	// Make streaming request to agent
	httpReq, err := http.NewRequestWithContext(ctx, "POST", ag.URLs.API, bytes.NewReader(body))
	if err != nil {
		eventCh <- &StreamEvent{Type: "error", Data: map[string]string{"error": err.Error()}}
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		eventCh <- &StreamEvent{Type: "error", Data: map[string]string{"error": err.Error()}}
		return
	}
	defer resp.Body.Close()

	// Parse SSE stream from agent
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			eventCh <- &StreamEvent{Type: "error", Data: map[string]string{"error": err.Error()}}
			return
		}

		// Parse SSE event
		event := parseSSELine(line)
		if event != nil {
			// Translate to AG-UI format if needed
			translated := s.translateEvent(event)
			eventCh <- translated
		}
	}
}

func (s *Service) executeTask(ctx context.Context, task *Task, ag *agent.Agent, req SendMessageRequest) {
	// Update status
	task.Status = StatusInProgress
	s.taskRepo.Update(ctx, task)

	// Build request to agent
	a2aReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "message/send",
		"id":      task.ID,
		"params": map[string]interface{}{
			"message": req.Message,
		},
	}

	body, _ := json.Marshal(a2aReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", ag.URLs.API, bytes.NewReader(body))
	if err != nil {
		task.Status = StatusFailed
		task.Error = err.Error()
		s.taskRepo.Update(ctx, task)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		task.Status = StatusFailed
		task.Error = err.Error()
		s.taskRepo.Update(ctx, task)
		return
	}
	defer resp.Body.Close()

	// Parse response
	var a2aResp A2AResponse
	if err := json.NewDecoder(resp.Body).Decode(&a2aResp); err != nil {
		task.Status = StatusFailed
		task.Error = err.Error()
		s.taskRepo.Update(ctx, task)
		return
	}

	task.Status = StatusCompleted
	task.Output = a2aResp.Result
	task.CompletedAt = time.Now()
	s.taskRepo.Update(ctx, task)
}

// GetTask retrieves a task by ID
func (s *Service) GetTask(ctx context.Context, taskID string) (*Task, error) {
	return s.taskRepo.GetByID(ctx, taskID)
}

// CancelTask cancels a running task
func (s *Service) CancelTask(ctx context.Context, taskID string) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status != StatusPending && task.Status != StatusInProgress {
		return fmt.Errorf("task cannot be cancelled")
	}

	task.Status = StatusCancelled
	return s.taskRepo.Update(ctx, task)
}

// Translate events from A2A format to AG-UI format
func (s *Service) translateEvent(event *StreamEvent) *StreamEvent {
	switch event.Type {
	case "status_update":
		return &StreamEvent{Type: "StateSnapshot", Data: event.Data}
	case "artifact":
		return &StreamEvent{Type: "TextMessageContent", Data: event.Data}
	default:
		return event
	}
}
```

---

## Week 3: Chat API & SSE Streaming

### 3.1 Chat Handler with SSE

**`internal/api/handlers/chat.go`**:
```go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/domain/agent"
	"github.com/raphaelmansuy/agentstack/internal/domain/chat"
	"github.com/raphaelmansuy/agentstack/internal/pkg/logger"
	"github.com/raphaelmansuy/agentstack/internal/pkg/ucm"
)

type ChatHandler struct {
	chatSvc  *chat.Service
	agentSvc *agent.Service
	log      *logger.Logger
}

func NewChatHandler(chatSvc *chat.Service, agentSvc *agent.Service, log *logger.Logger) *ChatHandler {
	return &ChatHandler{
		chatSvc:  chatSvc,
		agentSvc: agentSvc,
		log:      log,
	}
}

// ChatSync handles synchronous chat
// POST /v1/agents/{agentId}/chat
func (h *ChatHandler) ChatSync(c *fiber.Ctx) error {
	agentID := c.Params("agentId")
	auth := middleware.GetAuthContext(c)

	var req chat.Request
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Normalize input to UCM format
	content, err := ucm.Normalize(req.Message)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid message format")
	}

	resp, err := h.chatSvc.Chat(c.UserContext(), chat.ChatParams{
		AgentID:   agentID,
		ProjectID: auth.ProjectID,
		SessionID: req.SessionID,
		Content:   content,
	})
	if err != nil {
		return err
	}

	return c.JSON(resp)
}

// ChatStream handles streaming chat with SSE
// POST /v1/agents/{agentId}/chat/stream
func (h *ChatHandler) ChatStream(c *fiber.Ctx) error {
	agentID := c.Params("agentId")
	auth := middleware.GetAuthContext(c)

	var req chat.Request
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Normalize input to UCM format
	content, err := ucm.Normalize(req.Message)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid message format")
	}

	// Determine SSE format (AG-UI or legacy)
	format := c.Get("X-AgentStack-SSE-Format", "legacy")

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *fasthttp.StreamWriter) {
		eventCh, err := h.chatSvc.ChatStream(c.UserContext(), chat.ChatParams{
			AgentID:   agentID,
			ProjectID: auth.ProjectID,
			SessionID: req.SessionID,
			Content:   content,
		})
		if err != nil {
			writeSSEError(w, err, format)
			return
		}

		for event := range eventCh {
			writeSSEEvent(w, event, format)
		}
	})

	return nil
}

func writeSSEEvent(w *fasthttp.StreamWriter, event *chat.StreamEvent, format string) {
	// Translate to format if needed
	eventType := event.Type
	if format == "agui" {
		eventType = translateToAGUI(event.Type)
	}

	data, _ := json.Marshal(event.Data)
	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: %s\n\n", data)
	w.Flush()
}

func writeSSEError(w *fasthttp.StreamWriter, err error, format string) {
	eventType := "error"
	if format == "agui" {
		eventType = "RunError"
	}
	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: {\"error\": \"%s\"}\n\n", err.Error())
	w.Flush()
}

func translateToAGUI(legacyType string) string {
	mapping := map[string]string{
		"message_start":  "TextMessageStart",
		"content_delta":  "TextMessageContent",
		"message_end":    "TextMessageEnd",
		"tool_start":     "ToolCallStart",
		"tool_result":    "ToolCallEnd",
	}
	if agui, ok := mapping[legacyType]; ok {
		return agui
	}
	return legacyType
}
```

### 3.2 Universal Content Model

**`internal/pkg/ucm/ucm.go`**:
```go
package ucm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Part represents a universal content part
type Part struct {
	Type      PartType               `json:"type"`
	Text      string                 `json:"text,omitempty"`
	Data      string                 `json:"data,omitempty"`
	URI       string                 `json:"uri,omitempty"`
	FileID    string                 `json:"file_id,omitempty"`
	MimeType  string                 `json:"mime_type,omitempty"`
	Name      string                 `json:"name,omitempty"`
	CallID    string                 `json:"call_id,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	JSONData  map[string]interface{} `json:"json_data,omitempty"`
}

type PartType string

const (
	PartTypeText           PartType = "text"
	PartTypeImage          PartType = "image"
	PartTypeAudio          PartType = "audio"
	PartTypeVideo          PartType = "video"
	PartTypeDocument       PartType = "document"
	PartTypeFile           PartType = "file"
	PartTypeFunctionCall   PartType = "function_call"
	PartTypeFunctionResult PartType = "function_result"
	PartTypeData           PartType = "data"
)

// Content represents a complete message
type Content struct {
	Role  Role   `json:"role"`
	Parts []Part `json:"parts"`
}

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// Normalize converts various input formats to UCM
func Normalize(input interface{}) (*Content, error) {
	switch v := input.(type) {
	case string:
		// Simple string → text part
		return &Content{
			Role: RoleUser,
			Parts: []Part{
				{Type: PartTypeText, Text: v},
			},
		}, nil

	case map[string]interface{}:
		// Check if it's already Content format
		if role, ok := v["role"].(string); ok {
			return parseContent(v)
		}
		// Check if it's a single Part
		if partType, ok := v["type"].(string); ok {
			part, err := parsePart(v)
			if err != nil {
				return nil, err
			}
			return &Content{
				Role:  RoleUser,
				Parts: []Part{*part},
			}, nil
		}
		return nil, fmt.Errorf("unknown message format")

	case []interface{}:
		// Array of parts
		parts := make([]Part, len(v))
		for i, p := range v {
			pMap, ok := p.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid part at index %d", i)
			}
			part, err := parsePart(pMap)
			if err != nil {
				return nil, err
			}
			parts[i] = *part
		}
		return &Content{
			Role:  RoleUser,
			Parts: parts,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported input type: %T", input)
	}
}

func parseContent(m map[string]interface{}) (*Content, error) {
	content := &Content{}

	if role, ok := m["role"].(string); ok {
		content.Role = Role(role)
	}

	if parts, ok := m["parts"].([]interface{}); ok {
		content.Parts = make([]Part, len(parts))
		for i, p := range parts {
			pMap, ok := p.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid part at index %d", i)
			}
			part, err := parsePart(pMap)
			if err != nil {
				return nil, err
			}
			content.Parts[i] = *part
		}
	}

	return content, nil
}

func parsePart(m map[string]interface{}) (*Part, error) {
	part := &Part{}

	if t, ok := m["type"].(string); ok {
		part.Type = PartType(t)
	}

	// Map common fields
	if text, ok := m["text"].(string); ok {
		part.Text = text
	}
	if data, ok := m["data"].(string); ok {
		part.Data = data
	}
	if uri, ok := m["uri"].(string); ok {
		part.URI = uri
	}
	if mimeType, ok := m["mime_type"].(string); ok {
		part.MimeType = mimeType
	}
	if name, ok := m["name"].(string); ok {
		part.Name = name
	}
	if callID, ok := m["call_id"].(string); ok {
		part.CallID = callID
	}
	if args, ok := m["arguments"].(map[string]interface{}); ok {
		part.Arguments = args
	}

	return part, nil
}

// ToOpenAI translates UCM to OpenAI format
func (c *Content) ToOpenAI() map[string]interface{} {
	msg := map[string]interface{}{
		"role": string(c.Role),
	}

	// Simple case: single text part
	if len(c.Parts) == 1 && c.Parts[0].Type == PartTypeText {
		msg["content"] = c.Parts[0].Text
		return msg
	}

	// Multi-part content
	content := make([]map[string]interface{}, len(c.Parts))
	for i, part := range c.Parts {
		content[i] = part.toOpenAIPart()
	}
	msg["content"] = content

	return msg
}

func (p *Part) toOpenAIPart() map[string]interface{} {
	switch p.Type {
	case PartTypeText:
		return map[string]interface{}{
			"type": "text",
			"text": p.Text,
		}
	case PartTypeImage:
		if p.Data != "" {
			return map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]string{
					"url": fmt.Sprintf("data:%s;base64,%s", p.MimeType, p.Data),
				},
			}
		}
		return map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]string{
				"url": p.URI,
			},
		}
	case PartTypeFunctionCall:
		return map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":      p.Name,
				"arguments": p.Arguments,
			},
		}
	default:
		return map[string]interface{}{
			"type": string(p.Type),
		}
	}
}

// ToAnthropic translates UCM to Anthropic format
func (c *Content) ToAnthropic() map[string]interface{} {
	// Similar implementation for Anthropic
	// ...
	return nil
}

// ToGemini translates UCM to Gemini format
func (c *Content) ToGemini() map[string]interface{} {
	// Similar implementation for Gemini
	// ...
	return nil
}
```

---

## Deliverables Checklist

### Week 1
- [ ] Kubernetes client implementation
- [ ] Knative Service CRUD operations
- [ ] Deployment service
- [ ] Async deployment pipeline
- [ ] Status watching

### Week 2
- [ ] A2A Agent Card endpoint
- [ ] A2A SendMessage endpoint
- [ ] A2A StreamMessage endpoint (SSE)
- [ ] Task management (CRUD)
- [ ] Agent proxy to Knative services

### Week 3
- [ ] Chat API (sync and stream)
- [ ] Universal Content Model implementation
- [ ] AG-UI event format support
- [ ] Session management
- [ ] Integration tests

---

## Definition of Done

- [ ] Agent deployment creates working Knative Service
- [ ] A2A discovery returns valid Agent Card
- [ ] Streaming chat works end-to-end
- [ ] UCM translates correctly to OpenAI format
- [ ] Integration tests pass
- [ ] Performance: <500ms cold start (excluding Knative)

---

## Sage AI Guidance

### Key Architectural Decisions

1. **Async Deployment**: Deployment is async with status polling. Don't block on Knative readiness.

2. **SSE for Streaming**: Use Server-Sent Events, not WebSockets. Simpler, works through proxies.

3. **UCM Translation**: Translate at the gateway, not in agents. Agents speak UCM internally.

### Critical Path Items

| Item | Risk | Mitigation |
|------|------|------------|
| Knative integration | Medium | Test with mock first, then real cluster |
| SSE through Fiber | Low | Use `SetBodyStreamWriter`, tested pattern |
| A2A compatibility | Medium | Validate against A2A spec, use test suite |

### Testing Strategy

```text
1. Unit test UCM translation (all providers)
2. Integration test deployment flow (mock K8s)
3. E2E test chat streaming (real agent)
4. Load test SSE connections (1000 concurrent)
```

---

**Next Phase**: [003-evaluation-safety.md](003-evaluation-safety.md) - MLflow integration, safety gates
