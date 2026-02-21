# Installation & Quick Start

## Requirements

- Go 1.21+
- (Optional) Redis — for sessions, cache, queue
- (Optional) PostgreSQL / MySQL / SQLite — default is SQLite

---

## 1. Clone and install

```bash
git clone https://github.com/shashiranjanraj/kashvi
cd kashvi

# Install the CLI tool
make install     # runs: go install ./cmd/kashvi
```

Verify:
```bash
kashvi --help
```

---

## 2. Configure environment

Copy the example env file and edit it:

```bash
cp .env.example .env
```

Minimum required for development:
```ini
APP_ENV=local
APP_PORT=8080
JWT_SECRET=any-long-random-string
DB_DRIVER=sqlite
DATABASE_DSN=kashvi.db
```

---

## 3. Run migrations

```bash
kashvi migrate
```

---

## 4. Start the server

```bash
kashvi run
# → 🚀 Kashvi running on :8080  [env: local]
```

---

## 5. First API call

```bash
# Register a user
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","password":"secret123","password_confirm":"secret123"}'

# Login
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"secret123"}'

# Health check
curl http://localhost:8080/api/health
```

---

## 6. Scaffold your first resource

```bash
kashvi make:resource Post
```

This generates:
- `app/models/post.go`
- `app/controllers/post_controller.go` (full CRUD with `ctx.Context`)
- `app/services/postService_service.go`
- `database/migrations/TIMESTAMP_create_posts_table.go`
- `database/seeders/post_seeder.go`

Then add the routes (the command prints exactly what to paste), migrate, and run:

```bash
kashvi migrate
kashvi run
```

---

## Project Structure

```
kashvi/
├── app/
│   ├── controllers/     # HTTP handlers
│   ├── models/          # GORM models
│   ├── routes/          # api.go — register all routes here
│   └── services/        # Business logic layer
├── cmd/kashvi/          # CLI entrypoint (main + subcommands)
├── config/              # Env + JSON config loader
├── database/
│   ├── migrations/      # Migration files (register in init())
│   └── seeders/         # Seed data + RunAll runner
├── internal/
│   ├── kernel/          # HTTP middleware stack wiring
│   └── server/          # Boot sequence + graceful shutdown
└── pkg/                 # All reusable packages
    ├── auth/  bind/  cache/  ctx/  database/  logger/
    ├── metrics/  middleware/  migration/  orm/
    ├── queue/  reqid/  response/  router/  schedule/
    ├── session/  sse/  storage/  validate/  ws/
```
