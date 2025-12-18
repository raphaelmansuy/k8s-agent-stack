# Server-Sent Events (SSE) Streaming

Implementation patterns for real-time streaming in AgentStack.

## SSE Basics

SSE provides one-way server-to-client streaming over HTTP.

Format:
```
event: message_start
data: {"id": "msg_123"}

event: content_delta
data: {"delta": "Hello"}

event: message_end
data: {"usage": {"tokens": 42}}
```

## Fiber SSE Implementation

### Basic Streaming Handler

```go
// internal/api/handlers/chat.go
package handlers

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/valyala/fasthttp"
)

type ChatStreamRequest struct {
    Message   string `json:"message" validate:"required"`
    SessionID string `json:"session_id"`
}

func (h *AgentHandler) ChatStream(c *fiber.Ctx) error {
    // Parse request
    var req ChatStreamRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "Invalid request")
    }

    agentID := c.Params("agentId")
    projectID := c.Locals("projectID").(string)

    // Set SSE headers
    c.Set("Content-Type", "text/event-stream")
    c.Set("Cache-Control", "no-cache")
    c.Set("Connection", "keep-alive")
    c.Set("X-Accel-Buffering", "no") // Disable nginx buffering

    // Stream response
    c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
        ctx, cancel := context.WithTimeout(c.Context(), 5*time.Minute)
        defer cancel()

        // Start chat stream from service
        eventChan, errChan := h.service.StreamChat(ctx, StreamChatInput{
            AgentID:   agentID,
            ProjectID: projectID,
            Message:   req.Message,
            SessionID: req.SessionID,
        })

        for {
            select {
            case event, ok := <-eventChan:
                if !ok {
                    return // Channel closed
                }
                writeSSEEvent(w, event)
                w.Flush()

            case err := <-errChan:
                writeSSEError(w, err)
                w.Flush()
                return

            case <-ctx.Done():
                writeSSEEvent(w, SSEEvent{
                    Event: "error",
                    Data:  `{"error": "timeout"}`,
                })
                w.Flush()
                return
            }
        }
    })

    return nil
}

// SSE Event structure
type SSEEvent struct {
    Event string `json:"event"`
    Data  string `json:"data"`
    ID    string `json:"id,omitempty"`
    Retry int    `json:"retry,omitempty"`
}

func writeSSEEvent(w *bufio.Writer, event SSEEvent) {
    if event.ID != "" {
        fmt.Fprintf(w, "id: %s\n", event.ID)
    }
    if event.Retry > 0 {
        fmt.Fprintf(w, "retry: %d\n", event.Retry)
    }
    fmt.Fprintf(w, "event: %s\n", event.Event)
    fmt.Fprintf(w, "data: %s\n\n", event.Data)
}

func writeSSEError(w *bufio.Writer, err error) {
    data, _ := json.Marshal(map[string]string{"error": err.Error()})
    writeSSEEvent(w, SSEEvent{
        Event: "error",
        Data:  string(data),
    })
}
```

## AG-UI Protocol Events

Based on `/spec/api/018-agui-protocol.md`:

```go
// internal/api/dto/agui_events.go
package dto

// AG-UI Event Types
const (
    EventRunStarted       = "run_started"
    EventRunFinished      = "run_finished"
    EventRunError         = "run_error"
    EventTextMessageStart = "text_message_start"
    EventTextMessageDelta = "text_message_content"
    EventTextMessageEnd   = "text_message_end"
    EventToolCallStart    = "tool_call_start"
    EventToolCallArgs     = "tool_call_args"
    EventToolCallEnd      = "tool_call_end"
    EventStateSnapshot    = "state_snapshot"
    EventStateDelta       = "state_delta"
)

type RunStartedEvent struct {
    ThreadID string `json:"thread_id"`
    RunID    string `json:"run_id"`
}

type TextMessageDeltaEvent struct {
    MessageID string `json:"message_id"`
    Delta     string `json:"delta"`
}

type ToolCallStartEvent struct {
    ToolCallID string `json:"tool_call_id"`
    ToolName   string `json:"tool_name"`
}

type ToolCallArgsEvent struct {
    ToolCallID string `json:"tool_call_id"`
    Delta      string `json:"delta"` // JSON args delta
}

type ToolCallEndEvent struct {
    ToolCallID string `json:"tool_call_id"`
    Result     any    `json:"result"`
}

type RunFinishedEvent struct {
    ThreadID string    `json:"thread_id"`
    RunID    string    `json:"run_id"`
    Usage    UsageInfo `json:"usage"`
}

type UsageInfo struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
}
```

## Streaming Service Integration

```go
// internal/domain/agent/stream.go
package agent

import (
    "context"
)

type StreamChatInput struct {
    AgentID   string
    ProjectID string
    Message   string
    SessionID string
}

func (s *Service) StreamChat(ctx context.Context, input StreamChatInput) (<-chan SSEEvent, <-chan error) {
    eventChan := make(chan SSEEvent, 100)
    errChan := make(chan error, 1)

    go func() {
        defer close(eventChan)
        defer close(errChan)

        // Send run started
        eventChan <- SSEEvent{
            Event: "run_started",
            Data:  fmt.Sprintf(`{"thread_id": "%s", "run_id": "%s"}`, 
                input.SessionID, uuid.New().String()),
        }

        // Connect to agent backend (Knative service)
        stream, err := s.agentClient.Chat(ctx, input)
        if err != nil {
            errChan <- err
            return
        }

        // Forward events
        for event := range stream {
            eventChan <- event
        }

        // Send run finished
        eventChan <- SSEEvent{
            Event: "run_finished",
            Data:  fmt.Sprintf(`{"usage": {"input_tokens": %d, "output_tokens": %d}}`,
                stream.Usage.Input, stream.Usage.Output),
        }
    }()

    return eventChan, errChan
}
```

## Client Reconnection Handling

```go
// Support Last-Event-ID for resumption
func (h *AgentHandler) ChatStream(c *fiber.Ctx) error {
    lastEventID := c.Get("Last-Event-ID")
    
    if lastEventID != "" {
        // Resume from last event
        // Implementation depends on event storage
    }
    
    // ... rest of handler
}
```

## Heartbeat for Connection Keepalive

```go
func streamWithHeartbeat(w *bufio.Writer, events <-chan SSEEvent) {
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case event, ok := <-events:
            if !ok {
                return
            }
            writeSSEEvent(w, event)
            w.Flush()

        case <-ticker.C:
            // Send comment as heartbeat
            fmt.Fprintf(w, ": heartbeat\n\n")
            w.Flush()
        }
    }
}
```

## Testing SSE Endpoints

```go
func TestChatStream(t *testing.T) {
    app := setupTestApp()

    req := httptest.NewRequest("POST", "/v1/agents/agt_123/chat/stream",
        strings.NewReader(`{"message": "Hello"}`))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "text/event-stream")

    resp, _ := app.Test(req, -1) // -1 = no timeout
    
    reader := bufio.NewReader(resp.Body)
    
    // Read first event
    line, _ := reader.ReadString('\n')
    assert.Contains(t, line, "event: run_started")
}
```

## Error Handling in Streams

```go
func writeStreamError(w *bufio.Writer, err error, status int) {
    errorEvent := SSEEvent{
        Event: "run_error",
        Data: fmt.Sprintf(`{"status": %d, "message": %q}`, 
            status, err.Error()),
    }
    writeSSEEvent(w, errorEvent)
}
```
