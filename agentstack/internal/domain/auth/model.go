package auth

import (
	"time"
)

// APIKey represents an API key for authentication.
type APIKey struct {
	ID         string     `json:"id"`
	TeamID     string     `json:"team_id"`
	ProjectID  string     `json:"project_id,omitempty"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"` // Never expose the hash
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// APIKeyCreate represents the input for creating an API key.
type APIKeyCreate struct {
	TeamID    string     `json:"team_id"`
	ProjectID string     `json:"project_id,omitempty"`
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// APIKeyGenerated represents a newly created API key with the raw key.
type APIKeyGenerated struct {
	APIKey
	RawKey string `json:"api_key"`
}
