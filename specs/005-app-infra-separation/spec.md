# Feature Specification: Separate Business Application from Infrastructure Layers

**Feature Branch**: `005-app-infra-separation`

**Created**: 2026-11-12

**Status**: Draft

**Input**: User description: "Refactor the existing Go repository to establish a clear separation between Business Application and Infrastructure, following Clean Architecture and Dependency Inversion principles while preserving the existing Modular Monolith architecture. Prevent business modules from depending directly on PostgreSQL, SQLite, AWS SDK, SNS, SQS, S3, Redis, Redshift, or other concrete infrastructure technologies. This is an architectural refactoring task: preserve business rules, API contracts, database schemas, event payloads, authentication behavior, synchronization behavior, and existing tests."

## Clarifications

### Session 2026-11-12

- Q: How should application-owned data types be shaped when replacing generated database row types? → A: Use minimal row-shaped application-owned structs; keep the refactor mechanical and map persistence rows in infrastructure adapters rather than redesigning business entities.
- Q: Where should the shared authenticated-caller context live when every business module currently imports it? → A: Keep `Principal` as a public type in the auth application module; other business modules may depend on that public type while remaining independent of auth persistence and adapters.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Enforce Dependency Direction Across Layers (Priority: P1)

A maintainer can inspect the codebase and confirm that business modules depend only on their own domain types, application services, platform utilities, and interfaces they own. Concrete database, messaging, storage, cache, security, scheduler, and observability implementations live outside the business modules and are reached only through those interfaces.

**Why this priority**: The primary outcome of this refactor is an enforced dependency rule. Without it, the repository remains vulnerable to future business code importing infrastructure libraries directly.

**Independent Test**: Run an automated architecture-boundary check that analyzes package imports, then introduce a deliberate forbidden import in a temporary branch and confirm the check fails. Restore the import and confirm the check passes with no cycles.

**Acceptance Scenarios**:

1. **Given** the refactored repository, **When** package imports are analyzed, **Then** no package under the business application layer imports another module's concrete repository, a database driver, an AWS SDK package, a Redis client, a cloud identity SDK, or any infrastructure implementation package.
2. **Given** two business modules that collaborate, **When** their dependency is inspected, **Then** the consumer declares the interface it needs and the provider satisfies that interface without the consumer importing the provider's persistence or transport internals.
3. **Given** the complete package graph, **When** it is checked for cycles, **Then** no import cycle exists.
4. **Given** a proposed change that violates a layer rule, **When** the automated boundary check runs, **Then** the change is rejected with a message identifying the offending package and import.

---

### User Story 2 - Preserve Existing Business Behavior and API Contracts (Priority: P1)

An API consumer continues to use the same endpoints, request and response envelopes, validation messages, authentication and authorization rules, pagination behavior, idempotency behavior, audit records, and synchronization semantics after the refactor.

**Why this priority**: An architectural refactor is only acceptable if externally observable behavior is unchanged. Regressions here would break clients and violate the constitution's synchronization and data-ownership contracts.

**Independent Test**: Run the existing unit, integration, OpenAPI contract, route-inventory, authentication, idempotency, and synchronization tests without weakening them. Compare endpoint inventory and response contracts before and after the refactor.

**Acceptance Scenarios**:

1. **Given** the pre-refactor test suite, **When** the refactored code is tested, **Then** all behavior-preserving tests pass with only import, package, or test-location changes where unavoidable.
2. **Given** the documented API surface under `/api` plus the technical `/health` and `/ready` endpoints, **When** the route inventory is compared before and after the refactor, **Then** the route set, HTTP methods, status codes, request shapes, and response envelopes are unchanged.
3. **Given** existing authentication, refresh-token rotation, rate limiting, audit emission, booking idempotency, and guest document secrecy behavior, **When** their tests run, **Then** the behavior is unchanged.
4. **Given** existing PostgreSQL migrations and synchronization triggers, **When** the application is built and the migration command is run, **Then** no schema, migration, trigger, or change-log behavior has changed.

---

### User Story 3 - Keep Runtime Composition Explicit and Deployment-Specific (Priority: P2)

An operator or maintainer can see that the composition root assembles each runtime from explicitly chosen infrastructure adapters. The API runtime initializes only the dependencies it needs, the migration command remains an independent deployment step, and future worker or hub runtimes can reuse business modules without inheriting unrelated cloud infrastructure.

