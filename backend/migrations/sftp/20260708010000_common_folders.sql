-- +goose Up
-- Allow folders in the Common area (organisation-wide, navigable like My Files).
ALTER TABLE folders ADD COLUMN is_common BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_folders_common ON folders (parent_id) WHERE is_common AND deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_folders_common;
ALTER TABLE folders DROP COLUMN IF EXISTS is_common;
