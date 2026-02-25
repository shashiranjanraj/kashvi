# gRPC Server

Kashvi includes a production-ready gRPC server that runs **alongside** the HTTP server on a separate port. It ships with a health-check service, server reflection, and pre-wired Prometheus metrics.

---

## Configuration

```ini
# .env
GRPC_PORT=9090    # default: 9090
```

---

## What starts automatically

When you run `kashvi run`, **both** servers boot:

```
🚀 Kashvi HTTP  on :8080  [env: local]  [workers: 8]
🔌 Kashvi gRPC  on :9090
```

At shutdown (`Ctrl+C`), the gRPC server drains in-flight RPCs before exiting.

---

## Built-in interceptors (applied automatically)

| Order | Interceptor | What it does |
|-------|-------------|--------------|
| 1 | **Recovery** | Catches panics → returns `INTERNAL` status instead of crashing |
| 2 | **Logging** | Logs every RPC: `method`, `duration_ms`, `code` |
| 3 | **Prometheus** | `grpc_server_handled_total`, `grpc_server_handling_seconds` |

---

## Built-in services

### Health (grpc.health.v1.Health)

Always returns `SERVING`. Test with:

```bash
# brew install grpcurl
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
# → { "status": "SERVING" }
```

### Server Reflection

Enabled automatically — `grpcurl` works without proto files:

```bash
grpcurl -plaintext localhost:9090 list
# → grpc.health.v1.Health
```

---

## Registering your own service

```go
// pkg/grpc/server.go  — add after reflection.Register(srv)
mypb.RegisterUserServiceServer(srv, &UserServiceImpl{})
```

Or call `grpc.Start()` manually and register before the goroutine runs:

```go
grpcSrv, lis, _ := kashvigrpc.Start(config.GRPCPort())
mypb.RegisterUserServiceServer(grpcSrv, &UserServiceImpl{})
```

---

## Standalone gRPC server (CLI)

Run the gRPC server without the HTTP server:

```bash
kashvi grpc:serve
```

---

## Adding a custom interceptor

Edit `pkg/grpc/server.go` — add to `chainUnary(...)`:

```go
grpc.NewServer(
    grpc.UnaryInterceptor(
        chainUnary(
            recoveryInterceptor,
            loggingInterceptor,
            metricsInterceptor,
            myAuthInterceptor,  // ← add here
        ),
    ),
)
```

---

## Prometheus metrics

The gRPC metrics are available on the existing `/metrics` endpoint alongside HTTP metrics:

```
grpc_server_handled_total{grpc_method="/grpc.health.v1.Health/Check", grpc_code="OK"} 7
grpc_server_handling_seconds_bucket{grpc_method="...", le="0.01"} 7
```
