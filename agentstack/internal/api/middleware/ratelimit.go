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
	"fmt"
	"net/http"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
)

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Requests int           // Max requests per window
	Window   time.Duration // Time window
	KeyFunc  func(r *http.Request) string
}

// RateLimit returns a middleware that enforces rate limiting.
func RateLimit(redis *cache.Client, config RateLimitConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if redis == nil {
				// Skip rate limiting if Redis is not available
				next.ServeHTTP(w, r)
				return
			}

			key := config.KeyFunc(r)
			if key == "" {
				// Skip if no key can be determined
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			rateLimitKey := fmt.Sprintf("ratelimit:%s", key)
			allowed, remaining, err := redis.RateLimit(ctx, rateLimitKey, config.Requests, config.Window)
			if err != nil {
				// Allow request if rate limiting fails
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.Requests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(config.Window).Unix()))

			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(config.Window.Seconds())))
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultRateLimitKeyFunc returns the client IP as the rate limit key.
func DefaultRateLimitKeyFunc(r *http.Request) string {
	// Use X-Forwarded-For if available (behind proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}

// TenantRateLimitKeyFunc returns the team ID as the rate limit key.
func TenantRateLimitKeyFunc(r *http.Request) string {
	if teamID := GetTeamID(r.Context()); teamID != "" {
		return teamID
	}
	return DefaultRateLimitKeyFunc(r)
}

// Idempotency returns a middleware that handles idempotent requests.
func Idempotency(redis *cache.Client) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if redis == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Only apply to POST/PUT/PATCH
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			idemKey := r.Header.Get("Idempotency-Key")
			if idemKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			// Check for existing response (idempotency key already used)
			isDuplicate, err := redis.CheckIdempotency(ctx, idemKey, 24*time.Hour)
			if err == nil && isDuplicate {
				// Return a 409 Conflict for duplicate requests
				// In a real implementation, you'd store and return the original response
				w.Header().Set("Idempotent-Replayed", "true")
				http.Error(w, "Request already processed", http.StatusConflict)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TenantContext middleware injects tenant context for RLS.
func TenantContext() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get tenant from auth context or header (for internal services)
			teamID := GetTeamID(r.Context())
			if teamID == "" {
				teamID = r.Header.Get("X-Tenant-ID")
			}

			if teamID != "" {
				// No need to set ContextKeyTeamID anymore as GetTeamID uses AuthInfo
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
