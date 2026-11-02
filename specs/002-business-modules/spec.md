# Feature Specification: Initialize Business Modules and Database Models

**Feature Branch**: `002-business-modules`

**Created**: 2026-11-01

**Status**: Draft

**Input**: User description: "Initialize the business-layer foundation of the Yacht Management System backend using a Modular Monolith architecture. Implement the initial business modules, database models, repositories, services, and appropriate API foundation based on the existing PostgreSQL schema and architecture document. Every SQL table must be assigned to exactly one owning module. Do not implement advanced business workflows."

## Clarifications

### Session 2026-11-01

- Q: Which business modules should expose REST endpoints in this initial phase? → A: `tenant`, `ship`, `voyage`, `staff`, `partner`, `booking`, and `guest`; `auth`, `sync`, `audit`, `ai`, and `feedback` remain internal.
- Q: How should protected business endpoints behave before full login and token issuance exist? → A: Default-deny; local development may use an explicitly enabled development principal provider that cannot activate in production.

### Session 2026-11-02

- Q: When a new record references a parent row that does not exist or has been soft-deleted, what should the service do? → A: Reject uniformly: references to soft-deleted or nonexistent parents fail validation with a not-found/validation error for all modules.
- Q: When a client repeats a write request with the same idempotency key but a different request payload, what should the API return? → A: Reject with a 409-style conflict error indicating the idempotency key was already used with a different payload.
- Q: Within this phase's "update basic fields" scope, may API clients change a booking's or service booking's status directly, or is status left untouched until the later lifecycle workflows exist? → A: `bookings.status` is immutable this phase; `service_bookings.status` may be updated directly to its allowed check-constrained values.
- Q: Should basic business operations write entries to the audit log in this phase, or is audit wiring deferred to the later features that define full security workflows? → A: Sensitive-resource writes only: `guest`, `booking`, and `staff` create/update/soft-delete operations emit audit entries; other modules defer audit wiring.
- Q: What page size should list endpoints default to, and what is the maximum page size a client may request? → A: Default 20 records per page, maximum 100.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find One Clear Owner for Every Table (Priority: P1)

A backend developer inspects the project and can identify exactly one business module responsible for each table in the PostgreSQL schema. Modules expose small public interfaces and never reach into each other's internal packages or repositories.

**Why this priority**: Unclear ownership is the root cause of hidden coupling in a modular monolith. The feature is only maintainable if ownership is explicit before models, queries, and services are generated.

**Independent Test**: Compare every table in the authoritative schema against the ownership map and run the module-boundary test. The test fails if a table has zero or multiple owners, or if one module imports another module's internal implementation.

**Acceptance Scenarios**:

1. **Given** the authoritative PostgreSQL schema, **When** the ownership check runs, **Then** every table is assigned to exactly one module.
2. **Given** two business modules, **When** their dependencies are inspected, **Then** they communicate only through explicit interfaces and create no import cycle.
3. **Given** a new table is added later, **When** the ownership check runs, **Then** the missing assignment is reported as a failure rather than silently ignored.
4. **Given** the application starts, **When** modules are composed, **Then** all module dependencies are injected by the application layer.

---

### User Story 2 - Trust That Models Match the Real Schema (Priority: P1)

A backend developer generates and uses database models that faithfully represent the PostgreSQL schema, including nullability, UUIDs, JSONB, arrays, time-zone-aware timestamps, encrypted byte fields, generated identities, and database-managed versioning.

**Why this priority**: Models that drift from the schema cause data loss, incorrect null handling, and broken synchronization. Schema fidelity is a prerequisite for every repository and service.

**Independent Test**: Apply the schema to an empty PostgreSQL database, generate the database access layer, and verify that every table, column, primary key, foreign key, check constraint, index, and trigger is present with the expected type and nullability.

**Acceptance Scenarios**:

