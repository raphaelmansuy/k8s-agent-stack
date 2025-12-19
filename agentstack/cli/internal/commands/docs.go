package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newDocsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Open the API documentation",
		Long:  `Open the interactive API documentation (Swagger UI) in your default browser.`,
		Example: `  # Open the API documentation
  agentctl docs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := cfg.CurrentProfileConfig()
			if profile == nil {
				return fmt.Errorf("no profile configured")
			}

			ep := profile.Endpoint
			if endpoint != "" {
				ep = endpoint
			}

			if ep == "" {
				return fmt.Errorf("API endpoint not configured")
			}

			// Ensure the URL ends with /docs
			url := strings.TrimSuffix(ep, "/")
			url = fmt.Sprintf("%s/docs", url)

			fmt.Printf("🌐 Opening API documentation at %s...\n", url)
			openBrowser(url)

			return nil
		},
	}

	return cmd
}
