# 005 - Data Architecture

> Storage, State Management, and Persistence Patterns

**Version**: 1.0.0 | **Status**: Draft | **Last Updated**: 2025-12-17

---

## 1. Data Architecture Overview

```text
┌─────────────────────────────────────────────────────────────────┐
│                     Data Architecture                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    HOT DATA (ms latency)                │    │
│  │                    Redis Cluster                        │    │
│  │  • Session state    • Rate limiting    • Cache          │    │
│  │  • Pub/Sub          • Locks            • Counters       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   WARM DATA (10-100ms)                  │    │
│  │                   PostgreSQL                            │    │
│  │  • Agents           • Deployments      • Users          │    │
│  │  • Projects         • Audit logs       • Configs        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   COLD DATA (100ms+)                    │    │
│  │                   Object Storage (S3)                   │    │
│  │  • Build artifacts  • Logs archive     • Backups        │    │
│  │  • Agent source     • Analytics data                    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   VECTOR DATA                           │    │
│  │                   pgvector / Qdrant                     │    │
│  │  • Embeddings       • Semantic search  • Memory         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Database Selection

### Decision Matrix

| Requirement | PostgreSQL | CockroachDB | MySQL |
|-------------|------------|-------------|-------|
| **License** | PostgreSQL (permissive) | BSL 1.1 ⚠️ | GPL |
| **JSON Support** | ✅ JSONB | ✅ | Limited |
| **Vector Extension** | ✅ pgvector | ❌ | ❌ |
| **Partitioning** | ✅ Native | ✅ | ✅ |
| **Community** | Largest | Growing | Large |

**Decision**: PostgreSQL 16+ with pgvector

**Rationale**: Best license, native vector support, mature ecosystem.

---

## 3. Core Schema

### Entity Relationship

```text
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│    Team     │────<│   Project   │────<│    Agent    │
└─────────────┘     └─────────────┘     └──────┬──────┘
      │                                        │
      │                                        │
┌─────┴─────┐                           ┌──────┴──────┐
│TeamMember │                           │ Deployment  │
└───────────┘                           └──────┬──────┘
                                               │
                                        ┌──────┴──────┐
                                        │  Revision   │
                                        └─────────────┘

┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Session    │────<│   Message   │     │    Tool     │
└─────────────┘     └─────────────┘     └─────────────┘

┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Secret    │     │   Webhook   │     │   APIKey    │
└─────────────┘     └─────────────┘     └─────────────┘
```

### Core Tables

```sql
-- Teams (multi-tenancy root)
CREATE TABLE teams (
    id          TEXT PRIMARY KEY,  -- team_xxx
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    plan        TEXT NOT NULL DEFAULT 'free',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Projects (isolation boundary)
CREATE TABLE projects (
    id          TEXT PRIMARY KEY,  -- prj_xxx
    team_id     TEXT NOT NULL REFERENCES teams(id),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    settings    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (team_id, slug)
);

-- Agents
CREATE TABLE agents (
    id              TEXT PRIMARY KEY,  -- agt_xxx
    project_id      TEXT NOT NULL REFERENCES projects(id),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'inactive',
    framework       TEXT NOT NULL,
    source          JSONB,
    config          JSONB NOT NULL DEFAULT '{}',
    tags            TEXT[] DEFAULT '{}',
    deployment_id   TEXT,  -- current deployment
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, slug)
);

