# Worker Pool

`pkg/workerpool` provides a **bounded goroutine pool** that limits concurrent goroutine creation under high load. Use it for CPU-intensive or I/O-heavy tasks that should not run in unbounded goroutines.

---

## Why use a pool?

| Approach | Problem |
|----------|---------|
| `go doWork()` for every request | Goroutines spike unboundedly under load — OOM risk |
| Worker pool | Hard ceiling on concurrency — predictable memory |

---

## Configuration

```ini
# .env
WORKER_POOL_SIZE=50   # default: 50
```

---

## Basic usage

```go
import "github.com/shashiranjanraj/kashvi/pkg/workerpool"

// Create a pool (use config.WorkerPoolSize() for env-driven size)
pool := workerpool.New(config.WorkerPoolSize())
defer pool.Shutdown()

// Non-blocking submit
err := pool.Submit(func() {
    processImage(imageData)
})
if errors.Is(err, workerpool.ErrPoolFull) {
    // Pool is busy — return 429, push to queue, etc.
    c.JSON(http.StatusTooManyRequests, map[string]string{"error": "server busy"})
    return
}
```

---

## Blocking submit

When you want to wait until a slot is available:

```go
err := pool.SubmitWait(func() {
    sendReportEmail(userID)
})
if errors.Is(err, workerpool.ErrPoolClosed) {
    // Pool was shut down
}
```

---

## Shutdown

`Shutdown()` stops accepting new tasks, waits for all in-flight tasks to complete, then releases all worker goroutines. Safe to call multiple times.

```go
pool.Shutdown()
```

---

## Error reference

| Error | When |
|-------|------|
| `workerpool.ErrPoolFull` | All workers are busy and the queue buffer is full |
| `workerpool.ErrPoolClosed` | `Shutdown()` has been called |

---

## Panic safety

Workers recover from panics automatically — a bad task never kills the pool or unexpectedly terminates a goroutine. The next task runs as normal.

---

## Sizing guide

| Use case | Recommended size |
|----------|-----------------|
| Image processing | `runtime.NumCPU()` |
| Network I/O (external APIs) | 50–200 |
| DB queries | 20–50 (limited by DB connection pool) |
| Mixed workloads | `WORKER_POOL_SIZE=50` (default) |

---

## Buffer size

The internal task queue buffer is `2 × size`. This absorbs short bursts without returning `ErrPoolFull`. For example, a pool of 50 workers can buffer 100 pending tasks before backpressure kicks in.

---

## Integration with HTTP handlers

A good pattern: create one shared pool at app startup and use it across handlers.

```go
// internal/kernel/http.go
var Pool = workerpool.New(config.WorkerPoolSize())

// In a handler
func GenerateReport(c *ctx.Context) {
    err := kernel.Pool.Submit(func() {
        report := buildReport(c.ParamInt("id"))
        cache.Set("report:"+id, report, time.Hour)
    })
    if errors.Is(err, workerpool.ErrPoolFull) {
        c.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
        return
    }
    c.JSON(http.StatusAccepted, map[string]string{"status": "processing"})
}
```
