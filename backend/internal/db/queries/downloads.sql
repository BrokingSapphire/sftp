-- name: InsertDownload :exec
INSERT INTO downloads (file_id, user_id, share_id, bytes_sent, ip_address, user_agent)
VALUES (sqlc.narg('file_id'), sqlc.narg('user_id'), sqlc.narg('share_id'), $1, $2, $3);

-- name: CountDownloadsSince :one
SELECT count(*) FROM downloads WHERE created_at >= $1;

-- name: ListRecentDownloadsForUser :many
SELECT * FROM downloads
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;
