---
description: "Task list for Initialize Backend Foundation"
---

# Tasks: Initialize Backend Foundation

**Input**: Design documents from `/specs/001-backend-foundation/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included because FR-022, FR-023, SC-007, and SC-010 explicitly require unit and
integration tests.

**Organization**: Tasks are grouped by user story. The foundation is a shared technical base, so
Phase 2 contains the blocking scaffolding, and each story adds an independently testable increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)
- Every task includes an exact file path or command target

## Path Conventions

Single Go module at repository root: `cmd/`, `internal/`, `db/`, `terraform/`, plus root-level
build and documentation files. Generated sqlc code lives in `internal/database/sqlc/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize the Go module, repository files, local development assets, and base tooling.

- [X] T001 Create the Go module and directory layout at repository root: `go.mod`, `cmd/server/`,
  `cmd/migrate/`, `internal/app/`, `internal/config/`, `internal/logging/`, `internal/database/`,
  `internal/database/sqlc/`, `internal/awsconfig/`, `internal/httpserver/`,
  `internal/httpserver/handlers/`, `internal/httpserver/middleware/`, `internal/health/`,
  `internal/apperrors/`, `internal/testutil/`, `db/migrations/`, `db/queries/`, `terraform/`
- [X] T002 Add pinned dependencies to `go.mod` and `go.sum`: `github.com/go-chi/chi/v5`,
  `github.com/go-chi/cors`, `github.com/jackc/pgx/v5`, `github.com/pressly/goose/v3`,
  `github.com/spf13/viper`, `github.com/go-playground/validator/v10`,
  `github.com/aws/aws-sdk-go-v2/config`, and `github.com/testcontainers/testcontainers-go`
- [X] T003 [P] Add `.gitignore` (ignore `.env`, `bin/`, coverage files, Terraform state) and
  `.dockerignore` at repository root
- [X] T004 [P] Add `.env.example` at repository root matching every variable and default in
  `specs/001-backend-foundation/contracts/configuration.md`
- [X] T005 [P] Add `.golangci.yml` at repository root with a minimal linter set (`govet`, `errcheck`,
  `staticcheck`, `ineffassign`, `unused`, `gofmt`)
- [X] T006 Add `Makefile` at repository root with base targets: `fmt`, `vet`, `lint`, `test`,
  `test-integration`, `run`, `docker-build`, `docker-up`, `docker-down`
- [X] T007 [P] Add `Dockerfile` (multi-stage `golang:1.26-alpine` builder, `alpine:3.20` non-root
  runtime) and `docker-compose.yml` (PostgreSQL 16 with `pg_isready` health check, backend service,
  optional LocalStack profile)
- [X] T008 [P] Add `README.md` at repository root with prerequisites, setup outline, and placeholders
  for local run, testing, migrations, and infrastructure commands

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core packages that every user story depends on. No user story work begins until this
phase is complete.

**⚠️ CRITICAL**: This phase blocks all user stories.

- [X] T009 Implement configuration structs, explicit Viper environment bindings, and non-sensitive
  defaults in `internal/config/config.go` per `contracts/configuration.md`; do not add validation yet
- [X] T010 [P] Implement `slog` initialization (JSON for non-development, text for development) and
  context-aware request fields in `internal/logging/logging.go`
- [X] T011 [P] Implement shared error types with stable code, safe message, internal cause, and HTTP
  status in `internal/apperrors/errors.go`; the cause MUST never be serialized
- [X] T012 Implement `pgxpool` construction, startup ping bounded by `DB_CONNECT_TIMEOUT`, and
  idempotent close in `internal/database/postgres.go`
- [X] T013 Implement `http.Server` construction with host, port, read, write, and idle timeouts in
  `internal/httpserver/server.go`
- [X] T014 Implement the Chi router in `internal/httpserver/router.go` with request ID middleware (do
  not trust client-supplied forwarding headers), a placeholder middleware chain, and a route
  registration hook; do not add business routes
