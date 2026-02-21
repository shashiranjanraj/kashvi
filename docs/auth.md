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
