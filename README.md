# Tycoon Backend

Production-ready Go backend for the yacht management platform. It includes the technical foundation
(configuration, logging, database lifecycle, migrations, sqlc query generation, AWS integration
points, HTTP health endpoints, graceful shutdown, containerization, and a minimal Terraform
scaffold) and the `002-business-modules` increment: the 41-table business schema and REST modules
`tenant`, `ship`, `voyage`, `staff`, `partner`, `booking`, and `guest` under `/v1`. The `sync`, `ai`,
`feedback`, `auth`, and `audit` modules expose services only; they register no HTTP routes in this
phase. See `specs/002-business-modules/` for the specification, contracts, and ownership map.

## Prerequisites

- Go 1.26 or newer
- Docker with Docker Compose
- `make`
- Optional: Terraform 1.6 or newer (infrastructure validation only)

No AWS account or credentials are required for local development or tests.

## Quick Start

```bash
cp .env.example .env
docker compose up -d postgres   # PostgreSQL only; the API runs from source
make sqlc                       # regenerate sqlc models/queries (already committed)
make migrate-up                 # apply the foundation + 41-table business schema
make run                        # start the API on :8080
```

Then verify:

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

Business endpoints live under `/v1` and require an authenticated principal. Local development uses
the explicit allowlist provider from `.env`:

```bash
AUTH_MODE=dev
AUTH_DEV_PRINCIPALS=system:dev-1:00000000-0000-0000-0000-000000000000
# then send: X-Dev-Principal: system:dev-1:00000000-0000-0000-0000-000000000000
curl -i -H "X-Dev-Principal: system:dev-1:00000000-0000-0000-0000-000000000000" \
  http://localhost:8080/v1/tenants
```

`AUTH_MODE=dev` is rejected when `APP_ENV=production`. `AUTH_MODE=jwt` is reserved for the later auth
feature and currently denies all requests. No AWS account or credentials are required locally
(`AWS_EC2_METADATA_DISABLED=true` is set in `.env.example`).

Stop the database with `docker compose down` (add `-v` to remove the volume). If you prefer the full
containerized stack, use `make docker-up` instead of `docker compose up -d postgres && make run`.

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
`DB_USER`, and `DB_PASSWORD`; authentication requires `AUTH_MODE` (`dev` or `jwt`) and, in dev mode,
a non-empty `AUTH_DEV_PRINCIPALS` allowlist. The foundation contract is in
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

Technical endpoints:

- `GET /health` — liveness; does not check dependencies; returns `{"status":"ok"}`.
- `GET /ready` — readiness; pings PostgreSQL; returns `{"status":"ready"}` or HTTP 503
  `{"status":"not_ready"}`.

Business endpoints under `/v1` (all behind default-deny authentication): tenants, ships/decks/areas/
rooms, ports/voyages/port-calls, staff/leaves/credentials/shift templates/coverage/labor rules/
autonomy policies, partners/contracts/staff/services/slots, bookings/service-bookings, and
guests/boarding-passes. Cursor pagination uses `limit` and `cursor`; the response envelope is
`{"data": [...], "meta": {"next_cursor": ...}}`. Mutating booking and service-booking requests accept
an `Idempotency-Key` header. Internal tables (`change_log`, `ship_seq`, `refresh_tokens`, `audit_log`,
`ops_events`, `snapshots`, `job_runs`) deliberately expose no CRUD routes.

Neither technical endpoint exposes secrets, hosts, versions, or internal error details. Guest
responses never include document or emergency-contact ciphertext.

## Database

PostgreSQL is accessed through `pgxpool`. Migrations are managed by Goose and embedded in
`cmd/migrate`. Migrations are **never** run at application startup; run `make migrate-up` as a
separate step.

`00001` is a no-op technical migration that proves the pipeline. `00002_business_schema.sql` adds the
41-table business schema from `docs/schema.sql`, including triggers and exclusion constraints. Keep
`db/schema.sql`, the migration, and `docs/schema.sql` in sync.

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
  app/               Bootstrap, lifecycle, module composition
  tenant/            Tenant module (service, repo, handlers)
  ship/              Ships/decks/areas/rooms module
  voyage/            Ports/voyages/port-calls module
  staff/             Staff/leaves/credentials/shifts/coverage/labor/autonomy module
  partner/           Partners/contracts/staff/services/slots module
  booking/           Bookings and service bookings module
  guest/             Guests and boarding passes module
  sync/              Device/credential/snapshot/job lifecycle (no HTTP routes)
  ai/                AI parameter metadata (no HTTP routes)
  feedback/          Feedback responses and insights (no HTTP routes)
  auth/              Principal provider and account reference service
  audit/             Audit appender (same-transaction emission)
  ownership/         Table-to-module ownership map and CI guard
  httpx/             Shared JSON/error/pagination handler helpers
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

The `002-business-modules` increment implements the business schema and REST modules listed above.
It intentionally excludes token issuance/JWT verification workflows (dev allowlist only), sync
transport, AI signing/execution workflows, metrics, tracing, and profiling. See
`specs/002-business-modules/spec.md` and `specs/001-backend-foundation/spec.md` for details.

## Troubleshooting

- **Startup fails naming `DB_*`**: confirm PostgreSQL is running and `.env` matches
  `docker-compose.yml`.
- **Readiness returns 503**: the database is unreachable; the service will exit shortly by design.
- **Integration tests fail**: confirm the Docker daemon is running.
- **`make sqlc` cannot download sqlc**: confirm access to the Go module proxy or pre-install the
  pinned version.
- **Terraform commands fail**: Terraform is optional for running the service; install it and confirm
  network access to download the AWS provider.
