<!--
Sync Impact Report
==================
Version change: 1.0.0 → 2.0.0
Bump rationale: MAJOR — core principles were removed, renamed, and redefined, and the
principle set expanded from six to ten. Backward-incompatible governance changes.

Modified principles:
  - I. Offline-First Onboard Operations → III. Offline-First Architecture
  - II. Single Write Owner per Data Item → IV. Data Ownership
  - III. Hub-Initiated, Idempotent Synchronization → merged into III. Offline-First Architecture
  - V. Security and Data Protection by Default → VI. Security
  - VI. Simplicity and Operational Pragmatism → X. Simplicity

Added principles:
  - I. Modular Monolith
  - II. Database Integrity
  - V. AWS Readiness
  - VII. Reliability
  - VIII. Testability
  - IX. Specification-Driven Development

Removed principles:
  - IV. AI Recommends, Deterministic Rules Decide — retained as an implementation
    constraint under Technology Constraints rather than a top-level principle.

Renamed sections:
  - Technical & Data Constraints → Technology Constraints
  - Development Workflow & Quality Gates → Quality Standards

Added sections: none beyond the renames above
Removed sections: none

Follow-up TODOs: none
-->

# Yacht Management System Backend Constitution

This constitution governs the cloud backend of the yacht management platform. It applies to the
cloud API, workers, synchronization services, and their supporting infrastructure. Onboard hub,
staff iPad, and guest application behavior is included only where it constrains cloud contracts.

## Core Principles

### I. Modular Monolith

The cloud backend MUST use Go and a modular monolith architecture.

- Each business module MUST have an explicit boundary, its own package, and a documented
  responsibility.
- Modules MUST communicate through interfaces, not by reaching into each other's internal
  packages or tables.
- Cross-module effects MUST be expressed as explicit calls or domain events; hidden coupling
  through shared mutable state is prohibited.
- Microservices MUST NOT be introduced in the initial implementation. Splitting a module out
  requires an approved ADR and a constitution amendment.
- The module boundary MUST make it possible to test a module's business logic with fake
  implementations of its dependencies.

Rationale: a small team must operate the platform. A modular monolith keeps deployment simple
while preserving the option to extract services later if justified.

### II. Database Integrity

PostgreSQL is the source of truth for cloud master data.

- Database constraints MUST enforce data integrity. Application checks MAY supplement
  constraints but MUST NOT replace them.
- Every schema change MUST be versioned through a migration; ad-hoc schema edits are prohibited.
- The provided PostgreSQL schema (`docs/schema.sql`) MUST be treated as the baseline. Changes to
  it MUST be delivered as new migrations and MUST update the baseline documentation.
- Primary keys MUST use UUID v7. Synced tables MUST include `id`, `ship_id`, `created_at`,
  `updated_at`, `version`, and `deleted_at`.
- Statuses MUST be stored as `text` with a `CHECK` constraint.
- Soft delete MUST be used on synced tables; physical `DELETE` MUST NOT be used for synced data.
- Overlapping partner slots and voyages MUST be prevented by PostgreSQL exclusion constraints.
  Application-level checks and Redis locks MAY reduce load but MUST NOT be the final authority.

Rationale: the database is the last line of defense for correctness. Constraints survive bugs,
racing requests, and manual operations that application code cannot control.

### III. Offline-First Architecture

The cloud backend MUST support the cloud-to-hub synchronization model.

- The onboard hub operates independently when internet connectivity is unavailable. Cloud
  behavior MUST NOT assume the hub is reachable.
- The hub initiates all cloud connections. The cloud MUST NOT initiate inbound connections to
  the hub.
- Cloud-to-hub data MUST travel through snapshots and change-log deltas. Hub-to-cloud data MUST
  travel through an outbox. Database replication MUST NOT be used as the transport.
- The cloud MUST write `change_log` and allocate the per-ship `seq` in the same transaction as
  the data change, so sequence order matches commit order.
- Upstream batches MUST be deduplicated by `(hub_id, local_seq)`, and the cloud MUST reject data
  from an older `epoch`.
- Every write request MUST carry an `Idempotency-Key`. Hub requests MUST also carry
  `X-Hub-Epoch` and `X-Schema-Version`.
- A hub MUST receive data only for its own vessel and current voyage.

Rationale: intermittent satellite links make at-least-once delivery with idempotent application
the only safe synchronization model.

### IV. Data Ownership

Every data entity MUST have a clearly defined write owner.

- Cloud master data is owned by the cloud. This includes vessels, decks, rooms, staff profiles,
  partners, guest manifests, and AI parameters.
- Operational data is owned by the hub while the vessel is underway. This includes shift plans,
  tasks, attendance, check-ins, incidents, and AI audits.
- Synchronization MUST NOT introduce ambiguous write ownership. Dual writes to the same entity
  are prohibited.
- Ownership MUST follow documented voyage phases and MUST be exposed to clients through the
  `ownership` status (`cloud` or `ship`).
