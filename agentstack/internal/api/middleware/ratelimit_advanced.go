// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// AdvancedRateLimiter defines the interface for advanced rate limiters.
type AdvancedRateLimiter interface {
	// Allow checks if a request is allowed for the given key.
	Allow(ctx context.Context, key string) (allowed bool, remaining int, resetAt time.Time, err error)
}

// SlidingWindowConfig configures the sliding window rate limiter.
type SlidingWindowConfig struct {
	// RequestsPerWindow is the number of requests allowed per window.
	RequestsPerWindow int
	// WindowSize is the duration of the sliding window.
	WindowSize time.Duration
	// BurstSize allows temporary bursts above the limit.
	BurstSize int
	// KeyPrefix is the Redis key prefix.
	KeyPrefix string
}

// DefaultSlidingWindowConfig returns default sliding window configuration.
func DefaultSlidingWindowConfig() SlidingWindowConfig {
	return SlidingWindowConfig{
		RequestsPerWindow: 100,
		WindowSize:        time.Minute,
		BurstSize:         10,
		KeyPrefix:         "ratelimit:",
	}
}

// SlidingWindowLimiter implements a sliding window rate limiter using Redis.
type SlidingWindowLimiter struct {
	client *redis.Client
	config SlidingWindowConfig
}

// Lua script for atomic sliding window rate limiting
var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

-- Count current requests
local count = redis.call('ZCARD', key)

if count < limit then
    -- Add new request with current timestamp as score
    redis.call('ZADD', key, now, now .. ':' .. math.random())
    redis.call('EXPIRE', key, math.ceil(window / 1000))
    return {1, limit - count - 1, now + window}
else
    -- Get the oldest entry to calculate reset time
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local reset = now + window
    if #oldest > 0 then
        reset = tonumber(oldest[2]) + window
    end
    return {0, 0, reset}
end
`)

// NewSlidingWindowLimiter creates a new Redis-based sliding window rate limiter.
func NewSlidingWindowLimiter(client *redis.Client, config SlidingWindowConfig) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		client: client,
		config: config,
	}
}

// Allow checks if a request is allowed for the given key.
func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, int, time.Time, error) {
	fullKey := l.config.KeyPrefix + key
	now := time.Now().UnixMilli()
	windowMs := l.config.WindowSize.Milliseconds()
	limit := l.config.RequestsPerWindow + l.config.BurstSize

	result, err := slidingWindowScript.Run(ctx, l.client, []string{fullKey},
		now, windowMs, limit).Slice()
	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("rate limit script failed: %w", err)
	}

	allowed := result[0].(int64) == 1
	remaining := int(result[1].(int64))
	resetMs := result[2].(int64)
	resetAt := time.UnixMilli(resetMs)

	return allowed, remaining, resetAt, nil
}

// InMemoryRateLimiter provides an in-memory rate limiter for testing or single-instance deployments.
type InMemoryRateLimiter struct {
	config  SlidingWindowConfig
	windows sync.Map // map[string]*rateLimitWindow
}

type rateLimitWindow struct {
	mu       sync.Mutex
	requests []int64 // timestamps in milliseconds
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter.
func NewInMemoryRateLimiter(config SlidingWindowConfig) *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		config: config,
	}
}

// Allow checks if a request is allowed for the given key.
func (l *InMemoryRateLimiter) Allow(ctx context.Context, key string) (bool, int, time.Time, error) {
	now := time.Now().UnixMilli()
	windowMs := l.config.WindowSize.Milliseconds()
	limit := l.config.RequestsPerWindow + l.config.BurstSize

	// Get or create window
	val, _ := l.windows.LoadOrStore(key, &rateLimitWindow{requests: make([]int64, 0, limit)})
	w := val.(*rateLimitWindow)

	w.mu.Lock()
	defer w.mu.Unlock()

	// Remove expired entries
	cutoff := now - windowMs
	validRequests := make([]int64, 0, len(w.requests))
	for _, ts := range w.requests {
		if ts > cutoff {
			validRequests = append(validRequests, ts)
		}
	}
	w.requests = validRequests

	// Check limit
	if len(w.requests) < limit {
		w.requests = append(w.requests, now)
		remaining := limit - len(w.requests)
		resetAt := time.UnixMilli(now + windowMs)
		return true, remaining, resetAt, nil
	}

	// Calculate reset time
	var resetAt time.Time
	if len(w.requests) > 0 {
		resetAt = time.UnixMilli(w.requests[0] + windowMs)
	} else {
		resetAt = time.Now().Add(l.config.WindowSize)
	}

	return false, 0, resetAt, nil
}

// TokenBucketLimiter implements a token bucket rate limiter.
type TokenBucketLimiter struct {
	client     *redis.Client
	config     SlidingWindowConfig
	refillRate float64 // tokens per second
}

var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local refill_rate = tonumber(ARGV[3])

local bucket = redis.call('HMGET', key, 'tokens', 'last_update')
local tokens = tonumber(bucket[1]) or capacity
local last_update = tonumber(bucket[2]) or now

-- Calculate tokens to add based on time elapsed
local elapsed = (now - last_update) / 1000
local new_tokens = math.min(capacity, tokens + elapsed * refill_rate)

if new_tokens >= 1 then
    -- Consume a token
    new_tokens = new_tokens - 1
    redis.call('HMSET', key, 'tokens', new_tokens, 'last_update', now)
    redis.call('EXPIRE', key, math.ceil(capacity / refill_rate) + 1)
    return {1, math.floor(new_tokens), now + math.ceil((1 / refill_rate) * 1000)}
else
    -- No tokens available
    local wait_time = math.ceil((1 - new_tokens) / refill_rate * 1000)
    return {0, 0, now + wait_time}
end
`)

