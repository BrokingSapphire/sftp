# Frequently Asked Questions

- [General](#general)
- [Deployment & operations](#deployment--operations)
- [Security & compliance](#security--compliance)
- [Features](#features)
- [Development](#development)

## General

### What is Sapphire SFTP?

A self-hosted, on-premise file platform with Google Drive / Dropbox-style
features, a native SFTP endpoint, and a REST API — plus enterprise security,
compliance, and optional on-prem AI. Everything runs on your infrastructure.

### Is anything sent to the cloud?

No. There are no cloud dependencies. Even the optional AI features run on a
**self-hosted Ollama** container inside your network.

### Is it really free / open source?

Yes — released under the [MIT License](../LICENSE). You can use, modify, and
deploy it freely.

### Can I rebrand it as my own product?

Yes. Everything (name, logo, colours, emails, domains) is driven by
`brand.config.json`. Edit it, drop your logo in `frontend/public/`, and rebuild.
`deploy.sh` walks you through it interactively.

## Deployment & operations

### What are the minimum requirements?

A Linux host with Docker + Docker Compose, ~2 vCPU / 4 GB RAM for a small team,
and disk sized for your files. AI/Office profiles need more (models are several
GB; OnlyOffice ~2.5 GB).

### How do I deploy in one command?

```bash
./deploy.sh
```

It asks a few company questions, generates config + secrets, and starts the
stack. See the [README Quick Start](../README.md#-quick-start).

### How do I enable HTTPS?

Terminate TLS at Nginx (add certificates and a `443` server block) or put the
stack behind your existing reverse proxy / load balancer. See
[DEPLOYMENT.md](DEPLOYMENT.md).

### How do backups work?

A super-admin points a backup at a directory (e.g. a mounted removable disk). The
first run is a **full** encrypted backup; later runs are **incremental** (only
new/changed files). See [BACKUP.md](BACKUP.md). It's also available via
`POST /api/v1/admin/backup` for cron automation.

### I forgot the admin password.

The first login forces a password change. If the account is locked out, an
operator can reset the password hash directly in the database, or you can
re-bootstrap on an empty database via `BOOTSTRAP_ADMIN_PASSWORD`.

## Security & compliance

### Are files encrypted?

Set `STORAGE_ENCRYPTION_KEY` to enable AES-256 encryption at rest. The same key
encrypts backups. **If you lose it, encrypted data is unrecoverable** — store it
securely.

### Do the web UI and SFTP share permissions?

Yes. The web app, REST API, and SFTP endpoint use the same accounts, RBAC, and
audit trail.

### Can I restrict Microsoft SSO to my company?

Yes. Register a **single-tenant** Azure app and set your tenant GUID. Sign-in is
further restricted to `org.domains` by default, so guests/personal accounts are
rejected.

### What is logged?

Every meaningful action (and every UI click via telemetry) is written to an
immutable audit trail with actor, IP, device, and object. A background detector
raises alerts on anomalies (mass downloads, brute force, bulk deletes).

## Features

### What files can be previewed?

Images, PDF, audio/video, text/CSV/JSON/Markdown, and all common Office formats
(docx/xlsx/pptx) — rendered client-side.

### How does versioning work?

Re-uploading a file with the same name (or saving in the in-app editor) archives
the current content as a version and bumps the file. You can download or restore
any previous version.

### What is the "Common" area?

An organisation-wide space visible to everyone. It's **unlimited** — files there
don't count against personal quotas.

### How do I use the desktop sync agent?

Build `synccli` and point it at a folder with an API key. Use `--watch` to keep
syncing on change. See the agent's README in `backend/cmd/synccli/`.

## Development

### Where does the App Router live?

In **`frontend/app/`** (root app routing), not `src/app`.

### How do I add a database query?

Edit `backend/internal/db/queries/*.sql`, then run `sqlc generate`. Schema
changes go in `backend/migrations/sftp/` (goose), applied at startup.

### How do I run the tests?

```bash
(cd backend && go test ./...)
(cd frontend && npm run typecheck && npm test)
```

Still stuck? Ask in [Discussions](https://github.com/BrokingSapphire/sftp/discussions)
or see [TROUBLESHOOTING.md](TROUBLESHOOTING.md).
