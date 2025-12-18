// Package audit provides audit logging functionality.
package audit

import (
	"encoding/json"
	"time"
)

// EventType categorizes audit events.
type EventType string

const (
	// Authentication events
	EventLogin         EventType = "auth.login"
	EventLogout        EventType = "auth.logout"
	EventAPIKeyCreated EventType = "auth.api_key_created"
	EventAPIKeyRevoked EventType = "auth.api_key_revoked"

	// Resource events
	EventAgentCreated  EventType = "agent.created"
	EventAgentUpdated  EventType = "agent.updated"
	EventAgentDeleted  EventType = "agent.deleted"
	EventAgentDeployed EventType = "agent.deployed"
	EventAgentInvoked  EventType = "agent.invoked"

	EventProjectCreated EventType = "project.created"
	EventProjectUpdated EventType = "project.updated"
	EventProjectDeleted EventType = "project.deleted"

	EventDeploymentCreated  EventType = "deployment.created"
	EventDeploymentUpdated  EventType = "deployment.updated"
	EventDeploymentDeleted  EventType = "deployment.deleted"
	EventDeploymentScaled   EventType = "deployment.scaled"
	EventDeploymentRestart  EventType = "deployment.restarted"

	// Access events
	EventMemberInvited    EventType = "member.invited"
	EventMemberRemoved    EventType = "member.removed"
	EventRoleAssigned     EventType = "role.assigned"
	EventRoleRevoked      EventType = "role.revoked"
	EventPermissionDenied EventType = "permission.denied"

	// System events
	EventQuotaExceeded EventType = "quota.exceeded"
	EventRateLimited   EventType = "rate.limited"
)

// Event represents an audit log entry.
type Event struct {
	ID         string          `json:"id"`
	Type       EventType       `json:"type"`
	Timestamp  time.Time       `json:"timestamp"`
	TeamID     string          `json:"team_id"`
	ProjectID  string          `json:"project_id,omitempty"`
	ActorID    string          `json:"actor_id"`
	ActorType  string          `json:"actor_type"` // user, api_key, system
	ActorEmail string          `json:"actor_email,omitempty"`
	ResourceID string          `json:"resource_id,omitempty"`
	Resource   string          `json:"resource,omitempty"`
	Action     string          `json:"action"`
	Result     string          `json:"result"` // success, failure, denied
	IPAddress  string          `json:"ip_address,omitempty"`
	UserAgent  string          `json:"user_agent,omitempty"`
	RequestID  string          `json:"request_id,omitempty"`
	Details    json.RawMessage `json:"details,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// EventDetails holds additional event-specific information.
type EventDetails struct {
	// For resource events
	Changes map[string]Change `json:"changes,omitempty"`

	// For auth events
	Method     string `json:"method,omitempty"`
	FailReason string `json:"fail_reason,omitempty"`

	// For permission events
	Permission string `json:"permission,omitempty"`
	Scope      string `json:"scope,omitempty"`

	// For rate/quota events
	Limit   int64 `json:"limit,omitempty"`
	Current int64 `json:"current,omitempty"`
}

// Change represents a before/after change.
type Change struct {
	From interface{} `json:"from,omitempty"`
	To   interface{} `json:"to,omitempty"`
}

// LogParams holds parameters for logging an audit event.
type LogParams struct {
	Type       EventType
	TeamID     string
	ProjectID  string
	ActorID    string
	ActorType  string
	ActorEmail string
	ResourceID string
	Resource   string
	Action     string
	Result     string
	IPAddress  string
	UserAgent  string
	RequestID  string
	Details    interface{}
}

// Filter defines criteria for querying audit events.
type Filter struct {
	TeamID     string
	ProjectID  string
	ActorID    string
	EventTypes []EventType
	StartTime  time.Time
	EndTime    time.Time
	Resource   string
	ResourceID string
	Result     string
	Limit      int
	Offset     int
}

// EventResult constants.
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
	ResultDenied  = "denied"
)

// ActorType constants.
const (
	ActorUser   = "user"
	ActorAPIKey = "api_key"
	ActorSystem = "system"
)
