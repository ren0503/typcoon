# Module Interface Contracts: 002-business-modules

The Go-implementation contract for FR-005/006/007/008. Every module is one package `internal/<module>/` exposing exactly the surface below; everything else in the package is implementation. Cross-module code imports only these declarations, and `internal/app` is the sole composer.

## Shared infra (not business modules)

- `internal/database` — pool, Goose migration harness (`Provider` + `embed.FS`), generated `sqlc` package, `pgerrors` mapper (`*pgconn.PgError` → `*apperrors.Error`).
- `internal/apperrors` — `Error{Code, Message, HTTPStatus}`, `As/Wrap/Internal`.
- `internal/httpserver` — router bootstrap, middleware (RequestId, RequestLogger, Recoverer, CORS, `Authenticate`), mounts each exposed module's `RegisterRoutes`.
- `internal/config` — verbose fail-fast loading; `auth.mode` (`dev`|`jwt`), `auth.dev.principals` allowlist; startup error when `env=production` and `auth.mode=dev`.
- `internal/auth` (module, internal) — `PrincipalProvider` interface; `Principal{TenantID, ShipIDs, Role/ActorType, ActorID}` with methods `CanAccessTenant(id)`, `CanAccessShip(id)`; `DevHeaderProvider` implementation. Exposes it to other modules for audit actor attribution.

## Per-module public interfaces

| Module | Public type (interface) | Cross-module consumers |
|---|---|---|
| `tenant` | `TenantService` — CRUD + `GetActive(id, p)` scope-checked | `ship`, `voyage` (tenant validation), others resolve tenant via principal |
| `ship` | `ShipService` — ships CRUD + `Exists(shipID) (bool, error)`; deck/area/room services internal to HTTP surface | `partner`, `booking`, `voyage`, `staff` (parent + scope checks) |
| `voyage` | `VoyageService` — voyages/port-calls CRUD + ports; `GetVoyage(id)` for scope attribution | `booking` (parent + scope), `sync` (voyage scoping later) |
| `staff` | `StaffService` — CRUD; `StaffExists(id)`; credential ops (hash-only) | `booking`,`guest`,`partner` (none today; reserved), `audit` emits |
| `partner` | `PartnerService` — CRUD; `PartnerExists(id)` | `booking` (service validation) |
| `booking` | `BookingService` — CRUD + idempotency + `BookingExists(id)` | `guest` (parent resolution) |
| `guest` | `GuestService` — CRUD + boarding-pass ops; `GuestExists(id)` | booking (reference), later workflows |
| `auth` | `PrincipalProvider`, `AccountExists(accountID)` | `guest` (account parent), principal resolution |
| `sync` | No public business interface this phase (device/epoch ops internal; later sync feature formalizes) | — |
| `audit` | `Appender` — `AppendTx(ctx, tx, Entry) error` + query API | `guest`, `booking`, `staff` services (same-tx emission, FR-057) |
| `ai` | `AIParamService` — metadata CRUD | none (reserved for later features) |
| `feedback` | `FeedbackService` — submit/get/list | none |

## Interface rules (binding)

1. **Only the declarations above are importable cross-module.** depguard + boundary test enforce: importing `<module>/repo.go`, query files, or unexported types from a sibling module fails CI (FR-006).
2. **Dependency direction is acyclic** as drawn in `data-model.md`; the boundary test fails on any cycle (FR-007).
3. **Errors cross interfaces only as `*apperrors.Error`-compatible values** — repo/pg errors are mapped at the owning module boundary (FR-025/033).
4. **Context propagation**: every interface method takes `ctx context.Context` (FR-023); transaction scope is carried by the repository, never passed across service interfaces except into `audit.Appender.AppendTx` which receives the owning transaction explicitly.
5. **Request/response DTOs are owned by the module**; no shared generic model types between modules (FR-020 — no cross-module conversion layers).
6. **HTTP handlers live inside the owning module** and are registered by `internal/app` via the module's `RegisterRoutes(chi.Router)`; `auth`/`sync`/`audit`/`ai`/`feedback` register none (FR-044/045).
7. **New tables later** must be assigned an owner module and appear in the ownership check or CI fails (User Story 1 / SC-001).
