# Kapstan

Open-source cloud management platform. Deploy applications to Kubernetes and provision cloud infrastructure through a web UI and REST API.

## Quick Start

**Prerequisites:** Go 1.25+, Node 20+, PostgreSQL 15+

```bash
# Clone and run
git clone https://github.com/superset-studio/kapstan.git
cd kapstan

# Start backend + frontend in development mode
make dev
```

The API server runs on `http://localhost:8080` and the frontend on `http://localhost:5173`.

## Build

```bash
# Build the Go backend binary
make build

# Run directly
DATABASE_URL=postgres://localhost:5432/kapstan \
KAPSTAN_ENCRYPTION_KEY=$(openssl rand -hex 32) \
./bin/kapstan server
```

## Commands

```bash
make build        # Build Go backend binary to bin/kapstan
make dev          # Run Go backend (air) + Vite frontend with hot reload
make test         # Run Go tests
make lint         # Run golangci-lint
make migrate      # Run database migrations
make docker       # Build backend Docker image
make docker-web   # Build frontend Docker image
make ci           # lint + test + build
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `KAPSTAN_ENCRYPTION_KEY` | Yes | — | 32-byte hex key for AES-256-GCM |
| `PORT` | No | `8080` | HTTP listen port |
| `LOG_LEVEL` | No | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | No | `json` | `json` or `text` |

## License

MIT