1. **Given** an empty PostgreSQL database, **When** the versioned migration runs, **Then** the complete authoritative schema is created without modification.
2. **Given** the complete schema, **When** model generation runs, **Then** a model exists for every table and no handwritten duplicate model is introduced without justification.
3. **Given** a nullable column, **When** its model is inspected, **Then** nullability is represented explicitly rather than collapsed into a zero value.
4. **Given** JSONB, array, UUID, byte, date, time, numeric, and time-zone-aware timestamp columns, **When** their models are inspected, **Then** each is represented by an appropriate type.
5. **Given** an application-created record, **When** its identifier is generated, **Then** it is a UUID v7 generated before the insert.
6. **Given** an update to a versioned table, **When** the update is committed, **Then** the existing database triggers maintain `updated_at`, `version`, and `change_log` behavior.

---

### User Story 3 - Use Basic Data Access Without Raw SQL (Priority: P2)

A service developer can create, read, list, update, and soft delete supported business records through repository and service interfaces. System-managed and append-only tables expose only the operations that make sense for them.

**Why this priority**: The business layer needs a consistent data-access foundation before feature workflows can be built. Forcing CRUD onto every table would produce unsafe and misleading APIs.

**Independent Test**: For each module with applicable operations, exercise the basic service and repository against a real PostgreSQL database and confirm the expected rows, errors, soft-deletion behavior, and transaction outcomes.

**Acceptance Scenarios**:

1. **Given** a supported create request, **When** required inputs are valid, **Then** the record is stored and returned with database-generated values.
2. **Given** a record identifier, **When** the record is fetched, **Then** the repository returns it or a meaningful not-found error.
3. **Given** a list request with a page size and cursor, **When** records exist, **Then** a stable page and a next cursor are returned.
4. **Given** a supported update, **When** the change is valid, **Then** only permitted fields change and versioning remains database-managed.
5. **Given** a table with soft deletion, **When** a record is deleted, **Then** `deleted_at` is set and the record is excluded from normal reads.
6. **Given** a system-managed or append-only table, **When** its service interface is inspected, **Then** unsupported CRUD operations are absent.
7. **Given** multiple writes that must be atomic, **When** any operation fails, **Then** the transaction rolls back and no partial state is committed.
8. **Given** a database constraint violation, **When** the repository returns the error, **Then** the service maps it to a meaningful application error without leaking SQL details.

---

### User Story 4 - Consume a Safe API Foundation (Priority: P2)

A client developer can use a small, documented REST surface under `/v1` for business resources that are safe to expose. Protected endpoints enforce tenant and ship scope, validate input, paginate with cursors, return a consistent error format, and accept an idempotency key for applicable writes.

**Why this priority**: The business layer must eventually serve clients, but exposing every table would leak internal synchronization state and sensitive data. A deliberately narrow API foundation prevents security regressions.

**Independent Test**: Call documented endpoints with valid, invalid, unauthenticated, and cross-tenant requests and verify the documented success and error responses. Attempt to reach internal tables through ordinary CRUD routes and confirm they do not exist.

**Acceptance Scenarios**:

1. **Given** an unauthenticated request to a protected endpoint, **When** the request is processed, **Then** it is rejected without revealing resource existence.
2. **Given** an authenticated principal scoped to one tenant or ship, **When** it requests another tenant's or ship's data, **Then** access is denied.
3. **Given** an invalid request body, **When** it is submitted, **Then** the API returns a consistent validation error and no record is written.
4. **Given** a list endpoint, **When** a cursor and page size are supplied, **Then** results follow the documented cursor semantics.
5. **Given** a write endpoint that supports idempotency, **When** the same request is repeated with the same key, **Then** no duplicate record is created.
6. **Given** internal synchronization, token, and audit tables, **When** the API route table is inspected, **Then** no ordinary CRUD endpoints exist for those tables.
7. **Given** a sensitive field, **When** a response is produced or a log is emitted, **Then** the sensitive value is omitted, redacted, or represented only as an opaque reference.

---

### User Story 5 - Verify Correctness with Real Database Tests (Priority: P3)

A quality engineer can run unit tests and PostgreSQL-backed integration tests locally and in CI. The tests prove schema fidelity, constraint enforcement, foreign-key behavior, soft deletion, pagination, transaction rollback, and module boundaries.

**Why this priority**: Mocked repositories cannot prove that the real schema, triggers, exclusion constraints, and transaction behavior work. These tests protect the highest-risk parts of the business foundation.

**Independent Test**: Run the documented test commands from a clean checkout with container support and confirm all unit and integration tests pass.

**Acceptance Scenarios**:

