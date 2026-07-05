-- name: InsertActivity :exec
INSERT INTO user_activity (
    user_id, session_id, event_type, element, path, ip_address, user_agent, metadata
) VALUES (
    sqlc.narg('user_id'), sqlc.narg('session_id'), $1,
    sqlc.narg('element'), sqlc.narg('path'),
    sqlc.narg('ip_address'), sqlc.narg('user_agent'), $2
);

-- name: ListActivityByUser :many
SELECT * FROM user_activity
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListRecentActivity :many
SELECT * FROM user_activity
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: PurgeActivityBefore :exec
DELETE FROM user_activity WHERE created_at < $1;
