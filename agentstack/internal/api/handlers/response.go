// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Details any    `json:"details,omitempty"`
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, message string, err error) {
	resp := ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	}
	if err != nil {
		resp.Details = err.Error()
	}
	writeJSON(w, status, resp)
}
