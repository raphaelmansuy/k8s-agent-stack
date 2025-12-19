# Middleware Patterns

Production-ready middleware patterns for AgentStack API Gateway.

## Authentication Middleware

```go
// internal/api/middleware/auth.go
package middleware

import (
    "strings"
    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v5"
)

type AuthService interface {
    ValidateAPIKey(key string) (*APIKeyInfo, error)
    ValidateJWT(token string) (*Claims, error)
}

type APIKeyInfo struct {
    KeyID     string
    ProjectID string
    Scopes    []string
}

type Claims struct {
    UserID    string   `json:"sub"`
    Email     string   `json:"email"`
    ProjectID string   `json:"project_id"`
    Scopes    []string `json:"scopes"`
    jwt.RegisteredClaims
}

func Auth(authSvc AuthService) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Try API Key first
        apiKey := c.Get("X-API-Key")
        if apiKey != "" {
            info, err := authSvc.ValidateAPIKey(apiKey)
            if err != nil {
                return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                    "type":   "https://api.agentstack.io/errors/unauthorized",
                    "title":  "Unauthorized",
                    "status": 401,
                    "detail": "Invalid API key",
                })
            }
            c.Locals("authType", "api_key")
            c.Locals("projectID", info.ProjectID)
            c.Locals("scopes", info.Scopes)
            return c.Next()
        }

        // Try Bearer token
        authHeader := c.Get("Authorization")
        if strings.HasPrefix(authHeader, "Bearer ") {
            token := strings.TrimPrefix(authHeader, "Bearer ")
            claims, err := authSvc.ValidateJWT(token)
            if err != nil {
                return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                    "type":   "https://api.agentstack.io/errors/unauthorized",
                    "title":  "Unauthorized",
                    "status": 401,
                    "detail": "Invalid or expired token",
                })
            }
            c.Locals("authType", "jwt")
            c.Locals("userID", claims.UserID)
            c.Locals("projectID", claims.ProjectID)
            c.Locals("scopes", claims.Scopes)
            return c.Next()
        }

        // Check X-Project-ID header (requires project from another auth)
        projectID := c.Get("X-Project-ID")
        if projectID != "" {
            c.Locals("projectID", projectID)
        }

        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "type":   "https://api.agentstack.io/errors/unauthorized",
            "title":  "Unauthorized",
            "status": 401,
            "detail": "Missing authentication",
        })
    }
}
```

## Request Logging Middleware

```go
// internal/api/middleware/logging.go
package middleware

import (
    "time"
    "github.com/gofiber/fiber/v2"
    "go.uber.org/zap"
)

func Logger(logger *zap.Logger) fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        
        err := c.Next()
        
        duration := time.Since(start)
        
        fields := []zap.Field{
            zap.String("method", c.Method()),
            zap.String("path", c.Path()),
            zap.Int("status", c.Response().StatusCode()),
            zap.Duration("duration", duration),
            zap.String("request_id", c.Locals("requestID").(string)),
            zap.String("project_id", c.Locals("projectID").(string)),
            zap.String("remote_ip", c.IP()),
        }

        if err != nil {
            fields = append(fields, zap.Error(err))
            logger.Error("Request failed", fields...)
        } else if c.Response().StatusCode() >= 400 {
            logger.Warn("Request error", fields...)
        } else {
            logger.Info("Request completed", fields...)
        }

        return err
    }
}
```

## Rate Limiting Middleware

