-- +goose Up
-- Compliance controls on files:
--   legal_hold  — freezes a file (no delete, overwrite, rename or move) until an
--                 admin releases it, regardless of ownership.
--   retain_until — WORM retention: the file cannot be deleted or overwritten
--                 before this timestamp.
ALTER TABLE files
    ADD COLUMN legal_hold  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN retain_until TIMESTAMPTZ;

CREATE INDEX idx_files_legal_hold ON files (legal_hold) WHERE legal_hold = TRUE;

-- +goose Down
DROP INDEX IF EXISTS idx_files_legal_hold;
ALTER TABLE files DROP COLUMN IF EXISTS legal_hold, DROP COLUMN IF EXISTS retain_until;
