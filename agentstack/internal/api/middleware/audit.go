// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/raphaelmansuy/agentstack/internal/domain/audit"
)

// AuditMiddleware provides audit logging middleware.
type AuditMiddleware struct {
	auditSvc *audit.Service
}

// NewAuditMiddleware creates a new audit middleware.
func NewAuditMiddleware(auditSvc *audit.Service) *AuditMiddleware {
	return &AuditMiddleware{auditSvc: auditSvc}
}

// RequestLogger creates middleware that logs HTTP requests.
func (m *AuditMiddleware) RequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := uuid.New().String()

			// Add request ID to context
			ctx := SetRequestIDInContext(r.Context(), requestID)
			r = r.WithContext(ctx)

			// Add request ID to response headers
			w.Header().Set("X-Request-ID", requestID)

			// Wrap response writer to capture status code and size
			wrapped := &responseCapture{ResponseWriter: w, status: http.StatusOK}

			// Call the next handler
			next.ServeHTTP(wrapped, r)

			// Log the request after completion
			duration := time.Since(start)
			auth := GetAuthFromContext(r.Context())

			actorType := audit.ActorSystem
			actorID := "anonymous"
			actorEmail := ""
			teamID := ""
			if auth != nil {
				actorType = audit.ActorUser
				actorID = auth.UserID
				actorEmail = auth.Email
				teamID = auth.TeamID
			}

			details := map[string]any{
				"method":      r.Method,
				"path":        r.URL.Path,
				"query":       r.URL.RawQuery,
				"status":      wrapped.status,
				"size":        wrapped.size,
				"duration_ms": duration.Milliseconds(),
				"user_agent":  r.UserAgent(),
				"remote_addr": getClientIP(r),
			}

			result := audit.ResultSuccess
			if wrapped.status >= 400 {
				result = audit.ResultFailure
			}

			_ = m.auditSvc.LogAction(r.Context(), audit.LogParams{
				Type:       audit.EventAgentInvoked, // Generic API request type
				TeamID:     teamID,
				ActorID:    actorID,
				ActorType:  actorType,
				ActorEmail: actorEmail,
				Resource:   "http_request",
				ResourceID: requestID,
				Action:     r.Method,
				Result:     result,
				IPAddress:  getClientIP(r),
				UserAgent:  r.UserAgent(),
				RequestID:  requestID,
				Details:    details,
			})
		})
	}
}

// LogAction creates middleware that logs a specific action.
func (m *AuditMiddleware) LogAction(eventType audit.EventType, resourceType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap to capture status
			wrapped := &responseCapture{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			// Log after completion
			auth := GetAuthFromContext(r.Context())
			if auth == nil {
				return
			}

			resourceID := r.PathValue("id")
			if resourceID == "" {
				resourceID = GetRequestIDFromContext(r.Context())
			}

			result := audit.ResultSuccess
			if wrapped.status >= 400 {
				result = audit.ResultFailure
			}

			details := map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
				"status": wrapped.status,
			}

			// Get project ID if available
			projectID := auth.ProjectID
			if projectID == "" {
				projectID = r.PathValue("projectId")
			}
			if projectID == "" {
				projectID = r.URL.Query().Get("project_id")
			}

			_ = m.auditSvc.LogAction(r.Context(), audit.LogParams{
				Type:       eventType,
				TeamID:     auth.TeamID,
				ProjectID:  projectID,
				ActorID:    auth.UserID,
				ActorType:  audit.ActorUser,
				ActorEmail: auth.Email,
				Resource:   resourceType,
				ResourceID: resourceID,
				Action:     r.Method,
				Result:     result,
				IPAddress:  getClientIP(r),
				UserAgent:  r.UserAgent(),
				RequestID:  GetRequestIDFromContext(r.Context()),
				Details:    details,
			})
		})
	}
}

// responseCapture wraps ResponseWriter to capture status code and response size.
type responseCapture struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *responseCapture) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseCapture) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// Remove port if present
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
