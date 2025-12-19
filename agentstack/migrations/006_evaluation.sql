-- Phase 6: Evaluation Migration
-- Traces and Feedback for agent interactions

CREATE TABLE IF NOT EXISTS traces (
    id TEXT PRIMARY KEY DEFAULT 'trc_' || substr(uuid_generate_v4()::text, 1, 8),
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    input TEXT NOT NULL,
    output TEXT NOT NULL,
    latency_ms BIGINT NOT NULL,
    tokens_in INTEGER NOT NULL DEFAULT 0,
    tokens_out INTEGER NOT NULL DEFAULT 0,
    steps JSONB NOT NULL DEFAULT '[]',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_traces_agent_id ON traces(agent_id);
CREATE INDEX idx_traces_session_id ON traces(session_id);
CREATE INDEX idx_traces_created_at ON traces(created_at DESC);

CREATE TABLE IF NOT EXISTS feedback (
    id TEXT PRIMARY KEY DEFAULT 'fb_' || substr(uuid_generate_v4()::text, 1, 8),
    trace_id TEXT NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_trace_id ON feedback(trace_id);
CREATE INDEX idx_feedback_created_at ON feedback(created_at DESC);

-- RLS for evaluation tables
ALTER TABLE traces ENABLE ROW LEVEL SECURITY;
ALTER TABLE feedback ENABLE ROW LEVEL SECURITY;

CREATE POLICY trace_isolation ON traces
    USING (agent_id IN (
        SELECT a.id FROM agents a
        JOIN projects p ON a.project_id = p.id
        WHERE p.team_id = current_setting('app.tenant_id', true)
    ));

CREATE POLICY feedback_isolation ON feedback
    USING (trace_id IN (
        SELECT id FROM traces WHERE agent_id IN (
            SELECT a.id FROM agents a
            JOIN projects p ON a.project_id = p.id
            WHERE p.team_id = current_setting('app.tenant_id', true)
        )
    ));
