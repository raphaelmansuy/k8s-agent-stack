// Package sdk provides the AgentStack Go SDK for programmatic access to the API.
package sdk

import (
	"fmt"
)

// APIError represents an API error.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]interface{}
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("api error [%d] %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("api error [%d]: %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error is a not found error.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsUnauthorized returns true if the error is an unauthorized error.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsForbidden returns true if the error is a forbidden error.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == 403
}

// IsConflict returns true if the error is a conflict error.
func (e *APIError) IsConflict() bool {
	return e.StatusCode == 409
}

// IsRateLimited returns true if the error is a rate limit error.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}

// IsServerError returns true if the error is a server error.
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500
}

// IsNotFoundError checks if an error is a not found error.
func IsNotFoundError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsNotFound()
	}
	return false
}

// IsUnauthorizedError checks if an error is an unauthorized error.
func IsUnauthorizedError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsUnauthorized()
	}
	return false
}

// IsForbiddenError checks if an error is a forbidden error.
func IsForbiddenError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsForbidden()
	}
	return false
}

// IsConflictError checks if an error is a conflict error.
func IsConflictError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsConflict()
	}
	return false
}

// IsRateLimitedError checks if an error is a rate limit error.
func IsRateLimitedError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsRateLimited()
	}
	return false
}

// IsServerErrorType checks if an error is a server error.
func IsServerErrorType(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsServerError()
	}
	return false
}
