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
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Environment != "development" {
		t.Errorf("expected default environment 'development', got %s", cfg.Environment)
	}

	if cfg.Server.ReadTimeout != 15*time.Second {
		t.Errorf("expected read timeout 15s, got %s", cfg.Server.ReadTimeout)
	}
}

func TestLoadWithEnv(t *testing.T) {
	os.Setenv("AGENTSTACK_SERVER_PORT", "9090")
	os.Setenv("AGENTSTACK_ENVIRONMENT", "production")
	defer func() {
		os.Unsetenv("AGENTSTACK_SERVER_PORT")
		os.Unsetenv("AGENTSTACK_ENVIRONMENT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090 from env, got %d", cfg.Server.Port)
	}

	if cfg.Environment != "production" {
		t.Errorf("expected environment 'production' from env, got %s", cfg.Environment)
	}
}
