# Advanced Cobra Patterns

## Command Groups

```go
// Use command groups for organization
var agentCmd = &cobra.Command{
    Use:   "agent",
    Short: "Manage agents",
    Annotations: map[string]string{
        "group": "core",
    },
}

// In root.go, group commands
func init() {
    rootCmd.AddGroup(&cobra.Group{
        ID:    "core",
        Title: "Core Commands:",
    })
    rootCmd.AddGroup(&cobra.Group{
        ID:    "management",
        Title: "Management Commands:",
    })
}
```

## Persistent Pre/Post Run

```go
var rootCmd = &cobra.Command{
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // Runs before every command
        // Good for: loading config, initializing client
        return initConfig()
    },
    PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
        // Runs after every command
        // Good for: cleanup, analytics
        return nil
    },
}

// Skip for specific commands
var versionCmd = &cobra.Command{
    Use:   "version",
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // Override to skip config loading
        return nil
    },
}
```

## Flag Inheritance

```go
// Flags that apply to command and all subcommands
agentCmd.PersistentFlags().StringP("project", "p", "", "project to use")

// Flags only for this command
agentCmd.Flags().Bool("all", false, "show all agents")

// Required flags
agentDeployCmd.MarkFlagRequired("name")

// Mutually exclusive flags
agentCmd.MarkFlagsMutuallyExclusive("name", "all")

// Flags that must be used together
agentCmd.MarkFlagsRequiredTogether("region", "zone")
```

## Custom Flag Types

```go
// Duration flag with custom parsing
type durationValue time.Duration

func (d *durationValue) String() string {
    return time.Duration(*d).String()
}

func (d *durationValue) Set(s string) error {
    v, err := time.ParseDuration(s)
    if err != nil {
        return err
    }
    *d = durationValue(v)
    return nil
}

func (d *durationValue) Type() string {
    return "duration"
}

// Usage
var timeout durationValue
cmd.Flags().Var(&timeout, "timeout", "operation timeout")
```

## Context and Cancellation

```go
func runLongOperation(cmd *cobra.Command, args []string) error {
    // Use command context for cancellation
    ctx := cmd.Context()
    
    // Or create with timeout
    ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
    defer cancel()
    
    // Pass to operations
    result, err := client.Deploy(ctx, args[0])
    if err != nil {
        if errors.Is(err, context.Canceled) {
            return fmt.Errorf("operation cancelled")
        }
        return err
    }
    return nil
}
```

## Shell Completion

```go
// Static completions
cmd.RegisterFlagCompletionFunc("template", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return []string{"google-adk", "langchain", "custom"}, cobra.ShellCompDirectiveDefault
})

// Dynamic completions (from API)
cmd.RegisterFlagCompletionFunc("project", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    client := client.New(config.Load())
    projects, err := client.Projects.List(cmd.Context())
    if err != nil {
        return nil, cobra.ShellCompDirectiveError
    }
    
    var names []string
    for _, p := range projects {
        names = append(names, p.Name)
    }
    return names, cobra.ShellCompDirectiveDefault
})

// Arg completions
var agentDeleteCmd = &cobra.Command{
    Use:               "delete <name>",
    ValidArgsFunction: completeAgentNames,
}

func completeAgentNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    if len(args) > 0 {
        return nil, cobra.ShellCompDirectiveNoFileComp
    }
    
    client := client.New(config.Load())
    agents, _ := client.Agents.List(cmd.Context())
    
    var names []string
    for _, a := range agents {
        names = append(names, a.Name)
    }
    return names, cobra.ShellCompDirectiveDefault
}
```

## Error Handling

```go
// Custom error types for CLI
type UserError struct {
    Msg  string
    Hint string
}

func (e *UserError) Error() string {
    return e.Msg
}

// In main.go
func main() {
    if err := rootCmd.Execute(); err != nil {
        switch e := err.(type) {
        case *UserError:
            output.Error(e.Msg)
            if e.Hint != "" {
                fmt.Println("\nHint:", e.Hint)
            }
            os.Exit(1)
        default:
            output.Error(err.Error())
            os.Exit(1)
        }
    }
}

// Usage
return &UserError{
    Msg:  "project not found",
    Hint: "Run 'agentctl project list' to see available projects",
}
```

## Interactive Mode

```go
func interactiveMode(cmd *cobra.Command) error {
    // Check if running interactively
    if !term.IsTerminal(int(os.Stdin.Fd())) {
        return fmt.Errorf("interactive mode requires a terminal")
    }
    
    // Prompt for missing values
    name, _ := cmd.Flags().GetString("name")
    if name == "" {
        name, _ = prompt.Input("Agent name:", "my-agent")
    }
    
    template, _ := cmd.Flags().GetString("template")
    if template == "" {
        template, _ = prompt.Select("Template:", []string{
            "google-adk",
            "langchain",
            "custom",
        })
    }
    
    // Continue with values
    return scaffoldAgent(name, template)
}
```

## Progress and Spinners

```go
import "github.com/briandowns/spinner"

func deployWithProgress(ctx context.Context, path string) error {
    s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
    s.Suffix = " Deploying..."
    s.Start()
    defer s.Stop()
    
    // Update during operation
    s.Suffix = " Building image..."
    // ... build
    
    s.Suffix = " Pushing to registry..."
    // ... push
    
    s.Suffix = " Creating resources..."
    // ... create
    
    s.FinalMSG = "✓ Deployed successfully\n"
    return nil
}
```

## Parallel Flag Processing

```go
func init() {
    // Process flags in parallel for faster startup
    var wg sync.WaitGroup
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        viper.BindPFlag("endpoint", rootCmd.PersistentFlags().Lookup("endpoint"))
    }()
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        viper.BindEnv("token", "AGENTCTL_TOKEN")
    }()
    
    wg.Wait()
}
```

## Help Templates

```go
func init() {
    // Custom help template
    rootCmd.SetHelpTemplate(`{{.Long}}

Usage:
  {{.UseLine}}

{{if .HasAvailableSubCommands}}Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}

{{if .HasAvailableLocalFlags}}Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`)
    
    // Custom usage template
    rootCmd.SetUsageTemplate(`Usage:
  {{.UseLine}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}

Examples:
  # Create a new agent
  agentctl agent init my-agent --template google-adk
  
  # Deploy an agent
  agentctl agent deploy --wait
  
  # Run evaluation
  agentctl eval run my-agent --dataset golden.yaml
`)
}
```
