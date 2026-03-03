# Building a User CRUD API in Kashvi (5 Minutes)

Welcome to Kashvi! If you're a fresher or transitioning from PHP/Laravel, you'll feel right at home. We're going to build a fully functional `User` API in just a few minutes. 

No advanced features (like WebSockets or gRPC) here—just standard, clean RESTful architecture.

---

## 1. Create the Project
First, scaffold a fresh project. This creates a ready-to-use folder structure for you.

```bash
kashvi new my-api
cd my-api
```

*(This command generates your `main.go`, `app/` folder for logic, `database/` for schemas, and a `.env` file pre-configured for SQLite so you don't need any complex database setup yet).*

---

## 2. Generate the Resource
We need a Model (to represent the user), a Controller (to handle HTTP requests), a Migration (to create the database table), and a Seeder (to add dummy data).

Instead of creating these manually, Kashvi's CLI does it in one command:

```bash
kashvi make:resource User
```

**What this did:**
* `app/models/user.go`: Your data structure.
* `app/controllers/user_controller.go`: Where your API logic lives.
* `database/migrations/xxxx_create_users_table.go`: Instructions for creating the database table.
* `database/seeders/user_seeder.go`: A place to create fake users for testing.

---

## 3. Define the Database Table
Let's tell the database what a "User" looks like. Open the newly generated migration file inside the `database/migrations/` folder.

Add the `name` and `email` columns:

```go
// database/migrations/xxxx_create_users_table.go

func (m *Migration) Up() {
    table := m.CreateTable("users")
    table.String("name").NotNull()
    table.String("email").Unique().NotNull()
}
```

Now, run the migration to actually create the table in your SQLite database:
```bash
kashvi migrate
```

---

## 4. Write the Controller Logic
Open `app/controllers/user_controller.go`. We'll write the logic for creating a new user (the `Store` method).

Kashvi has built-in JSON validation. If the user doesn't send a name, we automatically reject the request!

```go
// app/controllers/user_controller.go
package controllers

import (
    "github.com/shashiranjanraj/kashvi/pkg/ctx"
    "my-api/app/models"
    "my-api/database" // Assuming you export your db connection here
)

func (c *UserController) Store(ctx *ctx.Context) {
    // 1. Define what JSON we expect and add validation rules!
    var input struct {
        Name  string `json:"name" validate:"required,min=2"`
        Email string `json:"email" validate:"required,email"`
    }

    // 2. Bind and Validate. If it fails, Kashvi automatically sends a 422 Error to the client.
    if !ctx.BindJSON(&input) { 
        return 
    }

    // 3. Save to database
    user := models.User{Name: input.Name, Email: input.Email}
    database.DB.Create(&user)
    
    // 4. Send success response (201 Created)
    ctx.Created(user)
}
```

---

## 5. Register the Route
We need to map a URL (like `POST /api/users`) to the controller we just wrote. 

Open `app/routes/api.go` and add this inside the `RegisterRoutes` function:

```go
// app/routes/api.go
import "my-api/app/controllers"

func RegisterAPI(r *router.Router) {
    api := r.Group("/api")
    
    // Initialize our controller
    userCtrl := controllers.NewUserController()

    // Map the POST request to the Store function
    api.Post("/users", "users.store", ctx.Wrap(userCtrl.Store))
}
```

---

## 6. Run the Server
You're done coding! Let's start the server.

```bash
kashvi serve
```
*You should see a message saying your server is running on port 8080.*

---

## 7. Test It Out
Open your terminal and run this `curl` command (or use Postman) to create a user:

```bash
curl -X POST http://localhost:8080/api/users \
     -H "Content-Type: application/json" \
     -d '{"name": "Rahul", "email": "rahul@example.com"}'
```

**Success Response:**
```json
{
  "name": "Rahul",
  "email": "rahul@example.com"
}
```

**What if we forget the email? (Validation Test)**
```bash
curl -X POST http://localhost:8080/api/users \
     -H "Content-Type: application/json" \
     -d '{"name": "Rahul"}'
```

**Error Response:**
```json
{
  "error": "validation failed",
  "details": {
    "email": "email is required"
  }
}
```

### 🎉 Congratulations!
You just built a production-grade, validated Go API without the confusing boilerplate. Welcome to Kashvi!
