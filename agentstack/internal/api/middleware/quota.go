// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"net/http"
	"strconv"

	"github.com/raphaelmansuy/agentstack/internal/domain/quota"
)

// QuotaMiddleware provides quota checking middleware.
type QuotaMiddleware struct {
	quotaSvc *quota.Service
}

// NewQuotaMiddleware creates a new quota middleware.
func NewQuotaMiddleware(quotaSvc *quota.Service) *QuotaMiddleware {
	return &QuotaMiddleware{quotaSvc: quotaSvc}
}

// CheckQuota creates middleware that checks if a quota allows the operation.
func (m *QuotaMiddleware) CheckQuota(quotaType quota.QuotaType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := GetAuthFromContext(r.Context())
			if auth == nil {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			// Check quota for the team (or user if no team)
			teamID := auth.TeamID
			if teamID == "" {
				teamID = auth.UserID
			}

			result, err := m.quotaSvc.CheckQuota(r.Context(), quota.CheckQuotaRequest{
				TeamID:    teamID,
				ProjectID: auth.ProjectID,
				Type:      quotaType,
				Amount:    1,
			})
			if err != nil {
				http.Error(w, "quota check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				w.Header().Set("X-Quota-Limit", formatInt64(result.Limit))
				w.Header().Set("X-Quota-Remaining", formatInt64(result.Remaining))
				http.Error(w, "quota exceeded", http.StatusTooManyRequests)
				return
			}

			// Add quota headers for successful requests too
			w.Header().Set("X-Quota-Limit", formatInt64(result.Limit))
			w.Header().Set("X-Quota-Remaining", formatInt64(result.Remaining))

			next.ServeHTTP(w, r)
		})
	}
}

// IncrementAfter creates middleware that increments usage after a successful request.
func (m *QuotaMiddleware) IncrementAfter(quotaType quota.QuotaType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap response writer to capture status code
			wrapped := &statusCapture{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			// Only increment on success (2xx status codes)
			if wrapped.status >= 200 && wrapped.status < 300 {
				auth := GetAuthFromContext(r.Context())
				if auth != nil {
					teamID := auth.TeamID
					if teamID == "" {
						teamID = auth.UserID
					}
					// Best effort increment - don't fail the request if this fails
					_ = m.quotaSvc.IncrementUsage(r.Context(), quota.IncrementUsageRequest{
						TeamID:    teamID,
						ProjectID: auth.ProjectID,
						Type:      quotaType,
						Amount:    1,
					})
				}
			}
		})
	}
}

// RateLimiter creates middleware that enforces rate limits.
func (m *QuotaMiddleware) RateLimiter(quotaType quota.QuotaType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := GetAuthFromContext(r.Context())
			if auth == nil {
				// For unauthenticated requests, use IP-based limiting
				// This is a simplified version - in production, use a more robust approach
				http.Error(w, "authentication required for rate limiting", http.StatusUnauthorized)
				return
			}

			teamID := auth.TeamID
			if teamID == "" {
				teamID = auth.UserID
			}

			// Check and increment in one call for rate limiting
			result, err := m.quotaSvc.CheckQuota(r.Context(), quota.CheckQuotaRequest{
				TeamID:    teamID,
				ProjectID: auth.ProjectID,
				Type:      quotaType,
				Amount:    1,
			})
			if err != nil {
				http.Error(w, "rate limit check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				w.Header().Set("X-RateLimit-Limit", formatInt64(result.Limit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Increment the rate limit counter
			_ = m.quotaSvc.IncrementUsage(r.Context(), quota.IncrementUsageRequest{
				TeamID:    teamID,
				ProjectID: auth.ProjectID,
				Type:      quotaType,
				Amount:    1,
			})

			w.Header().Set("X-RateLimit-Limit", formatInt64(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", formatInt64(result.Remaining-1))

			next.ServeHTTP(w, r)
		})
	}
}

// statusCapture wraps ResponseWriter to capture the status code.
type statusCapture struct {
	http.ResponseWriter
	status int
}

func (s *statusCapture) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}
