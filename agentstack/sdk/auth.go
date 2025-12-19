// Package sdk provides the AgentStack Go SDK for programmatic access to the API.
package sdk

import (
	"context"
)

// AuthService handles authentication operations.
type AuthService struct {
	client *Client
}

// Login authenticates a user and returns an auth token.
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*AuthToken, error) {
	var token AuthToken
	if err := s.client.Post(ctx, "/api/v1/auth/login", req, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// RefreshToken refreshes an access token.
func (s *AuthService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*AuthToken, error) {
	var token AuthToken
	if err := s.client.Post(ctx, "/api/v1/auth/refresh", req, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// Logout logs out the current user.
func (s *AuthService) Logout(ctx context.Context) error {
	return s.client.Post(ctx, "/api/v1/auth/logout", nil, nil)
}

// CurrentUser retrieves the current authenticated user.
func (s *AuthService) CurrentUser(ctx context.Context) (*User, error) {
	var user User
	if err := s.client.Get(ctx, "/api/v1/auth/me", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateAPIKey creates a new API key.
func (s *AuthService) CreateAPIKey(ctx context.Context, req *APIKeyRequest) (*APIKey, error) {
	var apiKey APIKey
	if err := s.client.Post(ctx, "/api/v1/api-keys", req, &apiKey); err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// ListAPIKeys lists all API keys for the current user.
func (s *AuthService) ListAPIKeys(ctx context.Context) (*ListAPIKeysResponse, error) {
	var response ListAPIKeysResponse
	if err := s.client.Get(ctx, "/api/v1/api-keys", &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// RevokeAPIKey revokes an API key by ID.
func (s *AuthService) RevokeAPIKey(ctx context.Context, id string) error {
	return s.client.Delete(ctx, "/api/v1/api-keys/"+id)
}

// RotateAPIKey rotates an API key by ID.
func (s *AuthService) RotateAPIKey(ctx context.Context, id string) (*APIKey, error) {
	var apiKey APIKey
	if err := s.client.Post(ctx, "/api/v1/api-keys/"+id+"/rotate", nil, &apiKey); err != nil {
		return nil, err
	}
	return &apiKey, nil
}
