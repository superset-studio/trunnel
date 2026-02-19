.DEFAULT_GOAL := help
.PHONY: build dev test test-integration lint migrate docker docker-web ci clean fmt install help

help:  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

# Go binaries
build:  ## Build server and CLI binaries
	cd api && go build -o ../bin/kapstan ./cmd/kapstan
	cd cli && go build -o ../bin/kapstanctl ./cmd/kapstanctl

test:  ## Run all Go tests
	cd api && go test ./...

test-integration:  ## Run integration tests (requires TEST_DATABASE_URL)
	cd api && go test -tags integration -v -count=1 ./internal/integration/

lint:  ## Run Go linter
	cd api && golangci-lint run

migrate:  ## Run database migrations
	cd api && go run ./cmd/kapstan migrate --path migrations

fmt:  ## Format Go code
	cd api && go fmt ./...
	cd cli && go fmt ./...

clean:  ## Remove build artifacts
	rm -rf bin/ web/dist/

install: build  ## Build and install to $$GOPATH/bin
	cp bin/kapstan bin/kapstanctl $(shell go env GOPATH)/bin/

# Development: run Go backend with air + Vite dev server
dev:  ## Run backend + frontend with hot reload
	@if ! command -v air > /dev/null 2>&1; then echo "Install air: go install github.com/air-verse/air@latest"; exit 1; fi
	@if [ ! -d web/node_modules ]; then echo "Installing frontend deps..."; cd web && npm install; fi
	@trap 'kill 0' EXIT; \
	cd api && air -- server & \
	cd web && npm run dev & \
	wait

# Docker
docker:  ## Build backend Docker image
	docker build -f deploy/Dockerfile -t kapstan-api .

docker-web:  ## Build frontend Docker image
	docker build -f deploy/Dockerfile.web -t kapstan-web .

# CI
ci: lint test build  ## Full CI check (lint + test + build)
