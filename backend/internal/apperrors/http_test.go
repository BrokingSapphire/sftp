package apperrors

import (
	"fmt"
	"net/http"
	"testing"
)

func TestHTTPStatus(t *testing.T) {
	cases := map[error]int{
		ErrUserNotFound:       http.StatusNotFound,
		ErrFileNotFound:       http.StatusNotFound,
		ErrAlreadyExists:      http.StatusConflict,
		ErrInvalidCredentials: http.StatusUnauthorized,
		ErrForbidden:          http.StatusForbidden,
		ErrQuotaExceeded:      http.StatusForbidden,
		ErrPayloadTooLarge:    http.StatusRequestEntityTooLarge,
		ErrRateLimitExceeded:  http.StatusTooManyRequests,
		ErrInvalidRequest:     http.StatusBadRequest,
		ErrInternal:           http.StatusInternalServerError,
	}
	for err, want := range cases {
		if got := HTTPStatus(err); got != want {
			t.Errorf("HTTPStatus(%v) = %d, want %d", err, got, want)
		}
	}
}

func TestHTTPStatusWrapped(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", ErrFileNotFound)
	if HTTPStatus(wrapped) != http.StatusNotFound {
		t.Fatal("wrapped error should map to 404")
	}
}

func TestErrCodeAndType(t *testing.T) {
	if ErrCode(ErrQuotaExceeded) != CodeQuotaExceeded {
		t.Fatal("bad code for quota exceeded")
	}
	if ErrCode(fmt.Errorf("unknown")) != CodeInternal {
		t.Fatal("unknown error should map to internal code")
	}
	if ErrTypeString(ErrInvalidCredentials) != "AUTHENTICATION_ERROR" {
		t.Fatal("bad type string")
	}
	if !IsClientError(ErrInvalidRequest) {
		t.Fatal("invalid request is a client error")
	}
	if IsClientError(ErrInternal) {
		t.Fatal("internal is not a client error")
	}
}
