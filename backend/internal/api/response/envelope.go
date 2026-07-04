// Package response defines the uniform success envelope returned by all
// JSON endpoints. Errors are returned separately as RFC 7807 problem+json
// (see internal/api/handlers).
package response

import (
	"context"
	"time"

	"github.com/go-fuego/fuego"

	"sapphirebroking.com/sftp_service/pkg/reqctx"
)

const defaultSuccessMessage = "Request completed successfully"

// Any is a convenience alias for handlers that return no data payload.
type Any interface{}

// Envelope is the standard success wrapper. Fuego serialises it and calls
// OutTransform to stamp request_id and timestamp automatically.
type Envelope[T any] struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Data      T      `json:"data,omitempty"`
	Meta      any    `json:"meta,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// OK wraps data with the default success message.
func OK[T any](data T) *Envelope[T] {
	return &Envelope[T]{Success: true, Message: defaultSuccessMessage, Data: data}
}

// OKWithMessage wraps data with a custom message.
func OKWithMessage[T any](data T, message string) *Envelope[T] {
	return &Envelope[T]{Success: true, Message: message, Data: data}
}

// Paginated wraps data with pagination metadata.
func Paginated[T any](data T, meta any) *Envelope[T] {
	return &Envelope[T]{Success: true, Message: defaultSuccessMessage, Data: data, Meta: meta}
}

// OutTransform stamps request-scoped fields just before serialisation.
func (e *Envelope[T]) OutTransform(ctx context.Context) error {
	if e.RequestID == "" {
		e.RequestID = reqctx.GetRequestID(ctx)
	}
	if e.Timestamp == 0 {
		e.Timestamp = time.Now().UnixMilli()
	}
	return nil
}

var _ fuego.OutTransformer = (*Envelope[struct{}])(nil)
