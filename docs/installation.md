# Installation & Quick Start

## Requirements

- Go 1.21+
- (Optional) Redis — for sessions, cache, queue
- (Optional) PostgreSQL / MySQL / SQLite — default is SQLite

---

```bash
# 1. Initialize a new Go project
mkdir my-app && cd my-app
go mod init my-app

# 2. Install Kashvi framework & CLI
go get github.com/shashiranjanraj/kashvi
go install github.com/shashiranjanraj/kashvi/cmd/kashvi@latest
```
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

## 3. Scaffold your first resource

```bash
kashvi make:crud Post --authorize
```

This generates:
- `app/models/post.go`
- `app/controllers/post_controller.go` (full CRUD with `ctx.Context`)
- `app/services/postService_service.go`
- `database/migrations/TIMESTAMP_create_posts_table.go`
- `database/seeders/post_seeder.go`

Then add the routes (the command prints exactly what to paste explicitly), migrate the database, and run your new server!

```bash
kashvi migrate
kashvi run
```

---

## Project Structure Overview

Kashvi strictly executes your project structures based on standard MVC formats automatically scaffolded via the CLI tools:
my-app/
├── app/
│   ├── controllers/     # HTTP handlers
│   ├── models/          # GORM models
│   ├── routes/          # api.go — register all routes here
│   └── services/        # Business logic layer
├── cmd/
│   └── server/          # Boot sequence + graceful shutdown
├── config/              # Env + JSON config loaders generated
├── database/
│   ├── migrations/      # Migration files (register in init())
│   └── seeders/         # Seed data + RunAll runner
├── testdata/            # testkit automated testing scenarios
├── .kashvi/
│   └── stubs/           # your custom make:crud CLI boilerplate templates
└── main.go              # project entry
```
