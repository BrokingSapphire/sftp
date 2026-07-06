-- name: ReassignUserFiles :exec
UPDATE files
SET owner_id = @to_user, transfer_pending = TRUE, transfer_deadline = @deadline,
    transfer_from = @from_user, updated_at = now()
WHERE owner_id = @from_user AND deleted_at IS NULL;

-- name: ReassignUserFolders :exec
UPDATE folders SET owner_id = @to_user, updated_at = now()
WHERE owner_id = @from_user AND deleted_at IS NULL;

-- name: ListInheritedFiles :many
SELECT * FROM files
WHERE owner_id = $1 AND transfer_pending = TRUE AND deleted_at IS NULL
ORDER BY transfer_deadline ASC;

-- name: CountInheritedFiles :one
SELECT count(*) FROM files
WHERE owner_id = $1 AND transfer_pending = TRUE AND deleted_at IS NULL;

-- name: ClearFilePending :exec
UPDATE files SET transfer_pending = FALSE, transfer_deadline = NULL, transfer_from = NULL, updated_at = now()
WHERE id = $1 AND owner_id = $2;

-- name: ClearAllPendingForUser :exec
UPDATE files SET transfer_pending = FALSE, transfer_deadline = NULL, transfer_from = NULL, updated_at = now()
WHERE owner_id = $1 AND transfer_pending = TRUE;

-- name: PendingTransfersByUser :many
SELECT owner_id, count(*) AS pending_count, min(transfer_deadline) AS earliest_deadline
FROM files
WHERE transfer_pending = TRUE AND deleted_at IS NULL
GROUP BY owner_id;

-- name: CreateNotification :exec
INSERT INTO notifications (user_id, type, title, body, link, metadata)
VALUES ($1, $2, $3, $4, sqlc.narg('link'), $5);

-- name: CountRecentNotifications :one
SELECT count(*) FROM notifications
WHERE user_id = $1 AND type = $2 AND created_at > $3;

-- name: ListNotifications :many
SELECT * FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUnreadNotifications :one
SELECT count(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE;

-- name: MarkNotificationRead :exec
UPDATE notifications SET is_read = TRUE, read_at = now() WHERE id = $1 AND user_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET is_read = TRUE, read_at = now() WHERE user_id = $1 AND is_read = FALSE;

-- name: ListInheritedWithSource :many
SELECT f.*,
       u.full_name AS from_name, u.username AS from_username, u.email AS from_email
FROM files f
LEFT JOIN users u ON u.id = f.transfer_from
WHERE f.owner_id = $1 AND f.transfer_pending = TRUE AND f.deleted_at IS NULL
ORDER BY COALESCE(u.full_name, u.username, 'zzz'), f.name;
