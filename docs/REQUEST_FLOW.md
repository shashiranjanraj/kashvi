# Kashvi Request Flow & Middleware

This document describes how an HTTP request travels through Kashvi and the order of all middleware and handlers.

---

## 1. High-level flow

```
                    ┌─────────────────────────────────────────────────────────────────┐
                    │                        HTTP REQUEST IN                             │
                    └─────────────────────────────────────────────────────────────────┘
                                                              │
    ┌─────────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────┐
    │  GLOBAL MIDDLEWARE (app/kernel.go) — applied in order; first registered = outermost = runs first on request IN   │
    ├───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
    │  ① Metrics       │ Start timer, track in-flight; record duration & status on response                             │
    │  ② Recovery      │ defer recover(); convert panic → 500 response                                                   │
    │  ③ Request ID    │ Inject unique X-Request-ID (or from header) into context                                        │
    │  ④ Logger        │ Log request (method, path, request_id from context)                                             │
    │  ⑤ Session       │ Load or create session (Redis), inject into context                                            │
    │  ⑥ CORS          │ Set Access-Control-* headers; handle OPTIONS preflight                                        │
    │  ⑦ Rate limit    │ Per-IP limit (e.g. 200/min); 429 if exceeded                                                    │
    └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
                                                              │
    ┌─────────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────┐
    │  ROUTE MATCH (Chi router) — method + path                                                                         │
    └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
                                                              │
    ┌─────────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────┐
    │  GROUP / PER-ROUTE MIDDLEWARE (if any) — e.g. api := r.Group("/api", middleware.Auth())                           │
    │  • Auth          │ Validate Bearer JWT, inject user_id + role into request context                               │
    │  • Custom        │ Any middleware passed to Group() or to Get/Post(..., middlewares...)                           │
    └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
                                                              │
    ┌─────────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────┐
    │  HANDLER                                                                                                          │
    │  • ctx.Wrap(Controller.Method)  →  *ctx.Context  →  Controller  →  Service (optional)  →  Repository  →  ORM/DB   │
    │  • router.Wrap(HandlerFunc)   →  (w, r) error  →  apperror used for response                                     │
    │  • raw http.HandlerFunc       →  (w, r)                                                                           │
    └───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
                                                              │
                    ┌─────────────────────────────────────────────────────────────────┐
                    │                       HTTP RESPONSE OUT                            │
                    │  (middleware runs in reverse order on the way out)                │
                    └─────────────────────────────────────────────────────────────────┘
```

---

## 2. Middleware order (request IN → response OUT)

| Order | Middleware      | Package        | When it runs (IN)              | What it does (OUT)              |
|-------|-----------------|----------------|--------------------------------|---------------------------------|
| 1     | **Metrics**     | `pkg/metrics`  | First (outermost)              | Records duration, status, size  |
| 2     | **Recovery**    | `pkg/middleware` | After Metrics                | Catches panic → 500             |
| 3     | **Request ID**  | `pkg/reqid`    | After Recovery                 | —                               |
| 4     | **Logger**      | `pkg/middleware` | After ReqID (has request_id) | —                               |
| 5     | **Session**     | `pkg/session`  | After Logger                   | —                               |
| 6     | **CORS**        | `pkg/middleware` | After Session                | Adds CORS headers               |
| 7     | **Rate limit**  | `pkg/middleware` | Last of globals (innermost)  | Returns 429 if over limit       |
| —     | **Group/route** | e.g. `Auth`   | After globals, before handler  | —                               |
| —     | **Handler**     | your code     | Innermost                      | Writes response                 |

On **response out**, the same middleware run in **reverse** (handler → Rate limit → CORS → … → Metrics).

---

## 3. Where middleware is registered

**Global stack** is built in `pkg/app/kernel.go`:

```go
r.Use(metrics.Middleware())
r.Use(middleware.Recovery)
r.Use(reqid.Middleware())
r.Use(middleware.Logger)
r.Use(session.Middleware(session.DefaultOptions()))
r.Use(middleware.CORS(middleware.DefaultCORSOptions()))
r.Use(middleware.RateLimit(200, time.Minute))
```

