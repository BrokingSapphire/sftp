-- name: UpsertFileText :exec
INSERT INTO file_text (file_id, content, tsv, ocr, bytes, lang)
VALUES ($1, $2, to_tsvector('english', $2), $3, $4, 'english')
ON CONFLICT (file_id) DO UPDATE
SET content = EXCLUDED.content,
    tsv = EXCLUDED.tsv,
    ocr = EXCLUDED.ocr,
    bytes = EXCLUDED.bytes,
    extracted_at = now();

-- name: HasFileText :one
SELECT EXISTS (SELECT 1 FROM file_text WHERE file_id = $1);

-- name: ListFilesMissingText :many
SELECT f.id, f.storage_key, f.mime_type, f.extension, f.size_bytes
FROM files f
LEFT JOIN file_text t ON t.file_id = f.id
WHERE f.deleted_at IS NULL
  AND t.file_id IS NULL
ORDER BY f.created_at DESC
LIMIT $1;

-- name: SearchFileContent :many
SELECT f.id, f.name, f.extension, f.mime_type, f.size_bytes, f.folder_id,
       f.is_starred, f.version_no, f.download_count, f.created_at, f.updated_at,
       ts_rank(t.tsv, websearch_to_tsquery('english', @query)) AS rank,
       ts_headline('english', t.content, websearch_to_tsquery('english', @query),
                   'MaxFragments=1,MaxWords=18,MinWords=5,StartSel=<<,StopSel=>>') AS snippet
FROM file_text t
JOIN files f ON f.id = t.file_id
WHERE f.owner_id = @owner_id
  AND f.deleted_at IS NULL
  AND t.tsv @@ websearch_to_tsquery('english', @query)
ORDER BY rank DESC
LIMIT @row_limit;

-- name: SetFileClassification :exec
UPDATE files SET sensitivity = @sensitivity, pii_types = @pii_types WHERE id = @id;
