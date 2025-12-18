package cache

import (
	"context"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/quota"
)

type QuotaCache struct {
	client *Client
}

func NewQuotaCache(client *Client) *QuotaCache {
	return &QuotaCache{
		client: client,
	}
}

func (c *QuotaCache) Get(ctx context.Context, key string) (int64, error) {
	return c.client.GetInt64(ctx, key)
}

func (c *QuotaCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, key, value)
}

func (c *QuotaCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration)
}

func (c *QuotaCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key)
}

// Ensure QuotaCache implements quota.Cache
var _ quota.Cache = (*QuotaCache)(nil)
