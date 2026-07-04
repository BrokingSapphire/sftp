package handlers

import (
	"context"
	"errors"

	"github.com/go-fuego/fuego"
)

// debugErrors toggles whether error detail is surfaced to clients.
var debugErrors bool

// SetDebugErrors enables/disables detailed error output (dev only).
func SetDebugErrors(enabled bool) { debugErrors = enabled }

// ErrorHandler is Fuego's global error handler. In debug mode it copies the
// wrapped cause into the problem `detail`; in production it stays minimal.
func ErrorHandler(ctx context.Context, err error) error {
	err = fuego.ErrorHandler(ctx, err)
	if !debugErrors {
		return err
	}
	var he fuego.HTTPError
	if errors.As(err, &he) && he.Detail == "" && he.Err != nil {
		he.Detail = he.Err.Error()
		return he
	}
	return err
}
