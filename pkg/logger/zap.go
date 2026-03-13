package logger

import (
	"os"

	"github.com/shashiranjanraj/kashvi/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger is a wrapper around zap.SugaredLogger that implements our Logger interface.
type ZapLogger struct {
	sugar *zap.SugaredLogger
}

func init() {
	L = NewZapDefault()
}

// NewZapDefault initializes a default Zap logger based on env configs.
func NewZapDefault() Logger {
	var level zapcore.Level
	switch config.LogLevel() {
	case "debug":
		level = zap.DebugLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if config.AppEnv() == "production" || config.AppEnv() == "prod" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		// Use colored console output for local development
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)

	// Inject MongoDB logging if configured
	uri := config.MongoURI()
	if uri != "" {
		mongoCore, err := NewMongoZapCore(level)
		if err == nil {
			core = zapcore.NewTee(core, mongoCore)
		} else {
			// Fallback strictly to stdout
			core = zapcore.NewTee(core)
		}
	}

	zapLogger := zap.New(core)
	return &ZapLogger{sugar: zapLogger.Sugar()}
}

func (l *ZapLogger) Debug(msg string, args ...interface{}) { l.sugar.Debugw(msg, args...) }
func (l *ZapLogger) Info(msg string, args ...interface{})  { l.sugar.Infow(msg, args...) }
func (l *ZapLogger) Warn(msg string, args ...interface{})  { l.sugar.Warnw(msg, args...) }
func (l *ZapLogger) Error(msg string, args ...interface{}) { l.sugar.Errorw(msg, args...) }
func (l *ZapLogger) Sync() error                           { return l.sugar.Sync() }

func (l *ZapLogger) With(args ...interface{}) Logger {
	return &ZapLogger{sugar: l.sugar.With(args...)}
}
