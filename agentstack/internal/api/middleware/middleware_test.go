package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultRateLimitKeyFunc(t *testing.T) {
	tests := []struct {
		name     string
		xff      string
		remote   string
		expected string
	}{
		{
			name:     "with X-Forwarded-For",
			xff:      "192.168.1.1",
			remote:   "127.0.0.1:8080",
			expected: "192.168.1.1",
		},
		{
			name:     "without X-Forwarded-For",
			xff:      "",
			remote:   "127.0.0.1:8080",
			expected: "127.0.0.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remote
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}

			result := DefaultRateLimitKeyFunc(req)
			if result != tt.expected {
				t.Errorf("DefaultRateLimitKeyFunc() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRateLimitConfig(t *testing.T) {
	config := RateLimitConfig{
		Requests: 100,
		Window:   time.Minute,
		KeyFunc:  DefaultRateLimitKeyFunc,
	}

	if config.Requests != 100 {
		t.Errorf("expected Requests 100, got %d", config.Requests)
	}

	if config.Window != time.Minute {
		t.Errorf("expected Window 1 minute, got %s", config.Window)
	}
}

func TestRateLimitWithoutRedis(t *testing.T) {
	config := RateLimitConfig{
		Requests: 10,
		Window:   time.Second,
		KeyFunc:  DefaultRateLimitKeyFunc,
	}

	middleware := RateLimit(nil, config)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestGetTeamID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "with team ID",
			ctx:      SetAuthInContext(context.Background(), &AuthInfo{TeamID: "team-123"}),
			expected: "team-123",
		},
		{
			name:     "without team ID",
			ctx:      context.Background(),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTeamID(tt.ctx)
			if result != tt.expected {
				t.Errorf("GetTeamID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetProjectID(t *testing.T) {
	ctx := SetAuthInContext(context.Background(), &AuthInfo{ProjectID: "project-456"})
	result := GetProjectID(ctx)
	if result != "project-456" {
		t.Errorf("GetProjectID() = %v, want project-456", result)
	}

	result = GetProjectID(context.Background())
	if result != "" {
		t.Errorf("GetProjectID() on empty context = %v, want empty string", result)
	}
}

func TestGetUserID(t *testing.T) {
	ctx := SetAuthInContext(context.Background(), &AuthInfo{UserID: "user-789"})
	result := GetUserID(ctx)
	if result != "user-789" {
		t.Errorf("GetUserID() = %v, want user-789", result)
	}
}

func TestTenantContext(t *testing.T) {
	mw := TenantContext()

	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestIdempotencyWithoutRedis(t *testing.T) {
	mw := Idempotency(nil)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Idempotency-Key", "key-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestIdempotencySkipsGET(t *testing.T) {
	mw := Idempotency(nil)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Idempotency-Key", "key-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
