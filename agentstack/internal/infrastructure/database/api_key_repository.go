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
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/raphaelmansuy/agentstack/internal/domain/auth"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
)

type APIKeyRepository struct {
	pool *Pool
}

func NewAPIKeyRepository(pool *Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *APIKeyRepository) CreateAPIKey(ctx context.Context, key *auth.APIKey) error {
	queries, cleanup, err := r.pool.Queries(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	var expiresAt pgtype.Timestamptz
	if key.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *key.ExpiresAt, Valid: true}
	}

	row, err := queries.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		TeamID:    key.TeamID,
		ProjectID: pgtype.Text{String: key.ProjectID, Valid: key.ProjectID != ""},
		Name:      key.Name,
		KeyHash:   key.KeyHash,
		KeyPrefix: key.KeyPrefix,
		Scopes:    key.Scopes,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return err
	}

	key.ID = row.ID
	key.CreatedAt = row.CreatedAt
	return nil
}

func (r *APIKeyRepository) GetAPIKey(ctx context.Context, id string) (*auth.APIKey, error) {
	queries, cleanup, err := r.pool.Queries(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	row, err := queries.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return r.mapAPIKey(row), nil
}

func (r *APIKeyRepository) GetAPIKeyByHash(ctx context.Context, hash string) (*auth.APIKey, error) {
	queries, cleanup, err := r.pool.Queries(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	row, err := queries.GetAPIKey(ctx, hash)
	if err != nil {
		return nil, err
	}

	return r.mapAPIKey(row), nil
}

func (r *APIKeyRepository) ListAPIKeys(ctx context.Context, teamID string) ([]*auth.APIKey, error) {
	queries, cleanup, err := r.pool.Queries(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	rows, err := queries.ListAPIKeys(ctx, teamID)
	if err != nil {
		return nil, err
	}

	keys := make([]*auth.APIKey, len(rows))
	for i, row := range rows {
		keys[i] = &auth.APIKey{
			ID:         row.ID,
			TeamID:     row.TeamID,
			ProjectID:  row.ProjectID.String,
			Name:       row.Name,
			KeyPrefix:  row.KeyPrefix,
			Scopes:     row.Scopes,
			LastUsedAt: r.mapTime(row.LastUsedAt),
			ExpiresAt:  r.mapTime(row.ExpiresAt),
			CreatedAt:  row.CreatedAt,
		}
	}
	return keys, nil
}

func (r *APIKeyRepository) DeleteAPIKey(ctx context.Context, id string) error {
	queries, cleanup, err := r.pool.Queries(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	return queries.DeleteAPIKey(ctx, id)
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	queries, cleanup, err := r.pool.Queries(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	return queries.UpdateAPIKeyLastUsed(ctx, id)
}

func (r *APIKeyRepository) RotateAPIKey(ctx context.Context, id, hash, prefix string) (*auth.APIKey, error) {
	queries, cleanup, err := r.pool.Queries(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	row, err := queries.RotateAPIKey(ctx, db.RotateAPIKeyParams{
		ID:        id,
		KeyHash:   hash,
		KeyPrefix: prefix,
	})
	if err != nil {
		return nil, err
	}

	return r.mapAPIKey(row), nil
}

func (r *APIKeyRepository) mapAPIKey(row db.ApiKey) *auth.APIKey {
	return &auth.APIKey{
		ID:         row.ID,
		TeamID:     row.TeamID,
		ProjectID:  row.ProjectID.String,
		Name:       row.Name,
		KeyHash:    row.KeyHash,
		KeyPrefix:  row.KeyPrefix,
		Scopes:     row.Scopes,
		LastUsedAt: r.mapTime(row.LastUsedAt),
		ExpiresAt:  r.mapTime(row.ExpiresAt),
		CreatedAt:  row.CreatedAt,
	}
}

func (r *APIKeyRepository) mapTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
