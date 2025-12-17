# Huma v2: The Impatient Engineer's Guide to Production Go APIs

**Assumptions**: You know Go fundamentals (structs, interfaces, error handling), have built at least one HTTP service, and understand REST basics. You want OpenAPI docs that don't drift from code.

---

## 1. Why Huma

- **Auto-generated OpenAPI 3.1** from Go types—your docs are always accurate because they're derived from your code, not maintained separately.
- **Router-agnostic**: works with Chi, Fiber, Echo, Gin, or stdlib `net/http`. Swap routers without rewriting handlers.
- **Validation built-in**: struct tags handle input validation; errors return RFC 7807 problem details automatically.
- **Modern defaults**: content negotiation, CBOR support, streaming responses, and CLI integration out of the box.
- **Lightweight**: ~5k LOC core, minimal dependencies, compiles fast.
- **Type-safe operations**: request/response types are compile-time checked, eliminating a class of runtime bugs.
- **Active maintenance**: strong community, regular releases, good Discord support.

---

## 2. What Problems It Solves (and Doesn't)

| Good Fit | Bad Fit |
|----------|---------|
| APIs needing accurate, auto-generated OpenAPI docs | GraphQL APIs (use gqlgen instead) |
| Teams tired of doc drift with swaggo annotations | Minimal microservices where OpenAPI overhead isn't justified |
| Mixed REST + streaming (SSE) in one service | Heavy real-time websocket apps (Huma supports it, but specialized libs are better) |
| Strict input validation requirements | Simple internal tools where validation verbosity hurts |
| Brownfield: adding OpenAPI to existing Chi/Fiber apps | Projects already committed to another framework's patterns |

**Real-world scenarios**:

1. **Fintech API**: Strict validation, audit trails, generated SDKs from OpenAPI—Huma excels here.
2. **Internal dashboard backend**: Quick CRUD with auto-docs for frontend devs—solid fit.
3. **High-frequency trading gateway**: Huma adds overhead; raw Fiber or fasthttp is better.

---

## 3. Mental Model / Key Concepts

### Core Primitives

```
┌─────────────────────────────────────────────────────────────────┐
│                          Huma API                               │
│  ┌──────────┐    ┌─────────────┐    ┌──────────────────────┐   │
│  │  Router  │───▶│  Operation  │───▶│  Handler Function    │   │
│  │(Chi/Fiber)│   │  (metadata) │    │  func(ctx, *Input)   │   │
│  └──────────┘    └─────────────┘    │       (*Output, err) │   │
│                         │           └──────────────────────┘   │
│                         ▼                                       │
│               ┌─────────────────┐                              │
│               │  OpenAPI Spec   │◀── auto-generated            │
│               │  (served at     │                              │
│               │   /openapi.json)│                              │
│               └─────────────────┘                              │
└─────────────────────────────────────────────────────────────────┘

Request Flow:
  HTTP Request
       │
       ▼
  ┌─────────────┐     ┌──────────────┐     ┌────────────┐
  │   Router    │────▶│  Middleware  │────▶│  Resolver  │
  │  (path     │     │  (auth, log) │     │  (parse +  │
  │   match)    │     └──────────────┘     │  validate) │
  └─────────────┘                          └─────┬──────┘
                                                 │
                         ┌───────────────────────┘
                         ▼
  ┌────────────┐    ┌─────────────┐    ┌──────────────┐
  │  Handler   │───▶│  Transform  │───▶│  HTTP        │
  │  (business │    │  (output    │    │  Response    │
  │   logic)   │    │   encode)   │    │              │
  └────────────┘    └─────────────┘    └──────────────┘
```

### How Primitives Interact

**API**: The central registry that holds your router adapter and all operations. Created once at startup.

**Operation**: Metadata about an endpoint—path, method, tags, summary, request/response types. Registered via `huma.Register()`.

**Input struct**: Defines everything coming in—path params, query params, headers, body. Tags control validation and OpenAPI docs.

**Output struct**: Defines response—status code, headers, body. Multiple outputs for different status codes.

**Resolver interface**: Optional; implement `Resolve(ctx huma.Context) []error` on inputs for custom validation.

### Essential Glossary

| Term | Definition |
|------|------------|
| **Operation** | A single API endpoint with its metadata and handler |
| **Input/Output** | Structs defining request/response shape with validation tags |
| **Resolver** | Interface for complex validation that runs after parsing |
| **Adapter** | Bridge between Huma and your router (Chi, Fiber, etc.) |
| **Transformer** | Modifies output before serialization (rarely needed) |
| **RFC 7807** | Standard error format Huma uses (`application/problem+json`) |

---

## 4. The Survival Kit

### Prioritized Checklist

