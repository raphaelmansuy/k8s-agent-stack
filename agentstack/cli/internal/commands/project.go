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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
)

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}
	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectGetCmd())
	cmd.AddCommand(newProjectCreateCmd())
	cmd.AddCommand(newProjectUpdateCmd())
	cmd.AddCommand(newProjectDeleteCmd())
	return cmd
}

func newProjectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all projects",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			resp, err := client.Projects.List(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to list projects: %w", err)
			}
			formatter, err := getFormatter()
			if err != nil {
				return err
			}
			if formatter.Format() == output.FormatJSON || formatter.Format() == output.FormatYAML {
				return formatter.Print(resp.Projects)
			}
			tp := output.NewTablePrinter()
			tp.SetHeaders("ID", "Name", "Description")
			for _, p := range resp.Projects {
				tp.AddRow(p.ID, p.Name, truncate(p.Description, 30))
			}
			return tp.Render()
		},
	}
}

func newProjectGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-name>",
		Short: "Get project details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			idOrName := args[0]
			project, err := client.Projects.Get(ctx, idOrName)
			if err != nil {
				project, err = client.Projects.GetByName(ctx, idOrName)
				if err != nil {
					return fmt.Errorf("failed to get project: %w", err)
				}
			}
			formatter, err := getFormatter()
			if err != nil {
				return err
			}
			if formatter.Format() == output.FormatJSON || formatter.Format() == output.FormatYAML {
				return formatter.Print(project)
			}
			fmt.Printf("ID:           %s\n", project.ID)
			fmt.Printf("Name:         %s\n", project.Name)
			fmt.Printf("Description:  %s\n", project.Description)
			fmt.Printf("Created:      %s\n", project.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
			return nil
		},
	}
}

func newProjectCreateCmd() *cobra.Command {
	var name, description string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			slug := Slugify(name)
			createReq := &sdk.CreateProjectRequest{
				Name:        name,
				Description: description,
				Slug:        slug,
			}
			spinner := output.NewSpinner(fmt.Sprintf("Creating project %s...", name))
			spinner.Start()
			project, err := client.Projects.Create(ctx, createReq)
			if err != nil {
				spinner.Fail("Failed to create project")
				return fmt.Errorf("failed to create project: %w", err)
			}
			spinner.Stop()

			formatter, err := getFormatter()
			if err != nil {
				return err
			}
			if formatter.Format() == output.FormatJSON || formatter.Format() == output.FormatYAML {
				return formatter.Print(project)
			}

			fmt.Printf("✓ Project created: %s\n", project.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "project name (required)")
	cmd.Flags().StringVar(&description, "description", "", "project description")
	return cmd
}

func newProjectUpdateCmd() *cobra.Command {
	var name, description string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			projectID := args[0]
			updateReq := &sdk.UpdateProjectRequest{}
			if cmd.Flags().Changed("name") {
				updateReq.Name = &name
			}
			if cmd.Flags().Changed("description") {
				updateReq.Description = &description
			}
			spinner := output.NewSpinner(fmt.Sprintf("Updating project %s...", projectID))
			spinner.Start()
			_, err := client.Projects.Update(ctx, projectID, updateReq)
			if err != nil {
				spinner.Fail("Failed to update project")
				return fmt.Errorf("failed to update project: %w", err)
			}
			spinner.Success("Project updated")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "project name")
	cmd.Flags().StringVar(&description, "description", "", "project description")
	return cmd
}

func newProjectDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			projectID := args[0]
			if !force {
				fmt.Printf("Are you sure you want to delete project %s? (y/N): ", projectID)
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Aborted")
					return nil
				}
			}
			spinner := output.NewSpinner(fmt.Sprintf("Deleting project %s...", projectID))
			spinner.Start()
			err := client.Projects.Delete(ctx, projectID)
			if err != nil {
				spinner.Fail("Failed to delete project")
				return fmt.Errorf("failed to delete project: %w", err)
			}
			spinner.Success("Project deleted")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation")
	return cmd
}
