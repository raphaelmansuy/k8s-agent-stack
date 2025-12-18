// Package sdk provides the AgentStack Go SDK for programmatic access to the API.
package sdk

import (
	"context"
	"fmt"
	"strconv"
)

// DeploymentsService handles deployment-related operations.
type DeploymentsService struct {
	client *Client
}

// List retrieves a paginated list of deployments.
func (s *DeploymentsService) List(ctx context.Context, opts *ListOptions) (*ListDeploymentsResponse, error) {
	params := make(map[string]string)
	if opts != nil {
		if opts.Page > 0 {
			params["page"] = strconv.Itoa(opts.Page)
		}
		if opts.PageSize > 0 {
			params["page_size"] = strconv.Itoa(opts.PageSize)
		}
		if opts.Sort != "" {
			params["sort"] = opts.Sort
		}
		if opts.Order != "" {
			params["order"] = opts.Order
		}
	}

	path := "/api/v1/deployments" + BuildQueryString(params)
	var response ListDeploymentsResponse
	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Get retrieves a deployment by ID.
func (s *DeploymentsService) Get(ctx context.Context, id string) (*Deployment, error) {
	path := fmt.Sprintf("/api/v1/deployments/%s", id)
	var deployment Deployment
	if err := s.client.Get(ctx, path, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// Create creates a new deployment.
func (s *DeploymentsService) Create(ctx context.Context, req *CreateDeploymentRequest) (*Deployment, error) {
	var deployment Deployment
	if err := s.client.Post(ctx, "/api/v1/deployments", req, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// Update updates an existing deployment.
func (s *DeploymentsService) Update(ctx context.Context, id string, req *UpdateDeploymentRequest) (*Deployment, error) {
	path := fmt.Sprintf("/api/v1/deployments/%s", id)
	var deployment Deployment
	if err := s.client.Patch(ctx, path, req, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// Delete deletes a deployment by ID.
func (s *DeploymentsService) Delete(ctx context.Context, id string) error {
	path := fmt.Sprintf("/api/v1/deployments/%s", id)
	return s.client.Delete(ctx, path)
}

// Scale scales a deployment to the specified number of replicas.
func (s *DeploymentsService) Scale(ctx context.Context, id string, replicas int) (*Deployment, error) {
	path := fmt.Sprintf("/api/v1/deployments/%s/scale", id)
	req := map[string]int{"replicas": replicas}
	var deployment Deployment
	if err := s.client.Post(ctx, path, req, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// Restart restarts a deployment.
func (s *DeploymentsService) Restart(ctx context.Context, id string) (*Deployment, error) {
	path := fmt.Sprintf("/api/v1/deployments/%s/restart", id)
	var deployment Deployment
	if err := s.client.Post(ctx, path, nil, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// Rollback rolls back a deployment to a previous version.
func (s *DeploymentsService) Rollback(ctx context.Context, id, version string) (*Deployment, error) {
	path := fmt.Sprintf("/api/v1/deployments/%s/rollback", id)
	req := map[string]string{"version": version}
	var deployment Deployment
	if err := s.client.Post(ctx, path, req, &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

// GetLogs retrieves logs for a deployment.
func (s *DeploymentsService) GetLogs(ctx context.Context, id string, req *ListLogsRequest) (*ListLogsResponse, error) {
	params := make(map[string]string)
	if req != nil {
		if req.Level != "" {
			params["level"] = req.Level
		}
		if req.Limit > 0 {
			params["limit"] = strconv.Itoa(req.Limit)
		}
		if !req.StartTime.IsZero() {
			params["start_time"] = req.StartTime.Format("2006-01-02T15:04:05Z07:00")
		}
		if !req.EndTime.IsZero() {
			params["end_time"] = req.EndTime.Format("2006-01-02T15:04:05Z07:00")
		}
	}

	path := fmt.Sprintf("/api/v1/deployments/%s/logs", id) + BuildQueryString(params)
	var response ListLogsResponse
	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// ListByAgent retrieves deployments for a specific agent.
func (s *DeploymentsService) ListByAgent(ctx context.Context, agentID string, opts *ListOptions) (*ListDeploymentsResponse, error) {
	params := map[string]string{
		"agent_id": agentID,
	}
	if opts != nil {
		if opts.Page > 0 {
			params["page"] = strconv.Itoa(opts.Page)
		}
		if opts.PageSize > 0 {
			params["page_size"] = strconv.Itoa(opts.PageSize)
		}
	}

	path := "/api/v1/deployments" + BuildQueryString(params)
	var response ListDeploymentsResponse
	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Status returns the status of a deployment.
func (s *DeploymentsService) Status(ctx context.Context, id string) (string, error) {
	deployment, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return deployment.Status, nil
}
