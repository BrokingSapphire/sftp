# Architecture

## Overview

```
                    ┌──────────────┐
   Browser  ───────▶│    Nginx     │  :80/:443  reverse proxy, TLS, gzip
                    └──────┬───────┘
                 /         │        /api
        ┌────────▼──┐   ┌──▼───────────┐
        │ Next.js   │   │  Go backend  │  :8080 REST (Fuego) + :2222 SFTP
        │ frontend  │   └──┬───────┬───┘
        └───────────┘      │       │
                    ┌──────▼──┐  ┌─▼──────────────┐
                    │Postgres │  │ Local filesystem│  (mounted drive / NAS)
                    └─────────┘  └────────────────┘
```

On-premise, no cloud dependencies. File **content** lives on the filesystem;
all **metadata** (users, folders, files, shares, audit) lives in PostgreSQL.

## Backend layers (`backend/`)

```
cmd/server            entrypoint, dependency wiring, graceful shutdown
internal/
  api/                Fuego router, server, typed handlers, middleware
    handlers/         one package per domain (auth, user, file, share, apikey, audit, sso)
    handlers/middleware  request-id, logging, recover, auth (JWT+API key), RBAC, audit
    response/         uniform success envelope
    params/           validated path/query extraction
  config/             viper + env loader (defaults → yaml → .env → env)
  db/                 pgx pool, transaction helper, goose migration runner
    sftpdb/           sqlc-generated, type-safe queries
    queries/          hand-written SQL (source for sqlc)
  service/            business logic per domain
  storage/            local filesystem engine (sharded keys, chunk assembler)
  worker/             background cleanup jobs
  models/             request/response DTOs
  apperrors/          domain errors + HTTP mapping
migrations/sftp/      goose SQL migrations (source of truth for the schema)
pkg/                  reusable: logger, jwt, argon2, apikey, headers, reqctx
```

**Request flow:** Nginx → global middleware (request-id, logging, panic recovery,
security headers, audit) → per-group auth (JWT **or** API key) → per-route RBAC
(`RequirePermission`) → typed handler → service → sqlc/storage.

## Key design decisions

- **Storage keys are opaque and sharded** (`ab/cd/<uuid>`), never derived from
  user input — immune to path traversal and filename collisions.
- **Resumable chunked uploads**: chunks land in a temp dir keyed by upload id,
  reassembled on completion with a one-pass SHA-256; supports files > 5 GB and
  resume after interruption.
- **Streaming downloads** via `http.ServeContent` (HTTP range requests).
- **Auth**: short-lived HS256 access tokens + rotating opaque refresh tokens
  (SHA-256 hashed at rest); Argon2id password hashing; account lockout.
- **RBAC**: role → permission matrix in the DB; `admin.all` wildcard.
- **Audit** is append-only and written asynchronously (sync fallback, never drops);
  every state-changing request is recorded, plus UI click telemetry.
- **Migrations are the source of truth** — tables are never auto-created from models.

## Frontend (`frontend/`)

Next.js 16 App Router (root `app/`), TypeScript, Tailwind v4, TanStack Query,
React Hook Form + Zod. A typed API client handles JWT refresh transparently.
Route group `(app)` is auth-guarded and renders the sidebar/topbar shell.
