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

package sdk

import (
	"fmt"
	"testing"
)

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name: "with code",
			err: &APIError{
				StatusCode: 404,
				Code:       "not_found",
				Message:    "Resource not found",
			},
			expected: "api error [404] not_found: Resource not found",
		},
		{
			name: "without code",
			err: &APIError{
				StatusCode: 500,
				Message:    "Internal server error",
			},
			expected: "api error [500]: Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAPIErrorStatusChecks(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		isNotFound     bool
		isUnauthorized bool
		isForbidden    bool
		isConflict     bool
		isRateLimited  bool
		isServerError  bool
	}{
		{
			name:       "404 not found",
			statusCode: 404,
			isNotFound: true,
		},
		{
			name:           "401 unauthorized",
			statusCode:     401,
			isUnauthorized: true,
		},
		{
			name:        "403 forbidden",
			statusCode:  403,
			isForbidden: true,
		},
		{
			name:       "409 conflict",
			statusCode: 409,
			isConflict: true,
		},
		{
			name:          "429 rate limited",
			statusCode:    429,
			isRateLimited: true,
		},
		{
			name:          "500 server error",
			statusCode:    500,
			isServerError: true,
		},
		{
			name:          "502 server error",
			statusCode:    502,
			isServerError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}

			if err.IsNotFound() != tt.isNotFound {
				t.Errorf("IsNotFound: expected %v, got %v", tt.isNotFound, err.IsNotFound())
			}
			if err.IsUnauthorized() != tt.isUnauthorized {
				t.Errorf("IsUnauthorized: expected %v, got %v", tt.isUnauthorized, err.IsUnauthorized())
			}
			if err.IsForbidden() != tt.isForbidden {
				t.Errorf("IsForbidden: expected %v, got %v", tt.isForbidden, err.IsForbidden())
			}
			if err.IsConflict() != tt.isConflict {
				t.Errorf("IsConflict: expected %v, got %v", tt.isConflict, err.IsConflict())
			}
			if err.IsRateLimited() != tt.isRateLimited {
				t.Errorf("IsRateLimited: expected %v, got %v", tt.isRateLimited, err.IsRateLimited())
			}
			if err.IsServerError() != tt.isServerError {
				t.Errorf("IsServerError: expected %v, got %v", tt.isServerError, err.IsServerError())
			}
		})
	}
}

func TestErrorHelperFunctions(t *testing.T) {
	notFoundErr := &APIError{StatusCode: 404}
	unauthorizedErr := &APIError{StatusCode: 401}
	forbiddenErr := &APIError{StatusCode: 403}
	conflictErr := &APIError{StatusCode: 409}
	rateLimitErr := &APIError{StatusCode: 429}
	serverErr := &APIError{StatusCode: 500}
	genericErr := fmt.Errorf("generic error")

	if !IsNotFoundError(notFoundErr) {
		t.Error("expected IsNotFoundError to return true for 404")
	}
	if IsNotFoundError(genericErr) {
		t.Error("expected IsNotFoundError to return false for generic error")
	}

	if !IsUnauthorizedError(unauthorizedErr) {
		t.Error("expected IsUnauthorizedError to return true for 401")
	}
	if IsUnauthorizedError(genericErr) {
		t.Error("expected IsUnauthorizedError to return false for generic error")
	}

	if !IsForbiddenError(forbiddenErr) {
		t.Error("expected IsForbiddenError to return true for 403")
	}

	if !IsConflictError(conflictErr) {
		t.Error("expected IsConflictError to return true for 409")
	}

	if !IsRateLimitedError(rateLimitErr) {
		t.Error("expected IsRateLimitedError to return true for 429")
	}

	if !IsServerErrorType(serverErr) {
		t.Error("expected IsServerErrorType to return true for 500")
	}
}
