-- +goose Up
-- Data classification derived from document content (see pkg/dlp). Drives DLP
-- controls on sharing and sensitivity badges in the UI.
ALTER TABLE files
    ADD COLUMN sensitivity TEXT NOT NULL DEFAULT 'public',
    ADD COLUMN pii_types   TEXT[] NOT NULL DEFAULT '{}';

CREATE INDEX idx_files_sensitivity ON files (sensitivity) WHERE sensitivity <> 'public';

-- +goose Down
DROP INDEX IF EXISTS idx_files_sensitivity;
ALTER TABLE files DROP COLUMN IF EXISTS sensitivity, DROP COLUMN IF EXISTS pii_types;
