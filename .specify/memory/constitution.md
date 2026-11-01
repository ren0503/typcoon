<!--
Sync Impact Report
==================
Version change: none (uninitialized scaffold) → 1.0.0
Modified principles: N/A (initial ratification; all template placeholders replaced)
Added sections:
  - Core Principles (6 principles)
  - Technical & Data Constraints
  - Development Workflow & Quality Gates
  - Governance
Removed sections: none
Follow-up TODOs: none
-->

# Cruise Ops Constitution
<!-- Yacht operations platform: cloud backend, onboard hub, staff iPad, and guest apps. -->

## Core Principles

### I. Offline-First Onboard Operations (NON-NEGOTIABLE)

All onboard business operations MUST continue to function when the satellite link is unavailable.
The ship hub MUST operate independently of the cloud, and every client MUST degrade gracefully
rather than block on connectivity.

- The staff iPad app MUST use local storage plus an operation queue with idempotency keys; only
  actions requiring an immediate decision (approvals, emergencies) may require a live hub connection.
- The guest app MUST serve cached data and queue writes when neither hub nor internet is reachable.
- Every degraded mode in the architecture document MUST have a defined behavior, and UI MUST
  display connection status and the last successful sync time.
- Operational time MUST come from the hub; client device clocks MUST NOT be trusted.

Rationale: vessels routinely lose connectivity at sea. Losing the network must never stop
scheduling, task execution, check-ins, or emergency response.

### II. Single Write Owner per Data Item

Each data item MUST have exactly one authoritative write owner at any point in time. Dual writes
to the same item are prohibited.

- The cloud is authoritative for master data (vessels, decks, rooms, staff profiles, partners,
  guest manifest) and for AI parameters.
- The hub is authoritative for operational data (shift plans, tasks, attendance, check-ins,
  incidents, AI audits) while the vessel is underway.
- Ownership MUST follow the documented voyage phases and MUST be exposed to clients via the
  `ownership` status (`cloud` or `ship`).
- Conflicts MUST be resolved by ownership, not by last-write-wins on timestamps.

Rationale: clear ownership prevents divergent state between PostgreSQL and SQLite and removes the
need for CRDTs or database-level replication.

### III. Hub-Initiated, Idempotent Synchronization

Only the hub MAY communicate with the cloud, and the hub MUST always initiate outbound
connections; the cloud MUST NOT connect inbound to a vessel.

- Cloud-to-hub data MUST travel through snapshots and change-log deltas; hub-to-cloud data MUST
  travel through an outbox. Database replication MUST NOT be used as the transport.
- The cloud MUST write `change_log` and allocate the per-ship `seq` in the same transaction as
  the data change, so sequence order matches commit order.
- The hub MUST apply each batch and update `last_seq` in a single transaction; replaying an old
  batch MUST be harmless because of `version`.
- Upstream batches MUST be deduplicated by `(hub_id, local_seq)`; the cloud MUST reject data from
  an older `epoch`.
- Every write request MUST carry an `Idempotency-Key`; hub requests MUST also carry
  `X-Hub-Epoch` and `X-Schema-Version`.
- A hub MUST receive data only for its own vessel and current voyage.

Rationale: intermittent satellite links and heterogeneous databases make at-least-once delivery
with idempotent application the only safe model.

### IV. AI Recommends, Deterministic Rules Decide (NON-NEGOTIABLE)

AI components (solver, forecasting model, local LLM) MUST only produce recommendations. A
deterministic rule engine, not AI, MUST validate hard constraints.

- Hard constraints (minimum rest, total hours, certifications, minimum security staffing) MUST be
  enforced by the rule engine before any plan is applied.
- Autonomy levels MUST be configured in the cloud, published to the hub with safe defaults, and
  enforced consistently.
- Only the Captain MAY override rest-period rules, and every override MUST be logged with a reason.
- An AI kill switch MUST always be available and MUST switch the system to manual assignment.
- Every AI decision and every applied plan MUST be auditable; plans MUST be versioned and
  recalculated on conflict rather than blindly overwritten.

Rationale: scheduling errors affect safety and may violate maritime labor regulations. AI output
is non-deterministic and cannot be the compliance boundary.

### V. Security and Data Protection by Default

Security controls MUST be applied by default and MUST NOT depend on optional configuration.

- All transport MUST use TLS; hub-to-cloud MUST use mTLS with X.509 device certificates from the
  project Private CA.
- Sensitive fields (passports, emergency contacts) MUST be encrypted at the application layer
  using envelope encryption with KMS; hub disks MUST be encrypted.
- Access tokens MUST be short-lived; refresh tokens MUST be rotated and stored only as hashes, and
  reuse MUST revoke the entire token family. Cryptography MUST use proven libraries, never
  hand-rolled implementations.