-- Indexes for common queries
CREATE INDEX idx_agents_project_status ON agents(project_id, status);
CREATE INDEX idx_agents_tags ON agents USING GIN(tags);
CREATE INDEX idx_agents_name_search ON agents USING GIN(name gin_trgm_ops);
```

### Partitioned Tables (High Volume)

```sql
-- Usage events (partitioned by month)
CREATE TABLE usage_events (
    id          BIGSERIAL,
    project_id  TEXT NOT NULL,
    agent_id    TEXT,
    event_type  TEXT NOT NULL,
    tokens_in   INTEGER,
    tokens_out  INTEGER,
    duration_ms INTEGER,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE usage_events_2025_01 PARTITION OF usage_events
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE usage_events_2025_02 PARTITION OF usage_events
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
-- Auto-create via cron job
```

---

## 4. Redis Architecture

### Data Patterns

```text
┌─────────────────────────────────────────────────────────────────┐
│                     Redis Data Patterns                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SESSION STATE                                                  │
│  Key:   session:{session_id}                                    │
│  Type:  HASH                                                    │
│  TTL:   24h                                                     │
│  Data:  {user_id, agent_id, context, last_message_at}           │
│                                                                 │
│  RATE LIMITING                                                  │
│  Key:   ratelimit:{user_id}:{minute}                            │
│  Type:  STRING (counter)                                        │
│  TTL:   60s                                                     │
│                                                                 │
│  CACHE                                                          │
│  Key:   cache:agent:{agent_id}                                  │
│  Type:  STRING (JSON)                                           │
│  TTL:   5m                                                      │
│                                                                 │
│  IDEMPOTENCY                                                    │
│  Key:   idem:{idempotency_key}                                  │
│  Type:  STRING (response JSON)                                  │
│  TTL:   24h                                                     │
│                                                                 │
│  PUBSUB                                                         │
│  Channel: events:{project_id}                                   │
│  Use:     Real-time updates to connected clients                │
│                                                                 │
│  LOCKS                                                          │
│  Key:   lock:deploy:{agent_id}                                  │
│  Type:  STRING                                                  │
│  TTL:   5m (with refresh)                                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Redis Cluster Configuration

```yaml
# Minimum 6 nodes (3 master + 3 replica)
redis:
  mode: cluster
  nodes: 6
  replicas: 1
  memory_per_node: 4Gi
  persistence:
    enabled: true
    type: appendonly
  eviction_policy: volatile-lru
```

---

## 5. Object Storage

### Bucket Structure

```text
agentstack-{env}/
├── builds/
│   └── {agent_id}/
│       └── {deployment_id}/
│           ├── source.tar.gz
│           ├── build.log
│           └── image-digest.txt
│
├── logs/
│   └── {date}/
│       └── {agent_id}/
│           └── {hour}.log.gz
│
├── backups/
│   └── postgres/
│       └── {date}/
│           └── full.sql.gz
│
└── artifacts/
    └── {project_id}/
        └── {artifact_id}
```

### Lifecycle Policies

```yaml
lifecycle_rules:
  - prefix: "builds/"
    transitions:
      - days: 30
        storage_class: STANDARD_IA
      - days: 90
        storage_class: GLACIER
    expiration:
      days: 365

  - prefix: "logs/"
    transitions:
      - days: 7
        storage_class: STANDARD_IA
    expiration:
      days: 90
```

---

## 6. Vector Storage

### pgvector Schema

```sql
-- Enable extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Conversation memory with embeddings
CREATE TABLE memory_embeddings (
    id          TEXT PRIMARY KEY,
    agent_id    TEXT NOT NULL REFERENCES agents(id),
    session_id  TEXT,
    content     TEXT NOT NULL,
    embedding   vector(1536),  -- OpenAI ada-002 dimension
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- HNSW index for fast similarity search
CREATE INDEX idx_memory_embedding ON memory_embeddings 
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Partition by agent for isolation
CREATE INDEX idx_memory_agent ON memory_embeddings(agent_id);

-- Similarity search function
CREATE OR REPLACE FUNCTION search_memory(
    p_agent_id TEXT,
    p_embedding vector(1536),
    p_limit INT DEFAULT 10
)
RETURNS TABLE (
    id TEXT,
    content TEXT,
    similarity FLOAT,
    metadata JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id,
        m.content,
        1 - (m.embedding <=> p_embedding) as similarity,
        m.metadata
    FROM memory_embeddings m
    WHERE m.agent_id = p_agent_id
    ORDER BY m.embedding <=> p_embedding
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
```

### Alternative: Qdrant (For Scale)

```yaml
# When pgvector is insufficient (>10M vectors)
qdrant:
  enabled: true
  collections:
    - name: agent_memory
      vector_size: 1536
      distance: Cosine
      replication_factor: 2
      shard_number: 4
  storage:
    type: persistent
    size: 100Gi
```

**Trade-off**: pgvector simpler (single DB), Qdrant better at scale.

---

## 7. Data Flow Patterns

### Write Path

```text
┌─────────────────────────────────────────────────────────────────┐
│                        Write Path                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  API Request                                                    │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────┐                                                │
│  │  Validate   │                                                │
│  └──────┬──────┘                                                │
│         │                                                       │
│         ▼                                                       │
│  ┌─────────────┐     ┌─────────────┐                            │
│  │  Write DB   │────►│ Invalidate  │                            │
│  │ (Postgres)  │     │   Cache     │                            │
│  └──────┬──────┘     └─────────────┘                            │
│         │                                                       │
│         ▼                                                       │
│  ┌─────────────┐     ┌─────────────┐                            │
│  │ Emit Event  │────►│  Pub/Sub    │                            │
│  │  (async)    │     │  (Redis)    │                            │
│  └─────────────┘     └─────────────┘                            │
│                              │                                  │
│                              ▼                                  │
│                      ┌─────────────┐                            │
│                      │  Webhooks   │                            │
│                      │ SSE Clients │                            │
│                      └─────────────┘                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Read Path

```text
┌─────────────────────────────────────────────────────────────────┐
│                        Read Path                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  API Request                                                    │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────┐                                                │
│  │ Check Cache │──── Hit ────► Return                           │
│  │   (Redis)   │                                                │
│  └──────┬──────┘                                                │
│         │ Miss                                                  │
│         ▼                                                       │
│  ┌─────────────┐                                                │
│  │  Read DB    │                                                │
│  │ (Postgres)  │                                                │
│  └──────┬──────┘                                                │
│         │                                                       │
│         ▼                                                       │
│  ┌─────────────┐                                                │
│  │ Populate    │                                                │
│  │   Cache     │                                                │
│  └──────┬──────┘                                                │
│         │                                                       │
│         ▼                                                       │
│      Return                                                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 8. Multi-Tenancy

### Isolation Model

```text
┌─────────────────────────────────────────────────────────────────┐
│                     Multi-Tenancy Model                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Approach: Shared Database, Separate Schemas (Row-Level)        │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    PostgreSQL                           │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │  All Tables                                     │    │    │
│  │  │  • project_id column on all tenant data         │    │    │
│  │  │  • RLS policies enforce isolation               │    │    │
│  │  │  • Indexes include project_id                   │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  RLS Policy Example:                                            │
│  ALTER TABLE agents ENABLE ROW LEVEL SECURITY;                  │
│  CREATE POLICY tenant_isolation ON agents                       │
│      USING (project_id = current_setting('app.project_id'));    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Enterprise Option: Schema-per-Tenant

```sql
-- For regulated industries requiring stronger isolation
CREATE SCHEMA tenant_acme;
CREATE TABLE tenant_acme.agents (...);

-- Connection routing
SET search_path TO tenant_acme, public;
```

**Trade-off**: Row-level simpler, schema-per-tenant for compliance.

---

## 9. Backup & Recovery

### Backup Strategy

| Data Type | Method | Frequency | Retention |
|-----------|--------|-----------|-----------|
| PostgreSQL | pg_dump + WAL | Continuous WAL, Daily full | 30 days |
| Redis | RDB + AOF | Every 1 min | 7 days |
| Object Storage | Cross-region replication | Real-time | Permanent |

### Recovery Objectives

| Metric | Target |
|--------|--------|
| **RPO** (data loss) | < 1 minute |
| **RTO** (downtime) | < 15 minutes |

### Disaster Recovery

```yaml
disaster_recovery:
  strategy: active-passive
  primary_region: eu-west-1
  secondary_region: eu-central-1
  replication:
    postgres: streaming_replica
    redis: cross-region_replication
    s3: cross-region_replication
  failover:
    automatic: true
    health_check_interval: 10s
    failover_threshold: 3
```

---

## 10. Migration Strategy

### Schema Migrations

```bash
# Using golang-migrate
migrate -path ./migrations -database $DB_URL up

# Migration file naming
000001_initial_schema.up.sql
000001_initial_schema.down.sql
000002_add_embeddings.up.sql
000002_add_embeddings.down.sql
```

### Zero-Downtime Migrations

```text
1. Add new column (nullable)
2. Deploy code that writes to both
3. Backfill existing data
4. Deploy code that reads from new
5. Add NOT NULL constraint
6. Remove old column (later)
```

---

## 11. Capacity Planning

### Storage Estimates

| Component | Per Agent/Month | 1K Agents | 10K Agents |
|-----------|-----------------|-----------|------------|
| Postgres (metadata) | 1 MB | 1 GB | 10 GB |
| Postgres (messages) | 100 MB | 100 GB | 1 TB |
| Redis (sessions) | 10 MB | 10 GB | 100 GB |
| S3 (logs) | 500 MB | 500 GB | 5 TB |
| Vectors | 50 MB | 50 GB | 500 GB |

### Scaling Triggers

| Metric | Threshold | Action |
|--------|-----------|--------|
| DB CPU | > 70% | Add read replica |
| DB connections | > 80% pool | Add PgBouncer |
| Redis memory | > 75% | Add shard |
| Vector search latency | > 100ms | Migrate to Qdrant |

---

**Previous**: [004-api-design.md](004-api-design.md)  
**Next**: [006-security-governance.md](006-security-governance.md)
