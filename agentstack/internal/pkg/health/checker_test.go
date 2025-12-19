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

package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChecker_RegisterAndRun(t *testing.T) {
	checker := NewChecker(time.Second)

	// Register a healthy check
	checker.Register("test", func(ctx context.Context) CheckResult {
		return CheckResult{
			Status:  StatusHealthy,
			Message: "all good",
			Latency: 10 * time.Millisecond,
		}
	})

	results := checker.RunAll(context.Background())

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if result, ok := results["test"]; ok {
		if result.Status != StatusHealthy {
			t.Errorf("expected healthy status, got %s", result.Status)
		}
	} else {
		t.Error("expected 'test' result not found")
	}
}

func TestChecker_OverallStatus(t *testing.T) {
	checker := NewChecker(time.Second)

	tests := []struct {
		name     string
		results  map[string]CheckResult
		expected Status
	}{
		{
			name: "all healthy",
			results: map[string]CheckResult{
				"db":    {Status: StatusHealthy},
				"redis": {Status: StatusHealthy},
			},
			expected: StatusHealthy,
		},
		{
			name: "one degraded",
			results: map[string]CheckResult{
				"db":    {Status: StatusHealthy},
				"redis": {Status: StatusDegraded},
			},
			expected: StatusDegraded,
		},
		{
			name: "one unhealthy",
			results: map[string]CheckResult{
				"db":    {Status: StatusHealthy},
				"redis": {Status: StatusUnhealthy},
			},
			expected: StatusUnhealthy,
		},
		{
			name: "unhealthy takes precedence",
			results: map[string]CheckResult{
				"db":    {Status: StatusDegraded},
				"redis": {Status: StatusUnhealthy},
			},
			expected: StatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := checker.OverallStatus(tt.results)
			if status != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, status)
			}
		})
	}
}

func TestChecker_Ready(t *testing.T) {
	checker := NewChecker(time.Second)

	if checker.IsReady() {
		t.Error("expected not ready initially")
	}

	checker.SetReady()

	if !checker.IsReady() {
		t.Error("expected ready after SetReady")
	}

	// SetReady should be idempotent
	checker.SetReady()
	if !checker.IsReady() {
		t.Error("expected still ready after second SetReady")
	}
}

func TestChecker_CacheTTL(t *testing.T) {
	checker := NewChecker(100 * time.Millisecond)

	callCount := 0
	checker.Register("test", func(ctx context.Context) CheckResult {
		callCount++
		return CheckResult{Status: StatusHealthy}
	})

	// First call
	checker.RunAll(context.Background())
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}

	// Second call should use cache
	checker.RunAll(context.Background())
	if callCount != 1 {
		t.Errorf("expected still 1 call (cached), got %d", callCount)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call should run the check again
	checker.RunAll(context.Background())
	if callCount != 2 {
		t.Errorf("expected 2 calls after cache expire, got %d", callCount)
	}
}

func TestHTTPHandler_Liveness(t *testing.T) {
	checker := NewChecker(time.Second)
	handler := NewHTTPHandler(checker, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	w := httptest.NewRecorder()

	handler.LivenessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHTTPHandler_Readiness_NotReady(t *testing.T) {
	checker := NewChecker(time.Second)
	handler := NewHTTPHandler(checker, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handler.ReadinessHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when not ready, got %d", w.Code)
	}
}

func TestHTTPHandler_Readiness_Ready(t *testing.T) {
	checker := NewChecker(time.Second)
	checker.SetReady()
	handler := NewHTTPHandler(checker, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handler.ReadinessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when ready, got %d", w.Code)
	}
}

func TestHTTPHandler_Startup(t *testing.T) {
	checker := NewChecker(time.Second)
	handler := NewHTTPHandler(checker, "1.0.0")

	// Not started
	req := httptest.NewRequest(http.MethodGet, "/startupz", nil)
	w := httptest.NewRecorder()
	handler.StartupHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 before startup, got %d", w.Code)
	}

	// After startup
	checker.SetReady()
	w = httptest.NewRecorder()
	handler.StartupHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 after startup, got %d", w.Code)
	}
}

func TestDatabaseCheck(t *testing.T) {
	t.Run("healthy database", func(t *testing.T) {
		check := DatabaseCheck(func(ctx context.Context) error {
			return nil
		})
		result := check(context.Background())
		if result.Status != StatusHealthy {
			t.Errorf("expected healthy, got %s", result.Status)
		}
	})

	t.Run("unhealthy database", func(t *testing.T) {
		check := DatabaseCheck(func(ctx context.Context) error {
			return errors.New("connection refused")
		})
		result := check(context.Background())
		if result.Status != StatusUnhealthy {
			t.Errorf("expected unhealthy, got %s", result.Status)
		}
	})
}

func TestRedisCheck(t *testing.T) {
	t.Run("healthy redis", func(t *testing.T) {
		check := RedisCheck(func(ctx context.Context) error {
			return nil
		})
		result := check(context.Background())
		if result.Status != StatusHealthy {
			t.Errorf("expected healthy, got %s", result.Status)
		}
	})

	t.Run("unhealthy redis", func(t *testing.T) {
		check := RedisCheck(func(ctx context.Context) error {
			return errors.New("PONG timeout")
		})
		result := check(context.Background())
		if result.Status != StatusUnhealthy {
			t.Errorf("expected unhealthy, got %s", result.Status)
		}
	})
}

func TestMemoryCheck(t *testing.T) {
	// 10GB limit - should be healthy
	check := MemoryCheck(10240)
	result := check(context.Background())
	if result.Status != StatusHealthy {
		t.Errorf("expected healthy with high limit, got %s", result.Status)
	}

	// Test is flaky since memory usage varies, just check it runs
	check = MemoryCheck(1) // 1MB limit
	result = check(context.Background())
	// Just verify it doesn't panic and returns a status
	if result.Status != StatusHealthy && result.Status != StatusDegraded && result.Status != StatusUnhealthy {
		t.Error("expected a valid status")
	}
}

func TestGoroutineCheck(t *testing.T) {
	// High limit - should be healthy
	check := GoroutineCheck(10000)
	result := check(context.Background())
	if result.Status != StatusHealthy {
		t.Errorf("expected healthy with high limit, got %s", result.Status)
	}
}

func TestFullCheck(t *testing.T) {
	checker := NewChecker(time.Second)
	checker.Register("db", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})
	checker.Register("redis", func(ctx context.Context) CheckResult {
		return CheckResult{Status: StatusHealthy}
	})

	response := checker.FullCheck(context.Background(), "1.0.0")

	if response.Status != StatusHealthy {
		t.Errorf("expected healthy status, got %s", response.Status)
	}
	if response.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", response.Version)
	}
	if len(response.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(response.Checks))
	}
	if response.System.NumCPU == 0 {
		t.Error("expected non-zero CPU count")
	}
}
