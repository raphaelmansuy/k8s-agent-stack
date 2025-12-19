# pgx + sqlc + PostgreSQL RLS: The Definitive Tutorial

**Assumptions:** You know Go basics (packages, interfaces, error handling), SQL fundamentals, and have PostgreSQL installed locally or via Docker.

---

## 1) Why pgx + sqlc

- **Fastest Go PostgreSQL driver**: pgx uses the native PostgreSQL wire protocol, avoiding database/sql overhead when used directly
- **Type-safe SQL without ORM pain**: sqlc generates Go code from SQL—you get compile-time safety without query builders hiding your SQL
- **First-class RLS support**: Direct connection control lets you set session variables (`SET app.user_id`) that RLS policies depend on
- **Zero reflection at runtime**: sqlc generates plain structs and functions—no magic, no runtime overhead
- **Production-proven**: Used at Stripe, Cloudflare, Heroku; both libraries have responsive maintainers and active communities
- **Replaces**: GORM, hand-written boilerplate, fragile string concatenation
- **Complements**: Your existing PostgreSQL knowledge transfers directly

---

## 2) What Problems It Solves (and What It Doesn't)

| Good Fit | Bad Fit |
|----------|---------|
| Multi-tenant apps needing row isolation | Rapid prototyping where schema changes hourly |
| Performance-critical read-heavy workloads | Apps needing database-agnostic portability |
| Teams that want SQL control + type safety | Projects where team doesn't know SQL |
| Microservices with well-defined data contracts | Complex dynamic query building (many optional filters) |
| Compliance requirements (audit, data isolation) | Simple scripts or one-off tools |

**Real-world scenarios:**

1. **SaaS multi-tenancy**: Each customer sees only their data. RLS policies enforce this at the database level—even buggy application code can't leak data across tenants.

2. **API backend with complex queries**: Your queries involve CTEs, window functions, lateral joins. ORMs mangle these; sqlc lets you write exact SQL and get type-safe Go code.

3. **High-throughput ingestion**: pgx's batch insert and COPY protocol support handle thousands of rows per second where database/sql chokes.

---

