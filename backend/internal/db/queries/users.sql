-- name: CreateUser :one
INSERT INTO users (
    email, username, password_hash, full_name, employee_id,
    department_id, role_id, phone, storage_quota, must_change_pw,
    password_changed_at, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), $11
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1 AND deleted_at IS NULL;

-- name: GetUserByEmailOrUsername :one
SELECT * FROM users
WHERE (email = $1 OR username = $1) AND deleted_at IS NULL;

-- name: UpdateLastLogin :exec
UPDATE users
SET last_login_at = now(), failed_attempts = 0, updated_at = now()
WHERE id = $1;

-- name: IncrementFailedAttempts :one
UPDATE users
SET failed_attempts = failed_attempts + 1, updated_at = now()
WHERE id = $1
RETURNING failed_attempts;

-- name: LockUser :exec
UPDATE users
SET is_locked = TRUE, locked_until = $2, updated_at = now()
WHERE id = $1;

-- name: UnlockUser :exec
UPDATE users
SET is_locked = FALSE, locked_until = NULL, failed_attempts = 0, updated_at = now()
WHERE id = $1;

-- name: SetUserPassword :exec
UPDATE users
SET password_hash = $2, password_changed_at = now(), must_change_pw = FALSE, updated_at = now()
WHERE id = $1;

-- name: ResetUserPassword :exec
-- Admin reset: force the user to change it on next login.
UPDATE users
SET password_hash = $2, password_changed_at = now(), must_change_pw = TRUE, updated_at = now()
WHERE id = $1;

-- name: SetUserActive :exec
UPDATE users SET is_active = $2, updated_at = now() WHERE id = $1;

-- name: UpdateUserProfile :one
UPDATE users
SET full_name = $2, phone = $3, department_id = $4, avatar_path = $5, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserRole :exec
UPDATE users SET role_id = $2, updated_at = now() WHERE id = $1;

-- name: SetUserAvatar :exec
UPDATE users SET avatar_path = $2, updated_at = now() WHERE id = $1;

-- name: GetUserAvatar :one
SELECT avatar_path FROM users WHERE id = $1;

-- name: UpdateUserQuota :exec
UPDATE users SET storage_quota = $2, updated_at = now() WHERE id = $1;

-- name: AddStorageUsed :exec
UPDATE users SET storage_used = storage_used + $2, updated_at = now() WHERE id = $1;

-- name: SoftDeleteUser :exec
UPDATE users SET deleted_at = now(), is_active = FALSE, updated_at = now() WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT count(*) FROM users WHERE deleted_at IS NULL;

-- name: CountActiveUsers :one
SELECT count(*) FROM users WHERE deleted_at IS NULL AND is_active = TRUE;
