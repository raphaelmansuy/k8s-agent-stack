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
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigDir(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	t.Run("with XDG_CONFIG_HOME", func(t *testing.T) {
		originalXDG := os.Getenv("XDG_CONFIG_HOME")
		defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

		os.Setenv("XDG_CONFIG_HOME", "/custom/config")
		expected := "/custom/config/agentstack"
		if got := DefaultConfigDir(); got != expected {
			t.Errorf("DefaultConfigDir() = %v, want %v", got, expected)
		}
	})

	// Test without XDG_CONFIG_HOME
	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		originalXDG := os.Getenv("XDG_CONFIG_HOME")
		defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

		os.Unsetenv("XDG_CONFIG_HOME")
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", "agentstack")
		if got := DefaultConfigDir(); got != expected {
			t.Errorf("DefaultConfigDir() = %v, want %v", got, expected)
		}
	})
}

func TestLoadSave(t *testing.T) {
	// Create a temporary directory for test configs
	tmpDir, err := os.MkdirTemp("", "agentstack-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test loading non-existent config (should return defaults)
	t.Run("load non-existent", func(t *testing.T) {
		config, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if config.CurrentProfile != "default" {
			t.Errorf("CurrentProfile = %v, want default", config.CurrentProfile)
		}
		if config.Profiles["default"] == nil {
			t.Error("Expected default profile to exist")
		}
	})

	// Test saving and loading config
	t.Run("save and load", func(t *testing.T) {
		config := &Config{
			CurrentProfile: "production",
			Profiles: map[string]*Profile{
				"production": {
					Endpoint:       "https://api.example.com",
					APIKey:         "secret-key",
					DefaultProject: "my-project",
					DefaultOutput:  "json",
				},
			},
		}

		err := Save(config, configPath)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		loaded, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if loaded.CurrentProfile != "production" {
			t.Errorf("CurrentProfile = %v, want production", loaded.CurrentProfile)
		}
		if loaded.Profiles["production"].Endpoint != "https://api.example.com" {
			t.Errorf("Endpoint = %v, want https://api.example.com", loaded.Profiles["production"].Endpoint)
		}
	})
}

func TestConfigProfiles(t *testing.T) {
	config := &Config{
		CurrentProfile: "default",
		Profiles: map[string]*Profile{
			"default": {
				Endpoint: "http://localhost:8080",
			},
		},
	}

	// Test CurrentProfileConfig
	t.Run("CurrentProfileConfig", func(t *testing.T) {
		profile := config.CurrentProfileConfig()
		if profile == nil {
			t.Fatal("CurrentProfileConfig() returned nil")
		}
		if profile.Endpoint != "http://localhost:8080" {
			t.Errorf("Endpoint = %v, want http://localhost:8080", profile.Endpoint)
		}
	})

	// Test SetProfile
	t.Run("SetProfile", func(t *testing.T) {
		config.SetProfile("staging", &Profile{
			Endpoint: "https://staging.example.com",
		})
		if config.Profiles["staging"] == nil {
			t.Error("Expected staging profile to exist")
		}
	})

	// Test UseProfile
	t.Run("UseProfile", func(t *testing.T) {
		err := config.UseProfile("staging")
		if err != nil {
			t.Fatalf("UseProfile() error = %v", err)
		}
		if config.CurrentProfile != "staging" {
			t.Errorf("CurrentProfile = %v, want staging", config.CurrentProfile)
		}
	})

	// Test UseProfile with non-existent profile
	t.Run("UseProfile non-existent", func(t *testing.T) {
		err := config.UseProfile("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent profile")
		}
	})

	// Test ListProfiles
	t.Run("ListProfiles", func(t *testing.T) {
		profiles := config.ListProfiles()
		if len(profiles) != 2 {
			t.Errorf("ListProfiles() returned %d profiles, want 2", len(profiles))
		}
	})

	// Test DeleteProfile
	t.Run("DeleteProfile", func(t *testing.T) {
		// Can't delete current profile
		config.CurrentProfile = "default"
		err := config.DeleteProfile("default")
		if err == nil {
			t.Error("Expected error when deleting current profile")
		}

		// Can delete non-current profile
		err = config.DeleteProfile("staging")
		if err != nil {
			t.Fatalf("DeleteProfile() error = %v", err)
		}
		if config.Profiles["staging"] != nil {
			t.Error("Expected staging profile to be deleted")
		}
	})
}

func TestConfigGetSet(t *testing.T) {
	config := &Config{
		CurrentProfile: "default",
		Profiles: map[string]*Profile{
			"default": {
				Endpoint:      "http://localhost:8080",
				DefaultOutput: "table",
			},
		},
	}

	// Test Get
	t.Run("Get", func(t *testing.T) {
		endpoint, err := config.Get("endpoint")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if endpoint != "http://localhost:8080" {
			t.Errorf("Get(endpoint) = %v, want http://localhost:8080", endpoint)
		}
	})

	// Test Get unknown key
	t.Run("Get unknown key", func(t *testing.T) {
		_, err := config.Get("unknown")
		if err == nil {
			t.Error("Expected error for unknown key")
		}
	})

	// Test Set
	t.Run("Set", func(t *testing.T) {
		err := config.Set("endpoint", "https://api.example.com")
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		endpoint, _ := config.Get("endpoint")
		if endpoint != "https://api.example.com" {
			t.Errorf("Endpoint after Set = %v, want https://api.example.com", endpoint)
		}
	})

	// Test Set unknown key
	t.Run("Set unknown key", func(t *testing.T) {
		err := config.Set("unknown", "value")
		if err == nil {
			t.Error("Expected error for unknown key")
		}
	})
}
