// Package config provides configuration management for the agentctl CLI.
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

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the CLI configuration.
type Config struct {
	// CurrentProfile is the name of the active profile.
	CurrentProfile string `yaml:"current_profile"`
	// Profiles is a map of profile names to their configurations.
	Profiles map[string]*Profile `yaml:"profiles"`
}

// Profile represents a named configuration profile.
type Profile struct {
	// Endpoint is the API endpoint URL.
	Endpoint string `yaml:"endpoint"`
	// APIKey is the API key for authentication.
	APIKey string `yaml:"api_key,omitempty"`
	// Token is the JWT token for authentication.
	Token string `yaml:"token,omitempty"`
	// RefreshToken is the refresh token for token renewal.
	RefreshToken string `yaml:"refresh_token,omitempty"`
	// DefaultProject is the default project for operations.
	DefaultProject string `yaml:"default_project,omitempty"`
	// DefaultOutput is the default output format.
	DefaultOutput string `yaml:"default_output,omitempty"`
}

// DefaultConfigDir returns the default configuration directory.
func DefaultConfigDir() string {
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "agentstack")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".agentstack")
	}
	return filepath.Join(home, ".config", "agentstack")
}

// DefaultConfigPath returns the default configuration file path.
func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.yaml")
}

// Load loads the configuration from the given path.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &Config{
				CurrentProfile: "default",
				Profiles: map[string]*Profile{
					"default": {
						Endpoint:      "http://localhost:8080/api",
						DefaultOutput: "table",
					},
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize profiles map if nil
	if config.Profiles == nil {
		config.Profiles = make(map[string]*Profile)
	}

	return &config, nil
}

// Save saves the configuration to the given path.
func Save(config *Config, path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CurrentProfile returns the current active profile.
func (c *Config) CurrentProfileConfig() *Profile {
	if c.Profiles == nil {
		return nil
	}
	return c.Profiles[c.CurrentProfile]
}

// SetProfile sets or updates a profile.
func (c *Config) SetProfile(name string, profile *Profile) {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}
	c.Profiles[name] = profile
}

// DeleteProfile removes a profile by name.
func (c *Config) DeleteProfile(name string) error {
	if c.Profiles == nil {
		return fmt.Errorf("profile not found: %s", name)
	}
	if _, exists := c.Profiles[name]; !exists {
		return fmt.Errorf("profile not found: %s", name)
	}
	if name == c.CurrentProfile {
		return fmt.Errorf("cannot delete current profile: %s", name)
	}
	delete(c.Profiles, name)
	return nil
}

// UseProfile sets the current profile.
func (c *Config) UseProfile(name string) error {
	if c.Profiles == nil {
		return fmt.Errorf("profile not found: %s", name)
	}
	if _, exists := c.Profiles[name]; !exists {
		return fmt.Errorf("profile not found: %s", name)
	}
	c.CurrentProfile = name
	return nil
}

// ListProfiles returns a list of all profile names.
func (c *Config) ListProfiles() []string {
	if c.Profiles == nil {
		return nil
	}
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

// Get retrieves a configuration value from the current profile.
func (c *Config) Get(key string) (string, error) {
	profile := c.CurrentProfileConfig()
	if profile == nil {
		return "", fmt.Errorf("no current profile configured")
	}

	switch key {
	case "endpoint":
		return profile.Endpoint, nil
	case "api_key":
		return profile.APIKey, nil
	case "token":
		return profile.Token, nil
	case "refresh_token":
		return profile.RefreshToken, nil
	case "default_project":
		return profile.DefaultProject, nil
	case "default_output":
		return profile.DefaultOutput, nil
	default:
		return "", fmt.Errorf("unknown configuration key: %s", key)
	}
}

// Set sets a configuration value in the current profile.
func (c *Config) Set(key, value string) error {
	profile := c.CurrentProfileConfig()
	if profile == nil {
		return fmt.Errorf("no current profile configured")
	}

	switch key {
	case "endpoint":
		profile.Endpoint = value
	case "api_key":
		profile.APIKey = value
	case "token":
		profile.Token = value
	case "refresh_token":
		profile.RefreshToken = value
	case "default_project":
		profile.DefaultProject = value
	case "default_output":
		profile.DefaultOutput = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}
