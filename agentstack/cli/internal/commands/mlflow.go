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
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/raphaelmansuy/agentstack/cli/internal/output"
)

func newMLflowCmd() *cobra.Command {
	var port int
	var namespace string
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "mlflow",
		Short: "Open the MLflow UI",
		Long:  `Open the MLflow UI in your default browser. This command starts a port-forward to the MLflow service in your Kubernetes cluster.`,
		Example: `  # Open the MLflow UI using default settings
  agentctl mlflow

  # Open the MLflow UI on a specific port
  agentctl mlflow --port 5001`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if kubectl is installed
			if _, err := exec.LookPath("kubectl"); err != nil {
				return fmt.Errorf("kubectl not found in PATH. Please install kubectl to use this command")
			}

			// Check if the service exists
			checkSvc := exec.CommandContext(cmd.Context(), "kubectl", "get", "svc", "agentstack-mlflow", "-n", namespace)
			if err := checkSvc.Run(); err != nil {
				return fmt.Errorf("agentstack-mlflow service not found in namespace %s. Is the stack deployed?", namespace)
			}

			url := fmt.Sprintf("http://localhost:%d", port)

			fmt.Printf("%s╔════════════════════════════════════════╗%s\n", output.ColorGreen, output.ColorReset)
			fmt.Printf("%s║  MLflow UI → %-25s ║%s\n", output.ColorGreen, url, output.ColorReset)
			fmt.Printf("%s╚════════════════════════════════════════╝%s\n", output.ColorGreen, output.ColorReset)
			fmt.Println()
			fmt.Println("🌐 Starting port-forward...")
			fmt.Println("   Keep this terminal open while using the UI")
			fmt.Println("   Press Ctrl+C to stop")
			fmt.Println()

			// Start port-forward for MLflow
			pfArgs := []string{"port-forward", "-n", namespace, "svc/agentstack-mlflow", fmt.Sprintf("%d:5000", port)}

			pfCmd := exec.CommandContext(cmd.Context(), "kubectl", pfArgs...)
			pfCmd.Stdout = os.Stdout
			pfCmd.Stderr = os.Stderr

			if err := pfCmd.Start(); err != nil {
				return fmt.Errorf("failed to start MLflow port-forward: %w", err)
			}

			// Wait a bit for port-forward to be ready
			time.Sleep(2 * time.Second)

			if !noBrowser {
				openBrowser(cmd.Context(), url)
			}

			// Handle graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\nStopping port-forward...")
			if pfCmd.Process != nil {
				_ = pfCmd.Process.Kill()
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "P", 5000, "local port for the MLflow UI")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "agentstack", "namespace where MLflow is deployed")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "do not open the browser automatically")

	return cmd
}
