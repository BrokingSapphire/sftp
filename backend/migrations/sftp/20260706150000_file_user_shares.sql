-- +goose Up
-- Enable upsert of a per-user file grant (internal "share with people").
CREATE UNIQUE INDEX idx_resperm_file_user
    ON resource_permissions(file_id, grantee_user_id)
    WHERE file_id IS NOT NULL AND grantee_user_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_resperm_file_user;
