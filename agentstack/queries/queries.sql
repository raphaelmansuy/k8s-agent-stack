-- name: GetTeam :one
SELECT * FROM teams
WHERE id = $1 LIMIT 1;

-- name: GetTeamBySlug :one
SELECT * FROM teams
WHERE slug = $1 LIMIT 1;

-- name: ListTeams :many
SELECT * FROM teams
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateTeam :one
INSERT INTO teams (name, slug, plan, settings)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateTeam :one
UPDATE teams
SET name = COALESCE($2, name),
    settings = COALESCE($3, settings)
WHERE id = $1
RETURNING *;

-- name: DeleteTeam :exec
DELETE FROM teams
WHERE id = $1;

-- name: GetProject :one
SELECT * FROM projects
WHERE id = $1 LIMIT 1;

-- name: GetProjectBySlug :one
SELECT * FROM projects
WHERE team_id = $1 AND slug = $2 LIMIT 1;

-- name: ListProjects :many
SELECT * FROM projects
WHERE team_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateProject :one
INSERT INTO projects (team_id, name, slug, settings)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET name = COALESCE($2, name),
    settings = COALESCE($3, settings)
WHERE id = $1
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;

-- name: GetAgent :one
SELECT * FROM agents
WHERE id = $1 LIMIT 1;

-- name: GetAgentBySlug :one
SELECT * FROM agents
WHERE project_id = $1 AND slug = $2 LIMIT 1;

-- name: ListAgents :many
SELECT * FROM agents
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAgentsByStatus :many
SELECT * FROM agents
WHERE project_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAgentsByTag :many
SELECT * FROM agents
WHERE project_id = $1 AND $2 = ANY(tags)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CreateAgent :one
INSERT INTO agents (project_id, name, slug, description, framework, config, tags)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateAgent :one
UPDATE agents
SET name = COALESCE($2, name),
    description = COALESCE($3, description),
    config = COALESCE($4, config),
    tags = COALESCE($5, tags)
WHERE id = $1
RETURNING *;

-- name: UpdateAgentStatus :one
UPDATE agents
SET status = $2,
    current_deployment_id = $3
WHERE id = $1
RETURNING *;

-- name: DeleteAgent :exec
DELETE FROM agents
WHERE id = $1;

-- name: GetDeployment :one
SELECT * FROM deployments
WHERE id = $1 LIMIT 1;

-- name: ListDeployments :many
SELECT * FROM deployments
WHERE agent_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateDeployment :one
INSERT INTO deployments (agent_id, version, status, image, config)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateDeploymentStatus :one
UPDATE deployments
SET status = $2,
    error = $3,
    deployed_at = CASE WHEN $2 = 'running' THEN NOW() ELSE deployed_at END,
    terminated_at = CASE WHEN $2 IN ('stopped', 'failed') THEN NOW() ELSE terminated_at END
WHERE id = $1
RETURNING *;

-- name: GetAPIKey :one
SELECT * FROM api_keys
WHERE key_hash = $1 AND (expires_at IS NULL OR expires_at > NOW()) LIMIT 1;

-- name: ListAPIKeys :many
SELECT id, team_id, project_id, name, key_prefix, scopes, last_used_at, expires_at, created_at
FROM api_keys
WHERE team_id = $1
ORDER BY created_at DESC;