1. **Given** a container runtime, **When** the integration test command runs, **Then** a disposable PostgreSQL instance is started and the schema is applied.
2. **Given** invalid data that violates a check, unique, foreign-key, or exclusion constraint, **When** the write is attempted, **Then** the database rejects it and the test observes a mapped application error.
3. **Given** a soft-deleted record, **When** normal list operations run, **Then** the record is not returned.
4. **Given** a cursor page, **When** the next page is requested, **Then** no record is duplicated or skipped.
5. **Given** a transaction failure, **When** the operation completes, **Then** the test verifies that earlier writes in the transaction were rolled back.
6. **Given** the module dependency graph, **When** the boundary test runs, **Then** no cross-module internal import or cycle is found.

---

### Edge Cases

- A caller supplies a duplicate value protected by a partial unique index, such as an email, IMO number, port code, staff code, room number, voyage code, port-call sequence, or idempotency key.
- A caller creates a voyage, partner slot, or room booking that overlaps an existing active record and the PostgreSQL exclusion constraint rejects it.
- A caller references a parent row that does not exist or has been soft-deleted; the database foreign key permits the reference, but the service rejects it uniformly with a not-found or validation error (see Session 2026-11-02 clarified rule).
- A caller supplies an invalid status, role, type, or department value that violates a check constraint.
- A caller supplies a JSONB value of the wrong shape, such as a non-array `certifications` value.
- A caller supplies an empty or malformed UUID; the application must generate UUID v7 for new records and validate supplied identifiers.
- A caller supplies timestamps without time-zone context or reverses a time range, such as `ends_at <= starts_at` or `arrival_at <= departure_at`.
- A caller soft-deletes a parent that still has active children.
- A caller supplies a cursor that is malformed, belongs to another filter, or points past the end of the result set.
- A caller repeats a request with the same idempotency key but a different payload; the API rejects it with a conflict error.
- Two callers attempt conflicting writes concurrently.
- A caller tries to read or write another tenant's or ship's data by guessing an identifier.
- A caller attempts to access a system-managed table through a business endpoint.
- A caller submits sensitive guest data as plaintext or requests it in a response.
- A development principal provider is enabled while the service is configured for production.
- The integration test environment cannot start PostgreSQL.

## Requirements *(mandatory)*

### Functional Requirements

#### Module Structure and Ownership

- **FR-001**: The business layer MUST be organized into the modules `tenant`, `ship`, `voyage`, `staff`, `partner`, `booking`, `guest`, `auth`, `sync`, `audit`, `ai`, and `feedback`.
- **FR-002**: Each module MUST have an explicit business responsibility and own its models, queries, repositories, services, and public interface.
- **FR-003**: Every table in the authoritative PostgreSQL schema MUST be owned by exactly one module.
- **FR-004**: The implementation MUST NOT create one module per table, and MUST NOT create a generic module that owns all tables.
- **FR-005**: Each module MUST expose a small public interface and keep implementation packages internal to that module.
- **FR-006**: Modules MUST NOT import another module's internal packages, repositories, or query implementations.
- **FR-007**: Cross-module communication MUST use explicit interfaces, and the dependency graph MUST remain acyclic.
- **FR-008**: The application layer MUST compose modules using dependency injection.
- **FR-009**: The module structure MUST preserve the ability to test business logic with fake implementations of dependencies.

#### Schema Fidelity and Database Models

- **FR-010**: The authoritative PostgreSQL schema MUST be delivered as a versioned migration without inventing, removing, or silently changing tables, columns, relationships, constraints, indexes, extensions, or triggers.
- **FR-011**: The migration MUST preserve PostgreSQL-specific features, including partial indexes, exclusion constraints, JSONB, UUID, arrays, check constraints, identity columns, and trigger functions.
- **FR-012**: Database models MUST be generated from the schema through the project's query-generation workflow and used as the database access representation.
- **FR-013**: A model MUST exist for every table, and a column model MUST exist for every column.
- **FR-014**: Nullable columns MUST be represented with an explicit nullable type or an equivalent, documented representation.
- **FR-015**: PostgreSQL UUID, JSONB, array, byte, date, time, numeric, interval-adjacent, and time-zone-aware timestamp types MUST map to appropriate Go types.
- **FR-016**: Composite and non-UUID primary keys, including the composite keys on synchronization tables and the identity key on the audit table, MUST be represented accurately.
- **FR-017**: Application-created UUID primary keys MUST be UUID v7 generated before insert; database defaults remain a fallback, not the primary strategy.
- **FR-018**: Database-generated values and database-managed columns, including `created_at`, `updated_at`, and `version`, MUST NOT be duplicated or contradicted by application logic.
- **FR-019**: Status, role, type, department, and level values protected by check constraints MUST be represented as validated application values in the model, query, or service layer.
- **FR-020**: Separate domain models MAY be introduced only where they provide a clear benefit; unnecessary conversion layers MUST be avoided.
- **FR-021**: Schema ambiguities or gaps MUST be documented separately and MUST NOT be resolved by silently editing the schema.

