package commands

import (
	"fmt"
	"time"

	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
	"github.com/spf13/cobra"
)

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Manage deployments",
	}
	cmd.AddCommand(newDeployListCmd())
	cmd.AddCommand(newDeployGetCmd())
	cmd.AddCommand(newDeployCreateCmd())
	cmd.AddCommand(newDeployScaleCmd())
	cmd.AddCommand(newDeployDeleteCmd())
	cmd.AddCommand(newDeployRestartCmd())
	cmd.AddCommand(newDeployStatusCmd())
	return cmd
}

func newDeployListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all deployments",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			opts := &sdk.ListOptions{}
			resp, err := client.Deployments.List(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list deployments: %w", err)
			}
			formatter, err := getFormatter()
			if err != nil {
				return err
			}
			if formatter.Format() == output.FormatJSON || formatter.Format() == output.FormatYAML {
				return formatter.Print(resp.Deployments)
			}
			tp := output.NewTablePrinter()
			tp.SetHeaders("ID", "Agent", "Replicas", "Status")
			for _, d := range resp.Deployments {
				replicas := fmt.Sprintf("%d", d.Replicas)
				status := output.ColorizeStatus(d.Status)
				tp.AddRow(d.ID, d.AgentID, replicas, status)
			}
			return tp.Render()
		},
	}
}

func newDeployGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get deployment details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			deploymentID := args[0]
			deployment, err := client.Deployments.Get(ctx, deploymentID)
			if err != nil {
				return fmt.Errorf("failed to get deployment: %w", err)
			}
			formatter, err := getFormatter()
			if err != nil {
				return err
			}
			if formatter.Format() == output.FormatJSON || formatter.Format() == output.FormatYAML {
				return formatter.Print(deployment)
			}
			fmt.Printf("ID:           %s\n", deployment.ID)
			fmt.Printf("Agent ID:     %s\n", deployment.AgentID)
			fmt.Printf("Status:       %s\n", output.ColorizeStatus(deployment.Status))
			fmt.Printf("Replicas:     %d\n", deployment.Replicas)
			fmt.Printf("Endpoint:     %s\n", deployment.Endpoint)
			return nil
		},
	}
}

func newDeployCreateCmd() *cobra.Command {
	var agentID, version, environment string
	var wait bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if agentID == "" {
				return fmt.Errorf("--agent is required")
			}
			createReq := &sdk.CreateDeploymentRequest{
				AgentID:     agentID,
				Version:     version,
				Environment: environment,
			}
			spinner := output.NewSpinner("Creating deployment...")
			spinner.Start()
			deployment, err := client.Deployments.Create(ctx, createReq)
			if err != nil {
				spinner.Fail("Failed to create deployment")
				return fmt.Errorf("failed to create deployment: %w", err)
			}
			spinner.Success(fmt.Sprintf("Deployment created: %s", deployment.ID))
			if wait {
				spinner = output.NewSpinner("Waiting for deployment...")
				spinner.Start()
				deadline := time.Now().Add(timeout)
				for time.Now().Before(deadline) {
					status, err := client.Deployments.Status(ctx, deployment.ID)
					if err != nil {
						spinner.Fail("Failed to get status")
						return err
					}
					if status == "running" || status == "ready" {
						spinner.Success("Deployment is ready")
						return nil
					}
					if status == "failed" {
						spinner.Fail("Deployment failed")
						return fmt.Errorf("deployment failed")
					}
					time.Sleep(2 * time.Second)
				}
				spinner.Fail("Deployment timed out")
				return fmt.Errorf("deployment timed out")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&agentID, "agent", "", "agent ID (required)")
	cmd.Flags().StringVar(&version, "version", "", "agent version")
	cmd.Flags().StringVar(&environment, "environment", "", "environment name")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for deployment")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "wait timeout")
	return cmd
}

func newDeployScaleCmd() *cobra.Command {
	var replicas int
	cmd := &cobra.Command{
		Use:   "scale <id>",
		Short: "Scale a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			deploymentID := args[0]
			spinner := output.NewSpinner(fmt.Sprintf("Scaling to %d replicas...", replicas))
			spinner.Start()
			_, err := client.Deployments.Scale(ctx, deploymentID, replicas)
			if err != nil {
				spinner.Fail("Failed to scale deployment")
				return fmt.Errorf("failed to scale deployment: %w", err)
			}
			spinner.Success("Deployment scaled")
			return nil
		},
	}
	cmd.Flags().IntVar(&replicas, "replicas", 1, "number of replicas")
	return cmd
}

func newDeployDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			deploymentID := args[0]
			if !force {
				fmt.Printf("Are you sure you want to delete deployment %s? (y/N): ", deploymentID)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Aborted")
					return nil
				}
			}
			spinner := output.NewSpinner(fmt.Sprintf("Deleting deployment %s...", deploymentID))
			spinner.Start()
			err := client.Deployments.Delete(ctx, deploymentID)
			if err != nil {
				spinner.Fail("Failed to delete deployment")
				return fmt.Errorf("failed to delete deployment: %w", err)
			}
			spinner.Success("Deployment deleted")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation")
	return cmd
}

func newDeployRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart <id>",
		Short: "Restart a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			deploymentID := args[0]
			spinner := output.NewSpinner("Restarting deployment...")
			spinner.Start()
			_, err := client.Deployments.Restart(ctx, deploymentID)
			if err != nil {
				spinner.Fail("Failed to restart deployment")
				return fmt.Errorf("failed to restart deployment: %w", err)
			}
			spinner.Success("Deployment restarted")
			return nil
		},
	}
}

func newDeployStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <id>",
		Short: "Get deployment status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			deploymentID := args[0]
			status, err := client.Deployments.Status(ctx, deploymentID)
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}
			fmt.Printf("Status: %s\n", status)
			return nil
		},
	}
}
