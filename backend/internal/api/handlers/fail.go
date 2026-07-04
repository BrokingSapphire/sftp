package handlers

import (
	"github.com/go-fuego/fuego"

	"sapphirebroking.com/sftp_service/internal/apperrors"
)

// Fail converts a domain error into a Fuego HTTP error (RFC 7807). Unknown
// errors (those mapping to 500) are returned as a generic internal error so
// their message never leaks to the client.
func Fail(err error) error {
	if err == nil {
		return nil
	}
	status := apperrors.HTTPStatus(err)
	if status >= 500 {
		return fuego.InternalServerError{Title: "internal server error", Err: err}
	}
	return fuego.HTTPError{
		Status: status,
		Title:  err.Error(),
		Err:    err,
	}
}