#### Repository Layer

- **FR-022**: Each module with ordinary data access MUST provide a repository that encapsulates its database queries.
- **FR-023**: Repositories MUST use the project's PostgreSQL driver and generated query layer and MUST propagate the caller's context.
- **FR-024**: Repositories MUST NOT expose raw SQL to the service layer.
- **FR-025**: Repositories MUST return meaningful errors that distinguish not-found, conflict, validation, and internal failures without leaking SQL or sensitive values.
- **FR-026**: Repositories MUST use transactions when multiple database operations must be atomic.
- **FR-027**: Repositories MUST respect foreign keys, unique constraints, check constraints, and exclusion constraints as the final authority.
- **FR-028**: A repository MUST provide only the basic operations needed by its module; generic repository abstractions MUST NOT be introduced.
- **FR-029**: System-managed, append-only, and internal tables MUST expose only the operations appropriate to their lifecycle.

#### Service Layer

- **FR-030**: Each applicable module MUST provide a basic service layer supporting the appropriate subset of create, get by identifier, list, update, and soft delete.
- **FR-031**: Business rules and input validation MUST live in the service layer.
- **FR-032**: Services MUST validate required inputs, enumerated values, ranges, time ordering, identifier format, and parent references before calling the repository. A reference to a parent row that does not exist or has been soft-deleted MUST fail validation with a not-found or validation error for every module; the schema foreign key alone MUST NOT be treated as sufficient because it permits deleted-parent references.
- **FR-033**: Services MUST return consistent application errors that can be mapped to API responses.
- **FR-034**: Services MUST use transactions where multiple operations or cross-module effects must remain atomic.
- **FR-035**: Services MUST NOT force create, update, or delete onto system-managed or append-only records.
- **FR-036**: Services MUST NOT implement advanced workflows in this phase, including AI shift scheduling, voyage preparation and finalization, snapshot generation, cloud-to-hub synchronization, outbox processing, notifications, complex booking lifecycle orchestration, payment processing, or advanced authorization policies.
- **FR-057**: Create, update, and soft-delete operations on `guest`, `booking`, and `staff` resources MUST append an audit entry through the audit module's public interface in the same transaction as the business write, so an audited operation cannot persist without its audit entry; audit wiring for other modules is deferred to later features.

#### API Foundation

- **FR-037**: Business endpoints MUST use REST under the `/v1` prefix and the project's existing router and middleware foundation.
- **FR-038**: The exposed business API MUST be documented with an OpenAPI description.
- **FR-039**: List endpoints MUST support cursor-based pagination with a stable ordering, a default page size of 20 records, and a maximum page size of 100 records; requests above the maximum MUST be rejected or clamped to the maximum, not served unbounded.
- **FR-040**: Error responses MUST use the project's consistent error envelope, including an application error code, a safe message, and the request identifier where available.
- **FR-041**: Request bodies and query parameters MUST be validated before service execution.
- **FR-042**: Write endpoints MUST accept an `Idempotency-Key` where the underlying schema supports idempotency, and MUST behave consistently on repeated requests. Repeating a request with the same key and payload MUST NOT create a duplicate; repeating the same key with a different payload MUST be rejected with a conflict error.
- **FR-043**: Protected endpoints MUST require an authenticated principal and MUST enforce tenant and ship access boundaries. Protected endpoints MUST default to deny; a development principal provider MAY be enabled only through explicit non-production configuration and MUST NOT activate in production.
- **FR-044**: The API MUST NOT expose ordinary CRUD endpoints for internal or system-managed tables, including at minimum `change_log`, `ship_seq`, `refresh_tokens`, `audit_log`, `ops_events`, `snapshots`, and `job_runs`.
- **FR-045**: The initial API MUST expose basic business endpoints only for the `tenant`, `ship`, `voyage`, `staff`, `partner`, `booking`, and `guest` modules. The `auth`, `sync`, `audit`, `ai`, and `feedback` modules MUST remain internal, and the API MUST NOT automatically expose CRUD endpoints for every table.
- **FR-046**: Sensitive fields, including guest document data, emergency contacts, credential hashes, and refresh-token hashes, MUST NOT be returned in API responses or written to logs.
- **FR-047**: Guest sensitive fields MUST be accepted and persisted only as application-layer encrypted values; the initial feature MUST NOT introduce a plaintext storage path.
- **FR-048**: Refresh tokens MUST be stored only as hashes and MUST never be retrievable in plaintext.

