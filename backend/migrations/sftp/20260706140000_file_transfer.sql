-- +goose Up
-- When a user is deleted their files are reassigned to another user, who must
-- act on them (keep or delete) within a deadline. Nothing is auto-deleted.
ALTER TABLE files ADD COLUMN transfer_pending  BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE files ADD COLUMN transfer_deadline TIMESTAMPTZ;
ALTER TABLE files ADD COLUMN transfer_from     UUID REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX idx_files_transfer ON files(owner_id, transfer_deadline) WHERE transfer_pending = TRUE;

-- +goose Down
DROP INDEX IF EXISTS idx_files_transfer;
ALTER TABLE files DROP COLUMN IF EXISTS transfer_from;
ALTER TABLE files DROP COLUMN IF EXISTS transfer_deadline;
ALTER TABLE files DROP COLUMN IF EXISTS transfer_pending;
