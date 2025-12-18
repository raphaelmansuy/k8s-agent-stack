// Package audit provides audit logging functionality.
package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Writer defines the interface for writing audit events.
type Writer interface {
	Write(ctx context.Context, event *Event) error
	Query(ctx context.Context, filter Filter) ([]Event, error)
}

// Service provides audit logging functionality.
type Service struct {
	writer Writer
}

// NewService creates a new audit service.
func NewService(writer Writer) *Service {
	return &Service{writer: writer}
}

// Log records an audit event.
func (s *Service) Log(ctx context.Context, event *Event) error {
	if event.ID == "" {
		event.ID = "evt_" + uuid.New().String()[:8]
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	return s.writer.Write(ctx, event)
}

// LogAction is a convenience method for logging common actions.
func (s *Service) LogAction(ctx context.Context, params LogParams) error {
	var details json.RawMessage
	if params.Details != nil {
		data, err := json.Marshal(params.Details)
		if err != nil {
			return err
		}
		details = data
	}

	event := &Event{
		ID:         "evt_" + uuid.New().String()[:8],
		Type:       params.Type,
		Timestamp:  time.Now().UTC(),
		TeamID:     params.TeamID,
		ProjectID:  params.ProjectID,
		ActorID:    params.ActorID,
		ActorType:  params.ActorType,
		ActorEmail: params.ActorEmail,
		ResourceID: params.ResourceID,
		Resource:   params.Resource,
		Action:     params.Action,
		Result:     params.Result,
		IPAddress:  params.IPAddress,
		UserAgent:  params.UserAgent,
		RequestID:  params.RequestID,
		Details:    details,
	}

	return s.writer.Write(ctx, event)
}

// Query retrieves audit events based on filter criteria.
func (s *Service) Query(ctx context.Context, filter Filter) ([]Event, error) {
	// Apply defaults
	if filter.Limit == 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}

	return s.writer.Query(ctx, filter)
}

// LogLogin logs a login event.
func (s *Service) LogLogin(ctx context.Context, teamID, userID, email, ipAddress, userAgent string, success bool) error {
	result := ResultSuccess
	if !success {
		result = ResultFailure
	}
	return s.LogAction(ctx, LogParams{
		Type:       EventLogin,
		TeamID:     teamID,
		ActorID:    userID,
		ActorType:  ActorUser,
		ActorEmail: email,
		Action:     "login",
		Result:     result,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})
}

// LogLogout logs a logout event.
func (s *Service) LogLogout(ctx context.Context, teamID, userID, email string) error {
	return s.LogAction(ctx, LogParams{
		Type:       EventLogout,
		TeamID:     teamID,
		ActorID:    userID,
		ActorType:  ActorUser,
		ActorEmail: email,
		Action:     "logout",
		Result:     ResultSuccess,
	})
}

// LogResourceCreated logs a resource creation event.
func (s *Service) LogResourceCreated(ctx context.Context, params ResourceEventParams) error {
	return s.LogAction(ctx, LogParams{
		Type:       params.EventType,
		TeamID:     params.TeamID,
		ProjectID:  params.ProjectID,
		ActorID:    params.ActorID,
		ActorType:  params.ActorType,
		ResourceID: params.ResourceID,
		Resource:   params.Resource,
		Action:     "create",
		Result:     ResultSuccess,
		IPAddress:  params.IPAddress,
		UserAgent:  params.UserAgent,
		RequestID:  params.RequestID,
	})
}

// LogResourceUpdated logs a resource update event.
func (s *Service) LogResourceUpdated(ctx context.Context, params ResourceEventParams, changes map[string]Change) error {
	return s.LogAction(ctx, LogParams{
		Type:       params.EventType,
		TeamID:     params.TeamID,
		ProjectID:  params.ProjectID,
		ActorID:    params.ActorID,
		ActorType:  params.ActorType,
		ResourceID: params.ResourceID,
		Resource:   params.Resource,
		Action:     "update",
		Result:     ResultSuccess,
		IPAddress:  params.IPAddress,
		UserAgent:  params.UserAgent,
		RequestID:  params.RequestID,
		Details:    EventDetails{Changes: changes},
	})
}

// LogResourceDeleted logs a resource deletion event.
func (s *Service) LogResourceDeleted(ctx context.Context, params ResourceEventParams) error {
	return s.LogAction(ctx, LogParams{
		Type:       params.EventType,
		TeamID:     params.TeamID,
		ProjectID:  params.ProjectID,
		ActorID:    params.ActorID,
		ActorType:  params.ActorType,
		ResourceID: params.ResourceID,
		Resource:   params.Resource,
		Action:     "delete",
		Result:     ResultSuccess,
		IPAddress:  params.IPAddress,
		UserAgent:  params.UserAgent,
		RequestID:  params.RequestID,
	})
}

// LogPermissionDenied logs a permission denied event.
func (s *Service) LogPermissionDenied(ctx context.Context, teamID, actorID, resource, action, ipAddress string) error {
	return s.LogAction(ctx, LogParams{
		Type:      EventPermissionDenied,
		TeamID:    teamID,
		ActorID:   actorID,
		ActorType: ActorUser,
		Resource:  resource,
		Action:    action,
		Result:    ResultDenied,
		IPAddress: ipAddress,
		Details: EventDetails{
			Permission: action,
		},
	})
}

// LogQuotaExceeded logs a quota exceeded event.
func (s *Service) LogQuotaExceeded(ctx context.Context, teamID, actorID, quotaType string, limit, current int64) error {
	return s.LogAction(ctx, LogParams{
		Type:      EventQuotaExceeded,
		TeamID:    teamID,
		ActorID:   actorID,
		ActorType: ActorUser,
		Resource:  quotaType,
		Action:    "check",
		Result:    ResultDenied,
		Details: EventDetails{
			Limit:   limit,
			Current: current,
		},
	})
}

// LogRateLimited logs a rate limit event.
func (s *Service) LogRateLimited(ctx context.Context, teamID, actorID, ipAddress string) error {
	return s.LogAction(ctx, LogParams{
		Type:      EventRateLimited,
		TeamID:    teamID,
		ActorID:   actorID,
		ActorType: ActorUser,
		Action:    "request",
		Result:    ResultDenied,
		IPAddress: ipAddress,
	})
}

// ResourceEventParams holds common parameters for resource events.
type ResourceEventParams struct {
	EventType  EventType
	TeamID     string
	ProjectID  string
	ActorID    string
	ActorType  string
	ResourceID string
	Resource   string
	IPAddress  string
	UserAgent  string
	RequestID  string
}
