// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Logger returns a middleware that logs HTTP requests.
func Logger(log *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				auth := GetAuthFromContext(r.Context())
				fields := []zap.Field{
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", ww.Status()),
					zap.Int("bytes", ww.BytesWritten()),
					zap.Duration("duration", time.Since(start)),
					zap.String("request_id", middleware.GetReqID(r.Context())),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("user_agent", r.UserAgent()),
				}

				if auth != nil {
					if auth.TeamID != "" {
						fields = append(fields, zap.String("team_id", auth.TeamID))
					}
					if auth.ProjectID != "" {
						fields = append(fields, zap.String("project_id", auth.ProjectID))
					}
					if auth.UserID != "" {
						fields = append(fields, zap.String("user_id", auth.UserID))
					}
				}

				if ww.Status() >= 500 {
					log.Error("HTTP request", fields...)
				} else if ww.Status() >= 400 {
					log.Warn("HTTP request", fields...)
				} else {
					log.Info("HTTP request", fields...)
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
