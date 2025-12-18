-- AgentStack Database Initialization
-- This file sets up the initial schema for development

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable pgvector for embeddings (optional, for RAG support)
-- CREATE EXTENSION IF NOT EXISTS vector;

-- Create MLflow database
CREATE DATABASE mlflow;

-- Teams (multi-tenancy root)
CREATE TABLE IF NOT EXISTS teams (
    id TEXT PRIMARY KEY DEFAULT 'team_' || substr(uuid_generate_v4()::text, 1, 8),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    plan TEXT NOT NULL DEFAULT 'free',
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Projects
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY DEFAULT 'prj_' || substr(uuid_generate_v4()::text, 1, 8),
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, slug)
);

CREATE INDEX idx_projects_team_id ON projects(team_id);

-- Agents
CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY DEFAULT 'agt_' || substr(uuid_generate_v4()::text, 1, 8),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'inactive',
    framework TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    source JSONB,
    tags TEXT[] DEFAULT '{}',
    current_deployment_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, slug)
);

CREATE INDEX idx_agents_project_id ON agents(project_id);
CREATE INDEX idx_agents_status ON agents(project_id, status);
CREATE INDEX idx_agents_tags ON agents USING GIN(tags);

-- Deployments
CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY DEFAULT 'dpl_' || substr(uuid_generate_v4()::text, 1, 8),
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    image TEXT,
    config JSONB NOT NULL DEFAULT '{}',
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deployed_at TIMESTAMPTZ,
    terminated_at TIMESTAMPTZ
);

CREATE INDEX idx_deployments_agent_id ON deployments(agent_id);
CREATE INDEX idx_deployments_status ON deployments(status);

-- API Keys
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY DEFAULT 'key_' || substr(uuid_generate_v4()::text, 1, 8),
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_team_id ON api_keys(team_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- Chat Sessions
CREATE TABLE IF NOT EXISTS chat_sessions (
    id TEXT PRIMARY KEY DEFAULT 'ses_' || substr(uuid_generate_v4()::text, 1, 8),
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_sessions_agent_id ON chat_sessions(agent_id);
CREATE INDEX idx_chat_sessions_status ON chat_sessions(status);

-- Chat Messages
CREATE TABLE IF NOT EXISTS chat_messages (
    id TEXT PRIMARY KEY DEFAULT 'msg_' || substr(uuid_generate_v4()::text, 1, 8),
    session_id TEXT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    tool_calls JSONB,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_messages_session_id ON chat_messages(session_id);
CREATE INDEX idx_chat_messages_created_at ON chat_messages(session_id, created_at);

-- Audit Log
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY DEFAULT 'aud_' || substr(uuid_generate_v4()::text, 1, 8),
    team_id TEXT NOT NULL,
    user_id TEXT,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    changes JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_team_id ON audit_logs(team_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Row Level Security (RLS) setup
ALTER TABLE teams ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE deployments ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE chat_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE chat_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- RLS Policies (tenant isolation)
-- These use current_setting('app.tenant_id') which is set per-connection

CREATE POLICY team_isolation ON teams
    USING (id = current_setting('app.tenant_id', true));

CREATE POLICY project_isolation ON projects
    USING (team_id = current_setting('app.tenant_id', true));

CREATE POLICY agent_isolation ON agents
    USING (project_id IN (
        SELECT id FROM projects WHERE team_id = current_setting('app.tenant_id', true)
    ));

CREATE POLICY deployment_isolation ON deployments
    USING (agent_id IN (
        SELECT a.id FROM agents a
        JOIN projects p ON a.project_id = p.id
        WHERE p.team_id = current_setting('app.tenant_id', true)
    ));

CREATE POLICY api_key_isolation ON api_keys
    USING (team_id = current_setting('app.tenant_id', true));

CREATE POLICY chat_session_isolation ON chat_sessions
    USING (agent_id IN (
        SELECT a.id FROM agents a
        JOIN projects p ON a.project_id = p.id
        WHERE p.team_id = current_setting('app.tenant_id', true)
    ));

CREATE POLICY chat_message_isolation ON chat_messages
    USING (session_id IN (
        SELECT cs.id FROM chat_sessions cs
        JOIN agents a ON cs.agent_id = a.id
        JOIN projects p ON a.project_id = p.id
        WHERE p.team_id = current_setting('app.tenant_id', true)
    ));

CREATE POLICY audit_log_isolation ON audit_logs
    USING (team_id = current_setting('app.tenant_id', true));

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers
CREATE TRIGGER update_teams_updated_at BEFORE UPDATE ON teams
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_deployments_updated_at BEFORE UPDATE ON deployments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_chat_sessions_updated_at BEFORE UPDATE ON chat_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Insert default team for development
INSERT INTO teams (id, name, slug, plan) VALUES
    ('team_dev', 'Development Team', 'dev-team', 'enterprise')
ON CONFLICT DO NOTHING;

-- Insert default project for development
INSERT INTO projects (id, team_id, name, slug) VALUES
    ('prj_dev', 'team_dev', 'Default Project', 'default')
ON CONFLICT DO NOTHING;

-- Grant permissions (for the agentstack user)
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO agentstack;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO agentstack;
