// Package cache provides caching strategies for production use.
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

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheAside implements the cache-aside (lazy-loading) pattern.
type CacheAside[T any] struct {
	client     *redis.Client
	prefix     string
	defaultTTL time.Duration
	marshaler  Marshaler[T]
}

// Marshaler handles serialization/deserialization.
type Marshaler[T any] interface {
	Marshal(v T) ([]byte, error)
	Unmarshal(data []byte, v *T) error
}

// JSONMarshaler uses JSON for serialization.
type JSONMarshaler[T any] struct{}

func (j JSONMarshaler[T]) Marshal(v T) ([]byte, error) {
	return json.Marshal(v)
}

func (j JSONMarshaler[T]) Unmarshal(data []byte, v *T) error {
	return json.Unmarshal(data, v)
}

// NewCacheAside creates a new cache-aside instance.
func NewCacheAside[T any](client *redis.Client, prefix string, ttl time.Duration) *CacheAside[T] {
	return &CacheAside[T]{
		client:     client,
		prefix:     prefix,
		defaultTTL: ttl,
		marshaler:  JSONMarshaler[T]{},
	}
}

// Get retrieves from cache or calls loader on cache miss.
func (c *CacheAside[T]) Get(ctx context.Context, key string, loader func() (T, error)) (T, error) {
	var result T
	fullKey := c.prefix + key

	// Try cache first
	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err == nil {
		err = c.marshaler.Unmarshal(data, &result)
		if err == nil {
			return result, nil
		}
	}

	// Cache miss - load from source
	result, err = loader()
	if err != nil {
		return result, err
	}

	// Store in cache asynchronously
	detachedCtx := context.WithoutCancel(ctx)
	go func(ctx context.Context) {
		data, err := c.marshaler.Marshal(result)
		if err != nil {
			return
		}
		_ = c.client.SetEx(ctx, fullKey, data, c.defaultTTL)
	}(detachedCtx)

	return result, nil
}

// GetWithTTL retrieves with custom TTL.
func (c *CacheAside[T]) GetWithTTL(ctx context.Context, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	var result T
	fullKey := c.prefix + key

	// Try cache first
	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err == nil {
		err = c.marshaler.Unmarshal(data, &result)
		if err == nil {
			return result, nil
		}
	}

	// Cache miss - load from source
	result, err = loader()
	if err != nil {
		return result, err
	}

	// Store in cache
	data, err = c.marshaler.Marshal(result)
	if err == nil {
		c.client.SetEx(ctx, fullKey, data, ttl)
	}

	return result, nil
}

// Set stores a value in cache.
func (c *CacheAside[T]) Set(ctx context.Context, key string, value T) error {
	data, err := c.marshaler.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.SetEx(ctx, c.prefix+key, data, c.defaultTTL).Err()
}

// Invalidate removes keys from cache.
func (c *CacheAside[T]) Invalidate(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, k := range keys {
		fullKeys[i] = c.prefix + k
	}

	return c.client.Del(ctx, fullKeys...).Err()
}

// InvalidatePattern removes keys matching a pattern.
func (c *CacheAside[T]) InvalidatePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, c.prefix+pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

// WriteThrough implements write-through caching pattern.
type WriteThrough[T any] struct {
	cache  *CacheAside[T]
	writer func(ctx context.Context, key string, value T) error
}

// NewWriteThrough creates a new write-through cache.
func NewWriteThrough[T any](cache *CacheAside[T], writer func(ctx context.Context, key string, value T) error) *WriteThrough[T] {
	return &WriteThrough[T]{
		cache:  cache,
		writer: writer,
	}
}

// Set writes to source first, then updates cache.
func (w *WriteThrough[T]) Set(ctx context.Context, key string, value T) error {
	// Write to source first
	if err := w.writer(ctx, key, value); err != nil {
		return err
	}

	// Then update cache
	return w.cache.Set(ctx, key, value)
}

// LocalCache provides an in-memory cache with TTL.
type LocalCache[T any] struct {
	mu      sync.RWMutex
	items   map[string]localCacheItem[T]
	ttl     time.Duration
	maxSize int
}

type localCacheItem[T any] struct {
	value     T
	expiresAt time.Time
}

// NewLocalCache creates a new in-memory cache.
func NewLocalCache[T any](ttl time.Duration, maxSize int) *LocalCache[T] {
	c := &LocalCache[T]{
		items:   make(map[string]localCacheItem[T]),
		ttl:     ttl,
		maxSize: maxSize,
	}
	go c.cleanup()
	return c
}

// Get retrieves a value from the local cache.
func (c *LocalCache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok || time.Now().After(item.expiresAt) {
		var zero T
		return zero, false
	}
	return item.value, true
}

// Set stores a value in the local cache.
func (c *LocalCache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = localCacheItem[T]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from the local cache.
func (c *LocalCache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *LocalCache[T]) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range c.items {
		if oldestKey == "" || v.expiresAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.expiresAt
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

func (c *LocalCache[T]) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.expiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}

// MultiTierCache combines local and Redis caches.
type MultiTierCache[T any] struct {
	local  *LocalCache[T]
	remote *CacheAside[T]
}

// NewMultiTierCache creates a new multi-tier cache.
func NewMultiTierCache[T any](local *LocalCache[T], remote *CacheAside[T]) *MultiTierCache[T] {
	return &MultiTierCache[T]{
		local:  local,
		remote: remote,
	}
}

// Get retrieves from local cache first, then remote, then loader.
func (m *MultiTierCache[T]) Get(ctx context.Context, key string, loader func() (T, error)) (T, error) {
	// Try local cache first
	if value, ok := m.local.Get(key); ok {
		return value, nil
	}

	// Try remote cache, which will load if miss
	value, err := m.remote.Get(ctx, key, loader)
	if err != nil {
		return value, err
	}

	// Store in local cache
	m.local.Set(key, value)
	return value, nil
}

// Invalidate removes from both caches.
func (m *MultiTierCache[T]) Invalidate(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		m.local.Delete(key)
	}
	return m.remote.Invalidate(ctx, keys...)
}

// CachedResponse caches HTTP responses.
type CachedResponse struct {
	client *redis.Client
	prefix string
}

// NewCachedResponse creates a new cached response handler.
func NewCachedResponse(client *redis.Client, prefix string) *CachedResponse {
	return &CachedResponse{
		client: client,
		prefix: prefix,
	}
}

// CacheJSON caches a JSON response.
func (c *CachedResponse) CacheJSON(ctx context.Context, key string, ttl time.Duration, generator func() (any, error)) ([]byte, error) {
	fullKey := c.prefix + key

	// Check cache
	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err == nil {
		return data, nil
	}

	// Generate response
	result, err := generator()
	if err != nil {
		return nil, err
	}

	data, err = json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// Cache asynchronously
	detachedCtx := context.WithoutCancel(ctx)
	go func(ctx context.Context) {
		_ = c.client.SetEx(ctx, fullKey, data, ttl)
	}(detachedCtx)

	return data, nil
}
