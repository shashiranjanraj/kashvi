# Kashvi Framework Documentation

> **Fast like Go, Elegant like Laravel — built with love for Kashvi ❤️**

Kashvi is a batteries-included Go web framework. It gives you everything you need to build a production API **in a single binary** — no boilerplate, no magic.

---

## Documentation Index

| Guide | Description |
|---|---|
| [Installation & Quick Start](./installation.md) | Setup, requirements, first server |
| [Configuration](./configuration.md) | `.env`, `config/app.json`, all env vars |
| [Routing](./routing.md) | Routes, groups, named routes, `route:list` |
| [Context API](./context.md) | `ctx.Context` — request helpers, responses, binding |
| [Middleware](./middleware.md) | Built-in middleware, custom middleware, ordering |
| [Validation](./validation.md) | All 28 rules, custom rules, struct tagging |
| [Authentication](./auth.md) | JWT tokens, bcrypt, RBAC role guards |
| [ORM & Database](./orm.md) | Query builder, pagination, relationships, parallel queries |
| [Migrations & Seeders](./migrations.md) | Up/Down/Rollback/Status, seeder runner |
| [Queue & Jobs](./queue.md) | In-memory + Redis driver, retries, delayed jobs, failed jobs |
| [Task Scheduler](./scheduler.md) | Cron jobs, overlap guard, hooks |
| [Storage](./storage.md) | Local disk, S3/MinIO/R2, `Disk` interface |
| [Cache](./cache.md) | Redis, Get/Set/Forget, ORM cache bridge |
| [WebSocket & SSE](./websocket.md) | `pkg/ws` Hub/Client, `pkg/sse` stream |
| [CLI Reference](./cli.md) | All `kashvi` commands |

---

## Quick Taste

```go
// app/controllers/post_controller.go
func (c *PostController) Store(ctx *appctx.Context) {
    var input struct {
        Title string `json:"title" validate:"required,min=3"`
        Body  string `json:"body"  validate:"required"`
    }
    if !ctx.BindJSON(&input) { // auto 422 on fail
        return
    }

    post := models.Post{Title: input.Title, Body: input.Body}
    orm.New(database.DB).Create(&post)
    ctx.Created(post)
}
```

```bash
# One command to scaffold a full resource
kashvi make:resource Post
kashvi migrate
kashvi run
```
