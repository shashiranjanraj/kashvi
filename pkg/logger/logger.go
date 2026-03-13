package logger

import (
	"context"
)

// Logger is the generic interface for logging in Kashvi.
// It accepts alternating key-value pairs in args...
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	
	With(args ...interface{}) Logger
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

func Debug(msg string, args ...interface{}) { L.Debug(msg, args...) }
func Info(msg string, args ...interface{})  { L.Info(msg, args...) }
func Warn(msg string, args ...interface{})  { L.Warn(msg, args...) }
func Error(msg string, args ...interface{}) { L.Error(msg, args...) }
func Sync() error                           { return L.Sync() }
