# Benchmarking Kashvi

This guide explains how to run and interpret benchmarks for the Kashvi framework.

---

## 1. Go benchmarks (unit / in-process)

Kashvi includes benchmark tests for the router, context, and validation. Run them with the standard Go tooling.

### Run all benchmarks

```bash
go test -bench=. -benchmem ./...
```

- `-bench=.` runs all benchmarks (regex: default is all).
- `-benchmem` reports memory allocations (B/op, allocs/op).

### Run benchmarks in a specific package

```bash
# Router + handler (raw and ctx.Wrap)
go test -bench=. -benchmem ./pkg/router/

# Context (Wrap, Success, pool)
go test -bench=. -benchmem ./pkg/ctx/

# Validation (struct-tag rules)
go test -bench=. -benchmem ./pkg/validate/
```

### Run a single benchmark

```bash
go test -bench=BenchmarkRouterWithCtx -benchmem ./pkg/router/
go test -bench=BenchmarkStructValid -benchmem ./pkg/validate/
```

### Run benchmarks for N seconds (smoother results)

```bash
go test -bench=. -benchmem -benchtime=3s ./pkg/router/
```

### Example output

```
pkg: github.com/shashiranjanraj/kashvi/pkg/router
BenchmarkRouterRawHandler-8          1234567    950 ns/op    320 B/op    5 allocs/op
BenchmarkRouterWithParam-8          1000000   1200 ns/op    400 B/op    6 allocs/op
BenchmarkRouterWithCtx-8             800000   1500 ns/op    512 B/op    8 allocs/op
```

- **ns/op** — nanoseconds per operation (lower is better).
- **B/op** — bytes allocated per operation (lower is better).
- **allocs/op** — number of allocations per operation (lower is better).

---

## 2. Available benchmarks

| Package     | Benchmark                 | What it measures |
|------------|---------------------------|-------------------|
| `pkg/router` | `BenchmarkRouterRawHandler`  | Route match + raw `http.HandlerFunc` |
| `pkg/router` | `BenchmarkRouterWithParam`   | Route match with path param `/users/{id}` |
| `pkg/router` | `BenchmarkRouterWithCtx`     | Route match + `ctx.Wrap` handler + JSON response |
| `pkg/ctx`    | `BenchmarkWrap`              | Context pool acquire/release + `c.Success` (JSON) |
| `pkg/ctx`    | `BenchmarkWrapWithParam`     | Same + `c.Param("id")` (no router) |
| `pkg/validate` | `BenchmarkStructValid`     | `validate.Struct` on valid input (all rules pass) |
| `pkg/validate` | `BenchmarkStructInvalid`   | `validate.Struct` on invalid input (required failures) |

---

## 3. End-to-end load test (real server + HTTP client)

To measure throughput and latency of a running server (including all middleware and real I/O), use a load-test tool against a live process.

### Start the server

From a Kashvi app (or the framework’s `cmd/server` with minimal routes):

```bash
go run . serve
# or
kashvi serve
```

Server will listen on the port from config (e.g. `:8080`).

### Using `hey`

Install and run:

```bash
go install github.com/rakyll/hey@latest

# 10,000 requests, 200 concurrent
hey -n 10000 -c 200 http://localhost:8080/

# With a JSON POST body
hey -n 5000 -c 100 -m POST -d '{"name":"test"}' -H "Content-Type: application/json" http://localhost:8080/api/products
```

Example output:

```
Summary:
  Total:        2.3456 secs
  Slowest:      0.0523 secs
  Fastest:      0.0001 secs
  Average:      0.0045 secs
  Requests/sec: 4263.1234
```

### Using `wrk`

```bash
# Install (macOS: brew install wrk)
wrk -t4 -c200 -d30s http://localhost:8080/
```

- `-t4` — 4 threads.
- `-c200` — 200 connections.
- `-d30s` — duration 30 seconds.

### Using `ab` (Apache Bench)

```bash
ab -n 10000 -c 200 http://localhost:8080/
```

---

## 4. Tips

- **Compare before/after** — Run the same benchmarks before and after changes to spot regressions.
- **Use `-benchtime`** — e.g. `-benchtime=3s` for more stable numbers.
- **Reduce noise** — Close other apps; run on a quiet machine; use `go test -count=3` and compare.
- **Profile** — For CPU: `go test -bench=. -cpuprofile=cpu.out ./pkg/router` then `go tool pprof cpu.out`. For memory: `-memprofile=mem.out` and `go tool pprof mem.out`.