- [X] T015 Implement bootstrap orchestration in `internal/app/app.go`: load config, initialize
  logging, construct the pool, construct the HTTP server, start serving, and stop components in
  order. Define minimal `Server` and `Database` lifecycle interfaces in `internal/app/app.go` so the
  orchestration can be tested with fakes; keep shutdown minimal for now
- [X] T016 Implement `cmd/server/main.go`: call the bootstrap, log startup failures with safe
  messages, and exit non-zero on failure
- [X] T017 [P] Implement a Testcontainers PostgreSQL helper in `internal/testutil/postgres.go`
  behind `//go:build integration`, including migration application and cleanup

**Checkpoint**: Foundation scaffolding is ready; user story implementation can begin.

---

## Phase 3: User Story 1 - Run the Backend Locally (Priority: P1) 🎯 MVP

**Goal**: A new developer can start local PostgreSQL, apply migrations, run the service, and receive
successful responses from `/health` and `/ready` without an AWS account.

**Independent Test**: From a clean checkout, follow the README to start dependencies, run
`make migrate-up`, run `make run`, and verify `/health` returns 200 `{"status":"ok"}` and `/ready`
returns 200 `{"status":"ready"}`.

### Tests for User Story 1

- [X] T018 [P] [US1] Add handler unit tests for `GET /health` and `GET /ready` in
  `internal/httpserver/handlers/health_test.go`, covering ready, not-ready, and no-leak assertions
- [X] T019 [P] [US1] Add an integration test for pool initialization and migration application in
  `internal/database/postgres_test.go` behind `//go:build integration`

### Implementation for User Story 1

- [X] T020 [US1] Implement the Goose migration runner with `embed.FS` and `up`, `down`, `status`, and
  `version` operations in `internal/database/migrate.go`
- [X] T021 [US1] Implement `cmd/migrate/main.go` that reads configuration, opens the pool, and
  dispatches the requested migration operation; migrations MUST NOT run at application startup
- [X] T022 [P] [US1] Add the initial no-op technical migration in `db/migrations/00001_init.sql` with
  Goose Up/Down annotations and no business table
- [X] T023 [US1] Implement `GET /health` and `GET /ready` handlers in
  `internal/httpserver/handlers/health.go`: liveness returns `{"status":"ok"}`; readiness pings the
  database with a bounded timeout and returns `{"status":"ready"}` or HTTP 503
  `{"status":"not_ready"}`; neither response exposes dependencies, versions, hosts, or errors
- [X] T024 [US1] Register only the technical routes in `internal/httpserver/router.go` and call the
  registration hook from the bootstrap
- [X] T025 [US1] Document the local run flow (environment copy, `make docker-up`, `make migrate-up`,
  `make run`, endpoint checks) in `README.md`

**Checkpoint**: User Story 1 is fully functional and is the MVP.

---

## Phase 4: User Story 2 - Fail Fast on Invalid Configuration (Priority: P2)

**Goal**: Missing or invalid required configuration prevents startup with a clear, non-sensitive
error; valid configuration applies defaults and starts successfully.

**Independent Test**: Start the service with each required variable missing or malformed and confirm
startup fails naming the setting; start with valid values and confirm startup succeeds.

### Tests for User Story 2

- [X] T026 [P] [US2] Add unit tests in `internal/config/config_test.go` for defaults, required
  variables (`DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`), enum validation (`APP_ENV`,
  `LOG_LEVEL`, `DB_SSLMODE`), port ranges, duration parsing, `AWS_ENDPOINT_URL` URL validation, and
  `DB_MIN_CONNS <= DB_MAX_CONNS`

### Implementation for User Story 2

- [X] T027 [US2] Add `validator/v10` tags and a validation entry point to
  `internal/config/config.go`, including required fields, enum checks, port ranges, positive
  durations, URL validation, and `DB_MIN_CONNS <= DB_MAX_CONNS`