**Group middleware** (e.g. auth) is added when you create a group:

```go
api := r.Group("/api", middlewares.Auth())
api.Get("/profile", "profile", ctx.Wrap(ctrl.Profile))  // Auth runs before ctrl.Profile
```

**Per-route middleware** can be passed as variadics:

```go
r.Get("/admin", "admin", ctx.Wrap(adminPage), middlewares.Auth(), middlewares.RequireRole("admin"))
```

Router’s `chain()` builds: **handler** wrapped by **route middlewares** (right-to-left), then that chain is what Chi calls after the global stack.

---

## 4. Handler types and application layers

Once the request reaches the **handler**, it can be:

| Style            | Registration              | Handler signature              | Typical flow                          |
|------------------|---------------------------|---------------------------------|----------------------------------------|
| **Context**      | `ctx.Wrap(Controller.X)`  | `func(c *ctx.Context)`         | Controller → Service → Repository → DB |
| **Error-return**| `router.Wrap(h)`          | `func(w, r) error`             | Handler returns `apperror.*`; router sends response |
| **Raw**          | `func(w, r)`             | `http.HandlerFunc`             | No framework context                   |

**Typical Kashvi flow (with repository layer):**

```
ctx.Wrap(ProductController.Index)
    → ProductController.Index(c *ctx.Context)
        → c.repo.All()                    // Repository
            → orm.DB().Model(&Product{}).Get(&list)   // ORM
        → ctx.Success(list)
    → response written
```

---

## 5. Special routes

- **`/metrics`** — registered with `r.HandleFunc("/metrics", metrics.Handler())`. No auth, no rate limit; only the global middleware stack runs.
- **404** — if no route matches, Chi returns 404 (after passing through the full global middleware stack).

---

## 6. Sequence (example: GET /api/products with Auth group)

```
Client
  │
  ├─ GET /api/products
  │
  ▼
① Metrics        start timer
  ▼
② Recovery       defer recover()
  ▼
③ Request ID     context += request_id
  ▼
④ Logger         log "GET /api/products" request_id=...
  ▼
⑤ Session        load session from Redis → context
  ▼
⑥ CORS           set CORS headers
  ▼
⑦ Rate limit     check IP limit → allow
  ▼
  Route match: GET /api/products → group /api (Auth)
  ▼
⑧ Auth           validate Bearer JWT → context += user_id, role (if protected group)
  ▼
⑨ Handler        ctx.Wrap(ProductController.Index)
                   → Controller.Index(c)
                   → c.repo.All()
                   → ctx.Success(products)
  ▼
  Response written
  ▼
⑧→① Middleware run in reverse on way out (e.g. Metrics records duration)
  ▼
Client receives response
```

---

## 7. Handler → DB request flow

Once the request reaches your handler, any database access follows this path. The handler never talks to the DB directly; it goes through **Controller → (Service) → Repository → ORM → Database**.

### 7.1 Flow diagram

