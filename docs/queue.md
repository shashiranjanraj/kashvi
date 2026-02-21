# Queue & Jobs

Kashvi's queue system (`pkg/queue`) supports background job processing with retry, backoff, and persistent failure tracking.

---

## Defining a Job

```go
// app/jobs/welcome_email_job.go
package jobs

type WelcomeEmailJob struct {
    UserID uint   `json:"user_id"`
    Email  string `json:"email"`
}

func (j WelcomeEmailJob) Handle() error {
    // send email...
    return mailer.Send(j.Email, "Welcome!", "welcome.html")
}
```

Register the job type at boot (so it can be deserialized):

```go
// In main.go init() or a jobs/register.go file:
queue.Register("jobs.WelcomeEmailJob", func() queue.Job {
    return &jobs.WelcomeEmailJob{}
})
```

---

## Dispatching Jobs

```go
import "github.com/shashiranjanraj/kashvi/pkg/queue"

// Immediate
queue.Dispatch(jobs.WelcomeEmailJob{UserID: user.ID, Email: user.Email})

// After a delay (5 minutes)
queue.DispatchAfter(jobs.WelcomeEmailJob{UserID: user.ID, Email: user.Email}, 5*time.Minute)
```

---

## Queue Drivers

### In-Memory (default — dev only)

Jobs are lost on restart. Good for development and testing.

```go
// Default — no configuration needed
queue.Dispatch(myJob)
```

### Redis Driver (production)

Jobs survive restarts. Delayed jobs use Redis sorted sets.

```go
// In server.go or a boot function, after cache.Connect():
import (
    "github.com/shashiranjanraj/kashvi/pkg/cache"
    "github.com/shashiranjanraj/kashvi/pkg/queue"
)

queue.SetDriver(queue.NewRedisDriver(cache.RDB))
```

Redis keys used:
- `kashvi:queue:jobs` — immediate job list (LPUSH/BRPOP)
- `kashvi:queue:delayed` — delayed job sorted set (score = Unix timestamp)

---

## Starting Workers

```bash
# From CLI (production)
kashvi queue:work --workers=5

# Or programmatically:
queue.StartWorkers(ctx, 5)
```

---

## Retry & Backoff

Failed jobs are automatically retried with linear backoff:
- Attempt 1 → wait 1s → Attempt 2 → wait 2s → Attempt 3

```go
// Change retry limit (default: 3)
queue.SetMaxRetry(5)
```

---

## Failed Jobs

After all retries are exhausted, the job is recorded in:
1. **In-memory** — available via `queue.FailedJobs()`
2. **Database** — `kashvi_failed_jobs` table (if `queue.UseDB()` is called)

The database persistence is wired automatically at server boot.

**Table structure:**

| Column | Type | Description |
|---|---|---|
| `id` | uint | Auto-increment PK |
| `job_type` | string | Go type name |
| `payload` | text | JSON-encoded job data |
| `error` | text | Last error message |
| `attempts` | int | Number of attempts made |
| `failed_at` | timestamp | When it failed |

**Querying failures:**

```go
// In memory
failed := queue.FailedJobs()
for _, f := range failed {
    fmt.Printf("%T failed after %d attempts: %v\n", f.Job, f.Attempts, f.Err)
}

// From DB
var records []queue.FailedJobRecord
database.DB.Order("failed_at desc").Find(&records)
```

---

## Full Example — Order Processing

```go
type ProcessOrderJob struct {
    OrderID uint `json:"order_id"`
}

func (j ProcessOrderJob) Handle() error {
    var order models.Order
    if err := database.DB.First(&order, j.OrderID).Error; err != nil {
        return err // will be retried
    }
    // charge card, update inventory, send confirmation...
    return nil
}

// In your controller:
func (c *OrderController) Store(ctx *appctx.Context) {
    // ... create order ...
    queue.Dispatch(ProcessOrderJob{OrderID: order.ID})
    ctx.Created(order)
}
```
