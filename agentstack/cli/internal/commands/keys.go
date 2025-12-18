package commands

import (
	"fmt"
	"time"

	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
	"github.com/spf13/cobra"
)

func newKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage API keys",
	}
	cmd.AddCommand(newKeysListCmd())
	cmd.AddCommand(newKeysCreateCmd())
	cmd.AddCommand(newKeysDeleteCmd())
	return cmd
}

func newKeysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all API keys",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			resp, err := client.Auth.ListAPIKeys(ctx)
			if err != nil {
				return fmt.Errorf("failed to list API keys: %w", err)
			}
			formatter, err := getFormatter()
			if err != nil {
				return err
			}
			if formatter.Format() == output.FormatJSON || formatter.Format() == output.FormatYAML {
				return formatter.Print(resp.Keys)
			}
			tp := output.NewTablePrinter()
			tp.SetHeaders("ID", "Name", "Prefix", "Scopes", "Created")
			for _, k := range resp.Keys {
				tp.AddRow(k.ID, k.Name, k.Prefix, fmt.Sprintf("%v", k.Scopes), k.CreatedAt.Format(time.RFC3339))
			}
			return tp.Render()
		},
	}
}

func newKeysCreateCmd() *cobra.Command {
	var name, projectID string
	var scopes []string
	var expiresDays int

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			req := &sdk.APIKeyRequest{
				Name:      name,
				ProjectID: projectID,
				Scopes:    scopes,
			}

			if expiresDays > 0 {
				expiresAt := time.Now().AddDate(0, 0, expiresDays)
				req.ExpiresAt = &expiresAt
			}

			spinner := output.NewSpinner(fmt.Sprintf("Creating API key %s...", name))
			spinner.Start()
			key, err := client.Auth.CreateAPIKey(ctx, req)
			if err != nil {
				spinner.Fail("Failed to create API key")
				return fmt.Errorf("failed to create API key: %w", err)
			}
			spinner.Success("API key created successfully")

			fmt.Printf("\nIMPORTANT: Copy this key now. It will not be shown again!\n")
			fmt.Printf("API Key: %s\n\n", key.Key)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Key name")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Optional project ID")
	cmd.Flags().StringSliceVar(&scopes, "scopes", []string{"*"}, "Optional scopes")
	cmd.Flags().IntVar(&expiresDays, "expires-days", 0, "Expiration in days (0 for never)")

	return cmd
}

func newKeysDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id := args[0]

			spinner := output.NewSpinner(fmt.Sprintf("Deleting API key %s...", id))
			spinner.Start()
			err := client.Auth.RevokeAPIKey(ctx, id)
			if err != nil {
				spinner.Fail("Failed to delete API key")
				return fmt.Errorf("failed to delete API key: %w", err)
			}
			spinner.Success("API key deleted successfully")
			return nil
		},
	}
}
