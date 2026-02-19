# Phase 0 — Project Scaffolding

Build Phase 0 from `docs/SPEC.md` — project scaffolding.

## Deliverables

1. `go mod init github.com/superset-studio/kapstan/api` (in `api/` directory) with Go 1.25
2. `api/cmd/kapstan/main.go` — cobra CLI with `kapstan server` command that starts an HTTP server
3. Echo HTTP router with:
   - `GET /healthz` — returns `{"status": "ok"}`
   - `GET /readyz` — returns 200 if DB is reachable, 503 if not
4. PostgreSQL connection setup using `sqlx` (with `pgx` stdlib driver) with connection pooling
5. golang-migrate integration for running migrations on startup (or via `kapstan migrate` subcommand). Migration files live in `api/migrations/`.
6. Configuration loading from environment variables:
   - `DATABASE_URL` (required)
   - `PORT` (default: 8080)
   - `KAPSTAN_ENCRYPTION_KEY` (required, validated as 32-byte hex)
   - `LOG_LEVEL` (default: info)
   - `LOG_FORMAT` (default: json, also supports text)
7. Structured logging with `slog` — JSON output by default, text for local dev
8. React app skeleton in `web/` (standalone service, not embedded):
   - Vite + TypeScript + React 18
   - `package.json` with scripts: `dev`, `build`, `lint`, `preview`
   - A simple landing page that says "Kapstan" and renders successfully
   - `tsconfig.json` with strict mode
   - Vite dev server proxies `/api` and `/healthz`/`/readyz` to the Go backend
9. Dockerfile for Go backend (`deploy/Dockerfile`):
    - Stage 1: `golang:1.25-alpine` — build Go binary
    - Stage 2: `alpine` — final image with just the binary
10. Dockerfile for frontend (`deploy/Dockerfile.web`):
    - Stage 1: `node:20-alpine` — build React app
    - Stage 2: `nginx:alpine` — serve static files
11. Makefile with targets:
    - `build` — build Go binary
    - `dev` — start Go server with `air` (hot reload) + Vite dev server concurrently
    - `test` — `go test ./...`
    - `lint` — `golangci-lint run`
    - `migrate` — run migrations via CLI
    - `docker` — build backend Docker image
    - `ci` — lint + test + build
12. GitHub Actions CI workflow (`.github/workflows/ci.yml`): checkout, setup Go, setup Node, install deps, lint, test, build
13. A `.golangci.yml` with sensible defaults
14. A first migration file `api/migrations/000001_init.up.sql` that is empty (or creates a simple `_health` check table) and its corresponding `down.sql` — just to prove the migration system works
15. A `README.md` with: what Kapstan is (one sentence), how to run locally (`make dev`), how to build (`make build`), environment variables

## Tech Stack (Use Exactly)

- **Go 1.25**
- **Echo** for HTTP framework
- **sqlx** for database access (write SQL directly, scan into structs)
- **golang-migrate** for schema migrations
- **slog** for structured logging
- **cobra** for CLI

## Constraints

- Do NOT create any domain tables (users, orgs, etc.) — that's Phase 1
- The Go binary must be runnable with just `DATABASE_URL` and `KAPSTAN_ENCRYPTION_KEY` set
- `make build` produces the Go API server binary. The frontend is built separately.
- Keep dependencies minimal — only add what's needed for this phase
- Use the project structure from `docs/SPEC.md` Section 2.1

## Verification

After building, these should all work:

```bash
# Build succeeds
make build

# Binary starts (with a running PostgreSQL)
DATABASE_URL=postgres://localhost:5432/kapstan KAPSTAN_ENCRYPTION_KEY=$(openssl rand -hex 32) ./bin/kapstan server

# Health check responds
curl http://localhost:8080/healthz
# → {"status": "ok"}

# Frontend dev server runs separately
cd web && npm run dev
# → http://localhost:5173

# Docker build succeeds
make docker
```
