-- +goose Up
-- Anomalies detected on the audit stream (mass downloads, login bursts, bulk
-- deletes, share spikes, off-hours activity). Surfaced to super admins.
CREATE TABLE security_alerts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type         VARCHAR(48) NOT NULL,
    severity     VARCHAR(16) NOT NULL DEFAULT 'medium',
    actor_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_email  CITEXT,
    summary      TEXT NOT NULL,
    event_count  INT NOT NULL DEFAULT 0,
    window_start TIMESTAMPTZ,
    window_end   TIMESTAMPTZ,
    metadata     JSONB NOT NULL DEFAULT '{}',
    resolved     BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    resolved_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_alerts_created ON security_alerts (created_at DESC);
CREATE INDEX idx_alerts_unresolved ON security_alerts (resolved) WHERE resolved = FALSE;

-- +goose Down
DROP TABLE IF EXISTS security_alerts;
