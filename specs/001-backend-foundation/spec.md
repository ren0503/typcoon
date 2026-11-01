# Feature Specification: Initialize Backend Foundation

**Feature Branch**: `001-backend-foundation`

**Created**: 2026-11-01

**Status**: Draft

**Input**: User description: "Initialize the initial source code foundation for a production-ready backend service: project structure, core libraries, configuration, infrastructure integration points, and development tooling. Technical foundation only — no business logic, domain models, workflows, or application features."

## Clarifications

### Session 2026-11-01

- Q: When the database is unavailable, should the service exit or keep running and report itself not ready? → A: Exit the process whenever the database is unavailable, at startup or during runtime.
- Q: What observability should this foundation include beyond structured logging and health endpoints? → A: Structured logging only, with health and readiness endpoints.
- Q: How should cross-origin access be controlled in this foundation? → A: Configurable allowlist of origins; deny all origins unless explicitly listed.
- Q: Should the application automatically load a local `.env` file, or read only environment variables provided by the process? → A: Read only process environment variables; `.env` is loaded by developer tooling, not the application.
- Q: Should the Terraform foundation use local state only, or include a remote state backend scaffold? → A: Local state only; document a remote backend as future work.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run the Backend Locally (Priority: P1)

A new developer clones the repository, follows the documented setup, starts the local dependencies,
applies migrations, and runs the service. The service starts, connects to the local database, and
answers health and readiness checks without requiring an AWS account.

**Why this priority**: The foundation is only useful if another developer can reproduce a working
environment from documented commands. This is the primary acceptance gate for the entire feature.

**Independent Test**: From a clean checkout with only documented prerequisites installed, follow the
README to start local dependencies, run migrations, and start the service; verify both technical
endpoints respond successfully.

**Acceptance Scenarios**:

1. **Given** a clean checkout and documented prerequisites, **When** the developer runs the documented
   local setup commands, **Then** the service starts and reports itself healthy.
2. **Given** the local database is running, **When** migrations are executed, **Then** the migration
   step completes successfully and can be re-run without corrupting state.
3. **Given** the service is running, **When** the developer requests the health and readiness
   endpoints, **Then** each returns a success response with no secrets or internal infrastructure
   details.
4. **Given** no AWS account or credentials are available, **When** the service starts, **Then**
   startup still succeeds.

---

### User Story 2 - Fail Fast on Invalid Configuration (Priority: P2)

An operator starts the service with missing or invalid required configuration. The service refuses
to start and reports which required settings are missing or invalid, without exposing secret values.

**Why this priority**: Misconfiguration discovered at runtime causes partial outages. Failing at
startup with actionable, non-sensitive diagnostics prevents silent misbehavior.

**Independent Test**: Start the service with each required setting missing or malformed and confirm
startup fails with a clear, non-sensitive error; then start with valid configuration and confirm
startup succeeds.

**Acceptance Scenarios**:

1. **Given** a required setting is missing, **When** the service starts, **Then** startup fails
   before serving traffic and the error identifies the missing setting.
2. **Given** a setting has an invalid value, **When** the service starts, **Then** startup fails
   and the error identifies the invalid setting.
3. **Given** valid configuration, **When** the service starts, **Then** defaults are applied for
   optional settings and startup succeeds.
4. **Given** a configuration error occurs, **When** the error is logged, **Then** secret values are
   never included in logs or error messages.

---

### User Story 3 - Observe and Shut Down Gracefully (Priority: P2)

An operator runs the service in an environment that sends termination signals. The service emits
structured logs with request identifiers, stops accepting new work, finishes in-flight requests, and
closes database connections within the configured shutdown window.

**Why this priority**: Production readiness depends on clean lifecycle behavior and diagnosable logs.
Abrupt termination risks dropped requests and leaked connections.

**Independent Test**: Send a termination signal while requests are in flight and confirm the service
stops accepting new requests, completes in-flight work, closes connections, and exits within the
configured timeout.

**Acceptance Scenarios**:

1. **Given** the service is serving requests, **When** a termination signal is received, **Then** the
   service stops accepting new requests and exits within the configured shutdown timeout.
2. **Given** an in-flight request, **When** shutdown begins, **Then** the request completes or is
   terminated cleanly before the process exits.
3. **Given** a request is handled, **When** logs are inspected, **Then** the request appears with a
   correlation identifier and structured fields.
4. **Given** a panic occurs during request handling, **When** the request completes, **Then** the
   service recovers, logs the failure safely, and continues serving other requests.

---

### User Story 4 - Prepare AWS Integration Without AWS Access (Priority: P3)

A developer runs the service locally without AWS credentials or network access to AWS. The service
initializes its AWS integration points using the standard credential chain and supports an optional
endpoint override for local emulation, but makes no real AWS API calls during startup.

**Why this priority**: The backend must remain runnable offline while still being ready to use AWS
services in deployed environments.

**Independent Test**: Start the service with no AWS credentials and confirm startup succeeds; start
with an endpoint override configured and confirm the override is accepted without making live calls.

**Acceptance Scenarios**:

1. **Given** no AWS credentials are present, **When** the service starts, **Then** startup succeeds
   and no live AWS API call is required.
