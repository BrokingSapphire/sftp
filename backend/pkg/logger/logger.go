// Package logger provides a structured, sink-based logger built on zap.
// It exposes a small interface so the rest of the codebase never imports
// zap directly, and supports multiple output sinks (stdout, rotating file).
package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/reqctx"
)

// Logger is the structured logging interface used across the service.
// Fields are passed as alternating key/value pairs: log.Info("msg", "k", v).
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})

	With(fields ...interface{}) Logger
	WithContext(ctx context.Context) Logger
	Named(name string) Logger
	Sync() error
}

// SimpleLogger is the zap-backed implementation of Logger.
type SimpleLogger struct {
	logger *zap.Logger
}

// BuildLogger tees the given sinks into a single Logger.
func BuildLogger(cfg *LoggingConfig, sinks []Sink) (Logger, error) {
	enc := buildEncoder(cfg.Format)

	cores := make([]zapcore.Core, 0, len(sinks))
	for _, s := range sinks {
		cores = append(cores, s.Core(enc))
	}

	opts := []zap.Option{zap.AddCallerSkip(1)}
	if cfg.EnableCaller {
		opts = append(opts, zap.AddCaller())
	}
	if cfg.EnableStackTrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	z := zap.New(zapcore.NewTee(cores...), opts...)
	return &SimpleLogger{logger: z}, nil
}

// ParseLevel converts a level string to a zapcore.Level (default: info).
func ParseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	l.logger.Debug(msg, l.convertFields(fields...)...)
}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	l.logger.Info(msg, l.convertFields(fields...)...)
}

func (l *SimpleLogger) Warn(msg string, fields ...interface{}) {
	l.logger.Warn(msg, l.convertFields(fields...)...)
}

func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	l.logger.Error(msg, l.convertFields(fields...)...)
}

func (l *SimpleLogger) Fatal(msg string, fields ...interface{}) {
	l.logger.Fatal(msg, l.convertFields(fields...)...)
	os.Exit(1)
}

func (l *SimpleLogger) With(fields ...interface{}) Logger {
	return &SimpleLogger{logger: l.logger.With(l.convertFields(fields...)...)}
}

// WithContext derives a child logger enriched with request-scoped fields
// (request_id, authenticated subject) pulled from the context.
func (l *SimpleLogger) WithContext(ctx context.Context) Logger {
	child := l.logger
	if id := reqctx.GetRequestID(ctx); id != "" {
		child = child.With(zap.String("request_id", id))
	}
	if claims := jwt.GetClaimsFromContext(ctx); claims != nil && claims.Sub != nil {
		child = child.With(zap.String("user_sub", *claims.Sub))
	}
	return &SimpleLogger{logger: child}
}

func (l *SimpleLogger) Named(name string) Logger {
	return &SimpleLogger{logger: l.logger.Named(name)}
}

func (l *SimpleLogger) convertFields(fields ...interface{}) []zap.Field {
	if len(fields)%2 != 0 {
		fields = append(fields, "")
	}
	zapFields := make([]zap.Field, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		zapFields = append(zapFields, zap.Any(key, fields[i+1]))
	}
	return zapFields
}

func (l *SimpleLogger) Sync() error { return l.logger.Sync() }

// NewNop returns a Logger that discards all output. Use in tests.
func NewNop() Logger {
	return &SimpleLogger{logger: zap.NewNop()}
}
