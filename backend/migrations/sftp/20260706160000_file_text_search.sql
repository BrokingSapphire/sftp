-- +goose Up
-- Extracted, searchable text for every file (feeds full-text search today and
-- semantic search / classification later). One row per file.
CREATE TABLE file_text (
    file_id      UUID PRIMARY KEY REFERENCES files(id) ON DELETE CASCADE,
    content      TEXT NOT NULL DEFAULT '',
    tsv          tsvector,
    lang         TEXT NOT NULL DEFAULT 'english',
    ocr          BOOLEAN NOT NULL DEFAULT FALSE,
    bytes        BIGINT NOT NULL DEFAULT 0,
    extracted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_file_text_tsv ON file_text USING GIN (tsv);

-- +goose Down
DROP TABLE IF EXISTS file_text;
