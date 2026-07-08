-- name: GrantFileToUser :one
INSERT INTO resource_permissions (file_id, grantee_user_id, can_read, can_write, can_download, can_share, created_by)
VALUES ($1, $2, TRUE, @can_write, TRUE, FALSE, @created_by)
ON CONFLICT (file_id, grantee_user_id) WHERE file_id IS NOT NULL AND grantee_user_id IS NOT NULL
DO UPDATE SET can_write = EXCLUDED.can_write
RETURNING *;

-- name: RevokeFileGrant :exec
DELETE FROM resource_permissions
WHERE file_id = $1 AND grantee_user_id = $2;

-- name: GetFileGrant :one
SELECT can_read, can_write, can_download
FROM resource_permissions
WHERE file_id = $1 AND grantee_user_id = $2;

-- name: ListSharedWithMe :many
SELECT f.*, rp.can_write, rp.created_at AS shared_at,
       u.full_name AS owner_name, u.username AS owner_username,
       (u.avatar_path IS NOT NULL AND u.avatar_path <> '') AS owner_has_avatar
FROM resource_permissions rp
JOIN files f ON f.id = rp.file_id AND f.deleted_at IS NULL
JOIN users u ON u.id = f.owner_id
WHERE rp.grantee_user_id = $1 AND rp.file_id IS NOT NULL
ORDER BY rp.created_at DESC;

-- name: ListFileGrants :many
SELECT rp.grantee_user_id, rp.can_write,
       u.full_name, u.username, u.email,
       (u.avatar_path IS NOT NULL AND u.avatar_path <> '') AS has_avatar
FROM resource_permissions rp
JOIN users u ON u.id = rp.grantee_user_id
WHERE rp.file_id = $1
ORDER BY u.full_name;

-- Folder-level internal shares (mirror the file-level grants above) -----------

-- name: GrantFolderToUser :one
INSERT INTO resource_permissions (folder_id, grantee_user_id, can_read, can_write, can_download, can_share, created_by)
VALUES ($1, $2, TRUE, @can_write, TRUE, FALSE, @created_by)
ON CONFLICT (folder_id, grantee_user_id) WHERE folder_id IS NOT NULL AND grantee_user_id IS NOT NULL
DO UPDATE SET can_write = EXCLUDED.can_write
RETURNING *;

-- name: RevokeFolderGrant :exec
DELETE FROM resource_permissions
WHERE folder_id = $1 AND grantee_user_id = $2;

-- name: GetFolderGrant :one
SELECT can_read, can_write, can_download
FROM resource_permissions
WHERE folder_id = $1 AND grantee_user_id = $2;

-- name: ListFolderGrants :many
SELECT rp.grantee_user_id, rp.can_write,
       u.full_name, u.username, u.email,
       (u.avatar_path IS NOT NULL AND u.avatar_path <> '') AS has_avatar
FROM resource_permissions rp
JOIN users u ON u.id = rp.grantee_user_id
WHERE rp.folder_id = $1
ORDER BY u.full_name;

-- name: ListSharedFoldersWithMe :many
SELECT fo.*, rp.can_write, rp.created_at AS shared_at,
       u.full_name AS owner_name, u.username AS owner_username,
       (u.avatar_path IS NOT NULL AND u.avatar_path <> '') AS owner_has_avatar
FROM resource_permissions rp
JOIN folders fo ON fo.id = rp.folder_id AND fo.deleted_at IS NULL
JOIN users u ON u.id = fo.owner_id
WHERE rp.grantee_user_id = $1 AND rp.folder_id IS NOT NULL
ORDER BY rp.created_at DESC;
