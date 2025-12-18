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
