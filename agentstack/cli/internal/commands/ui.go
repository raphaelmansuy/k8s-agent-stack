package commands

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/spf13/cobra"
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
			checkSvc := exec.Command("kubectl", "get", "svc", "agentstack-ui", "-n", namespace)
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

			uiPfCmd := exec.Command("kubectl", pfArgs...)
			uiPfCmd.Stdout = os.Stdout
			uiPfCmd.Stderr = os.Stderr

			if err := uiPfCmd.Start(); err != nil {
				return fmt.Errorf("failed to start UI port-forward: %w", err)
			}

			// Wait a bit for port-forward to be ready
			time.Sleep(2 * time.Second)

			if !noBrowser {
				openBrowser(url)
			}

			// Handle graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\nStopping port-forward...")
			if uiPfCmd.Process != nil {
				uiPfCmd.Process.Kill()
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "P", 3000, "local port for the UI")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "agentstack", "namespace where the UI is deployed")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "do not open the browser automatically")

	return cmd
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		fmt.Printf("Please open %s manually\n", url)
	}
}
