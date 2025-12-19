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

// ProjectsService handles project-related operations.
type ProjectsService struct {
	client *Client
}

// List retrieves a paginated list of projects.
func (s *ProjectsService) List(ctx context.Context, opts *ListOptions) (*ListProjectsResponse, error) {
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

	path := "/api/v1/projects" + BuildQueryString(params)
	var response ListProjectsResponse
	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Get retrieves a project by ID.
func (s *ProjectsService) Get(ctx context.Context, id string) (*Project, error) {
	path := fmt.Sprintf("/api/v1/projects/%s", id)
	var project Project
	if err := s.client.Get(ctx, path, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// Create creates a new project.
func (s *ProjectsService) Create(ctx context.Context, req *CreateProjectRequest) (*Project, error) {
	var project Project
	if err := s.client.Post(ctx, "/api/v1/projects", req, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// Update updates an existing project.
func (s *ProjectsService) Update(ctx context.Context, id string, req *UpdateProjectRequest) (*Project, error) {
	path := fmt.Sprintf("/api/v1/projects/%s", id)
	var project Project
	if err := s.client.Patch(ctx, path, req, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// Delete deletes a project by ID.
func (s *ProjectsService) Delete(ctx context.Context, id string) error {
	path := fmt.Sprintf("/api/v1/projects/%s", id)
	return s.client.Delete(ctx, path)
}

// GetByName retrieves a project by name.
func (s *ProjectsService) GetByName(ctx context.Context, name string) (*Project, error) {
	params := map[string]string{
		"name": name,
	}
	path := "/api/v1/projects" + BuildQueryString(params)
	var response ListProjectsResponse
	if err := s.client.Get(ctx, path, &response); err != nil {
		return nil, err
	}
	if len(response.Projects) == 0 {
		return nil, &APIError{
			StatusCode: 404,
			Code:       "project_not_found",
			Message:    fmt.Sprintf("project %s not found", name),
		}
	}
	return &response.Projects[0], nil
}
