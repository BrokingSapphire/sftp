-- name: CreateTeam :one
INSERT INTO teams (name, slug, description, storage_quota, color, created_by)
VALUES (@name, @slug, @description, @storage_quota, @color, @created_by)
RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams WHERE id = $1;

-- name: UpdateTeam :exec
UPDATE teams SET name = @name, description = @description, storage_quota = @storage_quota, color = @color, updated_at = now()
WHERE id = @id;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = $1;

-- name: AddTeamStorage :exec
UPDATE teams SET storage_used = GREATEST(0, storage_used + @delta) WHERE id = @id;

-- name: ListTeamsForUser :many
SELECT t.*, tm.role AS member_role,
       (SELECT count(*) FROM team_members x WHERE x.team_id = t.id) AS member_count
FROM teams t
JOIN team_members tm ON tm.team_id = t.id
WHERE tm.user_id = @user_id
ORDER BY t.name;

-- name: GetTeamMembership :one
SELECT role FROM team_members WHERE team_id = @team_id AND user_id = @user_id;

-- name: AddTeamMember :exec
INSERT INTO team_members (team_id, user_id, role)
VALUES (@team_id, @user_id, @role)
ON CONFLICT (team_id, user_id) DO UPDATE SET role = EXCLUDED.role;

-- name: RemoveTeamMember :exec
DELETE FROM team_members WHERE team_id = @team_id AND user_id = @user_id;

-- name: ListTeamMembers :many
SELECT tm.user_id, tm.role, u.full_name, u.username, u.email,
       (u.avatar_path IS NOT NULL AND u.avatar_path <> '') AS has_avatar
FROM team_members tm
JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = @team_id
ORDER BY tm.role, u.full_name;
