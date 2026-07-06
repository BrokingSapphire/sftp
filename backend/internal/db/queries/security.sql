-- Detection aggregations over the audit stream --------------------------------

-- name: CountActionsByActor :many
-- Groups matching actions by actor within a window, flagging actors at/over the
-- threshold. Pass a set of actions (e.g. ARRAY['file.download']).
SELECT actor_id, actor_email, COUNT(*)::int AS n,
       MIN(created_at) AS first_at, MAX(created_at) AS last_at
FROM audit_logs
WHERE created_at >= @since
  AND action = ANY(@actions::text[])
  AND actor_email IS NOT NULL
GROUP BY actor_id, actor_email
HAVING COUNT(*) >= @threshold;

-- name: CountFailedLoginsByEmail :many
SELECT actor_email, COUNT(*)::int AS n,
       MIN(created_at) AS first_at, MAX(created_at) AS last_at
FROM audit_logs
WHERE created_at >= @since
  AND action = 'auth.login'
  AND result <> 'success'
  AND actor_email IS NOT NULL
GROUP BY actor_email
HAVING COUNT(*) >= @threshold;

-- Alert lifecycle -------------------------------------------------------------

-- name: RecentAlertExists :one
SELECT EXISTS (
  SELECT 1 FROM security_alerts
  WHERE type = @type AND actor_email IS NOT DISTINCT FROM @actor_email
    AND created_at >= @since
);

-- name: InsertSecurityAlert :one
INSERT INTO security_alerts (type, severity, actor_id, actor_email, summary, event_count, window_start, window_end, metadata)
VALUES (@type, @severity, @actor_id, @actor_email, @summary, @event_count, @window_start, @window_end, @metadata)
RETURNING *;

-- name: ListSecurityAlerts :many
SELECT * FROM security_alerts
ORDER BY resolved ASC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUnresolvedAlerts :one
SELECT COUNT(*) FROM security_alerts WHERE resolved = FALSE;

-- name: ResolveSecurityAlert :exec
UPDATE security_alerts
SET resolved = TRUE, resolved_by = @resolved_by, resolved_at = now()
WHERE id = @id;

-- name: ListSuperAdminIDs :many
SELECT u.id FROM users u
JOIN roles r ON r.id = u.role_id
WHERE r.slug = 'super_admin' AND u.is_active = TRUE AND u.deleted_at IS NULL;
