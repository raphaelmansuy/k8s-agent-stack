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

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/raphaelmansuy/agentstack/internal/domain/safety"
)

// SafetyMiddleware provides HTTP middleware for content safety checking.
type SafetyMiddleware struct {
	safetySvc SafetyService
}

// SafetyService defines the interface for safety checking.
type SafetyService interface {
	CheckInput(ctx context.Context, input string) (*safety.CheckResult, error)
	CheckOutput(ctx context.Context, output string) (*safety.CheckResult, error)
	IsEnabled() bool
}

// NewSafetyMiddleware creates a new safety middleware.
func NewSafetyMiddleware(safetySvc SafetyService) *SafetyMiddleware {
	return &SafetyMiddleware{safetySvc: safetySvc}
}

// PreExecutionHandler returns an http.Handler that checks input before agent execution.
func (m *SafetyMiddleware) PreExecutionHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.safetySvc.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, `{"error":"failed to read request body"}`, http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		// Extract text content from body
		input := extractTextFromBody(body)
		if input == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check input safety
		result, err := m.safetySvc.CheckInput(r.Context(), input)
		if err != nil {
			// Log error but continue (fail open)
			next.ServeHTTP(w, r)
			return
		}

		if !result.Allowed {
			// Get the reason from the last failed check
			reason := "Content blocked by safety filter"
			for _, check := range result.Checks {
				if !check.Passed {
					reason = check.Reason
					break
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			resp := SafetyErrorResponse{
				Error:  "content_blocked",
				Reason: reason,
				Checks: result.Checks,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Store result for potential post-processing
		ctx := context.WithValue(r.Context(), safetyPreCheckKey, result)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// SafetyErrorResponse is the response when content is blocked.
type SafetyErrorResponse struct {
	Error  string         `json:"error"`
	Reason string         `json:"reason"`
	Checks []safety.Check `json:"checks,omitempty"`
}

type contextKey string

const safetyPreCheckKey contextKey = "safety_pre_check"

// GetSafetyPreCheck retrieves the safety pre-check result from context.
func GetSafetyPreCheck(ctx context.Context) *safety.CheckResult {
	if result, ok := ctx.Value(safetyPreCheckKey).(*safety.CheckResult); ok {
		return result
	}
	return nil
}

// extractTextFromBody attempts to extract text content from a JSON body.
func extractTextFromBody(body []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}

	// Try common field names for message content
	textFields := []string{"message", "content", "text", "input", "prompt", "query"}

	for _, field := range textFields {
		if val, ok := data[field]; ok {
			switch v := val.(type) {
			case string:
				return v
			case map[string]interface{}:
				// Nested content
				if text, ok := v["text"].(string); ok {
					return text
				}
				if content, ok := v["content"].(string); ok {
					return content
				}
			}
		}
	}

	// Try to find text in message parts (A2A format)
	if msg, ok := data["message"].(map[string]interface{}); ok {
		if parts, ok := msg["parts"].([]interface{}); ok {
			var texts []string
			for _, part := range parts {
				if p, ok := part.(map[string]interface{}); ok {
					if text, ok := p["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
			if len(texts) > 0 {
				return texts[0] // Return first text part
			}
		}
	}

	return ""
}

// SafetyResponseWriter wraps http.ResponseWriter to intercept and check output.
type SafetyResponseWriter struct {
	http.ResponseWriter
	safetySvc   SafetyService
	ctx         context.Context
	statusCode  int
	buffer      *bytes.Buffer
	wroteHeader bool
}

// NewSafetyResponseWriter creates a new safety response writer.
func NewSafetyResponseWriter(ctx context.Context, w http.ResponseWriter, safetySvc SafetyService) *SafetyResponseWriter {
	return &SafetyResponseWriter{
		ResponseWriter: w,
		safetySvc:      safetySvc,
		ctx:            ctx,
		buffer:         &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code.
func (w *SafetyResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.wroteHeader = true
}

// Write buffers the response for safety checking.
func (w *SafetyResponseWriter) Write(b []byte) (int, error) {
	return w.buffer.Write(b)
}

// Flush performs safety check and writes the response.
func (w *SafetyResponseWriter) Flush() error {
	body := w.buffer.Bytes()

	// Try to extract text content
	output := extractTextFromBody(body)

	if output != "" && w.safetySvc.IsEnabled() {
		result, err := w.safetySvc.CheckOutput(w.ctx, output)
		if err == nil && !result.Allowed {
			// Block the response
			w.ResponseWriter.Header().Set("Content-Type", "application/json")
			w.ResponseWriter.WriteHeader(http.StatusBadRequest)
			resp := SafetyErrorResponse{
				Error:  "output_blocked",
				Reason: "Response blocked by safety filter",
				Checks: result.Checks,
			}
			return json.NewEncoder(w.ResponseWriter).Encode(resp)
		}
	}

	// Write the original response
	if w.wroteHeader {
		w.ResponseWriter.WriteHeader(w.statusCode)
	}
	_, err := w.ResponseWriter.Write(body)
	return err
}
