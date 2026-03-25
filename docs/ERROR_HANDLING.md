# Error handling in Kashvi

Kashvi uses Go’s standard error model end to end: wrapped errors, `Unwrap`, and the helpers **`errors.Is`** and **`errors.As`**. HTTP handlers can return typed errors via **`pkg/apperror`**, which the router turns into JSON responses.

---

## 1. `apperror.Error` (HTTP-shaped errors)

Structured errors live in `github.com/shashiranjanraj/kashvi/pkg/apperror`. They carry an HTTP status, a safe client message, and an optional internal cause for logs.

```go
return apperror.NotFound("User not found")
return apperror.Internal(err) // wraps err; client still gets a safe 500 message
```

`(*apperror.Error).Unwrap()` returns the wrapped `Err`, so **`errors.Is` / `errors.As` work through `apperror`**.

---

## 2. When to use `errors.Is`

Use **`errors.Is(err, target)`** to test for a **specific sentinel error** anywhere in the chain (including wrapped errors from `fmt.Errorf("… %w", …)` or multiple errors joined with `errors.Join`).

```go
if errors.Is(err, sql.ErrNoRows) {
    return apperror.NotFound("Not found")
}
```

---

## 3. When to use `errors.As`

Use **`errors.As(err, &ptr)`** to extract a **typed error** (e.g. `*fs.PathError`, a domain error type, or `*apperror.Error`).

```go
var appErr *apperror.Error
if errors.As(err, &appErr) {
    // use appErr.StatusCode, appErr.Message
}
```

Kashvi’s **`apperror.TryConvert`** uses `errors.As` so that an `*apperror.Error` **wrapped** in another error is still recognized (for example when `router.Wrap` converts handler errors to HTTP responses).

---

## 4. Handlers that return `error`: `router.Wrap`

For handlers with signature `func(w http.ResponseWriter, r *http.Request) error`, register with **`router.Wrap`**. Returned errors are passed through **`apperror.TryConvert`**, then mapped to the correct HTTP response.

Prefer returning `apperror.*` for predictable status codes; any other error becomes a 500 with logging.

---

## 5. Multiple errors (`errors.Join`)

If you aggregate failures with **`errors.Join(err1, err2, …)`**, both **`errors.Is`** and **`errors.As`** still walk the joined chain. Prefer wrapping a single cause with `%w` when you only need one primary error for the client.

---

## 6. Quick reference

| Goal | API |
|------|-----|
| Compare to sentinel | `errors.Is(err, io.EOF)` |
| Extract typed error | `errors.As(err, &pe)` |
| HTTP error with status | `apperror.BadRequest("…")`, `NotFound`, `Internal`, … |
| Wrap for unwrap chain | `fmt.Errorf("context: %w", err)` |

See also: [REQUEST_FLOW.md](REQUEST_FLOW.md) (where handler errors are turned into responses).
