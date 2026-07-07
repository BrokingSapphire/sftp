-- +goose Up
ALTER TABLE users ADD COLUMN language VARCHAR(8) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS language;
