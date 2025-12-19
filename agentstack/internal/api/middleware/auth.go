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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	JWTSecret    string
	APIKeyLookup func(ctx context.Context, keyHash string) (*APIKeyInfo, error)
}

// APIKeyInfo represents API key metadata.
type APIKeyInfo struct {
	TeamID    string
	ProjectID string
	Scopes    []string
}

// Auth returns a middleware that validates authentication.
func Auth(config AuthConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health checks, metrics, and documentation
			path := r.URL.Path
			if path == "/health" || path == "/health/detailed" || path == "/healthz" || path == "/readyz" || path == "/livez" || path == "/metrics" ||
				strings.HasPrefix(path, "/api/docs") || strings.HasPrefix(path, "/api/openapi.") ||
				strings.HasPrefix(path, "/docs") || strings.HasPrefix(path, "/openapi.") {
				next.ServeHTTP(w, r)
				return
			}

			// Extract authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			var ctx context.Context

			// Handle Bearer token (JWT)
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				claims, err := validateJWT(token, config.JWTSecret)
				if err != nil {
					http.Error(w, "Invalid token", http.StatusUnauthorized)
					return
				}
				ctx = setClaimsToContext(r.Context(), claims)
			} else if strings.HasPrefix(authHeader, "ApiKey ") {
				// Handle API Key
				apiKey := strings.TrimPrefix(authHeader, "ApiKey ")
				keyInfo, err := config.APIKeyLookup(r.Context(), hashAPIKey(apiKey))
				if err != nil {
					http.Error(w, "Invalid API key", http.StatusUnauthorized)
					return
				}
				ctx = setAPIKeyInfoToContext(r.Context(), keyInfo)
			} else {
				http.Error(w, "Invalid Authorization format", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Claims represents JWT claims.
type Claims struct {
	TeamID    string   `json:"team_id"`
	ProjectID string   `json:"project_id"`
	UserID    string   `json:"user_id"`
	Scopes    []string `json:"scopes"`
	jwt.RegisteredClaims
}

func validateJWT(tokenString, secret string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		fmt.Printf("DEBUG: JWT validation error: %v\n", err)
		return nil, err
	}
	return claims, nil
}

func setClaimsToContext(ctx context.Context, claims *Claims) context.Context {
	auth := &AuthInfo{
		UserID:    claims.UserID,
		TeamID:    claims.TeamID,
		ProjectID: claims.ProjectID,
		Scopes:    claims.Scopes,
	}
	return SetAuthInContext(ctx, auth)
}

func setAPIKeyInfoToContext(ctx context.Context, info *APIKeyInfo) context.Context {
	auth := &AuthInfo{
		TeamID:    info.TeamID,
		ProjectID: info.ProjectID,
		Scopes:    info.Scopes,
	}
	return SetAuthInContext(ctx, auth)
}

func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// GetTeamID extracts team ID from context.
func GetTeamID(ctx context.Context) string {
	if auth := GetAuthFromContext(ctx); auth != nil {
		return auth.TeamID
	}
	return ""
}

// GetProjectID extracts project ID from context.
func GetProjectID(ctx context.Context) string {
	if auth := GetAuthFromContext(ctx); auth != nil {
		return auth.ProjectID
	}
	return ""
}

// GetUserID extracts user ID from context.
func GetUserID(ctx context.Context) string {
	if auth := GetAuthFromContext(ctx); auth != nil {
		return auth.UserID
	}
	return ""
}

// GetScopes extracts scopes from context.
func GetScopes(ctx context.Context) []string {
	if auth := GetAuthFromContext(ctx); auth != nil {
		return auth.Scopes
	}
	return nil
}

// HasScope checks if the context has a specific scope.
func HasScope(ctx context.Context, resource, action string) bool {
	scopes := GetScopes(ctx)
	if len(scopes) == 0 {
		return false
	}

	required := resource + ":" + action
	resourceWildcard := resource + ":*"

	for _, s := range scopes {
		if s == "*" || s == required || s == resourceWildcard {
			return true
		}
		// Handle "manage" action as a wildcard for all actions on a resource
		if action != "manage" && s == resource+":manage" {
			return true
		}
	}
	return false
}
