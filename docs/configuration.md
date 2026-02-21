# Configuration

Kashvi reads configuration from two sources, merged in order:

1. `config/app.json` — committed defaults
2. `.env` — local overrides (never commit this)

`.env` values always win over `config/app.json`.

---

## All Environment Variables

### Application

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `local` | `local` / `production` / `prod` |
| `APP_PORT` | `8080` | HTTP server port |
| `JWT_SECRET` | *(insecure default)* | **Must be changed in production** |
| `MAX_BODY_BYTES` | `4194304` (4 MB) | Max JSON request body size |

> [!CAUTION]
> The server **refuses to start** in production if `JWT_SECRET` is the default value.

---

### Database

| Variable | Default | Description |
|---|---|---|
| `DB_DRIVER` | `sqlite` | `sqlite` / `postgres` / `mysql` / `sqlserver` |
| `DATABASE_DSN` | `kashvi.db` | Full connection DSN |

**DSN examples:**
```ini
# SQLite (dev)
DATABASE_DSN=kashvi.db

# PostgreSQL
DATABASE_DSN=host=localhost user=postgres password=secret dbname=kashvi port=5432 sslmode=disable

# MySQL
DATABASE_DSN=root:secret@tcp(127.0.0.1:3306)/kashvi?charset=utf8mb4&parseTime=True&loc=Local
```

---

### Redis

| Variable | Default | Description |
|---|---|---|
| `REDIS_ADDR` | `localhost:6379` | Redis host:port |
| `REDIS_PASSWORD` | *(empty)* | Redis auth password |

> Redis is **non-fatal** — the server starts with a warning if Redis is unavailable and degrades gracefully (sessions won't persist, cache misses).

---

### Storage

| Variable | Default | Description |
|---|---|---|
| `STORAGE_DISK` | `local` | `local` or `s3` |
| `STORAGE_LOCAL_ROOT` | `storage` | Root directory for local disk |
| `STORAGE_URL` | `http://localhost:8080/storage` | Public URL for local files |

**S3 / MinIO / R2 / Spaces:**

| Variable | Default | Description |
|---|---|---|
| `S3_BUCKET` | *(required)* | Bucket name |
| `S3_REGION` | `us-east-1` | AWS region |
| `S3_KEY` | | Access key ID |
| `S3_SECRET` | | Secret access key |
| `S3_ENDPOINT` | | Custom endpoint (MinIO/R2 — leave empty for AWS) |
| `S3_URL` | | Public base URL (defaults to AWS URL pattern) |

---

## Reading Config in Code

```go
import "github.com/shashiranjanraj/kashvi/config"

port   := config.AppPort()      // "8080"
env    := config.AppEnv()       // "local"
secret := config.JWTSecret()
bucket := config.StorageS3Bucket()

// Generic getter with a default:
val := config.Get("MY_CUSTOM_VAR", "default-value")
```

---

## `config/app.json` Format

```json
{
  "app_env":      "local",
  "app_port":     "8080",
  "jwt_secret":   "change-me",
  "db_driver":    "sqlite",
  "database_dsn": "kashvi.db",
  "redis_addr":   "localhost:6379"
}
```

Keys in `app.json` map 1:1 to env variable names (lowercase, underscores).
