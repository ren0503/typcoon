# Phase 0 Research: Application/Infrastructure Separation

**Feature**: `005-app-infra-separation`
**Date**: 2026-11-12
**Status**: Complete; all decisions are reflected in `plan.md`, `data-model.md`, and contracts.

## R1. Layer classification and package ownership

**Decision**: Classify every package into exactly one of five layers.

- **Application** (`internal/application/<module>`): business row types, use-case services,
  validation/authorization/idempotency/audit decisions, module public interfaces, and
  consumer-owned capability ports.
- **Infrastructure** (`internal/infrastructure/...`): PostgreSQL/pgx/sqlc/Goose, error
  translation, AWS SDK configuration, KMS signing, Redis rate limiting, Firebase
  verification, local signing/cipher implementations, logging setup, and database health.
- **Transport** (`internal/transport/http/...`): chi router/server, HTTP handlers, DTOs,
  request decoding/validation mapping, response serialization, principal extraction, and
  middleware.
- **Platform** (`internal/platform/...`): configuration, shared application errors,
  cursor pagination, and test-only support. No business policy.
- **Composition root** (`internal/app`): the only package that imports application and
  infrastructure together to assemble an API or migration runtime.

**Rationale**: The spec mandates these five layers. Explicit ownership makes the boundary
mechanically testable and gives every file one home.

**Rejected alternatives**: Keeping handlers/repositories inside application modules with
naming conventions was rejected because the current boundaries test already shows that
conventions drift. A single `internal/core` package was rejected because it would recreate
a flat allowlist and hide dependency direction.

## R2. Application-owned data shape

**Decision**: Use minimal row-shaped application structs and module-local query parameter
structs. Do not introduce behavior-rich domain entities.

- Every `sqlc` model consumed by an application service or handler gets an equivalent
  module-local type with the same fields and JSON tags where applicable.
- Every `sqlc.*Params` passed into a repository becomes a module-local `Create*Params`,
  `Update*Params`, or `List*Params` type with the same fields.
- Row types retain standard/pure value types: `uuid.UUID`, `uuid.NullUUID`, `string`,
  pointers, slices, `json.RawMessage`, `time.Time`, and `*time.Time`.
- Driver/value conversions happen only in adapters:
  - `pgtype.Date` ↔ `*time.Time` (`Valid == false` ↔ `nil`).
  - `pgtype.Time` ↔ `staff.TimeOfDay{Microseconds, Valid}`; handlers format `HH:MM[:SS]`
    exactly as today.
  - `decimal.NullDecimal` ↔ `*float64` for `voyage.Port` latitude/longitude; handlers and
    service validation keep using `*float64`.
  - `[]byte` and `json.RawMessage` remain as-is.
- Public service interfaces return application row types, never `sqlc` types.
- Generated `sqlc` code remains generated and unmodified except for its package/import path.

**Rationale**: The clarification requires a mechanical refactor with row-shaped types, not
DDD entity redesign. This minimizes behavior drift and keeps handlers' JSON output stable.

**Rejected alternatives**: Behavior-rich entities were rejected by clarification. Re-exporting
`sqlc` types through type aliases was rejected because it would still expose generated
persistence types in application contracts and keep the dependency visible.

## R3. Transaction boundary: application-owned transaction port

**Decision**: Application services orchestrate transactions through module-owned `Tx`
interfaces. The adapter owns the driver transaction.

For modules with multi-write/audited use cases (`ship` structure-import confirmation,
`staff`, `booking`, `guest`):

```go
type Tx interface {
    // Module-specific reads/writes needed inside the transaction.
    CreateBooking(context.Context, CreateBookingParams) (Booking, error)
    GetBookingByIdempotency(context.Context, GetBookingByIdempotencyParams) (Booking, error)
    // Cross-cutting app-owned audit append; implemented on the same driver transaction.
    AppendAudit(context.Context, audit.Entry) error
    Commit(context.Context) error
    Rollback(context.Context) error
}

type Repository interface {
    Begin(context.Context) (Tx, error)
    // Non-transactional reads/writes used by the same module.
    GetBooking(context.Context, uuid.UUID) (Booking, error)
    // ...
}
```