- [X] T028 [US2] Wire fail-fast configuration validation into `internal/app/app.go` and
  `cmd/server/main.go` so startup exits non-zero before serving and error messages never include
  `DB_PASSWORD` or other secret values

**Checkpoint**: Invalid configuration fails fast; valid configuration starts with defaults.

---

## Phase 5: User Story 3 - Observe and Shut Down Gracefully (Priority: P2)

**Goal**: The service emits structured request logs with correlation identifiers, recovers from
panics, enforces a CORS allowlist, detects database loss, and shuts down within the configured
window.

**Independent Test**: Send `SIGTERM` while a request is in flight and confirm the service stops
accepting new requests, drains, closes the pool, and exits within `HTTP_SHUTDOWN_TIMEOUT`; stop
PostgreSQL and confirm the service exits rather than serving in a degraded state.

### Tests for User Story 3

- [X] T029 [P] [US3] Add unit tests for request logging and panic recovery in
  `internal/httpserver/middleware/request_logger_test.go` and
  `internal/httpserver/middleware/recoverer_test.go`, asserting structured fields, correlation IDs,
  recovery behavior, and absence of sensitive values
- [X] T030 [P] [US3] Add a lifecycle unit test in `internal/app/app_test.go` using fake components to
  assert stop order (HTTP server before database pool) and bounded shutdown

### Implementation for User Story 3

- [X] T031 [P] [US3] Implement structured request logging middleware in
  `internal/httpserver/middleware/request_logger.go` with `request_id`, `method`, `path`, `status`,
  `duration_ms`, `remote_ip`, and `user_agent`; never log query strings, authorization headers,
  cookies, or bodies
- [X] T032 [P] [US3] Implement panic recovery middleware in
  `internal/httpserver/middleware/recoverer.go` that logs the panic safely and returns a generic
  error response without internal details
- [X] T033 [P] [US3] Implement the CORS middleware wrapper in
  `internal/httpserver/middleware/cors.go` using `chi/cors` with `AllowedOrigins` from configuration;
  an empty allowlist MUST deny all origins
- [X] T034 [US3] Wire the middleware order (request ID, request logger, recoverer, CORS, routes) in
  `internal/httpserver/router.go`
- [X] T035 [US3] Implement the database health monitor in `internal/health/checker.go` with a
  periodic bounded ping; on failure it MUST signal the application to shut down gracefully
- [X] T036 [US3] Implement `signal.NotifyContext` handling for `SIGINT` and `SIGTERM`, bounded
  `http.Server.Shutdown`, pool close after server stop, and correct exit codes in
  `cmd/server/main.go` and `internal/app/app.go`

**Checkpoint**: Observability, CORS, panic safety, database-failure exit, and graceful shutdown work
independently.

---

## Phase 6: User Story 4 - Prepare AWS Integration Without AWS Access (Priority: P3)

**Goal**: The service loads reusable AWS SDK v2 configuration using the default credential chain and
optional endpoint override, without credentials, service clients, or API calls at startup.

**Independent Test**: Start with no AWS credentials and confirm startup succeeds; set
`AWS_REGION`/`AWS_ENDPOINT_URL` and confirm the loader accepts them without network calls.

### Tests for User Story 4

- [X] T037 [P] [US4] Add unit tests in `internal/awsconfig/config_test.go` for region selection,
  endpoint override acceptance, invalid endpoint rejection, and startup success with no credentials
  and `AWS_EC2_METADATA_DISABLED=true`

### Implementation for User Story 4

- [X] T038 [US4] Implement the AWS configuration loader in `internal/awsconfig/config.go` using
  `config.LoadDefaultConfig`, `config.WithRegion`, and an optional `aws.Config.BaseEndpoint`; do not
  construct S3, SQS, SNS, Secrets Manager, or CloudWatch clients
