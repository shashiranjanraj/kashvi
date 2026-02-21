# Migrations & Seeders

## Creating a Migration

```bash
kashvi make:migration create_posts_table
```

Edit the generated file:

```go
package migrations

import (
    "github.com/shashiranjanraj/kashvi/pkg/migration"
    "github.com/shashiranjanraj/kashvi/app/models"
    "gorm.io/gorm"
)

func init() {
    migration.Register("20260221_create_posts_table", &M_CreatePostsTable{})
}

type M_CreatePostsTable struct{}

func (m *M_CreatePostsTable) Up(db *gorm.DB) error {
    return db.AutoMigrate(&models.Post{})
}

func (m *M_CreatePostsTable) Down(db *gorm.DB) error {
    return db.Migrator().DropTable("posts")
}
```

## Running Migrations

```bash
kashvi migrate              # run all pending
kashvi migrate:rollback     # rollback last batch
kashvi migrate:status       # show status
```

## Seeders

```bash
kashvi make:seeder PostSeeder
```

```go
func PostSeeder(db *gorm.DB) error {
    posts := []models.Post{
        {Title: "Hello World", Body: "First post!", Published: true},
    }
    return db.Create(&posts).Error
}
```

Register in `database/seeders/run_all.go`:

```go
func RunAll(db *gorm.DB) error {
    for _, seeder := range []func(*gorm.DB) error{
        UserSeeder,
        PostSeeder,
    } {
        if err := seeder(db); err != nil {
            return err
        }
    }
    return nil
}
```

```bash
kashvi seed
```