-- name: CreateAPIKey :one
INSERT INTO api_keys (team_id, project_id, name, key_hash, key_prefix, scopes, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys
SET last_used_at = NOW()
WHERE id = $1;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys
WHERE id = $1;

-- name: GetChatSession :one
SELECT * FROM chat_sessions
WHERE id = $1 LIMIT 1;

-- name: ListChatSessions :many
SELECT * FROM chat_sessions
WHERE agent_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateChatSession :one
INSERT INTO chat_sessions (agent_id, status, metadata)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateChatSessionStatus :one
UPDATE chat_sessions
SET status = $2
WHERE id = $1
RETURNING *;

-- name: ListChatMessages :many
SELECT * FROM chat_messages
WHERE session_id = $1
ORDER BY created_at ASC;

-- name: CreateChatMessage :one
INSERT INTO chat_messages (session_id, role, content, tool_calls, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateAuditLog :one
INSERT INTO audit_logs (team_id, user_id, action, resource_type, resource_id, changes, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE team_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetQuota :one
SELECT q.*, COALESCE(qu.current_value, 0) as current_value
FROM quotas q
LEFT JOIN quota_usage qu ON q.id = qu.quota_id
WHERE q.team_id = $1 AND q.quota_type = $2 AND (q.project_id = $3 OR q.project_id IS NULL)
ORDER BY q.project_id NULLS LAST
LIMIT 1;

-- name: ListQuotas :many
SELECT q.*, COALESCE(qu.current_value, 0) as current_value
FROM quotas q
LEFT JOIN quota_usage qu ON q.id = qu.quota_id
WHERE q.team_id = $1;

-- name: SetQuota :one
INSERT INTO quotas (team_id, project_id, quota_type, limit_value, period)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (team_id, project_id, quota_type) WHERE project_id IS NOT NULL DO UPDATE
SET limit_value = EXCLUDED.limit_value,
    period = EXCLUDED.period,
    updated_at = NOW()
RETURNING *;

-- name: SetTeamQuota :one
INSERT INTO quotas (team_id, quota_type, limit_value, period)
VALUES ($1, $2, $3, $4)
ON CONFLICT (team_id, quota_type) WHERE project_id IS NULL DO UPDATE
SET limit_value = EXCLUDED.limit_value,
    period = EXCLUDED.period,
    updated_at = NOW()
RETURNING *;

-- name: GetQuotaUsage :one
SELECT * FROM quota_usage
WHERE quota_id = $1;

-- name: IncrementQuotaUsageByID :exec
INSERT INTO quota_usage (quota_id, current_value, updated_at)
VALUES (sqlc.arg(quota_id), GREATEST(0, sqlc.arg(amount)), NOW())
ON CONFLICT (quota_id) DO UPDATE
SET current_value = GREATEST(0, quota_usage.current_value + sqlc.arg(amount)),
    updated_at = NOW();

-- name: IncrementQuotaUsage :exec
SELECT increment_quota_usage($1, $2, $3, $4);

-- name: GetRole :one
SELECT * FROM roles
WHERE id = $1 LIMIT 1;

-- name: ListRoles :many
SELECT * FROM roles
WHERE team_id = $1 OR team_id IS NULL
ORDER BY is_system DESC, name ASC;

-- name: CreateRole :one
INSERT INTO roles (team_id, name, description, permissions, is_system)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateRole :one
UPDATE roles
SET name = COALESCE($2, name),
    description = COALESCE($3, description),
    permissions = COALESCE($4, permissions),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteRole :exec
DELETE FROM roles
WHERE id = $1;

-- name: GetRoleBinding :one
SELECT * FROM role_bindings
WHERE id = $1 LIMIT 1;

-- name: ListRoleBindingsByUser :many
SELECT rb.*, r.name as role_name, r.permissions
FROM role_bindings rb
LEFT JOIN roles r ON rb.role_id = r.id
WHERE rb.user_id = $1;

-- name: ListRoleBindingsByScope :many
SELECT rb.*, r.name as role_name, r.permissions
FROM role_bindings rb
LEFT JOIN roles r ON rb.role_id = r.id
WHERE rb.scope_type = $1 AND (rb.scope_team_id = $2 OR rb.scope_project_id = $3);

-- name: CreateRoleBinding :one
INSERT INTO role_bindings (user_id, role_id, scope_type, scope_team_id, scope_project_id, granted_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: DeleteRoleBinding :exec
DELETE FROM role_bindings
WHERE id = $1;

-- name: CreateTrace :one
INSERT INTO traces (agent_id, session_id, input, output, latency_ms, tokens_in, tokens_out, steps, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetTrace :one
SELECT * FROM traces
WHERE id = $1 LIMIT 1;

-- name: ListTraces :many
SELECT * FROM traces
WHERE (agent_id = $1 OR $1 = '')
  AND (session_id = $2 OR $2 = '')
  AND (created_at >= $3 OR $3 = '0001-01-01 00:00:00+00')
  AND (created_at <= $4 OR $4 = '0001-01-01 00:00:00+00')
ORDER BY created_at DESC
LIMIT $5;

-- name: CreateFeedback :one
INSERT INTO feedback (trace_id, rating, comment, tags)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListFeedbackByTrace :many
SELECT * FROM feedback
WHERE trace_id = $1
ORDER BY created_at DESC;
