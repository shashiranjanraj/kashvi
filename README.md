# Kashvi ❤️

> **Fast like Go, Elegant like Laravel — built with love for Kashvi.**

A batteries-included Go web framework inspired by Laravel.
Single binary, zero magic, production-ready out of the box.

---

## Quick Start

```bash
# Install the CLI
make install          # or: go install ./cmd/kashvi

# Bootstrap a new project (already done for you in this repo)
kashvi run            # start the HTTP server
```

---

## Features

| Category | Feature |
|---|---|
| **HTTP** | chi-backed router, groups, named routes, all HTTP methods |
| **Middleware** | Metrics → Recovery → ReqID → Logger → Session → CORS → Rate Limit |
| **Context** | `pkg/ctx` — gin-style `Context` with `BindJSON`, `Param`, `Success`, etc. |
| **Auth** | JWT (access + refresh), bcrypt passwords, RBAC role guards |
| **ORM** | Chainable query builder, pagination, parallel queries, cache bridge |
| **Validation** | 28 rules, zero deps — `required`, `email`, `min`, `max`, `confirmed`, ... |
| **Migrations** | `Up`/`Down`/`Rollback`/`Status`, batch-tracked |
| **Queue** | In-memory + Redis drivers, retries with backoff, persistent failed jobs |
| **Scheduler** | Cron-based task scheduler with overlap guard |
| **Storage** | Local disk + S3-compatible (AWS, MinIO, R2) |
| **Cache** | Redis backend with Laravel-style `Get`/`Set`/`Forget` |
| **WebSocket** | `pkg/ws` — Hub/Client/Broadcast pattern |
| **SSE** | `pkg/sse` — Server-Sent Events with client-disconnect detection |
| **Metrics** | Prometheus — HTTP, DB, queue, cache histograms/counters |
| **Logging** | `log/slog` — JSON in production, text in dev, request-ID tagged |
| **CLI** | `kashvi run`, `kashvi build`, `kashvi route:list`, `kashvi migrate`, `kashvi make:resource`, ... |

---

## Routing

```go
// app/routes/api.go
func RegisterAPI(r *router.Router) {
    api := r.Group("/api", middleware.RateLimit(120, time.Minute))

    api.Get("/users",       "users.index",  ctx.Wrap(ctrl.Index))
    api.Post("/users",      "users.store",  ctx.Wrap(ctrl.Store))
    api.Get("/users/{id}",  "users.show",   ctx.Wrap(ctrl.Show))
    api.Put("/users/{id}",  "users.update", ctx.Wrap(ctrl.Update))
    api.Delete("/users/{id}", "users.destroy", ctx.Wrap(ctrl.Destroy))
}
```

## Context API

```go
func CreatePost(c *ctx.Context) {
    var input struct {
        Title string `json:"title" validate:"required,min=3"`
        Body  string `json:"body"  validate:"required"`
    }
    if !c.BindJSON(&input) { // auto 422 on validation failure
        return
    }
    // ... save to DB ...
    c.Created(post)
}
```

## Validation

```go
type RegisterInput struct {
    Name            string `json:"name"     validate:"required,min=2"`
    Email           string `json:"email"    validate:"required,email"`
    Password        string `json:"password" validate:"required,min=8"`
    PasswordConfirm string `json:"password_confirm" validate:"confirmed=password"`
}
```

## Queue

```go
// Define a job
type WelcomeEmailJob struct{ UserID uint }
func (j WelcomeEmailJob) Handle() error {
    return mailer.Send(j.UserID)
}

// Dispatch immediately
queue.Dispatch(WelcomeEmailJob{UserID: 1})

// Dispatch after a delay (Redis sorted set — survives restarts)
queue.DispatchAfter(WelcomeEmailJob{UserID: 2}, 10*time.Minute)
```

## Storage

```go
// Upload
storage.Put("avatars/user-1.jpg", fileBytes)

// Get public URL
url := storage.URL("avatars/user-1.jpg")

// Use S3 explicitly
storage.Use("s3").Put("backups/db.sql.gz", data)
```

## WebSocket

```go
var ChatHub = ws.NewHub()
func init() { go ChatHub.Run() }

// In your route handler:
func ChatEndpoint(c *ctx.Context) {
    ws.Upgrade(c.W, c.R, ChatHub)
}

// Broadcast to all clients
ChatHub.Broadcast <- []byte(`{"msg":"hello"}`)
```

## Server-Sent Events

```go
func EventStream(c *ctx.Context) {
    stream := sse.New(c.W, c.R)
    for {
        stream.Send("update", map[string]any{"at": time.Now()})
        time.Sleep(time.Second)
        if stream.IsClosed() { break }
    }
}
```

---

## CLI Reference

```bash
kashvi run                    # start HTTP server
kashvi build                  # compile ./kashvi binary
kashvi route:list             # print all registered routes

kashvi migrate                # run pending migrations
kashvi migrate:rollback       # rollback last batch
kashvi migrate:status         # show migration status
kashvi seed                   # run all seeders

kashvi queue:work             # start queue workers
kashvi schedule:run           # start the scheduler

kashvi make:resource Post     # scaffold model + CRUD controller + migration + seeder
kashvi make:model Comment     # model only
kashvi make:controller Auth   # controller only
kashvi make:migration add_tags_to_posts
```

---

## Configuration (`.env`)

```ini
APP_ENV=production
APP_PORT=8080
JWT_SECRET=your-256-bit-secret

DB_DRIVER=postgres
DATABASE_DSN=host=localhost user=postgres dbname=kashvi sslmode=disable

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

STORAGE_DISK=s3
S3_BUCKET=my-bucket
S3_REGION=us-east-1
S3_KEY=AKIA...
S3_SECRET=...
```

---

## Architecture

```
kashvi/
├── app/
│   ├── controllers/     # HTTP handlers
│   ├── models/          # GORM models
│   ├── routes/          # Route registration
│   └── services/        # Business logic
├── cmd/kashvi/          # CLI entry point
├── config/              # Env + config loading
├── database/
│   ├── migrations/      # Migration files
│   └── seeders/         # Seed data
├── internal/
│   ├── kernel/          # HTTP middleware stack
│   └── server/          # Server boot + graceful shutdown
└── pkg/
    ├── auth/            # JWT + bcrypt
    ├── bind/            # JSON decoding + validation
    ├── cache/           # Redis cache
    ├── ctx/             # gin.Context equivalent
    ├── database/        # GORM connection
    ├── logger/          # slog wrapper
    ├── metrics/         # Prometheus
    ├── middleware/       # HTTP middleware
    ├── migration/        # Migration runner
    ├── orm/             # Query builder
    ├── queue/           # Background jobs
    ├── response/        # JSON response helpers
    ├── router/          # chi-backed router
    ├── schedule/        # Task scheduler
    ├── session/         # Session middleware
    ├── sse/             # Server-Sent Events
    ├── storage/         # File storage (local + S3)
    ├── validate/        # Validation engine
    └── ws/              # WebSocket (gorilla)
```

---

Built with ❤️ — **Fast like Go, Elegant like Laravel.**
