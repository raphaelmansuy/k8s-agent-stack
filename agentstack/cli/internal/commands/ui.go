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

func newUICmd() *cobra.Command {
	var port int
	var namespace string
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Open the Kagent Web UI",
		Long:  `Open the Kagent Web UI in your default browser. This command starts a port-forward to the UI service in your Kubernetes cluster.`,
		Example: `  # Open the UI using default settings
  agentctl ui

  # Open the UI on a specific port
  agentctl ui --port 9000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if kubectl is installed
			if _, err := exec.LookPath("kubectl"); err != nil {
				return fmt.Errorf("kubectl not found in PATH. Please install kubectl to use this command")
			}

			// Check if the service exists
			checkSvc := exec.CommandContext(cmd.Context(), "kubectl", "get", "svc", "agentstack-ui", "-n", namespace)
			if err := checkSvc.Run(); err != nil {
				return fmt.Errorf("agentstack-ui service not found in namespace %s. Is the stack deployed?", namespace)
			}

			url := fmt.Sprintf("http://localhost:%d", port)

			fmt.Printf("%s╔════════════════════════════════════════╗%s\n", output.ColorGreen, output.ColorReset)
			fmt.Printf("%s║  Kagent UI → %-25s ║%s\n", output.ColorGreen, url, output.ColorReset)
			fmt.Printf("%s╚════════════════════════════════════════╝%s\n", output.ColorGreen, output.ColorReset)
			fmt.Println()
			fmt.Println("🌐 Starting port-forward...")
			fmt.Println("   Keep this terminal open while using the UI")
			fmt.Println("   Press Ctrl+C to stop")
			fmt.Println()

			// Start port-forward for UI
			// We forward the user's port, 8080 (Next.js server), 8083 (API proxy), and 8081 (WS proxy)
			pfArgs := []string{"port-forward", "-n", namespace, "svc/agentstack-ui"}
			pfArgs = append(pfArgs, fmt.Sprintf("%d:80", port))
			if port != 8080 {
				pfArgs = append(pfArgs, "8080:80")
			}
			pfArgs = append(pfArgs, "8083:8083", "8081:8081")

			uiPfCmd := exec.CommandContext(cmd.Context(), "kubectl", pfArgs...)
			uiPfCmd.Stdout = os.Stdout
			uiPfCmd.Stderr = os.Stderr

			if err := uiPfCmd.Start(); err != nil {
				return fmt.Errorf("failed to start UI port-forward: %w", err)
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
			if uiPfCmd.Process != nil {
				_ = uiPfCmd.Process.Kill()
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "P", 3000, "local port for the UI")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "agentstack", "namespace where the UI is deployed")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "do not open the browser automatically")

	return cmd
}
