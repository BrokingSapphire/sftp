// Package apperrors defines the service's sentinel domain errors and their
// mapping to HTTP status codes and API error-type strings.
package apperrors

import "errors"

var (
	// Auth & credentials
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account locked")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
	ErrWeakPassword       = errors.New("password does not meet complexity requirements")
	ErrPasswordReused     = errors.New("password was used recently")

	// Users & RBAC
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrRoleNotFound      = errors.New("role not found")
	ErrRoleImmutable     = errors.New("system role cannot be modified")
	ErrForbidden         = errors.New("forbidden")
	ErrLegalHold         = errors.New("file is under legal hold and cannot be modified")
	ErrUnderRetention    = errors.New("file is under a retention lock and cannot be deleted or modified yet")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrSSODomainNotAllowed = errors.New("email domain not permitted for SSO")
	ErrSSONotConfigured    = errors.New("sso provider not configured")

	// API keys
	ErrAPIKeyNotFound = errors.New("api key not found")
	ErrAPIKeyRevoked  = errors.New("api key revoked")

	// Files & folders
	ErrFileNotFound      = errors.New("file not found")
	ErrFolderNotFound    = errors.New("folder not found")
	ErrAlreadyExists     = errors.New("a file or folder with that name already exists")
	ErrQuotaExceeded     = errors.New("storage quota exceeded")
	ErrPathTraversal     = errors.New("invalid path")
	ErrInvalidName       = errors.New("invalid file or folder name")
	ErrNotEmpty          = errors.New("folder is not empty")
	ErrChecksumMismatch  = errors.New("checksum mismatch")
	ErrUploadNotFound    = errors.New("upload session not found")
	ErrUploadExpired     = errors.New("upload session expired")
	ErrUploadIncomplete  = errors.New("upload is incomplete")

	// Sharing
	ErrShareNotFound       = errors.New("share not found")
	ErrShareExpired        = errors.New("share expired")
	ErrShareLimitReached   = errors.New("share download limit reached")
	ErrSharePasswordNeeded = errors.New("share password required")

	// Generic / transport
	ErrInvalidRequest     = errors.New("invalid request")
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrInternal           = errors.New("internal server error")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrPayloadTooLarge    = errors.New("payload too large")
)
