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

-- name: StorageByUser :many
SELECT u.id, u.username, u.full_name, u.email, r.slug AS role,
       u.storage_used, u.storage_quota,
       (SELECT count(*) FROM files f WHERE f.owner_id = u.id AND f.deleted_at IS NULL) AS file_count
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE u.deleted_at IS NULL
ORDER BY u.storage_used DESC;

-- name: MediaBreakdown :many
SELECT
  CASE
    WHEN extension IN ('png','jpg','jpeg','gif','svg','webp','bmp','tiff','ico','avif','heic','psd') THEN 'images'
    WHEN extension IN ('mp4','mov','webm','mkv','avi','flv','ogv') THEN 'video'
    WHEN extension IN ('mp3','wav','flac','ogg','m4a','aac') THEN 'audio'
    WHEN extension IN ('pdf','doc','docx','txt','md','rtf','odt','epub') THEN 'documents'
    WHEN extension IN ('xls','xlsx','xlsm','csv','tsv','ods') THEN 'spreadsheets'
    WHEN extension IN ('ppt','pptx','odp') THEN 'presentations'
    WHEN extension IN ('zip','rar','7z','tar','gz','bz2','xz','iso','dmg') THEN 'archives'
    WHEN extension IN ('js','ts','tsx','jsx','go','py','rb','php','java','c','h','cpp','cs','rs','html','css','json','xml','yaml','yml','sh','sql') THEN 'code'
    ELSE 'other'
  END AS category,
  COALESCE(sum(size_bytes), 0)::bigint AS total,
  count(*) AS files
FROM files
WHERE deleted_at IS NULL
GROUP BY category
ORDER BY total DESC;
