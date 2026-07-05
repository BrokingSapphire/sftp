# Troubleshooting

### Backend exits: `JWT_SECRET must be set and at least 32 characters`
Set a 32+ char `JWT_SECRET` in `.env`. Generate: `openssl rand -base64 48`.

### Backend can't connect to the database
- Check `DATABASE_URL` host — inside compose it is `postgres`, not `localhost`.
- Ensure Postgres is healthy: `docker compose ps`, `docker compose logs postgres`.

### `readyz` returns 503
The database is unreachable. Check the Postgres container and credentials.

### Can't log in / no admin account
Set `BOOTSTRAP_ADMIN_PASSWORD` **before the first boot** (empty DB only). If the
DB already has users, create accounts via an existing admin, or reset a password
with `task db` + SQL as a last resort.

### Uploads fail for large files behind a proxy
The bundled Nginx sets `client_max_body_size 0` and disables request buffering.
If you use your own proxy, replicate those settings and raise read/send timeouts.

### 401 immediately after login
The access token expired and refresh failed. Confirm the frontend can reach
`/api/v1/auth/refresh` and that the refresh token is stored (check browser
localStorage `sftp_refresh_token`).

### Microsoft SSO: `redirect_uri_mismatch`
`sso.microsoft.redirect_url` must exactly match a redirect URI registered on the
Entra app registration, including scheme and path.

### Migrations didn't apply
The backend runs `goose up` on start. Check backend logs. To run manually:
`cd backend && task db:migrate`.

### Reset everything (destroys data)
```bash
docker compose down -v
```

### Inspect logs
```bash
docker compose logs -f backend
docker compose logs -f nginx
```
