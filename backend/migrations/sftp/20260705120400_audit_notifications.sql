-- +goose Up

-- Audit logs (compliance-grade, append-only) ----------------------------
CREATE TABLE audit_logs (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_email CITEXT,
    action      VARCHAR(96) NOT NULL,
    category    VARCHAR(48) NOT NULL,
    object_type VARCHAR(48),
    object_id   VARCHAR(64),
    object_name TEXT,
    result      VARCHAR(24) NOT NULL DEFAULT 'success',
    ip_address  INET,
    user_agent  TEXT,
    browser     VARCHAR(96),
    os          VARCHAR(96),
    request_id  VARCHAR(64),
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_actor    ON audit_logs(actor_id);
CREATE INDEX idx_audit_action   ON audit_logs(action);
CREATE INDEX idx_audit_category ON audit_logs(category);
CREATE INDEX idx_audit_time     ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_object   ON audit_logs(object_type, object_id);

-- Revoke UPDATE/DELETE at the DB level to keep the trail immutable.
-- (Application connects as a non-owner role in production; see docs.)

-- User activity / click telemetry ---------------------------------------
CREATE TABLE user_activity (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
    event_type VARCHAR(48) NOT NULL,
    element    VARCHAR(160),
    path       TEXT,
    ip_address INET,
    user_agent TEXT,
    metadata   JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_activity_user ON user_activity(user_id);
CREATE INDEX idx_activity_time ON user_activity(created_at DESC);
CREATE INDEX idx_activity_type ON user_activity(event_type);

-- Notifications ----------------------------------------------------------
CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       VARCHAR(48) NOT NULL,
    title      VARCHAR(200) NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    link       TEXT,
    is_read    BOOLEAN NOT NULL DEFAULT FALSE,
    metadata   JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at    TIMESTAMPTZ
);
CREATE INDEX idx_notifications_user   ON notifications(user_id);
CREATE INDEX idx_notifications_unread ON notifications(user_id) WHERE is_read = FALSE;

-- Settings ---------------------------------------------------------------
CREATE TABLE settings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope      VARCHAR(16) NOT NULL DEFAULT 'system',
    user_id    UUID REFERENCES users(id) ON DELETE CASCADE,
    key        VARCHAR(96) NOT NULL,
    value      JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scope, user_id, key)
);

-- +goose Down
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS user_activity;
DROP TABLE IF EXISTS audit_logs;
