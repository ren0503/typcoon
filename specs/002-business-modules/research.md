# Research: 002-business-modules

Phase 0 output. All decisions below resolve the plan's open questions; none require schema changes.

## R1. sqlc/pgx type mapping strategy

**Decision**: Keep `sqlc.yaml` `pgx/v5`, enable `emit_json_tags`, `emit_interface`, `emit_pointers_for_null_types`, and add explicit overrides:
- `uuid` → `github.com/google/uuid.UUID` (and `uuid.NullUUID` for the nullable variant)
- `pg_catalog.timestamptz` → `time.Time` / `*time.Time`
- nullable `text`/`varchar` → `*string`
- `jsonb` → `encoding/json.RawMessage` (`*json.RawMessage` nullable)
- `pg_catalog.numeric` → `decimal.Decimal` / `decimal.NullDecimal`
- `date` stays `pgtype.Date` (date-only semantics for `hired_on`, `valid_from/to`, `date_of_birth`)
- `time` stays `pgtype.Time` (time-of-day semantics for shift templates)
- `bpchar` stays `string` (country/currency codes)

**Rationale**: pgtype-first defaults dominate call sites with nil/valid noise; these overrides produce native Go types while pgx v5 encodes them natively. `json.RawMessage` via `emit_json_tags` avoids the base64 `[]byte` trap. Only nullable OR non-nullable can be targeted per `db_type`, so both variants are listed.

**Alternatives considered**: accept pgtype everywhere (more verbose); map JSONB to concrete structs now (premature domain coupling); `float64` for numeric (unsafe); `guregu/null` (extra dep, less idiomatic).

## R2. UUID v7 generation

**Decision**: Application generates `uuid.NewV7()` before insert and passes the id as an explicit query parameter; `gen_random_uuid()` default remains a fallback only (`google/uuid` promoted to a direct dependency).

**Rationale**: Spec FR-017 requires this; v7 gives time-ordered, testable ids with better B-tree locality; PostgreSQL < 18 has no native `uuidv7()`.

**Alternatives considered**: DB default only (v4, not testable); custom SQL v7 (portability pain).

## R3. numeric handling (only `ports.latitude/longitude` use numeric(9,6))

**Decision**: Use `shopspring/decimal.Decimal` override + `jackc/pgx-shopspring-decimal` registered in `AfterConnect`.

**Rationale**: The pgx wiki's recommended path; avoids retrofitting generated signatures later. New deps are stable and pinned — satisfying TC-006 (three total new deps: decimal, pgx-decimal, pgerrcode).

**Alternatives considered**: `pgtype.Numeric` (`*big.Int` + exp pain); `float64` for coordinates (silent precision issues).
**Note**: if the team prefers zero new deps, the fallback is keeping `pgtype.Numeric` for these two columns; decision is implementation-cheap to change.

## R4. Migration delivery + verification

**Decision**: `db/migrations/00002_business_schema.sql` contains the authoritative DDL (`docs/schema.sql`) unmodified (extension, 41 tables, indexes, triggers, functions); add a trailing placeholder Gooses annotation only as required by the engine. Update the empty `db/schema.sql` placeholder to be the same schema content (sqlc input). Tests apply it via the Goose **Provider API** with `//go:embed` against a testcontainers Postgres.

**Rationale**: FR-010/011 (verbatim delivery), SC-004 (clean apply + re-run), and spec assumption "Goose is present". Provider API (goose ≥ 3.16) is the supported, concurrency-safe programmatic path.

**Alternatives considered**: exec goose CLI in tests (brittle); hand-run DDL in test harness (drift); editing schema during migration (forbidden by FR-010/TC-005).

## R5. Module boundaries and dependency enforcement

**Decision**: Each module owns one package `internal/<module>/` containing: `module.go` (public interface + constructor), `service.go` (validation/business rules), `repo.go` (sqlc-backed persistence, tx-scoped), `handlers.go` (chi transport for exposed resources; internal modules have none). `internal/app` is the composition root. Boundary enforcement: (1) `internal/httpserver` is the only package allowed to mount routes; (2) depguard rules in `.golangci.yml` forbid direct imports of another module's `repo`/`query` files; (3) a `TestModuleBoundaries` that parses `go list -deps` output asserting the dependency graph is acyclic and cross-module imports only touch the owning module's public files.

