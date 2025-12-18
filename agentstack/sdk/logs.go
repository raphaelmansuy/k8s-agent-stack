// Package sdk provides the AgentStack Go SDK for programmatic access to the API.
package sdk

import (
	"context"
	"fmt"
	"strconv"
)

// LogsService handles log-related operations.
type LogsService struct {
	client *Client
}

// NewLogsService creates a new logs service.
func NewLogsService(client *Client) *LogsService {
	return &LogsService{client: client}
}

// List retrieves logs based on the request parameters.
func (s *LogsService) List(ctx context.Context, req *ListLogsRequest) (*ListLogsResponse, error) {
	params := make(map[string]string)
	if req != nil {
		if req.AgentID != "" {
			params["agent_id"] = req.AgentID
		}
		if req.DeploymentID != "" {
			params["deployment_id"] = req.DeploymentID
		}
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

	path := "/api/v1/logs" + BuildQueryString(params)
	var response ListLogsResponse
	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetAgentLogs retrieves logs for a specific agent.
func (s *LogsService) GetAgentLogs(ctx context.Context, agentID string, limit int) (*ListLogsResponse, error) {
	return s.List(ctx, &ListLogsRequest{
		AgentID: agentID,
		Limit:   limit,
	})
}

// GetDeploymentLogs retrieves logs for a specific deployment.
func (s *LogsService) GetDeploymentLogs(ctx context.Context, deploymentID string, limit int) (*ListLogsResponse, error) {
	return s.List(ctx, &ListLogsRequest{
		DeploymentID: deploymentID,
		Limit:        limit,
	})
}

// Tail retrieves the latest logs for an agent or deployment.
func (s *LogsService) Tail(ctx context.Context, agentID, deploymentID string, lines int) ([]LogEntry, error) {
	req := &ListLogsRequest{
		AgentID:      agentID,
		DeploymentID: deploymentID,
		Limit:        lines,
	}
	resp, err := s.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Logs, nil
}

// Search searches logs by message content.
func (s *LogsService) Search(ctx context.Context, query string, req *ListLogsRequest) (*ListLogsResponse, error) {
	params := make(map[string]string)
	params["query"] = query

	if req != nil {
		if req.AgentID != "" {
			params["agent_id"] = req.AgentID
		}
		if req.DeploymentID != "" {
			params["deployment_id"] = req.DeploymentID
		}
		if req.Level != "" {
			params["level"] = req.Level
		}
		if req.Limit > 0 {
			params["limit"] = strconv.Itoa(req.Limit)
		}
	}

	path := "/api/v1/logs/search" + BuildQueryString(params)
	var response ListLogsResponse
	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetByLevel retrieves logs filtered by level.
func (s *LogsService) GetByLevel(ctx context.Context, level string, limit int) (*ListLogsResponse, error) {
	return s.List(ctx, &ListLogsRequest{
		Level: level,
		Limit: limit,
	})
}

// GetErrors retrieves error-level logs.
func (s *LogsService) GetErrors(ctx context.Context, limit int) (*ListLogsResponse, error) {
	return s.GetByLevel(ctx, "error", limit)
}

// GetWarnings retrieves warning-level logs.
func (s *LogsService) GetWarnings(ctx context.Context, limit int) (*ListLogsResponse, error) {
	return s.GetByLevel(ctx, "warning", limit)
}

// Export exports logs to a specified format.
func (s *LogsService) Export(ctx context.Context, req *ListLogsRequest, format string) ([]byte, error) {
	params := make(map[string]string)
	params["format"] = format
	if req != nil {
		if req.AgentID != "" {
			params["agent_id"] = req.AgentID
		}
		if req.DeploymentID != "" {
			params["deployment_id"] = req.DeploymentID
		}
		if req.Level != "" {
			params["level"] = req.Level
		}
	}

	path := fmt.Sprintf("/api/v1/logs/export%s", BuildQueryString(params))
	httpReq, err := s.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkError(resp); err != nil {
		return nil, err
	}

	var data []byte
	_, err = resp.Body.Read(data)
	return data, err
}
