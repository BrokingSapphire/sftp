-- name: CreateFile :one
INSERT INTO files (
    owner_id, folder_id, name, extension, mime_type,
    size_bytes, checksum_sha256, storage_key
) VALUES ($1, sqlc.narg('folder_id'), $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetFileByID :one
SELECT * FROM files WHERE id = $1 AND deleted_at IS NULL;

-- name: GetFileByIDIncludingTrashed :one
SELECT * FROM files WHERE id = $1;

-- name: ListFilesByFolder :many
SELECT * FROM files
WHERE owner_id = $1
  AND folder_id IS NOT DISTINCT FROM sqlc.narg('folder_id')
  AND deleted_at IS NULL
ORDER BY name ASC
LIMIT $2 OFFSET $3;

-- name: CountFilesByFolder :one
SELECT count(*) FROM files
WHERE owner_id = $1
  AND folder_id IS NOT DISTINCT FROM sqlc.narg('folder_id')
  AND deleted_at IS NULL;

-- name: RenameFile :exec
UPDATE files SET name = $2, extension = $3, updated_at = now() WHERE id = $1;

-- name: MoveFile :exec
UPDATE files SET folder_id = sqlc.narg('folder_id'), updated_at = now() WHERE id = $1;

-- name: SetFileStar :exec
UPDATE files SET is_starred = $2, updated_at = now() WHERE id = $1;

-- name: SoftDeleteFile :exec
UPDATE files SET deleted_at = now(), updated_at = now() WHERE id = $1;

-- name: RestoreFile :exec
UPDATE files SET deleted_at = NULL, updated_at = now() WHERE id = $1;

-- name: HardDeleteFile :one
DELETE FROM files WHERE id = $1 RETURNING storage_key;

-- name: IncrementDownloadCount :exec
UPDATE files SET download_count = download_count + 1 WHERE id = $1;

-- name: ListTrash :many
SELECT * FROM files
WHERE owner_id = $1 AND deleted_at IS NOT NULL
ORDER BY deleted_at DESC
LIMIT $2 OFFSET $3;

-- name: ListRecentFiles :many
SELECT * FROM files
WHERE owner_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2;

-- name: ListStarredFiles :many
SELECT * FROM files
WHERE owner_id = $1 AND is_starred = TRUE AND deleted_at IS NULL
ORDER BY updated_at DESC;

-- name: SearchFiles :many
SELECT * FROM files
WHERE owner_id = $1 AND deleted_at IS NULL
  AND name ILIKE '%' || $2 || '%'
ORDER BY name ASC
LIMIT $3 OFFSET $4;

-- name: PurgeExpiredTrash :many
DELETE FROM files
WHERE deleted_at IS NOT NULL AND deleted_at < $1
RETURNING storage_key;

-- name: SumFileSizesByOwner :one
SELECT COALESCE(sum(size_bytes), 0)::bigint FROM files
WHERE owner_id = $1 AND deleted_at IS NULL;

-- name: SetFileCommon :exec
UPDATE files SET is_common = $2, updated_at = now() WHERE id = $1;

-- name: ListCommonFiles :many
SELECT f.*, u.full_name AS uploader_name, u.username AS uploader_username,
       (u.avatar_path IS NOT NULL AND u.avatar_path <> '') AS uploader_has_avatar
FROM files f
JOIN users u ON u.id = f.owner_id
WHERE f.is_common = TRUE AND f.deleted_at IS NULL
ORDER BY f.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountCommonFiles :one
SELECT count(*) FROM files WHERE is_common = TRUE AND deleted_at IS NULL;

-- name: SoftDeleteFilesInFolders :exec
UPDATE files SET deleted_at = now(), updated_at = now()
WHERE folder_id = ANY(@folder_ids::uuid[])
  AND owner_id = @owner_id
  AND deleted_at IS NULL
  AND legal_hold = FALSE
  AND (retain_until IS NULL OR retain_until < now());

-- name: PurgeUserTrash :many
DELETE FROM files
WHERE owner_id = @owner_id AND deleted_at IS NOT NULL
  AND legal_hold = FALSE
  AND (retain_until IS NULL OR retain_until < now())
RETURNING storage_key, size_bytes, owner_id;

-- name: ListAllFilesForBackup :many
SELECT f.id, f.owner_id, u.username AS owner_username, u.email AS owner_email,
       f.name, f.storage_key, f.size_bytes, f.checksum_sha256, f.is_common,
       COALESCE(fo.path, '') AS folder_path
FROM files f
JOIN users u ON u.id = f.owner_id
LEFT JOIN folders fo ON fo.id = f.folder_id
WHERE f.deleted_at IS NULL
ORDER BY f.owner_id, f.id;
