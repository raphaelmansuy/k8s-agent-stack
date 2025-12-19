// Package database provides PostgreSQL database connection and utilities.
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
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raphaelmansuy/agentstack/internal/api/middleware"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database/db"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/telemetry"
)

type contextKey string

const (
	connKey contextKey = "db_conn"
)

// Pool wraps pgxpool.Pool with additional functionality.
type Pool struct {
	*pgxpool.Pool
	telemetry *telemetry.Telemetry
}

// NewPool creates a new PostgreSQL connection pool.
func NewPool(ctx context.Context, databaseURL string, t *telemetry.Telemetry) (*Pool, error) {
	fmt.Printf("DEBUG: Connecting to database at %s\n", databaseURL)
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure pool settings
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute

	// Configure connection settings
	config.ConnConfig.ConnectTimeout = 5 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Pool{Pool: pool, telemetry: t}, nil
}

// WithTenant returns a connection with the tenant context set for RLS.
func (p *Pool) WithTenant(ctx context.Context, teamID string) (*pgxpool.Conn, error) {
	conn, err := p.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	// Set the current team for Row-Level Security
	_, err = conn.Exec(ctx, "SELECT set_config('app.tenant_id', $1, false)", teamID)
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	return conn, nil
}

// Queries returns a Queries object bound to a connection with the tenant context set.
// The caller MUST call the returned cleanup function to release the connection.
func (p *Pool) Queries(ctx context.Context) (*db.Queries, func(), error) {
	// Try to get team ID from context using middleware helper
	teamID := middleware.GetTeamID(ctx)

	if teamID == "" {
		// If no team ID, return queries on the pool (no RLS)
		return db.New(p.Pool), func() {}, nil
	}

	conn, err := p.WithTenant(ctx, teamID)
	if err != nil {
		return nil, nil, err
	}

	return db.New(conn), func() { conn.Release() }, nil
}

// TenantDB implements db.DBTX and automatically uses the tenant connection from context if available.
type TenantDB struct {
	pool *Pool
}

func (d *TenantDB) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	var err error
	if d.pool.telemetry != nil {
		finish := d.pool.telemetry.DBQueryHook(ctx, "Exec", sql, arguments)
		defer func() { finish(err) }()
	}
	if conn, ok := ctx.Value(connKey).(*pgxpool.Conn); ok {
		tag, e := conn.Exec(ctx, sql, arguments...)
		err = e
		return tag, err
	}
	tag, e := d.pool.Exec(ctx, sql, arguments...)
	err = e
	return tag, err
}

func (d *TenantDB) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	var err error
	if d.pool.telemetry != nil {
		finish := d.pool.telemetry.DBQueryHook(ctx, "Query", sql, arguments)
		defer func() { finish(err) }()
	}
	if conn, ok := ctx.Value(connKey).(*pgxpool.Conn); ok {
		rows, e := conn.Query(ctx, sql, arguments...) //nolint:sqlclosecheck // Rows are closed by the caller
		err = e
		return rows, err
	}
	rows, e := d.pool.Query(ctx, sql, arguments...) //nolint:sqlclosecheck // Rows are closed by the caller
	err = e
	return rows, err
}

func (d *TenantDB) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	if d.pool.telemetry != nil {
		defer d.pool.telemetry.DBQueryHook(ctx, "QueryRow", sql, arguments)(nil)
	}
	if conn, ok := ctx.Value(connKey).(*pgxpool.Conn); ok {
		return conn.QueryRow(ctx, sql, arguments...)
	}
	return d.pool.QueryRow(ctx, sql, arguments...)
}

// NewTenantDB creates a new TenantDB.
func (p *Pool) NewTenantDB() *TenantDB {
	return &TenantDB{pool: p}
}

// TenantMiddleware returns a middleware that sets the tenant context for RLS.
func (p *Pool) TenantMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			teamID := middleware.GetTeamID(r.Context())
			if teamID == "" {
				next.ServeHTTP(w, r)
				return
			}

			conn, err := p.WithTenant(r.Context(), teamID)
			if err != nil {
				http.Error(w, "Failed to set tenant context", http.StatusInternalServerError)
				return
			}
			defer conn.Release()

			ctx := context.WithValue(r.Context(), connKey, conn)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Transaction executes a function within a database transaction.
func (p *Pool) Transaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %w, rollback error: %w", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

// HealthCheck verifies the database connection is healthy.
func (p *Pool) HealthCheck(ctx context.Context) error {
	return p.Ping(ctx)
}

// Stats returns pool statistics.
func (p *Pool) Stats() *pgxpool.Stat {
	return p.Stat()
}
