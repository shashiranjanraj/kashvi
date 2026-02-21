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
