package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type EvaluationWorker struct {
	redisClient *redis.Client
	evalService *evaluation.Service
	logger      *zap.Logger
	stream      string
	group       string
	consumer    string
}

func NewEvaluationWorker(redisClient *redis.Client, evalService *evaluation.Service, logger *zap.Logger) *EvaluationWorker {
	return &EvaluationWorker{
		redisClient: redisClient,
		evalService: evalService,
		logger:      logger,
		stream:      "evaluation:queue",
		group:       "evaluation:group",
		consumer:    "worker-1",
	}
}

func (w *EvaluationWorker) Start(ctx context.Context) {
	w.logger.Info("Starting evaluation worker")

	// Create consumer group if it doesn't exist
	err := w.redisClient.XGroupCreateMkStream(ctx, w.stream, w.group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		w.logger.Error("Failed to create consumer group", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping evaluation worker")
			return
		default:
			// Read from stream
			streams, err := w.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    w.group,
				Consumer: w.consumer,
				Streams:  []string{w.stream, ">"},
				Count:    1,
				Block:    5 * time.Second,
			}).Result()

			if err != nil {
				if err != redis.Nil {
					w.logger.Error("Error reading from stream", zap.Error(err))
					time.Sleep(time.Second)
				}
				continue
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					w.processMessage(ctx, message)
				}
			}
		}
	}
}

func (w *EvaluationWorker) processMessage(ctx context.Context, message redis.XMessage) {
	data, ok := message.Values["data"].(string)
	if !ok {
		w.logger.Error("Invalid message format", zap.String("id", message.ID))
		w.redisClient.XAck(ctx, w.stream, w.group, message.ID)
		return
	}

	var trace evaluation.Trace
	if err := json.Unmarshal([]byte(data), &trace); err != nil {
		w.logger.Error("Failed to unmarshal trace", zap.Error(err))
		w.redisClient.XAck(ctx, w.stream, w.group, message.ID)
		return
	}

	w.logger.Info("Processing evaluation for trace", zap.String("trace_id", trace.ID))

	_, err := w.evalService.RunEvaluation(ctx, &trace)
	if err != nil {
		w.logger.Error("Failed to run evaluation", zap.Error(err), zap.String("trace_id", trace.ID))
		// In a real app, we might want to retry or move to a DLQ
	}

	// Acknowledge message
	w.redisClient.XAck(ctx, w.stream, w.group, message.ID)
}