```
  HTTP Handler (e.g. ctx.Wrap(ProductController.Show))
       │
       ▼
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │  CONTROLLER                                                                  │
  │  • Receives *ctx.Context (from ctx.Wrap)                                     │
  │  • Parses input: ctx.Param("id"), ctx.BindJSON(&input)                      │
  │  • Calls repository (or service) — no orm.DB() here                          │
  └─────────────────────────────────────────────────────────────────────────────┘
       │
       │  e.g. c.repo.FindByID(uint(id))
       ▼
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │  SERVICE (optional)                                                          │
  │  • Business logic only                                                        │
  │  • Calls repository for data; may call multiple repos or external services  │
  └─────────────────────────────────────────────────────────────────────────────┘
       │
       │  e.g. repo.FindByID(id), repo.Create(&product)
       ▼
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │  REPOSITORY (pkg/repository + app/repositories)                               │
  │  • Concrete repo embeds repository.Base[models.Product]                      │
  │  • FindByID(id) → orm.DB().Model(new(T)).Where("id = ?", id).First(&dest)   │
  │  • All(), Create(m), Update(m), Delete(id), Query() for custom chains       │
  └─────────────────────────────────────────────────────────────────────────────┘
       │
       │  orm.DB() returns *orm.Query; Model/Where/First/Get/Create/Save/Delete
       ▼
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │  ORM (pkg/orm)                                                               │
  │  • orm.DB() → Query{ db: database.DB }                                       │
  │  • Query.Model(v).Where(...).First(dest) → q.db.First(dest) [GORM]          │
  │  • Query.Get(dest), Create(value), Save(value), Delete(value, id)            │
  └─────────────────────────────────────────────────────────────────────────────┘
       │
       │  q.db is *gorm.DB
       ▼
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │  DATABASE (pkg/database)                                                      │
  │  • database.DB *gorm.DB — single connection pool (set at startup)            │
  │  • GORM turns method calls into SQL and executes via the configured driver   │
  └─────────────────────────────────────────────────────────────────────────────┘
       │
       │  SQL (e.g. SELECT * FROM products WHERE id = ?)
       ▼
  ┌─────────────────────────────────────────────────────────────────────────────┐
  │  DB ENGINE (SQLite / PostgreSQL / MySQL / SQL Server)                         │
  └─────────────────────────────────────────────────────────────────────────────┘
```

### 7.2 Step-by-step (example: GET /api/products/1)

| Step | Layer        | Code / action |
|------|--------------|----------------|
| 1    | Handler      | `ctx.Wrap(ProductController.Show)` → `ProductController.Show(c *ctx.Context)` |
| 2    | Controller   | `id, _ := strconv.ParseUint(c.Param("id"), 10, 32)` then `product, err := c.repo.FindByID(uint(id))` |
| 3    | Repository   | `ProductRepository` (embeds `Base[models.Product]`). `FindByID(id)` implemented by Base. |
| 4    | Repository   | `orm.DB().Model(new(Product)).Where("id = ?", id).First(&dest)` → returns `(*Product, error)` |
| 5    | ORM          | `orm.DB()` returns `Query{ db: database.DB }`. `Model(...).Where(...).First(&dest)` → `q.db.First(&dest)` (GORM). |
| 6    | Database     | `database.DB` is the global `*gorm.DB`. GORM generates SQL and runs it (e.g. `SELECT * FROM products WHERE id = 1 AND deleted_at IS NULL`). |
| 7    | DB engine    | Driver (e.g. SQLite/Postgres) executes the query and returns rows. |
| 8    | Back up      | Rows → GORM fills `dest` → Repository returns `*Product` → Controller calls `c.Success(product)` → response written. |

### 7.3 Important points

- **Handler never sees `orm` or `database`** — only the controller (and optional service) call the repository.
- **Repository is the only layer that calls `orm.DB()`** — all DB access is behind the repository API (FindByID, All, Create, Update, Delete, Query()).
- **ORM is a thin wrapper** — `orm.Query` holds `*gorm.DB` and delegates to GORM’s `First`, `Find`, `Create`, `Save`, `Delete`, etc.
- **Single connection** — `database.DB` is one `*gorm.DB` (with connection pool), set once at startup in `database.Connect()` and used by all requests.

---

## 8. Validation and data extraction

**Validation** and **data extraction** from the request both happen in the **controller (handler)** layer, before any call to the repository or service. They are implemented by **pkg/ctx**, **pkg/bind**, and **pkg/validate**.

### 8.1 Where they sit in the flow

```
  Request reaches handler (Controller)
       │
       ├── DATA EXTRACTION (read from request)
       │   • pkg/ctx: Param("id"), Query("page"), PostForm("name"), Header("Authorization"), Cookie("session"), Body()
       │   • pkg/bind: JSON decode from r.Body into a struct (used by BindJSON)
       │
       ├── VALIDATION (rules on extracted data)
       │   • pkg/validate: Struct(dest) — runs validate tags (required, email, min, max, in, regex, etc.)
       │   • Triggered by: c.BindJSON(&input) [decode + validate], or c.Validate(v) for already-filled structs
       │
       └── If valid → controller calls repo/service and writes response
           If invalid → controller sends 400/422 (via c.Error / c.ValidationError) and returns
```