**Rationale**: FR-005/006/007/008; keeps testability with fakes (FR-009) at the interface level; DB transport stays inside the owning module where ids/status enums are known.

**Alternatives considered**: one god `handlers` package (boundary rot); global router init() registration (hidden deps); module-internal `internal/` subpackage trick (does not actually hide from sibling module packages).

## R6. Cross-module data access (FR-007 / ownership decisions)

**Decision**: Public `interface` file per module is the only import surface. Concretely: `partner` validates ships via `ship.ShipService`; `booking` needs `voyage`, `partner`, `ship` (and `guest` reference) interfaces; `guest` needs `booking` and `auth` (for account existence); `audit` exposes `Appender` that `booking`/`guest`/`staff` services depend on (FR-057 emission is transactional: the audit append runs inside the same tx as the business write via a passed-in `pgx.Tx`-capable appender interface).

**Rationale**: Ownership decisions in the spec name these exact seams; same-transaction audit prevents "operation succeeded, audit missing".

**Alternatives considered**: services reading peer tables directly (violates FR-006); audit emission best-effort outside the tx (silent audit loss).

## R7. Authentication / default-deny

**Decision**: One `Authenticate` middleware wraps the `/v1` tree; it resolves a `Principal` (tenant id, allowed ship ids, role) via a `PrincipalProvider` interface. Config `auth.mode = "dev" | "jwt"` selects the provider; `dev` requires a non-empty explicit principal allowlist and config validation hard-fails at startup when `env = production` (and logs a prominent marker in requests served under dev mode). Fine-grained ship/tenant checks (`Principal.CanAccessShip(id)` etc.) live in handlers/services where the "object" is known.

**Rationale**: Spec clarification "Default-deny; local development may use an explicitly enabled development principal provider that cannot activate in production" (FR-043); knowledge of object scope must sit near the resource.

