# Research: Initialize Backend Foundation

**Feature**: `001-backend-foundation` | **Date**: 2026-11-01

All Technical Context items are resolved below. No open questions remain.

## 1. Module Path and Go Version

- **Decision**: Use module path `github.com/ren0503/tycoon` and `go 1.26` in `go.mod`.
- **Rationale**: The module path matches the project name used throughout the documentation and
  avoids propagating the configured remote's spelling error into every import. The local toolchain
  is Go 1.26.8, so declaring 1.26 avoids relying on a toolchain that is not available in the
  development environment.
- **Alternatives considered**: A short local path such as `tycoon/server` (rejected because it is not
  a resolvable import path for future consumers); an older `go` directive such as 1.24 (rejected
  because it does not reflect the verified toolchain and may force unnecessary workarounds).

## 2. Repository Layout

- **Decision**: Use `cmd/server`, `cmd/migrate`, and `internal/*` packages. No `pkg/` directory and
  no empty feature-module directories.
- **Rationale**: `cmd/` separates executables from reusable code; `internal/` enforces the
  modular-monolith boundary by preventing external imports. Empty feature directories would violate
  the spec's prohibition on speculative abstractions.
- **Alternatives considered**: A flat root package (rejected: poor separation and harder testing); a
  `pkg/` layout (rejected: no public API exists); pre-created `modules/` tree (rejected: speculative).

## 3. Configuration Strategy

- **Decision**: Use Viper with explicit `BindEnv` mappings, `mapstructure` with
  `StringToTimeDurationHookFunc`, and `validator/v10` struct validation. Read only process
  environment variables; do not load `.env` inside the application.
- **Rationale**: Explicit bindings make the environment contract reviewable and avoid implicit
  prefix collisions. The duration decode hook lets timeouts be configured as human-readable values
  such as `5s`. Validator provides declarative required/range/enum checks. The clarification
  decision requires `.env` loading to be a developer-tooling concern (Makefile or Compose), not an
  application behavior.
- **Alternatives considered**: `os.Getenv` helpers (rejected: repetitive and harder to validate as a
  unit); automatic `godotenv` loading (rejected by clarification); a config file (rejected: the spec
  requires environment-based configuration).

### Environment Contract

| Variable | Required | Default | Validation |
|----------|----------|---------|------------|
| `APP_NAME` | No | `tycoon-backend` | non-empty |
| `APP_ENV` | No | `development` | one of `development`, `test`, `staging`, `production` |
| `LOG_LEVEL` | No | `info` | one of `debug`, `info`, `warn`, `error` |
| `HTTP_HOST` | No | `0.0.0.0` | valid host |
| `HTTP_PORT` | No | `8080` | 1–65535 |
| `HTTP_READ_TIMEOUT` | No | `5s` | positive duration |
| `HTTP_WRITE_TIMEOUT` | No | `10s` | positive duration |
| `HTTP_IDLE_TIMEOUT` | No | `60s` | positive duration |
| `HTTP_SHUTDOWN_TIMEOUT` | No | `10s` | positive duration |
| `DB_HOST` | Yes | — | non-empty |
| `DB_PORT` | No | `5432` | 1–65535 |
| `DB_NAME` | Yes | — | non-empty |
| `DB_USER` | Yes | — | non-empty |
| `DB_PASSWORD` | Yes | — | non-empty; never logged |
| `DB_SSLMODE` | No | `disable` | `disable`, `require`, `verify-ca`, `verify-full` |
| `DB_MAX_CONNS` | No | `10` | > 0 |
| `DB_MIN_CONNS` | No | `0` | >= 0 and <= max conns |
| `DB_MAX_CONN_LIFETIME` | No | `1h` | positive duration |
| `DB_MAX_CONN_IDLE_TIME` | No | `30m` | positive duration |
| `DB_CONNECT_TIMEOUT` | No | `5s` | positive duration |
| `AWS_REGION` | No | `us-east-1` | non-empty |
| `AWS_ENDPOINT_URL` | No | empty | valid URL when set |
| `CORS_ALLOWED_ORIGINS` | No | empty | comma-separated absolute origins; empty denies all |

- **Rationale for required set**: Database connectivity settings that identify the target
  (`DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`) are required so startup cannot silently point at
  the wrong database. Non-sensitive values receive safe defaults for local convenience.
- **Production note**: `DB_SSLMODE` defaults to `disable` only to keep the local Compose database
  usable; production environments MUST set `require` or stricter. This is documented in
  `contracts/configuration.md`.

## 4. PostgreSQL and Connection Pool

- **Decision**: Use `pgxpool` directly with an explicit `pgxpool.Config` built from application
  configuration. Ping on startup with the configured connect timeout. Close the pool during
  shutdown. Do not add a repository abstraction.
