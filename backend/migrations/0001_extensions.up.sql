-- Enable required PostgreSQL extensions.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- trigram search for filenames
CREATE EXTENSION IF NOT EXISTS "citext";     -- case-insensitive email/username
