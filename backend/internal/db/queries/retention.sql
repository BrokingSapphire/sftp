-- name: SetFileLegalHold :exec
UPDATE files SET legal_hold = @hold, updated_at = now() WHERE id = @id;

-- name: SetFileRetention :exec
UPDATE files SET retain_until = @retain_until, updated_at = now() WHERE id = @id;
