# Kashvi documentation index

This folder holds **focused guides** that complement the main **[README.md](../README.md)**. Use this page to jump straight to a topic or to search by keyword.

---

## Start here

| Guide | What you get |
|--------|----------------|
| [README.md](../README.md) | Install, quick start, CRUD walkthrough, CLI, packages — **primary reference** |
| [REQUEST_FLOW.md](REQUEST_FLOW.md) | HTTP path: global middleware order, groups, handlers, validation |
| [ERROR_HANDLING.md](ERROR_HANDLING.md) | `errors.Is` / `errors.As`, `apperror`, wrapping, `router.Wrap` |
| [SOLID_PRINCIPLES.md](SOLID_PRINCIPLES.md) | How Kashvi maps to SOLID; app-layer tips (interfaces, repos) |
| [BENCHMARK.md](BENCHMARK.md) | `go test -bench`, load testing with `hey` / `wrk` |

---

## Find by task

| I want to… | Read |
|------------|------|
| Install Go and the `kashvi` CLI | [Installation](../README.md#installation) |
| Create a project and run `serve` | [Quick start](../README.md#quick-start) |
| Understand middleware order (metrics → recovery → … → handler) | [REQUEST_FLOW.md §1–2](REQUEST_FLOW.md#1-high-level-flow) |
| Add JWT to a route group | [README — Authentication](../README.md#authentication) · `middleware.AuthMiddleware` |
| Log with `request_id` | [README — Logging](../README.md#logging-app-and-database) |
| Run migrations / seeders | [README — Database](../README.md#database) |
| Generate CRUD (model, DTO, repo, controller) | [README — `make:resource`](../README.md#code-generation) |
| Test HTTP with JSON scenarios | [README — Testing](../README.md#testing) · `pkg/testkit` |
| Compare handler performance | [BENCHMARK.md](BENCHMARK.md) |

---

## Keyword index (search / discover)

Use your editor search on this file or on [README.md](../README.md) for these terms:

- **HTTP:** `Router`, `Group`, `ctx.Wrap`, `router.Wrap`, Chi, route name  
- **Middleware:** `middleware.Recovery`, `middleware.Logger`, `middleware.AuthMiddleware`, `middleware.CORS`, `middleware.RateLimit`, `metrics.Middleware`, `reqid.Middleware`, `session.Middleware`  
- **Data:** GORM, `orm.DB()`, `repository.Base`, migrations, seeders  
- **Errors:** `apperror`, `errors.Is`, `errors.As`, `fmt.Errorf("… %w", err)`  
- **Auth:** JWT, `Authorization: Bearer`, `middleware.UserIDFromCtx`, `rbac.HasRole`  
- **Ops:** Prometheus `/metrics`, `LOG_LEVEL`, `DB_LOG_MODE`, Redis cache/queue  

---

## How this relates to other frameworks

Kashvi is **Laravel-inspired** but **Go-native**: routing and middleware follow standard `net/http` patterns (similar in spirit to **Echo** / **Chi** docs), while concepts like migrations, seeders, and `make:*` commands echo **Laravel**’s developer experience. The main [README.md](../README.md) is the single long-form guide; this directory splits **flow**, **errors**, **SOLID**, and **benchmarks** for easier deep dives.
