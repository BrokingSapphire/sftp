-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: GetRoleBySlug :one
SELECT * FROM roles WHERE slug = $1;

-- name: ListRoles :many
SELECT * FROM roles ORDER BY priority DESC;

-- name: CreateRole :one
INSERT INTO roles (name, slug, description, is_system, priority)
VALUES ($1, $2, $3, FALSE, $4)
RETURNING *;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1 AND is_system = FALSE;

-- name: GetPermissionsForRole :many
SELECT p.slug
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
WHERE rp.role_id = $1
ORDER BY p.slug;

-- name: GetPermissionsForUser :many
SELECT p.slug
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN users u ON u.role_id = rp.role_id
WHERE u.id = $1
ORDER BY p.slug;

-- name: ListPermissions :many
SELECT * FROM permissions ORDER BY category, slug;

-- name: SetRolePermissions :exec
INSERT INTO role_permissions (role_id, permission_id)
SELECT $1, p.id FROM permissions p WHERE p.slug = ANY($2::text[])
ON CONFLICT DO NOTHING;

-- name: ClearRolePermissions :exec
DELETE FROM role_permissions WHERE role_id = $1;