**Day 0** (2 hours):
- [ ] `go get github.com/danielgtaylor/huma/v2`
- [ ] Create hello-world endpoint with one input/output struct
- [ ] Visit `/docs` to see generated Swagger UI
- [ ] Add one validation tag, trigger a validation error, observe RFC 7807 response

**Week 1**:
- [ ] Build 3-5 CRUD endpoints for a real resource
- [ ] Implement custom `Resolver` for cross-field validation
- [ ] Add authentication middleware
- [ ] Write tests using `humatest` adapter
- [ ] Export OpenAPI spec, validate with spectral

**Week 2**:
- [ ] Add SSE endpoint for real-time updates
- [ ] Implement error handling with custom error types
- [ ] Set up CI pipeline generating SDK from OpenAPI
- [ ] Profile and optimize hot paths

### The 80/20 Features

1. **Struct tags for validation**: `required`, `minLength`, `maximum`, `pattern`—cover 90% of cases
2. **Path/query/header tags**: `path:"id"`, `query:"limit"`, `header:"X-Request-ID"`
3. **`huma.Register()`**: The only function you call repeatedly
4. **Built-in `/docs`**: Swagger UI served automatically
5. **`humatest.Wrap()`**: Test handlers without HTTP overhead

### Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| Forgetting `json` tags on output bodies | Always add `json:"fieldName"` or fields won't serialize |
| Putting body fields at Input struct root | Body must be in a nested struct with `Body` field name |
| Ignoring pointer vs value for optional fields | Use `*string` for optional, `string` for required |
| Not returning the Output struct | Handler must return `(*Output, error)`, not write directly |
| Middleware not seeing Huma context | Use adapter's native middleware, not Huma's context |

### Debugging Tips

- **Enable debug logging**: `huma.NewAPI(config, adapter)` with `config.Debug = true`
- **Check `/openapi.json`**: If your endpoint isn't there, registration failed silently
- **Validation errors**: Response includes `"location"` field showing exactly which input failed
- **Use `huma.Error()`**: Wraps errors with status codes correctly

### Performance & Security Gotchas

