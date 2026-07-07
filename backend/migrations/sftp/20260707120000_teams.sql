-- +goose Up
-- Team Spaces: group-owned shared drives. Files/folders can belong to a team
-- (team_id) instead of a single user; access is governed by team membership.
CREATE TABLE teams (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    slug          VARCHAR(64) NOT NULL UNIQUE,
    description   TEXT NOT NULL DEFAULT '',
    storage_quota BIGINT NOT NULL DEFAULT 0, -- 0 = unlimited
    storage_used  BIGINT NOT NULL DEFAULT 0,
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE team_members (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       VARCHAR(16) NOT NULL DEFAULT 'member', -- owner | admin | member | viewer
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, user_id)
);
CREATE INDEX idx_team_members_user ON team_members (user_id);

ALTER TABLE files   ADD COLUMN team_id UUID REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE folders ADD COLUMN team_id UUID REFERENCES teams(id) ON DELETE CASCADE;
CREATE INDEX idx_files_team   ON files   (team_id) WHERE team_id IS NOT NULL;
CREATE INDEX idx_folders_team ON folders (team_id) WHERE team_id IS NOT NULL;

-- +goose Down
ALTER TABLE files   DROP COLUMN IF EXISTS team_id;
ALTER TABLE folders DROP COLUMN IF EXISTS team_id;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
