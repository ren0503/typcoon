# Canonical entry points (identical locally and in CI):
#   make test             -> unit + OpenAPI spec lint, no infrastructure required
#   make test-integration -> build-tag integration suites via Testcontainers (Docker daemon required)
# CI should run both targets; if Docker is unavailable, only `make test` can run and the
# integration suites are compile-checked with:
#   go test -tags=integration -run '^$' ./...

SHELL := /bin/sh

GO ?= go
GOLANGCI_LINT ?= golangci-lint
SQLC_VERSION ?= v1.30.0

BINARY := bin/server
MIGRATE_BINARY := bin/migrate

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: fmt vet lint test test-integration run build sqlc \
	migrate-up migrate-down migrate-status \
	docker-build docker-up docker-down \
	terraform-fmt terraform-validate

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	$(GOLANGCI_LINT) run ./...

test:
	$(GO) test ./...

test-integration:
	$(GO) test -tags=integration ./...

run:
	$(GO) run ./cmd/server

build:
	$(GO) build -o $(BINARY) ./cmd/server
	$(GO) build -o $(MIGRATE_BINARY) ./cmd/migrate

sqlc:
	$(GO) run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

migrate-up:
	$(GO) run ./cmd/migrate up

migrate-down:
	$(GO) run ./cmd/migrate down

migrate-status:
	$(GO) run ./cmd/migrate status

docker-build:
	docker build -t tycoon-backend:local .

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

terraform-fmt:
	terraform -chdir=terraform fmt -recursive

terraform-validate:
	terraform -chdir=terraform init -backend=false
	terraform -chdir=terraform validate

# Local/CI parity: `make test` must stay infrastructure-free. `make test-integration`
# starts disposable PostgreSQL containers per test and is the only target that needs Docker.
# The OpenAPI contract suite lives under internal/transport/http and validates the spec in both modes
# (spec lint in `make test`, mounted-operation and request/response round-trip checks in
# `make test-integration`).
