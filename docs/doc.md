# Kashvi Documentation

This is a consolidated documentation file for the Kashvi framework.

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


---

# Building a User CRUD API in Kashvi (5 Minutes)

Welcome to Kashvi! If you're a fresher or transitioning from PHP/Laravel, you'll feel right at home. We're going to build a fully functional `User` API in just a few minutes. 

No advanced features (like WebSockets or gRPC) here—just standard, clean RESTful architecture.

---

## 1. Create the Project
First, scaffold a fresh project. This creates a ready-to-use folder structure for you.

```bash
kashvi new my-api
cd my-api
```

*(This command generates your `main.go`, `app/` folder for logic, `database/` for schemas, and a `.env` file pre-configured for SQLite so you don't need any complex database setup yet).*

---

## 2. Generate the Resource
We need a Model (to represent the user), a Controller (to handle HTTP requests), a Migration (to create the database table), and a Seeder (to add dummy data).

Instead of creating these manually, Kashvi's CLI does it in one command:

```bash
kashvi make:resource User
```

**What this did:**
* `app/models/user.go`: Your data structure.
* `app/controllers/user_controller.go`: Where your API logic lives.
* `database/migrations/xxxx_create_users_table.go`: Instructions for creating the database table.
* `database/seeders/user_seeder.go`: A place to create fake users for testing.

---

## 3. Define the Database Table
Let's tell the database what a "User" looks like. Open the newly generated migration file inside the `database/migrations/` folder.

Add the `name` and `email` columns:

```go
// database/migrations/xxxx_create_users_table.go

func (m *Migration) Up() {
    table := m.CreateTable("users")
    table.String("name").NotNull()
    table.String("email").Unique().NotNull()
}
```

Now, run the migration to actually create the table in your SQLite database:
```bash
kashvi migrate
```

---

## 4. Write the Controller Logic
Open `app/controllers/user_controller.go`. We'll write the logic for creating a new user (the `Store` method).

Kashvi has built-in JSON validation. If the user doesn't send a name, we automatically reject the request!

```go
// app/controllers/user_controller.go
package controllers

import (
    "github.com/shashiranjanraj/kashvi/pkg/ctx"
    "my-api/app/models"
    "my-api/database" // Assuming you export your db connection here
)

func (c *UserController) Store(ctx *ctx.Context) {
    // 1. Define what JSON we expect and add validation rules!
    var input struct {
        Name  string `json:"name" validate:"required,min=2"`
        Email string `json:"email" validate:"required,email"`
    }

    // 2. Bind and Validate. If it fails, Kashvi automatically sends a 422 Error to the client.
    if !ctx.BindJSON(&input) { 
        return 
    }

    // 3. Save to database
    user := models.User{Name: input.Name, Email: input.Email}
    database.DB.Create(&user)
    
    // 4. Send success response (201 Created)
    ctx.Created(user)
}
```

---

## 5. Register the Route
We need to map a URL (like `POST /api/users`) to the controller we just wrote. 

Open `app/routes/api.go` and add this inside the `RegisterRoutes` function:

```go
// app/routes/api.go
import "my-api/app/controllers"

func RegisterAPI(r *router.Router) {
    api := r.Group("/api")
    
    // Initialize our controller
    userCtrl := controllers.NewUserController()

    // Map the POST request to the Store function
    api.Post("/users", "users.store", ctx.Wrap(userCtrl.Store))
}
```

---

## 6. Run the Server
You're done coding! Let's start the server.

```bash
kashvi serve
```
*You should see a message saying your server is running on port 8080.*

---

## 7. Test It Out
Open your terminal and run this `curl` command (or use Postman) to create a user:

```bash
curl -X POST http://localhost:8080/api/users \
     -H "Content-Type: application/json" \
     -d '{"name": "Rahul", "email": "rahul@example.com"}'
```

**Success Response:**
```json
{
  "name": "Rahul",
  "email": "rahul@example.com"
}
```

**What if we forget the email? (Validation Test)**
```bash
curl -X POST http://localhost:8080/api/users \
     -H "Content-Type: application/json" \
     -d '{"name": "Rahul"}'
```

**Error Response:**
```json
{
  "error": "validation failed",
  "details": {
    "email": "email is required"
  }
}
```

### 🎉 Congratulations!
You just built a production-grade, validated Go API without the confusing boilerplate. Welcome to Kashvi!


---

# CRUD Walkthrough

This guide covers a full Create/Read/Update/Delete flow using Kashvi generators and runtime commands.

## 1. Scaffold a resource

Generate all CRUD files:

```bash
kashvi make:crud Post
```

You can also use flags:

```bash
kashvi make:crud Post --authorize --cache
```

- `--authorize`: route snippet printed by CLI includes an auth middleware placeholder.
- `--cache`: generated controller includes cache TODO placeholders in `Index` and `Show`.

## 2. Generated files

For `Post`, Kashvi creates:

- `app/models/post.go`
- `app/controllers/post_controller.go`
- `app/services/post_service.go`
- `database/migrations/<timestamp>_create_posts_table.go`
- `database/seeders/post_seeder.go`
- `testdata/post_scenarios.json`

## 3. Register routes

The generator prints lines to paste into your route setup. Typical wiring:

```go
package routes

import (
	"github.com/your-org/your-app/app/controllers"
	appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

func RegisterAPI(r *router.Router) {
	api := r.Group("/api")

	ctrl := controllers.NewPostController()
	api.Get("/posts", "posts.index", appctx.Wrap(ctrl.Index))
	api.Post("/posts", "posts.store", appctx.Wrap(ctrl.Store))
	api.Get("/posts/{id}", "posts.show", appctx.Wrap(ctrl.Show))
	api.Put("/posts/{id}", "posts.update", appctx.Wrap(ctrl.Update))
	api.Delete("/posts/{id}", "posts.destroy", appctx.Wrap(ctrl.Destroy))
}
```

Then ensure the route function is attached in `main.go`:

```go
app.New().
	Routes(routes.RegisterAPI).
	Run()
```

## 4. Implement migration

The generated migration registers automatically, but `Up` and `Down` are placeholders. Fill them:

```go
func (m *M_20260301010101_create_posts_table) Up(db *gorm.DB) error {
	type Post struct {
		gorm.Model
		Title string
		Body  string
	}
	return db.AutoMigrate(&Post{})
}

func (m *M_20260301010101_create_posts_table) Down(db *gorm.DB) error {
	return db.Migrator().DropTable("posts")
}
```

Run migration:

```bash
kashvi migrate
```

Inspect migration state:

```bash
kashvi migrate:status
```

## 5. Run and test endpoints

Start server:

```bash
kashvi serve
```

### Create

```bash
curl -X POST http://localhost:8080/api/posts \
  -H 'Content-Type: application/json' \
  -d '{}'
```

### List

```bash
curl http://localhost:8080/api/posts
```

### Show

```bash
curl http://localhost:8080/api/posts/1
```

### Update

```bash
curl -X PUT http://localhost:8080/api/posts/1 \
  -H 'Content-Type: application/json' \
  -d '{}'
```

### Delete

```bash
curl -X DELETE http://localhost:8080/api/posts/1 -i
```

The generated `Destroy` handler returns `204 No Content`.

## 6. Use generated test scenarios

`kashvi make:crud` creates `testdata/post_scenarios.json`. You can feed these scenarios into your `pkg/testkit` test runner and keep them as executable API documentation.

If you used `--authorize`, scenario entries include an `Authorization` header placeholder (`Bearer dummy-jwt-token`).

## 7. Next improvements

After initial scaffold, common upgrades are:

1. Replace empty request structs in controller methods with typed DTOs + validation tags.
2. Move DB logic into `app/services/post_service.go` and keep controllers thin.
3. Add middleware (`Auth`, rate-limit, role checks) per route group.
4. Add pagination in `Index` using `pkg/orm` pagination helpers.
5. Add queue jobs for side effects (notifications, emails, analytics writes).


---

# CLI Reference

All commands are run via the `kashvi` binary. Install with `make install`.

---

## Server Commands

### `kashvi run`
Start the HTTP server (`serve` alias). In project mode this delegates to your app entrypoint (`go run . serve`).

```bash
kashvi run
# → 🚀 Kashvi running on :8080  [env: local]
```

### `kashvi serve`
Alias for `kashvi run`.

### `kashvi build`
Compile the server binary to `./kashvi`.

```bash
kashvi build
# → ✅ Built: ./kashvi
```

### `kashvi route:list`
Print all named routes in a sorted table.

```bash
kashvi route:list

METHOD   PATH                         NAME
------   ----                         ----
DELETE   /api/posts/{id}              posts.destroy
GET      /api/health                  health
GET      /api/posts                   posts.index
GET      /api/posts/{id}              posts.show
GET      /api/profile                 auth.profile
POST     /api/login                   auth.login
POST     /api/posts                   posts.store
POST     /api/register                auth.register
PUT      /api/posts/{id}              posts.update
```

---

## Database Commands

### `kashvi migrate`
Run all pending migrations.

```bash
kashvi migrate
  ▶ Migrating: 20240101000000_create_users_table
  ✅ Migrated:  20240101000000_create_users_table
  ▶ Migrating: 20240102000000_create_posts_table
  ✅ Migrated:  20240102000000_create_posts_table
```

### `kashvi migrate:rollback`
Rollback the last batch of migrations.

```bash
kashvi migrate:rollback
  ◀ Rolling back: 20240102000000_create_posts_table
  ✅ Rolled back:  20240102000000_create_posts_table
```

### `kashvi migrate:status`
Show which migrations have been run.

```bash
kashvi migrate:status

Migration                                         Status    Batch
20240101000000_create_users_table                 Ran       1
20240102000000_create_posts_table                 Ran       1
20240103000000_add_role_to_users                  Pending   -
```

### `kashvi seed`
Run all database seeders.

```bash
kashvi seed
```

---

## Worker Commands

### `kashvi queue:work`
Start queue workers to process background jobs.

```bash
kashvi queue:work           # default: 5 workers
kashvi queue:work -w 10     # 10 workers
```

Workers run until SIGINT/SIGTERM, then finish the current job and exit.

### `kashvi schedule:run`
Start the task scheduler. Runs scheduled tasks at their configured times.

```bash
kashvi schedule:run
```

---

## Scaffold Commands

All scaffold commands create files in your project using a built-in `text/template` engine. They will **not overwrite** existing files.

### Template Overrides
You can customize the boilerplates for all scaffolding commands by mirroring the framework's `.stub` files into your project's `.kashvi/stubs/` directory.

```bash
mkdir -p .kashvi/stubs
# create .kashvi/stubs/model.stub to override the default model template
```

Available customizable stubs include: `model.stub`, `controller.stub`, `service.stub`, `migration.stub`, `seeder.stub`, and `test_scenario.stub`.

### `kashvi make:resource [Name]` (alias: `make:crud`)
**Most useful command.** Scaffolds a complete CRUD resource in one shot.

```bash
kashvi make:crud Post --authorize --cache
```

Creates:
- `app/models/post.go`
- `app/controllers/post_controller.go` (full CRUD using `ctx.Context`)
- `app/services/post_service.go`
- `database/migrations/TIMESTAMP_create_posts_table.go`
- `database/seeders/post_seeder.go`
- `testdata/post_scenarios.json` (Automated API tests)

Flags:
- `--authorize`: Injects standard Authentication router middleware and mocks JWT headers into the generated `test_scenario`.
- `--cache`: Adds caching template placeholders throughout the generated controller functions.

Prints the exact route lines to add to `api.go` with injected middleware flags accounted for.

---

### `kashvi make:model [Name]`
Scaffold a GORM model.

```bash
kashvi make:model Comment
# Creates: app/models/comment.go
```

### `kashvi make:controller [Name]`
Scaffold a basic controller.

```bash
kashvi make:controller Comment
# Creates: app/controllers/comment.go
```

### `kashvi make:service [Name]`
Scaffold a service layer struct.

```bash
kashvi make:service BillingService
# Creates: app/services/billingservice.go
```

### `kashvi make:migration [name]`
Create a new migration file with a timestamp prefix.

```bash
kashvi make:migration "add tags to posts"
# Creates: database/migrations/20260221170000_add_tags_to_posts.go
```

### `kashvi make:seeder [Name]`
Scaffold a seeder function.

```bash
kashvi make:seeder PostSeeder
# Creates: database/seeders/postseeder.go (name is lowercased)
```

---

## Tips

```bash
# See all available commands
kashvi --help

# See help for a specific command
kashvi make:resource --help
kashvi queue:work --help
```


---

# Configuration

Kashvi reads configuration from two sources, merged in order:

1. `config/app.json` — committed defaults
2. `.env` — local overrides (never commit this)

`.env` values always win over `config/app.json`.

---

## All Environment Variables

### Application

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `local` | `local` / `production` / `prod` |
| `APP_PORT` | `8080` | HTTP server port |
| `JWT_SECRET` | *(insecure default)* | **Must be changed in production** |
| `MAX_BODY_BYTES` | `4194304` (4 MB) | Max JSON request body size |

> [!CAUTION]
> The server **refuses to start** in production if `JWT_SECRET` is the default value.

---

### Database

| Variable | Default | Description |
|---|---|---|
| `DB_DRIVER` | `sqlite` | `sqlite` / `postgres` / `mysql` / `sqlserver` |
| `DATABASE_DSN` | `kashvi.db` | Full connection DSN |

**DSN examples:**
```ini
# SQLite (dev)
DATABASE_DSN=kashvi.db

# PostgreSQL
DATABASE_DSN=host=localhost user=postgres password=secret dbname=kashvi port=5432 sslmode=disable

# MySQL
DATABASE_DSN=root:secret@tcp(127.0.0.1:3306)/kashvi?charset=utf8mb4&parseTime=True&loc=Local
```

---

### Redis

| Variable | Default | Description |
|---|---|---|
| `REDIS_ADDR` | `localhost:6379` | Redis host:port |
| `REDIS_PASSWORD` | *(empty)* | Redis auth password |

> Redis is **non-fatal** — the server starts with a warning if Redis is unavailable and degrades gracefully (sessions won't persist, cache misses).

---

### Storage

| Variable | Default | Description |
|---|---|---|
| `STORAGE_DISK` | `local` | `local` or `s3` |
| `STORAGE_LOCAL_ROOT` | `storage` | Root directory for local disk |
| `STORAGE_URL` | `http://localhost:8080/storage` | Public URL for local files |

**S3 / MinIO / R2 / Spaces:**

| Variable | Default | Description |
|---|---|---|
| `S3_BUCKET` | *(required)* | Bucket name |
| `S3_REGION` | `us-east-1` | AWS region |
| `S3_KEY` | | Access key ID |
| `S3_SECRET` | | Secret access key |
| `S3_ENDPOINT` | | Custom endpoint (MinIO/R2 — leave empty for AWS) |
| `S3_URL` | | Public base URL (defaults to AWS URL pattern) |

---

## Reading Config in Code

```go
import "github.com/shashiranjanraj/kashvi/config"

port   := config.AppPort()      // "8080"
env    := config.AppEnv()       // "local"
secret := config.JWTSecret()
bucket := config.StorageS3Bucket()

// Generic getter with a default:
val := config.Get("MY_CUSTOM_VAR", "default-value")
```

---

## `config/app.json` Format

```json
{
  "app_env":      "local",
  "app_port":     "8080",
  "jwt_secret":   "change-me",
  "db_driver":    "sqlite",
  "database_dsn": "kashvi.db",
  "redis_addr":   "localhost:6379"
}
```

Keys in `app.json` map 1:1 to env variable names (lowercase, underscores).


---

# Routing

Routes are registered in `app/routes/api.go`.

---

## Basic Routes

```go
func RegisterAPI(r *router.Router) {
    r.Get("/ping", "ping", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    })

    r.Post("/users",      "users.store",   handler)
    r.Put("/users/{id}",  "users.update",  handler)
    r.Patch("/users/{id}","users.patch",   handler)
    r.Delete("/users/{id}","users.destroy",handler)
}
```

---

## Using `ctx.Context` (recommended)

```go
import appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"

r.Get("/users/{id}", "users.show", appctx.Wrap(func(c *appctx.Context) {
    id := c.Param("id")
    c.Success(map[string]any{"id": id})
}))
```

---

## Route Groups

Groups let you share a path prefix and/or middleware across multiple routes:

```go
// All routes under /api with rate limiting
api := r.Group("/api", middleware.RateLimit(120, time.Minute))

api.Get("/users", "users.index", appctx.Wrap(ctrl.Index))
api.Post("/users", "users.store", appctx.Wrap(ctrl.Store))

// Nested group: /api/admin with auth guard
admin := api.Group("/admin", middleware.AuthMiddleware, middleware.RequireRole("admin"))
admin.Get("/stats", "admin.stats", appctx.Wrap(adminCtrl.Stats))
```

---

## URL Parameters

```go
// Define: /articles/{slug}/comments/{id}
r.Get("/articles/{slug}/comments/{id}", "comments.show", appctx.Wrap(func(c *appctx.Context) {
    slug := c.Param("slug")
    id   := c.Param("id")
    // ...
}))
```

---

## Named Routes & URL Generation

Every route takes a name as the second argument. Names let you generate URLs safely:

```go
// Registration
r.Get("/users/{id}", "users.show", handler)

// URL generation (anywhere in your code)
url, err := myRouter.URL("users.show", map[string]string{"id": "42"})
// url = "/users/42"
```

---

## Mounting Third-Party Handlers

```go
// Prometheus metrics (already wired by framework)
r.HandleFunc("/metrics", metrics.Handler())

// Any http.Handler
r.Mount("/storage", http.FileServer(http.Dir("storage")))
```

---

## Listing All Routes

```bash
kashvi route:list
```

Output:
```
METHOD   PATH                    NAME
------   ----                    ----
DELETE   /api/users/{id}         users.destroy
GET      /api/health             health
GET      /api/users              users.index
GET      /api/users/{id}         users.show
POST     /api/login              auth.login
POST     /api/register           auth.register
POST     /api/users              users.store
PUT      /api/users/{id}         users.update
```

---

## Per-Route Middleware

Middleware can be applied to individual routes as variadic arguments:

```go
api.Get("/admin/report", "admin.report",
    appctx.Wrap(adminCtrl.Report),
    middleware.AuthMiddleware,
    middleware.RequireRole("admin"),
)
```


---

# Context API

`pkg/ctx` provides a `gin.Context`-inspired request context for Kashvi handlers.
Instead of `(http.ResponseWriter, *http.Request)`, your handler receives a single `*ctx.Context`.

---

## Handler Signature

```go
import appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"

func MyHandler(c *appctx.Context) {
    // use c for everything
}

// Register with ctx.Wrap():
r.Get("/path", "name", appctx.Wrap(MyHandler))
```

---

## Reading the Request

### URL Parameters
```go
id   := c.Param("id")     // /users/{id}
slug := c.Param("slug")   // /posts/{slug}
```

### Query String
```go
page    := c.Query("page")                  // "" if absent
sort    := c.DefaultQuery("sort", "created_at")
```

### Request Body (JSON)
```go
// Automatic — decodes + validates, sends 422 on failure
var input struct {
    Name  string `json:"name"  validate:"required,min=2"`
    Email string `json:"email" validate:"required,email"`
}
if !c.BindJSON(&input) {
    return  // response already sent
}

// Manual — returns errors to handle yourself
errs, err := c.ShouldBindJSON(&input)
if err != nil { /* bad JSON */ }
if len(errs) > 0 { /* validation errors */}
```

### Form Data
```go
name := c.PostForm("name")
```

### Headers & Cookies
```go
token  := c.Header("Authorization")
accept := c.Header("Accept")

val, err := c.Cookie("session_id")
```

### Metadata
```go
method := c.Method()     // "GET"
path   := c.Path()       // "/api/users/42"
full   := c.FullPath()   // "GET /api/users/42"
ip     := c.ClientIP()   // respects X-Forwarded-For
isXHR  := c.IsXHR()      // X-Requested-With: XMLHttpRequest
ctx    := c.Context()    // underlying context.Context
```

### Raw Body
```go
bytes, err := c.Body()
```

---

## Sending Responses

### JSON
```go
c.JSON(200, map[string]any{"key": "value"})

// Pre-wrapped envelopes:
c.Success(data)         // 200 {"status":200,"data":{...}}
c.Created(data)         // 201 {"status":201,"data":{...}}
c.Error(400, "Bad req") // 4xx {"status":400,"message":"..."}
c.ValidationError(errs) // 422 {"status":422,"message":"Validation failed","errors":{...}}

// Shortcuts:
c.Unauthorized()        // 401
c.Unauthorized("Token expired")
c.Forbidden()           // 403
c.NotFound()            // 404
c.NotFound("Post not found")
```

### Other response types
```go
c.String(200, "Hello, %s!", name)
c.Status(204)               // status only, no body
c.Redirect(302, "/login")
c.File("/path/to/file.pdf")
```

### Headers & Cookies
```go
c.SetHeader("X-Request-Id", "abc123")
c.SetCookie("token", value, 3600, "/", "", true, true)
```

---

## Per-Request Store

Pass values between middleware and handlers via the request-scoped store:

```go
// In middleware (e.g. AuthMiddleware):
c.Set("user_id", claims.UserID)
c.Set("role", claims.Role)

// In handler:
userID := c.GetUint("user_id")
role   := c.GetString("role")

// Generic (any type):
val, ok := c.Get("key")
val      = c.MustGet("key") // panics if missing
```

---

## Abort

```go
func AdminOnly(c *appctx.Context) {
    if c.GetString("role") != "admin" {
        c.Abort(403, "Admin access required")
        return
    }
    // continue
}
```

---

## Validate Without Binding

```go
type Input struct {
    Age int `json:"age" validate:"required,min=18"`
}
var input Input
// ... populate input ...
errs := c.Validate(&input)
if len(errs) > 0 {
    c.ValidationError(errs)
    return
}
```

---

## Pool Efficiency

`pkg/ctx` uses `sync.Pool` internally — `Context` objects are **recycled between requests**, resulting in zero allocations per request.


---

# Validation

Kashvi's validation engine lives in `pkg/validate`. It has **zero external dependencies** and supports 28 rules via struct tags.

---

## Struct Tags

Add a `validate` tag to any field:

```go
type RegisterInput struct {
    Name            string  `json:"name"             validate:"required,min=2,max=100"`
    Email           string  `json:"email"            validate:"required,email"`
    Age             int     `json:"age"              validate:"required,min=18,max=120"`
    Role            string  `json:"role"             validate:"in=admin,user,editor"`
    Password        string  `json:"password"         validate:"required,min=8"`
    PasswordConfirm string  `json:"password_confirm" validate:"confirmed=password"`
    Website         *string `json:"website"          validate:"nullable,url"`
}
```

---

## All Validation Rules

| Rule | Example Tag | Description |
|---|---|---|
| `required` | `validate:"required"` | Field must be non-zero |
| `email` | `validate:"email"` | Valid email address |
| `min` | `validate:"min=3"` | String min length / numeric min value |
| `max` | `validate:"max=100"` | String max length / numeric max value |
| `between` | `validate:"between=1,10"` | Numeric between two values (inclusive) |
| `in` | `validate:"in=a,b,c"` | Value must be one of the listed options |
| `not_in` | `validate:"not_in=bad,worse"` | Value must NOT be in the list |
| `confirmed` | `validate:"confirmed=password"` | Must match another field's value |
| `url` | `validate:"url"` | Valid HTTP/HTTPS URL |
| `alpha` | `validate:"alpha"` | Letters only |
| `alpha_num` | `validate:"alpha_num"` | Letters and numbers only |
| `alpha_dash` | `validate:"alpha_dash"` | Letters, numbers, `-`, `_` |
| `numeric` | `validate:"numeric"` | Any number (int or float) |
| `integer` | `validate:"integer"` | Must be an integer |
| `boolean` | `validate:"boolean"` | true or false |
| `ip` | `validate:"ip"` | Valid IPv4 or IPv6 address |
| `uuid` | `validate:"uuid"` | Valid UUID |
| `date` | `validate:"date"` | Valid date in `YYYY-MM-DD` format |
| `date_format` | `validate:"date_format=2006-01-02"` | Custom Go time layout |
| `starts_with` | `validate:"starts_with=https"` | String prefix check |
| `ends_with` | `validate:"ends_with=.go"` | String suffix check |
| `contains` | `validate:"contains=@"` | Substring check |
| `regex` | `validate:"regex=^[A-Z]+"` | Custom regex pattern |
| `json` | `validate:"json"` | Valid JSON string |
| `len` | `validate:"len=6"` | Exact string length |
| `same` | `validate:"same=other_field"` | Alias for `confirmed` |
| `different` | `validate:"different=old_password"` | Must differ from field |
| `nullable` | `validate:"nullable,email"` | Skip all other rules if the field is nil/zero |

---

## Using Validation Directly

### In a handler with `BindJSON`:
```go
func (ctrl *UserController) Register(c *appctx.Context) {
    var input RegisterInput
    if !c.BindJSON(&input) {
        return // 422 already sent
    }
    // input is valid here
}
```

### Manual validation:
```go
import "github.com/shashiranjanraj/kashvi/pkg/validate"

errs := validate.Struct(&input)
if validate.HasErrors(errs) {
    // errs = map[string]string{"email": "The email field must be a valid email address."}
}
```

---

## Error Messages

Errors are returned as `map[string]string` where the key is the JSON field name:

```json
{
  "status": 422,
  "message": "Validation failed",
  "errors": {
    "email": "The email field must be a valid email address.",
    "password": "The password field must be at least 8 characters.",
    "password_confirm": "The password_confirm field must match password."
  }
}
```

---

## Nullable Fields

Use `nullable` to skip all other rules when the field is empty/nil:

```go
type UpdateInput struct {
    // These are all optional — only validated if provided
    Bio     *string `json:"bio"     validate:"nullable,max=500"`
    Website *string `json:"website" validate:"nullable,url"`
    Age     *int    `json:"age"     validate:"nullable,min=18"`
}
```

---

## Combining Rules

Rules are comma-separated and evaluated in order. All failures are collected (not short-circuit):

```go
validate:"required,min=8,max=64,alpha_num"
```


---

# Authentication

Kashvi includes JWT-based authentication with bcrypt passwords and RBAC role guards via `pkg/auth`.

---

## Generating Tokens

```go
import "github.com/shashiranjanraj/kashvi/pkg/auth"

// Access token (24h)
token, err := auth.GenerateToken(user.ID, user.Role)

// Refresh token (7d)
refresh, err := auth.GenerateRefreshToken(user.ID, user.Role)
```

---

## Validating Tokens

```go
claims, err := auth.ValidateToken(tokenString)
if err != nil {
    // expired, invalid signature, etc.
}

userID := claims.UserID   // uint
role   := claims.Role     // string
```

---

## Password Hashing

```go
// Hash on register
hash, err := auth.HashPassword("secret123")

// Verify on login
if !auth.CheckPassword(storedHash, "secret123") {
    // wrong password
}
```

---

## Auth Middleware

Apply `middleware.AuthMiddleware` to protect routes:

```go
protected := api.Group("", middleware.AuthMiddleware)
protected.Get("/profile", "auth.profile", appctx.Wrap(ctrl.Profile))
```

The middleware:
1. Reads `Authorization: Bearer <token>` header
2. Validates the JWT
3. Stores `user_id` and `role` in the request context
4. Returns `401` if missing or invalid

**Reading the authenticated user in a handler:**

```go
func (ctrl *AuthController) Profile(c *appctx.Context) {
    userID := c.GetUint("user_id")
    role   := c.GetString("role")

    var user models.User
    database.DB.First(&user, userID)
    c.Success(user)
}
```

---

## Role-Based Access Control (RBAC)

### Require a specific role:

```go
adminRoutes := api.Group("/admin",
    middleware.AuthMiddleware,
    middleware.RequireRole("admin"),
)
adminRoutes.Get("/users", "admin.users", appctx.Wrap(ctrl.AllUsers))
```

### Require any of multiple roles:

```go
middleware.RequireRole("admin", "moderator")
```

### Allow guest access:

```go
// Route accessible without auth
api.Get("/posts", "posts.index", appctx.Wrap(ctrl.Index))
```

---

## Full Login Flow Example

```go
// POST /api/login
func (c *AuthController) Login(ctx *appctx.Context) {
    var input struct {
        Email    string `json:"email"    validate:"required,email"`
        Password string `json:"password" validate:"required"`
    }
    if !ctx.BindJSON(&input) {
        return
    }

    var user models.User
    if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
        ctx.Error(http.StatusUnauthorized, "Invalid credentials")
        return
    }

    if !auth.CheckPassword(user.Password, input.Password) {
        ctx.Error(http.StatusUnauthorized, "Invalid credentials")
        return
    }

    token, _   := auth.GenerateToken(user.ID, user.Role)
    refresh, _ := auth.GenerateRefreshToken(user.ID, user.Role)

    ctx.Success(map[string]any{
        "access_token":  token,
        "refresh_token": refresh,
        "user":          user,
    })
}
```

---

## JWT Configuration

| Env Var | Default | Notes |
|---|---|---|
| `JWT_SECRET` | *insecure* | **Must change in production** — server refuses to start otherwise |

Access tokens expire in **24 hours**, refresh tokens in **7 days**.
Both values can be changed in `pkg/auth/jwt.go`.


---

# ORM & Database

Kashvi wraps GORM with a fluent chainable query builder in `pkg/orm`.

---

## Connection

The database is connected at server boot. Configure via `.env`:

```ini
DB_DRIVER=postgres
DATABASE_DSN=host=localhost user=postgres dbname=kashvi sslmode=disable
```

Use `database.DB` to get the GORM instance anywhere:

```go
import "github.com/shashiranjanraj/kashvi/pkg/database"

database.DB.Create(&user)
```

---

## Basic CRUD

```go
import (
    "github.com/shashiranjanraj/kashvi/pkg/database"
    "github.com/shashiranjanraj/kashvi/pkg/orm"
)

q := orm.New(database.DB)

// Create
q.Create(&models.Post{Title: "Hello", Body: "World"})

// Find by ID
var post models.Post
q.Find(&post, 1)

// Update
q.Where("id = ?", 1).Update(&models.Post{}, map[string]any{"title": "Updated"})

// Delete
q.Where("id = ?", 1).Delete(&models.Post{})
```

---

## Query Builder

```go
q := orm.New(database.DB)

// Filtering
q.Where("status = ?", "active").
  Where("created_at > ?", time.Now().AddDate(0, -1, 0))

// Ordering & limiting
q.OrderBy("created_at DESC").Limit(10).Offset(20)

// Select specific columns
q.Select("id", "title", "created_at")

// Eager loading
q.With("Author", "Tags")

// Execute
var posts []models.Post
q.Get(&posts)
```

---

## Pagination

```go
func (ctrl *PostController) Index(c *appctx.Context) {
    var posts []models.Post

    pagination, err := orm.New(database.DB).
        Where("published = ?", true).
        OrderBy("created_at DESC").
        Paginate(&posts, c.R)  // reads ?page=1&per_page=15 from request

    if err != nil {
        c.Error(500, "Failed to fetch posts")
        return
    }

    response.Paginated(c.W, posts, pagination)
}
```

Response:
```json
{
  "status": 200,
  "data": {
    "items": [...],
    "pagination": {
      "total": 150,
      "per_page": 15,
      "current_page": 1,
      "last_page": 10,
      "from": 1,
      "to": 15
    }
  }
}
```

---

## Parallel Queries

Run multiple queries concurrently and wait for all results:

```go
var users []models.User
var posts []models.Post
var tags  []models.Tag

orm.Parallel(
    func() { database.DB.Find(&users) },
    func() { database.DB.Where("published = ?", true).Find(&posts) },
    func() { database.DB.Find(&tags) },
)

// All three queries ran simultaneously
```

---

## ORM Cache Bridge

Cache query results in Redis automatically:

```go
var user models.User
orm.New(database.DB).
    Cache("user:42", 5*time.Minute).
    Find(&user, 42)
// Second call hits Redis, not the DB
```

---

## Models

Define models in `app/models/`:

```go
package models

import "gorm.io/gorm"

type Post struct {
    gorm.Model          // ID, CreatedAt, UpdatedAt, DeletedAt
    Title     string    `gorm:"size:255;not null"`
    Body      string    `gorm:"type:text"`
    Published bool      `gorm:"default:false"`
    UserID    uint
    User      User      // belongs to
    Tags      []Tag     `gorm:"many2many:post_tags;"`
}
```

---

## Raw Queries

```go
var result []map[string]any
database.DB.Raw("SELECT id, title FROM posts WHERE published = ?", true).Scan(&result)
```

---

## Connection Pool Settings (auto-configured)

| Setting | Value |
|---|---|
| Max open connections | 25 |
| Max idle connections | 10 |
| Max conn lifetime | 5 minutes |
| Max idle time | 2 minutes |


---

# Migrations & Seeders

## Creating a Migration

```bash
kashvi make:migration create_posts_table
```

Edit the generated file:

```go
package migrations

import (
    "github.com/shashiranjanraj/kashvi/pkg/migration"
    "github.com/shashiranjanraj/kashvi/app/models"
    "gorm.io/gorm"
)

func init() {
    migration.Register("20260221_create_posts_table", &M_CreatePostsTable{})
}

type M_CreatePostsTable struct{}

func (m *M_CreatePostsTable) Up(db *gorm.DB) error {
    return db.AutoMigrate(&models.Post{})
}

func (m *M_CreatePostsTable) Down(db *gorm.DB) error {
    return db.Migrator().DropTable("posts")
}
```

## Running Migrations

```bash
kashvi migrate              # run all pending
kashvi migrate:rollback     # rollback last batch
kashvi migrate:status       # show status
```

## Seeders

```bash
kashvi make:seeder PostSeeder
```

```go
func PostSeeder(db *gorm.DB) error {
    posts := []models.Post{
        {Title: "Hello World", Body: "First post!", Published: true},
    }
    return db.Create(&posts).Error
}
```

Register in `database/seeders/run_all.go`:

```go
func RunAll(db *gorm.DB) error {
    for _, seeder := range []func(*gorm.DB) error{
        UserSeeder,
        PostSeeder,
    } {
        if err := seeder(db); err != nil {
            return err
        }
    }
    return nil
}
```

```bash
kashvi seed
```


---

# Queue & Jobs

Kashvi's queue system (`pkg/queue`) supports background job processing with retry, backoff, and persistent failure tracking.

---

## Defining a Job

```go
// app/jobs/welcome_email_job.go
package jobs

type WelcomeEmailJob struct {
    UserID uint   `json:"user_id"`
    Email  string `json:"email"`
}

func (j WelcomeEmailJob) Handle() error {
    // send email...
    return mailer.Send(j.Email, "Welcome!", "welcome.html")
}
```

Register the job type at boot (so it can be deserialized):

```go
// In main.go init() or a jobs/register.go file:
queue.Register("jobs.WelcomeEmailJob", func() queue.Job {
    return &jobs.WelcomeEmailJob{}
})
```

---

## Dispatching Jobs

```go
import "github.com/shashiranjanraj/kashvi/pkg/queue"

// Immediate
queue.Dispatch(jobs.WelcomeEmailJob{UserID: user.ID, Email: user.Email})

// After a delay (5 minutes)
queue.DispatchAfter(jobs.WelcomeEmailJob{UserID: user.ID, Email: user.Email}, 5*time.Minute)
```

---

## Queue Drivers

### In-Memory (default — dev only)

Jobs are lost on restart. Good for development and testing.

```go
// Default — no configuration needed
queue.Dispatch(myJob)
```

### Redis Driver (production)

Jobs survive restarts. Delayed jobs use Redis sorted sets.

```go
// In server.go or a boot function, after cache.Connect():
import (
    "github.com/shashiranjanraj/kashvi/pkg/cache"
    "github.com/shashiranjanraj/kashvi/pkg/queue"
)

queue.SetDriver(queue.NewRedisDriver(cache.RDB))
```

Redis keys used:
- `kashvi:queue:jobs` — immediate job list (LPUSH/BRPOP)
- `kashvi:queue:delayed` — delayed job sorted set (score = Unix timestamp)

---

## Starting Workers

```bash
# From CLI (production)
kashvi queue:work --workers=5

# Or programmatically:
queue.StartWorkers(ctx, 5)
```

---

## Retry & Backoff

Failed jobs are automatically retried with linear backoff:
- Attempt 1 → wait 1s → Attempt 2 → wait 2s → Attempt 3

```go
// Change retry limit (default: 3)
queue.SetMaxRetry(5)
```

---

## Failed Jobs

After all retries are exhausted, the job is recorded in:
1. **In-memory** — available via `queue.FailedJobs()`
2. **Database** — `kashvi_failed_jobs` table (if `queue.UseDB()` is called)

The database persistence is wired automatically at server boot.

**Table structure:**

| Column | Type | Description |
|---|---|---|
| `id` | uint | Auto-increment PK |
| `job_type` | string | Go type name |
| `payload` | text | JSON-encoded job data |
| `error` | text | Last error message |
| `attempts` | int | Number of attempts made |
| `failed_at` | timestamp | When it failed |

**Querying failures:**

```go
// In memory
failed := queue.FailedJobs()
for _, f := range failed {
    fmt.Printf("%T failed after %d attempts: %v\n", f.Job, f.Attempts, f.Err)
}

// From DB
var records []queue.FailedJobRecord
database.DB.Order("failed_at desc").Find(&records)
```

---

## Full Example — Order Processing

```go
type ProcessOrderJob struct {
    OrderID uint `json:"order_id"`
}

func (j ProcessOrderJob) Handle() error {
    var order models.Order
    if err := database.DB.First(&order, j.OrderID).Error; err != nil {
        return err // will be retried
    }
    // charge card, update inventory, send confirmation...
    return nil
}

// In your controller:
func (c *OrderController) Store(ctx *appctx.Context) {
    // ... create order ...
    queue.Dispatch(ProcessOrderJob{OrderID: order.ID})
    ctx.Created(order)
}
```


---

# Worker Pool

`pkg/workerpool` provides a **bounded goroutine pool** that limits concurrent goroutine creation under high load. Use it for CPU-intensive or I/O-heavy tasks that should not run in unbounded goroutines.

---

## Why use a pool?

| Approach | Problem |
|----------|---------|
| `go doWork()` for every request | Goroutines spike unboundedly under load — OOM risk |
| Worker pool | Hard ceiling on concurrency — predictable memory |

---

## Configuration

```ini
# .env
WORKER_POOL_SIZE=50   # default: 50
```

---

## Basic usage

```go
import "github.com/shashiranjanraj/kashvi/pkg/workerpool"

// Create a pool (use config.WorkerPoolSize() for env-driven size)
pool := workerpool.New(config.WorkerPoolSize())
defer pool.Shutdown()

// Non-blocking submit
err := pool.Submit(func() {
    processImage(imageData)
})
if errors.Is(err, workerpool.ErrPoolFull) {
    // Pool is busy — return 429, push to queue, etc.
    c.JSON(http.StatusTooManyRequests, map[string]string{"error": "server busy"})
    return
}
```

---

## Blocking submit

When you want to wait until a slot is available:

```go
err := pool.SubmitWait(func() {
    sendReportEmail(userID)
})
if errors.Is(err, workerpool.ErrPoolClosed) {
    // Pool was shut down
}
```

---

## Shutdown

`Shutdown()` stops accepting new tasks, waits for all in-flight tasks to complete, then releases all worker goroutines. Safe to call multiple times.

```go
pool.Shutdown()
```

---

## Error reference

| Error | When |
|-------|------|
| `workerpool.ErrPoolFull` | All workers are busy and the queue buffer is full |
| `workerpool.ErrPoolClosed` | `Shutdown()` has been called |

---

## Panic safety

Workers recover from panics automatically — a bad task never kills the pool or unexpectedly terminates a goroutine. The next task runs as normal.

---

## Sizing guide

| Use case | Recommended size |
|----------|-----------------|
| Image processing | `runtime.NumCPU()` |
| Network I/O (external APIs) | 50–200 |
| DB queries | 20–50 (limited by DB connection pool) |
| Mixed workloads | `WORKER_POOL_SIZE=50` (default) |

---

## Buffer size

The internal task queue buffer is `2 × size`. This absorbs short bursts without returning `ErrPoolFull`. For example, a pool of 50 workers can buffer 100 pending tasks before backpressure kicks in.

---

## Integration with HTTP handlers

A good pattern: create one shared pool at app startup and use it across handlers.

```go
// internal/kernel/http.go
var Pool = workerpool.New(config.WorkerPoolSize())

// In a handler
func GenerateReport(c *ctx.Context) {
    err := kernel.Pool.Submit(func() {
        report := buildReport(c.ParamInt("id"))
        cache.Set("report:"+id, report, time.Hour)
    })
    if errors.Is(err, workerpool.ErrPoolFull) {
        c.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
        return
    }
    c.JSON(http.StatusAccepted, map[string]string{"status": "processing"})
}
```


---

# Storage

`pkg/storage` provides a unified file-storage API inspired by Laravel's Storage facade.
Switch between local disk and S3-compatible storage with a single env variable.

---

## Configuration

```ini
STORAGE_DISK=local      # default driver: "local" or "s3"
```

---

## Using the Default Disk

```go
import "github.com/shashiranjanraj/kashvi/pkg/storage"

// Write
storage.Put("avatars/user-1.jpg", imageBytes)
storage.PutStream("uploads/file.pdf", r.Body)

// Read
data, err := storage.Get("avatars/user-1.jpg")
stream, err := storage.GetStream("uploads/file.pdf")
defer stream.Close()

// Metadata
exists  := storage.Exists("avatars/user-1.jpg")
missing := storage.Missing("avatars/user-1.jpg")
size, _ := storage.Size("avatars/user-1.jpg")
modTime, _ := storage.LastModified("avatars/user-1.jpg")

// Public URL
url := storage.URL("avatars/user-1.jpg")

// Delete
storage.Delete("avatars/user-1.jpg")

// Copy / Move
storage.Copy("tmp/upload.jpg", "images/final.jpg")
storage.Move("tmp/upload.jpg", "archive/old.jpg")

// Directories
files, _ := storage.Files("avatars")          // non-recursive
all, _   := storage.AllFiles("avatars")       // recursive
dirs, _  := storage.Directories("uploads")
storage.MakeDirectory("exports")
storage.DeleteDirectory("tmp")
```

---

## Using a Specific Disk

```go
// Use S3 explicitly
storage.Use("s3").Put("backups/db.sql.gz", data)

// Use local disk explicitly
storage.Use("local").Get("cache/data.json")
```

> Method name is `Use()` (not `Disk()`) to avoid conflict with the `Disk` interface type.

---

## File Upload Handler

```go
func (c *UploadController) Store(ctx *appctx.Context) {
    ctx.R.ParseMultipartForm(10 << 20) // 10MB max

    file, header, err := ctx.R.FormFile("file")
    if err != nil {
        ctx.Error(400, "No file uploaded")
        return
    }
    defer file.Close()

    path := fmt.Sprintf("uploads/%d_%s", time.Now().Unix(), header.Filename)
    if err := storage.PutStream(path, file); err != nil {
        ctx.Error(500, "Upload failed")
        return
    }

    ctx.Created(map[string]any{
        "path": path,
        "url":  storage.URL(path),
    })
}
```

---

## Local Disk

Files are stored relative to `STORAGE_LOCAL_ROOT` (default: `./storage`).

Public access: `GET /storage/{path}` is automatically mounted when `STORAGE_DISK=local`.

```ini
STORAGE_LOCAL_ROOT=storage
STORAGE_URL=http://localhost:8080/storage
```

---

## S3 / AWS

```ini
STORAGE_DISK=s3
S3_BUCKET=my-bucket
S3_REGION=us-east-1
S3_KEY=AKIAIOSFODNN7EXAMPLE
S3_SECRET=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
S3_URL=https://my-bucket.s3.us-east-1.amazonaws.com
```

---

## MinIO (self-hosted S3)

Run locally with Docker:

```bash
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"
```

```ini
STORAGE_DISK=s3
S3_BUCKET=my-bucket
S3_KEY=minioadmin
S3_SECRET=minioadmin
S3_ENDPOINT=http://localhost:9000
S3_REGION=us-east-1
```

Create the bucket at `http://localhost:9001` (MinIO console UI).

---

## Cloudflare R2 / DigitalOcean Spaces

Same as MinIO — just set `S3_ENDPOINT` to your provider's endpoint URL.

```ini
# Cloudflare R2
S3_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com

# DigitalOcean Spaces
S3_ENDPOINT=https://nyc3.digitaloceanspaces.com
```

---

## Custom Driver

Implement the `Disk` interface and register it:

```go
type MyDriver struct{}
func (d *MyDriver) Put(path string, content []byte) error { ... }
// ... implement all 16 Disk interface methods

// Register at boot:
storage.RegisterDisk("mydriver", &MyDriver{})

// Use:
storage.Use("mydriver").Put("file.txt", data)
```


---

# WebSocket & SSE

---

## WebSocket (`pkg/ws`)

Kashvi's WebSocket support uses the [gorilla/websocket](https://github.com/gorilla/websocket) library with a Hub/Client pattern for broadcasting to multiple connected clients.

### 1. Create and start a Hub

```go
// In your package (e.g. app/hubs/chat.go):
package hubs

import "github.com/shashiranjanraj/kashvi/pkg/ws"

var Chat = ws.NewHub()

func init() {
    go Chat.Run() // starts the event loop
}
```

### 2. Register the WebSocket route

```go
import (
    "github.com/shashiranjanraj/kashvi/app/hubs"
    appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"
    "github.com/shashiranjanraj/kashvi/pkg/ws"
)

r.Get("/ws/chat", "ws.chat", appctx.Wrap(func(c *appctx.Context) {
    ws.Upgrade(c.W, c.R, hubs.Chat)
}))
```

### 3. Handle inbound messages

```go
hubs.Chat.OnMessage = func(hub *ws.Hub, msg ws.Message) {
    // Echo back to all clients
    hub.Broadcast <- msg.Data

    // Or respond only to the sender
    msg.Client.Send([]byte(`{"type":"ack"}`))
}
```

### 4. Broadcast from anywhere

```go
// From a controller, job, or anywhere:
hubs.Chat.Broadcast <- []byte(`{"type":"update","data":"live score changed"}`)

// Check connected clients
count := hubs.Chat.ClientCount()
```

### Features

- **Ping/Pong keepalive** — automatically sends WebSocket `ping` frames every 54s
- **Client buffer** — each client has a 256-message send buffer; slow clients are disconnected
- **Origin check** — configurable (allow-all by default):
  ```go
  ws.SetCheckOrigin(func(r *http.Request) bool {
      return r.Header.Get("Origin") == "https://myapp.com"
  })
  ```

### WebSocket JavaScript client

```javascript
const socket = new WebSocket("ws://localhost:8080/ws/chat");

socket.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log("received:", data);
};

socket.send(JSON.stringify({ type: "message", text: "Hello!" }));
```

---

## Server-Sent Events (`pkg/sse`)

SSE is a one-way push from server to browser over a regular HTTP connection. Great for live feeds, notifications, dashboards.

### Route handler

```go
r.Get("/events/feed", "sse.feed", appctx.Wrap(func(c *appctx.Context) {
    stream := sse.New(c.W, c.R)
    if stream == nil {
        return // client doesn't support SSE
    }

    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case t := <-ticker.C:
            stream.Send("tick", map[string]any{
                "time":  t.Format(time.RFC3339),
                "count": hubs.Chat.ClientCount(),
            })
        }

        if stream.IsClosed() {
            break // client disconnected
        }
    }
}))
```

### SSE Methods

```go
stream := sse.New(w, r)

// Named event with JSON data
stream.Send("update", map[string]any{"id": 1, "status": "done"})

// Plain data line
stream.SendRaw("hello world")

// Keepalive heartbeat (comment line)
stream.Comment("heartbeat")

// Check if client disconnected
if stream.IsClosed() { return }
```

### JavaScript client

```javascript
const es = new EventSource("/events/feed");

es.addEventListener("tick", (event) => {
    const data = JSON.parse(event.data);
    console.log("tick:", data.time);
});

es.addEventListener("update", (event) => {
    const data = JSON.parse(event.data);
    console.log("update:", data);
});
```

---

## WebSocket vs SSE

| | WebSocket | SSE |
|---|---|---|
| Direction | Bidirectional | Server → Client only |
| Protocol | Custom (ws://) | HTTP |
| Reconnect | Manual | Automatic |
| Use case | Chat, games, live collab | Notifications, feeds, dashboards |
| Browser support | All | All (IE11+) |


---

# gRPC Server

Kashvi includes a production-ready gRPC server that runs **alongside** the HTTP server on a separate port. It ships with a health-check service, server reflection, and pre-wired Prometheus metrics.

---

## Configuration

```ini
# .env
GRPC_PORT=9090    # default: 9090
```

---

## What starts automatically

When you run `kashvi run`, **both** servers boot:

```
🚀 Kashvi HTTP  on :8080  [env: local]  [workers: 8]
🔌 Kashvi gRPC  on :9090
```

At shutdown (`Ctrl+C`), the gRPC server drains in-flight RPCs before exiting.

---

## Built-in interceptors (applied automatically)

| Order | Interceptor | What it does |
|-------|-------------|--------------|
| 1 | **Recovery** | Catches panics → returns `INTERNAL` status instead of crashing |
| 2 | **Logging** | Logs every RPC: `method`, `duration_ms`, `code` |
| 3 | **Prometheus** | `grpc_server_handled_total`, `grpc_server_handling_seconds` |

---

## Built-in services

### Health (grpc.health.v1.Health)

Always returns `SERVING`. Test with:

```bash
# brew install grpcurl
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
# → { "status": "SERVING" }
```

### Server Reflection

Enabled automatically — `grpcurl` works without proto files:

```bash
grpcurl -plaintext localhost:9090 list
# → grpc.health.v1.Health
```

---

## Registering your own service

```go
// pkg/grpc/server.go  — add after reflection.Register(srv)
mypb.RegisterUserServiceServer(srv, &UserServiceImpl{})
```

Or call `grpc.Start()` manually and register before the goroutine runs:

```go
grpcSrv, lis, _ := kashvigrpc.Start(config.GRPCPort())
mypb.RegisterUserServiceServer(grpcSrv, &UserServiceImpl{})
```

---

## Standalone gRPC server (CLI)

Run the gRPC server without the HTTP server:

```bash
kashvi grpc:serve
```

---

## Adding a custom interceptor

Edit `pkg/grpc/server.go` — add to `chainUnary(...)`:

```go
grpc.NewServer(
    grpc.UnaryInterceptor(
        chainUnary(
            recoveryInterceptor,
            loggingInterceptor,
            metricsInterceptor,
            myAuthInterceptor,  // ← add here
        ),
    ),
)
```

---

## Prometheus metrics

The gRPC metrics are available on the existing `/metrics` endpoint alongside HTTP metrics:

```
grpc_server_handled_total{grpc_method="/grpc.health.v1.Health/Check", grpc_code="OK"} 7
grpc_server_handling_seconds_bucket{grpc_method="...", le="0.01"} 7
```


---

# MongoDB Log Storage

Kashvi can mirror all application logs to **MongoDB** in addition to stdout. The integration is:

- **Async** — writes never block the request path
- **Batched** — up to 50 documents per `InsertMany`
- **Graceful** — remaining records are flushed before the server exits
- **Optional** — leave `MONGO_URI` blank to stay stdout-only (zero overhead)

---

## Configuration

```ini
# .env
MONGO_URI=mongodb://localhost:27017   # required to enable; leave blank to disable
MONGO_LOG_DB=kashvi_logs              # default: kashvi_logs
MONGO_LOG_COLLECTION=app_logs         # default: app_logs
```

With a MongoDB Atlas cluster:

```ini
MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net/?retryWrites=true
```

---

## Document shape

Each log record in MongoDB:

```json
{
  "time":       "2026-02-25T12:00:00Z",
  "level":      "INFO",
  "msg":        "user registered",
  "request_id": "a1b2c3d4",
  "attrs": {
    "email": "user@example.com",
    "plan":  "pro"
  }
}
```

A `{time: -1}` index is created on startup for efficient querying.

---

## Querying logs

```js
// mongosh — last 100 errors
db.app_logs.find({ level: "ERROR" }).sort({ time: -1 }).limit(100)

// All logs from a specific request
db.app_logs.find({ request_id: "a1b2c3d4" })

// Logs from the last hour
db.app_logs.find({ time: { $gt: new Date(Date.now() - 3600_000) } })
```

---

## TTL (auto-delete old logs)

Add a TTL index in MongoDB to keep only N days of logs:

```js
db.app_logs.createIndex(
  { time: 1 },
  { expireAfterSeconds: 30 * 24 * 3600 }  // 30 days
)
```

---

## Graceful flush on shutdown

`logger.CloseMongoHandler()` is called automatically during `kashvi run` shutdown.
If you start the server manually, call it yourself:

```go
defer logger.CloseMongoHandler()
```

---

## Internal design

| Detail | Value |
|--------|-------|
| Channel buffer | 4096 records |
| Batch size | 50 documents per InsertMany |
| Flush ticker | Every 2 seconds |
| On queue full | Record silently dropped — logging never blocks |
| Connection pool | Max 10 MongoDB connections |
| Connect timeout | 5 seconds (falls back to stdout if unreachable) |

If MongoDB is unreachable at startup, Kashvi logs a warning to stdout and continues without MongoDB — it never fails to start.


---

# TestKit — JSON-Scenario-Driven API Testing

`pkg/testkit` lets you write REST API integration tests **entirely in JSON**. One JSON file = one test case. No repeated Go boilerplate.

It is powered by [testify](https://github.com/stretchr/testify) — `testify/assert` for assertions and `testify/mock` for mocking side-effects.

---

## Concept

```
testdata/
  create_user.json          ← scenario (what to do & assert)
  create_user_req.json      ← request body
  create_user_res.json      ← expected response body
  health_check.json         ← another scenario
```

One Go test function runs all of them:

```go
func TestAPI(t *testing.T) {
    handler := kernel.NewHTTPKernel().Handler()
    testkit.RunDir(t, handler, "testdata")
}
```

## Data-Driven Test Suites

To execute multiple APIs spanning different isolated environments and handler overrides seamlessly instead of loading individual directory targets, `RunSuite` provides a Master Configuration approach driven entirely through JSON.

```go
func TestSuiteRun(t *testing.T) {
	// A map translating string identifiers in your config map into live Handler pointers
	handlers := map[string]http.HandlerFunc{
		"HandlerShipmentTracking":  api.ShipmentTrackingController,
		"HandlerBillingProcessing": api.BillingController,
	}

	// Loads and executes the testing Master Config definition JSON 
	testkit.RunSuite(t, "testdata/test_scenarios_master.json", handlers)
}
```

### Master Configuration Schema

The file supplied (`test_scenarios_master.json`) defines arrayed routes linking their respective URLs to handler implementations and target scenario specifications.

```json
[
  {
    "serviceName": "ShipmentTracking",
    "httpMethodType": "POST",
    "servicePath": "/api/v1/shipments/track",
    "filePath": "testdata/shipments",
    "scenariosFileName": "shipment_tracking_scenarios.json",
    "handlerName": "HandlerShipmentTracking"
  }
]
```

When parsed, TestKit loops across the specifications, injecting the proper URL and handler data onto dynamic mock routes to safely simulate full end-to-end framework testing directly mapped to isolated logic scenarios.

---

## Scenario JSON schema

```json
{
  "name":             "Create User",
  "description":      "POST /api/v1/users returns 201",
  "requestMethod":    "POST",
  "requestUrl":       "/api/v1/users",
  "requestFileName":  "create_user_req.json",
  "responseFileName": "create_user_res.json",
  "expectedCode":     201,
  "isMockRequired":   true,
  "isDbMocked":       false,
  "headers": {
    "Authorization": "Bearer test-token"
  },
  "netUtilMockStep": [
    {
      "method":    "httprequest",
      "isMock":    true,
      "matchUrl":  "https://verify.external.com/",
      "returnData": { "statusCode": 200, "body": "eyJ2ZXJpZmllZCI6dHJ1ZX0=" }
    },
    {
      "method":    "sendmail",
      "isMock":    true,
      "returnData": { "body": "" }
    }
  ]
}
```

### Field reference

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | **Required.** Test name (shown in `go test -v` output) |
| `description` | string | Human-readable description |
| `requestMethod` | string | HTTP method. Default: `GET` |
| `requestUrl` | string | **Required.** URL path to call (e.g. `/api/v1/users`) |
| `requestFileName` | string | Path to request body JSON file (relative to scenario dir) |
| `responseFileName` | string | Path to expected response JSON file (relative to scenario dir) |
| `expectedCode` | int | **Required.** Expected HTTP status code |
| `isMockRequired` | bool | If `true`, any un-mocked outgoing call fails the test |
| `isDbMocked` | bool | Informational flag — reserved for DB mock wiring |
| `headers` | object | Extra request headers (e.g. auth tokens) |
| `netUtilMockStep` | array | List of mock steps (see below) |

---

## Mock steps

### HTTP request mock (`method: "httprequest"`)

Intercepts outgoing calls made via `pkg/http`. Matched by URL **prefix**.

```json
{
  "method":   "httprequest",
  "isMock":   true,
  "matchUrl": "https://api.stripe.com/",
  "returnData": {
    "statusCode": 200,
    "body": "eyJpZCI6ImNoXzEyMyJ9"
  }
}
```

- `matchUrl` — prefix match. Empty string matches **any** URL.
- `returnData.body` — **base64-encoded** response body.
- `returnData.statusCode` — defaults to `200`.

### Function mock (`method: "sendmail"` / `"sms"` / `"notification"`)

Intercepts non-HTTP side-effects. Built-in methods:

| Method | Intercepts |
|--------|-----------|
| `sendmail` | `pkg/mail` sends |
| `sms` | SMS/notification sends |
| `notification` | Push notification sends |

```json
{ "method": "sendmail", "isMock": true, "returnData": { "body": "" } }
```

### Custom function mock

Register your own mocker once in a test init:

```go
func init() {
    testkit.RegisterMocker("payments", testkit.NewFuncMocker("payments"))
}
```

Then use in JSON: `"method": "payments"`.

---

## Base64 encoding the body

```bash
# Encode: {"verified":true}
echo -n '{"verified":true}' | base64
# → eyJ2ZXJpZmllZCI6dHJ1ZX0=
```

---

## Runner API

```go
// Run a single scenario
testkit.Run(t, handler, "testdata/create_user.json")

// Run all *.json files in a directory as subtests
testkit.RunDir(t, handler, "testdata")

// Run all array-based scenarios defined in a Master Configuration array via dynamically mapped handlers
testkit.RunSuite(t, "testdata/test_scenarios.json", handlersMap)
```

**Lifecycle per scenario:**
1. Load scenario JSON
2. Read request body from `requestFileName`
3. Install HTTP mock transport (`MockTransport`)
4. Activate function mocks (`sendmail`, `sms`, …)
5. Fire request against handler via `httptest`
6. Assert HTTP status code
7. JSON deep-diff actual vs expected response
8. Verify all `isMock: true` steps were called
9. Reset all mocks

---

## Advanced: testify mock expectations

Access the underlying `testify/mock.Mock` for custom assertions:

```go
func TestCreateUser(t *testing.T) {
    // Override the sendmail mocker
    mailer := testkit.NewFuncMocker("sendmail")
    mailer.Mock().On("Intercept", mock.Anything).Return(nil)
    testkit.RegisterMocker("sendmail", mailer)

    testkit.Run(t, handler, "testdata/create_user.json")

    // Assert it was called exactly once
    mailer.Mock().AssertNumberOfCalls(t, "Intercept", 1)
}
```

---

## Assertions

| Assertion | Behaviour |
|-----------|-----------|
| Status code | `testify/assert.Equal` — prints expected vs actual |
| Response body | JSON normalised (key order / whitespace ignored), `testify/assert.Equal` |
| HTTP mocks called | Fails per un-triggered `isMock: true` httprequest step |
| Func mocks called | Fails per un-triggered `isMock: true` func step |

---

## Debugging

Print a scenario summary to stdout:

```go
s, _ := testkit.LoadScenario("testdata/create_user.json")
testkit.DumpScenario(s)
```

Output:
```
Scenario: Create User
  POST /api/v1/users → 201
  requestFile:  create_user_req.json
  responseFile: create_user_res.json
  isMockRequired: true  isDbMocked: false
  mockStep[0]: method=httprequest  isMock=true  matchUrl="https://verify.external.com/"
  mockStep[1]: method=sendmail     isMock=true  matchUrl=""
```


---

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


---

# Building a User CRUD API in Kashvi (5 Minutes)

Welcome to Kashvi! If you're a fresher or transitioning from PHP/Laravel, you'll feel right at home. We're going to build a fully functional `User` API in just a few minutes. 

No advanced features (like WebSockets or gRPC) here—just standard, clean RESTful architecture.

---

## 1. Create the Project
First, scaffold a fresh project. This creates a ready-to-use folder structure for you.

```bash
kashvi new my-api
cd my-api
```

*(This command generates your `main.go`, `app/` folder for logic, `database/` for schemas, and a `.env` file pre-configured for SQLite so you don't need any complex database setup yet).*

---

## 2. Generate the Resource
We need a Model (to represent the user), a Controller (to handle HTTP requests), a Migration (to create the database table), and a Seeder (to add dummy data).

Instead of creating these manually, Kashvi's CLI does it in one command:

```bash
kashvi make:resource User
```

**What this did:**
* `app/models/user.go`: Your data structure.
* `app/controllers/user_controller.go`: Where your API logic lives.
* `database/migrations/xxxx_create_users_table.go`: Instructions for creating the database table.
* `database/seeders/user_seeder.go`: A place to create fake users for testing.

---

## 3. Define the Database Table
Let's tell the database what a "User" looks like. Open the newly generated migration file inside the `database/migrations/` folder.

Add the `name` and `email` columns:

```go
// database/migrations/xxxx_create_users_table.go

func (m *Migration) Up() {
    table := m.CreateTable("users")
    table.String("name").NotNull()
    table.String("email").Unique().NotNull()
}
```

Now, run the migration to actually create the table in your SQLite database:
```bash
kashvi migrate
```

---

## 4. Write the Controller Logic
Open `app/controllers/user_controller.go`. We'll write the logic for creating a new user (the `Store` method).

Kashvi has built-in JSON validation. If the user doesn't send a name, we automatically reject the request!

```go
// app/controllers/user_controller.go
package controllers

import (
    "github.com/shashiranjanraj/kashvi/pkg/ctx"
    "my-api/app/models"
    "my-api/database" // Assuming you export your db connection here
)

func (c *UserController) Store(ctx *ctx.Context) {
    // 1. Define what JSON we expect and add validation rules!
    var input struct {
        Name  string `json:"name" validate:"required,min=2"`
        Email string `json:"email" validate:"required,email"`
    }

    // 2. Bind and Validate. If it fails, Kashvi automatically sends a 422 Error to the client.
    if !ctx.BindJSON(&input) { 
        return 
    }

    // 3. Save to database
    user := models.User{Name: input.Name, Email: input.Email}
    database.DB.Create(&user)
    
    // 4. Send success response (201 Created)
    ctx.Created(user)
}
```

---

## 5. Register the Route
We need to map a URL (like `POST /api/users`) to the controller we just wrote. 

Open `app/routes/api.go` and add this inside the `RegisterRoutes` function:

```go
// app/routes/api.go
import "my-api/app/controllers"

func RegisterAPI(r *router.Router) {
    api := r.Group("/api")
    
    // Initialize our controller
    userCtrl := controllers.NewUserController()

    // Map the POST request to the Store function
    api.Post("/users", "users.store", ctx.Wrap(userCtrl.Store))
}
```

---

## 6. Run the Server
You're done coding! Let's start the server.

```bash
kashvi serve
```
*You should see a message saying your server is running on port 8080.*

---

## 7. Test It Out
Open your terminal and run this `curl` command (or use Postman) to create a user:

```bash
curl -X POST http://localhost:8080/api/users \
     -H "Content-Type: application/json" \
     -d '{"name": "Rahul", "email": "rahul@example.com"}'
```

**Success Response:**
```json
{
  "name": "Rahul",
  "email": "rahul@example.com"
}
```

**What if we forget the email? (Validation Test)**
```bash
curl -X POST http://localhost:8080/api/users \
     -H "Content-Type: application/json" \
     -d '{"name": "Rahul"}'
```

**Error Response:**
```json
{
  "error": "validation failed",
  "details": {
    "email": "email is required"
  }
}
```

### 🎉 Congratulations!
You just built a production-grade, validated Go API without the confusing boilerplate. Welcome to Kashvi!


---

# CRUD Walkthrough

This guide covers a full Create/Read/Update/Delete flow using Kashvi generators and runtime commands.

## 1. Scaffold a resource

Generate all CRUD files:

```bash
kashvi make:crud Post
```

You can also use flags:

```bash
kashvi make:crud Post --authorize --cache
```

- `--authorize`: route snippet printed by CLI includes an auth middleware placeholder.
- `--cache`: generated controller includes cache TODO placeholders in `Index` and `Show`.

## 2. Generated files

For `Post`, Kashvi creates:

- `app/models/post.go`
- `app/controllers/post_controller.go`
- `app/services/post_service.go`
- `database/migrations/<timestamp>_create_posts_table.go`
- `database/seeders/post_seeder.go`
- `testdata/post_scenarios.json`

## 3. Register routes

The generator prints lines to paste into your route setup. Typical wiring:

```go
package routes

import (
	"github.com/your-org/your-app/app/controllers"
	appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

func RegisterAPI(r *router.Router) {
	api := r.Group("/api")

	ctrl := controllers.NewPostController()
	api.Get("/posts", "posts.index", appctx.Wrap(ctrl.Index))
	api.Post("/posts", "posts.store", appctx.Wrap(ctrl.Store))
	api.Get("/posts/{id}", "posts.show", appctx.Wrap(ctrl.Show))
	api.Put("/posts/{id}", "posts.update", appctx.Wrap(ctrl.Update))
	api.Delete("/posts/{id}", "posts.destroy", appctx.Wrap(ctrl.Destroy))
}
```

Then ensure the route function is attached in `main.go`:

```go
app.New().
	Routes(routes.RegisterAPI).
	Run()
```

## 4. Implement migration

The generated migration registers automatically, but `Up` and `Down` are placeholders. Fill them:

```go
func (m *M_20260301010101_create_posts_table) Up(db *gorm.DB) error {
	type Post struct {
		gorm.Model
		Title string
		Body  string
	}
	return db.AutoMigrate(&Post{})
}

func (m *M_20260301010101_create_posts_table) Down(db *gorm.DB) error {
	return db.Migrator().DropTable("posts")
}
```

Run migration:

```bash
kashvi migrate
```

Inspect migration state:

```bash
kashvi migrate:status
```

## 5. Run and test endpoints

Start server:

```bash
kashvi serve
```

### Create

```bash
curl -X POST http://localhost:8080/api/posts \
  -H 'Content-Type: application/json' \
  -d '{}'
```

### List

```bash
curl http://localhost:8080/api/posts
```

### Show

```bash
curl http://localhost:8080/api/posts/1
```

### Update

```bash
curl -X PUT http://localhost:8080/api/posts/1 \
  -H 'Content-Type: application/json' \
  -d '{}'
```

### Delete

```bash
curl -X DELETE http://localhost:8080/api/posts/1 -i
```

The generated `Destroy` handler returns `204 No Content`.

## 6. Use generated test scenarios

`kashvi make:crud` creates `testdata/post_scenarios.json`. You can feed these scenarios into your `pkg/testkit` test runner and keep them as executable API documentation.

If you used `--authorize`, scenario entries include an `Authorization` header placeholder (`Bearer dummy-jwt-token`).

## 7. Next improvements

After initial scaffold, common upgrades are:

1. Replace empty request structs in controller methods with typed DTOs + validation tags.
2. Move DB logic into `app/services/post_service.go` and keep controllers thin.
3. Add middleware (`Auth`, rate-limit, role checks) per route group.
4. Add pagination in `Index` using `pkg/orm` pagination helpers.
5. Add queue jobs for side effects (notifications, emails, analytics writes).


---

# Routing

Routes are registered in `app/routes/api.go`.

---

## Basic Routes

```go
func RegisterAPI(r *router.Router) {
    r.Get("/ping", "ping", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    })

    r.Post("/users",      "users.store",   handler)
    r.Put("/users/{id}",  "users.update",  handler)
    r.Patch("/users/{id}","users.patch",   handler)
    r.Delete("/users/{id}","users.destroy",handler)
}
```

---

## Using `ctx.Context` (recommended)

```go
import appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"

r.Get("/users/{id}", "users.show", appctx.Wrap(func(c *appctx.Context) {
    id := c.Param("id")
    c.Success(map[string]any{"id": id})
}))
```

---

## Route Groups

Groups let you share a path prefix and/or middleware across multiple routes:

```go
// All routes under /api with rate limiting
api := r.Group("/api", middleware.RateLimit(120, time.Minute))

api.Get("/users", "users.index", appctx.Wrap(ctrl.Index))
api.Post("/users", "users.store", appctx.Wrap(ctrl.Store))

// Nested group: /api/admin with auth guard
admin := api.Group("/admin", middleware.AuthMiddleware, middleware.RequireRole("admin"))
admin.Get("/stats", "admin.stats", appctx.Wrap(adminCtrl.Stats))
```

---

## URL Parameters

```go
// Define: /articles/{slug}/comments/{id}
r.Get("/articles/{slug}/comments/{id}", "comments.show", appctx.Wrap(func(c *appctx.Context) {
    slug := c.Param("slug")
    id   := c.Param("id")
    // ...
}))
```

---

## Named Routes & URL Generation

Every route takes a name as the second argument. Names let you generate URLs safely:

```go
// Registration
r.Get("/users/{id}", "users.show", handler)

// URL generation (anywhere in your code)
url, err := myRouter.URL("users.show", map[string]string{"id": "42"})
// url = "/users/42"
```

---

## Mounting Third-Party Handlers

```go
// Prometheus metrics (already wired by framework)
r.HandleFunc("/metrics", metrics.Handler())

// Any http.Handler
r.Mount("/storage", http.FileServer(http.Dir("storage")))
```

---

## Listing All Routes

```bash
kashvi route:list
```

Output:
```
METHOD   PATH                    NAME
------   ----                    ----
DELETE   /api/users/{id}         users.destroy
GET      /api/health             health
GET      /api/users              users.index
GET      /api/users/{id}         users.show
POST     /api/login              auth.login
POST     /api/register           auth.register
POST     /api/users              users.store
PUT      /api/users/{id}         users.update
```

---

## Per-Route Middleware

Middleware can be applied to individual routes as variadic arguments:

```go
api.Get("/admin/report", "admin.report",
    appctx.Wrap(adminCtrl.Report),
    middleware.AuthMiddleware,
    middleware.RequireRole("admin"),
)
```


---

# Context API

`pkg/ctx` provides a `gin.Context`-inspired request context for Kashvi handlers.
Instead of `(http.ResponseWriter, *http.Request)`, your handler receives a single `*ctx.Context`.

---

## Handler Signature

```go
import appctx "github.com/shashiranjanraj/kashvi/pkg/ctx"

func MyHandler(c *appctx.Context) {
    // use c for everything
}

// Register with ctx.Wrap():
r.Get("/path", "name", appctx.Wrap(MyHandler))
```

---

## Reading the Request

### URL Parameters
```go
id   := c.Param("id")     // /users/{id}
slug := c.Param("slug")   // /posts/{slug}
```

### Query String
```go
page    := c.Query("page")                  // "" if absent
sort    := c.DefaultQuery("sort", "created_at")
```

### Request Body (JSON)
```go
// Automatic — decodes + validates, sends 422 on failure
var input struct {
    Name  string `json:"name"  validate:"required,min=2"`
    Email string `json:"email" validate:"required,email"`
}
if !c.BindJSON(&input) {
    return  // response already sent
}

// Manual — returns errors to handle yourself
errs, err := c.ShouldBindJSON(&input)
if err != nil { /* bad JSON */ }
if len(errs) > 0 { /* validation errors */}
```

### Form Data
```go
name := c.PostForm("name")
```

### Headers & Cookies
```go
token  := c.Header("Authorization")
accept := c.Header("Accept")

val, err := c.Cookie("session_id")
```

### Metadata
```go
method := c.Method()     // "GET"
path   := c.Path()       // "/api/users/42"
full   := c.FullPath()   // "GET /api/users/42"
ip     := c.ClientIP()   // respects X-Forwarded-For
isXHR  := c.IsXHR()      // X-Requested-With: XMLHttpRequest
ctx    := c.Context()    // underlying context.Context
```

### Raw Body
```go
bytes, err := c.Body()
```

---

## Sending Responses

### JSON
```go
c.JSON(200, map[string]any{"key": "value"})

// Pre-wrapped envelopes:
c.Success(data)         // 200 {"status":200,"data":{...}}
c.Created(data)         // 201 {"status":201,"data":{...}}
c.Error(400, "Bad req") // 4xx {"status":400,"message":"..."}
c.ValidationError(errs) // 422 {"status":422,"message":"Validation failed","errors":{...}}

// Shortcuts:
c.Unauthorized()        // 401
c.Unauthorized("Token expired")
c.Forbidden()           // 403
c.NotFound()            // 404
c.NotFound("Post not found")
```

### Other response types
```go
c.String(200, "Hello, %s!", name)
c.Status(204)               // status only, no body
c.Redirect(302, "/login")
c.File("/path/to/file.pdf")
```

### Headers & Cookies
```go
c.SetHeader("X-Request-Id", "abc123")
c.SetCookie("token", value, 3600, "/", "", true, true)
```

---

## Per-Request Store

Pass values between middleware and handlers via the request-scoped store:

```go
// In middleware (e.g. AuthMiddleware):
c.Set("user_id", claims.UserID)
c.Set("role", claims.Role)

// In handler:
userID := c.GetUint("user_id")
role   := c.GetString("role")

// Generic (any type):
val, ok := c.Get("key")
val      = c.MustGet("key") // panics if missing
```

---

## Abort

```go
func AdminOnly(c *appctx.Context) {
    if c.GetString("role") != "admin" {
        c.Abort(403, "Admin access required")
        return
    }
    // continue
}
```

---

## Validate Without Binding

```go
type Input struct {
    Age int `json:"age" validate:"required,min=18"`
}
var input Input
// ... populate input ...
errs := c.Validate(&input)
if len(errs) > 0 {
    c.ValidationError(errs)
    return
}
```

---

## Pool Efficiency

`pkg/ctx` uses `sync.Pool` internally — `Context` objects are **recycled between requests**, resulting in zero allocations per request.


---

# Validation

Kashvi's validation engine lives in `pkg/validate`. It has **zero external dependencies** and supports 28 rules via struct tags.

---

## Struct Tags

Add a `validate` tag to any field:

```go
type RegisterInput struct {
    Name            string  `json:"name"             validate:"required,min=2,max=100"`
    Email           string  `json:"email"            validate:"required,email"`
    Age             int     `json:"age"              validate:"required,min=18,max=120"`
    Role            string  `json:"role"             validate:"in=admin,user,editor"`
    Password        string  `json:"password"         validate:"required,min=8"`
    PasswordConfirm string  `json:"password_confirm" validate:"confirmed=password"`
    Website         *string `json:"website"          validate:"nullable,url"`
}
```

---

## All Validation Rules

| Rule | Example Tag | Description |
|---|---|---|
| `required` | `validate:"required"` | Field must be non-zero |
| `email` | `validate:"email"` | Valid email address |
| `min` | `validate:"min=3"` | String min length / numeric min value |
| `max` | `validate:"max=100"` | String max length / numeric max value |
| `between` | `validate:"between=1,10"` | Numeric between two values (inclusive) |
| `in` | `validate:"in=a,b,c"` | Value must be one of the listed options |
| `not_in` | `validate:"not_in=bad,worse"` | Value must NOT be in the list |
| `confirmed` | `validate:"confirmed=password"` | Must match another field's value |
| `url` | `validate:"url"` | Valid HTTP/HTTPS URL |
| `alpha` | `validate:"alpha"` | Letters only |
| `alpha_num` | `validate:"alpha_num"` | Letters and numbers only |
| `alpha_dash` | `validate:"alpha_dash"` | Letters, numbers, `-`, `_` |
| `numeric` | `validate:"numeric"` | Any number (int or float) |
| `integer` | `validate:"integer"` | Must be an integer |
| `boolean` | `validate:"boolean"` | true or false |
| `ip` | `validate:"ip"` | Valid IPv4 or IPv6 address |
| `uuid` | `validate:"uuid"` | Valid UUID |
| `date` | `validate:"date"` | Valid date in `YYYY-MM-DD` format |
| `date_format` | `validate:"date_format=2006-01-02"` | Custom Go time layout |
| `starts_with` | `validate:"starts_with=https"` | String prefix check |
| `ends_with` | `validate:"ends_with=.go"` | String suffix check |
| `contains` | `validate:"contains=@"` | Substring check |
| `regex` | `validate:"regex=^[A-Z]+"` | Custom regex pattern |
| `json` | `validate:"json"` | Valid JSON string |
| `len` | `validate:"len=6"` | Exact string length |
| `same` | `validate:"same=other_field"` | Alias for `confirmed` |
| `different` | `validate:"different=old_password"` | Must differ from field |
| `nullable` | `validate:"nullable,email"` | Skip all other rules if the field is nil/zero |

---

## Using Validation Directly

### In a handler with `BindJSON`:
```go
func (ctrl *UserController) Register(c *appctx.Context) {
    var input RegisterInput
    if !c.BindJSON(&input) {
        return // 422 already sent
    }
    // input is valid here
}
```

### Manual validation:
```go
import "github.com/shashiranjanraj/kashvi/pkg/validate"

errs := validate.Struct(&input)
if validate.HasErrors(errs) {
    // errs = map[string]string{"email": "The email field must be a valid email address."}
}
```

---

## Error Messages

Errors are returned as `map[string]string` where the key is the JSON field name:

```json
{
  "status": 422,
  "message": "Validation failed",
  "errors": {
    "email": "The email field must be a valid email address.",
    "password": "The password field must be at least 8 characters.",
    "password_confirm": "The password_confirm field must match password."
  }
}
```

---

## Nullable Fields

Use `nullable` to skip all other rules when the field is empty/nil:

```go
type UpdateInput struct {
    // These are all optional — only validated if provided
    Bio     *string `json:"bio"     validate:"nullable,max=500"`
    Website *string `json:"website" validate:"nullable,url"`
    Age     *int    `json:"age"     validate:"nullable,min=18"`
}
```

---

## Combining Rules

Rules are comma-separated and evaluated in order. All failures are collected (not short-circuit):

```go
validate:"required,min=8,max=64,alpha_num"
```


---

# Queue & Jobs

Kashvi's queue system (`pkg/queue`) supports background job processing with retry, backoff, and persistent failure tracking.

---

## Defining a Job

```go
// app/jobs/welcome_email_job.go
package jobs

type WelcomeEmailJob struct {
    UserID uint   `json:"user_id"`
    Email  string `json:"email"`
}

func (j WelcomeEmailJob) Handle() error {
    // send email...
    return mailer.Send(j.Email, "Welcome!", "welcome.html")
}
```

Register the job type at boot (so it can be deserialized):

```go
// In main.go init() or a jobs/register.go file:
queue.Register("jobs.WelcomeEmailJob", func() queue.Job {
    return &jobs.WelcomeEmailJob{}
})
```

---

## Dispatching Jobs

```go
import "github.com/shashiranjanraj/kashvi/pkg/queue"

// Immediate
queue.Dispatch(jobs.WelcomeEmailJob{UserID: user.ID, Email: user.Email})

// After a delay (5 minutes)
queue.DispatchAfter(jobs.WelcomeEmailJob{UserID: user.ID, Email: user.Email}, 5*time.Minute)
```

---

## Queue Drivers

### In-Memory (default — dev only)

Jobs are lost on restart. Good for development and testing.

```go
// Default — no configuration needed
queue.Dispatch(myJob)
```

### Redis Driver (production)

Jobs survive restarts. Delayed jobs use Redis sorted sets.

```go
// In server.go or a boot function, after cache.Connect():
import (
    "github.com/shashiranjanraj/kashvi/pkg/cache"
    "github.com/shashiranjanraj/kashvi/pkg/queue"
)

queue.SetDriver(queue.NewRedisDriver(cache.RDB))
```

Redis keys used:
- `kashvi:queue:jobs` — immediate job list (LPUSH/BRPOP)
- `kashvi:queue:delayed` — delayed job sorted set (score = Unix timestamp)

---

## Starting Workers

```bash
# From CLI (production)
kashvi queue:work --workers=5

# Or programmatically:
queue.StartWorkers(ctx, 5)
```

---

## Retry & Backoff

Failed jobs are automatically retried with linear backoff:
- Attempt 1 → wait 1s → Attempt 2 → wait 2s → Attempt 3

```go
// Change retry limit (default: 3)
queue.SetMaxRetry(5)
```

---

## Failed Jobs

After all retries are exhausted, the job is recorded in:
1. **In-memory** — available via `queue.FailedJobs()`
2. **Database** — `kashvi_failed_jobs` table (if `queue.UseDB()` is called)

The database persistence is wired automatically at server boot.

**Table structure:**

| Column | Type | Description |
|---|---|---|
| `id` | uint | Auto-increment PK |
| `job_type` | string | Go type name |
| `payload` | text | JSON-encoded job data |
| `error` | text | Last error message |
| `attempts` | int | Number of attempts made |
| `failed_at` | timestamp | When it failed |

**Querying failures:**

```go
// In memory
failed := queue.FailedJobs()
for _, f := range failed {
    fmt.Printf("%T failed after %d attempts: %v\n", f.Job, f.Attempts, f.Err)
}

// From DB
var records []queue.FailedJobRecord
database.DB.Order("failed_at desc").Find(&records)
```

---

## Full Example — Order Processing

```go
type ProcessOrderJob struct {
    OrderID uint `json:"order_id"`
}

func (j ProcessOrderJob) Handle() error {
    var order models.Order
    if err := database.DB.First(&order, j.OrderID).Error; err != nil {
        return err // will be retried
    }
    // charge card, update inventory, send confirmation...
    return nil
}

// In your controller:
func (c *OrderController) Store(ctx *appctx.Context) {
    // ... create order ...
    queue.Dispatch(ProcessOrderJob{OrderID: order.ID})
    ctx.Created(order)
}
```


---

# Worker Pool

`pkg/workerpool` provides a **bounded goroutine pool** that limits concurrent goroutine creation under high load. Use it for CPU-intensive or I/O-heavy tasks that should not run in unbounded goroutines.

---

## Why use a pool?

| Approach | Problem |
|----------|---------|
| `go doWork()` for every request | Goroutines spike unboundedly under load — OOM risk |
| Worker pool | Hard ceiling on concurrency — predictable memory |

---

## Configuration

```ini
# .env
WORKER_POOL_SIZE=50   # default: 50
```

---

## Basic usage

```go
import "github.com/shashiranjanraj/kashvi/pkg/workerpool"

// Create a pool (use config.WorkerPoolSize() for env-driven size)
pool := workerpool.New(config.WorkerPoolSize())
defer pool.Shutdown()

// Non-blocking submit
err := pool.Submit(func() {
    processImage(imageData)
})
if errors.Is(err, workerpool.ErrPoolFull) {
    // Pool is busy — return 429, push to queue, etc.
    c.JSON(http.StatusTooManyRequests, map[string]string{"error": "server busy"})
    return
}
```

---

## Blocking submit

When you want to wait until a slot is available:

```go
err := pool.SubmitWait(func() {
    sendReportEmail(userID)
})
if errors.Is(err, workerpool.ErrPoolClosed) {
    // Pool was shut down
}
```

---

## Shutdown

`Shutdown()` stops accepting new tasks, waits for all in-flight tasks to complete, then releases all worker goroutines. Safe to call multiple times.

```go
pool.Shutdown()
```

---

## Error reference

| Error | When |
|-------|------|
| `workerpool.ErrPoolFull` | All workers are busy and the queue buffer is full |
| `workerpool.ErrPoolClosed` | `Shutdown()` has been called |

---

## Panic safety

Workers recover from panics automatically — a bad task never kills the pool or unexpectedly terminates a goroutine. The next task runs as normal.

---

## Sizing guide

| Use case | Recommended size |
|----------|-----------------|
| Image processing | `runtime.NumCPU()` |
| Network I/O (external APIs) | 50–200 |
| DB queries | 20–50 (limited by DB connection pool) |
| Mixed workloads | `WORKER_POOL_SIZE=50` (default) |

---

## Buffer size

The internal task queue buffer is `2 × size`. This absorbs short bursts without returning `ErrPoolFull`. For example, a pool of 50 workers can buffer 100 pending tasks before backpressure kicks in.

---

## Integration with HTTP handlers

A good pattern: create one shared pool at app startup and use it across handlers.

```go
// internal/kernel/http.go
var Pool = workerpool.New(config.WorkerPoolSize())

// In a handler
func GenerateReport(c *ctx.Context) {
    err := kernel.Pool.Submit(func() {
        report := buildReport(c.ParamInt("id"))
        cache.Set("report:"+id, report, time.Hour)
    })
    if errors.Is(err, workerpool.ErrPoolFull) {
        c.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
        return
    }
    c.JSON(http.StatusAccepted, map[string]string{"status": "processing"})
}
```


---

# Storage

`pkg/storage` provides a unified file-storage API inspired by Laravel's Storage facade.
Switch between local disk and S3-compatible storage with a single env variable.

---

## Configuration

```ini
STORAGE_DISK=local      # default driver: "local" or "s3"
```

---

## Using the Default Disk

```go
import "github.com/shashiranjanraj/kashvi/pkg/storage"

// Write
storage.Put("avatars/user-1.jpg", imageBytes)
storage.PutStream("uploads/file.pdf", r.Body)

// Read
data, err := storage.Get("avatars/user-1.jpg")
stream, err := storage.GetStream("uploads/file.pdf")
defer stream.Close()

// Metadata
exists  := storage.Exists("avatars/user-1.jpg")
missing := storage.Missing("avatars/user-1.jpg")
size, _ := storage.Size("avatars/user-1.jpg")
modTime, _ := storage.LastModified("avatars/user-1.jpg")

// Public URL
url := storage.URL("avatars/user-1.jpg")

// Delete
storage.Delete("avatars/user-1.jpg")

// Copy / Move
storage.Copy("tmp/upload.jpg", "images/final.jpg")
storage.Move("tmp/upload.jpg", "archive/old.jpg")

// Directories
files, _ := storage.Files("avatars")          // non-recursive
all, _   := storage.AllFiles("avatars")       // recursive
dirs, _  := storage.Directories("uploads")
storage.MakeDirectory("exports")
storage.DeleteDirectory("tmp")
```

---

## Using a Specific Disk

```go
// Use S3 explicitly
storage.Use("s3").Put("backups/db.sql.gz", data)

// Use local disk explicitly
storage.Use("local").Get("cache/data.json")
```

> Method name is `Use()` (not `Disk()`) to avoid conflict with the `Disk` interface type.

---

## File Upload Handler

```go
func (c *UploadController) Store(ctx *appctx.Context) {
    ctx.R.ParseMultipartForm(10 << 20) // 10MB max

    file, header, err := ctx.R.FormFile("file")
    if err != nil {
        ctx.Error(400, "No file uploaded")
        return
    }
    defer file.Close()

    path := fmt.Sprintf("uploads/%d_%s", time.Now().Unix(), header.Filename)
    if err := storage.PutStream(path, file); err != nil {
        ctx.Error(500, "Upload failed")
        return
    }

    ctx.Created(map[string]any{
        "path": path,
        "url":  storage.URL(path),
    })
}
```

---

## Local Disk

Files are stored relative to `STORAGE_LOCAL_ROOT` (default: `./storage`).

Public access: `GET /storage/{path}` is automatically mounted when `STORAGE_DISK=local`.

```ini
STORAGE_LOCAL_ROOT=storage
STORAGE_URL=http://localhost:8080/storage
```

---

## S3 / AWS

```ini
STORAGE_DISK=s3
S3_BUCKET=my-bucket
S3_REGION=us-east-1
S3_KEY=AKIAIOSFODNN7EXAMPLE
S3_SECRET=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
S3_URL=https://my-bucket.s3.us-east-1.amazonaws.com
```

---

## MinIO (self-hosted S3)

Run locally with Docker:

```bash
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"
```

```ini
STORAGE_DISK=s3
S3_BUCKET=my-bucket
S3_KEY=minioadmin
S3_SECRET=minioadmin
S3_ENDPOINT=http://localhost:9000
S3_REGION=us-east-1
```

Create the bucket at `http://localhost:9001` (MinIO console UI).

---

## Cloudflare R2 / DigitalOcean Spaces

Same as MinIO — just set `S3_ENDPOINT` to your provider's endpoint URL.

```ini
# Cloudflare R2
S3_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com

# DigitalOcean Spaces
S3_ENDPOINT=https://nyc3.digitaloceanspaces.com
```

---

## Custom Driver

Implement the `Disk` interface and register it:

```go
type MyDriver struct{}
func (d *MyDriver) Put(path string, content []byte) error { ... }
// ... implement all 16 Disk interface methods

// Register at boot:
storage.RegisterDisk("mydriver", &MyDriver{})

// Use:
storage.Use("mydriver").Put("file.txt", data)
```


---

# gRPC Server

Kashvi includes a production-ready gRPC server that runs **alongside** the HTTP server on a separate port. It ships with a health-check service, server reflection, and pre-wired Prometheus metrics.

---

## Configuration

```ini
# .env
GRPC_PORT=9090    # default: 9090
```

---

## What starts automatically

When you run `kashvi run`, **both** servers boot:

```
🚀 Kashvi HTTP  on :8080  [env: local]  [workers: 8]
🔌 Kashvi gRPC  on :9090
```

At shutdown (`Ctrl+C`), the gRPC server drains in-flight RPCs before exiting.

---

## Built-in interceptors (applied automatically)

| Order | Interceptor | What it does |
|-------|-------------|--------------|
| 1 | **Recovery** | Catches panics → returns `INTERNAL` status instead of crashing |
| 2 | **Logging** | Logs every RPC: `method`, `duration_ms`, `code` |
| 3 | **Prometheus** | `grpc_server_handled_total`, `grpc_server_handling_seconds` |

---

## Built-in services

### Health (grpc.health.v1.Health)

Always returns `SERVING`. Test with:

```bash
# brew install grpcurl
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
# → { "status": "SERVING" }
```

### Server Reflection

Enabled automatically — `grpcurl` works without proto files:

```bash
grpcurl -plaintext localhost:9090 list
# → grpc.health.v1.Health
```

---

## Registering your own service

```go
// pkg/grpc/server.go  — add after reflection.Register(srv)
mypb.RegisterUserServiceServer(srv, &UserServiceImpl{})
```

Or call `grpc.Start()` manually and register before the goroutine runs:

```go
grpcSrv, lis, _ := kashvigrpc.Start(config.GRPCPort())
mypb.RegisterUserServiceServer(grpcSrv, &UserServiceImpl{})
```

---

## Standalone gRPC server (CLI)

Run the gRPC server without the HTTP server:

```bash
kashvi grpc:serve
```

---

## Adding a custom interceptor

Edit `pkg/grpc/server.go` — add to `chainUnary(...)`:

```go
grpc.NewServer(
    grpc.UnaryInterceptor(
        chainUnary(
            recoveryInterceptor,
            loggingInterceptor,
            metricsInterceptor,
            myAuthInterceptor,  // ← add here
        ),
    ),
)
```

---

## Prometheus metrics

The gRPC metrics are available on the existing `/metrics` endpoint alongside HTTP metrics:

```
grpc_server_handled_total{grpc_method="/grpc.health.v1.Health/Check", grpc_code="OK"} 7
grpc_server_handling_seconds_bucket{grpc_method="...", le="0.01"} 7
```


---

# MongoDB Log Storage

Kashvi can mirror all application logs to **MongoDB** in addition to stdout. The integration is:

- **Async** — writes never block the request path
- **Batched** — up to 50 documents per `InsertMany`
- **Graceful** — remaining records are flushed before the server exits
- **Optional** — leave `MONGO_URI` blank to stay stdout-only (zero overhead)

---

## Configuration

```ini
# .env
MONGO_URI=mongodb://localhost:27017   # required to enable; leave blank to disable
MONGO_LOG_DB=kashvi_logs              # default: kashvi_logs
MONGO_LOG_COLLECTION=app_logs         # default: app_logs
```

With a MongoDB Atlas cluster:

```ini
MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net/?retryWrites=true
```

---

## Document shape

Each log record in MongoDB:

```json
{
  "time":       "2026-02-25T12:00:00Z",
  "level":      "INFO",
  "msg":        "user registered",
  "request_id": "a1b2c3d4",
  "attrs": {
    "email": "user@example.com",
    "plan":  "pro"
  }
}
```

A `{time: -1}` index is created on startup for efficient querying.

---

## Querying logs

```js
// mongosh — last 100 errors
db.app_logs.find({ level: "ERROR" }).sort({ time: -1 }).limit(100)

// All logs from a specific request
db.app_logs.find({ request_id: "a1b2c3d4" })

// Logs from the last hour
db.app_logs.find({ time: { $gt: new Date(Date.now() - 3600_000) } })
```

---

## TTL (auto-delete old logs)

Add a TTL index in MongoDB to keep only N days of logs:

```js
db.app_logs.createIndex(
  { time: 1 },
  { expireAfterSeconds: 30 * 24 * 3600 }  // 30 days
)
```

---

## Graceful flush on shutdown

`logger.CloseMongoHandler()` is called automatically during `kashvi run` shutdown.
If you start the server manually, call it yourself:

```go
defer logger.CloseMongoHandler()
```

---

## Internal design

| Detail | Value |
|--------|-------|
| Channel buffer | 4096 records |
| Batch size | 50 documents per InsertMany |
| Flush ticker | Every 2 seconds |
| On queue full | Record silently dropped — logging never blocks |
| Connection pool | Max 10 MongoDB connections |
| Connect timeout | 5 seconds (falls back to stdout if unreachable) |

If MongoDB is unreachable at startup, Kashvi logs a warning to stdout and continues without MongoDB — it never fails to start.


---