- [X] T039 [US4] Wire AWS configuration loading into `internal/app/app.go` without making it fatal
  when credentials are absent and without making any AWS API call
- [X] T040 [US4] Document AWS credential-chain behavior, the endpoint override, and
  `AWS_EC2_METADATA_DISABLED=true` in `README.md`

**Checkpoint**: AWS integration points are ready and local development remains credential-free.

---

## Phase 7: User Story 5 - Extend the Foundation Safely (Priority: P3)

**Goal**: Generated database access is isolated, migrations are versioned and repeatable, developer
commands are reproducible, and the Terraform scaffold is valid and cheap by default.

**Independent Test**: Run `make sqlc` and confirm generated code appears only in
`internal/database/sqlc/`; add a migration and run `make migrate-up`; run `make test` and
`make test-integration`; run `terraform init` and `terraform validate` and confirm no resources are
planned.

### Tests for User Story 5

- [X] T041 [P] [US5] Add an integration test in `internal/database/sqlc_test.go` behind
  `//go:build integration` that calls the generated `Ping` query and asserts a successful result

### Implementation for User Story 5

- [X] T042 [P] [US5] Add `sqlc.yaml` at repository root with `sql_package: pgx/v5`, queries in
  `db/queries`, and output in `internal/database/sqlc`, plus an example `Ping :one` query whose body
  is `SELECT 1;` in `db/queries/health.sql`
- [X] T043 [US5] Generate sqlc code with the pinned command and commit the output under
  `internal/database/sqlc/`; generated files MUST NOT be hand-edited
- [X] T044 [P] [US5] Add the Terraform scaffold: `terraform/versions.tf`, `terraform/providers.tf`,
  `terraform/variables.tf`, `terraform/outputs.tf`, `terraform/main.tf` (locals for naming and tags
  only), `terraform/environments/dev/terraform.tfvars`,
  `terraform/environments/prod/terraform.tfvars`, and `terraform/README.md`; local state only, no
  resources, no modules
- [X] T045 [US5] Add `sqlc`, `migrate-up`, `migrate-down`, `migrate-status`, `terraform-fmt`, and
  `terraform-validate` targets to `Makefile`, pinning tool versions
- [X] T046 [US5] Document the extension workflow (adding queries, migrations, and future resources)
  and all developer commands in `README.md`

**Checkpoint**: The foundation is safely extensible with generated access, repeatable migrations,
reproducible commands, and a valid infrastructure scaffold.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final verification across all stories.

- [X] T047 [P] Run `gofmt -l .` and `go vet ./...` and fix all findings
- [X] T048 [P] Run `make lint` (`golangci-lint`) and fix all findings
- [X] T049 Run `make test` across `internal/...` and `cmd/...` and confirm all unit tests pass
  without Docker, PostgreSQL, or AWS
- [ ] T050 Run `make test-integration` using `internal/testutil/postgres.go` and confirm integration
  tests pass with Testcontainers and no AWS account
- [ ] T051 [P] Run `terraform fmt -check`, `terraform init -backend=false`, and `terraform validate`
  in `terraform/` and confirm no resources are planned
- [ ] T052 Run the `specs/001-backend-foundation/quickstart.md` validation steps 1–12 end to end
- [X] T053 [P] Perform a scope and security review of `internal/`, `db/`, `cmd/`, and routes: no
  business logic, business tables, business endpoints, authentication flows, secrets, metrics,
  tracing, or profiling endpoints; only `/health` and `/ready` exist
- [X] T054 Finalize `README.md` with troubleshooting and the verified command list

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup; BLOCKS all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational. Delivers the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational and touches `internal/config/config.go`,
  `internal/app/app.go`, and `cmd/server/main.go`; run after US1 to avoid file conflicts.
- **User Story 3 (Phase 5)**: Depends on Foundational and edits `internal/httpserver/router.go`,
  `internal/app/app.go`, and `cmd/server/main.go`; run after US1 and US2.
