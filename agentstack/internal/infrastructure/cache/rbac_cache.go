package cache

import (
	"context"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/rbac"
)

type RBACCache struct {
	client *Client
}

func NewRBACCache(client *Client) *RBACCache {
	return &RBACCache{
		client: client,
	}
}

func (c *RBACCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key)
}

func (c *RBACCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration)
}

func (c *RBACCache) Del(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := c.client.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (c *RBACCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	// Note: In a real production environment, KEYS should be used with caution.
	// SCAN is preferred. For now, we'll use the underlying redis client if needed.
	return nil, nil // TODO: Implement if needed
}

// Ensure RBACCache implements rbac.Cache
var _ rbac.Cache = (*RBACCache)(nil)
