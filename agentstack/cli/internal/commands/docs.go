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
			openBrowser(cmd.Context(), url)

			return nil
		},
	}

	return cmd
}
