# Data Model: Application/Infrastructure Separation

**Feature**: `005-app-infra-separation`
**Date**: 2026-11-12

## Scope

This feature changes **where** data types and persistence code live, not the database model.
There are no new tables, columns, indexes, constraints, triggers, queries, migrations, event
payloads, or API fields. `docs/schema.sql` and `db/migrations/` remain authoritative and
must not change functionally.

The application layer gains minimal, row-shaped types that mirror the fields currently
provided by `internal/database/sqlc/models.go`. Infrastructure adapters translate between
those types, pgx/sqlc values, and the unchanged SQL schema.

## Application Value Types by Module

| Module | Application types | Source table(s) |
|---|---|---|
| `tenant` | `Tenant` | `tenants` |
| `ship` | `Ship`, `Deck`, `Area`, `Room`, `ShipStructureImport`, import preview/payload types | `ships`, `decks`, `areas`, `rooms`, `ship_structure_imports` |
| `voyage` | `Port`, `Voyage`, `PortCall` | `ports`, `voyages`, `port_calls` |
| `staff` | `Staff`, `StaffLeafe`, `StaffCredential`, `ShiftTemplate`, `CoverageRequirement`, `LaborRule`, `AutonomyPolicy`, `TimeOfDay` | `staff`, `staff_leaves`, `staff_credentials`, `shift_templates`, `coverage_requirements`, `labor_rules`, `autonomy_policies` |
| `partner` | `Partner`, `PartnerContract`, `PartnerStaff`, `PartnerService`, `PartnerSlot` | `partners`, `partner_contracts`, `partner_staff`, `partner_services`, `partner_slots` |
| `booking` | `Booking`, `ServiceBooking` | `bookings`, `service_bookings` |
| `guest` | `Guest`, `BoardingPass` | `guests`, `boarding_passes` |
| `sync` | `HubDevice`, `ShipDevice`, `Revocation`, `Snapshot`, `JobRun`, `OpsEvent`, `ShipSeq`, `ChangeLog` | sync/device/job tables |
| `ai` | `AIParam` | `ai_params` |
| `feedback` | `FeedbackResponse`, `FeedbackInsight` | `feedback_responses`, `feedback_insights` |
| `auth` | `Account`, `AccountIdentity`, `AdminUser`, `AdminShipAccess`, `RefreshToken`, `Identity`, `AdminAuthority` | `accounts`, `account_identities`, `admin_users`, `admin_ship_access`, `refresh_tokens` |
| `audit` | `Entry`, `Record` | `audit_log` |

Notes:

- Existing exported names are retained where practical, including the generated
  `StaffLeafe` spelling, to keep this an architectural move rather than an unrelated rename.
- DTO/view types that already exist in services (for example `guest.View`,
  `sync.HubDeviceView`, `feedback.ResponseView`, and `ai.ParamView`) remain application
  types in their modules. Transport response types remain transport-owned.
- `ownership.Module`, `ownership.Modules`, and `ownership.TableOwner` move unchanged to
  `application/ownership`.

## Field and Value Conversions

The PostgreSQL adapter, not the application, owns all conversion below.

| Current persistence value | Application value | Mapping rule |
|---|---|---|
| `pgtype.Date` | `*time.Time` | `Valid == false` → `nil`; otherwise the date at its stored location. Inputs reverse the mapping. |
| `pgtype.Time` | `staff.TimeOfDay` | `time.Time`/`time.Duration` conversions use `Microseconds`; `Valid` preserved. Handlers format `HH:MM` or `HH:MM:SS` exactly as before. |
| `decimal.NullDecimal` (port latitude/longitude) | `*float64` | `Valid == false` → `nil`; otherwise `Float64()`. Inputs use `decimal.NewFromFloat`. |
| `uuid.NullUUID` | `uuid.NullUUID` | Retained; `google/uuid` is a pure value type, not infrastructure. |
| `json.RawMessage` | `json.RawMessage` | Retained byte-for-byte. |
| `[]byte` | `[]byte` | Retained byte-for-byte. |
| `time.Time`, `*time.Time` | `time.Time`, `*time.Time` | Retained; UTC/location behavior unchanged. |
| `int16`, `int32`, `int64`, `bool`, strings, pointers | Same Go types | Retained. |
| `sqlc.Xxx` row | `module.Xxx` | Field-for-field adapter mapping. |
| `sqlc.XxxParams` | `module.XxxParams` | Field-for-field adapter input. |