- Conflicts MUST be resolved by ownership, not by last-write-wins on timestamps.

Rationale: a single write owner prevents divergent state between PostgreSQL and SQLite and
removes the need for CRDTs or database-level replication.

### V. AWS Readiness

AWS integrations MUST use interfaces and dependency injection.

- AWS clients MUST be hidden behind project-owned interfaces so business logic does not depend on
  the AWS SDK directly.
- The backend MUST support local development without requiring AWS credentials. Local
  implementations or emulators MUST be selectable through configuration.
- Production infrastructure MUST be provisioned using Terraform. Manual console changes to
  production infrastructure are prohibited except during an documented incident response.
- Configuration MUST be environment-driven, and the application MUST fail fast when required
  configuration is missing or invalid.

Rationale: interface boundaries keep business logic testable and prevent the application from
becoming locked to one infrastructure implementation.

### VI. Security

All secrets MUST be managed through secure configuration mechanisms.

- No credentials or sensitive information may be committed to source control.
- Sensitive personal information MUST be encrypted. Passports, emergency contacts, and comparable
  fields MUST use application-layer envelope encryption with KMS; hub disks MUST be encrypted.
- All transport MUST use TLS. Hub-to-cloud communication MUST use mTLS with X.509 device
  certificates from the project Private CA.
- Access tokens MUST be short-lived. Refresh tokens MUST be rotated, stored only as hashes, and
  reuse MUST revoke the entire token family. Cryptography MUST use proven libraries, never
  hand-rolled implementations.
- Authorization MUST enforce tenant and ship-level access boundaries on every request. A caller
  MUST NOT read or write data outside its tenant or assigned ships.
- Revocation lists MUST be distributed to hubs for offline enforcement.
- Sensitive operations and all AI decisions MUST be recorded in audit logs. GDPR and applicable
  customer-country data-protection rules MUST be honored.

Rationale: the platform processes passports and location data, and edge devices operate in
physically accessible environments with untrusted local networks.

### VII. Reliability

All asynchronous jobs MUST be idempotent.

- All external dependencies MUST have appropriate timeouts. Unbounded calls are prohibited.
- Workers MUST support graceful shutdown: they MUST stop accepting new work, finish or release
  in-flight work, and exit within a bounded grace period.
- Database migrations MUST be executed as a separate deployment step and MUST NOT run at
  instance startup.
- Retries MUST be bounded and MUST use backoff. Jobs that exhaust retries MUST be routed to a
  dead-letter destination and MUST be observable.
- Long-running work MUST be safe to resume after a crash without duplicating effects.

Rationale: distributed systems fail partially. Idempotency, timeouts, and graceful shutdown
turn partial failures into recoverable events instead of data corruption.

### VIII. Testability

Business logic MUST be testable independently of infrastructure.

- Unit tests MUST cover business rules without requiring a database, network, or AWS account.
- Integration tests MUST be included for database constraints, migrations, repositories, and
  external adapters.
- Critical database constraints and synchronization behavior MUST be tested. Tests MUST prove
  that PostgreSQL and SQLite schemas match and that replaying an old batch is harmless.
- New behavior MUST include tests that fail without the change.
- Test infrastructure MUST be runnable locally and in CI with the same commands.

Rationale: the synchronization contract and data-integrity rules are the highest-risk parts of
the system. They must be verified continuously, not only during manual testing.

### IX. Specification-Driven Development

All major features MUST have specifications, implementation plans, and actionable tasks.

- Implementation MUST satisfy documented acceptance criteria.
- Architectural changes MUST be documented. Decisions that alter module boundaries, data
  ownership, synchronization, or infrastructure MUST be recorded in an ADR.
- Specifications and plans MUST be updated when implementation deviates from them; silent
  divergence is prohibited.
- Runtime guidance in `docs/system-architecture.md` and `docs/schema.sql` MUST be updated when
  architecture or schema decisions change.

Rationale: written specifications make acceptance criteria reviewable and keep the constitution,
plans, and code aligned as the system evolves.

### X. Simplicity

The system MUST favor the simplest architecture that satisfies the requirements.

- Complexity MUST be justified in an ADR before it is introduced.
- Kubernetes, Kafka, CRDTs, and unnecessary distributed systems components MUST NOT be
  introduced into the initial implementation.
- The backend MUST remain a modular monolith with modules communicating through interfaces; it
  MUST NOT be split into microservices without an approved ADR and a constitution amendment.
- Deployment MUST use EC2 with Auto Scaling unless an ADR supersedes this decision.
- One primary hub per vessel plus a standby hub MUST be preferred over leader election.
- APIs MUST be stateless; sessions MUST live in JWTs or Redis and files in S3.
- YAGNI MUST be applied: features explicitly out of scope MUST NOT be implemented without a
  constitution amendment.

Rationale: every extra moving part increases failure modes and on-call burden for a small team.

