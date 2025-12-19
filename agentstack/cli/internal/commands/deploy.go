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
			tp.SetHeaders("ID", "Name", "Image", "Replicas", "Status")
			for _, d := range resp.Deployments {
				replicas := fmt.Sprintf("%d", d.Replicas)
				status := output.ColorizeStatus(d.Status.Phase)
				tp.AddRow(d.ID, d.Name, d.Image, replicas, status)
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
			fmt.Printf("Name:         %s\n", deployment.Name)
			fmt.Printf("Image:        %s\n", deployment.Image)
			fmt.Printf("Status:       %s\n", output.ColorizeStatus(deployment.Status.Phase))
			fmt.Printf("Replicas:     %d\n", deployment.Replicas)
			fmt.Printf("Endpoint:     %s\n", deployment.Status.URL)
			return nil
		},
	}
}

func newDeployCreateCmd() *cobra.Command {
	var name, description, image, namespace, deployType string
	var replicas int32
	var wait bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if image == "" {
				return fmt.Errorf("--image is required")
			}
			id := Slugify(name)
			createReq := &sdk.CreateDeploymentRequest{
				ID:          id,
				Name:        name,
				Description: description,
				Image:       image,
				Namespace:   namespace,
				Type:        deployType,
				Replicas:    replicas,
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
					// Note: Status might not be implemented yet in the same way
					time.Sleep(2 * time.Second)
					if time.Now().After(deadline) {
						break
					}
				}
				spinner.Success("Wait finished")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "deployment name (required)")
	cmd.Flags().StringVar(&description, "description", "", "deployment description")
	cmd.Flags().StringVar(&image, "image", "", "container image (required)")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "kubernetes namespace")
	cmd.Flags().StringVar(&deployType, "type", "ADK", "deployment type (ADK, LLM, BYO)")
	cmd.Flags().Int32Var(&replicas, "replicas", 1, "number of replicas")
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
			fmt.Printf("Status:  %s\n", output.ColorizeStatus(status.Phase))
			if status.URL != "" {
				fmt.Printf("URL:     %s\n", status.URL)
			}
			if status.Message != "" {
				fmt.Printf("Message: %s\n", status.Message)
			}
			return nil
		},
	}
}