**Alternatives considered**: all authz in middleware (can't know per-object scope); no middleware, handler-only checks (one forgotten call = leak).

## R8. Error envelope and pgx → app error mapping

**Decision**: Reuse `internal/apperrors`. Add one `database/pgerrors.go` mapper: SQLSTATE-based via `pgerrcode` constants — `23505` → 409 conflict (map to `idempotency_conflict` when the retry was idempotency-keyed and payload differs), `23503` → 422 invalid reference, `23514` → 422 validation (constraint name → field hint), `23P01` → 409 overlap conflict, `40001`/`40P01` → bounded retry then 503, `22P02` → 400 invalid identifier, `22032` → 422 bad JSON, `pgx.ErrNoRows` → 404. Catch-all class mapping: `23*` → 409, `22*` → 422, else 500 with only the request id surfaced. `Idempotency-Key` replay with different payload is detected by comparing a canonical payload hash computed from the request body, returning 409 (per FR-042).

**Rationale**: FR-025/033/040 + consistent error envelope; SQLSTATE is stable and locale-independent; verified against PostgreSQL 16 errcodes.

**Alternatives considered**: regex on error text (locale-dependent); hand-written string constants for SQLSTATE codes (unnecessary when maintained `jackc/pgerrcode` exists).

## R9. Cursor pagination

**Decision**: Opaque cursor = base64(JSON `{v, sort fields..., id, f}`) where `id` is the row's uuid tiebreaker and `f` is a short hash of the canonicalized filter+sort parameters. Keyset predicate on `(sort_col, id)`; queries always `ORDER BY col..., id` and fetch `limit+1` to detect `next`; `next_cursor: null` at page end; decode errors, filter-hash mismatch, and end-of-list cursors → 400 with `error.code = invalid_cursor`; cursor length capped. Default limit 20, max 100 (FR-039), over-max clamped, not erroring.

**Rationale**: Stable ordering + tiebreaker is the canonical keyset pattern; filter-hash prevents "cursor belongs to another filter" ambiguity; no COUNT query; excludes soft-deleted rows so "no duplicates/skips" holds for list semantics.

**Alternatives considered**: offset pagination (skips under concurrent writes); HMAC-signed cursors (key machinery without security benefit; the tuple carries no secret state).

## R10. Idempotency behavior with schema-backed keys

**Decision**: Persisted idempotency applies only where the schema has key columns (`bookings(account_id, idempotency_key)`, `service_bookings(booking_id, idempotency_key)`). In the service, inside the create transaction: lookup by key → found and payload-hash matches → return the existing record with 201; on `23505` re-read and race-resolve. Key present with a different payload hash → 409 `idempotency_conflict`. Global idempotency for other writes stays out of scope (documented gap, spec Schema Observations).

**Rationale**: The unique constraint *is* the idempotency implementation; no new table (TC-005) and no snapshot store needed.

**Alternatives considered**: dedicated `idempotency_keys` table (needs migration = schema invention); treat mismatch as 422 (RFC 9110 15.5.10 favors 409).

## R11. Validation split (validator tags vs service rules)

**Decision**: `validator/v10` struct tags on request DTOs for shape/local invariants (required, format=uuid, oneof for transport enums mirroring CHECK values, gte/lte field checks); manual service validation for semantic/context rules (parent-reference existence/soft-delete state, enum legality vs lifecycle, time ordering interacting with voyage/exclusion logic, ownership rules).

**Rationale**: FR-032 + Clarification: parent references to soft-deleted rows are **rejected uniformly at the service layer**; these checks need I/O (repo lookups) and cannot live in static tags. Transport enums stay in tags to normalize before service logic.

**Alternatives considered**: all-manual (drift + verbose); tags reaching into repositories (layering violation).

## R12. OpenAPI authoring + contract enforcement

**Decision**: Hand-authored `contracts/openapi.yaml` (this plan phase) → delivered as the project's `openapi/openapi.yaml` during implementation (FR-038). CI/test enforcement uses `kin-openapi/openapi3` loader + spec lint, and request/response validation mounted as a test-only filter in the contract tests. No codegen dependency in production code paths.

**Rationale**: Hand-authored spec remains authoritative and refactor-proof; contract tests fail loudly on drift (spec US5.6 + checklist "OpenAPI contract tests"); keeps runtime deps unchanged (TC-006).

**Alternatives considered**: `swag` annotation scanning (silent drift); `ogen` full codegen (schema stops being authoritative; big new dependency); runtime schema validation middleware (per-request cost, YAGNI).

## R13. Audit emission (FR-057)

**Decision**: `audit.Appender` interface exposes transaction-aware append; `guest`, `booking`, and `staff` services call it inside the same create/update/soft-delete transaction. Record shape uses `audit_log` (`actor_type`, `action`, `entity`, `entity_id`, `ship_id`, `voyage_id`, `source='cloud'`, `details` JSONB with before/after diff hashes, never sensitive values). Audit table queries are append+query only.

**Rationale**: Clarification 2026-11-02 (Option C: guest/booking/staff only); same-tx append per constitution VI ("sensitive operations recorded") and FR-057; no best-effort emission.

**Alternatives considered**: async/queued audit outbox (needs worker; FR-036 excludes); audit-all-modules (overkill now).

## R14. Test infrastructure

**Decision**: Extend `internal/testutil`: start disposable Postgres via testcontainers module, apply migrations with Goose Provider + `embed.FS`, expose pool + context helpers. Unit tests remain DB-free (`sqlmock`-free; repo logic tested against real Postgres in integration). Suite covers: migration apply+re-run idempotency (FR-053), schema parity per SC-002, constraint violation mapping (unique/check/FK/exclusion), soft-delete visibility, cursor pagination (no dup/skip), tx rollback, module boundary graph, dev-principal production guard, idempotency replay/mismatch, and cross-tenant deny (SC-008).

**Rationale**: FR-049–053 and SC-007; same commands locally/CI via Make targets `test` and `test-integration` (already exist).

**Alternatives considered**: in-memory Postgres (not real); skip real DB in unit tests (fails SC-007).