So: **extraction** and **validation** are the first thing the controller does with the request; they do **not** run in middleware or in the repository layer.

### 8.2 Data extraction (where data comes from)

| Source        | How in controller        | Package / implementation |
|---------------|--------------------------|---------------------------|
| URL path      | `c.Param("id")`          | **pkg/ctx** → `chi.URLParam(c.R, key)` |
| Query string  | `c.Query("page")`, `c.DefaultQuery("page", "1")` | **pkg/ctx** → `c.R.URL.Query().Get(key)` |
| JSON body     | `c.BindJSON(&input)` or `c.ShouldBindJSON(&input)` | **pkg/ctx** + **pkg/bind** (decode `r.Body` with `json.Decoder`, size limit via `MaxBytesReader`) |
| Form body     | `c.PostForm("name")`     | **pkg/ctx** → `c.R.FormValue(key)` |
| Headers       | `c.Header("Authorization")` | **pkg/ctx** → `c.R.Header.Get(key)` |
| Cookies       | `c.Cookie("session")`    | **pkg/ctx** → `c.R.Cookie(name)` |
| Raw body      | `c.Body()`               | **pkg/ctx** → `io.ReadAll(c.R.Body)` |

All of this is **controller-side**: the handler uses `*ctx.Context` to read from the request. There is no separate “data extraction layer”; the controller is the place that pulls data out and then validates or passes it on.

### 8.3 Validation (where rules run)

| When / how                         | Where it runs        | Package |
|------------------------------------|----------------------|--------|
| With JSON body                     | Inside `c.BindJSON(&input)` or `bind.JSON(r, dest)` | **pkg/bind** decodes body, then calls **pkg/validate**.`Struct(dest)` |
| On already-filled struct           | `c.Validate(myStruct)` or `validate.Struct(v)`      | **pkg/validate** only |
| Rules                               | Struct tags `validate:"required,email,min=2,max=100"` | **pkg/validate** interprets tags and returns `map[field]errorMsg` |

Flow for a typical POST/PUT:

1. Controller calls `c.BindJSON(&input)`.
2. **pkg/bind**: `bind.JSON(c.R, dest)` → limit body size → `json.Decode(r.Body, dest)` (extraction), then `validate.Struct(dest)` (validation).
3. If decode fails → bind returns error → **ctx** sends 400 and returns false.
4. If validation fails → **ctx** sends 422 with `c.ValidationError(errs)` and returns false.
5. If OK → controller continues (e.g. calls repo.Create(&input)).

So **validation** always runs in the **controller’s call stack** (via **bind** + **validate**, or **ctx.Validate**). It does **not** run in middleware or in the repository.

### 8.4 Summary

- **Data extraction:** In the **controller**, using **pkg/ctx** (params, query, form, headers, cookies, body) and **pkg/bind** (JSON body decode).
- **Validation:** In the **controller**, using **pkg/validate** (struct-tag rules), triggered by **c.BindJSON** / **bind.JSON** or **c.Validate**.
- Both happen **before** any repository or service call; invalid input results in 400/422 from the controller and no DB access.

---

## 9. Summary

- **Request path:** `Metrics → Recovery → ReqID → Logger → Session → CORS → RateLimit → [Group MW e.g. Auth] → Handler`.
- **Response path:** Same middleware in **reverse** after the handler returns.
- **Handler:** Either context-based (`ctx.Wrap`), error-return (`router.Wrap`), or raw `http.HandlerFunc`.
- **Validation and data extraction:** In the **controller** only: **pkg/ctx** (Param, Query, PostForm, Header, Cookie, Body), **pkg/bind** (JSON decode + size limit), **pkg/validate** (struct-tag rules). Triggered by `c.BindJSON(&input)` or `c.Validate(v)`; invalid input → 400/422 before any repo call.
- **Application layers:** Controller → (Service) → Repository → ORM/DB; only the handler (and optionally service) call the repository; the repository is the only place that talks to the ORM.
