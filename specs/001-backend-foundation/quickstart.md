# Quickstart: Initialize Backend Foundation

**Feature**: `001-backend-foundation` | **Date**: 2026-11-01

This is a validation and run guide. It proves the foundation works end to end. It does not contain
implementation code, migrations, or test suites.

## Prerequisites

- Go 1.26 or newer.
- Docker with Docker Compose.
- `make`.
- Optional: Terraform 1.6 or newer for infrastructure validation.
- Optional: LocalStack for AWS emulation.

No AWS account, credentials, or AWS network access is required.

## 1. Configure the Environment

```bash
cp .env.example .env
```

The application reads process environment variables only. The Makefile and Docker Compose load
`.env` for local convenience. Edit `.env` if your local database settings differ.

## 2. Start Local Dependencies

```bash
make docker-up
```

Expected outcome: PostgreSQL 16 starts and reports healthy. The backend container may also start; if
so, stop it while iterating locally with `make docker-down` or run only the database service.

## 3. Apply Migrations

```bash
make migrate-up
make migrate-status
```

Expected outcome: Goose applies the initial technical migration and records its version. Re-running
`make migrate-up` is harmless.

## 4. Generate Database Access Code

```bash
make sqlc
```

Expected outcome: generated code appears under `internal/database/sqlc/`, separate from handwritten
code. Re-running the command produces no unexpected diff.

## 5. Run the Service Locally

```bash
make run
```

Expected outcome: the service starts, logs a structured startup message, and begins listening on the
configured address. Startup succeeds without AWS credentials.

## 6. Verify Technical Endpoints

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

Expected outcome:

- `/health` returns HTTP 200 with `{"status":"ok"}`.
- `/ready` returns HTTP 200 with `{"status":"ready"}` while the database is reachable.
- Neither response contains secrets, host names, versions, or internal error details.

## 7. Verify Graceful Shutdown

```bash
# Start the service in one terminal, then send SIGTERM to the process.
kill -TERM <pid>
```

Expected outcome: the service stops accepting new requests, finishes in-flight requests within the
configured shutdown timeout, closes the database pool, logs the shutdown, and exits.

## 8. Verify Database Failure Behavior

```bash
# With the service running, stop PostgreSQL.
docker compose stop postgres
```

Expected outcome: the service detects the unavailable database, stops serving, shuts down cleanly,
and exits non-zero. It does not continue serving in a degraded state.

## 9. Run Unit Tests

```bash
make test
```

Expected outcome: all unit tests pass without requiring Docker, PostgreSQL, or AWS.

## 10. Run Integration Tests

```bash
make test-integration
```

Expected outcome: integration tests start a PostgreSQL container through Testcontainers, apply
migrations, exercise the pool and readiness behavior, and pass. No AWS account is required.

## 11. Validate Containers

```bash
make docker-build
```

Expected outcome: a multi-stage build produces a non-root runtime image. The image starts with valid
configuration.

## 12. Validate Terraform Scaffold

```bash
cd terraform
terraform init
terraform validate
cd ..
```

Expected outcome: `terraform init` downloads the pinned AWS provider and `terraform validate`
succeeds. No resources are planned or created by default. A remote state backend is intentionally
absent; it is documented as future work.

## Acceptance Criteria Mapping

| Spec Criterion | Validation Step |
|----------------|-----------------|
| AC-1 Local run from documented commands | Steps 1–6 |
| AC-2 Starts with valid configuration | Step 5 |
| AC-3 Connects to local PostgreSQL | Steps 2–6 |
| AC-4 Migrations execute | Step 3 |
| AC-5 sqlc generation works | Step 4 |
| AC-6 Health and readiness respond correctly | Step 6 |
| AC-7 Graceful shutdown | Step 7 |
| AC-8 AWS SDK config without credentials or calls | Steps 5–6 (no AWS setup required) |
| AC-9 Terraform minimal structure valid | Step 12 |
| AC-10 Unit tests pass | Step 9 |
| AC-11 Integration tests run locally | Step 10 |
| AC-12 No business logic, schema, module, or API | Review of `internal/`, `db/`, and routes; only `/health` and `/ready` exist |

## Troubleshooting

- **Startup fails naming `DB_*`**: confirm PostgreSQL is running and `.env` values match
  `docker-compose.yml`.
- **Readiness returns 503**: confirm the database is reachable; the service will exit shortly by
  design.
- **sqlc download fails**: confirm network access to the Go module proxy or pre-install the pinned
  sqlc version.
- **Terraform provider download fails**: confirm network access; Terraform is optional for the
  application runtime.
