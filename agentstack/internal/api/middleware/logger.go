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

// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"context"
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

			defer func(ctx context.Context) {
				auth := GetAuthFromContext(ctx)
				fields := []zap.Field{
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", ww.Status()),
					zap.Int("bytes", ww.BytesWritten()),
					zap.Duration("duration", time.Since(start)),
					zap.String("request_id", middleware.GetReqID(ctx)),
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
			}(r.Context())

			next.ServeHTTP(ww, r)
		})
	}
}
