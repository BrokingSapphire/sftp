-- name: InsertFileVersion :exec
INSERT INTO file_versions (file_id, version_no, size_bytes, checksum_sha256, storage_key, created_by)
VALUES (@file_id, @version_no, @size_bytes, @checksum_sha256, @storage_key, @created_by)
ON CONFLICT (file_id, version_no) DO NOTHING;

-- name: BumpFileContent :one
UPDATE files
SET storage_key = @storage_key,
    size_bytes = @size_bytes,
    checksum_sha256 = @checksum_sha256,
    version_no = version_no + 1,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: ListFileVersions :many
SELECT v.version_no, v.size_bytes, v.checksum_sha256, v.created_at,
       u.full_name AS author_name, u.username AS author_username
FROM file_versions v
LEFT JOIN users u ON u.id = v.created_by
WHERE v.file_id = @file_id
ORDER BY v.version_no DESC;

-- name: GetFileVersion :one
SELECT version_no, size_bytes, checksum_sha256, storage_key
FROM file_versions
WHERE file_id = @file_id AND version_no = @version_no;