#### Testing

- **FR-049**: Tests MUST cover model mapping, repository operations, service operations, validation, database constraints, foreign-key relationships, soft deletion, pagination, transaction behavior, and module dependency boundaries.
- **FR-050**: Database integration tests MUST use a real PostgreSQL instance through the project's containerized test infrastructure.
- **FR-051**: Database correctness MUST NOT rely only on mocked repositories.
- **FR-052**: The documented test commands MUST run locally and in CI with the same behavior.
- **FR-053**: Tests MUST verify that re-running migrations is harmless and that the schema remains at the expected version.

#### Project Readiness

- **FR-054**: The project MUST compile and be runnable locally after this feature.
- **FR-055**: The implementation MUST NOT require a live AWS account for local development or automated tests.
- **FR-056**: The implementation MUST comply with the project constitution, including module boundaries, database integrity, security, testability, and simplicity.

### Table-to-Module Ownership Map

Every table in the authoritative schema is assigned below to exactly one module. The operation column defines the initial scope; it is not a commitment to expose every operation through the API.

| Table | Owning module | Initial operations | Notes |
|---|---|---|---|
| `tenants` | `tenant` | Create, get, list, update, soft delete | Root tenant aggregate; no cross-tenant access. |
| `admin_users` | `auth` | Create, get, list, update, soft delete | Role and status are check-constrained; email uniqueness is partial. |
| `admin_ship_access` | `auth` | Assign, revoke, list | Join table with no timestamps or soft delete; ship existence is validated through the ship module interface. |
| `accounts` | `auth` | Create, get, list, update, soft delete | Guest account identity; not tenant-scoped in the schema. See Schema Observations. |
| `account_identities` | `auth` | Link, get, list, unlink | Unique by provider and provider subject. |
| `refresh_tokens` | `auth` | Issue, rotate, revoke, validate | Internal only; hashes only; never exposed as CRUD. |
| `ships` | `ship` | Create, get, list, update, soft delete | Tenant-scoped; IMO number is unique. |
| `decks` | `ship` | Create, get, list, update, soft delete | Level is unique per ship among non-deleted rows. |
| `areas` | `ship` | Create, get, list, update, soft delete | Belongs to a ship and deck; layout is JSONB. |
| `rooms` | `ship` | Create, get, list, update, soft delete | Number is unique per ship among non-deleted rows; capacity must be positive. |
| `ports` | `voyage` | Create, get, list, update, soft delete | Shared reference data with no ship or tenant scope. Ownership decision documented below. |
| `voyages` | `voyage` | Create, get, list, update, soft delete | No lifecycle workflow; overlap and time-order constraints remain authoritative. |
| `port_calls` | `voyage` | Create, get, list, update, soft delete | Sequence is unique per voyage among non-deleted rows. |
| `staff` | `staff` | Create, get, list, update, soft delete | Tenant- and optionally ship-scoped; languages array and certifications JSONB. |
| `staff_leaves` | `staff` | Create, get, list, update, soft delete | End must be after start. |
| `staff_credentials` | `staff` | Set, rotate, read metadata | System-managed credential; PIN hash is never returned or logged. |
| `shift_templates` | `staff` | Create, get, list, update, soft delete | Code is unique per ship among non-deleted rows; times are time-of-day values. |
| `coverage_requirements` | `staff` | Create, get, list, update, soft delete | Days of week are a small-integer array; min/max staffing are check-constrained. |
| `labor_rules` | `staff` | Create, get, list, update, soft delete | Tenant-wide when ship is null; parameters are JSONB. |
| `autonomy_policies` | `staff` | Create, get, list, update, soft delete | Scheduling governance configuration consumed by the AI module through an interface. |
| `ai_params` | `ai` | Create, get, list, update metadata, soft delete | Signed-package release workflow and signing are out of scope. |
| `partners` | `partner` | Create, get, list, update, soft delete | Partner status is check-constrained. |
| `partner_contracts` | `partner` | Create, get, list, update, soft delete | Links partner to ship for a date range. |
| `partner_staff` | `partner` | Create, get, list, update, soft delete | Partner staff assignment and validity window. |
| `partner_services` | `partner` | Create, get, list, update, soft delete | Bookable service catalog entry. |
| `partner_slots` | `partner` | Create, get, list, update, soft delete | Exclusion constraint prevents overlapping approved slots by area and time. |
| `bookings` | `booking` | Create, get, list, update basic fields, soft delete | Room overlap exclusion; no booking lifecycle orchestration; `status` is immutable in this phase. |
| `service_bookings` | `booking` | Create, get, list, update basic status, soft delete | Idempotency key is unique per booking; `status` may be updated directly to allowed check-constrained values, without lifecycle orchestration. |
| `guests` | `guest` | Create, get, list, update, soft delete | Encrypted document and emergency-contact bytes are opaque; no plaintext path. |
| `boarding_passes` | `guest` | Issue, get, list, revoke | No update or soft delete; revocation uses `revoked_at`. See Schema Observations. |
| `ship_seq` | `sync` | Internal read only | Sequence counter; never exposed as CRUD. |
| `change_log` | `sync` | Internal append and read | Written by database triggers; never exposed as CRUD. |
| `hub_devices` | `sync` | Provision, get, list, update status, retire | Internal device lifecycle; epoch is authoritative. |
| `ship_devices` | `sync` | Register, list, revoke | Token hash is never returned. |
| `revocations` | `sync` | Create, get, list | Internal revocation list; no ordinary CRUD. |
| `snapshots` | `sync` | Record, get, list | System-managed package metadata; no update or delete API. |
| `job_runs` | `sync` | Claim, finish, list | System-managed job bookkeeping; no ordinary CRUD. |
| `ops_events` | `sync` | Append, read | Internal outbox ingestion; deduplicated by hub and local sequence. |
| `audit_log` | `audit` | Append, query | Append-only; no update or delete. |
| `feedback_responses` | `feedback` | Submit, get, list | Anonymized text only; no update or delete in this phase. |
| `feedback_insights` | `feedback` | Record, get, list | AI-generated analysis; append-only; no update or delete in this phase. |

