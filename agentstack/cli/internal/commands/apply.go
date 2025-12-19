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

package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
)

type Manifest struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   ManifestMetadata       `yaml:"metadata"`
	Spec       map[string]interface{} `yaml:"spec"`
}

type ManifestMetadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

func newApplyCmd() *cobra.Command {
	var filename string
	cmd := &cobra.Command{
		Use:   "apply -f <filename>",
		Short: "Apply a configuration to a resource by filename",
		Example: `  # Apply a single agent manifest
  agentctl apply -f agent.yaml

  # Apply multiple resources from a single file
  agentctl apply -f stack.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if filename == "" {
				return fmt.Errorf("-f is required")
			}
			data, err := os.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			decoder := yaml.NewDecoder(bytes.NewReader(data))
			ctx := cmd.Context()

			var created, updated, failed int
			for {
				var manifest Manifest
				err := decoder.Decode(&manifest)
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return fmt.Errorf("failed to parse manifest: %w", err)
				}

				if manifest.Kind == "" {
					continue
				}

				var action string
				var applyErr error
				switch manifest.Kind {
				case "Agent":
					action, applyErr = applyAgentWithAction(ctx, manifest)
				case "Deployment":
					action, applyErr = applyDeploymentWithAction(ctx, manifest)
				default:
					applyErr = fmt.Errorf("unsupported kind: %s", manifest.Kind)
				}

				if applyErr != nil {
					failed++
				} else {
					switch action {
					case "created":
						created++
					case "updated":
						updated++
					}
				}
			}

			if created > 0 || updated > 0 || failed > 0 {
				fmt.Println()
				fmt.Println("Summary:")
			}
			if created > 0 {
				fmt.Printf("  %sCreated: %d%s\n", output.ColorGreen, created, output.ColorReset)
			}
			if updated > 0 {
				fmt.Printf("  %sUpdated: %d%s\n", output.ColorBlue, updated, output.ColorReset)
			}
			if failed > 0 {
				fmt.Printf("  %sFailed:  %d%s\n", output.ColorRed, failed, output.ColorReset)
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&filename, "file", "f", "", "filename to apply")
	return cmd
}

func applyAgentWithAction(ctx context.Context, m Manifest) (string, error) {
	name := m.Metadata.Name
	if name == "" {
		return "", fmt.Errorf("metadata.name is required")
	}

	projectID := getProjectID()

	// Check if agent exists
	var existing *sdk.Agent
	if projectID != "" {
		var err error
		existing, err = client.Agents.GetByName(ctx, projectID, name)
		if err != nil {
			// Ignore error, assume it doesn't exist
			existing = nil
		}
	}

	description, _ := m.Spec["description"].(string)

	if existing == nil {
		// Create
		req := &sdk.CreateAgentRequest{
			Name:        name,
			Description: description,
			ProjectID:   projectID,
			Slug:        Slugify(name),
			Framework:   "custom",
		}

		// Basic mapping for declarative spec
		if decl, ok := m.Spec["declarative"].(map[string]interface{}); ok {
			if model, ok := decl["modelConfig"].(string); ok {
				req.ModelConfig = &sdk.ModelConfig{Model: model}
			}
		}

		spinner := output.NewSpinner(fmt.Sprintf("Creating agent %s...", name))
		spinner.Start()
		_, err := client.Agents.Create(ctx, req)
		if err != nil {
			spinner.FailErr(fmt.Sprintf("Failed to create agent %s", name), err)
			return "", err
		}
		spinner.Success(fmt.Sprintf("Agent %s created", name))
		return "created", nil
	}

	// Update
	req := &sdk.UpdateAgentRequest{
		Name:        &name,
		Description: &description,
	}

	spinner := output.NewSpinner(fmt.Sprintf("Updating agent %s...", name))
	spinner.Start()
	_, err := client.Agents.Update(ctx, existing.ID, req)
	if err != nil {
		spinner.FailErr(fmt.Sprintf("Failed to update agent %s", name), err)
		return "", err
	}
	spinner.Success(fmt.Sprintf("Agent %s updated", name))
	return "updated", nil
}

func applyDeploymentWithAction(ctx context.Context, m Manifest) (string, error) {
	name := m.Metadata.Name
	if name == "" {
		return "", fmt.Errorf("metadata.name is required")
	}

	// For deployments, we usually need an agent ID
	agentName, _ := m.Spec["agentName"].(string)
	if agentName == "" {
		return "", fmt.Errorf("spec.agentName is required for deployment")
	}

	spinner := output.NewSpinner(fmt.Sprintf("Applying deployment %s...", name))
	spinner.Start()

	projectID := getProjectID()
	_, err := client.Agents.GetByName(ctx, projectID, agentName)
	if err != nil {
		spinner.FailErr(fmt.Sprintf("Failed to find agent %s", agentName), err)
		return "", err
	}

	// Check if deployment exists (this is simplified, usually we'd check by name/label)
	// For now, let's just create a new one

	image, _ := m.Spec["image"].(string)
	if image == "" {
		image = "nginx" // Default image
	}

	var replicas int32 = 1
	if r, ok := m.Spec["replicas"].(int); ok {
		replicas = int32(r)
	} else if r, ok := m.Spec["replicas"].(float64); ok {
		replicas = int32(r)
	}

	req := &sdk.CreateDeploymentRequest{
		ID:       Slugify(name),
		Name:     name,
		Image:    image,
		Replicas: replicas,
	}

	_, err = client.Deployments.Create(ctx, req)
	if err != nil {
		spinner.FailErr(fmt.Sprintf("Failed to create deployment for agent %s", agentName), err)
		return "", err
	}
	spinner.Success(fmt.Sprintf("Deployment for agent %s created", agentName))

	return "created", nil
}
