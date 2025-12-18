// Package routes registers API routes
package routes

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/raphaelmansuy/agentstack/internal/api/handlers"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
)

// RegisterHealth registers health check endpoints.
func RegisterHealth(api huma.API) {
	handlers.RegisterHealthRoutes(api)
}

// RegisterAgents registers agent management endpoints.
func RegisterAgents(api huma.API, db *database.Pool) {
	handlers.RegisterAgentRoutes(api, db)
}

// RegisterProjects registers project management endpoints.
func RegisterProjects(api huma.API, db *database.Pool) {
	handlers.RegisterProjectRoutes(api, db)
}

// RegisterChat registers chat endpoints.
func RegisterChat(api huma.API, db *database.Pool, redis *cache.Client) {
	handlers.RegisterChatRoutes(api, db, redis)
}
