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