2. **Given** an AWS region and optional endpoint override are configured, **When** the service
   initializes, **Then** the integration points use those values.
3. **Given** local emulation is not configured, **When** the service starts, **Then** local
   development still works without it.

---

### User Story 5 - Extend the Foundation Safely (Priority: P3)

A developer adds a future feature module. The existing structure provides clear boundaries, generated
database access separated from handwritten code, versioned migrations, and infrastructure
configuration that can accommodate future resources without rework.

**Why this priority**: The foundation's value is measured by how safely it can be extended. Clear
boundaries and tooling reduce the cost and risk of future work.

**Independent Test**: Generate database access code from the example query, add and run a migration,
and verify unit and integration test commands succeed without touching business logic.

**Acceptance Scenarios**:

1. **Given** the database access configuration and example query, **When** code generation is run,
   **Then** generated code is produced separately from handwritten code.
2. **Given** a new migration file, **When** the migration command runs, **Then** it applies cleanly
   and is recorded as applied.
3. **Given** the documented developer commands, **When** formatting, linting, unit tests, and
   integration tests are run, **Then** each command completes using documented prerequisites only.
4. **Given** the infrastructure configuration, **When** it is validated, **Then** it is structurally
   valid and provisions no expensive resources by default.

---

### Edge Cases

- Required configuration is present but a dependency (database) is unreachable at startup; the
  service MUST exit with a clear, non-sensitive error.
- The readiness endpoint is called before the database connection is established; the service is
  still starting and has not begun serving.
- The database becomes unavailable after startup; the service MUST stop serving, shut down cleanly,
  and exit rather than continue in a degraded state.
- A termination signal arrives while the service is still starting.
- A termination signal arrives twice in quick succession.
- The configured shutdown timeout is shorter than an in-flight request.
- A request contains an oversized or malformed body.
- A request arrives without a correlation identifier.
- A cross-origin request arrives from an origin not present in the configured allowlist; the request
  MUST NOT receive permissive cross-origin headers.
- The service runs with an optional setting absent and must apply a safe default.
- Local AWS emulation is configured but unavailable.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST provide a clean, maintainable structure that separates the application
  entry point, bootstrap, HTTP layer, middleware, configuration, logging, database lifecycle,
  migrations, query layer, AWS integration, health checks, shutdown, shared errors, and test
  utilities.
- **FR-002**: Implementation details MUST be kept internal where appropriate, and the structure MUST
  support adding future feature modules without requiring restructuring.
- **FR-003**: The application MUST load configuration, initialize structured logging, initialize the
  database connection pool, initialize AWS integration, build the HTTP router and middleware,
  register technical endpoints, start serving, handle termination signals, and shut down cleanly.
- **FR-004**: The application MUST fail fast when required configuration or dependencies are invalid.
  If the database is unavailable at startup, the application MUST exit with a clear, non-sensitive
  error. If the database becomes unavailable after startup, the application MUST stop serving, shut
  down cleanly, and exit rather than continue in a degraded state.
- **FR-005**: Configuration MUST be environment-based and MUST be read only from process environment
  variables. The application MUST NOT automatically load a `.env` file; local tooling MAY load one
  before starting the application. Configuration MUST include application name, environment,
  HTTP host and port, HTTP read/write/idle/shutdown timeouts, log level, database connection
  settings, database pool settings, AWS region, optional AWS endpoint override, AWS credential
  configuration compatible with the default credential chain, and an optional allowlist of
  cross-origin request origins.
- **FR-006**: Configuration MUST provide defaults for non-sensitive settings, validation for all
  settings, a clear distinction between required and optional settings, and a documented example
  environment file.
- **FR-007**: The system MUST NOT contain hardcoded credentials, secrets, or production resource
  identifiers.
- **FR-008**: The system MUST establish a PostgreSQL connection pool, expose connection health
  checking, manage the database lifecycle, and support executing versioned migrations.
- **FR-009**: The database access layer MUST include generation configuration, a minimal example query
  that verifies generation, and a clear separation between generated and handwritten code.
- **FR-010**: The system MUST NOT create business tables, business entities, domain schemas, or
  feature-specific database structures. Any initial migration MUST be limited to minimum technical
  requirements.
- **FR-011**: The system MUST initialize reusable AWS client configuration using the default
  credential provider chain, support IAM roles and temporary credentials, support region
  configuration, support endpoint overrides for local development, and avoid hardcoded credentials.
- **FR-012**: The system MUST NOT initialize AWS services that are not currently needed and MUST NOT
  make real AWS API calls during startup unless explicitly required.
- **FR-013**: The HTTP server MUST provide request correlation, structured request logging, panic
  recovery, configurable timeouts, and graceful shutdown. Cross-origin access MUST use a
  configurable allowlist and MUST deny all origins that are not explicitly listed.
- **FR-014**: The HTTP layer MUST expose only technical endpoints for liveness and readiness. These
  endpoints MUST NOT expose secrets, credentials, internal infrastructure details, or sensitive
  error messages.
- **FR-015**: The system MUST NOT expose business APIs, CRUD endpoints, authentication flows, or
  domain-specific routes.
