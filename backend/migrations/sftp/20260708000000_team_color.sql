-- +goose Up
ALTER TABLE teams ADD COLUMN color VARCHAR(16) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE teams DROP COLUMN IF EXISTS color;