### Ownership Decisions

The initial grouping was reviewed against the actual schema. The following decisions keep ownership consistent:

- **`guests` and `boarding_passes` are owned by `guest`, not `booking`.** The booking module owns the reservation and service-booking transaction. The guest module owns the guest manifest record and the boarding credential. The `booking` capture scope in the database trigger routes guest changes to the right ship; that is a synchronization detail, not a change of ownership. The booking module accesses guest references through the guest module interface.
- **`ports` is owned by `voyage`, not `ship`.** Ports are global reference data consumed by itinerary port calls. The table has no `ship_id` and is not created or managed per vessel. Keeping it in the ship module would imply vessel ownership that the schema does not express.
- **`autonomy_policies` remains in `staff`.** It configures scheduling governance alongside shift templates, coverage requirements, and labor rules. The AI module consumes policy decisions through a staff interface rather than reading the table directly.
- **`accounts` and `account_identities` remain in `auth`.** Authentication and identity lifecycle own these records. The guest module references accounts through the auth interface and does not query auth tables directly.
- **`job_runs` remains in `sync`.** The records coordinate scheduled voyage-phase jobs that produce snapshots and boarding transitions. The voyage module requests work through an interface; it does not read job-run bookkeeping directly.
- **`audit_log` is owned by `audit`.** Other modules emit audit entries through the audit interface and never write the audit table directly. In this phase, `guest`, `booking`, and `staff` write operations emit audit entries in the same transaction as the business write; other modules defer audit wiring.
- **`feedback_insights` is owned by `feedback`.** The AI module may produce analysis, but the feedback module owns persistence and retrieval of the insight record.

