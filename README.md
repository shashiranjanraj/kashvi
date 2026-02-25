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
| **gRPC** | Standalone gRPC server — recovery/logging/Prometheus interceptors, health-check, reflection |
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
| **Metrics** | Prometheus — HTTP, gRPC, DB, queue, cache histograms/counters |
| **Logging** | `log/slog` — JSON in prod, text in dev, request-ID tagged, **MongoDB async log sink** |
| **Worker Pool** | `pkg/workerpool` — bounded goroutine pool with backpressure (`ErrPoolFull`) |
| **TestKit** | `pkg/testkit` — JSON-scenario-driven REST API tests with testify mocks |
| **CLI** | `kashvi run`, `kashvi grpc:serve`, `kashvi route:list`, `kashvi migrate`, `kashvi make:resource`, ... |

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
kashvi run                    # start HTTP + gRPC servers
kashvi grpc:serve             # start gRPC server only
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

## gRPC

```bash
# Server starts automatically alongside HTTP on GRPC_PORT (default 9090)
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
# → { "status": "SERVING" }
```

See [docs/grpc.md](docs/grpc.md) for registering services and custom interceptors.

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

# gRPC
GRPC_PORT=9090

# MongoDB log storage (leave blank to disable)
MONGO_URI=mongodb://localhost:27017
MONGO_LOG_DB=kashvi_logs
MONGO_LOG_COLLECTION=app_logs

# Performance
WORKER_POOL_SIZE=50
RATE_LIMIT_MAX=2000

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
├── docs/                # Documentation
├── internal/
│   ├── kernel/          # HTTP middleware stack
│   └── server/          # HTTP + gRPC boot + graceful shutdown
└── pkg/
    ├── auth/            # JWT + bcrypt
    ├── bind/            # JSON decoding + validation
    ├── cache/           # Redis cache
    ├── ctx/             # gin.Context equivalent
    ├── database/        # GORM connection
    ├── grpc/            # gRPC server + interceptors + health service
    ├── logger/          # slog wrapper + MongoDB async handler
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
    ├── testkit/         # JSON-scenario-driven API test framework
    ├── validate/        # Validation engine
    ├── workerpool/      # Bounded goroutine pool
    └── ws/              # WebSocket (gorilla)
```

---

## Documentation

| Topic | File |
|-------|------|
| Routing | [docs/routing.md](docs/routing.md) |
| Context API | [docs/context.md](docs/context.md) |
| Validation | [docs/validation.md](docs/validation.md) |
| ORM | [docs/orm.md](docs/orm.md) |
| Auth (JWT + RBAC) | [docs/auth.md](docs/auth.md) |
| Queue & Jobs | [docs/queue.md](docs/queue.md) |
| Storage | [docs/storage.md](docs/storage.md) |
| WebSocket & SSE | [docs/websocket.md](docs/websocket.md) |
| Migrations | [docs/migrations.md](docs/migrations.md) |
| CLI Reference | [docs/cli.md](docs/cli.md) |
| Configuration | [docs/configuration.md](docs/configuration.md) |
| **gRPC Server** | [docs/grpc.md](docs/grpc.md) |
| **MongoDB Logging** | [docs/logging.md](docs/logging.md) |
| **Worker Pool** | [docs/workerpool.md](docs/workerpool.md) |
| **TestKit** | [docs/testkit.md](docs/testkit.md) |

---

Built with ❤️ — **Fast like Go, Elegant like Laravel.**
