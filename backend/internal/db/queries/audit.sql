-- name: InsertAuditLog :exec
INSERT INTO audit_logs (
    actor_id, actor_email, action, category, object_type, object_id,
    object_name, result, ip_address, user_agent, browser, os, request_id, metadata
) VALUES (
    sqlc.narg('actor_id'), sqlc.narg('actor_email'), $1, $2,
    sqlc.narg('object_type'), sqlc.narg('object_id'), sqlc.narg('object_name'),
    $3, sqlc.narg('ip_address'), sqlc.narg('user_agent'),
    sqlc.narg('browser'), sqlc.narg('os'), sqlc.narg('request_id'), $4
);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListAuditLogsByActor :many
SELECT * FROM audit_logs
WHERE actor_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByCategory :many
SELECT * FROM audit_logs
WHERE category = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAuditLogs :one
SELECT count(*) FROM audit_logs;

-- name: PurgeAuditLogsBefore :exec
DELETE FROM audit_logs WHERE created_at < $1;