- **Body size limits**: Set `config.MaxBodyBytes` (default is 1MB)
- **Timeouts**: Huma doesn't set these; configure at router/server level
- **CORS**: Not built-in; add middleware (Chi's `cors` or similar)
- **Auth**: Middleware responsibility; Huma validates after auth runs

---

## 5. Progressive Complexity Examples

### Example 1: Hello, Core Primitive

**Problem**: Create a greeting endpoint that takes a name and returns a personalized message.

```go
package main

import (
    "context"
    "fmt"
    "net/http"

    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/adapters/humachi"
    "github.com/go-chi/chi/v5"
)

type GreetInput struct {
    Name string `path:"name" minLength:"1" maxLength:"100" example:"world"`
}

type GreetOutput struct {
    Body struct {
        Message string `json:"message" example:"Hello, world!"`
    }
}

func main() {
    r := chi.NewMux()
    api := humachi.New(r, huma.DefaultConfig("Greeting API", "1.0.0"))

    huma.Register(api, huma.Operation{
        OperationID: "greet",
        Method:      http.MethodGet,
        Path:        "/greet/{name}",
        Summary:     "Greet someone",
        Tags:        []string{"Greetings"},
    }, func(ctx context.Context, input *GreetInput) (*GreetOutput, error) {
        resp := &GreetOutput{}
        resp.Body.Message = fmt.Sprintf("Hello, %s!", input.Name)
        return resp, nil
    })

    http.ListenAndServe(":8080", r)
}
```

**How it works**: The `GreetInput` struct defines a path parameter via the `path:"name"` tag. Huma parses the URL, validates against `minLength`/`maxLength`, and passes a populated struct to your handler. The output struct's nested `Body` becomes the JSON response. Visit `localhost:8080/docs` to see interactive documentation.

**When to use**: Every endpoint starts here. This pattern scales to complex inputs.

**Upgrade idea**: Add `query:"uppercase"` boolean to optionally transform the response.

---

### Example 2: Typical CRUD Workflow

**Problem**: Implement create and get endpoints for a user resource with proper validation.

```go
type User struct {
    ID        string    `json:"id" example:"usr_123"`
    Email     string    `json:"email" example:"user@example.com"`
    Name      string    `json:"name" example:"Jane Doe"`
    CreatedAt time.Time `json:"created_at"`
}

// CREATE
type CreateUserInput struct {
    Body struct {
        Email string `json:"email" required:"true" format:"email" maxLength:"254"`
        Name  string `json:"name" required:"true" minLength:"1" maxLength:"100"`
    }
}

type CreateUserOutput struct {
    Body User
}

func (o *CreateUserOutput) SetStatusCode() int { return http.StatusCreated }

// GET
type GetUserInput struct {
    ID string `path:"id" pattern:"^usr_[a-zA-Z0-9]{8,}$"`
}

type GetUserOutput struct {
    Body User
}

// Registration
huma.Register(api, huma.Operation{
    OperationID:   "createUser",
    Method:        http.MethodPost,
    Path:          "/users",
    Summary:       "Create a user",
    Tags:          []string{"Users"},
    DefaultStatus: http.StatusCreated,
}, func(ctx context.Context, input *CreateUserInput) (*CreateUserOutput, error) {
    user := User{
        ID:        "usr_" + generateID(),
        Email:     input.Body.Email,
        Name:      input.Body.Name,
        CreatedAt: time.Now(),
    }
    // save to database...
    return &CreateUserOutput{Body: user}, nil
})

huma.Register(api, huma.Operation{
    OperationID: "getUser",
    Method:      http.MethodGet,
    Path:        "/users/{id}",
    Summary:     "Get a user by ID",
    Tags:        []string{"Users"},
}, func(ctx context.Context, input *GetUserInput) (*GetUserOutput, error) {
    user, err := db.FindUser(input.ID)
    if err != nil {
        return nil, huma.Error404NotFound("user not found")
    }
    return &GetUserOutput{Body: user}, nil
})
```

**How it works**: The `Body` struct in `CreateUserInput` receives JSON. The `format:"email"` tag validates email format automatically. For `GetUserInput`, the `pattern` tag ensures IDs match your format before hitting the database. Returning `huma.Error404NotFound()` generates proper RFC 7807 error responses.

**When to use**: Any resource-oriented API. This is your bread and butter.

**Upgrade idea**: Add `Resolve()` method to check email uniqueness against database before handler runs.

---

### Example 3: Production Pattern with Custom Validation

**Problem**: Create an order endpoint with cross-field validation and custom error handling.

```go
type CreateOrderInput struct {
    Body struct {
        Items []OrderItem `json:"items" required:"true" minItems:"1" maxItems:"100"`
        PromoCode *string `json:"promo_code,omitempty" maxLength:"20"`
    }
}

type OrderItem struct {
    ProductID string `json:"product_id" required:"true"`
    Quantity  int    `json:"quantity" required:"true" minimum:"1" maximum:"99"`
}

// Custom validation via Resolver
func (i *CreateOrderInput) Resolve(ctx huma.Context) []error {
    var errs []error
    
    seen := make(map[string]bool)
    for idx, item := range i.Body.Items {
        if seen[item.ProductID] {
            errs = append(errs, &huma.ErrorDetail{
                Location: fmt.Sprintf("body.items[%d].product_id", idx),
                Message:  "duplicate product ID",
                Value:    item.ProductID,
            })
        }
        seen[item.ProductID] = true
    }
    
    return errs
}

// Custom error type for business logic errors
type InsufficientStockError struct {
    ProductID string
    Available int
    Requested int
}

func (e *InsufficientStockError) Error() string {
    return fmt.Sprintf("insufficient stock for %s", e.ProductID)
}

func (e *InsufficientStockError) GetStatus() int {
    return http.StatusConflict
}

// Handler
func createOrder(ctx context.Context, input *CreateOrderInput) (*CreateOrderOutput, error) {
    for _, item := range input.Body.Items {
        stock, _ := inventory.Check(item.ProductID)
        if stock < item.Quantity {
            return nil, huma.Error409Conflict(
                "insufficient stock",
                &huma.ErrorDetail{
                    Location: "body.items",
                    Message:  fmt.Sprintf("only %d available", stock),
                    Value:    item.ProductID,
                },
            )
        }
    }
    // create order...
    return &CreateOrderOutput{Body: order}, nil
}
```

**How it works**: The `Resolve()` method runs after parsing but before your handler, catching business rule violations early. Custom errors implementing `GetStatus()` integrate with Huma's error system. The `ErrorDetail` struct provides precise error locations for API consumers.

**When to use**: Any endpoint with validation rules spanning multiple fields or requiring database lookups.

**Upgrade idea**: Move stock checking into `Resolve()` with a database connection passed via context.

---

### Example 4: Server-Sent Events (SSE)

**Problem**: Stream real-time updates to clients for a long-running process.

```go
type StreamInput struct {
    JobID string `path:"job_id"`
}

type StreamOutput struct {
    Body io.Reader
}

func (o *StreamOutput) ContentType() string {
    return "text/event-stream"
}

huma.Register(api, huma.Operation{
    OperationID: "streamJobProgress",
    Method:      http.MethodGet,
    Path:        "/jobs/{job_id}/stream",
    Summary:     "Stream job progress",
}, func(ctx context.Context, input *StreamInput) (*StreamOutput, error) {
    pr, pw := io.Pipe()
    
    go func() {
        defer pw.Close()
        
        for progress := range jobProgress(input.JobID) {
            fmt.Fprintf(pw, "event: progress\ndata: %s\n\n", progress.JSON())
            
            if ctx.Err() != nil {
                return // client disconnected
            }
        }
        fmt.Fprintf(pw, "event: complete\ndata: {\"status\":\"done\"}\n\n")
    }()
    
    return &StreamOutput{Body: pr}, nil
})
```

```
SSE Flow:
  Client                    Server
    │                          │
    │──GET /jobs/123/stream───▶│
    │                          │
    │◀──event: progress────────│
    │   data: {"pct": 10}      │
    │                          │
    │◀──event: progress────────│
    │   data: {"pct": 50}      │
    │                          │
    │◀──event: complete────────│
    │   data: {"status":"done"}│
    │                          │
    └──────connection closes───┘
```

**How it works**: Returning `io.Reader` as the body enables streaming. The `ContentType()` method sets the correct MIME type. The goroutine writes SSE-formatted messages, and context cancellation handles client disconnects gracefully.

**When to use**: Progress updates, live feeds, any scenario where polling is wasteful.

**Upgrade idea**: Add heartbeat events every 30s to detect dead connections through proxies.

---

## 6. Cheat Sheet

### Commands & Patterns

```go
// Create API with router
api := humachi.New(router, huma.DefaultConfig("API", "1.0.0"))

// Register endpoint
huma.Register(api, huma.Operation{...}, handlerFunc)

// Path parameter
ID string `path:"id"`

// Query parameter (optional)
Limit *int `query:"limit" minimum:"1" maximum:"100"`

// Required query parameter
Page int `query:"page" required:"true"`

// Header
Token string `header:"Authorization"`

// Body (always nested)
Body struct { Field string `json:"field"` }

// Validation tags
`required:"true" minLength:"1" maxLength:"100" pattern:"^[a-z]+$"`
`minimum:"0" maximum:"100" multipleOf:"5"`
`format:"email"` `format:"uri"` `format:"uuid"` `format:"date-time"`
`enum:"pending,active,done"`
`minItems:"1" maxItems:"10"` // for slices

// Return errors
return nil, huma.Error404NotFound("not found")
return nil, huma.Error400BadRequest("bad input", &huma.ErrorDetail{...})

// Test without HTTP
adapter := humatest.NewAdapter(t, api)
resp := adapter.Get("/users/123")

// Get OpenAPI spec programmatically
spec := api.OpenAPI()
```

### If You Only Remember 5 Things

1. **Input struct = request shape** — path/query/header tags for params, nested `Body` for JSON
2. **Output struct = response shape** — nested `Body` becomes JSON, implement `SetStatusCode()` for non-200
3. **`huma.Register(api, Operation{}, handler)`** — this is your main API
4. **Validation via struct tags** — `required`, `minLength`, `pattern`, `format`
5. **`huma.Error4xxYyy()`** — proper RFC 7807 errors with no effort

---

## 7. Related Technologies

| Category | Technology | Choose When... |
|----------|------------|----------------|
| **Alternatives** | Echo + swaggo | You prefer annotation-based OpenAPI |
| | Fiber + swaggo | Raw speed matters more than stdlib compatibility |
| | go-swagger | You want code generation from OpenAPI (design-first) |
| | Ogen | Full client+server generation from OpenAPI spec |
| **Complements** | Chi | Lightweight stdlib-compatible router (default choice) |
| | sqlc | Type-safe SQL for your data layer |
| | Scalar | Modern API docs UI (alternative to Swagger UI) |
| | oapi-codegen | Generate client SDKs from your Huma-generated spec |
| **Prereqs** | Go stdlib `net/http` | Understand before adding abstractions |
| | OpenAPI 3.x spec | Know what Huma generates for you |
| **Next steps** | go-sse | Dedicated SSE library for complex streaming |
| | jrpc2 | Add JSON-RPC alongside REST in same service |
| | ko / goreleaser | Container builds and releases |

---

## 8. Resources

| Resource | URL | Description |
|----------|-----|-------------|
| Huma Documentation | https://huma.rocks | Official docs, examples, and API reference |
| Huma GitHub | https://github.com/danielgtaylor/huma | Source code, issues, discussions |
| OpenAPI 3.1 Spec | https://spec.openapis.org/oas/v3.1.0 | Understand what Huma generates |
| Chi Router | https://github.com/go-chi/chi | Recommended router for Huma |
| Scalar API Docs | https://github.com/scalar/scalar | Modern alternative to Swagger UI |
| RFC 7807 Problem Details | https://www.rfc-editor.org/rfc/rfc7807 | Error format Huma uses |
| Go Validator | https://github.com/go-playground/validator | Underlying validation library concepts |
| sqlc | https://sqlc.dev | Recommended database layer |
| jrpc2 | https://github.com/creachadair/jrpc2 | JSON-RPC companion library |

---

**You're now equipped to build production Go APIs with auto-generated, always-accurate OpenAPI docs. Start with Example 1, get it running, then layer complexity as needed.**