package apperrors

import (
	"errors"
	"net/http"
)

// IsClientError reports whether err maps to a 4xx status (caller fault).
func IsClientError(err error) bool {
	s := HTTPStatus(err)
	return s >= 400 && s < 500
}

// HTTPStatus maps a domain error to an HTTP status code (500 fallback).
func HTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrNotFound),
		errors.Is(err, ErrRoleNotFound), errors.Is(err, ErrFileNotFound),
		errors.Is(err, ErrFolderNotFound), errors.Is(err, ErrShareNotFound),
		errors.Is(err, ErrAPIKeyNotFound), errors.Is(err, ErrUploadNotFound),
		errors.Is(err, ErrSessionNotFound):
		return http.StatusNotFound

	case errors.Is(err, ErrUserAlreadyExists), errors.Is(err, ErrConflict),
		errors.Is(err, ErrAlreadyExists), errors.Is(err, ErrNotEmpty),
		errors.Is(err, ErrSessionActive):
		return http.StatusConflict

	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidToken),
		errors.Is(err, ErrSessionExpired), errors.Is(err, ErrUnauthorized),
		errors.Is(err, ErrAPIKeyRevoked), errors.Is(err, ErrSharePasswordNeeded):
		return http.StatusUnauthorized

	case errors.Is(err, ErrForbidden), errors.Is(err, ErrAccountLocked),
		errors.Is(err, ErrAccountDisabled), errors.Is(err, ErrRoleImmutable),
		errors.Is(err, ErrSSODomainNotAllowed),
		errors.Is(err, ErrLegalHold), errors.Is(err, ErrUnderRetention),
		errors.Is(err, ErrDLPBlocked):
		return http.StatusForbidden

	case errors.Is(err, ErrSSONotConfigured):
		return http.StatusServiceUnavailable

	case errors.Is(err, ErrWeakPassword), errors.Is(err, ErrPasswordReused),
		errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrPathTraversal),
		errors.Is(err, ErrInvalidName), errors.Is(err, ErrChecksumMismatch),
		errors.Is(err, ErrUploadExpired), errors.Is(err, ErrUploadIncomplete),
		errors.Is(err, ErrShareExpired):
		return http.StatusBadRequest

	case errors.Is(err, ErrQuotaExceeded), errors.Is(err, ErrShareLimitReached):
		return http.StatusForbidden

	case errors.Is(err, ErrPayloadTooLarge):
		return http.StatusRequestEntityTooLarge

	case errors.Is(err, ErrRateLimitExceeded):
		return http.StatusTooManyRequests

	case errors.Is(err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable

	default:
		return http.StatusInternalServerError
	}
}

// ErrTypeString maps a domain error to the API response error-type string.
func ErrTypeString(err error) string {
	switch {
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidToken),
		errors.Is(err, ErrSessionExpired), errors.Is(err, ErrSessionNotFound),
		errors.Is(err, ErrAPIKeyRevoked), errors.Is(err, ErrSharePasswordNeeded):
		return "AUTHENTICATION_ERROR"

	case errors.Is(err, ErrForbidden), errors.Is(err, ErrUnauthorized),
		errors.Is(err, ErrAccountLocked), errors.Is(err, ErrAccountDisabled),
		errors.Is(err, ErrRoleImmutable), errors.Is(err, ErrQuotaExceeded),
		errors.Is(err, ErrShareLimitReached):
		return "AUTHORIZATION_ERROR"

	case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrNotFound),
		errors.Is(err, ErrRoleNotFound), errors.Is(err, ErrFileNotFound),
		errors.Is(err, ErrFolderNotFound), errors.Is(err, ErrShareNotFound),
		errors.Is(err, ErrAPIKeyNotFound), errors.Is(err, ErrUploadNotFound):
		return "NOT_FOUND"

	case errors.Is(err, ErrUserAlreadyExists), errors.Is(err, ErrConflict),
		errors.Is(err, ErrAlreadyExists), errors.Is(err, ErrNotEmpty):
		return "CONFLICT"

	case errors.Is(err, ErrRateLimitExceeded):
		return "RATE_LIMIT_ERROR"

	case errors.Is(err, ErrServiceUnavailable):
		return "THIRD_PARTY_ERROR"

	case errors.Is(err, ErrPayloadTooLarge):
		return "PAYLOAD_TOO_LARGE"

	case errors.Is(err, ErrWeakPassword), errors.Is(err, ErrPasswordReused),
		errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrPathTraversal),
		errors.Is(err, ErrInvalidName), errors.Is(err, ErrChecksumMismatch),
		errors.Is(err, ErrUploadExpired), errors.Is(err, ErrUploadIncomplete),
		errors.Is(err, ErrShareExpired):
		return "VALIDATION_ERROR"

	default:
		return "INTERNAL_SERVER_ERROR"
	}
}