- **FR-016**: The project MUST provide a multi-stage container build, a local orchestration
  configuration, a local database service, environment configuration, dependency health checks, and
  ignore files for containers and version control.
- **FR-017**: The backend MUST be runnable locally without an AWS account, and local AWS service
  emulation MUST be optional rather than mandatory.
- **FR-018**: The project MUST provide a minimal, valid infrastructure-as-code structure with
  version and provider configuration, variables, outputs, environment-specific structure, and naming
  and tagging conventions. Terraform MUST use local state in this foundation; a remote state backend
  MUST be documented as future work and MUST NOT be scaffolded now.
- **FR-019**: The infrastructure structure MUST accommodate future networking, compute, load
  balancing, database, storage, messaging, secrets, and monitoring resources without provisioning
  them now, and MUST NOT introduce Kubernetes, EKS, or a serverless architecture.
- **FR-020**: The project MUST provide documented, reproducible developer commands for formatting,
  linting, unit testing, integration testing, migrations, query generation, container build, and
  container run.
- **FR-021**: The project MUST provide setup documentation sufficient for a new developer to run the
  backend locally.
- **FR-022**: The project MUST include tests for configuration loading and validation, bootstrap
  where practical, health and readiness endpoints, middleware, database connection initialization,
  and graceful shutdown where practical.
- **FR-023**: Unit tests and integration tests MUST be separated. Integration tests MUST be runnable
  locally using containerized dependencies, and no test MUST require a real AWS account.
- **FR-024**: The system MUST NOT include business logic, domain entities or aggregates, product
  modules, business database tables, business API endpoints, CRUD features, authentication or
  authorization workflows, business event processing, or business-specific queues or scheduled jobs.
- **FR-025**: The foundation MUST provide structured logging and technical health endpoints only.
  Metrics, tracing, and profiling endpoints MUST NOT be included in this foundation.

### Technical Constraints

This is an infrastructure foundation feature; the following constraints are part of the agreed scope.

- **TC-001**: The implementation MUST use Go, Chi, PostgreSQL, pgx, sqlc, Goose, Viper, `log/slog`,
  `go-playground/validator/v10`, AWS SDK for Go v2, Go's standard testing package with Testcontainers
  where appropriate, Docker and Docker Compose, and Terraform.
- **TC-002**: Library versions MUST be stable and mutually compatible. Unnecessary dependencies MUST
  be avoided.
- **TC-003**: The solution MUST remain a modular monolith and MUST NOT introduce Kubernetes, EKS, or
  a serverless architecture.
- **TC-004**: The initial infrastructure MUST NOT create expensive AWS resources by default and MUST
  NOT provision production AWS resources beyond the approved foundation.
- **TC-005**: The foundation MUST comply with the project constitution, including module boundaries,
  fail-fast configuration, structured logging, graceful shutdown, idempotent jobs, dependency
  timeouts, and secure handling of secrets.

### Out of Scope

- Business logic, domain models, aggregates, and workflows.
- Yacht, vessel, voyage, staff, guest, booking, or partner modules.
- Business database tables, schemas, and entities.
- Business API endpoints, CRUD features, and authentication or authorization workflows.
- Business event processing, queues, and scheduled jobs.
- Production AWS deployment and real AWS resource provisioning.
- Placeholder business modules and speculative abstractions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new developer can go from a clean checkout to a running service with both technical
  endpoints responding by following only the documented commands, in under 15 minutes on a machine
  with the documented prerequisites.
- **SC-002**: 100% of required configuration settings produce a clear startup failure when missing or
  invalid, and no failure message contains a secret value.
- **SC-003**: The service starts successfully and answers readiness checks with no AWS account,
  credentials, or network access to AWS.
- **SC-004**: The service exits within the configured shutdown window after a termination signal, and
  no in-flight request is abandoned without a logged outcome.
- **SC-005**: Database migrations apply cleanly from an empty database and can be re-run without
  corrupting state.
- **SC-006**: Query generation succeeds from the example query and produces generated code separate
  from handwritten code.
- **SC-007**: All unit tests pass, and integration tests pass locally using containerized
  dependencies, with zero tests requiring a real AWS account.
- **SC-008**: The infrastructure configuration validates successfully and provisions no expensive
  resources by default.
- **SC-009**: A review of the delivered code finds zero business tables, business endpoints, domain
  modules, or business logic.
- **SC-010**: When the database is unavailable at startup or becomes unavailable during runtime, the
  service exits within the configured shutdown window, emits a clear non-sensitive error, and never
  continues serving traffic in a degraded state.

## Assumptions

- Developers have the documented local prerequisites available, including a Go toolchain, Docker, and
  a container runtime compatible with Docker Compose.
- Local development uses containerized PostgreSQL; a remote database is not required.
- AWS access is not available or required during local development or automated tests.
- The initial infrastructure configuration is a scaffold for future deployment, not a deployment
  itself.
- The project uses sequential feature numbering and a single backend service in this phase.
- Non-sensitive settings receive safe defaults; sensitive settings are supplied through the
  environment and are never committed.
- The example database query exists only to prove generation and is not a business feature.