## Technology Constraints

The following constraints are non-negotiable implementation standards for all features.

- **Cloud stack**: Go, Chi, PostgreSQL, pgx, sqlc, goose, Redis/Valkey, AWS SDK for Go v2,
  Terraform, and Docker.
- **Cloud AWS services**: CloudFront + WAF + ALB, EC2 + Auto Scaling, RDS PostgreSQL (Multi-AZ)
  with RDS Proxy, ElastiCache, SQS/SNS, EventBridge Scheduler, S3, KMS/Private CA/Secrets
  Manager, and CloudWatch.
- **Edge stack**: Go and SQLite (WAL mode, single writer) on Ubuntu/Jetson, with processes
  isolated and resource-limited via systemd and cgroups.
- **Schema conventions**: UUID v7 primary keys; common columns `id`, `ship_id`, `created_at`,
  `updated_at`, `version`, `deleted_at`; statuses as `text` + `CHECK`; soft delete only on
  synced tables (no physical `DELETE`).
- **Cross-engine parity**: PostgreSQL and SQLite MUST maintain separate migration sets but share
  a synchronization contract (table list, columns, rules) and automated tests that prevent
  schema drift. The repository layer MUST be an interface with Postgres and SQLite
  implementations.
- **Integrity**: overlapping partner slots and voyages MUST be prevented by PostgreSQL exclusion
  constraints; where SQLite lacks them, the application MUST use a `BEGIN IMMEDIATE`
  transaction. Redis locks MAY reduce load but MUST NOT be the final authority.
- **API conventions**: REST per OpenAPI, `/v1` prefix, cursor-based pagination, consistent error
  format, and the documented status codes (409 stale epoch, 426 schema upgrade required, 429 rate
  limited).
- **Migrations**: database migrations MUST run as a separate step before deployment and MUST NOT
  run at instance startup.
- **AI and automation**: AI components (solver, forecasting model, local LLM) MUST only produce
  recommendations. A deterministic rule engine, not AI, MUST validate hard constraints (minimum
  rest, total hours, certifications, minimum security staffing) before any plan is applied.
  Autonomy levels MUST be configured in the cloud, published to the hub with safe defaults, and
  enforced consistently. Only the Captain MAY override rest-period rules, and every override
  MUST be logged with a reason. An AI kill switch MUST always be available.

## Quality Standards

Every implementation MUST:

- Compile.
- Pass relevant tests.
- Follow Go conventions.
- Include structured logging.
- Support graceful shutdown.
- Include appropriate error handling.
- Avoid leaking sensitive information.
- Maintain clear module boundaries.

Every change MUST also pass these gates before merge and deployment.

- **Spec before code**: features MUST start from a written specification and plan; implementation
  MUST trace to requirements.
- **Testing**: unit and integration tests MUST run against real PostgreSQL (testcontainers) and
  real SQLite; OpenAPI contract tests MUST pass.
- **Synchronization contract tests**: automated tests MUST prove PostgreSQL and SQLite schemas
  match and that replaying an old batch is harmless.
- **Offline and networking tests**: changes affecting sync or clients MUST be validated by
  disconnect tests, hub restarts with iPads in use, and simulated satellite latency.
- **AI tests**: rule-engine edge cases MUST be covered, solver results MUST be compared against
  the heuristic fallback, and new AI parameters MUST run in shadow mode before activation.
- **Security**: dependency and image scanning, OAuth flow tests, device-revocation tests, and
  VLAN-isolation checks MUST pass.
- **CI/CD**: linting, tests, security scanning, staging deployment, then production, with a
  documented rollback plan. Production MUST NOT receive an unreviewed change.
- **Review**: every pull request MUST verify compliance with this constitution; any violation
  MUST be fixed or explicitly justified and recorded in an ADR.

## Governance

This constitution supersedes all other development practices and ad-hoc conventions. When a
conflict exists, this document wins until amended.

- **Amendments** MUST be proposed through the `/speckit.constitution` workflow, include a written
  rationale and a Sync Impact Report, and be reviewed before adoption. Amendments that change
  safety, data ownership, or security rules MUST also be approved by the project maintainers.
- **Versioning** follows semantic versioning:
  - MAJOR for backward-incompatible removals or redefinitions of principles or governance.
  - MINOR for a new principle or section, or materially expanded guidance.
  - PATCH for clarifications, wording fixes, and non-semantic refinements.
- **Compliance review**: every pull request and architecture decision MUST be checked against the
  Core Principles. Complexity that cannot be justified against Principle X MUST be rejected.
- **Runtime guidance**: `docs/system-architecture.md` and `docs/schema.sql` are the authoritative
  technical references and MUST be updated when architecture or schema decisions change.
- **Deferred scope**: items declared out of scope in the architecture document remain out of scope
  until this constitution is amended.

**Version**: 2.0.0 | **Ratified**: 2026-11-01 | **Last Amended**: 2026-11-01
