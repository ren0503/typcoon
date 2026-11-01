# Tycoon Backend

Production-ready backend foundation for the yacht management platform. This repository currently
contains only the technical foundation: configuration, logging, database lifecycle, migrations,
query generation, AWS integration points, HTTP health endpoints, graceful shutdown, containerization,
and a minimal Terraform scaffold.

**No business logic, domain entities, business tables, or business APIs are implemented.**

## Prerequisites

- Go 1.26 or newer
- Docker with Docker Compose
- `make`
- Optional: Terraform 1.6 or newer (infrastructure validation only)

No AWS account or credentials are required for local development or tests.

## Quick Start

```bash
cp .env.example .env
make docker-up          # start PostgreSQL (and the backend container)
make migrate-up         # apply migrations from the host
make run                # run the backend from source
```

Then verify:

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

Expected responses:

```json
{"status":"ok"}
{"status":"ready"}
```

If the database is unreachable, the service exits rather than serving in a degraded state.

## Configuration

Configuration is read **only from process environment variables**. The application does not load
`.env`; the Makefile and Docker Compose load it for local convenience.

See `.env.example` for every variable and its default. Required settings are `DB_HOST`, `DB_NAME`,
`DB_USER`, and `DB_PASSWORD`. The full contract is in
`specs/001-backend-foundation/contracts/configuration.md`.

Production environments MUST set `DB_SSLMODE=require` or stricter.

## Commands

| Command | Purpose |
|---------|---------|
| `make fmt` | Format Go code |
| `make vet` | Run `go vet` |
| `make lint` | Run `golangci-lint` |
| `make test` | Run unit tests (no infrastructure) |
| `make test-integration` | Run integration tests with Testcontainers |
| `make run` | Run the backend locally |
| `make build` | Build `bin/server` and `bin/migrate` |
| `make sqlc` | Generate database access code |
| `make migrate-up` | Apply all migrations |
| `make migrate-down` | Roll back the latest migration |
| `make migrate-status` | Show migration status |
| `make docker-build` | Build the container image |
| `make docker-up` | Start local services |
| `make docker-down` | Stop local services |
| `make terraform-fmt` | Format Terraform files |
| `make terraform-validate` | Initialize and validate Terraform |

## Endpoints

Only technical endpoints exist:

- `GET /health` — liveness; does not check dependencies; returns `{"status":"ok"}`.
- `GET /ready` — readiness; pings PostgreSQL; returns `{"status":"ready"}` or HTTP 503
  `{"status":"not_ready"}`.

Neither endpoint exposes secrets, hosts, versions, or internal error details.

## Database

PostgreSQL is accessed through `pgxpool`. Migrations are managed by Goose and embedded in
`cmd/migrate`. Migrations are **never** run at application startup; run `make migrate-up` as a
separate step.

The only migration is a no-op technical migration that proves the pipeline. No business tables exist.

## Database Access Generation

Queries live in `db/queries`; generated code lives in `internal/database/sqlc` and must not be
hand-edited. Regenerate with `make sqlc`.

## AWS

The service loads AWS SDK v2 configuration using the default credential provider chain and supports
an optional `AWS_ENDPOINT_URL` override for LocalStack or tests. It creates no AWS service clients and
makes no AWS API calls at startup.

For local development without AWS, `AWS_EC2_METADATA_DISABLED=true` prevents instance-metadata
discovery.

Optional local emulation:

```bash
docker compose --profile localstack up -d localstack
```

## Terraform

The scaffold in `terraform/` uses local state and declares no resources by default. It provides
version constraints, provider configuration, variables, outputs, environment-specific values, and
naming/tagging conventions for future AWS resources.

```bash
cd terraform
terraform init
terraform validate
```

A remote state backend is intentionally deferred to future work.

## Project Layout

```text
cmd/                 Executables (server, migrate)
internal/            Private application packages
  app/               Bootstrap and lifecycle orchestration
  config/            Environment configuration and validation
  logging/           Structured logging
  database/          pgxpool lifecycle and migrations
  database/sqlc/     Generated database access
  awsconfig/         AWS SDK v2 configuration
  httpserver/        HTTP server, router, handlers, middleware
  health/            Database health monitor
  apperrors/         Shared error types
  testutil/          Integration test helpers
db/                  Migrations and queries
terraform/           Minimal AWS infrastructure scaffold
```

## Testing

- Unit tests: `make test`
- Integration tests: `make test-integration` (requires a working Docker daemon)
- No test requires AWS credentials.

## Scope

This foundation intentionally excludes business logic, domain models, business tables, business APIs,
authentication/authorization workflows, business event processing, metrics, tracing, and profiling.
See `specs/001-backend-foundation/spec.md` for the full specification.

## Troubleshooting

- **Startup fails naming `DB_*`**: confirm PostgreSQL is running and `.env` matches
  `docker-compose.yml`.
- **Readiness returns 503**: the database is unreachable; the service will exit shortly by design.
- **Integration tests fail**: confirm the Docker daemon is running.
- **`make sqlc` cannot download sqlc**: confirm access to the Go module proxy or pre-install the
  pinned version.
- **Terraform commands fail**: Terraform is optional for running the service; install it and confirm
  network access to download the AWS provider.
