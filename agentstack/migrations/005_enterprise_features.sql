-- Phase 5: Enterprise Features Migration
-- RBAC, Quotas, and enhanced Audit logging

-- ============================================================================
-- RBAC Tables
-- ============================================================================

-- Custom Roles (beyond system roles)
CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY DEFAULT 'role_' || substr(uuid_generate_v4()::text, 1, 8),
    team_id TEXT REFERENCES teams(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    permissions JSONB NOT NULL DEFAULT '[]',
    is_system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, name)
);

CREATE INDEX idx_roles_team_id ON roles(team_id);

-- Role Bindings (user-role assignments)
CREATE TABLE IF NOT EXISTS role_bindings (
    id TEXT PRIMARY KEY DEFAULT 'rb_' || substr(uuid_generate_v4()::text, 1, 8),
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    scope_type TEXT NOT NULL DEFAULT 'team', -- global, team, project
    scope_team_id TEXT REFERENCES teams(id) ON DELETE CASCADE,
    scope_project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    granted_by TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_scope CHECK (
        (scope_type = 'global') OR
        (scope_type = 'team' AND scope_team_id IS NOT NULL) OR
        (scope_type = 'project' AND scope_project_id IS NOT NULL)
    )
);

CREATE INDEX idx_role_bindings_user_id ON role_bindings(user_id);
CREATE INDEX idx_role_bindings_role_id ON role_bindings(role_id);
CREATE INDEX idx_role_bindings_scope_team ON role_bindings(scope_team_id);
CREATE INDEX idx_role_bindings_scope_project ON role_bindings(scope_project_id);
CREATE INDEX idx_role_bindings_expires_at ON role_bindings(expires_at) WHERE expires_at IS NOT NULL;

-- RLS for role tables
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE role_bindings ENABLE ROW LEVEL SECURITY;

CREATE POLICY role_isolation ON roles
    USING (team_id = current_setting('app.tenant_id', true) OR team_id IS NULL);

CREATE POLICY role_binding_isolation ON role_bindings
    USING (scope_team_id = current_setting('app.tenant_id', true) OR scope_team_id IS NULL);

-- ============================================================================
-- Quota Tables
-- ============================================================================

-- Quotas (limits per team/project)
CREATE TABLE IF NOT EXISTS quotas (
    id TEXT PRIMARY KEY DEFAULT 'quota_' || substr(uuid_generate_v4()::text, 1, 8),
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    quota_type TEXT NOT NULL,
    limit_value BIGINT NOT NULL,
    period TEXT, -- NULL for absolute, 'hour'/'day'/'month' for rate limits
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_quotas_team_project_type ON quotas(team_id, project_id, quota_type) WHERE project_id IS NOT NULL;
CREATE UNIQUE INDEX idx_quotas_team_type ON quotas(team_id, quota_type) WHERE project_id IS NULL;

CREATE INDEX idx_quotas_team_id ON quotas(team_id);
CREATE INDEX idx_quotas_project_id ON quotas(project_id);
CREATE INDEX idx_quotas_type ON quotas(quota_type);

-- Quota Usage (current consumption)
CREATE TABLE IF NOT EXISTS quota_usage (
    quota_id TEXT PRIMARY KEY REFERENCES quotas(id) ON DELETE CASCADE,
    current_value BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RLS for quota tables
ALTER TABLE quotas ENABLE ROW LEVEL SECURITY;
ALTER TABLE quota_usage ENABLE ROW LEVEL SECURITY;

CREATE POLICY quota_isolation ON quotas
    USING (team_id = current_setting('app.tenant_id', true));

CREATE POLICY quota_usage_isolation ON quota_usage
    USING (quota_id IN (
        SELECT id FROM quotas WHERE team_id = current_setting('app.tenant_id', true)
    ));

-- ============================================================================
-- Enhanced Audit Log
-- ============================================================================

-- Add new columns to audit_logs if they don't exist
DO $$ 
BEGIN
    -- Add event_type column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'audit_logs' AND column_name = 'event_type') THEN
        ALTER TABLE audit_logs ADD COLUMN event_type TEXT;
    END IF;
    
    -- Add actor_type column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'audit_logs' AND column_name = 'actor_type') THEN
        ALTER TABLE audit_logs ADD COLUMN actor_type TEXT DEFAULT 'user';
    END IF;
    
    -- Add actor_email column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'audit_logs' AND column_name = 'actor_email') THEN
        ALTER TABLE audit_logs ADD COLUMN actor_email TEXT;
    END IF;
    
    -- Add project_id column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'audit_logs' AND column_name = 'project_id') THEN
        ALTER TABLE audit_logs ADD COLUMN project_id TEXT REFERENCES projects(id) ON DELETE SET NULL;
    END IF;
    
    -- Add result column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'audit_logs' AND column_name = 'result') THEN
        ALTER TABLE audit_logs ADD COLUMN result TEXT DEFAULT 'success';
    END IF;
    
    -- Add request_id column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'audit_logs' AND column_name = 'request_id') THEN
        ALTER TABLE audit_logs ADD COLUMN request_id TEXT;
    END IF;
    
    -- Add details column (JSONB for structured data)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'audit_logs' AND column_name = 'details') THEN
        ALTER TABLE audit_logs ADD COLUMN details JSONB;
    END IF;
END $$;

-- Create additional indexes for audit queries
CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_result ON audit_logs(result);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_project_id ON audit_logs(project_id);

-- ============================================================================
-- Updated_at Triggers
-- ============================================================================

CREATE TRIGGER update_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_quotas_updated_at
    BEFORE UPDATE ON quotas
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_quota_usage_updated_at
    BEFORE UPDATE ON quota_usage
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- ============================================================================
-- Functions for RBAC
-- ============================================================================

