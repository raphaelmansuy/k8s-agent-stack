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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/raphaelmansuy/agentstack/cli/internal/config"
	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
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
				err = config.Save(loadedCfg, cfgFile)
				if err != nil {
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
