package commands

import (
	"fmt"

	"github.com/raphaelmansuy/agentstack/cli/internal/config"
	"github.com/raphaelmansuy/agentstack/cli/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}
	cmd.AddCommand(newConfigViewCmd())
	cmd.AddCommand(newConfigProfilesCmd())
	cmd.AddCommand(newConfigUseProfileCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigDeleteProfileCmd())
	return cmd
}

func newConfigViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "View current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			loadedCfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			data, err := yaml.Marshal(loadedCfg)
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}
			fmt.Println(string(data))
			return nil
		},
	}
}

func newConfigProfilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profiles",
		Short: "List available profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			loadedCfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			formatter, err := getFormatter()
			if err != nil {
				return err
			}
			if formatter.Format() == output.FormatJSON || formatter.Format() == output.FormatYAML {
				profiles := make([]map[string]string, 0)
				for name := range loadedCfg.Profiles {
					current := ""
					if name == loadedCfg.CurrentProfile {
						current = "*"
					}
					profiles = append(profiles, map[string]string{
						"name":     name,
						"current":  current,
						"endpoint": loadedCfg.Profiles[name].Endpoint,
					})
				}
				return formatter.Print(profiles)
			}
			tp := output.NewTablePrinter()
			tp.SetHeaders("Name", "Current", "Endpoint")
			for name, profile := range loadedCfg.Profiles {
				current := ""
				if name == loadedCfg.CurrentProfile {
					current = "*"
				}
				tp.AddRow(name, current, profile.Endpoint)
			}
			return tp.Render()
		},
	}
}

func newConfigUseProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use-profile <name>",
		Short: "Switch to a different profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := args[0]
			loadedCfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if err := loadedCfg.UseProfile(profileName); err != nil {
				return err
			}
			if err := config.Save(loadedCfg, cfgFile); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Printf("Switched to profile: %s\n", profileName)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			loadedCfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if err := loadedCfg.Set(key, value); err != nil {
				return err
			}
			if err := config.Save(loadedCfg, cfgFile); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Printf("Set %s = %s\n", key, value)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			loadedCfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			value, err := loadedCfg.Get(key)
			if err != nil {
				return err
			}
			fmt.Println(value)
			return nil
		},
	}
}

func newConfigDeleteProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-profile <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := args[0]
			loadedCfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if err := loadedCfg.DeleteProfile(profileName); err != nil {
				return err
			}
			if err := config.Save(loadedCfg, cfgFile); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Printf("Deleted profile: %s\n", profileName)
			return nil
		},
	}
}
