-- name: CountFilesByOwner :one
SELECT count(*) FROM files WHERE owner_id = $1 AND deleted_at IS NULL;

-- name: CountFoldersByOwner :one
SELECT count(*) FROM folders WHERE owner_id = $1 AND deleted_at IS NULL;

-- name: CountTrashByOwner :one
SELECT count(*) FROM files WHERE owner_id = $1 AND deleted_at IS NOT NULL;

-- name: LargestFilesByOwner :many
SELECT * FROM files
WHERE owner_id = $1 AND deleted_at IS NULL
ORDER BY size_bytes DESC
LIMIT $2;

-- name: SystemFileCount :one
SELECT count(*) FROM files WHERE deleted_at IS NULL;

-- name: SystemStorageUsed :one
SELECT COALESCE(sum(size_bytes), 0)::bigint FROM files WHERE deleted_at IS NULL;
