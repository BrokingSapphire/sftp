# SFTP — Enterprise Self-Hosted File Transfer Platform

A production-grade, on-premise file management platform (Google Drive / Dropbox Business style)
built for internal corporate networks. Fully self-hosted, no cloud dependencies.

## Stack

| Layer          | Technology                                   |
| -------------- | -------------------------------------------- |
| Backend        | Go 1.25+, Gin, GORM, PostgreSQL, JWT, Argon2 |
| Frontend       | Next.js 16 (App Router), TypeScript, Tailwind, shadcn/ui, TanStack Query |
| Database       | PostgreSQL 16                                |
| Cache / Jobs   | Redis (optional)                             |
| Storage        | Local Linux filesystem (mounted drives / NAS)|
| Reverse Proxy  | Nginx                                         |
| Deployment     | Docker + Docker Compose                       |
| Logging        | Zap (structured)                              |
| Config         | Viper + `.env`                                |

## Repository Layout

```
backend/     Go API server (clean architecture: config, db, api, services, repos)
frontend/    Next.js 16 web client
docker/      Dockerfiles, compose, nginx config
docs/        Installation, deployment, architecture, API, backup/restore guides
scripts/     Operational helper scripts
```

## Quick Start (Development)

```bash
# 1. Backend
cd backend
cp ../.env.example .env
go run ./cmd/server

# 2. Frontend
cd frontend
npm install
npm run dev
```

## Documentation

See [`docs/`](./docs) for the Installation, Deployment, Architecture, Database Schema,
API, Backup, Restore, Upgrade and Troubleshooting guides.

## Status

Under active development. Built incrementally — see commit history.
