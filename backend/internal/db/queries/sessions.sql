-- name: CreateSession :one
INSERT INTO sessions (
    user_id, refresh_token_hash, user_agent, ip_address,
    device_label, remember_me, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetSessionByHash :one
SELECT * FROM sessions
WHERE refresh_token_hash = $1 AND revoked_at IS NULL AND expires_at > now();

-- name: GetSessionByID :one
SELECT * FROM sessions WHERE id = $1;

-- name: RotateSession :exec
UPDATE sessions
SET refresh_token_hash = $2, expires_at = $3, last_seen_at = now()
WHERE id = $1;

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = now() WHERE id = $1;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = now() WHERE id = $1;

-- name: RevokeSessionByHash :exec
UPDATE sessions SET revoked_at = now() WHERE refresh_token_hash = $1;

-- name: RevokeAllUserSessions :exec
UPDATE sessions SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: ListUserSessions :many
SELECT * FROM sessions
WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
ORDER BY last_seen_at DESC;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < now() OR (revoked_at IS NOT NULL AND revoked_at < now() - interval '7 days');