**Why this priority**: The project serves a cloud API and an offline-first ship hub. A single global initialization path that forces every runtime to depend on every technology would undermine the separation and the offline-first architecture.

**Independent Test**: Start the API runtime and the migration command with valid configuration for the current repository. Inspect the composition root to confirm that PostgreSQL, AWS configuration, signing, rate limiting, and Firebase adapters are selected explicitly and that no business package performs this wiring.

**Acceptance Scenarios**:

1. **Given** the API runtime configuration, **When** the application starts, **Then** the composition root wires the PostgreSQL repositories, authentication adapters, and HTTP transport, and the business modules receive only interfaces and platform values.
2. **Given** the migration runtime, **When** the migration command runs, **Then** it uses the PostgreSQL persistence adapter and does not require HTTP, Firebase, Redis, or signing dependencies.
3. **Given** no worker or hub entrypoint exists in the current repository, **When** the target structure is documented, **Then** no empty worker or hub package is created solely to match the target diagram.
4. **Given** a future SQLite-backed hub runtime, **When** it is designed, **Then** it can reuse business modules by supplying its own persistence and external-capability adapters without importing PostgreSQL or cloud-only adapters.

---

### User Story 4 - Establish Clear Ownership of Business and Technical Code (Priority: P2)

A developer can locate a file by responsibility: business rules and use cases in the application layer, protocol handling in transport, external integrations in infrastructure, generic utilities in platform, and wiring in the composition root.

**Why this priority**: Clear ownership reduces accidental coupling and makes future changes reviewable against the constitution's modular-monolith and AWS-readiness principles.

**Independent Test**: Review the final package layout and verify that each package has a single documented responsibility, that HTTP route registration has moved out of business modules, and that persistence implementations are physically separate from use cases.

**Acceptance Scenarios**:

1. **Given** any business module, **When** its package is inspected, **Then** it contains domain models, business rules, use-case services, and interfaces it owns, but no HTTP handlers, route registration, SQL query construction, or SDK client construction.
2. **Given** any HTTP endpoint, **When** its implementation is traced, **Then** request decoding, validation mapping, and response serialization occur in the transport layer, which calls an application interface.
3. **Given** any external integration, **When** its implementation is traced, **Then** the concrete client lives in an infrastructure package and implements an interface declared where the capability is consumed.
4. **Given** a shared utility, **When** its package is inspected, **Then** it contains no business rule or module-specific policy.

---

### User Story 5 - Prevent Regressions with Documentation and Guardrails (Priority: P3)

A maintainer can rely on updated architecture documentation, an architectural decision record, and an automated dependency check to keep the new structure from eroding over time.

**Why this priority**: Documentation and automated enforcement turn a one-time refactor into a durable architectural boundary.

**Independent Test**: Read the architecture documentation and ADR, then run the architecture test. Confirm the documented structure matches the actual repository and that the test detects a deliberately introduced violation.

**Acceptance Scenarios**:

1. **Given** the completed refactor, **When** the architecture documentation and README are reviewed, **Then** they describe the new layer structure, package mapping, runtime composition, and dependency rules.
2. **Given** the constitution requires an ADR for changes to module boundaries and infrastructure, **When** the refactor is delivered, **Then** an ADR records the decision, rationale, alternatives, and consequences.
3. **Given** the new boundary rules, **When** the repository is checked in continuous integration, **Then** the architecture test enforces them automatically.

---

### Edge Cases

