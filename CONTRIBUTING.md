# Contributing

Thanks for your interest in improving the SFTP platform. This is an open-source,
self-hosted file-transfer platform for on-premise deployments.

## Development setup

Prerequisites: Go 1.25+, Docker, Node.js 22+, and [go-task](https://taskfile.dev).

```bash
cd backend
task bootstrap        # env files, deps, infra, migrations
task dev              # run API with hot reload
```

## Project layout

```
backend/
  cmd/server          entrypoint
  internal/
    api/              chi router, handlers, middleware
    config/           viper + env loader
    db/               pgx pool, tx helpers, sqlc-generated code
    repository/       data-access layer (per domain)
    service/          business logic (per domain)
    models/           request/response DTOs
    apperrors/        domain errors + HTTP mapping
    httpresponse/     uniform response builder
  pkg/                reusable, service-agnostic packages
  migrations/sftp/    goose SQL migrations (source of truth)
frontend/             Next.js 16 app-router client
docker/               deployment assets
docs/                 guides
```

## Conventions

- **Database**: migrations are the source of truth; never auto-create tables.
  Add one with `task db:create -- <name>`, then regenerate with `task gen`.
- **Errors**: return sentinels from `internal/apperrors`; handlers use
  `NewResponse(w, r).Fail(err)`.
- **Logging**: use the context logger; every state-changing action is audited.
- **Security**: validate all input, sanitise all paths, never log secrets.

## Before opening a PR

```bash
task check            # fmt + vet + tests
```

Please keep commits focused and write clear messages. By contributing you agree
your work is licensed under the MIT License.
