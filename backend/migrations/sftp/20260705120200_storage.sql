-- +goose Up

-- Folders ----------------------------------------------------------------
CREATE TABLE folders (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id  UUID REFERENCES folders(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    path       TEXT NOT NULL,
    depth      INT NOT NULL DEFAULT 0,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_folders_owner  ON folders(owner_id);
CREATE INDEX idx_folders_parent ON folders(parent_id);
CREATE INDEX idx_folders_path   ON folders USING gin (path gin_trgm_ops);
CREATE UNIQUE INDEX uq_folders_name ON folders(owner_id, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid), name) WHERE deleted_at IS NULL;

-- Files ------------------------------------------------------------------
CREATE TABLE files (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_id       UUID REFERENCES folders(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    extension       VARCHAR(32) NOT NULL DEFAULT '',
    mime_type       VARCHAR(160) NOT NULL DEFAULT 'application/octet-stream',
    size_bytes      BIGINT NOT NULL DEFAULT 0,
    checksum_sha256 CHAR(64),
    storage_key     TEXT NOT NULL,
    thumbnail_key   TEXT,
    is_starred      BOOLEAN NOT NULL DEFAULT FALSE,
    version_no      INT NOT NULL DEFAULT 1,
    download_count  BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX idx_files_owner    ON files(owner_id);
CREATE INDEX idx_files_folder   ON files(folder_id);
CREATE INDEX idx_files_deleted  ON files(deleted_at);
CREATE INDEX idx_files_name     ON files USING gin (name gin_trgm_ops);
CREATE INDEX idx_files_checksum ON files(checksum_sha256);
CREATE UNIQUE INDEX uq_files_name ON files(owner_id, COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'::uuid), name) WHERE deleted_at IS NULL;

-- File versions ----------------------------------------------------------
CREATE TABLE file_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id         UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    version_no      INT NOT NULL,
    size_bytes      BIGINT NOT NULL,
    checksum_sha256 CHAR(64),
    storage_key     TEXT NOT NULL,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (file_id, version_no)
);

-- Tags -------------------------------------------------------------------
CREATE TABLE tags (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name     VARCHAR(64) NOT NULL,
    color    VARCHAR(16) NOT NULL DEFAULT '#6366f1',
    UNIQUE (owner_id, name)
);

CREATE TABLE file_tags (
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (file_id, tag_id)
);

-- Favorites --------------------------------------------------------------
CREATE TABLE favorites (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_id    UUID REFERENCES files(id) ON DELETE CASCADE,
    folder_id  UUID REFERENCES folders(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (file_id IS NOT NULL OR folder_id IS NOT NULL),
    UNIQUE (user_id, file_id, folder_id)
);

-- Uploads (resumable / chunked) -----------------------------------------
CREATE TABLE uploads (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_id       UUID REFERENCES folders(id) ON DELETE SET NULL,
    filename        VARCHAR(255) NOT NULL,
    total_size      BIGINT NOT NULL,
    chunk_size      BIGINT NOT NULL,
    total_chunks    INT NOT NULL,
    uploaded_chunks INT NOT NULL DEFAULT 0,
    received_bytes  BIGINT NOT NULL DEFAULT 0,
    temp_key        TEXT NOT NULL,
    checksum_sha256 CHAR(64),
    status          VARCHAR(24) NOT NULL DEFAULT 'in_progress',
    file_id         UUID REFERENCES files(id) ON DELETE SET NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_uploads_user   ON uploads(user_id);
CREATE INDEX idx_uploads_status ON uploads(status);

CREATE TABLE upload_chunks (
    upload_id   UUID NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    size_bytes  BIGINT NOT NULL,
    checksum    CHAR(64),
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (upload_id, chunk_index)
);

-- Downloads --------------------------------------------------------------
CREATE TABLE downloads (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id    UUID REFERENCES files(id) ON DELETE SET NULL,
    user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    share_id   UUID,
    bytes_sent BIGINT NOT NULL DEFAULT 0,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_downloads_file ON downloads(file_id);
CREATE INDEX idx_downloads_user ON downloads(user_id);
CREATE INDEX idx_downloads_time ON downloads(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS downloads;
DROP TABLE IF EXISTS upload_chunks;
DROP TABLE IF EXISTS uploads;
DROP TABLE IF EXISTS favorites;
DROP TABLE IF EXISTS file_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS file_versions;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS folders;