// ErrCode maps a domain error to its integer code.
func ErrCode(err error) int {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return CodeInvalidCredentials
	case errors.Is(err, ErrAccountLocked):
		return CodeAccountLocked
	case errors.Is(err, ErrAccountDisabled):
		return CodeAccountDisabled
	case errors.Is(err, ErrInvalidToken):
		return CodeInvalidToken
	case errors.Is(err, ErrSessionNotFound):
		return CodeSessionNotFound
	case errors.Is(err, ErrSessionExpired):
		return CodeSessionExpired
	case errors.Is(err, ErrWeakPassword):
		return CodeWeakPassword
	case errors.Is(err, ErrPasswordReused):
		return CodePasswordReused
	case errors.Is(err, ErrUserNotFound):
		return CodeUserNotFound
	case errors.Is(err, ErrUserAlreadyExists):
		return CodeUserAlreadyExists
	case errors.Is(err, ErrRoleNotFound):
		return CodeRoleNotFound
	case errors.Is(err, ErrRoleImmutable):
		return CodeRoleImmutable
	case errors.Is(err, ErrForbidden):
		return CodeForbidden
	case errors.Is(err, ErrUnauthorized):
		return CodeUnauthorized
	case errors.Is(err, ErrAPIKeyNotFound):
		return CodeAPIKeyNotFound
	case errors.Is(err, ErrAPIKeyRevoked):
		return CodeAPIKeyRevoked
	case errors.Is(err, ErrFileNotFound):
		return CodeFileNotFound
	case errors.Is(err, ErrFolderNotFound):
		return CodeFolderNotFound
	case errors.Is(err, ErrAlreadyExists):
		return CodeAlreadyExists
	case errors.Is(err, ErrQuotaExceeded):
		return CodeQuotaExceeded
	case errors.Is(err, ErrPathTraversal):
		return CodePathTraversal
	case errors.Is(err, ErrInvalidName):
		return CodeInvalidName
	case errors.Is(err, ErrNotEmpty):
		return CodeNotEmpty
	case errors.Is(err, ErrChecksumMismatch):
		return CodeChecksumMismatch
	case errors.Is(err, ErrUploadNotFound):
		return CodeUploadNotFound
	case errors.Is(err, ErrUploadExpired):
		return CodeUploadExpired
	case errors.Is(err, ErrUploadIncomplete):
		return CodeUploadIncomplete
	case errors.Is(err, ErrShareNotFound):
		return CodeShareNotFound
	case errors.Is(err, ErrShareExpired):
		return CodeShareExpired
	case errors.Is(err, ErrShareLimitReached):
		return CodeShareLimitReached
	case errors.Is(err, ErrSharePasswordNeeded):
		return CodeSharePasswordNeeded
	case errors.Is(err, ErrInvalidRequest):
		return CodeInvalidRequest
	case errors.Is(err, ErrNotFound):
		return CodeNotFound
	case errors.Is(err, ErrConflict):
		return CodeConflict
	case errors.Is(err, ErrRateLimitExceeded):
		return CodeRateLimitExceeded
	case errors.Is(err, ErrServiceUnavailable):
		return CodeServiceUnavailable
	case errors.Is(err, ErrPayloadTooLarge):
		return CodePayloadTooLarge
	default:
		return CodeInternal
	}
}
