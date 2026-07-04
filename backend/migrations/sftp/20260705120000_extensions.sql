-- +goose Up
-- Enable required PostgreSQL extensions.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- trigram search for filenames
CREATE EXTENSION IF NOT EXISTS "citext";     -- case-insensitive email/username

-- +goose Down
DROP EXTENSION IF EXISTS "citext";
DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "pgcrypto";
