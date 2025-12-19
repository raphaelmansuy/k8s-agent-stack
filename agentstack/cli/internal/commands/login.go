package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/raphaelmansuy/agentstack/cli/internal/config"
	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCmd() *cobra.Command {
	var loginEndpoint string
	var loginAPIKey string
	var profileName string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the AgentStack API",
		RunE: func(cmd *cobra.Command, args []string) error {
			loadedCfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			ep := loginEndpoint
			if ep == "" {
				ep = "http://localhost:8080"
				fmt.Printf("Endpoint [%s]: ", ep)
				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)
				if input != "" {
					ep = input
				}
			}
			pName := profileName
			if pName == "" {
				pName = "default"
			}
			if loginAPIKey != "" {
				loginAPIKey = strings.TrimSpace(loginAPIKey)
				loadedCfg.SetProfile(pName, &config.Profile{
					Endpoint: ep,
					APIKey:   loginAPIKey,
				})
				loadedCfg.CurrentProfile = pName
				if err := config.Save(loadedCfg, cfgFile); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				fmt.Println(output.Success("Logged in with API key"))
				return nil
			}
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Username: ")
			username, _ := reader.ReadString('\n')
			username = strings.TrimSpace(username)
			fmt.Print("Password: ")
			passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password := string(passwordBytes)
			// Create client with empty API key for login
			loginClient := sdk.NewClient(ep, "")
			spinner := output.NewSpinner("Authenticating...")
			spinner.Start()
			loginReq := &sdk.LoginRequest{Email: username, Password: password}
			authResponse, err := loginClient.Auth.Login(cmd.Context(), loginReq)
			if err != nil {
				spinner.Fail("Authentication failed")
				return fmt.Errorf("login failed: %w", err)
			}
			spinner.Success("Authenticated successfully")
			loadedCfg.SetProfile(pName, &config.Profile{
				Endpoint:     ep,
				Token:        authResponse.AccessToken,
				RefreshToken: authResponse.RefreshToken,
			})
			loadedCfg.CurrentProfile = pName
			if err := config.Save(loadedCfg, cfgFile); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Printf("Logged in as %s\n", username)
			return nil
		},
	}
	cmd.Flags().StringVar(&loginEndpoint, "endpoint", "", "API endpoint URL")
	cmd.Flags().StringVar(&loginAPIKey, "api-key", "", "API key for authentication")
	cmd.Flags().StringVar(&profileName, "profile", "", "profile name")
	return cmd
}
