// Package health provides comprehensive health check functionality.
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
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
	StatusUnknown   Status = "unknown"
)

// CheckResult represents the result of a health check.
type CheckResult struct {
	Status      Status            `json:"status"`
	Message     string            `json:"message,omitempty"`
	Latency     time.Duration     `json:"latency_ms"`
	LastChecked time.Time         `json:"last_checked"`
	Details     map[string]string `json:"details,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for CheckResult.
func (r CheckResult) MarshalJSON() ([]byte, error) {
	type Alias CheckResult
	return json.Marshal(&struct {
		*Alias
		Latency int64 `json:"latency_ms"`
	}{
		Alias:   (*Alias)(&r),
		Latency: r.Latency.Milliseconds(),
	})
}

// Check is a function that performs a health check.
type Check func(ctx context.Context) CheckResult

// Checker manages health checks.
type Checker struct {
	mu          sync.RWMutex
	checks      map[string]Check
	cache       map[string]CheckResult
	cacheTTL    time.Duration
	startTime   time.Time
	readySignal chan struct{}
	ready       bool
}

// NewChecker creates a new health checker.
func NewChecker(cacheTTL time.Duration) *Checker {
	return &Checker{
		checks:      make(map[string]Check),
		cache:       make(map[string]CheckResult),
		cacheTTL:    cacheTTL,
		startTime:   time.Now(),
		readySignal: make(chan struct{}),
	}
}

// Register registers a health check.
func (c *Checker) Register(name string, check Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = check
}

// SetReady signals that the service is ready.
func (c *Checker) SetReady() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ready {
		c.ready = true
		close(c.readySignal)
	}
}

// IsReady returns whether the service is ready.
func (c *Checker) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

// RunAll runs all health checks.
func (c *Checker) RunAll(ctx context.Context) map[string]CheckResult {
	c.mu.RLock()
	checks := make(map[string]Check, len(c.checks))
	for name, check := range c.checks {
		checks[name] = check
	}
	c.mu.RUnlock()

	results := make(map[string]CheckResult, len(checks))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, check := range checks {
		wg.Add(1)
		go func(name string, check Check) {
			defer wg.Done()

			// Check cache first
			c.mu.RLock()
			cached, ok := c.cache[name]
			c.mu.RUnlock()

			if ok && time.Since(cached.LastChecked) < c.cacheTTL {
				mu.Lock()
				results[name] = cached
				mu.Unlock()
				return
			}

			// Run the check
			result := check(ctx)
			result.LastChecked = time.Now()

			// Update cache
			c.mu.Lock()
			c.cache[name] = result
			c.mu.Unlock()

			mu.Lock()
			results[name] = result
			mu.Unlock()
		}(name, check)
	}

	wg.Wait()
	return results
}

// OverallStatus returns the overall health status based on all checks.
func (c *Checker) OverallStatus(results map[string]CheckResult) Status {
	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		switch result.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		case StatusHealthy, StatusUnknown:
			// No action needed
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}
	return StatusHealthy
}

// HealthResponse represents the full health response.
type HealthResponse struct {
	Status  Status                 `json:"status"`
	Version string                 `json:"version"`
	Uptime  string                 `json:"uptime"`
	Checks  map[string]CheckResult `json:"checks"`
	System  SystemInfo             `json:"system"`
}

// SystemInfo contains system information.
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	MemoryAlloc  uint64 `json:"memory_alloc_mb"`
}

// FullCheck performs all health checks and returns a complete response.
func (c *Checker) FullCheck(ctx context.Context, version string) HealthResponse {
	results := c.RunAll(ctx)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return HealthResponse{
		Status:  c.OverallStatus(results),
		Version: version,
		Uptime:  time.Since(c.startTime).Round(time.Second).String(),
		Checks:  results,
		System: SystemInfo{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			MemoryAlloc:  memStats.Alloc / 1024 / 1024,
		},
	}
}

// HTTPHandler returns HTTP handlers for health checks.
type HTTPHandler struct {
	checker *Checker
	version string
}

// NewHTTPHandler creates a new HTTP handler for health checks.
func NewHTTPHandler(checker *Checker, version string) *HTTPHandler {
	return &HTTPHandler{
		checker: checker,
		version: version,
	}
}

// LivenessHandler handles liveness probe requests.
func (h *HTTPHandler) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ReadinessHandler handles readiness probe requests.
func (h *HTTPHandler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !h.checker.IsReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "not_ready",
			"ready":  false,
		})
		return
	}

	// Check critical dependencies
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	results := h.checker.RunAll(ctx)
	status := h.checker.OverallStatus(results)

	if status == StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": status,
		"ready":  status != StatusUnhealthy,
	})
}

// StartupHandler handles startup probe requests.
func (h *HTTPHandler) StartupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !h.checker.IsReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "starting",
			"started": false,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "started",
		"started": true,
	})
}

// HealthHandler handles detailed health check requests.
func (h *HTTPHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response := h.checker.FullCheck(ctx, h.version)

	w.Header().Set("Content-Type", "application/json")

	switch response.Status {
	case StatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	case StatusDegraded:
		w.WriteHeader(http.StatusOK) // Still OK but degraded
	default:
		w.WriteHeader(http.StatusOK)
	}

	_ = json.NewEncoder(w).Encode(response)
}

// Common health checks

// DatabaseCheck creates a database health check.
func DatabaseCheck(pingFn func(context.Context) error) Check {
	return func(ctx context.Context) CheckResult {
		start := time.Now()
		err := pingFn(ctx)
		latency := time.Since(start)

		if err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: fmt.Sprintf("database connection failed: %v", err),
				Latency: latency,
			}
		}

		status := StatusHealthy
		if latency > 100*time.Millisecond {
			status = StatusDegraded
		}

		return CheckResult{
			Status:  status,
			Message: "database is healthy",
			Latency: latency,
			Details: map[string]string{
				"response_time": latency.String(),
			},
		}
	}
}

// RedisCheck creates a Redis health check.
func RedisCheck(pingFn func(context.Context) error) Check {
	return func(ctx context.Context) CheckResult {
		start := time.Now()
		err := pingFn(ctx)
		latency := time.Since(start)

		if err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: fmt.Sprintf("redis connection failed: %v", err),
				Latency: latency,
			}
		}

		status := StatusHealthy
		if latency > 50*time.Millisecond {
			status = StatusDegraded
		}

		return CheckResult{
			Status:  status,
			Message: "redis is healthy",
			Latency: latency,
		}
	}
}

// MemoryCheck creates a memory usage health check.
func MemoryCheck(maxMB uint64) Check {
	return func(ctx context.Context) CheckResult {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		allocMB := memStats.Alloc / 1024 / 1024
		status := StatusHealthy

		if allocMB > maxMB {
			status = StatusUnhealthy
		} else if allocMB > maxMB*80/100 {
			status = StatusDegraded
		}

		return CheckResult{
			Status:  status,
			Message: fmt.Sprintf("memory usage: %d MB", allocMB),
			Details: map[string]string{
				"alloc_mb":       fmt.Sprintf("%d", allocMB),
				"total_alloc_mb": fmt.Sprintf("%d", memStats.TotalAlloc/1024/1024),
				"sys_mb":         fmt.Sprintf("%d", memStats.Sys/1024/1024),
				"num_gc":         fmt.Sprintf("%d", memStats.NumGC),
			},
		}
	}
}

// GoroutineCheck creates a goroutine count health check.
func GoroutineCheck(maxGoroutines int) Check {
	return func(ctx context.Context) CheckResult {
		count := runtime.NumGoroutine()
		status := StatusHealthy

		if count > maxGoroutines {
			status = StatusUnhealthy
		} else if count > maxGoroutines*80/100 {
			status = StatusDegraded
		}

		return CheckResult{
			Status:  status,
			Message: fmt.Sprintf("goroutine count: %d", count),
			Details: map[string]string{
				"count": fmt.Sprintf("%d", count),
				"max":   fmt.Sprintf("%d", maxGoroutines),
			},
		}
	}
}
