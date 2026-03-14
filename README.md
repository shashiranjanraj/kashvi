# Kashvi Framework Documentation

## Overview

Kashvi is a Laravel-inspired Go web framework designed for rapid application development. It provides a clean, expressive API with powerful features like ORM, migrations, authentication, caching, queues, and more. Built on top of proven libraries like GORM, Chi router, and Redis, Kashvi helps you build scalable web applications and APIs quickly.

*Made with ❤️ by an Indian developer*

### Key Features

- **MVC Architecture**: Model-View-Controller pattern with clear separation of concerns
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

Kashvi follows a layered architecture with a **repository layer** so controllers and services do not call the ORM directly:

```
┌─────────────────┐
│   Controllers   │ ← Handle HTTP, call services or repositories
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

Controllers and services depend on **repositories** (e.g. `UserRepository`) instead of `orm.DB()`. Repositories expose methods like `FindByID`, `All`, `Create`, `Update`, `Delete` and keep all data access in one place.

For a detailed diagram of how a request travels through Kashvi (all middleware and handler flow), see **[docs/REQUEST_FLOW.md](docs/REQUEST_FLOW.md)**. For how the design aligns with **SOLID** principles, see **[docs/SOLID_PRINCIPLES.md](docs/SOLID_PRINCIPLES.md)**. To run and interpret **benchmarks**, see **[docs/BENCHMARK.md](docs/BENCHMARK.md)**.

## Installation

### 1. Install Go

Ensure you have Go 1.22 or later installed.

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

Edit `.env` with your database and other settings:

```env
DB_DRIVER=sqlite
DATABASE_DSN=kashvi.db
JWT_SECRET=your-secret-key
APP_PORT=8080
APP_ENV=local
REDIS_ADDR=localhost:6379
```

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
    r.Get("/users", "users.index", userController.Index)
    r.Post("/users", "users.store", userController.Store)
    r.Get("/users/{id}", "users.show", userController.Show)
    r.Put("/users/{id}", "users.update", userController.Update)
    r.Delete("/users/{id}", "users.destroy", userController.Destroy)
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

In your controller, inject the repository and use it:

```go
type UserController struct {
	repo *repositories.UserRepository
}

func NewUserController(repo *repositories.UserRepository) *UserController {
	return &UserController{repo: repo}
}

func (c *UserController) Show(ctx *appctx.Context) {
	id, _ := strconv.ParseUint(ctx.Param("id"), 10, 32)
	user, err := c.repo.FindByID(uint(id))
	if err != nil {
		ctx.NotFound("User not found")
		return
	}
	ctx.Success(user)
}
```

`kashvi make:resource Product` generates a repository and a controller that uses it (no direct orm calls in the controller).

## 5-Minute CRUD Creation Deep Dive

Let's create a complete CRUD API for a "Product" resource in 5 minutes.

### Step 1: Generate the Resource (1 minute)

```bash
kashvi make:resource Product
```

This creates:
- `app/models/product.go` - Product model
- `app/repositories/product.go` - Product repository (data layer; controller uses this instead of orm)
- `app/controllers/product_controller.go` - CRUD controller (uses repository)
- `app/services/product_service.go` - Business logic service
- `database/migrations/...` - Migration
- `database/seeders/product_seeder.go` - Database seeder
- `testdata/product_scenarios.json` - Test scenarios

### Step 2: Customize the Model (1 minute)

Edit `app/models/product.go`:

```go
package models

import "gorm.io/gorm"

type Product struct {
    gorm.Model
    Name        string  `json:"name" gorm:"not null"`
    Description string  `json:"description"`
    Price       float64 `json:"price" gorm:"not null"`
    SKU         string  `json:"sku" gorm:"unique;not null"`
}
```

### Step 3: Define the Migration (1 minute)

Edit `database/migrations/20240101000000_create_products_table.go`:

```go
package migrations

import (
    "github.com/shashiranjanraj/kashvi/app/models"
    "gorm.io/gorm"
)

type CreateProductsTable struct{}

func (m *CreateProductsTable) Up(db *gorm.DB) error {
    return db.AutoMigrate(&models.Product{})
}

func (m *CreateProductsTable) Down(db *gorm.DB) error {
    return db.Migrator().DropTable("products")
}
```

### Step 4: Update the Controller (1 minute)

Edit `app/controllers/product_controller.go`:

```go
func (c *ProductController) Index(ctx *appctx.Context) {
    var products []models.Product
    if err := orm.DB().Get(&products); err != nil {
        ctx.Error(http.StatusInternalServerError, "Failed to fetch products")
        return
    }
    ctx.Success(products)
}

func (c *ProductController) Store(ctx *appctx.Context) {
    var input struct {
        Name        string  `json:"name" validate:"required"`
        Description string  `json:"description"`
        Price       float64 `json:"price" validate:"required,min=0"`
        SKU         string  `json:"sku" validate:"required"`
    }
    
    if !ctx.BindJSON(&input) {
        return
    }
    
    product := &models.Product{
        Name:        input.Name,
        Description: input.Description,
        Price:       input.Price,
        SKU:         input.SKU,
    }
    
    if err := orm.DB().Create(product).Error; err != nil {
        ctx.Error(http.StatusBadRequest, "Failed to create product")
        return
    }
    
    ctx.Created(product)
}

func (c *ProductController) Show(ctx *appctx.Context) {
    id := ctx.Param("id")
    var product models.Product
    
    if err := orm.DB().Where("id = ?", id).First(&product).Error; err != nil {
        ctx.Error(http.StatusNotFound, "Product not found")
        return
    }
    
    ctx.Success(product)
}

