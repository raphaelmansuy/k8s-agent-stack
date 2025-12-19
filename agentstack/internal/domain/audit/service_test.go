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

package audit

import (
	"context"
	"testing"
	"time"
)

// mockWriter implements Writer for testing.
type mockWriter struct {
	events []Event
}

func (m *mockWriter) Write(ctx context.Context, event *Event) error {
	m.events = append(m.events, *event)
	return nil
}

func (m *mockWriter) Query(ctx context.Context, filter Filter) ([]Event, error) {
	result := make([]Event, 0, len(m.events))
	for _, e := range m.events {
		if filter.TeamID != "" && e.TeamID != filter.TeamID {
			continue
		}
		if filter.ActorID != "" && e.ActorID != filter.ActorID {
			continue
		}
		if filter.ResourceID != "" && e.ResourceID != filter.ResourceID {
			continue
		}
		result = append(result, e)
	}

	// Apply limit
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result, nil
}

func TestLog(t *testing.T) {
	writer := &mockWriter{}
	svc := NewService(writer)

	event := &Event{
		Type:       EventAgentCreated,
		TeamID:     "team1",
		ActorID:    "user1",
		ActorType:  ActorUser,
		ResourceID: "agent1",
		Resource:   "agent",
		Action:     "create",
		Result:     ResultSuccess,
	}

	err := svc.Log(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writer.events) != 1 {
		t.Errorf("got %d events, want 1", len(writer.events))
	}

	logged := writer.events[0]
	if logged.ID == "" {
		t.Error("event should have an ID")
	}
	if logged.Timestamp.IsZero() {
		t.Error("event should have a timestamp")
	}
}

func TestLogAction(t *testing.T) {
	writer := &mockWriter{}
	svc := NewService(writer)

	err := svc.LogAction(context.Background(), LogParams{
		Type:       EventAgentCreated,
		TeamID:     "team1",
		ActorID:    "user1",
		ActorType:  ActorUser,
		ResourceID: "agent1",
		Resource:   "agent",
		Action:     "create",
		Result:     ResultSuccess,
		Details:    map[string]string{"name": "test-agent"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writer.events) != 1 {
		t.Errorf("got %d events, want 1", len(writer.events))
	}

	logged := writer.events[0]
	if logged.Details == nil {
		t.Error("event should have details")
	}
}

func TestQuery(t *testing.T) {
	writer := &mockWriter{
		events: []Event{
			{ID: "1", TeamID: "team1", ActorID: "user1"},
			{ID: "2", TeamID: "team1", ActorID: "user2"},
			{ID: "3", TeamID: "team2", ActorID: "user1"},
		},
	}
	svc := NewService(writer)

	// Filter by team
	events, err := svc.Query(context.Background(), Filter{
		TeamID: "team1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("got %d events, want 2", len(events))
	}

	// Filter by actor
	events, err = svc.Query(context.Background(), Filter{
		ActorID: "user1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("got %d events, want 2", len(events))
	}
}

func TestQueryLimit(t *testing.T) {
	writer := &mockWriter{}
	svc := NewService(writer)

	// Query with default limit
	_, err := svc.Query(context.Background(), Filter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Query with large limit (should be capped to 1000)
	_, err = svc.Query(context.Background(), Filter{Limit: 10000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogLogin(t *testing.T) {
	writer := &mockWriter{}
	svc := NewService(writer)

	err := svc.LogLogin(context.Background(), "team1", "user1", "user@example.com", "127.0.0.1", "Mozilla/5.0", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writer.events) != 1 {
		t.Errorf("got %d events, want 1", len(writer.events))
	}

	logged := writer.events[0]
	if logged.Type != EventLogin {
		t.Errorf("got type=%s, want %s", logged.Type, EventLogin)
	}
	if logged.Result != ResultSuccess {
		t.Errorf("got result=%s, want %s", logged.Result, ResultSuccess)
	}
}

func TestLogPermissionDenied(t *testing.T) {
	writer := &mockWriter{}
	svc := NewService(writer)

	err := svc.LogPermissionDenied(context.Background(), "team1", "user1", "agent", "delete", "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writer.events) != 1 {
		t.Errorf("got %d events, want 1", len(writer.events))
	}

	logged := writer.events[0]
	if logged.Type != EventPermissionDenied {
		t.Errorf("got type=%s, want %s", logged.Type, EventPermissionDenied)
	}
	if logged.Result != ResultDenied {
		t.Errorf("got result=%s, want %s", logged.Result, ResultDenied)
	}
}

func TestLogQuotaExceeded(t *testing.T) {
	writer := &mockWriter{}
	svc := NewService(writer)

	err := svc.LogQuotaExceeded(context.Background(), "team1", "user1", "agents", 5, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writer.events) != 1 {
		t.Errorf("got %d events, want 1", len(writer.events))
	}

	logged := writer.events[0]
	if logged.Type != EventQuotaExceeded {
		t.Errorf("got type=%s, want %s", logged.Type, EventQuotaExceeded)
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Verify event type strings
	if EventLogin != "auth.login" {
		t.Errorf("EventLogin = %s, want auth.login", EventLogin)
	}
	if EventAgentCreated != "agent.created" {
		t.Errorf("EventAgentCreated = %s, want agent.created", EventAgentCreated)
	}
}

func TestResultConstants(t *testing.T) {
	if ResultSuccess != "success" {
		t.Errorf("ResultSuccess = %s, want success", ResultSuccess)
	}
	if ResultFailure != "failure" {
		t.Errorf("ResultFailure = %s, want failure", ResultFailure)
	}
	if ResultDenied != "denied" {
		t.Errorf("ResultDenied = %s, want denied", ResultDenied)
	}
}

func TestEventTimestamp(t *testing.T) {
	writer := &mockWriter{}
	svc := NewService(writer)

	before := time.Now().UTC()
	err := svc.Log(context.Background(), &Event{
		Type:   EventAgentCreated,
		TeamID: "team1",
	})
	after := time.Now().UTC()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logged := writer.events[0]
	if logged.Timestamp.Before(before) || logged.Timestamp.After(after) {
		t.Errorf("timestamp %v should be between %v and %v", logged.Timestamp, before, after)
	}
}
