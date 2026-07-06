-- +goose Up
-- Organisation-wide "Common" files: visible to everyone, uploaded by anyone,
-- deletable only by the uploader (owner_id) or an admin.
ALTER TABLE files ADD COLUMN is_common BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_files_common ON files(is_common, created_at DESC) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_files_common;
ALTER TABLE files DROP COLUMN IF EXISTS is_common;
