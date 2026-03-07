# Middleware

Kashvi provides built-in HTTP middleware to handle common tasks like CORS, rate limiting, logging, and recovery.

## Using Middleware in Routes

Middlewares in Kashvi wrap standard standard Go `http.Handler` interfaces. You can apply middlewares globally or to specific route groups.

### Global Middleware

```go
app.New().
    WithMiddleware(
        middleware.Logger,
        middleware.Recovery,
    ).
    Routes(...)
```

### Group Middleware
Apply middleware only to specific endpoints (e.g., locking down `/api` with rate limiting and CORS).

```go
func RegisterAPI(r *router.Router) {
    api := r.Group("/api").WithMiddleware(
        middleware.CORS(middleware.DefaultCORSOptions()),
        middleware.RateLimit(100, time.Minute), 
    )

    api.Get("/data", "api.data", appctx.Wrap(func(c *appctx.Context) {
        c.Success("Hello with CORS!")
    }))
}
```

## Built-in Middlewares

All middlewares are available under `pkg/middleware`.

### CORS
Enables Cross-Origin Resource Sharing. `DefaultCORSOptions()` returns permissive settings suitable for local development.

```go
middleware.CORS(middleware.CORSOptions{
    AllowedOrigins: []string{"https://myfrontend.com"},
    AllowedMethods: []string{"GET", "POST", "OPTIONS"},
    AllowedHeaders: []string{"Authorization", "Content-Type"},
})
```

### Rate limiting
In-memory IP-based rate limiting. Rejects requests with HTTP 429 Too Many Requests once the limit is hit.
```go
// 100 requests per minute
middleware.RateLimit(100, time.Minute)
```

### Logging
Structured request logging to standard out or MongoDB (if configured). Automatically tracks request timing and status codes.
```go
middleware.Logger
```

### Panic Recovery
Recovers from panics within a handler and returns a clean HTTP 500 error instead of crashing the server.
```go
middleware.Recovery
```

### Authentication
JWT token validation. Rejects requests lacking a valid Bearer token.
```go
middleware.AuthMiddleware
```

## Creating Custom Middleware

Because Kashvi relies on standard `net/http` handlers, custom middleware is simply a function that returns an `http.Handler`:

```go
func MyCustomMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Pre-request logic (e.g., check headers)
        if r.Header.Get("X-Custom-Header") == "" {
            http.Error(w, "Missing Header", http.StatusBadRequest)
            return
        }

        next.ServeHTTP(w, r) // pass to the next handler
        
        // Post-request logic
    })
}
```
