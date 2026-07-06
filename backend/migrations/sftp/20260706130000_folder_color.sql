-- +goose Up
ALTER TABLE folders ADD COLUMN color VARCHAR(16) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE folders DROP COLUMN IF EXISTS color;
