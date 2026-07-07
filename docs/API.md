# API Reference

Base path: `/api/v1`. Responses use a uniform envelope; errors are RFC 7807
`application/problem+json`.

## Authentication

Send **either**:

- `Authorization: Bearer <access_token>` (web sessions), or
- `X-API-Key: sftp_<prefix>_<secret>` (programmatic).

Interactive docs (Swagger UI) are served in development at `/swagger/index.html`,
and the OpenAPI spec is generated to `backend/docs/openapi.json`.

## Response envelope

```json
{ "success": true, "message": "…", "data": { }, "meta": { },
  "request_id": "…", "timestamp": 1730000000000 }
```

## Endpoint groups

| Group | Prefix | Auth |
| ----- | ------ | ---- |
| Health | `/health-check`, `/info`, `/healthz`, `/readyz` | none |
| Auth | `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/me`, `/auth/change-password` | mixed |
| SSO | `/auth/sso/microsoft/login`, `/auth/sso/microsoft/callback` | none |
| Users | `/users` (CRUD, role, quota, status, reset-password) | `users.*` |
| Roles | `/roles` | `users.read` |
| Folders | `/folders` (create, rename, move, star, delete) | `folders.*` |
| Files | `/files` (list, get, rename, move, star, trash, restore, delete, recent, starred, search) | `files.*` |
| Uploads | `/files/upload` (simple), `/files/uploads` (init/chunk/status/complete/abort) | `files.upload` |
| Download | `/files/{id}/download` (range) | `files.read` |
| Shares | `/shares` (create/list/revoke), public `/share/{token}`, `/share/{token}/download` | mixed |
| API keys | `/api-keys` (create/list/revoke) | `apikeys.manage` |
| Audit | `/audit` (read), `/activity` (telemetry ingest) | `audit.read` / any |

## Example: resumable upload

```bash
# 1. init
curl -X POST /api/v1/files/uploads -H "Authorization: Bearer $T" \
  -d '{"filename":"big.zip","total_size":5368709120,"chunk_size":8388608}'
# → { upload_id, total_chunks, received_chunks }

# 2. PUT each chunk (raw body)
curl -X PUT "/api/v1/files/uploads/$ID/chunks/0" -H "Authorization: Bearer $T" \
  --data-binary @chunk0

# 3. complete → returns the created file
curl -X POST "/api/v1/files/uploads/$ID/complete" -H "Authorization: Bearer $T"
```

Interrupted? Call `GET /files/uploads/{id}` to see `received_chunks` and resume.

## Single active session

Each account may have only one active session at a time.

- `POST /api/v1/auth/login` returns **409 Conflict** (`a session is already
  active for this account`) if another session is live.
- Re-send the same request with `"force": true` to terminate the existing
  session and sign in here. The web UI prompts the user before doing so.
