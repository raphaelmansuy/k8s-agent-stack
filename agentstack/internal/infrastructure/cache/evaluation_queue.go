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

	"github.com/redis/go-redis/v9"

	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
)

type EvaluationQueue struct {
	client *Client
}

func NewEvaluationQueue(client *Client) *EvaluationQueue {
	return &EvaluationQueue{
		client: client,
	}
}

func (q *EvaluationQueue) XAdd(ctx context.Context, stream string, data []byte) error {
	if q.client.rdb == nil {
		return nil // Or return error if queue is mandatory
	}

	return q.client.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"data": data,
		},
	}).Err()
}

// Ensure EvaluationQueue implements evaluation.Queue.
var _ evaluation.Queue = (*EvaluationQueue)(nil)
