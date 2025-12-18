// Package db provides optimized database connection management.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig contains production-tuned pool settings.
type PoolConfig struct {
	DSN                string
	MaxConns           int32
	MinConns           int32
	MaxConnLifetime    time.Duration
	MaxConnIdleTime    time.Duration
	HealthCheckPeriod  time.Duration
	ConnectTimeout     time.Duration
	StatementCacheSize int
}

// DefaultPoolConfig returns production-tuned defaults.
func DefaultPoolConfig(dsn string) *PoolConfig {
	return &PoolConfig{
		DSN:                dsn,
		MaxConns:           100,
		MinConns:           10,
		MaxConnLifetime:    1 * time.Hour,
		MaxConnIdleTime:    30 * time.Minute,
		HealthCheckPeriod:  1 * time.Minute,
		ConnectTimeout:     5 * time.Second,
		StatementCacheSize: 512,
	}
}

// NewOptimizedPool creates a production-optimized connection pool.
func NewOptimizedPool(ctx context.Context, cfg *PoolConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Connection pool settings
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	// Connection settings
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	// Prepared statement cache for better performance
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement

	// Build describer for prepared statements
	poolConfig.ConnConfig.DescriptionCacheCapacity = cfg.StatementCacheSize

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// PoolMetrics provides metrics about the connection pool.
type PoolMetrics struct {
	pool *pgxpool.Pool
}

// NewPoolMetrics creates a new pool metrics collector.
func NewPoolMetrics(pool *pgxpool.Pool) *PoolMetrics {
	return &PoolMetrics{pool: pool}
}

// Stats returns current pool statistics.
func (m *PoolMetrics) Stats() PoolStats {
	stat := m.pool.Stat()
	return PoolStats{
		TotalConns:       stat.TotalConns(),
		AcquiredConns:    stat.AcquiredConns(),
		IdleConns:        stat.IdleConns(),
		MaxConns:         stat.MaxConns(),
		AcquireCount:     stat.AcquireCount(),
		AcquireDuration:  stat.AcquireDuration(),
		EmptyAcquires:    stat.EmptyAcquireCount(),
		CanceledAcquires: stat.CanceledAcquireCount(),
	}
}

// PoolStats contains pool statistics.
type PoolStats struct {
	TotalConns       int32
	AcquiredConns    int32
	IdleConns        int32
	MaxConns         int32
	AcquireCount     int64
	AcquireDuration  time.Duration
	EmptyAcquires    int64
	CanceledAcquires int64
}

// HealthCheck verifies the pool is healthy.
func (m *PoolMetrics) HealthCheck(ctx context.Context) error {
	conn, err := m.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	return conn.Ping(ctx)
}

// Query represents a query with arguments.
type Query struct {
	SQL  string
	Args []any
}

// BatchExecutor executes multiple queries in a single round-trip.
type BatchExecutor struct {
	pool *pgxpool.Pool
}

// NewBatchExecutor creates a new batch executor.
func NewBatchExecutor(pool *pgxpool.Pool) *BatchExecutor {
	return &BatchExecutor{pool: pool}
}

// Execute runs multiple queries in a batch.
func (b *BatchExecutor) Execute(ctx context.Context, queries []Query) error {
	batch := &pgx.Batch{}

	for _, q := range queries {
		batch.Queue(q.SQL, q.Args...)
	}

	results := b.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range queries {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("batch query failed: %w", err)
		}
	}

	return nil
}

// BulkInsert performs efficient bulk inserts using COPY.
func (b *BatchExecutor) BulkInsert(ctx context.Context, table string, columns []string, rows [][]any) (int64, error) {
	copyCount, err := b.pool.CopyFrom(
		ctx,
		pgx.Identifier{table},
		columns,
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		return 0, fmt.Errorf("bulk insert failed: %w", err)
	}

	return copyCount, nil
}

// IndexRecommendation suggests an index for a table.
type IndexRecommendation struct {
	Table          string  `json:"table"`
	SeqScans       int64   `json:"seq_scans"`
	IdxScans       int64   `json:"idx_scans"`
	LiveTuples     int64   `json:"live_tuples"`
	Ratio          float64 `json:"scan_ratio"`
	Recommendation string  `json:"recommendation"`
}

// AnalyzeSlowTables identifies tables that might benefit from indexing.
func AnalyzeSlowTables(ctx context.Context, pool *pgxpool.Pool) ([]IndexRecommendation, error) {
	query := `
		SELECT 
			schemaname || '.' || relname as table_name,
			seq_scan,
			idx_scan,
			n_live_tup
		FROM pg_stat_user_tables
		WHERE seq_scan > COALESCE(idx_scan, 0) * 10
		AND n_live_tup > 10000
		ORDER BY seq_scan DESC
		LIMIT 10
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze tables: %w", err)
	}
	defer rows.Close()

	var recommendations []IndexRecommendation
	for rows.Next() {
		var rec IndexRecommendation
		var idxScan *int64
		err := rows.Scan(&rec.Table, &rec.SeqScans, &idxScan, &rec.LiveTuples)
		if err != nil {
			return nil, err
		}

		if idxScan != nil {
			rec.IdxScans = *idxScan
		}

		if rec.IdxScans > 0 {
			rec.Ratio = float64(rec.SeqScans) / float64(rec.IdxScans)
		} else {
			rec.Ratio = float64(rec.SeqScans)
		}

		rec.Recommendation = fmt.Sprintf("Consider adding index on %s - %d seq scans vs %d idx scans",
			rec.Table, rec.SeqScans, rec.IdxScans)

		recommendations = append(recommendations, rec)
	}

	return recommendations, rows.Err()
}

// QueryStats provides query performance statistics.
type QueryStats struct {
	Query     string        `json:"query"`
	Calls     int64         `json:"calls"`
	TotalTime time.Duration `json:"total_time"`
	MeanTime  time.Duration `json:"mean_time"`
	MinTime   time.Duration `json:"min_time"`
	MaxTime   time.Duration `json:"max_time"`
	Rows      int64         `json:"rows"`
}

// GetSlowQueries returns the slowest queries (requires pg_stat_statements).
func GetSlowQueries(ctx context.Context, pool *pgxpool.Pool, limit int) ([]QueryStats, error) {
	query := `
		SELECT 
			query,
			calls,
			total_exec_time * interval '1 millisecond' as total_time,
			mean_exec_time * interval '1 millisecond' as mean_time,
			min_exec_time * interval '1 millisecond' as min_time,
			max_exec_time * interval '1 millisecond' as max_time,
			rows
		FROM pg_stat_statements
		WHERE userid = (SELECT usesysid FROM pg_user WHERE usename = current_user)
		ORDER BY total_exec_time DESC
		LIMIT $1
	`

	rows, err := pool.Query(ctx, query, limit)
	if err != nil {
		// pg_stat_statements might not be installed
		return nil, nil
	}
	defer rows.Close()

	var stats []QueryStats
	for rows.Next() {
		var s QueryStats
		err := rows.Scan(&s.Query, &s.Calls, &s.TotalTime, &s.MeanTime, &s.MinTime, &s.MaxTime, &s.Rows)
		if err != nil {
			continue
		}
		stats = append(stats, s)
	}

	return stats, nil
}
