package cache

import (
	"context"

	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
	"github.com/redis/go-redis/v9"
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

// Ensure EvaluationQueue implements evaluation.Queue
var _ evaluation.Queue = (*EvaluationQueue)(nil)
