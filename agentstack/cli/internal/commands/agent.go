package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage AI agents",
	}
	cmd.AddCommand(newAgentListCmd())
	cmd.AddCommand(newAgentGetCmd())
	cmd.AddCommand(newAgentCreateCmd())
	cmd.AddCommand(newAgentUpdateCmd())
	cmd.AddCommand(newAgentDeleteCmd())
	cmd.AddCommand(newAgentDeployCmd())
	cmd.AddCommand(newLogsCmd())
	return cmd
}

func newAgentDeployCmd() *cobra.Command {
	var filename string
	cmd := &cobra.Command{
		Use:   "deploy -f <filename>",
		Short: "Deploy an agent from a manifest file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if filename == "" {
				return fmt.Errorf("-f is required")
			}
			data, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			var manifest Manifest
			if err := yaml.Unmarshal(data, &manifest); err != nil {
				return err
			}
			if manifest.Kind != "Agent" {
				return fmt.Errorf("manifest kind must be Agent")
			}
			_, err = applyAgentWithAction(cmd.Context(), manifest)
			return err
		},
	}
	cmd.Flags().StringVarP(&filename, "file", "f", "", "agent manifest file")
	return cmd
}

func newAgentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all agents",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			opts := &sdk.ListOptions{}
			resp, err := client.Agents.List(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list agents: %w", err)
			}
			formatter, err := getFormatter()
			if err != nil {
				return err
			}
			if formatter.Format() == output.FormatJSON || formatter.Format() == output.FormatYAML {
				return formatter.Print(resp.Agents)
			}
			tp := output.NewTablePrinter()
			tp.SetHeaders("ID", "Name", "Status", "Created")
			for _, agent := range resp.Agents {
				created := formatTime(agent.CreatedAt)
				status := output.ColorizeStatus(agent.Status)
				tp.AddRow(agent.ID, agent.Name, status, created)
			}
			return tp.Render()
		},
	}
}

func newAgentGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-name>",
		Short: "Get agent details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			idOrName := args[0]
			agent, err := client.Agents.Get(ctx, idOrName)
			if err != nil {
				projectID := getProjectID()
				if projectID != "" {
					agent, err = client.Agents.GetByName(ctx, projectID, idOrName)
				}
				if err != nil {
					return fmt.Errorf("failed to get agent: %w", err)
				}
			}
			formatter, err := getFormatter()
			if err != nil {
				return err
			}
			if formatter.Format() == output.FormatJSON || formatter.Format() == output.FormatYAML {
				return formatter.Print(agent)
			}
			fmt.Printf("ID:           %s\n", agent.ID)
			fmt.Printf("Name:         %s\n", agent.Name)
			fmt.Printf("Description:  %s\n", agent.Description)
			fmt.Printf("Status:       %s\n", agent.Status)
			fmt.Printf("Version:      %s\n", agent.Version)
			fmt.Printf("Created:      %s\n", agent.CreatedAt.Format(time.RFC3339))
			return nil
		},
	}
}

func newAgentCreateCmd() *cobra.Command {
	var name, description string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			createReq := &sdk.CreateAgentRequest{
				Name:        name,
				Description: description,
				ProjectID:   getProjectID(),
			}
			spinner := output.NewSpinner(fmt.Sprintf("Creating agent %s...", name))
			spinner.Start()
			agent, err := client.Agents.Create(ctx, createReq)
			if err != nil {
				spinner.Fail("Failed to create agent")
				return fmt.Errorf("failed to create agent: %w", err)
			}
			spinner.Success(fmt.Sprintf("Agent created: %s", agent.ID))
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "agent name (required)")
	cmd.Flags().StringVar(&description, "description", "", "agent description")
	return cmd
}

func newAgentUpdateCmd() *cobra.Command {
	var name, description string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			agentID := args[0]
			updateReq := &sdk.UpdateAgentRequest{}
			if cmd.Flags().Changed("name") {
				updateReq.Name = &name
			}
			if cmd.Flags().Changed("description") {
				updateReq.Description = &description
			}
			spinner := output.NewSpinner(fmt.Sprintf("Updating agent %s...", agentID))
			spinner.Start()
			_, err := client.Agents.Update(ctx, agentID, updateReq)
			if err != nil {
				spinner.Fail("Failed to update agent")
				return fmt.Errorf("failed to update agent: %w", err)
			}
			spinner.Success("Agent updated")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "agent name")
	cmd.Flags().StringVar(&description, "description", "", "agent description")
	return cmd
}

func newAgentDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			agentID := args[0]
			if !force {
				fmt.Printf("Are you sure you want to delete agent %s? (y/N): ", agentID)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Aborted")
					return nil
				}
			}
			spinner := output.NewSpinner(fmt.Sprintf("Deleting agent %s...", agentID))
			spinner.Start()
			err := client.Agents.Delete(ctx, agentID)
			if err != nil {
				spinner.Fail("Failed to delete agent")
				return fmt.Errorf("failed to delete agent: %w", err)
			}
			spinner.Success("Agent deleted")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation")
	return cmd
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	duration := time.Since(t)
	return output.FormatDuration(duration) + " ago"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
