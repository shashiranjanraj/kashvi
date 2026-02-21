# CLI Reference

All commands are run via the `kashvi` binary. Install with `make install`.

---

## Server Commands

### `kashvi run`
Start the HTTP server. Boots DB, Redis, storage, then listens forever until SIGINT/SIGTERM.

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
kashvi queue:work           # default: 3 workers
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

All scaffold commands create files in your project. They will **not overwrite** existing files.

### `kashvi make:resource [Name]`
**Most useful command.** Scaffolds a complete CRUD resource in one shot.

```bash
kashvi make:resource Post
```

Creates:
- `app/models/post.go`
- `app/controllers/post_controller.go` (full CRUD using `ctx.Context`)
- `app/services/postService_service.go`
- `database/migrations/TIMESTAMP_create_posts_table.go`
- `database/seeders/post_seeder.go`

And prints the exact route lines to add to `api.go`.

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
# Creates: database/seeders/postseeder.go
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
