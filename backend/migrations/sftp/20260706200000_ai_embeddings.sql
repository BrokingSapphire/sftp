-- +goose Up
-- Embedding vectors for semantic search / RAG (optional AI feature). Stored as
-- JSONB float arrays so no database extension (pgvector) is required — the
-- migration is safe on a stock Postgres image. Similarity is computed in Go.
-- Only populated when AI is enabled.
CREATE TABLE file_embeddings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id    UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chunk_no   INT NOT NULL,
    content    TEXT NOT NULL,
    embedding  JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (file_id, chunk_no)
);
CREATE INDEX idx_embeddings_owner ON file_embeddings (owner_id);

-- +goose Down
DROP TABLE IF EXISTS file_embeddings;
