-- name: InsertLoginHistory :exec
INSERT INTO login_history (
    user_id, email, success, reason, ip_address, user_agent, browser, os
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListLoginHistoryForUser :many
SELECT * FROM login_history
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListRecentLoginHistory :many
SELECT * FROM login_history
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