func (c *ProductController) Update(ctx *appctx.Context) {
    id := ctx.Param("id")
    var product models.Product
    
    if err := orm.DB().Where("id = ?", id).First(&product).Error; err != nil {
        ctx.Error(http.StatusNotFound, "Product not found")
        return
    }
    
    var input struct {
        Name        string  `json:"name"`
        Description string  `json:"description"`
        Price       float64 `json:"price"`
        SKU         string  `json:"sku"`
    }
    
    if !ctx.BindJSON(&input) {
        return
    }
    
    product.Name = input.Name
    product.Description = input.Description
    product.Price = input.Price
    product.SKU = input.SKU
    
    if err := orm.DB().Save(&product).Error; err != nil {
        ctx.Error(http.StatusBadRequest, "Failed to update product")
        return
    }
    
    ctx.Success(product)
}

func (c *ProductController) Destroy(ctx *appctx.Context) {
    id := ctx.Param("id")
    
    if err := orm.DB().Where("id = ?", id).Delete(&models.Product{}).Error; err != nil {
        ctx.Error(http.StatusInternalServerError, "Failed to delete product")
        return
    }
    
    ctx.Status(http.StatusNoContent)
}
```

### Step 5: Register Routes (1 minute)

Add to your routes in `main.go` or `app/routes/api.go` (controller receives the repository):

```go
repo := repositories.NewProductRepository()
ctrl := controllers.NewProductController(repo)
api := r.Group("/api")

api.Get("/products", "products.index", ctx.Wrap(ctrl.Index))
api.Post("/products", "products.store", ctx.Wrap(ctrl.Store))
api.Get("/products/{id}", "products.show", ctx.Wrap(ctrl.Show))
api.Put("/products/{id}", "products.update", ctx.Wrap(ctrl.Update))
api.Delete("/products/{id}", "products.destroy", ctx.Wrap(ctrl.Destroy))
```

### Step 6: Run Migrations and Test (1 minute)

```bash
kashvi migrate
kashvi serve
```

Test the API:

```bash
# Create a product
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Laptop","description":"Gaming laptop","price":999.99,"sku":"LAPTOP001"}'

# Get all products
curl http://localhost:8080/api/products

# Get specific product
curl http://localhost:8080/api/products/1

# Update product
curl -X PUT http://localhost:8080/api/products/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Gaming Laptop","price":1099.99}'

# Delete product
curl -X DELETE http://localhost:8080/api/products/1
```

## CLI Commands

### Project Management

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
- `kashvi make:controller <Name>` - Create a controller
- `kashvi make:repository <Name>` - Create a repository (data layer for a model)
- `kashvi make:service <Name>` - Create a service
- `kashvi make:migration <name>` - Create a migration
- `kashvi make:seeder <Name>` - Create a seeder
- `kashvi make:resource <Name>` - Create complete CRUD resource (model + repository + controller + service + migration + seeder)

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

### Auth Middleware

```go
r.Use(middlewares.Auth())
```

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
    if err := orm.DB().Where("email = ?", input.Email).First(user).Error; err != nil {
        c.Error(http.StatusUnauthorized, "Invalid credentials")
        return
    }
    
    // Generate token
    token, err := auth.GenerateJWT(user.ID)
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
func (c *ProductController) Index(ctx *appctx.Context) {
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
    // Build your app handler
    handler := app.BuildHandler()
    
    // Run all scenarios in testdata/
    testkit.RunDir(t, handler, "testdata")
}
```

## Middleware

### Built-in Middleware

- `middlewares.Auth()` - JWT authentication
- `middlewares.CORS()` - Cross-origin resource sharing
- `middlewares.Logger()` - Request logging
- `middlewares.RateLimit()` - Rate limiting
- `middlewares.Recover()` - Panic recovery

### Usage

```go
r.Use(middlewares.Logger())
r.Use(middlewares.CORS())
r.Use(middlewares.Recover())

// Protected routes
protected := r.Group("/api", middlewares.Auth())
protected.Get("/profile", "profile", ctx.Wrap(getProfile))
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
FROM golang:1.22-alpine AS builder
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

### Validation

```go
type CreateUserInput struct {
    Name     string `json:"name" validate:"required,min=2,max=100"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}
```

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
protected := r.Group("/api", middleware.Auth())
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
    handler = app.BuildHandler(func(r *router.Router) {
        r.Get("/api/users", "users.index", userHandler)
    })
}

func Handler(w http.ResponseWriter, r *http.Request) {
    handler.ServeHTTP(w, r)
}
```

#### Docker for Serverless

```dockerfile
# Dockerfile.serverless
FROM golang:1.22-alpine AS builder
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
protected := r.Group("/api", middleware.Auth())
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
    handler = app.BuildHandler(func(r *router.Router) {
        r.Get("/api/users", "users.index", userHandler)
    })
}

func Handler(w http.ResponseWriter, r *http.Request) {
    handler.ServeHTTP(w, r)
}
```

#### Docker for Serverless

```dockerfile
# Dockerfile.serverless
FROM golang:1.22-alpine AS builder
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
</xai:function_call) 

<xai:function_call name="replace_string_in_file">
<parameter name="filePath">/Users/shashi/devlopment/kashvi/README.md</content>
<parameter name="filePath">/Users/shashi/devlopment/kashvi/README.md