### Key Entities

- **Tenant**: The top-level customer organization that owns ships and tenant-scoped configuration.
- **Ship and spatial hierarchy**: A vessel with decks, areas, and rooms used for operations, cleaning, and guest placement.
- **Port and voyage**: Shared port reference data, a voyage with departure and arrival windows, and ordered port calls.
- **Staff and scheduling configuration**: Staff profiles, leave, credentials, shift templates, coverage requirements, labor rules, and autonomy policies.
- **Partner ecosystem**: Partners, contracts, partner staff, bookable services, and time-limited slots.
- **Booking and guest manifest**: Reservations, service bookings, guest records with encrypted sensitive fields, and revocable boarding passes.
- **Identity and access**: Administrative users, ship access assignments, guest accounts, linked identities, and hashed refresh tokens.
- **Synchronization and audit state**: Per-ship sequence counters, change log, hub and ship devices, revocations, snapshots, job runs, received operational events, and append-only audit records.
- **AI and feedback configuration**: Versioned AI parameter metadata and anonymized post-voyage feedback with derived insights.

## Schema Observations and Deferred Decisions

The following observations are documented to avoid silently changing the schema. No schema modification is authorized by this feature.

- **Guest accounts are global, not tenant-scoped.** `accounts` has no `tenant_id`. Access to an account must be derived through its bookings or an explicit authorization decision. A future schema decision may be required if tenant isolation of guest identities is mandatory.
- **Refresh-token subjects are polymorphic.** `refresh_tokens.subject_type` and `subject_id` have no foreign key. The auth module must enforce subject validity in application code.
- **Administrative ship access has no tenant column.** `admin_ship_access` relies on the referenced admin user and ship for scope. The auth module must validate both through interfaces.
- **Several tables intentionally lack common columns.** `admin_ship_access`, `account_identities`, `refresh_tokens`, `boarding_passes`, `ship_seq`, `change_log`, `snapshots`, `job_runs`, `ops_events`, `audit_log`, `feedback_responses`, and `feedback_insights` do not define the full `created_at`/`updated_at`/`version`/`deleted_at` set. Models and services MUST NOT invent those columns.
- **Some uniqueness constraints are not partial.** `voyages(ship_id, code)`, `ships.imo_number`, and `ports.un_locode` remain unique even after soft deletion. A soft-deleted value cannot be reused without a future schema decision.
- **`boarding_passes` has no synchronization trigger and no `ship_id`.** The schema currently cannot route boarding-pass issuance or revocation through `change_log`. Offline issuance and revocation synchronization must be addressed in a later specification and may require a schema change.
- **`audit_log` uses a bigint identity key, not a UUID.** Its model and API representation must not assume a UUID identifier.
- **Synchronization tables use composite or non-UUID keys.** `change_log`, `ship_seq`, and `ops_events` require composite-key handling rather than a generic single-`id` model.
- **Guest sensitive data is already defined as encrypted bytes.** `document_enc` and `emergency_contact_enc` are opaque byte arrays. The concrete KMS envelope implementation is out of scope for this feature, but no plaintext alternative may be introduced.
- **Idempotency is only schema-backed for two tables.** `bookings` and `service_bookings` define idempotency keys. A shared idempotency store for every write would require a new table and is out of scope. The initial API therefore enforces persistent idempotency only where the schema supports it and documents the remaining gap for a later security or platform decision.
- **AI parameter release is a workflow, not basic CRUD.** `ai_params` can be stored and inspected, but signing, approval, release, shadow-mode execution, and rollback are deferred.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of the 41 tables in the authoritative schema are assigned to exactly one module, verified by an automated ownership check.
- **SC-002**: 100% of schema tables and columns are represented in generated database models, verified by schema-parity tests.
- **SC-003**: Query and model generation completes with zero errors, and generated code is used directly rather than duplicated by handwritten models.
- **SC-004**: The full schema migration applies cleanly to an empty database and re-runs without corrupting state, with all constraints, indexes, extensions, and triggers present.
- **SC-005**: Every module with ordinary data access has a repository and service layer covering only its documented basic operations; system-managed tables expose no ordinary CRUD.
- **SC-006**: All exposed business endpoints are under `/v1`, documented in OpenAPI, and no ordinary CRUD route exists for the internal tables named in FR-044.
- **SC-007**: Integration tests pass against a real, disposable PostgreSQL instance and cover model mapping, constraints, foreign keys, soft deletion, pagination, transaction rollback, and module boundaries.
- **SC-008**: Attempts to read or write another tenant's or ship's data are denied, and sensitive fields are absent from responses and logs in 100% of reviewed cases.
- **SC-009**: A new developer can apply migrations, run the service, and execute the documented test commands locally without a live AWS account.
- **SC-010**: A review finds no advanced business workflow from FR-036 implemented in this feature.