// NewTokenBucketLimiter creates a new token bucket rate limiter.
func NewTokenBucketLimiter(client *redis.Client, config SlidingWindowConfig) *TokenBucketLimiter {
	refillRate := float64(config.RequestsPerWindow) / config.WindowSize.Seconds()
	return &TokenBucketLimiter{
		client:     client,
		config:     config,
		refillRate: refillRate,
	}
}

// Allow checks if a request is allowed for the given key.
func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, int, time.Time, error) {
	fullKey := l.config.KeyPrefix + "bucket:" + key
	now := time.Now().UnixMilli()
	capacity := l.config.RequestsPerWindow + l.config.BurstSize

	result, err := tokenBucketScript.Run(ctx, l.client, []string{fullKey},
		now, capacity, l.refillRate).Slice()
	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("token bucket script failed: %w", err)
	}

	allowed := result[0].(int64) == 1
	remaining := int(result[1].(int64))
	resetMs := result[2].(int64)
	resetAt := time.UnixMilli(resetMs)

	return allowed, remaining, resetAt, nil
}

// AdvancedRateLimitMiddleware creates HTTP middleware for advanced rate limiting.
func AdvancedRateLimitMiddleware(limiter AdvancedRateLimiter, keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			allowed, remaining, resetAt, err := limiter.Allow(r.Context(), key)

			if err != nil {
				// On error, allow the request but log
				// In production, you might want to fail closed instead
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

			if !allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(time.Until(resetAt).Seconds()), 10))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]any{
					"error":       "rate limit exceeded",
					"retry_after": time.Until(resetAt).Seconds(),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPRateLimitKeyFunc returns the client IP as the rate limit key.
func IPRateLimitKeyFunc(r *http.Request) string {
	// Check X-Forwarded-For first for proxied requests
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	ip := r.RemoteAddr
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	return ip
}

// UserRateLimitKeyFunc returns the user ID as the rate limit key.
func UserRateLimitKeyFunc(userIDHeader string) func(*http.Request) string {
	return func(r *http.Request) string {
		userID := r.Header.Get(userIDHeader)
		if userID == "" {
			return IPRateLimitKeyFunc(r) // Fall back to IP
		}
		return "user:" + userID
	}
}

// CombinedRateLimitKeyFunc combines IP and path for more granular rate limiting.
func CombinedRateLimitKeyFunc(r *http.Request) string {
	ip := IPRateLimitKeyFunc(r)
	path := r.URL.Path
	return ip + ":" + path
}

// TieredRateLimiter applies different limits based on request characteristics.
type TieredRateLimiter struct {
	tiers          []TierConfig
	defaultLimiter AdvancedRateLimiter
}

// TierConfig configures a rate limit tier.
type TierConfig struct {
	Name    string
	Match   func(*http.Request) bool
	Limiter AdvancedRateLimiter
	KeyFunc func(*http.Request) string
}

// NewTieredRateLimiter creates a tiered rate limiter.
func NewTieredRateLimiter(defaultLimiter AdvancedRateLimiter, tiers ...TierConfig) *TieredRateLimiter {
	return &TieredRateLimiter{
		tiers:          tiers,
		defaultLimiter: defaultLimiter,
	}
}

// Middleware returns the tiered rate limiting middleware.
func (t *TieredRateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Find matching tier
			for _, tier := range t.tiers {
				if tier.Match(r) {
					keyFunc := tier.KeyFunc
					if keyFunc == nil {
						keyFunc = IPRateLimitKeyFunc
					}
					AdvancedRateLimitMiddleware(tier.Limiter, keyFunc)(next).ServeHTTP(w, r)
					return
				}
			}

			// Use default limiter
			AdvancedRateLimitMiddleware(t.defaultLimiter, IPRateLimitKeyFunc)(next).ServeHTTP(w, r)
		})
	}
}
