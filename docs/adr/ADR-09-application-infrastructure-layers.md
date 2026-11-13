# ADR-09: Application and Infrastructure Layers

- Status: Accepted
- Date: 2026-11-13
- Feature: `005-app-infra-separation`

## Context

The backend started as a modular monolith whose business modules imported pgx, sqlc, chi,
AWS, Redis, and Firebase directly. That made business rules hard to test without
infrastructure, mixed HTTP concerns into use cases, and allowed SDK or SQL details to leak
across module boundaries.

## Decision

The codebase is organized into five layers with a single-direction dependency graph:

| Layer | Path | Responsibility |
| --- | --- | --- |
| Application | `internal/application/<module>` | Business types, rules, use cases, and consumer-owned interfaces. |
| Infrastructure | `internal/infrastructure/...` | pgx/sqlc persistence, AWS/KMS, Redis, Firebase, local security, health, logging. |
| Transport | `internal/transport/http/...` | Chi router, middleware, handlers, DTOs, and HTTP request/response mapping. |
| Platform | `internal/platform/...` | Config, shared errors, pagination, and test-only support. |
| Composition root | `internal/app` | Runtime assembly (`RunAPI`, `RunMigrations`) and lifecycle only. |

Application packages never import infrastructure, transport, `internal/app`, or provider SDKs.
Infrastructure imports application interfaces and platform packages. Transport imports
application ports and platform helpers. Only `internal/app` wires application and
infrastructure together. `internal/app/boundaries_test.go` enforces this matrix with
`go list` and fails on forbidden imports, legacy package paths, and import cycles.

### Application-owned types and ports

Application modules declare the row/value types and repository or service interfaces they
consume. Postgres adapters translate between application types and generated `sqlc` types with
`convert.To`/`convert.Slice`; `sqlc.*` and `pgtype.*` do not appear in application source.
Value conversions happen at the port boundary: `pgtype.Date` becomes `*time.Time`,
`pgtype.Time` becomes `staff.TimeOfDay`, and `decimal.NullDecimal` becomes `*float64`.

### Transaction and audit ownership

Application services own transaction boundaries through module-owned `Tx` interfaces.
Infrastructure implements `Begin`/`Commit`/`Rollback` on one pgx transaction. Audit policy and
`audit.Entry` validation live in `internal/application/audit`; audit SQL lives in
infrastructure. `Tx.AppendAudit` uses the same driver transaction as the business writes, so a
failed audit append rolls the business write back.

### Runtime composition

`internal/app/api.go` assembles configuration, PostgreSQL, AWS configuration, auth adapters,
application modules, HTTP transport, and health checks. `internal/app/migrate.go` assembles
only configuration, PostgreSQL, and Goose. Both command entrypoints load config, logger, and
signals and call one composition function. The migration runtime initializes no Firebase,
Redis, KMS, or HTTP server.

## Rejected alternatives

- Keeping handlers or SQL in application modules: preserves the original coupling and makes
  the boundary impossible to enforce mechanically.
- A generic repository abstraction over all modules: hides transaction and audit semantics and
  encourages lowest-common-denominator queries.
- Building the worker or hub runtime now: deferred because no worker, hub, SQLite, SNS, SQS,
  S3, Redshift, or EventBridge behavior is required in this increment.
- Introducing a DI container: explicit constructor wiring in `internal/app` is small enough
  and keeps initialization order auditable.

## Consequences

- Application tests run without PostgreSQL, HTTP, or cloud SDKs.
- Persistence and transport changes stay behind adapters.
- Adding a new dependency to an application package requires a boundary-test update, which
  forces an explicit architectural decision.
- The composition root grows with each runtime; it remains the only package allowed to know
  both application and infrastructure types.

## Deferred decisions

SQLite hub persistence, worker/hub processes, SNS/SQS/S3/Redshift/EventBridge integrations,
and generated certificates remain out of scope for this ADR and the feature.