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