-- Function to check if a user has a permission
CREATE OR REPLACE FUNCTION check_permission(
    p_user_id TEXT,
    p_resource TEXT,
    p_action TEXT,
    p_scope_type TEXT DEFAULT 'team',
    p_scope_id TEXT DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    v_has_permission BOOLEAN := false;
    v_binding RECORD;
    v_role RECORD;
    v_perm JSONB;
BEGIN
    -- Get all active role bindings for the user
    FOR v_binding IN
        SELECT rb.*, r.permissions
        FROM role_bindings rb
        LEFT JOIN roles r ON rb.role_id = r.id
        WHERE rb.user_id = p_user_id
        AND (rb.expires_at IS NULL OR rb.expires_at > NOW())
        AND (
            rb.scope_type = 'global' OR
            (rb.scope_type = p_scope_type AND 
             CASE p_scope_type
                WHEN 'team' THEN rb.scope_team_id = p_scope_id
                WHEN 'project' THEN rb.scope_project_id = p_scope_id
                ELSE true
             END)
        )
    LOOP
        -- Check each permission in the role
        FOR v_perm IN SELECT * FROM jsonb_array_elements(COALESCE(v_binding.permissions, '[]'::jsonb))
        LOOP
            IF (v_perm->>'resource' = p_resource OR v_perm->>'resource' = '*') AND
               (v_perm->>'action' = p_action OR v_perm->>'action' = 'manage' OR v_perm->>'action' = '*') THEN
                v_has_permission := true;
                EXIT;
            END IF;
        END LOOP;
        
        IF v_has_permission THEN
            EXIT;
        END IF;
    END LOOP;
    
    RETURN v_has_permission;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================================
-- Functions for Quotas
-- ============================================================================

-- Function to check if quota allows an operation
CREATE OR REPLACE FUNCTION check_quota(
    p_team_id TEXT,
    p_quota_type TEXT,
    p_amount BIGINT DEFAULT 1,
    p_project_id TEXT DEFAULT NULL
) RETURNS TABLE(allowed BOOLEAN, current_val BIGINT, limit_val BIGINT, remaining BIGINT) AS $$
DECLARE
    v_quota RECORD;
    v_usage BIGINT;
BEGIN
    -- Get quota (project-specific first, then team-level)
    SELECT q.*, COALESCE(qu.current_value, 0) as usage
    INTO v_quota
    FROM quotas q
    LEFT JOIN quota_usage qu ON q.id = qu.quota_id
    WHERE q.team_id = p_team_id
    AND q.quota_type = p_quota_type
    AND (
        (p_project_id IS NOT NULL AND q.project_id = p_project_id) OR
        (q.project_id IS NULL)
    )
    ORDER BY q.project_id NULLS LAST
    LIMIT 1;
    
    -- No quota found = unlimited
    IF v_quota IS NULL THEN
        RETURN QUERY SELECT true, 0::BIGINT, -1::BIGINT, -1::BIGINT;
        RETURN;
    END IF;
    
    -- Unlimited quota (-1)
    IF v_quota.limit_value < 0 THEN
        RETURN QUERY SELECT true, v_quota.usage, -1::BIGINT, -1::BIGINT;
        RETURN;
    END IF;
    
    -- Check if allowed
    RETURN QUERY SELECT 
        (v_quota.usage + p_amount) <= v_quota.limit_value,
        v_quota.usage,
        v_quota.limit_value,
        v_quota.limit_value - v_quota.usage;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to increment quota usage
CREATE OR REPLACE FUNCTION increment_quota_usage(
    p_team_id TEXT,
    p_quota_type TEXT,
    p_amount BIGINT DEFAULT 1,
    p_project_id TEXT DEFAULT NULL
) RETURNS VOID AS $$
DECLARE
    v_quota_id TEXT;
BEGIN
    -- Find the quota
    SELECT id INTO v_quota_id
    FROM quotas
    WHERE team_id = p_team_id
    AND quota_type = p_quota_type
    AND (
        (p_project_id IS NOT NULL AND project_id = p_project_id) OR
        (project_id IS NULL)
    )
    ORDER BY project_id NULLS LAST
    LIMIT 1;
    
    IF v_quota_id IS NOT NULL THEN
        -- Upsert usage
        INSERT INTO quota_usage (quota_id, current_value, updated_at)
        VALUES (v_quota_id, GREATEST(0, p_amount), NOW())
        ON CONFLICT (quota_id) DO UPDATE
        SET current_value = GREATEST(0, quota_usage.current_value + p_amount),
            updated_at = NOW();
    END IF;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================================
-- Sample Data (Development Only)
-- ============================================================================

-- Insert default quotas for the demo team
DO $$
DECLARE
    v_team_id TEXT;
BEGIN
    -- Get the first team
    SELECT id INTO v_team_id FROM teams LIMIT 1;
    
    IF v_team_id IS NOT NULL THEN
        -- Insert default quotas (Free plan)
        INSERT INTO quotas (team_id, quota_type, limit_value, period) VALUES
            (v_team_id, 'agents', 3, NULL),
            (v_team_id, 'deployments', 10, NULL),
            (v_team_id, 'storage_bytes', 104857600, NULL), -- 100MB
            (v_team_id, 'concurrent_chats', 5, NULL),
            (v_team_id, 'api_requests', 1000, 'hour'),
            (v_team_id, 'chat_messages', 500, 'day'),
            (v_team_id, 'tokens', 100000, 'month')
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'Phase 5 migration completed successfully: RBAC, Quotas, and enhanced Audit logging';
END $$;
