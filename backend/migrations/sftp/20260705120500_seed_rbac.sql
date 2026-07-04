-- +goose Up

-- Built-in roles.
INSERT INTO roles (name, slug, description, is_system, priority) VALUES
    ('Super Admin', 'super_admin', 'Full unrestricted access',   TRUE, 100),
    ('Admin',       'admin',       'Administrative access',       TRUE, 80),
    ('Manager',     'manager',     'Team/department management',  TRUE, 60),
    ('Employee',    'employee',    'Standard user',               TRUE, 40),
    ('Guest',       'guest',       'Limited read-only access',    TRUE, 20)
ON CONFLICT (slug) DO NOTHING;

-- Permission catalogue.
INSERT INTO permissions (slug, category, description) VALUES
    ('files.read',      'files',   'View and download files'),
    ('files.upload',    'files',   'Upload new files'),
    ('files.write',     'files',   'Rename/edit/move files'),
    ('files.delete',    'files',   'Delete files'),
    ('files.share',     'files',   'Create share links'),
    ('folders.read',    'folders', 'View folders'),
    ('folders.write',   'folders', 'Create/rename/move folders'),
    ('folders.delete',  'folders', 'Delete folders'),
    ('users.read',      'users',   'View users'),
    ('users.manage',    'users',   'Create/edit/disable users'),
    ('roles.manage',    'roles',   'Manage roles and permissions'),
    ('audit.read',      'audit',   'View audit logs'),
    ('storage.manage',  'storage', 'Manage storage locations/quotas'),
    ('settings.manage', 'admin',   'Manage system settings'),
    ('apikeys.manage',  'apikeys', 'Create/revoke own API keys'),
    ('admin.all',       'admin',   'Full administrative control')
ON CONFLICT (slug) DO NOTHING;

-- +goose StatementBegin
DO $$
DECLARE
    r_super UUID; r_admin UUID; r_mgr UUID; r_emp UUID; r_guest UUID;
BEGIN
    SELECT id INTO r_super FROM roles WHERE slug='super_admin';
    SELECT id INTO r_admin FROM roles WHERE slug='admin';
    SELECT id INTO r_mgr   FROM roles WHERE slug='manager';
    SELECT id INTO r_emp   FROM roles WHERE slug='employee';
    SELECT id INTO r_guest FROM roles WHERE slug='guest';

    INSERT INTO role_permissions (role_id, permission_id)
        SELECT r_super, id FROM permissions
        ON CONFLICT DO NOTHING;

    INSERT INTO role_permissions (role_id, permission_id)
        SELECT r_admin, id FROM permissions
        WHERE slug IN ('files.read','files.upload','files.write','files.delete','files.share',
                       'folders.read','folders.write','folders.delete',
                       'users.read','users.manage','audit.read','storage.manage',
                       'settings.manage','apikeys.manage')
        ON CONFLICT DO NOTHING;

    INSERT INTO role_permissions (role_id, permission_id)
        SELECT r_mgr, id FROM permissions
        WHERE slug IN ('files.read','files.upload','files.write','files.delete','files.share',
                       'folders.read','folders.write','folders.delete',
                       'users.read','apikeys.manage')
        ON CONFLICT DO NOTHING;

    INSERT INTO role_permissions (role_id, permission_id)
        SELECT r_emp, id FROM permissions
        WHERE slug IN ('files.read','files.upload','files.write','files.delete','files.share',
                       'folders.read','folders.write','folders.delete','apikeys.manage')
        ON CONFLICT DO NOTHING;

    INSERT INTO role_permissions (role_id, permission_id)
        SELECT r_guest, id FROM permissions
        WHERE slug IN ('files.read','folders.read')
        ON CONFLICT DO NOTHING;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM role_permissions
    WHERE role_id IN (SELECT id FROM roles WHERE is_system = TRUE);
-- +goose StatementEnd
DELETE FROM roles WHERE is_system = TRUE;
DELETE FROM permissions;
