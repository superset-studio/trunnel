# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Kapstan — an open-source cloud management platform. Go backend API + React frontend as separate services, modular monolith, PostgreSQL as the only required dependency. See `docs/SPEC.md` for the full technical specification.

## Architecture

- **Modular monolith** — Go backend, service packages, no microservices
- **REST API** — Echo framework, OpenAPI spec, JWT + API key auth
- **PostgreSQL** — only required dependency. Row-level multi-tenancy via `tenant_id`
- **Database access** — sqlx (thin wrapper over database/sql, write SQL directly)
- **Background jobs** — River (PostgreSQL-backed job queue)
- **Frontend** — React 18 + TypeScript + Vite (standalone service, separate from backend)
- **Cloud providers** — abstracted behind a `Provider` interface. AWS first.
- **IaC** — Terraform/OpenTofu execution for infrastructure provisioning
- **K8s deploys** — Helm Go SDK

## Project Structure

```
api/                                 # Go backend (module: github.com/superset-studio/kapstan/api)
  cmd/kapstan/main.go                # entry point: `kapstan server`
  internal/
    controllers/                     # HTTP handlers + middleware + routing
    models/                          # shared domain model structs (imported by all layers)
    services/                        # business logic (no HTTP, no SQL)
      organization/                  # tenants, users, roles
      workspace/                     # workspaces
      connection/                    # cloud credentials
      infrastructure/                # resources, provisioning
      application/                   # apps, containers, deployments
      notification/                  # notification dispatch
    provider/                        # cloud provider interface + implementations
      provider.go                    # interface definition
      aws/                           # AWS implementation
    jobs/                            # background job runner (River)
    platform/                        # config, database, crypto, auth, logging
    repositories/                    # data access layer (sqlx queries)
  migrations/                        # SQL migration files (golang-migrate)
web/                                 # React frontend (standalone service)
deploy/                              # Dockerfiles (backend + frontend) + Helm chart
```

## Commands

```bash
make dev          # Run Go backend + Vite frontend with hot reload
make build        # Build Go backend binary
make test         # Run all tests
make lint         # Run linter
make migrate      # Run database migrations
make docker       # Build Docker image
make ci           # Full CI check (lint + test + build)
```

## Key Conventions

- **sqlx** for database access — write SQL directly in Go, scan into structs. No ORM, no code generation.
- **golang-migrate** for schema changes — add SQL files to `migrations/`
- **Echo** for HTTP routing — high-performance, expressive API, built-in middleware
- **slog** for logging — standard library structured logging
- Model structs live in `internal/models/` — plain Go structs shared across controllers, services, and repositories.
- Service packages (`internal/services/`) contain business logic only. No HTTP types, no SQL. They accept and return model structs.
- Repository packages (`internal/repositories/`) contain hand-written SQL queries using sqlx.
- Controllers in `internal/controllers/` translate HTTP ↔ service calls.
- Provider implementations in `internal/provider/` are the only place cloud SDKs are imported.
- All tenant-scoped queries filter by `tenant_id`. The middleware sets this from the JWT.
- Secrets at rest use AES-256-GCM. Encryption key from `KAPSTAN_ENCRYPTION_KEY` env var.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `KAPSTAN_ENCRYPTION_KEY` | Yes | — | 32-byte hex key for AES-256-GCM |
| `PORT` | No | `8080` | HTTP listen port |
| `JWT_SECRET` | No | derived from encryption key | JWT signing key |
| `REDIS_URL` | No | — | Optional Redis for caching |
| `LOG_LEVEL` | No | `info` | Logging level |

## Build Phases

The project is being built incrementally per `docs/SPEC.md` Section 8:
- Phase 0: Scaffolding
- Phase 1: Auth + Multi-Tenancy
- Phase 2: Connections
- Phase 3: Workspaces + Infrastructure
- Phase 4: Applications
- Phase 5: Deployment
- Phase 6: Cost + Monitoring
- Phase 7: Polish + GCP Provider
