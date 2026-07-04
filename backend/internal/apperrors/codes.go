package apperrors

// Error code constants — unique integer IDs per domain error.
// Range allocation:
//
//	1100–1199  Auth & credentials
//	1200–1299  Users & RBAC
//	1300–1399  API keys
//	1400–1499  Files & folders
//	1500–1599  Sharing
//	9000–9099  Generic / transport
const (
	// Auth & credentials
	CodeInvalidCredentials = 1101
	CodeAccountLocked      = 1102
	CodeAccountDisabled    = 1103
	CodeInvalidToken       = 1104
	CodeSessionNotFound    = 1105
	CodeSessionExpired     = 1106
	CodeWeakPassword       = 1107
	CodePasswordReused     = 1108

	// Users & RBAC
	CodeUserNotFound      = 1201
	CodeUserAlreadyExists = 1202
	CodeRoleNotFound      = 1203
	CodeRoleImmutable     = 1204
	CodeForbidden         = 1205
	CodeUnauthorized      = 1206

	// API keys
	CodeAPIKeyNotFound = 1301
	CodeAPIKeyRevoked  = 1302

	// Files & folders
	CodeFileNotFound     = 1401
	CodeFolderNotFound   = 1402
	CodeAlreadyExists    = 1403
	CodeQuotaExceeded    = 1404
	CodePathTraversal    = 1405
	CodeInvalidName      = 1406
	CodeNotEmpty         = 1407
	CodeChecksumMismatch = 1408
	CodeUploadNotFound   = 1409
	CodeUploadExpired    = 1410
	CodeUploadIncomplete = 1411

	// Sharing
	CodeShareNotFound       = 1501
	CodeShareExpired        = 1502
	CodeShareLimitReached   = 1503
	CodeSharePasswordNeeded = 1504

	// Generic / transport
	CodeInvalidRequest     = 9001
	CodeNotFound           = 9002
	CodeConflict           = 9003
	CodeRateLimitExceeded  = 9004
	CodeInternal           = 9005
	CodeServiceUnavailable = 9006
	CodePayloadTooLarge    = 9007
)