- A business module needs data from another module: the consumer must define the interface it needs and receive an implementation from the composition root; it must not import the provider's repository or service implementation.
- A shared helper is needed by both application and infrastructure: it belongs in the platform layer only if it contains no business rules; otherwise it must remain in the owning application module.
- A generated database type is used in an existing public interface: the refactor must replace it with a minimal row-shaped application-owned type, with mapping performed by the persistence adapter and no change to business behavior.
- A development-only implementation exists alongside a production one, such as a local signing key, a development document cipher, or a development principal provider: it must live outside business packages or behind an application-owned interface so production code does not imply a cloud dependency.
- A capability exists only as an interface with no production adapter yet: do not create an empty infrastructure package; keep the interface only if the business use case genuinely needs it.
- The target structure lists SQLite, SNS, SQS, S3, Redshift, or EventBridge but the current repository has no corresponding implementation: do not create placeholder packages or fake adapters.
- A business package currently contains both HTTP route registration and domain services: the package must be split rather than moved wholesale.
- A test depends on a package's old import path: update the import path without changing the test's assertions or coverage.
- The migration command and server currently share initialization code that performs cloud-only setup: separate their composition so the migration runtime does not acquire unnecessary dependencies.
- The architecture check itself imports or enumerates sealed package lists: its rules must be updated to reflect layers rather than becoming a stale denylist that allows new violations.
- A module is present in the codebase but absent from the initial target mapping, such as `tenant`: it must still be migrated according to its actual responsibility rather than deleted or left behind.
- An infrastructure adapter needs to translate driver-specific errors into application errors: translation belongs in the adapter or a persistence-specific helper, not in the business module.

## Requirements *(mandatory)*

### Dependency Direction

- **FR-001**: The repository MUST be organized into five logical layers: application (business modules and use cases), infrastructure (technical adapters), transport (external protocols), platform (generic technical utilities), and composition root (runtime assembly).
- **FR-002**: Packages in the application layer MUST NOT import infrastructure packages, AWS SDK packages, database drivers, SQL query generation packages, Redis clients, cloud identity SDKs, or other concrete external-system clients.
- **FR-003**: Packages in the application layer MUST NOT import the transport layer or contain HTTP request decoding, HTTP response serialization, route registration, or middleware behavior.
- **FR-004**: Infrastructure packages MUST implement interfaces owned by the application modules or platform packages they serve. They MUST NOT force application modules to import infrastructure packages to obtain those interfaces.
- **FR-005**: Transport packages MUST contain HTTP server construction, routing, handlers, middleware, request validation mapping, and response serialization. They MUST NOT contain business rules, direct database access, or direct external-system calls.
- **FR-006**: Platform packages MUST contain only generic technical utilities such as configuration loading, common errors, pagination primitives, and test utilities. Business-specific policy MUST NOT be moved into platform.
- **FR-007**: The composition root MUST be the only layer allowed to import both application and infrastructure packages for runtime assembly, and it MUST construct adapters and inject them into application use cases.
- **FR-008**: Command entrypoints MUST delegate runtime assembly to the composition root rather than constructing infrastructure clients or business services themselves.
- **FR-009**: The package graph MUST remain acyclic.
- **FR-010**: Interfaces that a business module consumes from another module or from an external capability MUST be declared by the consuming business module, not by the provider or the infrastructure adapter. Shared public value types, such as the authenticated principal owned by the auth application module, MAY be imported directly; this exception does not permit importing that module's repositories, adapters, or transport code.
- **FR-011**: No dependency-injection framework may be introduced unless one already exists in the repository.

### Package Migration

