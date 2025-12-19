package commands

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/raphaelmansuy/agentstack/cli/internal/config"
	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
	"github.com/spf13/cobra"
)

var (
	cfgFile      string
	outputFormat string
	apiKey       string
	endpoint     string
	verbose      bool
	projectID    string
	cfg          *config.Config
	client       *sdk.Client
)

var rootCmd = &cobra.Command{
	Use:   "agentctl",
	Short: "AgentStack CLI - Manage AI agents and deployments",
	Long:  "agentctl is a command-line interface for the AgentStack platform.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "completion" || cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		profile := cfg.CurrentProfileConfig()
		if profile == nil {
			return fmt.Errorf("no profile configured")
		}
		ep := profile.Endpoint
		if endpoint != "" {
			ep = endpoint
		}
		key := profile.APIKey
		if apiKey != "" {
			key = apiKey
		}
		// Use token if API key is not set
		if key == "" && profile.Token != "" {
			key = profile.Token
		}
		client = sdk.NewClient(ep, key)
		return nil
	},
}

func Slugify(s string) string {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key")
	rootCmd.PersistentFlags().StringVar(&endpoint, "endpoint", "", "API endpoint")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&projectID, "project", "p", "", "project ID")
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newAgentCmd())
	rootCmd.AddCommand(newProjectCmd())
	rootCmd.AddCommand(newDeployCmd())
	rootCmd.AddCommand(newChatCmd())
	rootCmd.AddCommand(newLogsCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newKeysCmd())
	rootCmd.AddCommand(newApplyCmd())
	rootCmd.AddCommand(newUICmd())
	rootCmd.AddCommand(newDocsCmd())
}

func getFormatter() (*output.Formatter, error) {
	format, err := output.ParseFormat(outputFormat)
	if err != nil {
		return nil, err
	}
	return output.NewFormatter(format), nil
}

func getProjectID() string {
	if projectID != "" {
		return projectID
	}
	if cfg != nil && cfg.CurrentProfileConfig() != nil {
		return cfg.CurrentProfileConfig().DefaultProject
	}
	return ""
}
