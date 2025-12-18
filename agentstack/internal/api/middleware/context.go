// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"context"

	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

// Context keys for storing values in request context.
// Note: contextKey is defined in safety.go
const (
	// ContextKeyAuth is the key for authentication info in context.
	ContextKeyAuth contextKey = "auth"
	// ContextKeyRBACResult is the key for RBAC result in context.
	ContextKeyRBACResult contextKey = "rbac_result"
	// ContextKeyResourceOwner is the key for resource owner ID in context.
	ContextKeyResourceOwner contextKey = "resource_owner"
	// ContextKeyRequestID is the key for request ID in context.
	ContextKeyRequestID contextKey = "request_id"
)

// AuthInfo contains authentication information for a request.
type AuthInfo struct {
	UserID    string
	TeamID    string
	ProjectID string
	Email     string
	Name      string
	IsAdmin   bool
}

// SetAuthInContext adds authentication info to the context.
func SetAuthInContext(ctx context.Context, auth *AuthInfo) context.Context {
	return context.WithValue(ctx, ContextKeyAuth, auth)
}

// GetAuthFromContext retrieves authentication info from the context.
func GetAuthFromContext(ctx context.Context) *AuthInfo {
	if auth, ok := ctx.Value(ContextKeyAuth).(*AuthInfo); ok {
		return auth
	}
	return nil
}

// SetRBACResultInContext adds the RBAC result to the context.
func SetRBACResultInContext(ctx context.Context, result *rbac.PermissionResult) context.Context {
	return context.WithValue(ctx, ContextKeyRBACResult, result)
}

// GetRBACResultFromContext retrieves the RBAC result from the context.
func GetRBACResultFromContext(ctx context.Context) *rbac.PermissionResult {
	if result, ok := ctx.Value(ContextKeyRBACResult).(*rbac.PermissionResult); ok {
		return result
	}
	return nil
}

// SetResourceOwnerInContext adds the resource owner ID to the context.
func SetResourceOwnerInContext(ctx context.Context, ownerID string) context.Context {
	return context.WithValue(ctx, ContextKeyResourceOwner, ownerID)
}

// GetResourceOwnerFromContext retrieves the resource owner ID from the context.
func GetResourceOwnerFromContext(ctx context.Context) string {
	if ownerID, ok := ctx.Value(ContextKeyResourceOwner).(string); ok {
		return ownerID
	}
	return ""
}

// SetRequestIDInContext adds the request ID to the context.
func SetRequestIDInContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

// GetRequestIDFromContext retrieves the request ID from the context.
func GetRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return requestID
	}
	return ""
}