## 3) Mental Model / Key Concepts

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Your Go Application                       │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │   sqlc      │    │  Queries    │    │   Business Logic    │  │
│  │  Generated  │◄───│  (*.sql)    │    │   (your code)       │  │
│  │    Code     │    └─────────────┘    └──────────┬──────────┘  │
│  └──────┬──────┘                                  │             │
│         │ uses                                    │ calls       │
│         ▼                                         ▼             │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                     pgxpool.Pool                            ││
│  │              (connection pool, thread-safe)                 ││
│  └─────────────────────────┬───────────────────────────────────┘│
└────────────────────────────┼────────────────────────────────────┘
                             │ PostgreSQL protocol
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        PostgreSQL                                │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ RLS Policy: USING (tenant_id = current_setting('app.tid'))│   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                          │
│  │ Table A │  │ Table B │  │ Table C │                          │
│  └─────────┘  └─────────┘  └─────────┘                          │
└─────────────────────────────────────────────────────────────────┘
```

### How the Pieces Interact

1. **You write SQL files** → sqlc reads them at build time
2. **sqlc generates Go code** → structs matching your tables, functions matching your queries
3. **Your code calls generated functions** → they use pgx under the hood
4. **pgx manages connections** → pooling, prepared statements, protocol handling
5. **For RLS**: You set session variables on a connection before queries execute
6. **PostgreSQL enforces RLS** → policies filter rows based on session variables

### Essential Glossary

| Term | Meaning |
|------|---------|
| **pgxpool.Pool** | Thread-safe connection pool; your app holds one instance |
| **pgx.Conn** | Single database connection; acquired from pool, must be released |
| **pgx.Tx** | Transaction handle; commit or rollback when done |
| **sqlc.yaml** | Configuration file telling sqlc where to find SQL and where to output Go |
| **Query annotation** | Comments like `-- name: GetUser :one` that tell sqlc how to generate code |
| **RLS (Row-Level Security)** | PostgreSQL feature where policies filter rows per-session |
| **current_setting()** | PostgreSQL function to read session variables set via `SET` |
| **BYPASSRLS** | Role attribute that skips RLS; use for migrations, never for app connections |

---

## 4) The Survival Kit

### Prioritized Checklist

**Day 0 (2 hours)**
- [ ] Install: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- [ ] Create `sqlc.yaml` with pgx/v5 configuration
- [ ] Write one query file, run `sqlc generate`, inspect output
- [ ] Connect to PostgreSQL with pgxpool, run generated query

**Week 1**
- [ ] Set up migrations (goose or golang-migrate)
- [ ] Implement RLS pattern with connection wrapper
- [ ] Write queries for all CRUD operations
- [ ] Add transaction support for multi-step operations
- [ ] Set up sqlc in CI (fail build on SQL errors)

**Week 2**
- [ ] Implement batch operations for bulk inserts
- [ ] Add query timing/logging middleware
- [ ] Write integration tests with testcontainers-go
- [ ] Profile connection pool settings for your load

### The 20% That Gives 80%

1. **`:one`, `:many`, `:exec`** — three query annotations cover 90% of cases
2. **`pgxpool.Pool`** — always use the pool, never raw connections
3. **`sqlc.arg()`** — name your parameters for clarity
4. **`pool.Acquire()` + `defer Release()`** — the RLS pattern
5. **`COALESCE` and `sqlc.narg()`** — handle nullable parameters

### Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| Connection leak | Always `defer conn.Release()` or `defer tx.Rollback()` |
| RLS not applying | Verify `ALTER TABLE ... ENABLE ROW LEVEL SECURITY` and role has no BYPASSRLS |
| Pool exhaustion | Set `MaxConns` appropriately; use context timeouts |
| sqlc type mismatch | Use `sqlc.yaml` overrides for custom types |
| Slow first query | Enable prepared statement caching in connection string |

### Debugging Tips

```go
// Enable query logging
config, _ := pgxpool.ParseConfig(connString)
config.ConnConfig.Tracer = &tracelog.TraceLog{
    Logger:   pgxLogger,
    LogLevel: tracelog.LogLevelTrace,
}
```

- Use `EXPLAIN ANALYZE` in psql first, then port query to sqlc
- Check `pg_stat_activity` for connection count and blocked queries
- For RLS debugging: `SELECT current_setting('app.tenant_id', true)`

### Performance & Security Gotchas

- **Pool size**: Start with `MaxConns = (CPU cores * 2) + effective_spindle_count`
- **Prepared statements**: pgx caches them per connection; this is usually good
- **RLS bypass**: Never give app user `BYPASSRLS`; use separate migration role
- **Context deadlines**: Always pass context with timeout to database calls
- **Connection string secrets**: Use environment variables, never hardcode

---

## 5) Progressive Complexity Examples

### Example 1: Hello, Core Primitive

**Problem:** Connect to PostgreSQL and run a simple query.

**Solution:**

```go
// main.go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx := context.Background()
    
    pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    var greeting string
    err = pool.QueryRow(ctx, "SELECT 'Hello, pgx!'").Scan(&greeting)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(greeting)
}
```

**How it works:** `pgxpool.New` parses the connection string and creates a pool. The pool manages connections automatically—acquiring for queries, returning after use. `QueryRow` executes a query expecting one row, and `Scan` maps columns to variables.

**When to use:** Bootstrapping, health checks, one-off scripts.

**Upgrade idea:** Add `pool.Ping(ctx)` for connection verification at startup.

---

### Example 2: Typical sqlc Workflow

**Problem:** Create type-safe CRUD operations for a users table.

**Solution:**

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "schema/"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
```

```sql
-- schema/001_users.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

```sql
-- queries/users.sql
-- name: CreateUser :one
INSERT INTO users (email, name)
VALUES (@email, @name)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = @id;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT @limit;

-- name: UpdateUserName :exec
UPDATE users SET name = @name WHERE id = @id;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = @id;
```

```go
// Usage in your application
package main

import (
    "context"
    "yourapp/internal/db"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx := context.Background()
    pool, _ := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
    
    queries := db.New(pool)
    
    // Create
    user, err := queries.CreateUser(ctx, db.CreateUserParams{
        Email: "alice@example.com",
        Name:  "Alice",
    })
    
    // Read
    user, err = queries.GetUserByID(ctx, user.ID)
    
    // List
    users, err := queries.ListUsers(ctx, 10)
    
    // Update
    err = queries.UpdateUserName(ctx, db.UpdateUserNameParams{
        ID:   user.ID,
        Name: "Alice Smith",
    })
}
```

**How it works:** Run `sqlc generate` after writing SQL. sqlc parses your schema and queries, generating `db.Queries` with methods matching each `-- name:` annotation. `:one` returns a single struct, `:many` returns a slice, `:exec` returns only an error.

**When to use:** Every project. This is your default pattern.

**Upgrade idea:** Add `-- name: GetUserByEmail :one` with a partial index on `lower(email)`.

---

### Example 3: Production RLS Pattern

**Problem:** Implement multi-tenant data isolation using RLS.

**Solution:**

```sql
-- schema/002_tenants.sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
);

CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    title TEXT NOT NULL,
    content TEXT
);

