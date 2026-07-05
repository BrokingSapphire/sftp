# Environment & Configuration

Configuration resolves in this order (later overrides earlier):

1. Built-in defaults
2. `config.yaml` (non-secret settings; path via `CONFIG_FILE`)
3. `.env` file (path via `ENV_FILE`)
4. Process environment variables

Env var names derive from the YAML path: `SECTION_FIELD` (uppercase).
Secrets (`JWT_SECRET`, `DATABASE_URL`, SSO client secret) should come from the
environment, never the config file.

## Core

| Env var | YAML | Default | Notes |
| --- | --- | --- | --- |
| `APP_ENVIRONMENT` | `app.environment` | `local` | `local`/`development`/`staging`/`production` |
| `APP_PORT` | `app.port` | `8080` | REST API port |
| `APP_SELF_BASE_URL` | `app.self_base_url` | `http://localhost:8080` | Used for share links |
| `DATABASE_URL` | `database.url` | — | **required** Postgres DSN |
| `JWT_SECRET` | `jwt.secret` | — | **required**, min 32 chars |
| `JWT_ACCESS_TTL` | `jwt.access_ttl` | `15m` | Access-token lifetime |
| `JWT_REFRESH_TTL` | `jwt.refresh_ttl` | `168h` | Refresh-token lifetime |

## Storage

| Env var | Default | Notes |
| --- | --- | --- |
| `STORAGE_ROOT_PATH` | `./storage/files` | File content root (mount a drive here) |
| `STORAGE_TEMP_PATH` | `./storage/tmp` | Chunk assembly / temp |
| `STORAGE_MAX_UPLOAD_SIZE` | `0` | Bytes; `0` = unlimited |
| `STORAGE_CHUNK_SIZE` | `8388608` | 8 MiB default chunk |
| `STORAGE_TRASH_RETENTION_DAYS` | `30` | Recycle-bin purge window |

## Security

| Env var | Default | Notes |
| --- | --- | --- |
| `SECURITY_PASSWORD_MIN_LENGTH` | `12` | |
| `SECURITY_MAX_LOGIN_ATTEMPTS` | `5` | Then lockout |
| `SECURITY_LOCKOUT_DURATION` | `15m` | |
| `SECURITY_ARGON_MEMORY_KIB` | `65536` | Argon2id memory |

## Bootstrap admin

| Env var | Default |
| --- | --- |
| `BOOTSTRAP_ADMIN_EMAIL` | `admin@sapphirebroking.com` |
| `BOOTSTRAP_ADMIN_USERNAME` | `admin` |
| `BOOTSTRAP_ADMIN_PASSWORD` | — (unset = no admin created) |

## SFTP protocol server

| Env var | YAML | Default |
| --- | --- | --- |
| `SFTP_ENABLED` | `sftp.enabled` | `true` |
| `SFTP_PORT` | `sftp.port` | `2222` |
| `SFTP_HOST_KEY_PATH` | `sftp.host_key_path` | `./storage/ssh_host_ed25519_key` |

## Microsoft Entra ID SSO

| Env var | YAML | Notes |
| --- | --- | --- |
| — | `sso.microsoft.enabled` | `true` to enable |
| — | `sso.microsoft.tenant_id` | Tenant GUID / `organizations` |
| `SSO_MICROSOFT_CLIENT_ID` | `sso.microsoft.client_id` | App registration client id |
| `SSO_MICROSOFT_CLIENT_SECRET` | `sso.microsoft.client_secret` | Secret |
| — | `sso.microsoft.redirect_url` | Must match the app registration redirect URI |
| — | `sso.microsoft.allowed_domains` | Optional email-domain allowlist |

## CORS

| Env var | Default |
| --- | --- |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` |
