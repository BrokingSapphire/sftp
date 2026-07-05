-- name: CreateShare :one
INSERT INTO shares (
    token, owner_id, file_id, folder_id, permission,
    password_hash, download_limit, expires_at
) VALUES (
    $1, $2, sqlc.narg('file_id'), sqlc.narg('folder_id'), $3,
    sqlc.narg('password_hash'), sqlc.narg('download_limit'), sqlc.narg('expires_at')
)
RETURNING *;

-- name: GetShareByToken :one
SELECT * FROM shares
WHERE token = $1 AND is_active = TRUE;

-- name: ListSharesByOwner :many
SELECT * FROM shares
WHERE owner_id = $1
ORDER BY created_at DESC;

-- name: RevokeShare :exec
UPDATE shares SET is_active = FALSE, updated_at = now()
WHERE id = $1 AND owner_id = $2;

-- name: IncrementShareDownload :exec
UPDATE shares SET download_count = download_count + 1, updated_at = now()
WHERE id = $1;

-- name: DeactivateExpiredShares :exec
UPDATE shares SET is_active = FALSE, updated_at = now()
WHERE is_active = TRUE AND expires_at IS NOT NULL AND expires_at < now();
