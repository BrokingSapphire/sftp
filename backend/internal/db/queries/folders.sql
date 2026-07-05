-- name: CreateFolder :one
INSERT INTO folders (owner_id, parent_id, name, path, depth)
VALUES ($1, sqlc.narg('parent_id'), $2, $3, $4)
RETURNING *;

-- name: GetFolderByID :one
SELECT * FROM folders WHERE id = $1 AND deleted_at IS NULL;

-- name: GetFolderByOwnerPath :one
SELECT * FROM folders
WHERE owner_id = $1 AND path = $2 AND deleted_at IS NULL;

-- name: GetFileByOwnerFolderName :one
SELECT * FROM files
WHERE owner_id = $1
  AND folder_id IS NOT DISTINCT FROM sqlc.narg('folder_id')
  AND name = $2 AND deleted_at IS NULL;

-- name: ListFoldersByParent :many
SELECT * FROM folders
WHERE owner_id = $1
  AND parent_id IS NOT DISTINCT FROM sqlc.narg('parent_id')
  AND deleted_at IS NULL
ORDER BY name ASC;

-- name: RenameFolder :exec
UPDATE folders SET name = $2, path = $3, updated_at = now() WHERE id = $1;

-- name: MoveFolder :exec
UPDATE folders
SET parent_id = sqlc.narg('parent_id'), path = $2, depth = $3, updated_at = now()
WHERE id = $1;

-- name: SoftDeleteFolder :exec
UPDATE folders SET deleted_at = now(), updated_at = now() WHERE id = $1;

-- name: SetFolderStar :exec
UPDATE folders SET is_starred = $2, updated_at = now() WHERE id = $1;

-- name: CountFolderChildren :one
SELECT
  (SELECT count(*) FROM folders WHERE parent_id = $1 AND deleted_at IS NULL) +
  (SELECT count(*) FROM files   WHERE folder_id = $1 AND deleted_at IS NULL) AS total;

-- name: UpdateFolderSize :exec
UPDATE folders SET size_bytes = $2, updated_at = now() WHERE id = $1;
