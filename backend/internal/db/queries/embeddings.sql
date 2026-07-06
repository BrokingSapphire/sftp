-- name: DeleteFileEmbeddings :exec
DELETE FROM file_embeddings WHERE file_id = $1;

-- name: InsertFileEmbedding :exec
INSERT INTO file_embeddings (file_id, owner_id, chunk_no, content, embedding)
VALUES (@file_id, @owner_id, @chunk_no, @content, @embedding);

-- name: ListEmbeddingsByOwner :many
SELECT e.file_id, e.content, e.embedding, f.name
FROM file_embeddings e
JOIN files f ON f.id = e.file_id AND f.deleted_at IS NULL
WHERE e.owner_id = @owner_id
LIMIT @row_limit;

-- name: ListFilesNeedingEmbedding :many
SELECT t.file_id, t.content, f.owner_id
FROM file_text t
JOIN files f ON f.id = t.file_id AND f.deleted_at IS NULL
LEFT JOIN file_embeddings e ON e.file_id = t.file_id
WHERE e.file_id IS NULL AND length(t.content) > 0
LIMIT @row_limit;
