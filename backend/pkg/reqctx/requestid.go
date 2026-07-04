// Package reqctx stores request-scoped values (correlation ID) on context.
package reqctx

import "context"

type requestIDKey struct{}

// NewContextWithRequestID returns a copy of ctx carrying the request ID.
func NewContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// GetRequestID returns the request ID stored in ctx, or "" if none.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
