# Contributing to Sapphire SFTP

First off — **thank you** for taking the time to contribute! 🎉 Whether it's a
typo fix, a bug report, a new feature, or docs, every contribution is welcome and
valued.

This guide gets you from zero to a merged pull request. Please also read our
[Code of Conduct](CODE_OF_CONDUCT.md).

## Table of Contents

- [Ways to contribute](#ways-to-contribute)
- [Development setup](#development-setup)
- [Project layout](#project-layout)
- [Making a change](#making-a-change)
- [Commit conventions](#commit-conventions)
- [Coding guidelines](#coding-guidelines)
- [Tests](#tests)
- [Opening a pull request](#opening-a-pull-request)
- [Reporting bugs & requesting features](#reporting-bugs--requesting-features)

## Ways to contribute

- 🐛 **Report bugs** — [open a bug report](https://github.com/BrokingSapphire/sftp/issues/new?template=bug_report.yml)
- ✨ **Suggest features** — [open a feature request](https://github.com/BrokingSapphire/sftp/issues/new?template=feature_request.yml)
- 📝 **Improve docs** — even fixing a typo helps
- 💻 **Write code** — pick a [`good first issue`](https://github.com/BrokingSapphire/sftp/labels/good%20first%20issue) or discuss a bigger idea first
- 💬 **Help others** — answer questions in [Discussions](https://github.com/BrokingSapphire/sftp/discussions)

## Development setup

**Prerequisites:** Go 1.26+, Node.js 22+, Docker + Docker Compose. (`python3` for
`deploy.sh`.)

```bash
# 1. Fork, then clone your fork
git clone https://github.com/<you>/sftp.git
cd sftp

# 2. Fastest path — the whole stack in Docker
cp backend/.env.example .env    # set JWT_SECRET + POSTGRES_PASSWORD
docker compose up -d --build    # http://localhost
```

Or run each side natively for hot-reload development:

```bash
# Backend (needs a reachable Postgres + config/.env)
cd backend
go run ./cmd/server

# Frontend
cd frontend
npm install
npm run dev                     # http://localhost:3000
```

## Project layout

```text
backend/
  cmd/server/         API + SFTP entrypoint
  cmd/synccli/        desktop sync agent
  internal/
    api/              routing, handlers, middleware (Fuego)
    service/          business logic per domain
    db/queries/       .sql sources → sqlc-generated code in db/sftpdb
    storage/          filesystem engine (sharded, encrypted)
    worker/           background jobs
    config/, models/, apperrors/
  pkg/                reusable libs (argon2, jwt, filecrypt, dlp, cache, …)
  migrations/sftp/    goose SQL migrations (source of truth)
frontend/             Next.js 16 (root app/ routing)
docs/                 documentation
```

## Making a change

1. Create a branch off `main`:
   ```bash
   git checkout -b feat/short-description
   ```
2. Make focused commits (see [Commit conventions](#commit-conventions)).
3. Run the checks locally before pushing.
4. Push and [open a PR](#opening-a-pull-request).

Keep pull requests **small and focused** — one logical change per PR is much
easier to review and merge.

## Commit conventions

We follow [**Conventional Commits**](https://www.conventionalcommits.org/):

```
<type>(<optional scope>): <description>

[optional body]
```

Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`.

Examples:

```
feat(shares): add per-user viewer/editor grants
fix(upload): stream chunks to avoid buffering large files
docs(readme): document the AI profile
```

Do **not** add "Co-Authored-By" or tool attribution lines.

## Coding guidelines

**General**

- Match the surrounding code's style, naming, and comment density.
- Validate all input; sanitise all paths; **never log secrets**.
- Every state-changing action should be audited.

**Backend (Go)**

- Database access goes through **sqlc**: add/edit `internal/db/queries/*.sql`,
  then run `sqlc generate`. Migrations in `migrations/sftp/` are the source of
  truth — never auto-create tables at runtime.
- Return sentinel errors from `internal/apperrors`; handlers map them via
  `handlers.Fail(err)`.
- Keep `go vet ./...` and `gofmt` clean.

**Frontend (TypeScript / Next.js)**

- App Router lives in **`app/`** (not `src/app`).
- Type everything; keep `npm run typecheck` green.
- Reuse the design system in `components/ui` and brand tokens from
  `lib/brand.ts` — don't hard-code company names or colours.

## Tests

```bash
# Backend
cd backend && go test ./...          # add -race for concurrency-sensitive code

# Frontend
cd frontend && npm run typecheck && npm test
```

Add tests for new behaviour where it makes sense (crypto, parsing, business
rules especially).

## Opening a pull request

1. Ensure CI-equivalent checks pass locally:
   ```bash
   (cd backend && go vet ./... && go test ./...)
   (cd frontend && npm run typecheck && npm test && npm run build)
   ```
2. Push your branch and open a PR against `main`.
3. Fill in the PR template (summary, type, testing, checklist).
4. A maintainer will review. Please respond to feedback and keep the branch up
   to date. Once approved, we'll merge. 🎉

## Reporting bugs & requesting features

Use the issue templates — they collect the details we need to help quickly. For
**security vulnerabilities**, do **not** open a public issue; follow
[SECURITY.md](.github/SECURITY.md).

---

By contributing, you agree that your contributions are licensed under the
[MIT License](LICENSE).
