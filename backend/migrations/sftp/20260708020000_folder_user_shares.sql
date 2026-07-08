-- +goose Up
-- Enable upsert of a per-user folder grant (internal "share folder with people"),
-- mirroring the existing file-level grant index.
CREATE UNIQUE INDEX idx_resperm_folder_user
    ON resource_permissions(folder_id, grantee_user_id)
    WHERE folder_id IS NOT NULL AND grantee_user_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_resperm_folder_user;
