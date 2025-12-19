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

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/raphaelmansuy/agentstack/internal/domain/quota"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
)

type QuotaRepository struct {
	queries *db.Queries
}

func NewQuotaRepository(queries *db.Queries) *QuotaRepository {
	return &QuotaRepository{
		queries: queries,
	}
}

func (r *QuotaRepository) GetQuota(ctx context.Context, teamID string, quotaType quota.QuotaType) (*quota.Quota, error) {
	row, err := r.queries.GetQuota(ctx, db.GetQuotaParams{
		TeamID:    teamID,
		QuotaType: string(quotaType),
		ProjectID: pgtype.Text{Valid: false},
	})
	if err != nil {
		return nil, err
	}

	return &quota.Quota{
		ID:        row.ID,
		TeamID:    row.TeamID,
		ProjectID: row.ProjectID.String,
		Type:      quota.QuotaType(row.QuotaType),
		Limit:     row.LimitValue,
		Period:    row.Period.String,
	}, nil
}

func (r *QuotaRepository) GetQuotas(ctx context.Context, teamID string) ([]quota.Quota, error) {
	rows, err := r.queries.ListQuotas(ctx, teamID)
	if err != nil {
		return nil, err
	}

	quotas := make([]quota.Quota, len(rows))
	for i, row := range rows {
		quotas[i] = quota.Quota{
			ID:        row.ID,
			TeamID:    row.TeamID,
			ProjectID: row.ProjectID.String,
			Type:      quota.QuotaType(row.QuotaType),
			Limit:     row.LimitValue,
			Period:    row.Period.String,
		}
	}
	return quotas, nil
}

func (r *QuotaRepository) SetQuota(ctx context.Context, q *quota.Quota) error {
	if q.ProjectID != "" {
		_, err := r.queries.SetQuota(ctx, db.SetQuotaParams{
			TeamID:     q.TeamID,
			ProjectID:  pgtype.Text{String: q.ProjectID, Valid: true},
			QuotaType:  string(q.Type),
			LimitValue: q.Limit,
			Period:     pgtype.Text{String: q.Period, Valid: q.Period != ""},
		})
		return err
	}

	_, err := r.queries.SetTeamQuota(ctx, db.SetTeamQuotaParams{
		TeamID:     q.TeamID,
		QuotaType:  string(q.Type),
		LimitValue: q.Limit,
		Period:     pgtype.Text{String: q.Period, Valid: q.Period != ""},
	})
	return err
}

func (r *QuotaRepository) GetUsage(ctx context.Context, quotaID string) (*quota.Usage, error) {
	row, err := r.queries.GetQuotaUsage(ctx, quotaID)
	if err != nil {
		return nil, err
	}

	return &quota.Usage{
		QuotaID: row.QuotaID,
		Current: row.CurrentValue,
	}, nil
}

func (r *QuotaRepository) IncrementUsage(ctx context.Context, quotaID string, amount int64) error {
	return r.queries.IncrementQuotaUsageByID(ctx, db.IncrementQuotaUsageByIDParams{
		QuotaID: quotaID,
		Amount:  amount,
	})
}
