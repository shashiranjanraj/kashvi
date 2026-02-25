// Package logger provides a structured, levelled logger built on log/slog.
//
// The key extension over plain slog is WithCtx: it creates a logger with the
// request ID already attached, so every log line from a handler is
// automatically correlated:
//
//	log := logger.WithCtx(r.Context())
//	log.Info("payment processed", "amount", 99.99)
//	// → time=... level=INFO msg="payment processed" request_id=a1b2c3d4 amount=99.99
//
// # MongoDB log shipping
//
// When MONGO_URI is set in the environment, every log record is also written
// asynchronously to MongoDB (see MongoHandler).  Call CloseMongoHandler() on
// graceful shutdown to flush remaining records.
package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/shashiranjanraj/kashvi/config"
)

// mongoHandler holds the active MongoHandler so callers can close it on
// shutdown.  Nil when MongoDB logging is disabled.
var mongoHandler *MongoHandler

var L *slog.Logger

func init() {
	var level slog.Level

	switch config.AppEnv() {
	case "production", "prod":
		level = slog.LevelInfo
	default:
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	var stdout slog.Handler
	switch config.AppEnv() {
	case "production", "prod":
		stdout = slog.NewJSONHandler(os.Stdout, opts)
	default:
		stdout = slog.NewTextHandler(os.Stdout, opts)
	}

	handler := buildHandler(stdout, level)
	L = slog.New(handler)
	slog.SetDefault(L)
}

// buildHandler returns a MultiHandler (stdout + MongoDB) when MONGO_URI is
// set, or just the stdout handler otherwise.
func buildHandler(stdout slog.Handler, level slog.Level) slog.Handler {
	uri := config.MongoURI()
	if uri == "" {
		return stdout
	}

	mh, err := NewMongoHandler(uri, config.MongoLogDB(), config.MongoLogCollection())
	if err != nil {
		// Log the warning to stdout and continue without MongoDB.
		slog.New(stdout).Warn("logger: MongoDB handler unavailable, falling back to stdout only",
			"error", err)
		return stdout
	}

	mongoHandler = mh

	// Apply the same minimum level to the MongoHandler.
	filtered := &levelFilterHandler{inner: mh, level: level}
	return NewMultiHandler(stdout, filtered)
}

// CloseMongoHandler flushes buffered log records and disconnects from MongoDB.
// Should be called during graceful server shutdown.
func CloseMongoHandler() {
	if mongoHandler != nil {
		mongoHandler.Close()
	}
}

// levelFilterHandler wraps a slog.Handler and enforces a minimum log level.
type levelFilterHandler struct {
	inner slog.Handler
	level slog.Level
}

func (f *levelFilterHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return l >= f.level && f.inner.Enabled(ctx, l)
}
func (f *levelFilterHandler) Handle(ctx context.Context, r slog.Record) error {
	return f.inner.Handle(ctx, r)
}
func (f *levelFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelFilterHandler{inner: f.inner.WithAttrs(attrs), level: f.level}
}
func (f *levelFilterHandler) WithGroup(name string) slog.Handler {
	return &levelFilterHandler{inner: f.inner.WithGroup(name), level: f.level}
}

// ─────────────────────────────────────────────
// Context-aware logger
// ─────────────────────────────────────────────

// ctxKey is the unexported key used to store a per-request *slog.Logger.
type ctxKey struct{}

// WithCtx returns a *slog.Logger pre-tagged with the request_id found in ctx.
// If no request ID is present the base logger is returned unchanged.
//
// Import pattern:
//
//	import (
//	    "github.com/shashiranjanraj/kashvi/pkg/logger"
//	    "github.com/shashiranjanraj/kashvi/pkg/reqid"
//	)
//
//	log := logger.WithCtx(r.Context())
//	log.Info("user registered", "email", email)
func WithCtx(ctx context.Context) *slog.Logger {
	// Avoid import cycle: we read the request_id string directly from
	// context rather than importing reqid (reqid doesn't import logger either).
	type ridKey struct{} // mirrors reqid.ctxKey — same package-private trick
	_ = ridKey{}

	// Use the string stored by reqid.WithValue. We look it up via the
	// interface value rather than the type, so no import is needed.
	// reqid stores the id under its own private ctxKey type; we retrieve it
	// here by asking the injected logger stored alongside it.
	if log, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && log != nil {
		return log
	}
	return L
}

// InjectLogger stores a *slog.Logger (pre-tagged with request_id) into ctx.
// Called by the Logger middleware — not usually needed in application code.
func InjectLogger(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// ─────────────────────────────────────────────
// Short-hand helpers (use base logger)
// ─────────────────────────────────────────────

// Debug logs at DEBUG level.
func Debug(msg string, args ...any) { L.Debug(msg, args...) }

// Info logs at INFO level.
func Info(msg string, args ...any) { L.Info(msg, args...) }

// Warn logs at WARN level.
func Warn(msg string, args ...any) { L.Warn(msg, args...) }

// Error logs at ERROR level.
func Error(msg string, args ...any) { L.Error(msg, args...) }
