package logger

import "context"

type loggerContextKey struct{}

// NewContext returns a copy of ctx carrying the given Logger.
func NewContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, l)
}

// FromContext returns the Logger stored in ctx, or fallback if none.
func FromContext(ctx context.Context, fallback Logger) Logger {
	if l, ok := ctx.Value(loggerContextKey{}).(Logger); ok {
		return l
	}
	return fallback
}