- The hub MUST store guest data only for the current voyage and MUST delete it after the voyage.
- Revocation lists MUST be distributed to hubs for offline enforcement.
- Sensitive operations and all AI decisions MUST be recorded in audit logs. GDPR and applicable
  customer-country data-protection rules MUST be honored.

Rationale: the platform processes passports and location data, and devices operate in physically
accessible environments with untrusted local networks.

### VI. Simplicity and Operational Pragmatism

The system MUST favor the simplest architecture that meets the requirements, and added complexity
MUST be justified in an ADR.

- The cloud backend MUST remain a Go modular monolith with modules communicating through
  interfaces; it MUST NOT be split into microservices without an approved ADR.
- Deployment MUST use EC2 with Auto Scaling, not Kubernetes, unless an ADR supersedes ADR-01.
- One primary hub per vessel plus a standby hub MUST be preferred over leader election.
- APIs MUST be stateless; sessions MUST live in JWTs or Redis and files in S3.
- YAGNI MUST be applied: features explicitly out of scope (payments, accounting, third-party PMS,
  walk-in ticket sales) MUST NOT be implemented without a constitution amendment.

Rationale: a small team must operate a fleet of edge devices and a cloud platform; every extra
moving part increases failure modes and on-call burden.

## Technical & Data Constraints

The following constraints are non-negotiable implementation standards for all features.

- **Cloud stack**: Go, PostgreSQL 15+, EC2 + Auto Scaling, RDS Multi-AZ, ElastiCache, SQS/SNS,
  EventBridge Scheduler, S3, KMS/Private CA, CloudWatch. Infrastructure MUST be managed as code
  (Terraform or CDK).
- **Edge stack**: Go and SQLite (WAL mode, single writer) on Ubuntu/Jetson, with processes isolated
  and resource-limited via systemd and cgroups.
- **Schema conventions**: UUID v7 primary keys; common columns `id`, `ship_id`, `created_at`,
  `updated_at`, `version`, `deleted_at`; statuses as `text` + `CHECK`; soft delete only on synced
  tables (no physical DELETE).
- **Cross-engine parity**: PostgreSQL and SQLite MUST maintain separate migration sets but share a
  synchronization contract (table list, columns, rules) and automated tests that prevent schema
  drift. The repository layer MUST be an interface with Postgres and SQLite implementations.
- **Integrity**: overlapping partner slots and voyages MUST be prevented by PostgreSQL exclusion
  constraints; where SQLite lacks them, the application MUST use a `BEGIN IMMEDIATE` transaction.
  Redis locks MAY reduce load but MUST NOT be the final authority.
- **API conventions**: REST per OpenAPI, `/v1` prefix, cursor-based pagination, consistent error
  format, and the documented status codes (409 stale epoch, 426 schema upgrade required, 429 rate
  limited).
- **Migrations**: database migrations MUST run as a separate step before deployment and MUST NOT
  run at instance startup.

## Development Workflow & Quality Gates

Every change MUST pass these gates before merge and deployment.

- **Spec before code**: features MUST start from a written specification and plan; implementation
  MUST trace to requirements.
- **Testing**: unit and integration tests MUST run against real PostgreSQL (testcontainers) and
  real SQLite; OpenAPI contract tests MUST pass.
- **Synchronization contract tests**: automated tests MUST prove PostgreSQL and SQLite schemas
  match and that replaying an old batch is harmless.
- **Offline and networking tests**: changes affecting sync or clients MUST be validated by
  disconnect tests, hub restarts with iPads in use, and simulated satellite latency.
- **AI tests**: rule-engine edge cases MUST be covered, solver results MUST be compared against the
  heuristic fallback, and new AI parameters MUST run in shadow mode before activation.
- **Security**: dependency and image scanning, OAuth flow tests, device-revocation tests, and
  VLAN-isolation checks MUST pass.
- **CI/CD**: linting, tests, security scanning, staging deployment, then production, with a
  documented rollback plan. Production MUST NOT receive an unreviewed change.
- **Review**: every pull request MUST verify compliance with this constitution; any violation MUST
  be fixed or explicitly justified and recorded in an ADR.

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
  Core Principles. Complexity that cannot be justified against Principle VI MUST be rejected.
- **Runtime guidance**: `docs/system-architecture.md` and `docs/schema.sql` are the authoritative
  technical references and MUST be updated when architecture or schema decisions change.
- **Deferred scope**: items declared out of scope in the architecture document remain out of scope
  until this constitution is amended.

**Version**: 1.0.0 | **Ratified**: 2026-11-01 | **Last Amended**: 2026-11-01
