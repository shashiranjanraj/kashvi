# ORM & Database

Kashvi wraps GORM with a fluent chainable query builder in `pkg/orm`.

---

## Connection

The database is connected at server boot. Configure via `.env`:

```ini
DB_DRIVER=postgres
DATABASE_DSN=host=localhost user=postgres dbname=kashvi sslmode=disable
```

Use `database.DB` to get the GORM instance anywhere:

```go
import "github.com/shashiranjanraj/kashvi/pkg/database"

database.DB.Create(&user)
```

---

## Basic CRUD

```go
import (
    "github.com/shashiranjanraj/kashvi/pkg/database"
    "github.com/shashiranjanraj/kashvi/pkg/orm"
)

q := orm.New(database.DB)

// Create
q.Create(&models.Post{Title: "Hello", Body: "World"})

// Find by ID
var post models.Post
q.Find(&post, 1)

// Update
q.Where("id = ?", 1).Update(&models.Post{}, map[string]any{"title": "Updated"})

// Delete
q.Where("id = ?", 1).Delete(&models.Post{})
```

---

## Query Builder

```go
q := orm.New(database.DB)

// Filtering
q.Where("status = ?", "active").
  Where("created_at > ?", time.Now().AddDate(0, -1, 0))

// Ordering & limiting
q.OrderBy("created_at DESC").Limit(10).Offset(20)

// Select specific columns
q.Select("id", "title", "created_at")

// Eager loading
q.With("Author", "Tags")

// Execute
var posts []models.Post
q.Get(&posts)
```

---

## Pagination

```go
func (ctrl *PostController) Index(c *appctx.Context) {
    var posts []models.Post

    pagination, err := orm.New(database.DB).
        Where("published = ?", true).
        OrderBy("created_at DESC").
        Paginate(&posts, c.R)  // reads ?page=1&per_page=15 from request

    if err != nil {
        c.Error(500, "Failed to fetch posts")
        return
    }

    response.Paginated(c.W, posts, pagination)
}
```

Response:
```json
{
  "status": 200,
  "data": {
    "items": [...],
    "pagination": {
      "total": 150,
      "per_page": 15,
      "current_page": 1,
      "last_page": 10,
      "from": 1,
      "to": 15
    }
  }
}
```

---

## Parallel Queries

Run multiple queries concurrently and wait for all results:

```go
var users []models.User
var posts []models.Post
var tags  []models.Tag

orm.Parallel(
    func() { database.DB.Find(&users) },
    func() { database.DB.Where("published = ?", true).Find(&posts) },
    func() { database.DB.Find(&tags) },
)

// All three queries ran simultaneously
```

---

## ORM Cache Bridge

Cache query results in Redis automatically:

```go
var user models.User
orm.New(database.DB).
    Cache("user:42", 5*time.Minute).
    Find(&user, 42)
// Second call hits Redis, not the DB
```

---

## Models

Define models in `app/models/`:

```go
package models

import "gorm.io/gorm"

type Post struct {
    gorm.Model          // ID, CreatedAt, UpdatedAt, DeletedAt
    Title     string    `gorm:"size:255;not null"`
    Body      string    `gorm:"type:text"`
    Published bool      `gorm:"default:false"`
    UserID    uint
    User      User      // belongs to
    Tags      []Tag     `gorm:"many2many:post_tags;"`
}
```

---

## Raw Queries

```go
var result []map[string]any
database.DB.Raw("SELECT id, title FROM posts WHERE published = ?", true).Scan(&result)
```

---

## Connection Pool Settings (auto-configured)

| Setting | Value |
|---|---|
| Max open connections | 25 |
| Max idle connections | 10 |
| Max conn lifetime | 5 minutes |
| Max idle time | 2 minutes |