- **Rationale**: `pgxpool` is the selected driver's native pool and satisfies lifecycle, health, and
  performance needs. The spec explicitly says not to introduce a repository abstraction unless the
  database access approach requires it; sqlc-generated queries are the access layer.
- **Alternatives considered**: `database/sql` with `pgx` stdlib adapter (rejected: unnecessary
  indirection); an ORM (rejected: not in the agreed stack); a repository interface layer (rejected:
  speculative for a foundation with no business queries).

## 5. Migrations

- **Decision**: Use Goose v3 as a library through `cmd/migrate`, with migrations embedded via
  `embed.FS`. Support `up`, `down`, `status`, and `version`. Provide one no-op technical migration
  (`00001_init.sql`) so the pipeline is exercised without creating a business table. Never run
  migrations automatically at application startup.
- **Rationale**: Using the library avoids requiring a separately installed CLI, keeps commands
  reproducible, and satisfies the constitution's requirement that migrations run as a separate
  deployment step. A no-op migration proves the toolchain without inventing schema.
- **Alternatives considered**: Goose CLI only (rejected: adds an undocumented local tool
  requirement); an empty migration directory (rejected: does not demonstrate a working pipeline);
  creating a technical metadata table (rejected: unnecessary schema).

## 6. sqlc

- **Decision**: Use sqlc with `sql_package: pgx/v5`, queries in `db/queries`, and generated output in
  `internal/database/sqlc`. Include one example query `Ping :one SELECT 1;`. Pin the sqlc version in
  the Makefile via `go run github.com/sqlc-dev/sqlc/cmd/sqlc@<pinned> generate`.
- **Rationale**: Generated code stays isolated from handwritten code, the example query verifies the
  toolchain, and `go run` makes the version reproducible without a global install.
- **Alternatives considered**: A globally installed sqlc (rejected: version drift); generating into
  `db/` (rejected: mixes generated and source artifacts); skipping the example query (rejected by
  the spec).

## 7. AWS SDK v2 Configuration

- **Decision**: Load `aws.Config` once with `config.LoadDefaultConfig` using `config.WithRegion` and
  an optional `aws.Config.BaseEndpoint` override. Use the SDK default credential chain. Do not create
  S3, SQS, SNS, Secrets Manager, or CloudWatch clients during startup.
- **Rationale**: `LoadDefaultConfig` supports IAM roles and temporary credentials and does not fetch
  credentials or make service API calls at load time. Deferring client creation avoids unnecessary
  initialization and satisfies the "no real AWS API calls at startup" requirement. The endpoint
  override supports LocalStack without making it mandatory.
- **Alternatives considered**: Eagerly constructing all five service clients (rejected: unnecessary
  and contradicts the spec); a custom credential provider (rejected: breaks the default chain);
  skipping AWS initialization entirely (rejected: the spec requires the integration point).
- **Local development note**: `AWS_EC2_METADATA_DISABLED=true` is documented for local use so the SDK
  never attempts instance-metadata discovery when credentials are absent.

## 8. HTTP Server, Middleware, and CORS

- **Decision**: Use Chi v5. Middleware order: request ID, real IP, structured request logger, panic
  recoverer, CORS, routes. Use `chi/cors` with `AllowedOrigins` from configuration; an empty
  allowlist denies all cross-origin access. Configure read, write, idle, and shutdown timeouts.
- **Rationale**: Chi is the agreed router and provides request ID and real-IP middleware. A custom
  slog logger and recoverer keep logs structured and avoid leaking panic details to clients.
  `chi/cors` is small, maintained, and safer than hand-rolled CORS logic.
- **Alternatives considered**: Hand-rolled CORS (rejected: more security-sensitive code to maintain);
  Chi's default `Recoverer` (rejected: logs through the standard logger, not slog); permissive
  wildcard CORS (rejected by clarification and security principle).

## 9. Health, Readiness, and Database Failure Policy

- **Decision**: `GET /health` is a dependency-free liveness check returning `{"status":"ok"}`.
  `GET /ready` pings the database with a bounded timeout and returns `{"status":"ready"}` or HTTP 503
  `{"status":"not_ready"}`. A background database monitor pings periodically; on failure it signals
  the application to shut down gracefully and exit, matching the clarification decision.
- **Rationale**: Liveness must not depend on the database, while readiness must reflect whether the
  instance can serve. The monitor enforces the "exit when the database is unavailable" decision even
  when no readiness request arrives.
- **Alternatives considered**: Readiness only, without a monitor (rejected: the process could remain
  alive indefinitely in a degraded state); liveness that pings the database (rejected: would cause
  restarts for dependency outages); exposing dependency details in the response (rejected: leaks
  infrastructure information).

