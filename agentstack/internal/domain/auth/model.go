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