- The service keeps the current sequence: `Begin`, business reads/writes, audit append,
  `Commit`, and deferred `Rollback` exactly where it exists today.
- The PostgreSQL adapter `Tx` wraps one `pgx.Tx`; its methods call `sqlc.New(tx)` and map
  rows/errors. `Commit`/`Rollback` call pgx and translate errors.
- The app transaction port exposes no `pgx.Tx`, `pgconn.CommandTag`, `sqlc.Queries`, or
  `pgtype` value.
- A module with no genuine multi-write transaction gets only normal repository methods; no
  empty or speculative `Tx` interface is added.
- `booking` idempotency checks and guest/audit writes remain inside the same transaction.
- Ship import confirmation keeps its existing mid-transaction commit paths for terminal
  failure/already-complete states, because `Commit` remains available on the app-owned `Tx`.

**Rationale**: Transactions are a use-case concern and must remain visible to services to
preserve audit, idempotency, and rollback semantics. A callback-only `WithinTx` would force
special cases for import confirmation and would hide existing commit points. An app-owned
`Begin` handle removes driver types with the smallest behavioral change.

**Rejected alternatives**: Returning `pgx.Tx` behind a type alias still violates FR-002/017.
A high-level repository method containing all confirmation logic would move business
decisions into infrastructure. A transaction callback with no commit control could not
preserve the import's existing terminal commit states.

## R4. Audit policy and same-transaction persistence

**Decision**: Split audit into application policy/contract and PostgreSQL persistence.

- `application/audit` owns `Entry`, entry validation (`Validate`), and audit action/entity
  semantics. No database imports.
- Each module `Tx` embeds an `AppendAudit(context.Context, audit.Entry) error` operation.
- `infrastructure/persistence/postgres/audit` implements `AppendTx(ctx, pgx.Tx, Entry)` for
  adapters and `List` for read support. It calls `audit.Validate`, writes the same columns
  with the same defaults, and uses the caller's pgx transaction.
- `application/auth` continues to construct audit entries and pass them to its high-level
  `Store` methods. The auth adapter appends those entries in the same transaction.
- `audit.Appender`/`audit.Execer` and `pgconn.CommandTag` disappear from application
  contracts.

**Rationale**: FR-018 requires policy/contract vs persistence separation and unchanged
same-transaction semantics. `AppendAudit` on the module transaction is explicit and keeps
audit decisions in the service.

**Rejected alternatives**: A global appender that opens its own transaction would break
rollback semantics. Passing an `Execer` would keep a driver-shaped interface in application
code.

## R5. Cross-module dependencies

**Decision**: Each consumer declares the minimal interface it needs. Public value types may
be imported; implementation packages may not.

| Consumer | Consumer-declared ports | Public value imported |
|---|---|---|
| `voyage` | ship tenant lookup | `auth.Principal` only |
| `staff` | ship tenant lookup | `auth.Principal` only |
| `partner` | ship tenant lookup | `auth.Principal` only |
| `booking` | ship scope lookup, voyage lookup, partner service/slot existence, account existence | `auth.Principal`, `voyage.Voyage` |
| `guest` | booking lookup | `auth.Principal`, `booking.Booking` |
| `feedback` | voyage lookup, ship tenant lookup | `auth.Principal`, `voyage.Voyage` |
| `sync` | voyage lookup, ship tenant lookup | `auth.Principal`, `voyage.Voyage` |

`auth.Principal` stays in `application/auth` per clarification. Importing a provider's
public row value is limited to the method result needed by the consumer; consumers never
import provider repositories, SQL, handlers, or adapters.

**Rationale**: This satisfies FR-010/FR-028 without inventing duplicate row types or
provider-owned consumer interfaces.

**Rejected alternatives**: Keeping provider `Service` interfaces as dependencies was
rejected because it couples consumers to the provider's full surface. Moving shared rows to
a generic `platform/types` package was rejected as a business-type dumping ground.

## R6. Authentication split

**Decision**:

- `application/auth` keeps `Principal`, `Service`, exchange/refresh/logout/link policy,
  refresh rotation/reuse logic, JWT claims/issuance, and application-owned ports:
  `Store`, `Verifier`, `KeySet`, `Signer`, `RateLimiter`, `AccountLookup`, `Clock`.
