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

func (c *RBACCache) Set(ctx context.Context, key, value string, expiration time.Duration) error {
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

// Ensure RBACCache implements rbac.Cache.
var _ rbac.Cache = (*RBACCache)(nil)
