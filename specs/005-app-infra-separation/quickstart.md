# Quickstart: Validate the Application/Infrastructure Separation

**Feature**: `005-app-infra-separation`
**Date**: 2026-11-12

This is an architectural refactor. There is no new schema, migration, endpoint, environment
variable, or runtime. Use the existing Makefile targets and commands.

## Prerequisites

- Go 1.26
- Docker daemon for integration tests (testcontainers PostgreSQL)
- Existing local configuration only if starting the API; migration tests use testcontainers

No AWS account, Redis, Firebase project, or external cloud service is required for the
unit/architecture test path.

## 1. Format, Vet, Unit Test, Build

```bash
make fmt
make vet
make test
make build
```

Expected: all pass. `make test` must remain infrastructure-free and still runs route
inventory, OpenAPI lint, auth unit/signature tests, and the architecture-boundary test.

## 2. Integration Tests

```bash
make test-integration
```

Expected: all existing integration suites pass against disposable PostgreSQL, including:

- migrations and schema/trigger behavior
- repository CRUD and constraint mapping
- ship import validation/confirmation rollback and audit
- booking idempotency
- guest document secrecy and boarding passes
- auth exchange, refresh rotation/reuse, logout-all
- sync/change-log behavior
- OpenAPI mounted-operation contract checks

If Docker is unavailable, compile-check integration packages without running them:

```bash
go test -tags=integration -run '^$' ./...
```

## 3. Verify the Layer Boundary

The architecture-boundary test must fail on a forbidden import. Temporarily add an import
such as:

```go
// inside an application package
import _ "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
```

Run:

```bash
go test ./internal/app -run TestLayerBoundaries -count=1
```

Expected: failure naming the offending application package and forbidden import. Remove the
import and rerun; expected: pass.

Repeat with a forbidden external import such as `net/http` or
`github.com/jackc/pgx/v5` if desired. Restore the file exactly after the mutation check.

## 4. Inspect the Package Graph Manually

List application imports to confirm no infrastructure/transport edges:

```bash
go list -f '{{.ImportPath}} -> {{join .Imports " "}}' ./internal/application/...
```

Confirm:

- no `internal/infrastructure/...`
- no `internal/transport/...`
- no `internal/app`
- no `net/http`, `pgx`, `sqlc`, `aws-sdk-go-v2`, `redis`, firebase, or chi

List composition imports to confirm only `internal/app` imports both layers:

```bash
go list -f '{{.ImportPath}} -> {{join .Imports " "}}' ./internal/app/...
```

## 5. Verify API and Migration Runtimes Separately

Migration runtime:

```bash
make migrate-status
make migrate-up
```

Expected: existing behavior unchanged. The migration runtime must not initialize Firebase,
Redis, KMS, HTTP server, or auth providers.

API runtime (requires existing local configuration):

```bash
make run
curl -fsS http://localhost:8080/health
curl -fsS http://localhost:8080/ready
```

Expected: `{"status":"ok"}` and `{"status":"ready"}` (or the existing not-ready response when
PostgreSQL is unavailable).

## 6. Verify Behavior Preservation

Run the existing suites without editing their assertions. Confirm:

```bash
go test ./internal/transport/http/... ./internal/application/... ./internal/app/...
go test -tags=integration ./internal/infrastructure/persistence/postgres/... \
  ./internal/application/... ./internal/transport/http/...
```

Expected observations:

- route inventory, `/api` prefix, and technical routes unchanged
- OpenAPI schemas and mounted operations unchanged
- auth token format, refresh rotation/reuse, and rate-limit behavior unchanged
- audit records are emitted inside the same transaction as business writes
- booking/import idempotency and conflict codes unchanged
- guest document ciphertext validation and secrecy unchanged
- PostgreSQL migrations, triggers, change log, and ownership mapping unchanged

## 7. Final Gate

```bash
go fmt ./...
go vet ./...
go test ./...
go build ./...
```

Also confirm:

- `README.md` and `docs/system-architecture.md` describe the new layout
- the ADR exists and records alternatives/deferred decisions
- no empty `worker`, `hub`, SQLite, S3, SNS, SQS, Redshift, EventBridge, certificate, or
  metrics packages were created
- `docs/schema.sql`, `db/migrations/`, and `openapi/` have no functional diff
