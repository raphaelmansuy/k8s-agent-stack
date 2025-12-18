package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Repository defines the interface for API key data access.
type Repository interface {
	CreateAPIKey(ctx context.Context, key *APIKey) error
	GetAPIKey(ctx context.Context, id string) (*APIKey, error)
	GetAPIKeyByHash(ctx context.Context, hash string) (*APIKey, error)
	ListAPIKeys(ctx context.Context, teamID string) ([]*APIKey, error)
	DeleteAPIKey(ctx context.Context, id string) error
	UpdateLastUsed(ctx context.Context, id string) error
}

// Service provides API key management functionality.
type Service struct {
	repo Repository
}

// NewService creates a new API key service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateKey generates and stores a new API key.
func (s *Service) CreateKey(ctx context.Context, input *APIKeyCreate) (*APIKeyGenerated, error) {
	// Generate raw key: sk_live_<32 chars>
	rawKey, err := s.generateRandomKey(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	
	fullKey := fmt.Sprintf("sk_live_%s", rawKey)
	prefix := fullKey[:8] // sk_live_
	hash := s.HashKey(fullKey)

	key := &APIKey{
		TeamID:    input.TeamID,
		ProjectID: input.ProjectID,
		Name:      input.Name,
		KeyHash:   hash,
		KeyPrefix: prefix,
		Scopes:    input.Scopes,
		ExpiresAt: input.ExpiresAt,
	}

	if err := s.repo.CreateAPIKey(ctx, key); err != nil {
		return nil, err
	}

	return &APIKeyGenerated{
		APIKey: *key,
		RawKey: fullKey,
	}, nil
}

// HashKey returns a SHA-256 hash of the key.
func (s *Service) HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func (s *Service) generateRandomKey(length int) (string, error) {
	b := make([]byte, length/2)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) ListKeys(ctx context.Context, teamID string) ([]*APIKey, error) {
	return s.repo.ListAPIKeys(ctx, teamID)
}

func (s *Service) DeleteKey(ctx context.Context, id string) error {
	return s.repo.DeleteAPIKey(ctx, id)
}

func (s *Service) GetKeyByHash(ctx context.Context, hash string) (*APIKey, error) {
	return s.repo.GetAPIKeyByHash(ctx, hash)
}