- `infrastructure/persistence/postgres/auth` implements `Store` and account lookup with
  pgx/sqlc and same-transaction audit.
- `infrastructure/identity/firebase` implements `Verifier`.
- `infrastructure/security/kms` implements `Signer`; `infrastructure/security/local`
  implements local `Signer`/key loading.
- `infrastructure/cache/redis` implements `RateLimiter`.
- `transport/http/middleware` owns the HTTP-facing `Provider` contract and the
  `AccessTokenProvider`, `DevHeaderProvider`, and `DenyAllProvider` implementations. The
  token provider consumes application `TokenValidator`/`AccountLookup` ports and returns
  `auth.Principal`.

**Rationale**: `*http.Request` is transport-specific, while token issuance and principal
policy remain application concerns. This satisfies FR-020/FR-021 and keeps local
development implementations out of business packages.

**Rejected alternatives**: Leaving `auth.Provider` in application would force `net/http`
into the application layer. Moving token issuance entirely to infrastructure would move
security policy out of the application.

## R7. Guest document cipher

**Decision**: Keep `guest.DocumentCipher` and the encryption-presence rule in
`application/guest`. Move `LocalDocumentCipher`, `NewLocalDocumentCipher`, and
`NewDevDocumentCipher` to `infrastructure/security/local`. Tests and composition inject the
local implementation through the application interface.

**Rationale**: FR-022 explicitly requires the rule/interface in application and concrete
implementations in infrastructure/test support. Current behavior and error messages remain.

## R8. PostgreSQL packaging and generated queries

**Decision**: Use `internal/infrastructure/persistence/postgres` for pool/DSN/migrations/error
mapping and `internal/infrastructure/persistence/postgres/sqlc` for generated code.
Per-module adapters live in subpackages (`.../postgres/ship`, `.../postgres/booking`, etc.).
Audit persistence is `.../postgres/audit`.

- `db/migrations/` and `db/queries/` remain authoritative locations.
- `docs/schema.sql` remains the authoritative schema baseline.
- sqlc generation configuration is updated only for the output/package path and import
  path; generated SQL and models stay semantically identical.
- `database.MapPgError` moves to `postgres/errors.go`, including constraint-name mappings.
- There is no generic repository abstraction and no SQLite implementation.

**Rationale**: FR-031/FR-034 require unchanged schema and persistence technology. Per-module
adapter packages keep SQL translation close to the module while physically separated from
use cases.

**Rejected alternatives**: One monolithic `postgres` package was rejected because it makes
module ownership hard to review and encourages cross-module SQL reuse. A generic query
interface was rejected as an unnecessary cross-database abstraction.

## R9. AWS configuration and signing

**Decision**: Move `awsconfig` to `infrastructure/aws`; move KMS signing to
`infrastructure/security/kms`. The application `Signer`/`KeySet` contracts remain in
`application/auth`. The KMS adapter uses the AWS SDK and a narrow `KMSClient` interface.
Local signing loads keys with standard crypto in `infrastructure/security/local`.

**Rationale**: This isolates SDK types and preserves local development without AWS
credentials. It satisfies FR-020, FR-021, and Principle V.

## R10. Shared errors, configuration, pagination, HTTP helpers

**Decision**:

- `apperrors` moves to `internal/platform/errors` but keeps the Go package name
  `apperrors` and all exported symbols/messages to avoid a mass error-code rename.
- `config` moves to `internal/platform/config`; environment names and validation remain.
- `pagination` moves to `internal/platform/pagination`.
- `httpx` moves to `internal/transport/http/httpx`; only HTTP helpers live there.
- `logging` moves to `internal/infrastructure/observability/logging`.
- `testutil` moves to `internal/platform/testutil`; it may import infrastructure because it
  is test-only, as allowed by the spec assumptions.

**Rationale**: These are generic technical utilities, not business policy. Moving them
prevents application modules from importing transport-only helpers while keeping stable
public symbols.

## R11. Runtime composition and command entrypoints

**Decision**:

- `internal/app/api.go` assembles PostgreSQL pool, AWS config, auth adapters, modules, router,
  server, and health monitor.
