// Package utils holds small cross-cutting helpers (validation, etc.).
package utils

import (
	"github.com/go-playground/validator/v10"
)

// validate is a process-wide validator instance (safe for concurrent use).
var validate = validator.New(validator.WithRequiredStructEnabled())

// Validate runs struct validation using `validate` tags and returns the first
// error, or nil if the value is valid.
func Validate(v interface{}) error {
	return validate.Struct(v)
}
