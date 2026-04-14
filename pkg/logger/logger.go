package logger

import (
	"context"
)

// Logger is the generic interface for logging in Kashvi.
// It accepts alternating key-value pairs in args...
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	With(args ...any) Logger
	Sync() error
}

// L is the global default logger instance.
var L Logger

// ─────────────────────────────────────────────
// Context-aware logger
// ─────────────────────────────────────────────

type ctxKey struct{}

// WithCtx returns a Logger pre-tagged with the request_id from context.
func WithCtx(ctx context.Context) Logger {
	type ridKey struct{}
	_ = ridKey{}

	if log, ok := ctx.Value(ctxKey{}).(Logger); ok && log != nil {
		return log
	}
	return L
}

// InjectLogger stores a Logger into ctx.
func InjectLogger(ctx context.Context, log Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// ─────────────────────────────────────────────
// Short-hand helpers (use base logger)
// ─────────────────────────────────────────────

func Debug(msg string, args ...any) { L.Debug(msg, args...) }
func Info(msg string, args ...any)  { L.Info(msg, args...) }
func Warn(msg string, args ...any)  { L.Warn(msg, args...) }
func Error(msg string, args ...any) { L.Error(msg, args...) }
func Sync() error                   { return L.Sync() }