## Technical Constraints

These constraints are part of the agreed scope because this is a backend foundation feature.

- **TC-001**: The implementation MUST use Go, PostgreSQL 15 or later, pgx, sqlc, Goose, Chi, and the existing AWS SDK for Go v2 configuration.
- **TC-002**: Database integration tests MUST use Testcontainers with a real PostgreSQL image from the project's test infrastructure.
- **TC-003**: The solution MUST remain a modular monolith and MUST NOT introduce microservices, Kubernetes, Kafka, CRDTs, or unnecessary distributed components.
- **TC-004**: The feature MUST reuse the existing project structure, configuration, logging, error envelope, middleware, migration runner, and query-generation setup where present.
- **TC-005**: The feature MUST NOT introduce new business tables, columns, relationships, or schema conventions.
- **TC-006**: Library versions MUST remain stable and compatible with the existing project; unnecessary dependencies MUST be avoided.
- **TC-007**: The feature MUST NOT require production AWS resources or credentials for local development or tests.

## Out of Scope

- AI shift scheduling, solver integration, forecasting, and autonomy enforcement workflows.
- Voyage preparation, boarding-open, manifest-finalization, and cutoff workflows.
- Hub snapshot generation, cloud-to-hub synchronization processing, hub outbox processing, and device enrollment.
- Notifications, payment processing, folios, accounting, and third-party PMS integrations.
- Public business API endpoints for the `auth`, `sync`, `audit`, `ai`, and `feedback` modules.
- Complex booking lifecycle orchestration, waitlists, upgrades, and refunds.
- Full OAuth, SSO, MFA, token issuance, and advanced authorization policy administration.
- AI parameter signing, release, shadow-mode execution, and rollback.
- Feedback analysis pipelines and LLM integration.
- Any schema modification or resolution of the documented schema observations.

## Assumptions

- The repository's `docs/schema.sql` is the authoritative PostgreSQL schema. The external file named in the request is assumed to be the same artifact. Where the two differ, the repository baseline wins and no silent change is made.
- The technical foundation from feature `001-backend-foundation` is available; this feature extends it rather than rebuilding configuration, logging, HTTP middleware, database lifecycle, or migration tooling.
- PostgreSQL 16 is acceptable for local and containerized tests even though the schema targets PostgreSQL 15 or later.
- UUID v7 generation uses the project's Go UUID dependency or an equivalent established library; the database `gen_random_uuid()` default remains a fallback.
- Soft deletion applies only to tables that define `deleted_at`. Other tables retain their schema-defined lifecycle.
- The initial API exposes basic business endpoints only for `tenant`, `ship`, `voyage`, `staff`, `partner`, `booking`, and `guest`. Authentication, synchronization, audit, AI, and feedback remain internal in this phase.
- Persistent idempotency is limited to writes whose tables define an idempotency key; the gap for other writes is documented rather than solved by inventing a table.
- Sensitive guest encryption is provided behind an interface. Local development and tests may use a non-KMS implementation, but plaintext storage and plaintext logging are prohibited.
- Authentication and authorization are limited to default-deny enforcement of an authenticated principal and tenant or ship scope. Full login, token issuance, and policy-administration workflows are delivered by later specifications.
- The project uses sequential feature numbering and a single backend service in this phase.
