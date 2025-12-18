package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if filename == "" {
				return fmt.Errorf("-f is required")
			}
			data, err := os.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			var manifest Manifest
			if err := yaml.Unmarshal(data, &manifest); err != nil {
				return fmt.Errorf("failed to parse manifest: %w", err)
			}

			ctx := cmd.Context()
			switch manifest.Kind {
			case "Agent":
				return applyAgent(ctx, manifest)
			case "Deployment":
				return applyDeployment(ctx, manifest)
			default:
				return fmt.Errorf("unsupported kind: %s", manifest.Kind)
			}
		},
	}
	cmd.Flags().StringVarP(&filename, "file", "f", "", "filename to apply")
	return cmd
}

func applyAgent(ctx context.Context, m Manifest) error {
	name := m.Metadata.Name
	if name == "" {
		return fmt.Errorf("metadata.name is required")
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
			spinner.Fail("Failed to create agent")
			return err
		}
		spinner.Success(fmt.Sprintf("Agent %s created", name))
	} else {
		// Update
		req := &sdk.UpdateAgentRequest{
			Name:        &name,
			Description: &description,
		}
		
		spinner := output.NewSpinner(fmt.Sprintf("Updating agent %s...", name))
		spinner.Start()
		_, err := client.Agents.Update(ctx, existing.ID, req)
		if err != nil {
			spinner.Fail("Failed to update agent")
			return err
		}
		spinner.Success(fmt.Sprintf("Agent %s updated", name))
	}
	return nil
}

func applyDeployment(ctx context.Context, m Manifest) error {
	name := m.Metadata.Name
	if name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	// For deployments, we usually need an agent ID
	agentName, _ := m.Spec["agentName"].(string)
	if agentName == "" {
		return fmt.Errorf("spec.agentName is required for deployment")
	}

	projectID := getProjectID()
	agent, err := client.Agents.GetByName(ctx, projectID, agentName)
	if err != nil {
		return fmt.Errorf("failed to find agent %s: %w", agentName, err)
	}

	// Check if deployment exists (this is simplified, usually we'd check by name/label)
	// For now, let's just create a new one
	
	req := &sdk.CreateDeploymentRequest{
		AgentID: agent.ID,
	}
	
	if version, ok := m.Spec["version"].(string); ok {
		req.Version = version
	}
	
	spinner := output.NewSpinner(fmt.Sprintf("Creating deployment for agent %s...", agentName))
	spinner.Start()
	_, err = client.Deployments.Create(ctx, req)
	if err != nil {
		spinner.Fail("Failed to create deployment")
		return err
	}
	spinner.Success(fmt.Sprintf("Deployment for agent %s created", agentName))
	
	return nil
}
