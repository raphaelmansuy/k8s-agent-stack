# Phase 4: Developer Experience

> CLI Tool, Go SDK, Documentation Site

**Duration**: 2 weeks | **Status**: Not Started | **Priority**: High  
**Depends On**: Phase 1 (Core API)

---

## Objectives

1. Build `agentctl` CLI tool in Go with Cobra
2. Create Go SDK for programmatic access
3. Implement local development workflow
4. Generate OpenAPI documentation
5. Create getting-started tutorials

---

## Architecture Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                      Developer Experience Stack                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                         Developer                                │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                    │                    │                               │
│           ┌───────┴───────┐    ┌───────┴───────┐                       │
│           ▼               ▼    ▼               ▼                       │
│   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                   │
│   │   agentctl   │ │   Go SDK     │ │  OpenAPI     │                   │
│   │     CLI      │ │              │ │   Docs       │                   │
│   └──────┬───────┘ └──────┬───────┘ └──────────────┘                   │
│          │                │                                             │
│          │   Uses SDK     │                                             │
│          └───────┬────────┘                                             │
│                  ▼                                                      │
│   ┌───────────────────────────────────────────────────────────────┐    │
│   │                    AgentStack API Gateway                      │    │
│   │                    (v1/*, a2a/v1/*)                           │    │
│   └───────────────────────────────────────────────────────────────┘    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Week 1: CLI Development

### 1.1 Project Structure

```text
cli/
├── cmd/
│   └── agentctl/
│       └── main.go
├── internal/
│   ├── commands/
│   │   ├── agent.go
│   │   ├── chat.go
│   │   ├── config.go
│   │   ├── deploy.go
│   │   ├── login.go
│   │   ├── project.go
│   │   └── root.go
│   ├── config/
│   │   └── config.go
│   └── output/
│       └── formatter.go
├── go.mod
├── go.sum
└── Makefile
```

### 1.2 Main Entry Point

**`cli/cmd/agentctl/main.go`**:
```go
package main

import (
	"fmt"
	"os"

	"github.com/raphaelmansuy/agentstack/cli/internal/commands"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd := commands.NewRootCommand(version, commit, date)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

### 1.3 Root Command

**`cli/internal/commands/root.go`**:
```go
package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/raphaelmansuy/agentstack/cli/internal/config"
)

func NewRootCommand(version, commit, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "agentctl",
		Short: "AgentStack CLI - Manage AI agents on Kubernetes",
		Long: `agentctl is the command-line tool for AgentStack.

It allows you to create, deploy, manage, and interact with AI agents
running on Kubernetes clusters with Knative.

Documentation: https://docs.agentstack.io/cli`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return config.Initialize()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringP("profile", "p", "default", "Configuration profile to use")
	rootCmd.PersistentFlags().StringP("output", "o", "table", "Output format: table, json, yaml")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("api-url", "", "API server URL (overrides config)")

	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))

	// Add sub-commands
	rootCmd.AddCommand(newLoginCommand())
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newProjectCommand())
	rootCmd.AddCommand(newAgentCommand())
	rootCmd.AddCommand(newDeployCommand())
	rootCmd.AddCommand(newChatCommand())
	rootCmd.AddCommand(newLogsCommand())
	rootCmd.AddCommand(newCompletionCommand())

	return rootCmd
}
```

### 1.4 Login Command

**`cli/internal/commands/login.go`**:
```go
package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/raphaelmansuy/agentstack/cli/internal/config"
	"github.com/raphaelmansuy/agentstack/sdk"
)

func newLoginCommand() *cobra.Command {
	var apiURL string
	var apiKey string
	var interactive bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with AgentStack API",
		Long: `Authenticate with an AgentStack API server.

You can provide credentials interactively, via flags, or environment variables.

Environment variables:
  AGENTSTACK_API_URL    API server URL
  AGENTSTACK_API_KEY    API key for authentication`,
		Example: `  # Interactive login
  agentctl login

  # Login with API key
  agentctl login --api-url https://api.agentstack.io --api-key your-key

  # Login with environment variables
  export AGENTSTACK_API_URL=https://api.agentstack.io
  export AGENTSTACK_API_KEY=your-key
  agentctl login`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(apiURL, apiKey, interactive)
		},
	}

	cmd.Flags().StringVar(&apiURL, "api-url", "", "API server URL")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "Interactive mode")

	return cmd
}

func runLogin(apiURL, apiKey string, interactive bool) error {
	// Check environment variables
	if apiURL == "" {
		apiURL = os.Getenv("AGENTSTACK_API_URL")
	}
	if apiKey == "" {
		apiKey = os.Getenv("AGENTSTACK_API_KEY")
	}

	// Interactive input if needed
	if interactive {
		reader := bufio.NewReader(os.Stdin)

		if apiURL == "" {
			fmt.Print("API URL (default: https://api.agentstack.io): ")
			input, _ := reader.ReadString('\n')
			apiURL = strings.TrimSpace(input)
			if apiURL == "" {
				apiURL = "https://api.agentstack.io"
			}
		}

		if apiKey == "" {
			fmt.Print("API Key: ")
			keyBytes, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("failed to read API key: %w", err)
			}
			apiKey = string(keyBytes)
			fmt.Println()
		}
	}

	if apiURL == "" || apiKey == "" {
		return fmt.Errorf("API URL and API key are required")
	}

	// Validate credentials
	client := sdk.NewClient(apiURL, apiKey)
	user, err := client.Auth.WhoAmI()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Save to config
	profile := config.Profile{
		APIURL: apiURL,
		APIKey: apiKey,
		TeamID: user.TeamID,
	}

	if err := config.SaveProfile("default", profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("✓ Logged in as %s (%s)\n", user.Email, user.TeamID)
	return nil
}
```

### 1.5 Agent Commands

**`cli/internal/commands/agent.go`**:
```go
package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/raphaelmansuy/agentstack/cli/internal/config"
	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/raphaelmansuy/agentstack/sdk"
)

func newAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents",
		Long:  `Create, list, update, and delete agents.`,
	}

	cmd.AddCommand(newAgentListCommand())
	cmd.AddCommand(newAgentCreateCommand())
	cmd.AddCommand(newAgentGetCommand())
	cmd.AddCommand(newAgentDeleteCommand())
	cmd.AddCommand(newAgentUpdateCommand())

	return cmd
}

func newAgentListCommand() *cobra.Command {
	var projectID string
	var limit int

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List agents",
		Example: `  agentctl agent list
  agentctl agent list --project proj_123
  agentctl agent list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			agents, err := client.Agents.List(sdk.ListAgentsParams{
				ProjectID: projectID,
				Limit:     limit,
			})
			if err != nil {
				return err
			}

			return output.Print(agents, output.GetFormat(cmd))
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "Filter by project ID")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of agents to return")

	return cmd
}

func newAgentCreateCommand() *cobra.Command {
	var projectID string
	var name string
	var description string
	var fromFile string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent",
		Long: `Create a new agent in a project.

You can specify the agent configuration inline or from a YAML file.`,
		Example: `  # Create from flags
  agentctl agent create --project proj_123 --name "My Agent" --description "Does stuff"

  # Create from file
  agentctl agent create --from-file agent.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			var req sdk.CreateAgentRequest

			if fromFile != "" {
				// Load from file
				data, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("failed to read file: %w", err)
				}
				if err := yaml.Unmarshal(data, &req); err != nil {
					return fmt.Errorf("failed to parse file: %w", err)
				}
			} else {
				req = sdk.CreateAgentRequest{
					ProjectID:   projectID,
					Name:        name,
					Description: description,
				}
			}

			agent, err := client.Agents.Create(req)
			if err != nil {
				return err
			}

			fmt.Printf("✓ Agent created: %s (%s)\n", agent.Name, agent.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "Project ID")
	cmd.Flags().StringVar(&name, "name", "", "Agent name")
	cmd.Flags().StringVar(&description, "description", "", "Agent description")
	cmd.Flags().StringVarP(&fromFile, "from-file", "f", "", "Create from YAML file")

	return cmd
}

func newAgentGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [agent-id]",
		Short: "Get agent details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()

			agent, err := client.Agents.Get(args[0])
			if err != nil {
				return err
			}

			return output.Print(agent, output.GetFormat(cmd))
		},
	}

	return cmd
}

func newAgentDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [agent-id]",
		Short: "Delete an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			agentID := args[0]

			if !force {
				fmt.Printf("Are you sure you want to delete agent %s? [y/N]: ", agentID)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			if err := client.Agents.Delete(agentID); err != nil {
				return err
			}

			fmt.Printf("✓ Agent %s deleted\n", agentID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation")

	return cmd
}

func newAgentUpdateCommand() *cobra.Command {
	var name string
	var description string
	var fromFile string

	cmd := &cobra.Command{
		Use:   "update [agent-id]",
		Short: "Update an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient()
			agentID := args[0]

			req := sdk.UpdateAgentRequest{}

			if fromFile != "" {
				data, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("failed to read file: %w", err)
				}
				if err := yaml.Unmarshal(data, &req); err != nil {
					return fmt.Errorf("failed to parse file: %w", err)
				}
			} else {
				if name != "" {
					req.Name = &name
				}
				if description != "" {
					req.Description = &description
				}
			}

			agent, err := client.Agents.Update(agentID, req)
			if err != nil {
				return err
			}

			fmt.Printf("✓ Agent updated: %s\n", agent.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	cmd.Flags().StringVarP(&fromFile, "from-file", "f", "", "Update from YAML file")

	return cmd
}

func getClient() *sdk.Client {
	profile := config.GetCurrentProfile()
	return sdk.NewClient(profile.APIURL, profile.APIKey)
}
```

### 1.6 Chat Command (Interactive)

**`cli/internal/commands/chat.go`**:
```go
package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raphaelmansuy/agentstack/sdk"
)

func newChatCommand() *cobra.Command {
	var agentID string
	var sessionID string
	var streaming bool
	var oneShot string

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat with an agent",
		Long: `Start an interactive chat session with an agent.

Use --one-shot for a single message without entering interactive mode.`,
		Example: `  # Interactive chat
  agentctl chat --agent agt_123

  # Single message
  agentctl chat --agent agt_123 --one-shot "Hello, what can you do?"

  # Continue a session
  agentctl chat --agent agt_123 --session ses_456`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if agentID == "" {
				return fmt.Errorf("--agent is required")
			}

			client := getClient()

			if oneShot != "" {
				return runSingleMessage(client, agentID, sessionID, oneShot, streaming)
			}

			return runInteractiveChat(client, agentID, sessionID, streaming)
		},
	}

	cmd.Flags().StringVar(&agentID, "agent", "", "Agent ID (required)")
	cmd.Flags().StringVar(&sessionID, "session", "", "Session ID (optional, creates new if not provided)")
	cmd.Flags().BoolVar(&streaming, "stream", true, "Enable streaming responses")
	cmd.Flags().StringVar(&oneShot, "one-shot", "", "Send a single message and exit")

	cmd.MarkFlagRequired("agent")

	return cmd
}

func runSingleMessage(client *sdk.Client, agentID, sessionID, message string, streaming bool) error {
	if streaming {
		stream, err := client.Chat.Stream(agentID, sdk.ChatRequest{
			SessionID: sessionID,
			Message:   message,
		})
		if err != nil {
			return err
		}

		for event := range stream {
			switch event.Type {
			case "TextMessageContent":
				fmt.Print(event.Data["delta"])
			case "TextMessageEnd":
				fmt.Println()
			case "error":
				return fmt.Errorf("error: %s", event.Data["error"])
			}
		}
		return nil
	}

	resp, err := client.Chat.Send(agentID, sdk.ChatRequest{
		SessionID: sessionID,
		Message:   message,
	})
	if err != nil {
		return err
	}

	fmt.Println(resp.Message)
	return nil
}

func runInteractiveChat(client *sdk.Client, agentID, sessionID string, streaming bool) error {
	fmt.Println("Starting chat with agent", agentID)
	fmt.Println("Type 'exit' or press Ctrl+C to quit")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return nil
		}

		fmt.Print("Agent: ")

		if streaming {
			stream, err := client.Chat.Stream(agentID, sdk.ChatRequest{
				SessionID: sessionID,
				Message:   input,
			})
			if err != nil {
				fmt.Printf("\nError: %s\n", err)
				continue
			}

			for event := range stream {
				switch event.Type {
				case "TextMessageContent":
					fmt.Print(event.Data["delta"])
				case "TextMessageEnd":
					fmt.Println()
				case "SessionCreated":
					if sid, ok := event.Data["session_id"].(string); ok {
						sessionID = sid
					}
				}
			}
		} else {
			resp, err := client.Chat.Send(agentID, sdk.ChatRequest{
				SessionID: sessionID,
				Message:   input,
			})
			if err != nil {
				fmt.Printf("\nError: %s\n", err)
				continue
			}
			sessionID = resp.SessionID
			fmt.Println(resp.Message)
		}

		fmt.Println()
	}
}
```

### 1.7 Deploy Command

**`cli/internal/commands/deploy.go`**:
```go
package commands

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/raphaelmansuy/agentstack/sdk"
)

func newDeployCommand() *cobra.Command {
	var agentID string
	var image string
	var wait bool
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy an agent",
		Long: `Deploy an agent to the Knative cluster.

This creates or updates the Knative Service for the agent and waits for it to become ready.`,
		Example: `  # Deploy an agent
  agentctl deploy --agent agt_123

  # Deploy with specific image
  agentctl deploy --agent agt_123 --image my-registry/my-agent:v1

  # Deploy and wait for ready
  agentctl deploy --agent agt_123 --wait --timeout 5m`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if agentID == "" {
				return fmt.Errorf("--agent is required")
			}

			client := getClient()

			// Start deployment
			deployment, err := client.Deployments.Create(agentID, sdk.CreateDeploymentRequest{
				Image: image,
			})
			if err != nil {
				return fmt.Errorf("failed to start deployment: %w", err)
			}

			fmt.Printf("🚀 Deployment started: %s\n", deployment.ID)

			if !wait {
				fmt.Println("Use 'agentctl deploy status", deployment.ID, "' to check progress")
				return nil
			}

			// Wait for deployment
			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			s.Suffix = " Waiting for deployment..."
			s.Start()

			deadline := time.Now().Add(timeout)
			for time.Now().Before(deadline) {
				status, err := client.Deployments.Get(deployment.ID)
				if err != nil {
					s.Stop()
					return err
				}

				switch status.Status {
				case "ready":
					s.Stop()
					fmt.Printf("✅ Deployment ready!\n")
					fmt.Printf("   URL: %s\n", status.URL)
					return nil
				case "failed":
					s.Stop()
					return fmt.Errorf("deployment failed: %s", status.Message)
				}

				s.Suffix = fmt.Sprintf(" %s: %s", status.Status, status.Message)
				time.Sleep(2 * time.Second)
			}

			s.Stop()
			return fmt.Errorf("deployment timed out")
		},
	}

	cmd.Flags().StringVar(&agentID, "agent", "", "Agent ID (required)")
	cmd.Flags().StringVar(&image, "image", "", "Container image (optional)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for deployment to complete")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for --wait")

	cmd.MarkFlagRequired("agent")

	return cmd
}
```

### 1.8 Config Management

**`cli/internal/config/config.go`**:
```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	CurrentProfile string             `yaml:"current_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

type Profile struct {
	APIURL string `yaml:"api_url"`
	APIKey string `yaml:"api_key"`
	TeamID string `yaml:"team_id"`
}

var cfg Config

func Initialize() error {
	configPath := getConfigPath()

	// Create config dir if not exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return err
	}

	// Read or create config
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		cfg = Config{
			CurrentProfile: "default",
			Profiles:       make(map[string]Profile),
		}
		return nil
	} else if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	// Override from command line
	if profile := viper.GetString("profile"); profile != "" {
		cfg.CurrentProfile = profile
	}

	return nil
}

func GetCurrentProfile() Profile {
	return cfg.Profiles[cfg.CurrentProfile]
}

func SaveProfile(name string, profile Profile) error {
	cfg.Profiles[name] = profile
	cfg.CurrentProfile = name
	return save()
}

func ListProfiles() map[string]Profile {
	return cfg.Profiles
}

func save() error {
	configPath := getConfigPath()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func getConfigPath() string {
	// Check XDG config
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "agentstack", "config.yaml")
	}

	// Default to ~/.config
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agentstack", "config.yaml")
}
```

---

## Week 2: Go SDK

### 2.1 SDK Structure

```text
sdk/
├── client.go
├── agents.go
├── chat.go
├── deployments.go
├── auth.go
├── types.go
├── errors.go
├── streaming.go
└── README.md
```

### 2.2 SDK Client

**`sdk/client.go`**:
```go
package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the AgentStack SDK client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client

	// Service clients
	Auth        *AuthService
	Agents      *AgentsService
	Chat        *ChatService
	Deployments *DeploymentsService
}

// NewClient creates a new SDK client
func NewClient(baseURL, apiKey string) *Client {
	c := &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	c.Auth = &AuthService{client: c}
	c.Agents = &AgentsService{client: c}
	c.Chat = &ChatService{client: c}
	c.Deployments = &DeploymentsService{client: c}

	return c
}

// WithHTTPClient sets a custom HTTP client
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	c.httpClient = httpClient
	return c
}

// Request makes an HTTP request
func (c *Client) Request(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "agentstack-go-sdk/1.0.0")

	return c.httpClient.Do(req)
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	resp, err := c.Request(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := checkError(resp); err != nil {
		return err
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body, result interface{}) error {
	resp, err := c.Request(ctx, "POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := checkError(resp); err != nil {
		return err
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Patch performs a PATCH request
func (c *Client) Patch(ctx context.Context, path string, body, result interface{}) error {
	resp, err := c.Request(ctx, "PATCH", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := checkError(resp); err != nil {
		return err
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.Request(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return checkError(resp)
}

func checkError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    resp.Status,
		}
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Code:       errResp.Error.Code,
		Message:    errResp.Error.Message,
	}
}
```

### 2.3 Agents Service

**`sdk/agents.go`**:
```go
package sdk

import (
	"context"
	"fmt"
)

type AgentsService struct {
	client *Client
}

type Agent struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	ProjectID   string       `json:"project_id"`
	Config      AgentConfig  `json:"config"`
	URLs        AgentURLs    `json:"urls,omitempty"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
}

type AgentConfig struct {
	Model    string   `json:"model"`
	Tools    []string `json:"tools"`
	MaxTurns int      `json:"max_turns"`
}

type AgentURLs struct {
	API  string `json:"api"`
	Card string `json:"card"`
}

type ListAgentsParams struct {
	ProjectID string
	Status    string
	Limit     int
	Offset    int
}

type CreateAgentRequest struct {
	ProjectID   string      `json:"project_id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Config      AgentConfig `json:"config,omitempty"`
}

type UpdateAgentRequest struct {
	Name        *string      `json:"name,omitempty"`
	Description *string      `json:"description,omitempty"`
	Config      *AgentConfig `json:"config,omitempty"`
}

// List returns a list of agents
func (s *AgentsService) List(params ListAgentsParams) ([]Agent, error) {
	path := "/v1/agents?"
	if params.ProjectID != "" {
		path += fmt.Sprintf("project_id=%s&", params.ProjectID)
	}
	if params.Status != "" {
		path += fmt.Sprintf("status=%s&", params.Status)
	}
	if params.Limit > 0 {
		path += fmt.Sprintf("limit=%d&", params.Limit)
	}
	if params.Offset > 0 {
		path += fmt.Sprintf("offset=%d&", params.Offset)
	}

	var result struct {
		Agents []Agent `json:"agents"`
	}

	if err := s.client.Get(context.Background(), path, &result); err != nil {
		return nil, err
	}

	return result.Agents, nil
}

// Get returns an agent by ID
func (s *AgentsService) Get(id string) (*Agent, error) {
	var agent Agent
	if err := s.client.Get(context.Background(), "/v1/agents/"+id, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Create creates a new agent
func (s *AgentsService) Create(req CreateAgentRequest) (*Agent, error) {
	var agent Agent
	if err := s.client.Post(context.Background(), "/v1/agents", req, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Update updates an agent
func (s *AgentsService) Update(id string, req UpdateAgentRequest) (*Agent, error) {
	var agent Agent
	if err := s.client.Patch(context.Background(), "/v1/agents/"+id, req, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Delete deletes an agent
func (s *AgentsService) Delete(id string) error {
	return s.client.Delete(context.Background(), "/v1/agents/"+id)
}
```

### 2.4 Chat Service with Streaming

**`sdk/chat.go`**:
```go
package sdk

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ChatService struct {
	client *Client
}

type ChatRequest struct {
	SessionID string      `json:"session_id,omitempty"`
	Message   interface{} `json:"message"`
}

type ChatResponse struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	TaskID    string `json:"task_id"`
}

type StreamEvent struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// Send sends a chat message and waits for response
func (s *ChatService) Send(agentID string, req ChatRequest) (*ChatResponse, error) {
	var resp ChatResponse
	path := fmt.Sprintf("/v1/agents/%s/chat", agentID)
	if err := s.client.Post(context.Background(), path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Stream sends a chat message and returns a channel of streaming events
func (s *ChatService) Stream(agentID string, req ChatRequest) (<-chan StreamEvent, error) {
	path := fmt.Sprintf("/v1/agents/%s/chat/stream", agentID)

	httpReq, err := s.client.Request(context.Background(), "POST", path, req)
	if err != nil {
		return nil, err
	}

	// Set accept header for SSE
	httpReq.Header.Set("Accept", "text/event-stream")

	eventCh := make(chan StreamEvent, 100)

	go func() {
		defer close(eventCh)

		resp, err := s.client.httpClient.Do(httpReq)
		if err != nil {
			eventCh <- StreamEvent{
				Type: "error",
				Data: map[string]interface{}{"error": err.Error()},
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			eventCh <- StreamEvent{
				Type: "error",
				Data: map[string]interface{}{"error": resp.Status},
			}
			return
		}

		// Parse SSE stream
		scanner := bufio.NewScanner(resp.Body)
		var eventType string
		var eventData strings.Builder

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				eventData.WriteString(strings.TrimPrefix(line, "data: "))
			} else if line == "" && eventType != "" {
				// End of event
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(eventData.String()), &data); err == nil {
					eventCh <- StreamEvent{
						Type: eventType,
						Data: data,
					}
				}
				eventType = ""
				eventData.Reset()
			}
		}
	}()

	return eventCh, nil
}

// StreamContext is like Stream but with context
func (s *ChatService) StreamContext(ctx context.Context, agentID string, req ChatRequest) (<-chan StreamEvent, error) {
	// Similar to Stream but respects context cancellation
	// Implementation omitted for brevity
	return nil, nil
}
```

### 2.5 CLI Makefile

**`cli/Makefile`**:
```makefile
.PHONY: build install test lint clean release

# Variables
BINARY := agentctl
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Build
build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/agentctl

# Install locally
install: build
	cp bin/$(BINARY) $(GOPATH)/bin/

# Run tests
test:
	go test -v ./...

# Lint
lint:
	golangci-lint run

# Clean
clean:
	rm -rf bin/
	rm -rf dist/

# Release builds for all platforms
release: clean
	mkdir -p dist
	
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 ./cmd/agentctl
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 ./cmd/agentctl
	
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 ./cmd/agentctl
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 ./cmd/agentctl
	
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe ./cmd/agentctl
	
	# Create checksums
	cd dist && shasum -a 256 * > checksums.txt

# Generate shell completions
completions:
	mkdir -p completions
	bin/$(BINARY) completion bash > completions/$(BINARY).bash
	bin/$(BINARY) completion zsh > completions/$(BINARY).zsh
	bin/$(BINARY) completion fish > completions/$(BINARY).fish
```

---

## OpenAPI Documentation

### 2.6 OpenAPI Generation

Use `oapi-codegen` to generate types and optionally server stubs from OpenAPI spec:

**`api/openapi.yaml`** (excerpt):
```yaml
openapi: 3.0.3
info:
  title: AgentStack API
  version: 1.0.0
  description: API for managing AI agents on Kubernetes

servers:
  - url: https://api.agentstack.io/v1
    description: Production
  - url: http://localhost:8080/v1
    description: Local development

security:
  - BearerAuth: []

paths:
  /agents:
    get:
      summary: List agents
      operationId: listAgents
      tags: [Agents]
      parameters:
        - name: project_id
          in: query
          schema:
            type: string
        - name: limit
          in: query
          schema:
            type: integer
            default: 20
      responses:
        '200':
          description: List of agents
          content:
            application/json:
              schema:
                type: object
                properties:
                  agents:
                    type: array
                    items:
                      $ref: '#/components/schemas/Agent'

    post:
      summary: Create agent
      operationId: createAgent
      tags: [Agents]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateAgentRequest'
      responses:
        '201':
          description: Agent created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Agent'

components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer

  schemas:
    Agent:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        slug:
          type: string
        description:
          type: string
        status:
          type: string
          enum: [draft, deploying, active, failed, archived]
        project_id:
          type: string
        config:
          $ref: '#/components/schemas/AgentConfig'
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time

    AgentConfig:
      type: object
      properties:
        model:
          type: string
        tools:
          type: array
          items:
            type: string
        max_turns:
          type: integer

    CreateAgentRequest:
      type: object
      required:
        - project_id
        - name
      properties:
        project_id:
          type: string
        name:
          type: string
        description:
          type: string
        config:
          $ref: '#/components/schemas/AgentConfig'
```

---

## Deliverables Checklist

### Week 1 - CLI
- [ ] CLI project structure
- [ ] Login/auth command
- [ ] Config management
- [ ] Agent CRUD commands
- [ ] Deploy command with wait
- [ ] Interactive chat command
- [ ] Shell completions

### Week 2 - SDK & Docs
- [ ] SDK client implementation
- [ ] Agents service
- [ ] Chat service with streaming
- [ ] Deployments service
- [ ] OpenAPI spec
- [ ] SDK documentation
- [ ] CLI documentation

---

## Definition of Done

- [ ] `agentctl login` works with API key
- [ ] `agentctl agent list` shows agents
- [ ] `agentctl chat --agent xxx` works with streaming
- [ ] `agentctl deploy --wait` shows progress
- [ ] SDK can be imported: `import "github.com/raphaelmansuy/agentstack/sdk"`
- [ ] OpenAPI spec validates

---

## Sage AI Guidance

### CLI Best Practices

1. **Use Cobra**: It's the standard. Don't reinvent.
2. **Viper for Config**: Handles files, env vars, flags.
3. **Spinners for Waiting**: Use `briandowns/spinner` for deploy waits.
4. **Table Output**: Use `olekukonko/tablewriter` for tables.
5. **JSON Output**: Always support `-o json` for automation.

### SDK Best Practices

1. **Minimal Dependencies**: Only `net/http`, no heavy frameworks.
2. **Context Everywhere**: All methods should accept `context.Context`.
3. **Error Types**: Use typed errors (`APIError`) for better handling.
4. **Streaming**: Return channels, not callbacks.

### Distribution

| Platform | Method |
|----------|--------|
| macOS | Homebrew tap |
| Linux | Deb/RPM packages, curl install |
| Windows | Scoop/Chocolatey, exe download |
| Go developers | `go install github.com/raphaelmansuy/agentstack/cli/cmd/agentctl@latest` |

---

**Next Phase**: [005-enterprise-features.md](005-enterprise-features.md) - RBAC, Quotas, Audit