## 10. Lifecycle and Graceful Shutdown

- **Decision**: Use `signal.NotifyContext` for `SIGINT` and `SIGTERM`. On cancellation, call
  `http.Server.Shutdown` with `HTTP_SHUTDOWN_TIMEOUT`, then close the database pool. Exit non-zero on
  startup failure or unexpected shutdown errors; exit zero on a clean signal-triggered shutdown.
- **Rationale**: Context-based signal handling is idiomatic and testable. Bounded shutdown prevents
  hung processes. Closing the pool after the HTTP server stops prevents in-flight requests from
  losing their database connection prematurely.
- **Alternatives considered**: `os/signal` channel without context (rejected: more manual wiring);
  closing the pool before the HTTP server (rejected: breaks in-flight requests); no timeout
  (rejected: violates reliability requirements).

## 11. Logging and Errors

- **Decision**: Initialize `slog` with a JSON handler in non-development environments and a text
  handler in development. Attach `request_id` and other request fields through context-aware
  helpers. Add `internal/apperrors` with a stable error code, a safe public message, and an optional
  wrapped cause that is logged but never returned to clients.
- **Rationale**: Structured logs are required by the constitution and support later aggregation.
  A shared error type prevents handlers from leaking internal details and gives future feature
  modules a consistent error contract.
- **Alternatives considered**: Third-party logging libraries (rejected: `log/slog` is standard and
  required by the spec); returning raw `err.Error()` to clients (rejected: may leak sensitive data).

## 12. Testing Strategy

- **Decision**: Unit tests use only the standard library and fakes; integration tests carry
  `//go:build integration` and use `testcontainers-go` with the PostgreSQL module. Provide a
  `testutil` helper that starts one container per test package and applies migrations. No test
  requires AWS credentials or network access.
- **Rationale**: The build tag cleanly separates fast unit runs from container-dependent tests.
  Testcontainers gives real PostgreSQL behavior for pool and migration tests without a shared
  developer database.
- **Alternatives considered**: `sqlmock` (rejected: does not exercise pgx or migrations); a shared
  local database (rejected: non-reproducible and stateful); Docker Compose for tests (rejected:
  Testcontainers manages lifecycle more reliably).

## 13. Containers and Local Development

- **Decision**: Multi-stage Dockerfile: `golang:1.26-alpine` builder producing a static binary, final
  `alpine:3.20` image running as a non-root user with `wget` available for container health checks.
  Docker Compose provides PostgreSQL 16 with `pg_isready` health check, the backend service, and an
  optional LocalStack profile. `.env.example` documents every variable.
- **Rationale**: A static binary avoids libc coupling. A non-root final image is a baseline security
  control. Alpine provides a small, well-known health-check tool. LocalStack remains opt-in so the
  backend runs without AWS.
- **Alternatives considered**: `scratch`/distroless (rejected: no shell or health-check utility);
  single-stage build (rejected: large image and build tooling in production); requiring LocalStack
  (rejected by the spec).

## 14. Terraform Scaffold

- **Decision**: Root module with `versions.tf`, `providers.tf`, `variables.tf`, `outputs.tf`, and
  `main.tf` containing only locals for naming and tagging. Environment-specific values live in
  `environments/dev` and `environments/prod` `.tfvars` files. Use local state only. No resources are
  declared.
- **Rationale**: The spec requires a valid minimal structure that accommodates future resources
  without provisioning them. Local state matches the clarification and avoids bootstrapping remote
  storage. Naming and tagging locals establish conventions early.
- **Alternatives considered**: Declaring placeholder resources (rejected: creates cost and violates
  the spec); remote state backend (rejected by clarification); a `modules/` tree (rejected:
  speculative abstraction).
- **Verification note**: `terraform validate` requires `terraform init` to download the AWS provider;
  the README documents Terraform as an optional prerequisite for infrastructure commands.

## 15. Developer Tooling

- **Decision**: Provide a Makefile with targets for `fmt`, `vet`, `lint`, `test`, `test-integration`,
  `sqlc`, `migrate-up`, `migrate-down`, `migrate-status`, `docker-build`, `docker-up`,
  `docker-down`, `terraform-fmt`, and `terraform-validate`. Pin tool versions in the Makefile.
  Provide a `.golangci.yml` with a minimal linter set.
- **Rationale**: Reproducible commands satisfy the developer-experience requirements and avoid
  undocumented global tool installations. Pinning avoids "works on my machine" drift.
- **Alternatives considered**: Shell scripts only (rejected: less discoverable); requiring global
  installs of sqlc, Goose, and Terraform (rejected: poor reproducibility); a full CI pipeline
  (deferred: the spec asks for local developer tooling, not production CI).
