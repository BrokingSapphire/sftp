-- name: CreateUpload :one
INSERT INTO uploads (
    user_id, folder_id, filename, total_size, chunk_size,
    total_chunks, temp_key, checksum_sha256, expires_at
) VALUES ($1, sqlc.narg('folder_id'), $2, $3, $4, $5, $6, sqlc.narg('checksum'), $7)
RETURNING *;

-- name: GetUpload :one
SELECT * FROM uploads WHERE id = $1;

-- name: GetUploadForUser :one
SELECT * FROM uploads WHERE id = $1 AND user_id = $2;

-- name: RecordChunk :exec
INSERT INTO upload_chunks (upload_id, chunk_index, size_bytes, checksum)
VALUES ($1, $2, $3, sqlc.narg('checksum'))
ON CONFLICT (upload_id, chunk_index) DO NOTHING;

-- name: ListReceivedChunks :many
SELECT chunk_index FROM upload_chunks
WHERE upload_id = $1
ORDER BY chunk_index ASC;

-- name: UpdateUploadProgress :exec
UPDATE uploads
SET uploaded_chunks = $2, received_bytes = $3, updated_at = now()
WHERE id = $1;

-- name: CompleteUpload :exec
UPDATE uploads
SET status = 'completed', file_id = $2, updated_at = now()
WHERE id = $1;

-- name: SetUploadStatus :exec
UPDATE uploads SET status = $2, updated_at = now() WHERE id = $1;

-- name: DeleteExpiredUploads :many
DELETE FROM uploads
WHERE status = 'in_progress' AND expires_at < now()
RETURNING id, temp_key;