- **FR-012**: Every existing business module MUST move under the application layer according to its responsibility. The current business modules are `tenant`, `ship`, `voyage`, `staff`, `partner`, `booking`, `guest`, `sync`, `ai`, `feedback`, `auth`, and `audit`.
- **FR-013**: The `tenant` module is a real business module and MUST be migrated with the other business modules even though it is not listed in the initial target mapping.
- **FR-014**: HTTP handlers, route registration, and request/response mapping currently located inside business modules MUST move to the transport layer and call application interfaces.
- **FR-015**: PostgreSQL repositories, generated query usage, transaction helpers, and driver-specific error mapping currently located inside business modules MUST move to infrastructure persistence adapters.
- **FR-016**: Application services MUST retain business validations, authorization checks, audit decisions, idempotency decisions, and orchestration, and MUST depend on persistence through interfaces.
- **FR-017**: Application public contracts MUST NOT expose generated database row types or driver-specific types. Application-owned, minimal row-shaped data types MUST be used instead, preserving the fields and behavior currently consumed by services, with translation performed in infrastructure adapters. This refactor MUST NOT redesign entities into behavior-rich domain models.
- **FR-018**: The `audit` capability MUST be split into application-owned audit policy and contract definitions and infrastructure-owned audit persistence. Same-transaction audit emission semantics MUST remain unchanged.
- **FR-019**: The `ai` capability MUST keep AI parameter metadata use cases in the application layer. No concrete LLM or provider integration may be invented because none exists in the current repository.
- **FR-020**: The `auth` capability MUST keep SSO exchange, account linking, token issuance and rotation, principal resolution, and authorization policy in the application layer. Cloud identity verification, KMS signing, Redis rate limiting, and authentication persistence MUST be infrastructure adapters behind application-owned interfaces.
- **FR-021**: Development-only authentication implementations, such as local key signing and development header principals, MUST remain outside application business packages or be isolated behind application-owned interfaces.
- **FR-022**: The `guest` document-encryption rule and cipher interface MUST remain in the application layer. Concrete local or KMS-backed cipher implementations MUST live in infrastructure or test support.
- **FR-023**: The `ownership` table-to-module map is architecture metadata used to guard module ownership. It MUST retain its current ownership semantics and MUST move together with the business modules according to the target mapping.
- **FR-024**: The database health monitor is a technical connectivity probe and MUST live outside business modules. Liveness and readiness HTTP handlers MUST remain in the transport layer.
- **FR-025**: Shared HTTP helpers MUST move to the transport layer. Shared configuration, application error types, pagination primitives, and test utilities MUST move to the platform layer.
- **FR-026**: Infrastructure packages MUST be created only for implementations that exist or are strictly required by the current runtime. The repository currently contains PostgreSQL persistence, AWS SDK configuration, KMS signing, Redis rate limiting, cloud identity verification, structured logging, and a database health probe. Empty packages for SNS, SQS, S3, Redshift, EventBridge, SQLite, certificates, or metrics MUST NOT be created.
- **FR-027**: The current repository has no worker or hub entrypoint. Worker and hub composition packages MUST NOT be created as empty placeholders; they MUST be introduced only when a real runtime exists.
- **FR-028**: Cross-module calls MUST continue to use explicit interfaces. A module MUST NOT reach into another module's repository, generated query layer, transport handlers, or unexported implementation.

### Preservation Constraints

- **FR-029**: Existing business rules, validations, authorization boundaries, status transitions, idempotency behavior, audit decisions, and error semantics MUST NOT change.
- **FR-030**: Existing API paths, HTTP methods, status codes, request shapes, response envelopes, pagination behavior, and stable error codes MUST NOT change.
- **FR-031**: Existing PostgreSQL schemas, migrations, triggers, change-log sequencing, snapshots, and synchronization behavior MUST NOT change. Migration files and SQL schemas MUST remain in their current authoritative locations unless a move is strictly required for packaging.
- **FR-032**: Existing authentication behavior MUST NOT change, including Firebase exchange, access-token format, refresh-token rotation and reuse detection, logout-all, rate limiting, and principal authorization.
- **FR-033**: Existing event payloads, event types, and audit record shapes MUST NOT change.
- **FR-034**: Existing PostgreSQL persistence behavior MUST remain intact. The refactor MUST NOT replace pgx, sqlc, or Goose without a technical necessity.
- **FR-035**: The current repository implements PostgreSQL only. SQLite support MUST NOT be fabricated during this refactor. Shared application contracts MUST remain adaptable to a future separate SQLite implementation, but no generic database abstraction may be introduced that hides meaningful PostgreSQL or SQLite differences.
- **FR-036**: The API runtime and migration runtime MUST remain independently initializable. Each runtime MUST initialize only the infrastructure it requires.
- **FR-037**: Existing configuration loading, environment variable names, validation rules, logging conventions, error handling conventions, and graceful-shutdown behavior MUST NOT change.
- **FR-038**: Existing tests MUST be updated only for imports, package locations, and adapted construction. Tests MUST NOT be deleted, skipped, weakened, or rewritten to accept a behavioral change.

### Validation and Documentation

- **FR-039**: The automated architecture-boundary test MUST be updated to enforce the new layer rules instead of the previous flat module allowlist.
- **FR-040**: The architecture test MUST fail when an application package imports an infrastructure package, an AWS SDK package, a database driver, a SQL generation package, a Redis client, or a cloud identity SDK.
- **FR-041**: The architecture test MUST fail when an import cycle exists.
- **FR-042**: The refactor MUST pass `go fmt ./...`, `go vet ./...`, `go test ./...`, and `go build ./...`.
- **FR-043**: Integration tests that can run in the available environment MUST be executed and must pass. Tests requiring unavailable external services may be skipped only according to their existing skip conditions, not by new suppressions.
- **FR-044**: The README project layout and `docs/system-architecture.md` MUST be updated to describe the final layer structure and package mapping.
- **FR-045**: An architectural decision record MUST document the boundary change, the dependency-inversion decision, the runtime composition split, alternatives considered, and any decisions deferred for review.
- **FR-046**: The final report MUST state which packages were moved, split, or merged; which violations were found and resolved; which target packages were intentionally not created; and the exact validation results.