```go
// internal/api/middleware/ratelimit.go
package middleware

import (
    "context"
    "fmt"
    "strconv"
    "time"
    "github.com/gofiber/fiber/v2"
    "github.com/redis/go-redis/v9"
)

type RateLimiter struct {
    redis  *redis.Client
    limits map[string]RateLimit
}

type RateLimit struct {
    Requests int
    Window   time.Duration
    Burst    int
}

var DefaultLimits = map[string]RateLimit{
    "free":       {Requests: 60, Window: time.Minute, Burst: 100},
    "pro":        {Requests: 600, Window: time.Minute, Burst: 1000},
    "enterprise": {Requests: 6000, Window: time.Minute, Burst: 10000},
}

func RateLimit(limiter *RateLimiter) fiber.Handler {
    return func(c *fiber.Ctx) error {
        projectID := c.Locals("projectID").(string)
        plan := c.Locals("plan").(string) // Set by auth middleware
        
        limit := limiter.limits[plan]
        key := fmt.Sprintf("ratelimit:%s:%d", projectID, time.Now().Unix()/60)

        count, err := limiter.redis.Incr(context.Background(), key).Result()
        if err != nil {
            // Fail open - allow request if Redis is down
            return c.Next()
        }

        if count == 1 {
            limiter.redis.Expire(context.Background(), key, limit.Window)
        }

        // Set rate limit headers
        c.Set("X-RateLimit-Limit", strconv.Itoa(limit.Requests))
        c.Set("X-RateLimit-Remaining", strconv.Itoa(max(0, limit.Requests-int(count))))
        c.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(limit.Window).Unix(), 10))

        if int(count) > limit.Requests {
            c.Set("Retry-After", "60")
            return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
                "type":   "https://api.agentstack.io/errors/rate-limit",
                "title":  "Rate Limit Exceeded",
                "status": 429,
                "detail": fmt.Sprintf("Rate limit of %d requests per minute exceeded", limit.Requests),
            })
        }

        return c.Next()
    }
}
```

## OpenTelemetry Tracing Middleware

```go
// internal/api/middleware/tracing.go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("agentstack-api")

func Tracing() fiber.Handler {
    return func(c *fiber.Ctx) error {
        ctx, span := tracer.Start(c.Context(), c.Method()+" "+c.Path(),
            trace.WithAttributes(
                attribute.String("http.method", c.Method()),
                attribute.String("http.url", c.OriginalURL()),
                attribute.String("http.route", c.Route().Path),
                attribute.String("project.id", c.Locals("projectID").(string)),
            ),
        )
        defer span.End()

        // Store trace ID in locals for error responses
        c.Locals("traceID", span.SpanContext().TraceID().String())
        
        // Replace context
        c.SetUserContext(ctx)

        err := c.Next()

        // Record response status
        span.SetAttributes(
            attribute.Int("http.status_code", c.Response().StatusCode()),
        )

        if err != nil {
            span.RecordError(err)
        }

        return err
    }
}
```

## Request ID Middleware

```go
// internal/api/middleware/requestid.go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
)

func RequestID() fiber.Handler {
    return func(c *fiber.Ctx) error {
        requestID := c.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        c.Locals("requestID", requestID)
        c.Set("X-Request-ID", requestID)
        
        return c.Next()
    }
}
```

## Idempotency Middleware

```go
// internal/api/middleware/idempotency.go
package middleware

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "time"
    "github.com/gofiber/fiber/v2"
    "github.com/redis/go-redis/v9"
)

type IdempotencyStore struct {
    redis *redis.Client
    ttl   time.Duration
}

func Idempotency(store *IdempotencyStore) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Only apply to mutating methods
        if c.Method() != "POST" && c.Method() != "PUT" && c.Method() != "PATCH" {
            return c.Next()
        }

        key := c.Get("Idempotency-Key")
        if key == "" {
            return c.Next()
        }

        projectID := c.Locals("projectID").(string)
        cacheKey := fmt.Sprintf("idempotency:%s:%s", projectID, key)

        // Check for existing response
        cached, err := store.redis.Get(context.Background(), cacheKey).Bytes()
        if err == nil {
            var resp CachedResponse
            json.Unmarshal(cached, &resp)
            c.Set("X-Idempotent-Replayed", "true")
            return c.Status(resp.Status).Send(resp.Body)
        }

        // Execute request
        err = c.Next()

        // Cache response
        resp := CachedResponse{
            Status: c.Response().StatusCode(),
            Body:   c.Response().Body(),
        }
        data, _ := json.Marshal(resp)
        store.redis.Set(context.Background(), cacheKey, data, store.ttl)

        return err
    }
}

type CachedResponse struct {
    Status int    `json:"status"`
    Body   []byte `json:"body"`
}
```

## Combining Middleware

```go
// internal/api/routes/router.go
func SetupMiddleware(app *fiber.App, deps *Dependencies) {
    // Global middleware (applied to all routes)
    app.Use(recover.New())
    app.Use(RequestID())
    
    // API middleware stack
    api := app.Group("/v1")
    api.Use(Tracing())
    api.Use(Logger(deps.Logger))
    api.Use(Auth(deps.AuthService))
    api.Use(RateLimit(deps.RateLimiter))
    api.Use(Idempotency(deps.IdempotencyStore))
}
```
