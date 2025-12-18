// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/raphaelmansuy/agentstack/internal/infrastructure/cache"
	"github.com/raphaelmansuy/agentstack/internal/infrastructure/database"
)

// HealthInput is the input for health check endpoints.
type HealthInput struct{}

// HealthOutput is the output for health check endpoints.
type HealthOutput struct {
	Body struct {
		Status  string            `json:"status" example:"healthy" doc:"Health status"`
		Version string            `json:"version" example:"1.0.0" doc:"API version"`
		Checks  map[string]string `json:"checks,omitempty" doc:"Individual health checks"`
	}
}

// LivenessOutput is the output for liveness probe.
type LivenessOutput struct {
	Body struct {
		Status string `json:"status" example:"ok" doc:"Liveness status"`
	}
}

// ReadinessOutput is the output for readiness probe.
type ReadinessOutput struct {
	Body struct {
		Status string `json:"status" example:"ready" doc:"Readiness status"`
		Ready  bool   `json:"ready" doc:"Whether the service is ready"`
	}
}

// RegisterHealthRoutes registers health check routes.
func RegisterHealthRoutes(api huma.API) {
	// Liveness probe
	huma.Get(api, "/healthz", func(ctx context.Context, input *HealthInput) (*LivenessOutput, error) {
		return &LivenessOutput{
			Body: struct {
				Status string `json:"status" example:"ok" doc:"Liveness status"`
			}{
				Status: "ok",
			},
		}, nil
	})

	// Readiness probe
	huma.Get(api, "/readyz", func(ctx context.Context, input *HealthInput) (*ReadinessOutput, error) {
		return &ReadinessOutput{
			Body: struct {
				Status string `json:"status" example:"ready" doc:"Readiness status"`
				Ready  bool   `json:"ready" doc:"Whether the service is ready"`
			}{
				Status: "ready",
				Ready:  true,
			},
		}, nil
	})

	// Detailed health check
	huma.Get(api, "/health", func(ctx context.Context, input *HealthInput) (*HealthOutput, error) {
		checks := make(map[string]string)
		checks["api"] = "healthy"

		return &HealthOutput{
			Body: struct {
				Status  string            `json:"status" example:"healthy" doc:"Health status"`
				Version string            `json:"version" example:"1.0.0" doc:"API version"`
				Checks  map[string]string `json:"checks,omitempty" doc:"Individual health checks"`
			}{
				Status:  "healthy",
				Version: "1.0.0",
				Checks:  checks,
			},
		}, nil
	})
}

// RegisterHealthRoutesWithDeps registers health routes with database and cache dependencies.
func RegisterHealthRoutesWithDeps(api huma.API, db *database.Pool, redis *cache.Client) {
	huma.Get(api, "/health/detailed", func(ctx context.Context, input *HealthInput) (*HealthOutput, error) {
		checks := make(map[string]string)
		overallStatus := "healthy"

		// Check database
		if db != nil {
			if err := db.HealthCheck(ctx); err != nil {
				checks["database"] = "unhealthy: " + err.Error()
				overallStatus = "degraded"
			} else {
				checks["database"] = "healthy"
			}
		} else {
			checks["database"] = "not configured"
		}

		// Check Redis
		if redis != nil {
			if err := redis.HealthCheck(ctx); err != nil {
				checks["redis"] = "unhealthy: " + err.Error()
				overallStatus = "degraded"
			} else {
				checks["redis"] = "healthy"
			}
		} else {
			checks["redis"] = "not configured"
		}

		return &HealthOutput{
			Body: struct {
				Status  string            `json:"status" example:"healthy" doc:"Health status"`
				Version string            `json:"version" example:"1.0.0" doc:"API version"`
				Checks  map[string]string `json:"checks,omitempty" doc:"Individual health checks"`
			}{
				Status:  overallStatus,
				Version: "1.0.0",
				Checks:  checks,
			},
		}, nil
	})
}