### Key Entities

- **Business module**: A bounded area of domain responsibility, such as ship, voyage, staff, partner, guest, booking, auth, tenant, sync, ai, feedback, audit, or ownership. It owns its domain model, rules, use cases, and consumed interfaces.
- **Application-owned contract**: An interface declared by the business module that needs a capability, such as a repository, audit appender, signer, verifier, rate limiter, or cipher.
- **Shared application value type**: A public, business-meaningful value type owned by one application module and safely imported by other application modules, such as the authenticated principal. It must not carry persistence or transport implementation details.
- **Infrastructure adapter**: A concrete implementation of an application-owned contract, such as a PostgreSQL repository, KMS signer, Firebase verifier, Redis limiter, or local development cipher.
- **Transport handler**: A protocol-specific entrypoint that decodes a request, invokes an application use case, and serializes a response.
- **Platform utility**: A generic, business-agnostic helper such as configuration, shared errors, pagination tokens, or test harnesses.
- **Composition root**: The runtime assembly layer that selects infrastructure implementations, injects them into application services, and starts the API or migration runtime.
- **Layer violation**: Any import that contradicts the required dependency direction, including application-to-infrastructure imports, business-to-transport imports, or package cycles.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Zero application-layer packages import infrastructure, AWS SDK, database driver, SQL generation, Redis, cloud identity, or transport packages, as verified by an automated import analysis.
- **SC-002**: The complete package graph contains zero import cycles.
- **SC-003**: One hundred percent of existing route-inventory and OpenAPI contract tests pass against the same API surface and response envelopes.
- **SC-004**: One hundred percent of existing behavior-preserving unit tests pass, with changes limited to imports, package locations, and construction wiring.
- **SC-005**: All existing integration tests that can execute in the available environment pass; no test is newly skipped or weakened.
- **SC-006**: The API and migration runtimes both build and start with their existing configuration; readiness and liveness behavior is unchanged.
- **SC-007**: Database schema, migration files, triggers, event payloads, and synchronization behavior show no functional change.
- **SC-008**: A deliberately introduced layer violation causes the architecture check to fail, demonstrating that the rule is enforced rather than merely documented.
- **SC-009**: The documented directory structure matches the repository, and every created package corresponds to a real responsibility.
- **SC-010**: No empty infrastructure package or placeholder runtime is introduced solely to mirror the target diagram.

## Assumptions

- The Go module path remains `github.com/ren0503/tycoon`, and the build and test commands remain the existing repository commands.
- The current repository implements a cloud API and a migration command only. The target diagram's worker and hub runtimes are architectural direction, not current implementations.
- The current repository implements PostgreSQL persistence only. No SQLite implementation exists. SQLite directory placeholders and a generic cross-database abstraction are therefore out of scope for this refactor.
- The current repository has no SNS, SQS, S3, Redshift, EventBridge, certificate-authority, or metrics implementation. Only AWS SDK configuration and KMS signing are present. Infrastructure packages for absent technologies must not be created.
- Redis is used for authentication rate limiting, Firebase is used for cloud identity verification, and KMS is used for signing in production. These are the concrete integrations that must move behind application-owned contracts.
- The `tenant` module is retained as a business module even though the initial target mapping omits it.
- The `ownership` table-to-module map is treated as architecture metadata and follows the requested mapping into the application layer.
- Test utilities may depend on infrastructure packages because they are test support, but production application packages may not.
- The HTTP API prefix remains `/api`, and the technical endpoint paths `/health` and `/ready` remain unchanged.
- Existing naming conventions, error messages, logging fields, package-level documentation style, and code standards are preserved.
- The refactor is delivered incrementally and must not introduce features, new AWS services, new libraries, or schema changes.
- Architecture terminology and layer names are the externally mandated product of this refactor, not incidental implementation choices. This exception is recorded in the requirements checklist.
