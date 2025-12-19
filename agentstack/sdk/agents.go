// Package sdk provides the AgentStack Go SDK for programmatic access to the API.
/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sdk

import (
	"context"
	"fmt"
	"strconv"
)

// AgentsService handles agent-related operations.
type AgentsService struct {
	client *Client
}

// List retrieves a paginated list of agents.
func (s *AgentsService) List(ctx context.Context, opts *ListOptions) (*ListAgentsResponse, error) {
	params := make(map[string]string)
	if opts != nil {
		if opts.ProjectID != "" {
			params["project_id"] = opts.ProjectID
		}
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

	path := "/api/v1/agents" + BuildQueryString(params)
	var response ListAgentsResponse
	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Get retrieves an agent by ID.
func (s *AgentsService) Get(ctx context.Context, id string) (*Agent, error) {
	path := fmt.Sprintf("/api/v1/agents/%s", id)
	var agent Agent
	if err := s.client.Get(ctx, path, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Create creates a new agent.
func (s *AgentsService) Create(ctx context.Context, req *CreateAgentRequest) (*Agent, error) {
	var agent Agent
	if err := s.client.Post(ctx, "/api/v1/agents", req, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Update updates an existing agent.
func (s *AgentsService) Update(ctx context.Context, id string, req *UpdateAgentRequest) (*Agent, error) {
	path := fmt.Sprintf("/api/v1/agents/%s", id)
	var agent Agent
	if err := s.client.Patch(ctx, path, req, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Delete deletes an agent by ID.
func (s *AgentsService) Delete(ctx context.Context, id string) error {
	path := fmt.Sprintf("/api/v1/agents/%s", id)
	return s.client.Delete(ctx, path)
}

// GetByName retrieves an agent by name within a project.
func (s *AgentsService) GetByName(ctx context.Context, projectID, name string) (*Agent, error) {
	params := map[string]string{
		"project_id": projectID,
		"name":       name,
	}
	path := "/api/v1/agents" + BuildQueryString(params)
	var response ListAgentsResponse
	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}
	if len(response.Agents) == 0 {
		return nil, &APIError{
			StatusCode: 404,
			Code:       "agent_not_found",
			Message:    fmt.Sprintf("agent %s not found in project %s", name, projectID),
		}
	}
	return &response.Agents[0], nil
}

// ListByProject retrieves agents for a specific project.
func (s *AgentsService) ListByProject(ctx context.Context, projectID string, opts *ListOptions) (*ListAgentsResponse, error) {
	params := map[string]string{
		"project_id": projectID,
	}
	if opts != nil {
		if opts.Page > 0 {
			params["page"] = strconv.Itoa(opts.Page)
		}
		if opts.PageSize > 0 {
			params["page_size"] = strconv.Itoa(opts.PageSize)
		}
	}

	path := "/api/v1/agents" + BuildQueryString(params)
	var response ListAgentsResponse
	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
