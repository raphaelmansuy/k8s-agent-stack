// Package cache provides Redis caching and session management.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client with additional functionality.
type Client struct {
	rdb *redis.Client
}

// NewRedisClient creates a new Redis client from a URL.
func NewRedisClient(url string) (*Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Configure client options
	opts.PoolSize = 10
	opts.MinIdleConns = 2
	opts.MaxRetries = 3
	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second

	client := redis.NewClient(opts)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &Client{rdb: client}, nil
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	if c.rdb != nil {
		return c.rdb.Close()
	}
	return nil
}

// Get retrieves a value from cache.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if c.rdb == nil {
		return "", errors.New("redis client not initialized")
	}
	return c.rdb.Get(ctx, key).Result()
}

// Set stores a value in cache with expiration.
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Delete removes a value from cache.
func (c *Client) Delete(ctx context.Context, key string) error {
	if c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.Del(ctx, key).Err()
}

// GetJSON retrieves and unmarshals a JSON value.
func (c *Client) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// SetJSON marshals and stores a JSON value.
func (c *Client) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return c.Set(ctx, key, data, expiration)
}

// RateLimit checks and updates rate limit for a key.
func (c *Client) RateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	if c.rdb == nil {
		return true, limit, nil // Allow if Redis is not available
	}

	pipe := c.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, limit, err // Allow on error
	}

	count := int(incr.Val())
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	return count <= limit, remaining, nil
}

// SetSession stores a session with expiration.
func (c *Client) SetSession(ctx context.Context, sessionID string, data map[string]interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.SetJSON(ctx, key, data, expiration)
}

// GetSession retrieves a session.
func (c *Client) GetSession(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	var data map[string]interface{}
	err := c.GetJSON(ctx, key, &data)
	return data, err
}

// DeleteSession removes a session.
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.Delete(ctx, key)
}

// PublishMessage publishes a message to a Redis channel.
func (c *Client) PublishMessage(ctx context.Context, channel string, message interface{}) error {
	if c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	return c.rdb.Publish(ctx, channel, data).Err()
}

// Subscribe creates a subscription to Redis channels.
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	if c.rdb == nil {
		return nil
	}
	return c.rdb.Subscribe(ctx, channels...)
}

// HealthCheck verifies Redis connection is healthy.
func (c *Client) HealthCheck(ctx context.Context) error {
	if c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.Ping(ctx).Err()
}

// CheckIdempotency checks if an idempotency key has been used.
func (c *Client) CheckIdempotency(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	if c.rdb == nil {
		return false, nil // Skip if Redis is not available
	}

	idemKey := fmt.Sprintf("idem:%s", key)
	result, err := c.rdb.SetNX(ctx, idemKey, "1", expiration).Result()
	if err != nil {
		return false, err
	}

	return !result, nil // Returns true if key already exists (duplicate request)
}