Forbidden in application contracts:

- `pgx.Tx`, `pgx.Rows`, `pgxpool.Pool`
- `pgtype.*`
- `pgconn.CommandTag`, `pgconn.PgError`
- `sqlc.*`
- `decimal.NullDecimal`
- `net/http.Request` / `net/http.ResponseWriter`
- AWS SDK, Redis, Firebase, chi, Goose, testcontainers types

## Persistence Ports and Transaction Model

### Plain repository modules

Modules that do not require multiple writes in one use case expose only repository methods
with application types:

```go
type Repository interface {
    Create(context.Context, CreateParams) (Tenant, error)
    Get(context.Context, uuid.UUID) (Tenant, error)
    List(context.Context, ListParams) ([]Tenant, error)
    Update(context.Context, UpdateParams) (Tenant, error)
    SoftDelete(context.Context, uuid.UUID) error
}
```

### Audited/multi-write modules

`ship` import confirmation, `staff`, `booking`, and `guest` expose an application-owned
transaction handle plus non-transactional repository methods:

```go
type Tx interface {
    CreateBooking(context.Context, CreateBookingParams) (Booking, error)
    GetBookingByIdempotency(context.Context, GetBookingByIdempotencyParams) (Booking, error)
    AppendAudit(context.Context, audit.Entry) error
    Commit(context.Context) error
    Rollback(context.Context) error
}

type Repository interface {
    Begin(context.Context) (Tx, error)
    GetBooking(context.Context, uuid.UUID) (Booking, error)
    // ...
}
```

The adapter implements `Tx` with one `pgx.Tx` and `sqlc.New(tx)`. `AppendAudit` writes
through the same transaction. This preserves the current commit/rollback semantics,
including ship import confirmation's terminal commit paths.

### Auth store

`application/auth` uses a high-level transactional `Store` port. It returns application
types and accepts `audit.Entry` values. The adapter opens one transaction for operations such
as guest provisioning, refresh rotation, reuse revocation, and logout-all, and appends audit
inside that transaction.

### Cross-module value use

Consumers declare minimal ports. They may name another module's public application row type
when a method result requires it (`booking.Booking`, `voyage.Voyage`). They must not import
the provider's repository, adapter, SQL, or transport code.

## Audit Model

`audit.Entry` keeps the current fields and semantics:

| Field | Type | Notes |
|---|---|---|
| `OccurredAt` | `time.Time` | Zero means adapter supplies current UTC time, as today. |
| `Source` | `string` | Must be `cloud` or `hub`; validated in application. |
| `ActorType` | `string` | Must be one of `admin`, `account`, `staff`, `hub`, `system`, `ai`. |
| `ActorID` | `*string` | Optional. |
| `Action` | `string` | Required. |
| `Entity` | `*string` | Optional. |
| `EntityID` | `*string` | Optional. |
| `ShipID` | `*uuid.UUID` | Optional. |
| `VoyageID` | `*uuid.UUID` | Optional. |
| `Details` | `json.RawMessage` | Empty becomes `{}` in the adapter, as today. |

`Record` keeps the stored/read shape (`ID int64`, `OccurredAt`, and the same audit fields).
Entry validation moves to `application/audit.Validate`; the PostgreSQL writer calls it before
inserting. A failed audit append still aborts the caller's transaction.

## Data Integrity and Preservation Invariants

- PostgreSQL remains the source of truth; all constraints and triggers remain authoritative.
- UUID v7 generation remains in application services.
- Soft delete remains the only delete path for synced tables; existing SQL is unchanged.
- `change_log` and `ship_seq` remain trigger-owned; application code never writes them.
- Idempotency-key checks and conflict responses remain in application services and
  adapters' existing SQL.
- Status validation and `CHECK` constraint behavior are unchanged.
- Pagination cursors, filter hashes, ordering, and limit behavior are unchanged.
- Event payloads, audit payloads, sync snapshot/delta/outbox shapes, and OpenAPI schemas are
  unchanged.
- Database error translation remains behaviorally identical, including stable codes,
  HTTP statuses, and constraint-to-field messages.

## No Schema Delta

This feature introduces **zero** schema deltas. The migrations, `docs/schema.sql`, sqlc
queries, table ownership, and seed/fixture behavior remain functionally identical.
