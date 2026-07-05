# Installation (Development)

## Prerequisites

- Go 1.25+
- Node.js 22+
- Docker (for PostgreSQL) and [go-task](https://taskfile.dev)

## Backend

```bash
cd backend
cp .env.example .env          # set DATABASE_URL + JWT_SECRET (min 32 chars)
task dev:infra                # start Postgres in Docker
task db:migrate               # apply migrations
task dev                      # run API with hot reload (air) on :8080
```

Handy tasks:

```bash
task            # list all tasks
task gen        # regenerate sqlc code after editing queries/migrations
task test       # run tests
task build      # build the binary
```

The interactive API docs (Swagger UI) are served in development at
`http://localhost:8080/swagger/index.html`.

## Frontend

```bash
cd frontend
cp .env.example .env.local     # NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
npm install
npm run dev                    # http://localhost:3000
```

The dev server proxies `/api/*` to the backend, so there is no CORS setup needed.

## Create the first user

Set `BOOTSTRAP_ADMIN_PASSWORD` in `backend/.env` before the first run; the backend
creates an `admin` super-admin on an empty database. Then log in at
`http://localhost:3000/login`.
