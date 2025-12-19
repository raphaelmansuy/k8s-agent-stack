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

package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
)

type EvaluationRepository struct {
	queries *db.Queries
}

func NewEvaluationRepository(queries *db.Queries) *EvaluationRepository {
	return &EvaluationRepository{
		queries: queries,
	}
}

func (r *EvaluationRepository) SaveTrace(ctx context.Context, trace *evaluation.Trace) error {
	metadata, _ := json.Marshal(trace.Metadata)
	steps, _ := json.Marshal(trace.Steps)

	_, err := r.queries.CreateTrace(ctx, db.CreateTraceParams{
		AgentID:   trace.AgentID,
		SessionID: trace.SessionID,
		Input:     trace.Input,
		Output:    trace.Output,
		LatencyMs: trace.Latency.Milliseconds(),
		TokensIn:  int32(trace.TokensIn),
		TokensOut: int32(trace.TokensOut),
		Steps:     steps,
		Metadata:  metadata,
	})
	return err
}

func (r *EvaluationRepository) GetTrace(ctx context.Context, id string) (*evaluation.Trace, error) {
	row, err := r.queries.GetTrace(ctx, id)
	if err != nil {
		return nil, err
	}

	return r.mapTrace(row), nil
}

func (r *EvaluationRepository) ListTraces(ctx context.Context, filter evaluation.TraceFilter) ([]*evaluation.Trace, error) {
	limit := int32(filter.Limit)
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.queries.ListTraces(ctx, db.ListTracesParams{
		AgentID:     filter.AgentID,
		SessionID:   filter.SessionID,
		CreatedAt:   filter.StartTime,
		CreatedAt_2: filter.EndTime,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}

	traces := make([]*evaluation.Trace, len(rows))
	for i, row := range rows {
		traces[i] = r.mapTrace(row)
	}
	return traces, nil
}

func (r *EvaluationRepository) SaveFeedback(ctx context.Context, feedback *evaluation.Feedback) error {
	_, err := r.queries.CreateFeedback(ctx, db.CreateFeedbackParams{
		TraceID: feedback.TraceID,
		Rating:  int32(feedback.Rating),
		Comment: pgtype.Text{String: feedback.Comment, Valid: feedback.Comment != ""},
		Tags:    feedback.Tags,
	})
	return err
}

func (r *EvaluationRepository) GetFeedback(ctx context.Context, traceID string) ([]*evaluation.Feedback, error) {
	rows, err := r.queries.ListFeedbackByTrace(ctx, traceID)
	if err != nil {
		return nil, err
	}

	feedbacks := make([]*evaluation.Feedback, len(rows))
	for i, row := range rows {
		feedbacks[i] = &evaluation.Feedback{
			ID:        row.ID,
			TraceID:   row.TraceID,
			Rating:    int(row.Rating),
			Comment:   row.Comment.String,
			Tags:      row.Tags,
			CreatedAt: row.CreatedAt,
		}
	}
	return feedbacks, nil
}

func (r *EvaluationRepository) mapTrace(row db.Trace) *evaluation.Trace {
	var metadata map[string]interface{}
	_ = json.Unmarshal(row.Metadata, &metadata)

	var steps []evaluation.Step
	_ = json.Unmarshal(row.Steps, &steps)

	return &evaluation.Trace{
		ID:        row.ID,
		AgentID:   row.AgentID,
		SessionID: row.SessionID,
		Input:     row.Input,
		Output:    row.Output,
		Latency:   time.Duration(row.LatencyMs) * time.Millisecond,
		TokensIn:  int(row.TokensIn),
		TokensOut: int(row.TokensOut),
		Steps:     steps,
		Metadata:  metadata,
		CreatedAt: row.CreatedAt,
	}
}
