-- +goose Up

-- Share links -----------------------------------------------------------
CREATE TABLE shares (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token          VARCHAR(64) NOT NULL UNIQUE,
    owner_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_id        UUID REFERENCES files(id) ON DELETE CASCADE,
    folder_id      UUID REFERENCES folders(id) ON DELETE CASCADE,
    permission     VARCHAR(16) NOT NULL DEFAULT 'read',
    password_hash  TEXT,
    download_limit INT,
    download_count INT NOT NULL DEFAULT 0,
    expires_at     TIMESTAMPTZ,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (file_id IS NOT NULL OR folder_id IS NOT NULL)
);
CREATE INDEX idx_shares_owner ON shares(owner_id);
CREATE INDEX idx_shares_token ON shares(token);

ALTER TABLE downloads
    ADD CONSTRAINT fk_downloads_share
    FOREIGN KEY (share_id) REFERENCES shares(id) ON DELETE SET NULL;

-- Direct internal shares to specific users ------------------------------
CREATE TABLE share_recipients (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    share_id   UUID NOT NULL REFERENCES shares(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (share_id, user_id)
);

-- Resource ACLs ----------------------------------------------------------
CREATE TABLE resource_permissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id         UUID REFERENCES files(id) ON DELETE CASCADE,
    folder_id       UUID REFERENCES folders(id) ON DELETE CASCADE,
    grantee_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    grantee_role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    can_read     BOOLEAN NOT NULL DEFAULT TRUE,
    can_write    BOOLEAN NOT NULL DEFAULT FALSE,
    can_delete   BOOLEAN NOT NULL DEFAULT FALSE,
    can_move     BOOLEAN NOT NULL DEFAULT FALSE,
    can_share    BOOLEAN NOT NULL DEFAULT FALSE,
    can_download BOOLEAN NOT NULL DEFAULT TRUE,
    can_upload   BOOLEAN NOT NULL DEFAULT FALSE,
    inherit      BOOLEAN NOT NULL DEFAULT TRUE,
    created_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (file_id IS NOT NULL OR folder_id IS NOT NULL),
    CHECK (grantee_user_id IS NOT NULL OR grantee_role_id IS NOT NULL)
);
CREATE INDEX idx_resperm_file   ON resource_permissions(file_id);
CREATE INDEX idx_resperm_folder ON resource_permissions(folder_id);
CREATE INDEX idx_resperm_user   ON resource_permissions(grantee_user_id);
CREATE INDEX idx_resperm_role   ON resource_permissions(grantee_role_id);

-- +goose Down
DROP TABLE IF EXISTS resource_permissions;
DROP TABLE IF EXISTS share_recipients;
ALTER TABLE downloads DROP CONSTRAINT IF EXISTS fk_downloads_share;
DROP TABLE IF EXISTS shares;
