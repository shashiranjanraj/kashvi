# Kashvi Framework Documentation

Find topics quickly: **[Documentation map](#documentation-map)** · **[Docs index](docs/README.md)** · **[Request flow](docs/REQUEST_FLOW.md)** · **[Error handling](docs/ERROR_HANDLING.md)** · Search this page with your editor (`Cmd+F` / `Ctrl+F`).

## Documentation map

| Goal | Where to read |
|------|----------------|
| Install CLI, new project, first run | [Installation](#installation) · [Quick start](#quick-start) |
| Middleware order, validation placement, handler → DB | [`docs/REQUEST_FLOW.md`](docs/REQUEST_FLOW.md) |
| Full CRUD (model → DTO → migration → repo → tests) | [Complete CRUD walkthrough](#complete-crud-walkthrough-model--migration--repository--service--controller--validation--auth--seed--test) |
| App logs and SQL (`LOG_LEVEL`, `DB_LOG_MODE`) | [Logging](#logging-app-and-database) |
| `errors.Is` / `errors.As`, `apperror`, `router.Wrap` | [`docs/ERROR_HANDLING.md`](docs/ERROR_HANDLING.md) |
| CLI (`migrate`, `make:resource`, …) | [CLI commands](#cli-commands) |
| Browse by keyword and package | [`docs/README.md`](docs/README.md) |

**Popular links:** [Routing](#routing) · [Context](#context) · [Authentication](#authentication) · [Middleware](#middleware) · [Database](#database) · [Validation](#validation) · [Queues](#queues) · [Testing](#testing) · [SOLID](docs/SOLID_PRINCIPLES.md) · [Benchmarks](docs/BENCHMARK.md)

---

## Overview

Kashvi is a Laravel-inspired Go web framework designed for rapid application development. It provides a clean, expressive API with powerful features like ORM, migrations, authentication, caching, queues, and more. Built on top of proven libraries like GORM, Chi router, and Redis, Kashvi helps you build scalable web applications and APIs quickly.

*Made with ❤️ by an Indian developer*

### Key Features

- **MVC Architecture**: Controllers (HTTP), services (business logic), repositories (data), models, and DTOs (request/response shapes)
- **Database ORM**: GORM-powered query builder with support for MySQL, PostgreSQL, SQLite, and SQL Server
- **Migrations**: Database schema versioning and management
- **Authentication**: JWT-based auth with middleware
- **Caching**: Redis-backed caching system
- **Queues**: Background job processing with Redis
- **WebSockets**: Real-time communication support
- **gRPC Support**: High-performance RPC services
- **Testing**: Built-in test kit with JSON scenario files
- **CLI Tool**: Powerful scaffolding commands for rapid development

### Architecture

Kashvi follows **MVC** with a repository layer and DTOs. Controllers handle HTTP; they bind DTOs, call services/repositories, and respond:

```
┌─────────────────┐
│  Controllers    │ ← HTTP; bind DTOs, call services/repositories, respond
├─────────────────┤
│      DTOs       │ ← Request/response structs (CreateXRequest, UpdateXRequest, XResponse)
├─────────────────┤
│    Services     │ ← Business logic (optional)
├─────────────────┤
│  Repositories   │ ← Data access; encapsulate orm/DB calls
├─────────────────┤
│     Models      │ ← Data structures (GORM models)
├─────────────────┤
│   Database      │ ← GORM with migrations
└─────────────────┘
```

Controllers bind incoming JSON to **DTOs** (e.g. `CreateProductRequest`), validate them, then map to **models** and use **repositories** (e.g. `ProductRepository`) instead of calling `orm.DB()` directly.

**Where to start:** Use the [documentation map](#documentation-map) above, or go straight to [Installation](#installation), [Configure environment](#4-configure-environment), [Logging](#logging-app-and-database), then the [Complete CRUD walkthrough](#complete-crud-walkthrough-model--migration--repository--service--controller--validation--auth--seed--test). Deeper dives: **[docs/REQUEST_FLOW.md](docs/REQUEST_FLOW.md)** (middleware and flow), **[docs/ERROR_HANDLING.md](docs/ERROR_HANDLING.md)** (errors), **[docs/SOLID_PRINCIPLES.md](docs/SOLID_PRINCIPLES.md)**, **[docs/BENCHMARK.md](docs/BENCHMARK.md)**.

## Installation

### 1. Install Go

Ensure you have Go 1.24 or later installed (see `go.mod` in this repo).

### 2. Install Kashvi CLI

```bash
go install github.com/shashiranjanraj/kashvi/cmd/kashvi@latest
```

### 3. Create a New Project

```bash
kashvi new myproject
cd myproject
```

### 4. Configure Environment

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` with your database and other settings (see **Logging** for `LOG_LEVEL` and `DB_LOG_MODE`):

```env
DB_DRIVER=sqlite
DATABASE_DSN=kashvi.db
JWT_SECRET=your-secret-key
APP_PORT=8080
APP_ENV=local
REDIS_ADDR=localhost:6379
LOG_LEVEL=info
DB_LOG_MODE=silent
```

## Logging (app and database)

A common need is to see **what the app is doing** (info logs) and **what SQL is running** (database logs). Kashvi uses structured logging and supports both.

### Environment

In `.env`:

```env
# App log level: debug | info | warn | error (default: info)
LOG_LEVEL=debug

# GORM/SQL log level: silent | error | warn | info (default: silent)
# Use "info" in development to log every query; "silent" in production.
DB_LOG_MODE=info
```

- **LOG_LEVEL** — Controls the application logger (startup, requests, errors). Use `debug` in development to see route registration and detailed messages; `info` or `warn` in production.
- **DB_LOG_MODE** — Controls GORM’s SQL logging. Use `info` while developing to see queries and bindings; set to `silent` or `error` in production to reduce noise.

### In your code

Use the global logger for app-level messages:

```go
import "github.com/shashiranjanraj/kashvi/pkg/logger"

logger.Info("user_created", "user_id", user.ID, "email", user.Email)
logger.Debug("cache_miss", "key", cacheKey)
logger.Warn("rate_limit_approaching", "ip", ip, "count", n)
logger.Error("payment_failed", "error", err, "order_id", orderID)
```

Arguments are key-value pairs (alternating); they appear as structured fields in the log output.

### Request-scoped logs (with request_id)

The framework injects a **request_id** per request. Use it so you can trace one request across logs:

```go
// In a handler that has *http.Request (e.g. after ctx.Wrap, use c.R.Context())
log := logger.WithCtx(c.R.Context())
log.Info("order_created", "order_id", order.ID, "user_id", userID)
```

If the request passed through `reqid.Middleware()` and `middleware.Logger()`, `WithCtx` returns a logger that already includes `request_id`. The same ID is logged for that request in the HTTP access line (method, path, status, duration).

### Summary

| Goal | What to set / use |
|------|-------------------|
| See app info and debug messages | `LOG_LEVEL=debug` (or `info`) |
| See SQL queries in development | `DB_LOG_MODE=info` |
| Trace one HTTP request in logs | `logger.WithCtx(r.Context())` in handlers |
| Log from anywhere | `logger.Info/Debug/Warn/Error("msg", "key", value)` |

## Quick Start

### Basic Application

Create `main.go`:

```go
package main

import (
    "net/http"
    "github.com/shashiranjanraj/kashvi/pkg/app"
    "github.com/shashiranjanraj/kashvi/pkg/router"
)

func main() {
    app.New().
        Routes(func(r *router.Router) {
            r.Get("/hello", "hello", func(w http.ResponseWriter, req *http.Request) {
                w.Write([]byte("Hello from Kashvi!"))
            })
        }).
        Run()
}
```

Run the application:

```bash
go run main.go serve
# or
kashvi serve
```

### Database Setup

For database support, add models and migrations:

```go
app.New().
    Routes(func(r *router.Router) {
        // routes here
    }).
    AutoMigrate(&User{}).
    Run()
```

## Core Concepts

### Application Builder

The `app.New()` builder configures your application:

```go
app.New().
    Routes(func(r *router.Router) {
        // define routes
    }).
    AutoMigrate(&User{}, &Product{}).
    Seeders(userSeeder).
    Run()
```

### Routing

Kashvi uses Chi router with named routes:

```go
func(r *router.Router) {
    r.Get("/users", "users.index", ctx.Wrap(userController.Index))
    r.Post("/users", "users.store", ctx.Wrap(userController.Store))
    r.Get("/users/{id}", "users.show", ctx.Wrap(userController.Show))
    r.Put("/users/{id}", "users.update", ctx.Wrap(userController.Update))
    r.Delete("/users/{id}", "users.destroy", ctx.Wrap(userController.Destroy))
}
```

### Context

Instead of raw `http.ResponseWriter` and `*http.Request`, Kashvi provides a context object:

```go
func GetUser(c *ctx.Context) {
    id := c.Param("id")
    user := &User{}
    
    if err := orm.DB().Where("id = ?", id).First(user).Error; err != nil {
        c.Error(http.StatusNotFound, "User not found")
        return
    }
    
    c.Success(user)
}
```

Register with `ctx.Wrap`:

```go
r.Get("/users/{id}", "users.show", ctx.Wrap(GetUser))
```

**Handler styles:** Use `ctx.Wrap` for handlers that accept `*ctx.Context` and call `c.Success`, `c.Error`, etc. Use `router.Wrap` for handlers with signature `func(w http.ResponseWriter, r *http.Request) error`; returned errors are converted to HTTP responses via `apperror` (e.g. `apperror.NotFound("msg")`).

### Models

Models are GORM structs:

```go
type User struct {
    gorm.Model
    Name  string `json:"name" gorm:"not null"`
    Email string `json:"email" gorm:"unique;not null"`
}
```

### ORM

Kashvi provides a fluent query builder:

```go
// Get all users
users := []User{}
orm.DB().Get(&users)

// Get with conditions
user := &User{}
orm.DB().Where("email = ?", "john@example.com").First(user)

// Pagination
users, pagination := orm.DB().Paginate(1, 10).GetWithPagination(&users)
```

### Repository layer

Keep data access out of controllers by using **repositories**. Controllers and services call a repository instead of `orm.DB()` directly.

Use the generic `repository.Base[T]` and embed it in a concrete repository per model:

```go
// app/repositories/user.go
package repositories

import (
	"yourapp/app/models"
	"github.com/shashiranjanraj/kashvi/pkg/repository"
)

type UserRepository struct {
	repository.Base[models.User]
}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// Optional: add custom queries
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var u models.User
	err := r.Query().Where("email = ?", email).First(&u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
```

**Base methods:** `FindByID(id uint)`, `All()`, `Create(m *T)`, `Update(m *T)`, `Delete(id uint)`, `Exists(id uint)`, `Count()`, `Paginate(page, limit int)`, and `Query()` for custom chains. Use `Query()` for filters, joins, and scoped queries while keeping all DB access inside the repository.

### DTOs (request / response)

**DTOs** (Data Transfer Objects) define the API request and response shapes separately from your domain models. Controllers bind JSON to DTOs (e.g. `CreateProductRequest`, `UpdateProductRequest`) and validate them before mapping to models and calling the repository. This keeps validation and API contracts in one place.

- **CreateXRequest** — request body for `POST /resources`; use `validate` tags.
- **UpdateXRequest** — request body for `PUT /resources/{id}`; use pointers for optional fields.
- **XResponse** — optional response shape (e.g. to hide or format fields).

Generate DTOs with `kashvi make:dto Product` or as part of `kashvi make:resource Product`. Edit `app/dto/*_dto.go` to match your API.

In your controller, inject the repository and use it:

```go
type UserController struct {
	repo *repositories.UserRepository
}

func NewUserController(repo *repositories.UserRepository) *UserController {
	return &UserController{repo: repo}
}

func (c *UserController) Show(ctx *ctx.Context) {
	id, _ := strconv.ParseUint(ctx.Param("id"), 10, 32)
	user, err := c.repo.FindByID(uint(id))
	if err != nil {
		ctx.NotFound("User not found")
		return
	}
	ctx.Success(user)
}
```

`kashvi make:resource Product` generates a repository, **DTOs** (request/response), and a **controller** that uses them (no direct orm calls in the controller).

## Complete CRUD walkthrough (model → DTO → migration → repository → controller → validation → auth → seed → test)

This section walks through building a full **Product** API using every layer: **model**, **DTOs**, **migration**, **repository**, **controller**, **validation**, **auth** (optional), **seeder**, and **test scenarios**. The controller binds JSON to **DTOs**, validates, maps to the model, and uses the **repository** (no direct `orm` calls).

---

### Step 1: Generate the resource

From your project root (where `go.mod` lives):

```bash
kashvi make:resource Product
```

This creates:

| File | Purpose |
|------|--------|
| `app/models/product.go` | Product model (GORM struct) |
| `app/dto/product_dto.go` | Request/response DTOs (CreateProductRequest, UpdateProductRequest, ProductResponse) |
| `app/repositories/product.go` | Data layer; controller calls this instead of orm |
| `app/controllers/product_controller.go` | CRUD controller (uses repository + DTOs) |
| `app/services/product_service.go` | Business logic (holds repository) |
| `database/migrations/..._create_products_table.go` | Migration with AutoMigrate & DropTable |
| `database/seeders/product_seeder.go` | Seeder using `app.RegisterSeeder` |
| `testdata/product_scenarios.json` | Test scenarios (list, create, get, update, delete) |

The CLI will print the exact route registration snippet to paste into your routes file.

---

### Step 2: Model and validation tags

Edit `app/models/product.go`: add fields and **gorm** / **json** tags. Use **validate** tags so that `ctx.BindJSON(&input)` can validate the body:

```go
package models

import "gorm.io/gorm"

type Product struct {
    gorm.Model
    Name        string  `json:"name"        gorm:"not null"              validate:"required,min=1,max=255"`
    Description string  `json:"description"`
    Price       float64 `json:"price"        gorm:"not null"              validate:"required,gte=0"`
    SKU         string  `json:"sku"         gorm:"uniqueIndex;not null"   validate:"required"`
}
```

Validation runs automatically when the controller calls `ctx.BindJSON(&input)` with a struct that has `validate` tags. Supported rules include `required`, `email`, `min`, `max`, `gte`, `lte`, `in`, `url`, etc. (see **Validation** section later).

---

### Step 3: Migration

The generated migration already:

- **Up:** runs `db.AutoMigrate(&models.Product{})` so the table matches your model.
- **Down:** runs `db.Migrator().DropTable("products")`.

Imports use your module path from `go.mod`. You only need to change the migration if you add custom SQL or indexes.

---

### Step 4: Repository

The generated `app/repositories/product.go` embeds `repository.Base[models.Product]` and exposes:

- `FindByID(id uint)`, `All()`, `Create(m *Product)`, `Update(m *Product)`, `Delete(id uint)`
- `Query()` for custom chains (e.g. filters, pagination)

No change needed for basic CRUD. Add methods like `FindBySKU(sku string)` in this file if you need them; keep all data access here, not in the controller.

---

### Step 5: Service (optional)

The generated service has a **repository** field. Use it for business logic (e.g. checks, multiple repo calls, events):

```go
// app/services/product_service.go
func (s *ProductService) CreateProduct(input *models.Product) error {
    if err := s.repo.Create(input); err != nil {
        return err
    }
    // e.g. logger.Info("product_created", "id", input.ID)
    return nil
}
```

For simple CRUD, the controller can call the repository directly; the service is optional.

---

### Step 6: DTOs (request/response)

The generated `app/dto/product_dto.go` defines:

- **CreateProductRequest** — JSON body for `POST /products`; use `validate` tags for validation.
- **UpdateProductRequest** — JSON body for `PUT /products/{id}`; use pointers for optional fields.
- **ProductResponse** — Optional response shape (e.g. to hide or format fields); Index/Show can return the model as-is or map to this.

Controllers bind to these DTOs with `ctx.BindJSON(&input)` so validation runs before touching the model or repository.

---

### Step 7: Controller and validation

The generated controller uses the **repository** and **DTOs** (no `orm` in the controller). It:

- **Store:** binds JSON to `dto.CreateProductRequest`, validates, maps to `models.Product`, then `repo.Create(model)`.
- **Update:** loads by ID via repo, binds to `dto.UpdateProductRequest`, applies non-nil fields to the model, then `repo.Update(item)`.
- **Index / Show / Destroy:** call `repo.All()`, `repo.FindByID`, `repo.Delete`.

To add **logging** (e.g. for create):

```go
import "github.com/shashiranjanraj/kashvi/pkg/logger"

func (c *ProductController) Store(ctx *ctx.Context) {
    var input dto.CreateProductRequest
    if !ctx.BindJSON(&input) {
        return
    }
    model := &models.Product{Name: input.Name, Description: input.Description, Price: input.Price, SKU: input.SKU}
    if err := c.repo.Create(model); err != nil {
        ctx.Error(http.StatusBadRequest, "Failed to create product")
        return
    }
    log := logger.WithCtx(ctx.R.Context())
    log.Info("product_created", "id", model.ID, "sku", model.SKU)
    ctx.Created(model)
}
```

---

### Step 8: Routes and optional auth

Register routes (and optionally the service) in `main.go` or `app/routes/api.go`:

```go
repo := repositories.NewProductRepository()
svc := services.NewProductService(repo)
ctrl := controllers.NewProductController(repo)

api := r.Group("/api")

// Public
api.Get("/products", "products.index", ctx.Wrap(ctrl.Index))
api.Get("/products/{id}", "products.show", ctx.Wrap(ctrl.Show))

// Protected (JWT required). Import: "github.com/shashiranjanraj/kashvi/pkg/middleware"
protected := api.Group("", middleware.AuthMiddleware)
protected.Post("/products", "products.store", ctx.Wrap(ctrl.Store))
protected.Put("/products/{id}", "products.update", ctx.Wrap(ctrl.Update))
protected.Delete("/products/{id}", "products.destroy", ctx.Wrap(ctrl.Destroy))
```

- **Auth:** use `middleware.AuthMiddleware` on the router or group. The middleware validates the `Authorization: Bearer <token>` header and injects `user_id` and `role` into the request context. In handlers, use `middleware.UserIDFromCtx(r)` or `middleware.RoleFromCtx(r)` to read the current user.

---

### Step 9: Seeder

The generated seeder uses `app.RegisterSeeder("products", func() { ... })`. Edit `database/seeders/product_seeder.go` and add sample data:

```go
func init() {
    app.RegisterSeeder("products", func() {
        database.DB.Create(&[]models.Product{
            {Name: "Laptop", Description: "Gaming", Price: 999.99, SKU: "LAPTOP001"},
            {Name: "Mouse", Description: "Wireless", Price: 29.99, SKU: "MOUSE001"},
        })
    })
}
```

Ensure your app’s migrations (and optionally seeders) are registered (e.g. blank import `_ "yourmodule/database/migrations"` and `_ "yourmodule/database/seeders"` in `main.go`).

---

### Step 10: Run migrations, seed, and serve

```bash
kashvi migrate
kashvi seed
kashvi serve
```

Set **LOG_LEVEL** and **DB_LOG_MODE** in `.env` (see **Logging** section) to see app and SQL logs.

---

### Step 11: Test (curl and test scenarios)

**Manual (curl):**

```bash
# List
curl http://localhost:8080/api/products

# Create (if not using auth)
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Laptop","description":"Gaming","price":999.99,"sku":"LAPTOP001"}'

# Get one
curl http://localhost:8080/api/products/1

# Update
curl -X PUT http://localhost:8080/api/products/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Gaming Laptop","description":"Gaming","price":1099.99,"sku":"LAPTOP001"}'

# Delete
curl -X DELETE http://localhost:8080/api/products/1
```

**Automated (testkit):** the generated `testdata/product_scenarios.json` defines scenarios (list, create, get, update, delete). Build your app’s `http.Handler` (e.g. from `app` + routes) and run:

```go
func TestProductAPI(t *testing.T) {
    handler := app.New().
        Routes(RegisterRoutes).
        Handler()
    testkit.RunDir(t, handler, "testdata")
}
```

Put request/response JSON fixtures in `testdata/` as referenced by the scenario files (e.g. `product_create_req.json`, `product_create_res.json`).

---

### Flow summary

| Layer | Role |
|-------|------|
| **Model** | Struct + gorm/json/validate tags |
| **Migration** | AutoMigrate / DropTable for the model |
| **Repository** | All DB access (FindByID, All, Create, Update, Delete, Query) |
| **Service** | Optional business logic using repository |
| **Controller** | HTTP only: bind DTOs, validate, call repo/service, logger.WithCtx, respond |
| **DTO** | Request/response structs (CreateXRequest, UpdateXRequest, XResponse); validate tags |
| **Validation** | Via `validate` tags and `ctx.BindJSON` (422 on failure) |
| **Auth** | `middleware.AuthMiddleware` on routes; JWT in `Authorization` header |
| **Seeder** | `app.RegisterSeeder("key", func() { ... })` |
| **Test** | testkit + JSON scenarios in `testdata/` |

**See also:** [Request flow & middleware](docs/REQUEST_FLOW.md) · [Error handling](docs/ERROR_HANDLING.md) · [Validation](#validation) · [Authentication](#authentication) · [Database](#database) (migrations, seeders) · [Benchmarks](docs/BENCHMARK.md)

## CLI Commands

### Project Management

- `kashvi version` - Print the Kashvi framework version (e.g. `Kashvi Framework v1.0.16`)
- `kashvi new <name>` - Create a new Kashvi project
- `kashvi serve` - Start the development server
- `kashvi build` - Build the application binary

### Database

- `kashvi migrate` - Run pending migrations
- `kashvi migrate:rollback` - Rollback last migration batch
- `kashvi migrate:status` - Show migration status
- `kashvi seed` - Run database seeders

### Code Generation

- `kashvi make:model <Name>` - Create a model
- `kashvi make:dto <Name>` - Create request/response DTOs for a resource
- `kashvi make:controller <Name>` - Create a controller (MVC)
- `kashvi make:repository <Name>` - Create a repository (data layer for a model)
- `kashvi make:service <Name>` - Create a service
- `kashvi make:migration <name>` - Create a migration
- `kashvi make:seeder <Name>` - Create a seeder
- `kashvi make:resource <Name>` - Create complete CRUD resource (model + dto + repository + controller + service + migration + seeder)

### Background Tasks

- `kashvi queue:work` - Start queue worker
- `kashvi schedule:run` - Run scheduled tasks

## Database

### Supported Drivers

- SQLite (default)
- PostgreSQL
- MySQL
- SQL Server

### Configuration

Set in `.env`:

```env
DB_DRIVER=postgres
DATABASE_DSN=host=localhost user=postgres password=postgres dbname=kashvi port=5432 sslmode=disable
```

### Migrations

Create a migration:

```bash
kashvi make:migration create_users_table
```

Edit the generated file:

```go
func (m *CreateUsersTable) Up(db *gorm.DB) error {
    return db.AutoMigrate(&models.User{})
}

func (m *CreateUsersTable) Down(db *gorm.DB) error {
    return db.Migrator().DropTable("users")
}
```

Register in `database/migrations/register.go`:

```go
func init() {
    migration.Register("20240101000000_create_users_table", &CreateUsersTable{})
}
```

### Seeders

Create a seeder:

```bash
kashvi make:seeder UserSeeder
```

Edit the seeder:

```go
func init() {
    app.RegisterSeeder("users", func() {
        users := []models.User{
            {Name: "John Doe", Email: "john@example.com"},
            {Name: "Jane Doe", Email: "jane@example.com"},
        }
        database.DB.Create(&users)
    })
}
```

## Authentication

### JWT Setup

Configure JWT secret in `.env`:

```env
JWT_SECRET=your-super-secret-key
```

### Auth middleware

Apply JWT validation to a **route group** (recommended). The package is `github.com/shashiranjanraj/kashvi/pkg/middleware`.

```go
api := r.Group("/api", middleware.AuthMiddleware)
api.Get("/me", "me", ctx.Wrap(meHandler))
```

For role checks after auth, use `github.com/shashiranjanraj/kashvi/pkg/rbac` (e.g. `rbac.HasRole("admin")`) on a nested group.

### Login Example

```go
func Login(c *ctx.Context) {
    var input struct {
        Email    string `json:"email" validate:"required,email"`
        Password string `json:"password" validate:"required"`
    }
    
    if !c.BindJSON(&input) {
        return
    }
    
    // Verify credentials
    user := &models.User{}
    if err := orm.DB().Where("email = ?", input.Email).First(user); err != nil {
        c.Error(http.StatusUnauthorized, "Invalid credentials")
        return
    }
    
    // Generate token
    token, err := auth.GenerateToken(user.ID, "user")
    if err != nil {
        c.Error(http.StatusInternalServerError, "Failed to generate token")
        return
    }
    
    c.Success(map[string]any{
        "user":  user,
        "token": token,
    })
}
```

## Caching

### Basic Usage

```go
// Set cache
cache.Set("key", "value", time.Hour)

// Get cache
value, found := cache.Get("key")

// Remember (get or set)
value := cache.Remember("key", time.Hour, func() interface{} {
    return expensiveOperation()
})
```

### In Controllers

```go
func (c *ProductController) Index(ctx *ctx.Context) {
    var products []models.Product
    
    // Try cache first
    if cached, found := cache.Get("products"); found {
        ctx.Success(cached)
        return
    }
    
    // Fetch from database
    orm.DB().Get(&products)
    
    // Cache for 5 minutes
    cache.Set("products", products, 5*time.Minute)
    
    ctx.Success(products)
}
```

## Queues

### Dispatching Jobs

```go
// Simple job
queue.Dispatch(&SendEmail{Email: "user@example.com", Message: "Hello!"})

// Delayed job
queue.DispatchIn(&SendEmail{Email: "user@example.com"}, time.Hour)

// Job on specific queue
queue.DispatchOn("emails", &SendEmail{Email: "user@example.com"})
```

### Defining Jobs

```go
type SendEmail struct {
    Email   string
    Message string
}

func (j *SendEmail) Handle() error {
    return mail.Send(j.Email, "Subject", j.Message)
}
```

### Running Workers

```bash
kashvi queue:work
```

## Testing

### Test Scenarios

Kashvi uses JSON-based test scenarios. Create `testdata/user_scenarios.json`:

```json
[
  {
    "name": "TestUserCreateSuccess",
    "description": "Verify successful user creation",
    "requestUrl": "/api/users",
    "requestMethod": "POST",
    "requestFileName": "create_user_req.json",
    "responseFileName": "create_user_res.json",
    "expectedStatusCode": 201
  }
]
```

### Running Tests

```go
func TestUserAPI(t *testing.T) {
    // Use the same Routes(...) registration as production.
    handler := app.New().
        Routes(RegisterRoutes).
        Handler()
    testkit.RunDir(t, handler, "testdata")
}
```

## Middleware

### Built-in middleware

The **HTTP kernel** already registers metrics, recovery, request ID, logger, session, CORS, and rate limiting (see [`docs/REQUEST_FLOW.md`](docs/REQUEST_FLOW.md)). In your routes you typically add **JWT** and optional **RBAC** per group.

| Symbol | Role |
|--------|------|
| `middleware.AuthMiddleware` | Validates `Authorization: Bearer` and sets user context |
| `middleware.CORS(opts)` | CORS; use `middleware.DefaultCORSOptions()` or custom `CORSOptions` |
| `middleware.Logger` | Request logging (also applied globally in the kernel) |
| `middleware.RateLimit(max, window)` | Rate limiter (also applied globally) |
| `middleware.Recovery` | Panic recovery (also applied globally) |

### Usage (typical: protect a group)

```go
import (
    "github.com/shashiranjanraj/kashvi/pkg/middleware"
    "github.com/shashiranjanraj/kashvi/pkg/rbac"
)

api := r.Group("/api", middleware.AuthMiddleware)
api.Get("/profile", "profile", ctx.Wrap(getProfile))

admin := api.Group("/admin", rbac.HasRole("admin"))
admin.Get("/reports", "admin.reports", ctx.Wrap(reportsHandler))
```

## WebSockets

### Basic WebSocket Handler

```go
func ChatHandler(c *ctx.Context) {
    conn, err := ws.Upgrade(c.W, c.R)
    if err != nil {
        c.Error(http.StatusBadRequest, "Failed to upgrade connection")
        return
    }
    defer conn.Close()
    
    for {
        messageType, message, err := conn.ReadMessage()
        if err != nil {
            break
        }
        
        // Echo the message
        conn.WriteMessage(messageType, message)
    }
}
```

## gRPC Support

### Defining Services

Create `grpc/server.go`:

```go
type server struct {
    pb.UnimplementedUserServiceServer
}

func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    // Implementation
}
```

### Starting gRPC Server

The gRPC server starts automatically alongside the HTTP server. Configure the port in `.env`:

```env
GRPC_PORT=50051
```

## Deployment

### Building for Production

```bash
kashvi build
```

### Docker

Create `Dockerfile`:

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
CMD ["./main"]
```

### Environment Variables

For production, set:

```env
APP_ENV=production
JWT_SECRET=your-production-secret
DB_DRIVER=postgres
DATABASE_DSN=your-production-dsn
REDIS_ADDR=your-redis-host:6379
```

## Best Practices

### Project Structure

```
myproject/
├── main.go
├── config/
│   └── app.json
├── app/
│   ├── controllers/
│   ├── dto/
│   ├── models/
│   ├── repositories/
│   ├── services/
│   └── routes/
├── database/
│   ├── migrations/
│   └── seeders/
├── testdata/
├── go.mod
└── .env
```

### Error Handling

```go
func CreateUser(c *ctx.Context) {
    var input CreateUserInput
    if !c.BindJSON(&input) {
        return // BindJSON handles validation errors
    }
    
    // Business logic
    if err := userService.Create(&input); err != nil {
        c.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    c.Success(map[string]string{"message": "User created"})
}
```

For `errors.Is`, `errors.As`, wrapping with `%w`, and HTTP errors from handlers using `router.Wrap`, see **[docs/ERROR_HANDLING.md](docs/ERROR_HANDLING.md)**.

### Validation

Validation runs when you call `ctx.BindJSON(&input)` in a controller (typically binding to a DTO). Use **validate** struct tags on the model or on a dedicated input struct. On failure the framework returns **422** with validation errors.

```go
type CreateUserInput struct {
    Name     string `json:"name" validate:"required,min=2,max=100"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

// In handler:
if !ctx.BindJSON(&input) {
    return  // 422 and error details already sent
}
```

Common rules: `required`, `email`, `min`, `max`, `gte`, `lte`, `oneof`, `url`, `uuid`. Used in the **Complete CRUD walkthrough** on the model (e.g. `validate:"required,gte=0"` on price).

### Logging

```go
logger.Info("user_created", "user_id", user.ID, "email", user.Email)
logger.Error("database_error", "error", err)
```

## API Reference

### Context Methods

- `c.Param(key)` - Get URL parameter
- `c.Query(key)` - Get query string parameter
- `c.BindJSON(&struct{})` - Parse JSON body with validation
- `c.Success(data)` - Return 200 OK with data
- `c.Created(data)` - Return 201 Created
- `c.Error(status, message)` - Return error response
- `c.Status(code)` - Set status code only

### ORM Methods

- `orm.DB().Where(condition, args)` - Add WHERE clause
- `orm.DB().OrderBy(column, direction)` - Add ORDER BY
- `orm.DB().Paginate(page, limit)` - Add pagination
- `orm.DB().With(relation)` - Eager load relation
- `orm.DB().Get(&slice)` - Execute SELECT
- `orm.DB().First(&struct)` - Get first record
- `orm.DB().Create(&struct)` - Insert record
- `orm.DB().Save(&struct)` - Update record
- `orm.DB().Delete(&struct)` - Delete record

### Cache Methods

- `cache.Set(key, value, ttl)` - Set cache value
- `cache.Get(key)` - Get cache value
- `cache.Remember(key, ttl, fn)` - Get or set with callback
- `cache.Delete(key)` - Delete cache key
- `cache.Flush()` - Clear all cache

### Queue Methods

- `queue.Dispatch(job)` - Dispatch job immediately
- `queue.DispatchIn(job, delay)` - Dispatch job with delay
- `queue.DispatchOn(queue, job)` - Dispatch to specific queue

## Advanced Configuration & Extensions

### Custom Middleware

Kashvi middleware follows the standard `func(http.Handler) http.Handler` pattern. Create custom middleware in `pkg/middleware/` or your project's middleware directory.

#### Creating Custom Middleware

```go
// pkg/middleware/custom.go
package middleware

import (
    "net/http"
    "time"
    
    "github.com/shashiranjanraj/kashvi/pkg/logger"
)

// RequestTimer adds request timing to responses
func RequestTimer(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Create a response writer wrapper to capture status
        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        next.ServeHTTP(rw, r)
        
        duration := time.Since(start)
        
        // Add timing header
        w.Header().Set("X-Response-Time", duration.String())
        
        // Log slow requests
        if duration > 500*time.Millisecond {
            logger.WithCtx(r.Context()).Warn("slow_request",
                "method", r.Method,
                "path", r.URL.Path,
                "duration", duration.String(),
            )
        }
    })
}

// responseWriter captures the status code
type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

#### Using Custom Middleware

```go
func main() {
    app.New().
        Routes(func(r *router.Router) {
            // Apply to all routes
            r.Use(middleware.RequestTimer())
            
            r.Get("/api/users", "users.index", ctx.Wrap(userController.Index))
        }).
        Run()
}
```

#### Authentication Middleware Example

```go
// middleware/auth.go
func Auth() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "Missing authorization header", http.StatusUnauthorized)
                return
            }
            
            // Extract token
            tokenString := strings.TrimPrefix(authHeader, "Bearer ")
            if tokenString == authHeader {
                http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
                return
            }
            
            // Validate token
            claims, err := auth.ValidateToken(tokenString)
            if err != nil {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }
            
            // Add user ID to context
            ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
            r = r.WithContext(ctx)
            
            next.ServeHTTP(w, r)
        })
    }
}

// Usage
protected := r.Group("/api", middleware.AuthMiddleware)
protected.Get("/profile", "profile", ctx.Wrap(getProfile))
```

### Advanced CORS Configuration

The CORS middleware supports detailed configuration for production environments.

#### Production CORS Setup

```go
// config/cors.go
package config

import "github.com/shashiranjanraj/kashvi/pkg/middleware"

func CORSConfig() middleware.CORSOptions {
    if AppEnv() == "production" {
        return middleware.CORSOptions{
            AllowedOrigins: []string{
                "https://yourapp.com",
                "https://admin.yourapp.com",
            },
            AllowedMethods: []string{
                "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
            },
            AllowedHeaders: []string{
                "Accept",
                "Authorization", 
                "Content-Type",
                "X-CSRF-Token",
                "X-Requested-With",
            },
            MaxAge: 86400, // 24 hours
        }
    }
    
    // Development - permissive
    return middleware.DefaultCORSOptions()
}
```

#### Using Custom CORS

```go
func main() {
    app.New().
        Routes(func(r *router.Router) {
            r.Use(middleware.CORS(config.CORSConfig()))
            // ... routes
        }).
        Run()
}
```

#### Dynamic CORS Origins

For multi-tenant applications:

```go
func DynamicCORS() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            
            // Check if origin is allowed (database lookup, etc.)
            if isAllowedOrigin(origin) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
                w.Header().Set("Access-Control-Allow-Credentials", "true")
            }
            
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### Advanced Logging Setup

Kashvi supports structured logging with MongoDB integration and custom handlers.

#### Custom Log Levels

```go
// config/logger.go
func SetupLogging() {
    var level slog.Level
    
    switch config.AppEnv() {
    case "production":
        level = slog.LevelInfo
    case "staging":
        level = slog.LevelDebug
    default:
        level = slog.LevelDebug
    }
    
    opts := &slog.HandlerOptions{
        Level: level,
        AddSource: config.AppEnv() == "development",
    }
    
    handler := slog.NewJSONHandler(os.Stdout, opts)
    logger := slog.New(handler)
    
    // Set as default
    slog.SetDefault(logger)
}
```

#### MongoDB Log Shipping

Configure in `.env`:

```env
MONGO_URI=mongodb://localhost:27017
MONGO_LOG_DB=kashvi_logs
MONGO_LOG_COLLECTION=app_logs
```

All logs will be automatically shipped to MongoDB:

```go
// In main.go
func main() {
    defer logger.CloseMongoHandler() // Flush logs on shutdown
    
    app.New().
        // ... routes
        Run()
}
```

#### Custom Log Handlers

```go
// logger/file_handler.go
type FileHandler struct {
    file *os.File
    slog.Handler
}

func NewFileHandler(path string) (*FileHandler, error) {
    file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    
    return &FileHandler{
        file: file,
        Handler: slog.NewJSONHandler(file, &slog.HandlerOptions{}),
    }, nil
}

func (h *FileHandler) Close() error {
    return h.file.Close()
}
```

#### Contextual Logging

```go
func ProcessPayment(c *ctx.Context) {
    log := logger.WithCtx(c.R.Context())
    
    log.Info("starting_payment", "amount", 99.99, "user_id", 123)
    
    // Process payment...
    
    if err := processPayment(); err != nil {
        log.Error("payment_failed", "error", err, "user_id", 123)
        c.Error(http.StatusBadRequest, "Payment failed")
        return
    }
    
    log.Info("payment_success", "transaction_id", "txn_123")
    c.Success(map[string]string{"status": "paid"})
}
```

### Advanced Database Configuration

#### Connection Pool Tuning

```go
// config/database.go
func ConfigureDatabase() {
    db, err := gorm.Open(sqlite.Open("kashvi.db"), &gorm.Config{})
    if err != nil {
        panic(err)
    }
    
    sqlDB, err := db.DB()
    if err != nil {
        panic(err)
    }
    
    // Production tuning
    sqlDB.SetMaxOpenConns(100)                 // Maximum open connections
    sqlDB.SetMaxIdleConns(10)                  // Maximum idle connections
    sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Connection max lifetime
    sqlDB.SetConnMaxIdleTime(2 * time.Minute)  // Idle connection max lifetime
}
```

#### Read/Write Database Split

```go
// database/connection.go
var ReadDB *gorm.DB
var WriteDB *gorm.DB

func InitDB() {
    // Write database (master)
    writeDSN := "master-db-connection-string"
    WriteDB, _ = gorm.Open(postgres.Open(writeDSN), &gorm.Config{})
    
    // Read database (replica)
    readDSN := "replica-db-connection-string" 
    ReadDB, _ = gorm.Open(postgres.Open(readDSN), &gorm.Config{})
}

func GetDB(readonly bool) *gorm.DB {
    if readonly {
        return ReadDB
    }
    return WriteDB
}
```

#### Custom Database Hooks

```go
// models/base.go
type BaseModel struct {
    ID        uint      `gorm:"primarykey"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate hook
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
    // Set default values, validate, etc.
    return nil
}

// AfterCreate hook
func (b *BaseModel) AfterCreate(tx *gorm.DB) error {
    // Post-creation logic (cache invalidation, notifications, etc.)
    return nil
}
```

#### Database Migrations with Rollback

```go
// database/migrations/001_create_users.go
func (m *CreateUsers) Up(db *gorm.DB) error {
    return db.Exec(`
        CREATE TABLE users (
            id SERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            email VARCHAR(255) UNIQUE NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        
        CREATE INDEX idx_users_email ON users(email);
    `).Error
}

func (m *CreateUsers) Down(db *gorm.DB) error {
    return db.Exec(`
        DROP INDEX IF EXISTS idx_users_email;
        DROP TABLE IF EXISTS users;
    `).Error
}
```

### Prometheus Metrics & Monitoring

Kashvi includes built-in Prometheus metrics with histogram support.

#### Built-in Metrics

```go
// In routes
r.Use(metrics.Middleware())
r.Get("/metrics", "metrics", metrics.Handler())
```

Available metrics:
- `kashvi_http_request_duration_seconds` - Request duration histogram
- `kashvi_http_requests_total` - Total request counter
- `kashvi_http_requests_in_flight` - Current requests gauge
- `kashvi_http_response_size_bytes` - Response size histogram
- `kashvi_db_query_duration_seconds` - Database query duration
- `kashvi_queue_jobs_processed_total` - Queue job counter

#### Custom Metrics

```go
// metrics/custom.go
var (
    // Business metrics
    UsersCreated = prometheus.NewCounter(prometheus.CounterOpts{
        Namespace: "kashvi",
        Subsystem: "business",
        Name:      "users_created_total",
        Help:      "Total number of users created",
    })
    
    // Performance metrics
    CacheHitRatio = prometheus.NewGauge(prometheus.GaugeOpts{
        Namespace: "kashvi",
        Subsystem: "cache",
        Name:      "hit_ratio",
        Help:      "Cache hit ratio (0.0 to 1.0)",
    })
    
    // Histogram for payment processing
    PaymentDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: "kashvi",
            Subsystem: "payment",
            Name:      "processing_duration_seconds",
            Help:      "Payment processing duration",
            Buckets:   []float64{.1, .5, 1, 2.5, 5, 10},
        },
        []string{"method", "currency"},
    )
)

func init() {
    prometheus.MustRegister(UsersCreated, CacheHitRatio, PaymentDuration)
}
```

#### Using Custom Metrics

```go
func CreateUser(c *ctx.Context) {
    start := time.Now()
    
    // ... create user logic ...
    
    metrics.UsersCreated.Inc()
    metrics.PaymentDuration.WithLabelValues("stripe", "usd").Observe(time.Since(start).Seconds())
    
    c.Success(user)
}
```

#### Health Checks

```go
func HealthCheck(c *ctx.Context) {
    // Database health
    dbHealth := "ok"
    if err := database.DB.Exec("SELECT 1").Error; err != nil {
        dbHealth = "error"
    }
    
    // Redis health
    redisHealth := "ok"
    if err := cache.Ping(); err != nil {
        redisHealth = "error"
    }
    
    c.Success(map[string]string{
        "status":      "ok",
        "database":    dbHealth,
        "redis":       redisHealth,
        "timestamp":   time.Now().Format(time.RFC3339),
    })
}

// Register health check
r.Get("/health", "health", ctx.Wrap(HealthCheck))
```

### Message Queue Extensions

#### Current Queue Drivers

Kashvi supports:
- **Memory Driver**: For development/testing
- **Redis Driver**: For production use

#### Adding Kafka Support

To add Kafka support, implement the `Driver` interface:

```go
// queue/kafka_driver.go
package queue

import (
    "context"
    "github.com/segmentio/kafka-go"
)

type KafkaDriver struct {
    writer *kafka.Writer
    reader *kafka.Reader
}

func NewKafkaDriver(brokers []string, topic string) *KafkaDriver {
    return &KafkaDriver{
        writer: &kafka.Writer{
            Addr:     kafka.TCP(brokers...),
            Topic:    topic,
            Balancer: &kafka.LeastBytes{},
        },
        reader: kafka.NewReader(kafka.ReaderConfig{
            Brokers: brokers,
            Topic:   topic,
            GroupID: "kashvi-queue",
        }),
    }
}

func (d *KafkaDriver) Push(payload []byte) error {
    return d.writer.WriteMessages(context.Background(), kafka.Message{
        Value: payload,
    })
}

func (d *KafkaDriver) Pop(ctx context.Context) ([]byte, error) {
    msg, err := d.reader.ReadMessage(ctx)
    if err != nil {
        return nil, err
    }
    
    return msg.Value, nil
}

func (d *KafkaDriver) Close() error {
    if err := d.writer.Close(); err != nil {
        return err
    }
    return d.reader.Close()
}
```

#### Using Kafka Driver

```go
// config/queue.go
func InitQueue() {
    if config.QueueDriver() == "kafka" {
        driver := queue.NewKafkaDriver(
            []string{"localhost:9092"},
            "kashvi-jobs",
        )
        queue.SetDriver(driver)
    }
}
```

### Microservices Architecture

Kashvi is well-suited for microservices with built-in gRPC support.

#### Service Definition

```go
// proto/user.proto
syntax = "proto3";

package user;

service UserService {
    rpc GetUser(GetUserRequest) returns (GetUserResponse);
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
}

message GetUserRequest {
    uint32 id = 1;
}

message GetUserResponse {
    uint32 id = 1;
    string name = 2;
    string email = 3;
}
```

#### gRPC Service Implementation

```go
// grpc/user_server.go
type UserServer struct {
    pb.UnimplementedUserServiceServer
}

func (s *UserServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    var user models.User
    if err := orm.DB().Where("id = ?", req.Id).First(&user).Error; err != nil {
        return nil, status.Error(codes.NotFound, "user not found")
    }
    
    return &pb.GetUserResponse{
        Id:    uint32(user.ID),
        Name:  user.Name,
        Email: user.Email,
    }, nil
}
```

#### Service Registration

```go
// grpc/server.go
func RegisterServices(server *grpc.Server) {
    pb.RegisterUserServiceServer(server, &UserServer{})
}
```

#### Inter-Service Communication

```go
// client/user_client.go
func GetUserFromService(userID uint32) (*pb.GetUserResponse, error) {
    conn, err := grpc.Dial("user-service:50051", grpc.WithInsecure())
    if err != nil {
        return nil, err
    }
    defer conn.Close()
    
    client := pb.NewUserServiceClient(conn)
    return client.GetUser(context.Background(), &pb.GetUserRequest{Id: userID})
}
```

### Serverless Deployment

Kashvi can be adapted for serverless environments, though it requires modifications since it starts its own HTTP server.

#### AWS Lambda Adapter

```go
// serverless/lambda.go
package serverless

import (
    "context"
    "net/http"
    
    "github.com/aws/aws/aws-lambda-go/events"
    "github.com/aws/aws/aws-lambda-go/lambda"
)

type LambdaHandler struct {
    handler http.Handler
}

func (h *LambdaHandler) Handle(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Convert API Gateway request to HTTP request
    httpReq, err := h.convertToHTTPRequest(req)
    if err != nil {
        return events.APIGatewayProxyResponse{StatusCode: 400}, err
    }
    
    // Create response recorder
    recorder := &responseRecorder{}
    
    // Serve request
    h.handler.ServeHTTP(recorder, httpReq)
    
    // Convert back to API Gateway response
    return events.APIGatewayProxyResponse{
        StatusCode: recorder.statusCode,
        Headers:    recorder.headers,
        Body:       recorder.body.String(),
    }, nil
}

func StartLambda(handler http.Handler) {
    lambdaHandler := &LambdaHandler{handler: handler}
    lambda.Start(lambdaHandler.Handle)
}
```

#### Vercel/Next.js Integration

```go
// api/index.go
package main

import (
    "net/http"
    "github.com/shashiranjanraj/kashvi/pkg/app"
    "github.com/shashiranjanraj/kashvi/pkg/router"
)

var handler http.Handler

func init() {
    handler = app.New().
        Routes(func(r *router.Router) {
            r.Get("/api/users", "users.index", userHandler)
        }).
        Handler()
}

func Handler(w http.ResponseWriter, r *http.Request) {
    handler.ServeHTTP(w, r)
}
```

#### Docker for Serverless

```dockerfile
# Dockerfile.serverless
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main ./cmd/serverless

FROM public.ecr.aws/lambda/go:1
COPY --from=builder /app/main ${LAMBDA_TASK_ROOT}
CMD ["main"]
```

### Additional Extensions

#### Rate Limiting with Redis

```go
// middleware/rate_limit.go
func RateLimit(rps int) func(http.Handler) http.Handler {
    limiter := tollbooth.NewLimiter(float64(rps), nil)
    limiter.SetIPLookups([]string{"X-Real-IP", "X-Forwarded-For"})
    return tollbooth.LimitHandler(limiter)
}

// Usage
r.Use(middleware.RateLimit(10)) // 10 requests per second
```

#### File Upload Handling

```go
// middleware/upload.go
func FileUpload(maxSize int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxSize)
            next.ServeHTTP(w, r)
        })
    }
}

// Controller
func UploadFile(c *ctx.Context) {
    file, header, err := c.R.FormFile("file")
    if err != nil {
        c.Error(http.StatusBadRequest, "Failed to get file")
        return
    }
    defer file.Close()
    
    // Save to storage
    path, err := storage.Put(file, header.Filename)
    if err != nil {
        c.Error(http.StatusInternalServerError, "Failed to save file")
        return
    }
    
    c.Success(map[string]string{"path": path})
}
```

#### API Versioning

```go
// middleware/version.go
func Version(version string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := context.WithValue(r.Context(), "api_version", version)
            r = r.WithContext(ctx)
            w.Header().Set("X-API-Version", version)
            next.ServeHTTP(w, r)
        })
    }
}

// Routes
v1 := r.Group("/api/v1", middleware.Version("v1"))
v1.Get("/users", "users.index", ctx.Wrap(userController.Index))

v2 := r.Group("/api/v2", middleware.Version("v2"))
v2.Get("/users", "users.index", ctx.Wrap(userControllerV2.Index))
```

#### Database Sharding

```go
// database/shard.go
type ShardManager struct {
    shards map[string]*gorm.DB
}

func (sm *ShardManager) GetShard(userID uint) *gorm.DB {
    shardKey := fmt.Sprintf("shard_%d", userID%sm.numShards)
    return sm.shards[shardKey]
}

// Usage
shard := GetShardManager().GetShard(user.ID)
shard.Create(&user)
```</content>
<parameter name="newString">## Advanced Configuration & Extensions

### Custom Middleware

Kashvi middleware follows the standard `func(http.Handler) http.Handler` pattern. Create custom middleware in `pkg/middleware/` or your project's middleware directory.

#### Creating Custom Middleware

```go
// pkg/middleware/custom.go
package middleware

import (
    "net/http"
    "time"
    
    "github.com/shashiranjanraj/kashvi/pkg/logger"
)

// RequestTimer adds request timing to responses
func RequestTimer(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Create a response writer wrapper to capture status
        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        next.ServeHTTP(rw, r)
        
        duration := time.Since(start)
        
        // Add timing header
        w.Header().Set("X-Response-Time", duration.String())
        
        // Log slow requests
        if duration > 500*time.Millisecond {
            logger.WithCtx(r.Context()).Warn("slow_request",
                "method", r.Method,
                "path", r.URL.Path,
                "duration", duration.String(),
            )
        }
    })
}

// responseWriter captures the status code
type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

#### Using Custom Middleware

```go
func main() {
    app.New().
        Routes(func(r *router.Router) {
            // Apply to all routes
            r.Use(middleware.RequestTimer())
            
            r.Get("/api/users", "users.index", ctx.Wrap(userController.Index))
        }).
        Run()
}
```

#### Authentication Middleware Example

```go
// middleware/auth.go
func Auth() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "Missing authorization header", http.StatusUnauthorized)
                return
            }
            
            // Extract token
            tokenString := strings.TrimPrefix(authHeader, "Bearer ")
            if tokenString == authHeader {
                http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
                return
            }
            
            // Validate token
            claims, err := auth.ValidateToken(tokenString)
            if err != nil {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }
            
            // Add user ID to context
            ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
            r = r.WithContext(ctx)
            
            next.ServeHTTP(w, r)
        })
    }
}

// Usage
protected := r.Group("/api", middleware.AuthMiddleware)
protected.Get("/profile", "profile", ctx.Wrap(getProfile))
```

### Advanced CORS Configuration

The CORS middleware supports detailed configuration for production environments.

#### Production CORS Setup

```go
// config/cors.go
package config

import "github.com/shashiranjanraj/kashvi/pkg/middleware"

func CORSConfig() middleware.CORSOptions {
    if AppEnv() == "production" {
        return middleware.CORSOptions{
            AllowedOrigins: []string{
                "https://yourapp.com",
                "https://admin.yourapp.com",
            },
            AllowedMethods: []string{
                "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
            },
            AllowedHeaders: []string{
                "Accept",
                "Authorization", 
                "Content-Type",
                "X-CSRF-Token",
                "X-Requested-With",
            },
            MaxAge: 86400, // 24 hours
        }
    }
    
    // Development - permissive
    return middleware.DefaultCORSOptions()
}
```

#### Using Custom CORS

```go
func main() {
    app.New().
        Routes(func(r *router.Router) {
            r.Use(middleware.CORS(config.CORSConfig()))
            // ... routes
        }).
        Run()
}
```

#### Dynamic CORS Origins

For multi-tenant applications:

```go
func DynamicCORS() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            
            // Check if origin is allowed (database lookup, etc.)
            if isAllowedOrigin(origin) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
                w.Header().Set("Access-Control-Allow-Credentials", "true")
            }
            
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### Advanced Logging Setup

Kashvi supports structured logging with MongoDB integration and custom handlers.

#### Custom Log Levels

```go
// config/logger.go
func SetupLogging() {
    var level slog.Level
    
    switch config.AppEnv() {
    case "production":
        level = slog.LevelInfo
    case "staging":
        level = slog.LevelDebug
    default:
        level = slog.LevelDebug
    }
    
    opts := &slog.HandlerOptions{
        Level: level,
        AddSource: config.AppEnv() == "development",
    }
    
    handler := slog.NewJSONHandler(os.Stdout, opts)
    logger := slog.New(handler)
    
    // Set as default
    slog.SetDefault(logger)
}
```

#### MongoDB Log Shipping

Configure in `.env`:

```env
MONGO_URI=mongodb://localhost:27017
MONGO_LOG_DB=kashvi_logs
MONGO_LOG_COLLECTION=app_logs
```

All logs will be automatically shipped to MongoDB:

```go
// In main.go
func main() {
    defer logger.CloseMongoHandler() // Flush logs on shutdown
    
    app.New().
        // ... routes
        Run()
}
```

#### Custom Log Handlers

```go
// logger/file_handler.go
type FileHandler struct {
    file *os.File
    slog.Handler
}

func NewFileHandler(path string) (*FileHandler, error) {
    file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    
    return &FileHandler{
        file: file,
        Handler: slog.NewJSONHandler(file, &slog.HandlerOptions{}),
    }, nil
}

func (h *FileHandler) Close() error {
    return h.file.Close()
}
```

#### Contextual Logging

```go
func ProcessPayment(c *ctx.Context) {
    log := logger.WithCtx(c.R.Context())
    
    log.Info("starting_payment", "amount", 99.99, "user_id", 123)
    
    // Process payment...
    
    if err := processPayment(); err != nil {
        log.Error("payment_failed", "error", err, "user_id", 123)
        c.Error(http.StatusBadRequest, "Payment failed")
        return
    }
    
    log.Info("payment_success", "transaction_id", "txn_123")
    c.Success(map[string]string{"status": "paid"})
}
```

### Advanced Database Configuration

#### Connection Pool Tuning

```go
// config/database.go
func ConfigureDatabase() {
    db, err := gorm.Open(sqlite.Open("kashvi.db"), &gorm.Config{})
    if err != nil {
        panic(err)
    }
    
    sqlDB, err := db.DB()
    if err != nil {
        panic(err)
    }
    
    // Production tuning
    sqlDB.SetMaxOpenConns(100)                 // Maximum open connections
    sqlDB.SetMaxIdleConns(10)                  // Maximum idle connections
    sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Connection max lifetime
    sqlDB.SetConnMaxIdleTime(2 * time.Minute)  // Idle connection max lifetime
}
```

#### Read/Write Database Split

```go
// database/connection.go
var ReadDB *gorm.DB
var WriteDB *gorm.DB

func InitDB() {
    // Write database (master)
    writeDSN := "master-db-connection-string"
    WriteDB, _ = gorm.Open(postgres.Open(writeDSN), &gorm.Config{})
    
    // Read database (replica)
    readDSN := "replica-db-connection-string" 
    ReadDB, _ = gorm.Open(postgres.Open(readDSN), &gorm.Config{})
}

func GetDB(readonly bool) *gorm.DB {
    if readonly {
        return ReadDB
    }
    return WriteDB
}
```

#### Custom Database Hooks

```go
// models/base.go
type BaseModel struct {
    ID        uint      `gorm:"primarykey"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate hook
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
    // Set default values, validate, etc.
    return nil
}

// AfterCreate hook
func (b *BaseModel) AfterCreate(tx *gorm.DB) error {
    // Post-creation logic (cache invalidation, notifications, etc.)
    return nil
}
```

#### Database Migrations with Rollback

```go
// database/migrations/001_create_users.go
func (m *CreateUsers) Up(db *gorm.DB) error {
    return db.Exec(`
        CREATE TABLE users (
            id SERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            email VARCHAR(255) UNIQUE NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        
        CREATE INDEX idx_users_email ON users(email);
    `).Error
}

func (m *CreateUsers) Down(db *gorm.DB) error {
    return db.Exec(`
        DROP INDEX IF EXISTS idx_users_email;
        DROP TABLE IF EXISTS users;
    `).Error
}
```

### Prometheus Metrics & Monitoring

Kashvi includes built-in Prometheus metrics with histogram support.

#### Built-in Metrics

```go
// In routes
r.Use(metrics.Middleware())
r.Get("/metrics", "metrics", metrics.Handler())
```

Available metrics:
- `kashvi_http_request_duration_seconds` - Request duration histogram
- `kashvi_http_requests_total` - Total request counter
- `kashvi_http_requests_in_flight` - Current requests gauge
- `kashvi_http_response_size_bytes` - Response size histogram
- `kashvi_db_query_duration_seconds` - Database query duration
- `kashvi_queue_jobs_processed_total` - Queue job counter

#### Custom Metrics

```go
// metrics/custom.go
var (
    // Business metrics
    UsersCreated = prometheus.NewCounter(prometheus.CounterOpts{
        Namespace: "kashvi",
        Subsystem: "business",
        Name:      "users_created_total",
        Help:      "Total number of users created",
    })
    
    // Performance metrics
    CacheHitRatio = prometheus.NewGauge(prometheus.GaugeOpts{
        Namespace: "kashvi",
        Subsystem: "cache",
        Name:      "hit_ratio",
        Help:      "Cache hit ratio (0.0 to 1.0)",
    })
    
    // Histogram for payment processing
    PaymentDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: "kashvi",
            Subsystem: "payment",
            Name:      "processing_duration_seconds",
            Help:      "Payment processing duration",
            Buckets:   []float64{.1, .5, 1, 2.5, 5, 10},
        },
        []string{"method", "currency"},
    )
)

func init() {
    prometheus.MustRegister(UsersCreated, CacheHitRatio, PaymentDuration)
}
```

#### Using Custom Metrics

```go
func CreateUser(c *ctx.Context) {
    start := time.Now()
    
    // ... create user logic ...
    
    metrics.UsersCreated.Inc()
    metrics.PaymentDuration.WithLabelValues("stripe", "usd").Observe(time.Since(start).Seconds())
    
    c.Success(user)
}
```

#### Health Checks

```go
func HealthCheck(c *ctx.Context) {
    // Database health
    dbHealth := "ok"
    if err := database.DB.Exec("SELECT 1").Error; err != nil {
        dbHealth = "error"
    }
    
    // Redis health
    redisHealth := "ok"
    if err := cache.Ping(); err != nil {
        redisHealth = "error"
    }
    
    c.Success(map[string]string{
        "status":      "ok",
        "database":    dbHealth,
        "redis":       redisHealth,
        "timestamp":   time.Now().Format(time.RFC3339),
    })
}

// Register health check
r.Get("/health", "health", ctx.Wrap(HealthCheck))
```

### Message Queue Extensions

#### Current Queue Drivers

Kashvi supports:
- **Memory Driver**: For development/testing
- **Redis Driver**: For production use

#### Adding Kafka Support

To add Kafka support, implement the `Driver` interface:

```go
// queue/kafka_driver.go
package queue

import (
    "context"
    "github.com/segmentio/kafka-go"
)

type KafkaDriver struct {
    writer *kafka.Writer
    reader *kafka.Reader
}

func NewKafkaDriver(brokers []string, topic string) *KafkaDriver {
    return &KafkaDriver{
        writer: &kafka.Writer{
            Addr:     kafka.TCP(brokers...),
            Topic:    topic,
            Balancer: &kafka.LeastBytes{},
        },
        reader: kafka.NewReader(kafka.ReaderConfig{
            Brokers: brokers,
            Topic:   topic,
            GroupID: "kashvi-queue",
        }),
    }
}

func (d *KafkaDriver) Push(payload []byte) error {
    return d.writer.WriteMessages(context.Background(), kafka.Message{
        Value: payload,
    })
}

func (d *KafkaDriver) Pop(ctx context.Context) ([]byte, error) {
    msg, err := d.reader.ReadMessage(ctx)
    if err != nil {
        return nil, err
    }
    
    return msg.Value, nil
}

func (d *KafkaDriver) Close() error {
    if err := d.writer.Close(); err != nil {
        return err
    }
    return d.reader.Close()
}
```

#### Using Kafka Driver

```go
// config/queue.go
func InitQueue() {
    if config.QueueDriver() == "kafka" {
        driver := queue.NewKafkaDriver(
            []string{"localhost:9092"},
            "kashvi-jobs",
        )
        queue.SetDriver(driver)
    }
}
```

### Microservices Architecture

Kashvi is well-suited for microservices with built-in gRPC support.

#### Service Definition

```go
// proto/user.proto
syntax = "proto3";

package user;

service UserService {
    rpc GetUser(GetUserRequest) returns (GetUserResponse);
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
}

message GetUserRequest {
    uint32 id = 1;
}

message GetUserResponse {
    uint32 id = 1;
    string name = 2;
    string email = 3;
}
```

#### gRPC Service Implementation

```go
// grpc/user_server.go
type UserServer struct {
    pb.UnimplementedUserServiceServer
}

func (s *UserServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    var user models.User
    if err := orm.DB().Where("id = ?", req.Id).First(&user).Error; err != nil {
        return nil, status.Error(codes.NotFound, "user not found")
    }
    
    return &pb.GetUserResponse{
        Id:    uint32(user.ID),
        Name:  user.Name,
        Email: user.Email,
    }, nil
}
```

#### Service Registration

```go
// grpc/server.go
func RegisterServices(server *grpc.Server) {
    pb.RegisterUserServiceServer(server, &UserServer{})
}
```

#### Inter-Service Communication

```go
// client/user_client.go
func GetUserFromService(userID uint32) (*pb.GetUserResponse, error) {
    conn, err := grpc.Dial("user-service:50051", grpc.WithInsecure())
    if err != nil {
        return nil, err
    }
    defer conn.Close()
    
    client := pb.NewUserServiceClient(conn)
    return client.GetUser(context.Background(), &pb.GetUserRequest{Id: userID})
}
```

### Serverless Deployment

Kashvi can be adapted for serverless environments, though it requires modifications since it starts its own HTTP server.

#### AWS Lambda Adapter

```go
// serverless/lambda.go
package serverless

import (
    "context"
    "net/http"
    
    "github.com/aws/aws/aws-lambda-go/events"
    "github.com/aws/aws/aws-lambda-go/lambda"
)

type LambdaHandler struct {
    handler http.Handler
}

func (h *LambdaHandler) Handle(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Convert API Gateway request to HTTP request
    httpReq, err := h.convertToHTTPRequest(req)
    if err != nil {
        return events.APIGatewayProxyResponse{StatusCode: 400}, err
    }
    
    // Create response recorder
    recorder := &responseRecorder{}
    
    // Serve request
    h.handler.ServeHTTP(recorder, httpReq)
    
    // Convert back to API Gateway response
    return events.APIGatewayProxyResponse{
        StatusCode: recorder.statusCode,
        Headers:    recorder.headers,
        Body:       recorder.body.String(),
    }, nil
}

func StartLambda(handler http.Handler) {
    lambdaHandler := &LambdaHandler{handler: handler}
    lambda.Start(lambdaHandler.Handle)
}
```

#### Vercel/Next.js Integration

```go
// api/index.go
package main

import (
    "net/http"
    "github.com/shashiranjanraj/kashvi/pkg/app"
    "github.com/shashiranjanraj/kashvi/pkg/router"
)

var handler http.Handler

func init() {
    handler = app.New().
        Routes(func(r *router.Router) {
            r.Get("/api/users", "users.index", userHandler)
        }).
        Handler()
}

func Handler(w http.ResponseWriter, r *http.Request) {
    handler.ServeHTTP(w, r)
}
```

#### Docker for Serverless

```dockerfile
# Dockerfile.serverless
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main ./cmd/serverless

FROM public.ecr.aws/lambda/go:1
COPY --from=builder /app/main ${LAMBDA_TASK_ROOT}
CMD ["main"]
```

### Additional Extensions

#### Rate Limiting with Redis

```go
// middleware/rate_limit.go
func RateLimit(rps int) func(http.Handler) http.Handler {
    limiter := tollbooth.NewLimiter(float64(rps), nil)
    limiter.SetIPLookups([]string{"X-Real-IP", "X-Forwarded-For"})
    return tollbooth.LimitHandler(limiter)
}

// Usage
r.Use(middleware.RateLimit(10)) // 10 requests per second
```

#### File Upload Handling

```go
// middleware/upload.go
func FileUpload(maxSize int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxSize)
            next.ServeHTTP(w, r)
        })
    }
}

// Controller
func UploadFile(c *ctx.Context) {
    file, header, err := c.R.FormFile("file")
    if err != nil {
        c.Error(http.StatusBadRequest, "Failed to get file")
        return
    }
    defer file.Close()
    
    // Save to storage
    path, err := storage.Put(file, header.Filename)
    if err != nil {
        c.Error(http.StatusInternalServerError, "Failed to save file")
        return
    }
    
    c.Success(map[string]string{"path": path})
}
```

#### API Versioning

```go
// middleware/version.go
func Version(version string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := context.WithValue(r.Context(), "api_version", version)
            r = r.WithContext(ctx)
            w.Header().Set("X-API-Version", version)
            next.ServeHTTP(w, r)
        })
    }
}

// Routes
v1 := r.Group("/api/v1", middleware.Version("v1"))
v1.Get("/users", "users.index", ctx.Wrap(userController.Index))

v2 := r.Group("/api/v2", middleware.Version("v2"))
v2.Get("/users", "users.index", ctx.Wrap(userControllerV2.Index))
```

#### Database Sharding

```go
// database/shard.go
type ShardManager struct {
    shards map[string]*gorm.DB
}

func (sm *ShardManager) GetShard(userID uint) *gorm.DB {
    shardKey := fmt.Sprintf("shard_%d", userID%sm.numShards)
    return sm.shards[shardKey]
}

// Usage
shard := GetShardManager().GetShard(user.ID)
shard.Create(&user)
```