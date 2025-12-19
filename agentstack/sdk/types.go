// Package sdk provides the AgentStack Go SDK for programmatic access to the API.
package sdk

import (
	"encoding/json"
	"time"
)

// Agent represents an agent in the system.
type Agent struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	ProjectID   string            `json:"project_id"`
	Version     string            `json:"version,omitempty"`
	Status      string            `json:"status"`
	Endpoint    string            `json:"endpoint,omitempty"`
	Tools       []AgentTool       `json:"tools,omitempty"`
	ModelConfig *ModelConfig      `json:"model_config,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// AgentTool represents a tool available to an agent.
type AgentTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ModelConfig represents model configuration for an agent.
type ModelConfig struct {
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	Temperature   *float64 `json:"temperature,omitempty"`
	MaxTokens     *int     `json:"max_tokens,omitempty"`
	TopP          *float64 `json:"top_p,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`
}

// CreateAgentRequest represents a request to create an agent.
type CreateAgentRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Slug        string            `json:"slug"`
	ProjectID   string            `json:"project_id"`
	Framework   string            `json:"framework,omitempty"`
	Tools       []AgentTool       `json:"tools,omitempty"`
	ModelConfig *ModelConfig      `json:"model_config,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// UpdateAgentRequest represents a request to update an agent.
type UpdateAgentRequest struct {
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Tools       []AgentTool       `json:"tools,omitempty"`
	ModelConfig *ModelConfig      `json:"model_config,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ListAgentsResponse represents a paginated list of agents.
type ListAgentsResponse struct {
	Agents     []Agent `json:"agents"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// Project represents a project in the system.
type Project struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// CreateProjectRequest represents a request to create a project.
type CreateProjectRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Slug        string            `json:"slug"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// UpdateProjectRequest represents a request to update a project.
type UpdateProjectRequest struct {
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ListProjectsResponse represents a paginated list of projects.
type ListProjectsResponse struct {
	Projects   []Project `json:"projects"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

// Deployment represents a deployment in the system.
type Deployment struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Image       string            `json:"image"`
	Namespace   string            `json:"namespace"`
	Type        string            `json:"type"`
	Env         map[string]string `json:"env,omitempty"`
	Replicas    int32             `json:"replicas,omitempty"`
	Status      DeploymentStatus  `json:"status"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// DeploymentStatus represents the status of a deployment.
type DeploymentStatus struct {
	Phase      string   `json:"phase"`
	URL        string   `json:"url,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
	Message    string   `json:"message,omitempty"`
}

// DeploymentConfig represents deployment configuration.
type DeploymentConfig struct {
	Memory      string             `json:"memory,omitempty"`
	CPU         string             `json:"cpu,omitempty"`
	Replicas    int                `json:"replicas,omitempty"`
	Environment map[string]string  `json:"environment,omitempty"`
	Secrets     []string           `json:"secrets,omitempty"`
	AutoScaling *AutoScalingConfig `json:"auto_scaling,omitempty"`
}

// AutoScalingConfig represents auto-scaling configuration.
type AutoScalingConfig struct {
	Enabled     bool `json:"enabled"`
	MinReplicas int  `json:"min_replicas"`
	MaxReplicas int  `json:"max_replicas"`
	TargetCPU   int  `json:"target_cpu_percent,omitempty"`
}

// CreateDeploymentRequest represents a request to create a deployment.
type CreateDeploymentRequest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Image       string            `json:"image"`
	Namespace   string            `json:"namespace,omitempty"`
	Type        string            `json:"type,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Replicas    int32             `json:"replicas,omitempty"`
}

// UpdateDeploymentRequest represents a request to update a deployment.
type UpdateDeploymentRequest struct {
	Replicas int               `json:"replicas,omitempty"`
	Config   *DeploymentConfig `json:"config,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ListDeploymentsResponse represents a paginated list of deployments.
type ListDeploymentsResponse struct {
	Deployments []Deployment `json:"deployments"`
	Total       int          `json:"total"`
	Page        int          `json:"page"`
	PageSize    int          `json:"page_size"`
	TotalPages  int          `json:"total_pages"`
}

// ChatMessage represents a message in a chat.
type ChatMessage struct {
	Role     string                 `json:"role"`
	Content  string                 `json:"content"`
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ChatRequest represents a chat request.
type ChatRequest struct {
	AgentID   string            `json:"agent_id"`
	SessionID string            `json:"session_id,omitempty"`
	Messages  []ChatMessage     `json:"messages"`
	Stream    bool              `json:"stream,omitempty"`
	Options   *ChatOptions      `json:"options,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ChatOptions represents options for a chat request.
type ChatOptions struct {
	Temperature   *float64 `json:"temperature,omitempty"`
	MaxTokens     *int     `json:"max_tokens,omitempty"`
	TopP          *float64 `json:"top_p,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`
}

// ChatResponse represents a non-streaming chat response.
type ChatResponse struct {
	ID        string      `json:"id"`
	SessionID string      `json:"session_id"`
	Message   ChatMessage `json:"message"`
	Usage     *Usage      `json:"usage,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

// Usage represents token usage statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamEvent represents a server-sent event.
type StreamEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// StreamDelta represents a streaming delta event.
type StreamDelta struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id,omitempty"`
	Content   string `json:"content"`
	Role      string `json:"role,omitempty"`
	Done      bool   `json:"done"`
}

// AuthToken represents an authentication token.
type AuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshTokenRequest represents a token refresh request.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// User represents a user in the system.
type User struct {
	ID        string            `json:"id"`
	Email     string            `json:"email"`
	Name      string            `json:"name,omitempty"`
	Role      string            `json:"role,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// APIKeyRequest represents a request to create an API key.
type APIKeyRequest struct {
	Name      string     `json:"name"`
	ProjectID string     `json:"project_id,omitempty"`
	Scopes    []string   `json:"scopes,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// APIKey represents an API key.
type APIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Key        string     `json:"api_key,omitempty"` // Only present on creation
	Prefix     string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ListAPIKeysResponse represents a list of API keys.
type ListAPIKeysResponse struct {
	Keys []APIKey `json:"keys"`
}

// LogEntry represents a log entry.
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ListLogsRequest represents a request to list logs.
type ListLogsRequest struct {
	AgentID      string    `json:"agent_id,omitempty"`
	DeploymentID string    `json:"deployment_id,omitempty"`
	Level        string    `json:"level,omitempty"`
	StartTime    time.Time `json:"start_time,omitempty"`
	EndTime      time.Time `json:"end_time,omitempty"`
	Limit        int       `json:"limit,omitempty"`
}

// ListLogsResponse represents a list of logs.
type ListLogsResponse struct {
	Logs       []LogEntry `json:"logs"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details.
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ListOptions represents common list options.
type ListOptions struct {
	ProjectID string `json:"project_id,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
	Sort      string `json:"sort,omitempty"`
	Order     string `json:"order,omitempty"`
}