-- Enable RLS
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;

-- Policy: users see only their tenant's documents
CREATE POLICY tenant_isolation ON documents
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- App role must not bypass RLS
ALTER ROLE app_user SET row_security = on;
```

```sql
-- queries/documents.sql
-- name: ListDocuments :many
SELECT * FROM documents ORDER BY title;

-- name: CreateDocument :one
INSERT INTO documents (tenant_id, title, content)
VALUES (current_setting('app.tenant_id')::uuid, @title, @content)
RETURNING *;
```

```go
// internal/database/rls.go
package database

import (
    "context"
    "yourapp/internal/db"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

type TenantQueries struct {
    *db.Queries
    conn *pgxpool.Conn
}

func WithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(*TenantQueries) error) error {
    conn, err := pool.Acquire(ctx)
    if err != nil {
        return err
    }
    defer conn.Release()

    // Set tenant context for RLS
    _, err = conn.Exec(ctx, "SELECT set_config('app.tenant_id', $1, false)", tenantID.String())
    if err != nil {
        return err
    }

    tq := &TenantQueries{
        Queries: db.New(conn),
        conn:    conn,
    }

    return fn(tq)
}

// Usage
func ListTenantDocuments(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) ([]db.Document, error) {
    var docs []db.Document
    
    err := WithTenant(ctx, pool, tenantID, func(tq *TenantQueries) error {
        var err error
        docs, err = tq.ListDocuments(ctx)
        return err
    })
    
    return docs, err
}
```

**Request flow with RLS:**

```
Request (tenant: ABC)
       │
       ▼
┌──────────────┐
│ HTTP Handler │──extracts tenant ID from JWT/header
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ WithTenant() │──acquires connection, sets app.tenant_id
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   sqlc       │──executes query (no WHERE tenant_id needed)
│   Query      │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ PostgreSQL   │──RLS policy filters: tenant_id = current_setting(...)
│   + RLS      │
└──────────────┘
       │
       ▼
Only tenant ABC's rows returned
```

**How it works:** RLS policies reference `current_setting('app.tenant_id')`. Before each request's queries, you acquire a dedicated connection and set that session variable. All subsequent queries on that connection are automatically filtered. The `defer conn.Release()` ensures the connection returns to the pool.

**When to use:** Any multi-tenant SaaS, HIPAA/compliance scenarios, anywhere data leakage is unacceptable.

**When not to use:** Single-tenant apps, when you need cross-tenant queries (use a BYPASSRLS role for admin operations).

**Upgrade idea:** Wrap in a transaction for multi-query consistency:

```go
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(*db.Queries, pgx.Tx) error) error {
    conn, _ := pool.Acquire(ctx)
    defer conn.Release()
    
    tx, _ := conn.Begin(ctx)
    defer tx.Rollback(ctx)
    
    tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()) // true = local to tx
    
    if err := fn(db.New(tx), tx); err != nil {
        return err
    }
    
    return tx.Commit(ctx)
}
```

---

### Example 4: Batch Operations

**Problem:** Insert thousands of rows efficiently.

**Solution:**

```go
func BulkInsertDocuments(ctx context.Context, pool *pgxpool.Pool, docs []Document) error {
    conn, err := pool.Acquire(ctx)
    if err != nil {
        return err
    }
    defer conn.Release()

    // pgx batch for moderate sizes (100s of rows)
    batch := &pgx.Batch{}
    for _, doc := range docs {
        batch.Queue(
            "INSERT INTO documents (tenant_id, title, content) VALUES ($1, $2, $3)",
            doc.TenantID, doc.Title, doc.Content,
        )
    }
    
    br := conn.SendBatch(ctx, batch)
    defer br.Close()
    
    for range docs {
        _, err := br.Exec()
        if err != nil {
            return err
        }
    }
    
    return nil
}

