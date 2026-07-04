# SFTP — Enterprise Self-Hosted File Transfer Platform

A production-grade, on-premise file management platform (Google Drive / Dropbox
Business style) built for internal corporate networks. Fully self-hosted, no
cloud dependencies. Open source (MIT).

## Stack

| Layer          | Technology                                              |
| -------------- | ------------------------------------------------------- |
| Backend        | Go 1.25+, chi, pgx, sqlc, goose, JWT, Argon2id          |
| Frontend       | Next.js 16 (App Router), TypeScript, Tailwind, shadcn/ui, TanStack Query |
| Database       | PostgreSQL 16                                           |
| Cache / Jobs   | Redis (optional)                                        |
| Storage        | Local Linux filesystem (mounted drives / NAS)           |
| Protocols      | REST API (`/api/v1`) + native SFTP over SSH             |
| Reverse Proxy  | Nginx                                                    |
| Task runner    | go-task (`Taskfile.yml`)                                |
| Logging        | Zap (structured, multi-sink)                            |
| Config         | Viper + YAML + `.env`                                   |
| Deployment     | Docker + Docker Compose                                 |

## Repository layout

```
backend/     Go API + SFTP server (chi/pgx/sqlc, clean architecture)
frontend/    Next.js 16 web client (app-router)
docker/      Deployment assets (compose overrides, nginx)
docs/        Installation, deployment, architecture, API, backup guides
scripts/     Operational helper scripts
```

## Quick start (development)

```bash
cd backend
task bootstrap        # copies env files, installs deps, starts Postgres, migrates
task dev              # runs the API with hot reload (air)
```

Or bring the whole stack up in Docker:

```bash
cd backend
task up               # Postgres + API + SFTP  →  http://localhost:8080
```

Set `DATABASE_URL` and a 32+ char `JWT_SECRET` in `.env` first (see
`.env.example`). Non-secret settings live in `config.yaml` (see
`config.example.yaml`).

## Documentation

See [`docs/`](./docs) for the Installation, Deployment, Architecture, Database
Schema, API, Backup, Restore, Upgrade and Troubleshooting guides.

## License

[MIT](./LICENSE). Contributions welcome — see [CONTRIBUTING.md](./CONTRIBUTING.md).

## Status

Under active development, built incrementally. See commit history.