- **User Story 4 (Phase 6)**: Depends on Foundational; edits `internal/app/app.go`, so run after US3.
- **User Story 5 (Phase 7)**: Depends on Foundational; independent of US2–US4 and can run in parallel
  with them if file ownership is coordinated.
- **Polish (Phase 8)**: Depends on all desired stories.

### User Story Dependencies

- **US1 (P1)**: No story dependencies; MVP.
- **US2 (P2)**: No functional dependency on US1, but shares config/bootstrap files; sequence after
  US1.
- **US3 (P2)**: Uses the router and bootstrap from Foundational/US1; sequence after US2.
- **US4 (P3)**: Uses bootstrap; sequence after US3.
- **US5 (P3)**: Independent of US2–US4; can proceed once Foundational is complete.

### Within Each User Story

- Tests are written first and must fail before implementation.
- Package implementation precedes wiring.
- Wiring precedes documentation.
- Each story completes before the next priority starts, unless explicitly parallelized.

### Parallel Opportunities

- Setup tasks T003, T004, T005, T007, T008 use different files and can run together after T001–T002.
- Foundational tasks T010, T011, T017 use different files and can run together; T009 must precede
  T012 and T015.
- US1 tests T018 and T019 and migration T022 use different files and can run together.
- US3 middleware tasks T031, T032, T033 use different files and can run together; T034 depends on
  them.
- US5 T041, T042, and T044 use different files and can run together.
- Polish T047, T048, T051, T053 use different scopes and can run together.

---

## Parallel Example: Setup

```bash
# After T001 and T002:
Task: "Add .gitignore and .dockerignore at repository root"
Task: "Add .env.example at repository root"
Task: "Add .golangci.yml at repository root"
Task: "Add Dockerfile and docker-compose.yml"
Task: "Add README.md skeleton"
```

## Parallel Example: User Story 3

```bash
# Middleware files are independent:
Task: "Implement request logger in internal/httpserver/middleware/request_logger.go"
Task: "Implement recoverer in internal/httpserver/middleware/recoverer.go"
Task: "Implement CORS wrapper in internal/httpserver/middleware/cors.go"

# Then wire them:
Task: "Wire middleware order in internal/httpserver/router.go"
```

## Parallel Example: User Story 5

```bash
Task: "Add sqlc.yaml and db/queries/health.sql"
Task: "Add Terraform scaffold under terraform/"
Task: "Add integration test in internal/database/sqlc_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. **STOP and VALIDATE**: run `quickstart.md` steps 1–6 and confirm local run, migrations, health,
   and readiness.
5. This is the minimum usable foundation.

### Incremental Delivery

1. Setup + Foundational → scaffolding ready.
2. US1 → local run works (MVP).
3. US2 → invalid configuration fails safely.
4. US3 → observability, CORS, database-failure exit, graceful shutdown.
5. US4 → AWS integration point ready without credentials.
6. US5 → generated access, repeatable migrations, Terraform scaffold, reproducible tooling.
7. Polish → full quickstart validation and scope/security review.

### Parallel Team Strategy

1. Team completes Setup + Foundational together.
2. After Foundational:
   - Developer A: US1 (MVP).
   - Developer B: US5 (sqlc/Terraform/tooling) in parallel, coordinating root-level files.
3. After US1:
   - Developer A: US2 then US3.
   - Developer B: US5 completion.
4. After US3: US4, then Polish.

---

## Notes

- `[P]` marks tasks that touch different files and have no incomplete dependencies.
- `[US#]` labels map tasks to spec user stories for traceability.
- Generated sqlc code in `internal/database/sqlc/` MUST NOT be hand-edited.
- Migrations MUST run through `cmd/migrate` or `make migrate-*`, never at application startup.
- Health and readiness responses MUST never expose secrets, hosts, versions, or internal errors.
- The database-failure policy is strict: any database unavailability causes a clean exit.
- Commit after each task or logical group.