// For massive inserts (10k+ rows), use COPY
func CopyInsertDocuments(ctx context.Context, pool *pgxpool.Pool, docs []Document) (int64, error) {
    conn, _ := pool.Acquire(ctx)
    defer conn.Release()
    
    return conn.CopyFrom(
        ctx,
        pgx.Identifier{"documents"},
        []string{"tenant_id", "title", "content"},
        pgx.CopyFromSlice(len(docs), func(i int) ([]any, error) {
            return []any{docs[i].TenantID, docs[i].Title, docs[i].Content}, nil
        }),
    )
}
```

**How it works:** `SendBatch` groups multiple statements into one network round-trip. `CopyFrom` uses PostgreSQL's COPY protocol—the fastest bulk loading method, bypassing normal INSERT overhead.

**When to use:** Batch for 10–1000 rows, COPY for 1000+ rows.

**Upgrade idea:** Combine with temporary table for upsert: COPY to temp table, then `INSERT ... ON CONFLICT`.

---

## 6) Cheat Sheet

### sqlc Query Annotations

```sql
-- name: Get :one          -- Returns single row (struct, error)
-- name: List :many        -- Returns slice of rows ([]struct, error)
-- name: Create :one       -- Insert returning row
-- name: Update :exec      -- Returns only error
-- name: Delete :exec
-- name: Count :one        -- For COUNT(*) queries
-- name: Upsert :one       -- INSERT ... ON CONFLICT
-- name: Exists :one       -- SELECT EXISTS(...)
```

### pgx Connection Patterns

```go
pool.Query(ctx, sql, args...)     // Multiple rows
pool.QueryRow(ctx, sql, args...)  // Single row
pool.Exec(ctx, sql, args...)      // No rows (INSERT/UPDATE/DELETE)
pool.Begin(ctx)                   // Start transaction
pool.Acquire(ctx)                 // Get dedicated connection (for RLS)
conn.Release()                    // Return to pool
tx.Commit(ctx) / tx.Rollback(ctx)
```

### sqlc.yaml Quick Reference

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "schema/"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
```

### RLS Essentials

```sql
ALTER TABLE t ENABLE ROW LEVEL SECURITY;
CREATE POLICY name ON t USING (condition);
CREATE POLICY name ON t FOR SELECT/INSERT/UPDATE/DELETE USING (...) WITH CHECK (...);
SELECT current_setting('app.tenant_id', true);  -- true = return null if not set
SET app.tenant_id = 'value';                    -- session level
SELECT set_config('app.tenant_id', 'val', true); -- true = transaction local
```

### If You Only Remember 5 Things

1. **Always use `pgxpool.Pool`**, never create connections manually
2. **Run `sqlc generate`** after every SQL change; commit generated code
3. **For RLS: `Acquire` → `set_config` → queries → `Release`**
4. **Use `:one` for single row, `:many` for lists, `:exec` for mutations without return**
5. **Context with timeout on every database call**

---

## 7) Related Technologies & Concepts

### Alternatives

| Technology | Choose when... |
|------------|----------------|
| **GORM** | Team prefers ORM, rapid prototyping, simpler apps |
| **Bun** | Want lightweight ORM with better performance than GORM |
| **ent** | Need code-generated schema, complex graph traversals |
| **database/sql + pq** | Need database/sql compatibility (legacy code, testing) |

### Complements

| Technology | Purpose |
|------------|---------|
| **goose** | SQL migrations with Go or SQL files |
| **golang-migrate** | Alternative migration tool, more formats |
| **testcontainers-go** | Spin up PostgreSQL in tests |
| **pgbouncer** | External connection pooling (high-scale) |
| **PostgREST** | Auto-generate REST API (different architecture) |

### Prerequisites

| Concept | Why needed |
|---------|-----------|
| **SQL fluency** | sqlc doesn't hide SQL; you must write it |
| **PostgreSQL specifics** | CTEs, window functions, JSON ops, etc. |
| **Go context package** | Timeouts, cancellation, request-scoped values |

### Next Steps

| Topic | When ready to... |
|-------|-----------------|
| **pgx tracing** | Add OpenTelemetry integration |
| **LISTEN/NOTIFY** | Build real-time features |
| **Logical replication** | Stream changes to other systems |
| **pg_stat_statements** | Performance analysis |

---

## 8) Resources & Documentation

| Resource | URL | Description |
|----------|-----|-------------|
| **pgx GitHub** | https://github.com/jackc/pgx | Driver source, examples, issues |
| **pgx Documentation** | https://pkg.go.dev/github.com/jackc/pgx/v5 | Go package docs |
| **sqlc Documentation** | https://docs.sqlc.dev | Official guides and reference |
| **sqlc GitHub** | https://github.com/sqlc-dev/sqlc | Source, issues, discussions |
| **sqlc Playground** | https://play.sqlc.dev | Try sqlc in browser |
| **PostgreSQL RLS Docs** | https://www.postgresql.org/docs/current/ddl-rowsecurity.html | Official RLS reference |
| **goose Migrations** | https://github.com/pressly/goose | Migration tool |
| **golang-migrate** | https://github.com/golang-migrate/migrate | Alternative migrations |
| **testcontainers-go** | https://golang.testcontainers.org | Integration testing |
| **pgx Examples** | https://github.com/jackc/pgx/tree/master/examples | Official examples |

---

*Total time to productivity: Day 0 gets you connected and querying. Week 1 gets you production-ready patterns. Week 2 gets you optimized.*