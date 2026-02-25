# MongoDB Log Storage

Kashvi can mirror all application logs to **MongoDB** in addition to stdout. The integration is:

- **Async** — writes never block the request path
- **Batched** — up to 50 documents per `InsertMany`
- **Graceful** — remaining records are flushed before the server exits
- **Optional** — leave `MONGO_URI` blank to stay stdout-only (zero overhead)

---

## Configuration

```ini
# .env
MONGO_URI=mongodb://localhost:27017   # required to enable; leave blank to disable
MONGO_LOG_DB=kashvi_logs              # default: kashvi_logs
MONGO_LOG_COLLECTION=app_logs         # default: app_logs
```

With a MongoDB Atlas cluster:

```ini
MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net/?retryWrites=true
```

---

## Document shape

Each log record in MongoDB:

```json
{
  "time":       "2026-02-25T12:00:00Z",
  "level":      "INFO",
  "msg":        "user registered",
  "request_id": "a1b2c3d4",
  "attrs": {
    "email": "user@example.com",
    "plan":  "pro"
  }
}
```

A `{time: -1}` index is created on startup for efficient querying.

---

## Querying logs

```js
// mongosh — last 100 errors
db.app_logs.find({ level: "ERROR" }).sort({ time: -1 }).limit(100)

// All logs from a specific request
db.app_logs.find({ request_id: "a1b2c3d4" })

// Logs from the last hour
db.app_logs.find({ time: { $gt: new Date(Date.now() - 3600_000) } })
```

---

## TTL (auto-delete old logs)

Add a TTL index in MongoDB to keep only N days of logs:

```js
db.app_logs.createIndex(
  { time: 1 },
  { expireAfterSeconds: 30 * 24 * 3600 }  // 30 days
)
```

---

## Graceful flush on shutdown

`logger.CloseMongoHandler()` is called automatically during `kashvi run` shutdown.
If you start the server manually, call it yourself:

```go
defer logger.CloseMongoHandler()
```

---

## Internal design

| Detail | Value |
|--------|-------|
| Channel buffer | 4096 records |
| Batch size | 50 documents per InsertMany |
| Flush ticker | Every 2 seconds |
| On queue full | Record silently dropped — logging never blocks |
| Connection pool | Max 10 MongoDB connections |
| Connect timeout | 5 seconds (falls back to stdout if unreachable) |

If MongoDB is unreachable at startup, Kashvi logs a warning to stdout and continues without MongoDB — it never fails to start.