- `internal/app/migrate.go` assembles only PostgreSQL configuration/pool/Goose.
- `internal/app/modules.go` constructs application services from injected ports.
- `internal/app/app.go` retains lifecycle and graceful shutdown.
- `cmd/server/main.go` and `cmd/migrate/main.go` only load config/logger, establish signal
  context, and call the composition root.
- No machinery is added for worker/hub runtimes; the composition code is structured so a
  future runtime can add its own constructor.

**Rationale**: FR-007/FR-008/FR-036 require explicit runtime-specific assembly. The
migration command must not acquire HTTP/Firebase/Redis/KMS dependencies.

**Rejected alternatives**: Keeping AWS/Firebase/Redis construction in `cmd/server/auth.go`
was rejected because command entrypoints would still assemble infrastructure. A single
global `BuildAll` was rejected because it would force cloud-only dependencies into migration.

## R12. Architecture-boundary enforcement

**Decision**: Replace `internal/app/boundaries_test.go` with a layer-aware test using
`go list -json ./...`.

Rules:

- `internal/application/...` may import only standard library, other
  `internal/application/...`, and `internal/platform/...` (plus pure helper libraries such
  as `google/uuid`, `samber/lo`, `golang-jwt/jwt/v5`, `golang.org/x/crypto`).
- Application packages must fail on `net/http`, chi, `database/sql`, pgx/pgtype/pgconn,
  sqlc, Goose, AWS SDK, Redis, Firebase, testcontainers, and any `internal/infrastructure`,
  `internal/transport`, or `internal/app` import.
- `internal/infrastructure/...` may import application/platform and external adapters, but
  not transport or `internal/app`.
- `internal/transport/...` may import application/platform and HTTP libraries, but not
  infrastructure or `internal/app`.
- `internal/platform/...` may not import application/infrastructure/transport/app, except
  `internal/platform/testutil` may import infrastructure test support.
- Only `internal/app` may import both application and infrastructure.
- The test also performs a full import-cycle check.
- Test contains a table of representative forbidden imports and checks the rule message
  includes package and import; validation includes a temporary forbidden-import mutation.

**Rationale**: FR-039/FR-040/FR-041 require layer rules rather than a stale flat denylist.
Prefix-based classification prevents new layers from silently escaping enforcement.

## R13. Test migration and preservation

**Decision**: Update only imports, package locations, and construction wiring. Preserve all
assertions and integration coverage.

- Handler tests move with handlers; service tests move with services.
- Integration tests move with modules and use relocated `platform/testutil`.
- Existing testcontainers harness, migration application, fixture behavior, and build tags
  remain unchanged.
- Route-inventory and OpenAPI tests continue to assert the same `/api`, `/health`, and
  `/ready` surface.
- Auth freshness/rotation/reuse, booking idempotency, guest secrecy, import rollback, audit,
  and sync tests must remain equivalent.
- No test is newly skipped. Environment-dependent skips remain as they are.

**Rationale**: FR-038 forbids weakening tests. Import-only churn is the expected evidence
that behavior is preserved.

## R14. Documentation and ADR

**Decision**: Add an ADR covering:

- five-layer boundary and dependency direction;
- application-owned row types and persistence ports;
- module-owned transaction handles and same-transaction audit;
- runtime-specific composition;
- alternatives considered (flat module allowlist, callback-only transactions, generic
  repository abstraction);
- deferred decisions (SQLite hub adapter, worker/hub runtimes, future messaging/storage).

Update `README.md` project layout and `docs/system-architecture.md` package/module mapping.
Leave `docs/schema.sql`, `db/migrations/`, and `openapi/` functionally unchanged.

**Rationale**: Principle IX and FR-044/FR-045 require the decision and final structure to be
documented. The ADR prevents future maintainers from reintroducing direct infrastructure
imports.

## Deferred / intentionally not created

- SQLite persistence, migrations, or cross-engine abstraction.
- SNS, SQS, S3, Redshift, EventBridge, certificate-authority, and metrics packages.
- Worker and hub compositions/entrypoints.
- Any new LLM/provider integration.
- A dependency-injection framework.
- Empty transport handlers for `ai`, `feedback`, or `sync` (they expose no HTTP routes).
