# Storage

`pkg/storage` provides a unified file-storage API inspired by Laravel's Storage facade.
Switch between local disk and S3-compatible storage with a single env variable.

---

## Configuration

```ini
STORAGE_DISK=local      # default driver: "local" or "s3"
```

---

## Using the Default Disk

```go
import "github.com/shashiranjanraj/kashvi/pkg/storage"

// Write
storage.Put("avatars/user-1.jpg", imageBytes)
storage.PutStream("uploads/file.pdf", r.Body)

// Read
data, err := storage.Get("avatars/user-1.jpg")
stream, err := storage.GetStream("uploads/file.pdf")
defer stream.Close()

// Metadata
exists  := storage.Exists("avatars/user-1.jpg")
missing := storage.Missing("avatars/user-1.jpg")
size, _ := storage.Size("avatars/user-1.jpg")
modTime, _ := storage.LastModified("avatars/user-1.jpg")

// Public URL
url := storage.URL("avatars/user-1.jpg")

// Delete
storage.Delete("avatars/user-1.jpg")

// Copy / Move
storage.Copy("tmp/upload.jpg", "images/final.jpg")
storage.Move("tmp/upload.jpg", "archive/old.jpg")

// Directories
files, _ := storage.Files("avatars")          // non-recursive
all, _   := storage.AllFiles("avatars")       // recursive
dirs, _  := storage.Directories("uploads")
storage.MakeDirectory("exports")
storage.DeleteDirectory("tmp")
```

---

## Using a Specific Disk

```go
// Use S3 explicitly
storage.Use("s3").Put("backups/db.sql.gz", data)

// Use local disk explicitly
storage.Use("local").Get("cache/data.json")
```

> Method name is `Use()` (not `Disk()`) to avoid conflict with the `Disk` interface type.

---

## File Upload Handler

```go
func (c *UploadController) Store(ctx *appctx.Context) {
    ctx.R.ParseMultipartForm(10 << 20) // 10MB max

    file, header, err := ctx.R.FormFile("file")
    if err != nil {
        ctx.Error(400, "No file uploaded")
        return
    }
    defer file.Close()

    path := fmt.Sprintf("uploads/%d_%s", time.Now().Unix(), header.Filename)
    if err := storage.PutStream(path, file); err != nil {
        ctx.Error(500, "Upload failed")
        return
    }

    ctx.Created(map[string]any{
        "path": path,
        "url":  storage.URL(path),
    })
}
```

---

## Local Disk

Files are stored relative to `STORAGE_LOCAL_ROOT` (default: `./storage`).

Public access: `GET /storage/{path}` is automatically mounted when `STORAGE_DISK=local`.

```ini
STORAGE_LOCAL_ROOT=storage
STORAGE_URL=http://localhost:8080/storage
```

---

## S3 / AWS

```ini
STORAGE_DISK=s3
S3_BUCKET=my-bucket
S3_REGION=us-east-1
S3_KEY=AKIAIOSFODNN7EXAMPLE
S3_SECRET=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
S3_URL=https://my-bucket.s3.us-east-1.amazonaws.com
```

---

## MinIO (self-hosted S3)

Run locally with Docker:

```bash
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"
```

```ini
STORAGE_DISK=s3
S3_BUCKET=my-bucket
S3_KEY=minioadmin
S3_SECRET=minioadmin
S3_ENDPOINT=http://localhost:9000
S3_REGION=us-east-1
```

Create the bucket at `http://localhost:9001` (MinIO console UI).

---

## Cloudflare R2 / DigitalOcean Spaces

Same as MinIO — just set `S3_ENDPOINT` to your provider's endpoint URL.

```ini
# Cloudflare R2
S3_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com

# DigitalOcean Spaces
S3_ENDPOINT=https://nyc3.digitaloceanspaces.com
```

---

## Custom Driver

Implement the `Disk` interface and register it:

```go
type MyDriver struct{}
func (d *MyDriver) Put(path string, content []byte) error { ... }
// ... implement all 16 Disk interface methods

// Register at boot:
storage.RegisterDisk("mydriver", &MyDriver{})

// Use:
storage.Use("mydriver").Put("file.txt", data)
```
