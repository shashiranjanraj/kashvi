# Installation & Quick Start

This guide sets up a new Kashvi project from zero to a running server.

## Requirements

- Go `1.25+` (matches this framework's `go.mod`)
- Optional: Redis (session, queue, cache features)
- Optional: Postgres/MySQL/SQL Server (SQLite works by default)

## Step 1: Install the CLI

Install the global `kashvi` command once:

```bash
go install github.com/shashiranjanraj/kashvi/cmd/kashvi@latest
kashvi --help
```

If you are developing the framework repository itself, you can also run:

```bash
make install
```

## Step 2: Create a project

```bash
mkdir my-app
cd my-app
go mod init my-app
go get github.com/shashiranjanraj/kashvi
```

Create `main.go`:

```go
package main

import (
	"github.com/shashiranjanraj/kashvi/pkg/app"
	appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

func main() {
	app.New().
		Routes(func(r *router.Router) {
			r.Get("/health", "health", appctx.Wrap(func(c *appctx.Context) {
				c.Success(map[string]any{"ok": true})
			}))
		}).
		Run()
}
```

## Step 3: Add environment config

Create `.env`:

```ini
APP_ENV=local
APP_PORT=8080
JWT_SECRET=replace-with-long-random-secret

DB_DRIVER=sqlite
DATABASE_DSN=kashvi.db

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
```

Notes:
- `DB_DRIVER` supports: `sqlite`, `postgres`, `mysql`, `sqlserver`.
- Kashvi reads both `config/app.json` and `.env` (then applies defaults).

## Step 4: Run the app

From the project directory:

```bash
kashvi serve
```

The CLI delegates to your project entrypoint (`go run . serve`), so your own routes/migrations/seeders are used.

Quick checks:

```bash
curl http://localhost:8080/health
kashvi route:list
```

## Step 5: Add your first resource

```bash
kashvi make:crud Post
```

This generates model/controller/service/migration/seeder/test-scenario files. Then:

```bash
kashvi migrate
kashvi serve
```

For full resource wiring and CRUD API flow, continue to [CRUD Walkthrough](./crud.md).
