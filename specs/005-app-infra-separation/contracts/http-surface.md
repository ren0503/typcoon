# Contract: Preserved HTTP Surface and Transport Ownership

**Feature**: `005-app-infra-separation`
**Date**: 2026-11-12

The refactor must preserve the existing API exactly. Route paths, methods, status codes,
request/response envelopes, validation messages, error codes, pagination, idempotency, and
authentication behavior are unchanged. `openapi/` remains the machine-readable contract.

## Transport Ownership

| Concern | Layer / package |
|---|---|
| Router, server, route mounting | `internal/transport/http` |
| Module handlers, DTOs, validation mapping | `internal/transport/http/handlers/<module>` |
| Technical `/health`, `/ready`, not-found, method-not-allowed | `internal/transport/http/handlers/health` |
| Auth principal extraction and middleware | `internal/transport/http/middleware` |
| JSON/upload helpers | `internal/transport/http/httpx` |
| Business validation, authorization, idempotency, audit decisions | `internal/application/<module>` |
| Database/external calls | `internal/infrastructure/...` through application ports |

Handlers call application service interfaces and map application row types to existing
response DTOs. Handlers must not contain SQL, SDK calls, or business policy.

## Technical Endpoints

| Method | Path | Behavior |
|---|---|---|
| `GET` | `/health` | Liveness, `200 {"status":"ok"}` |
| `GET` | `/ready` | PostgreSQL readiness probe, `200 {"status":"ready"}` or `503 {"status":"not_ready"}` |

## API Prefix

All business and auth endpoints remain under `/api`. `cmd/server` still mounts public auth
routes before the bearer-authentication group and protected routes after it. The global
middleware chain remains request ID, request logging, recovery, CORS, then bearer auth for
protected routes.

## Public Auth Routes (`/api`, no bearer middleware)

| Method | Path |
|---|---|
| `POST` | `/api/auth/firebase/exchange` |
| `POST` | `/api/auth/refresh` |
| `POST` | `/api/auth/logout` |
| `GET` | `/api/auth/jwks` |

## Protected Auth Routes (`/api`, bearer middleware)

| Method | Path |
|---|---|
| `POST` | `/api/auth/logout-all` |
| `POST` | `/api/auth/identities/link` |
| `DELETE` | `/api/auth/identities/{identityId}` |

## Business Routes (`/api`, bearer middleware)

### Tenants

| Method | Path |
|---|---|
| `GET`, `POST` | `/api/tenants` |
| `GET`, `PATCH`, `DELETE` | `/api/tenants/{id}` |

### Ships and Structure Imports

| Method | Path |
|---|---|
| `GET`, `POST` | `/api/ships` |
| `GET` | `/api/ships/structure/import-template` |
| `POST` | `/api/ships/{shipId}/structure/imports` |
| `GET` | `/api/ships/{shipId}/structure/imports/{importId}` |
| `GET` | `/api/ships/{shipId}/structure/imports/{importId}/preview` |
| `POST` | `/api/ships/{shipId}/structure/imports/{importId}/confirm` |
| `GET`, `PATCH`, `DELETE` | `/api/ships/{id}` |
| `GET`, `POST` | `/api/ships/{shipId}/decks` |
| `GET`, `POST` | `/api/ships/{shipId}/areas` |
| `GET`, `POST` | `/api/ships/{shipId}/rooms` |
| `GET`, `PATCH`, `DELETE` | `/api/decks/{id}` |
| `GET`, `PATCH`, `DELETE` | `/api/areas/{id}` |
| `GET`, `PATCH`, `DELETE` | `/api/rooms/{id}` |

### Voyages and Ports

| Method | Path |
|---|---|
| `GET`, `POST` | `/api/ports` |
| `GET`, `PATCH`, `DELETE` | `/api/ports/{id}` |
| `GET`, `POST` | `/api/voyages` |
| `GET`, `PATCH`, `DELETE` | `/api/voyages/{id}` |
| `GET`, `POST` | `/api/voyages/{voyageId}/port-calls` |
| `GET`, `PATCH`, `DELETE` | `/api/port-calls/{id}` |

### Staff

| Method | Path |
|---|---|
| `GET`, `POST` | `/api/staff` |
| `GET`, `PATCH`, `DELETE` | `/api/staff/{id}` |
| `GET`, `POST` | `/api/staff/{staffId}/leaves` |
| `GET` | `/api/staff/{staffId}/credential` |
| `GET`, `PATCH`, `DELETE` | `/api/leaves/{id}` |
| `GET`, `POST` | `/api/ships/{shipId}/shift-templates` |
| `GET`, `PATCH`, `DELETE` | `/api/shift-templates/{id}` |
| `GET`, `POST` | `/api/ships/{shipId}/coverage-requirements` |
| `GET`, `PATCH`, `DELETE` | `/api/coverage-requirements/{id}` |
| `GET`, `POST` | `/api/labor-rules` |
| `GET`, `PATCH`, `DELETE` | `/api/labor-rules/{id}` |
| `GET`, `POST` | `/api/autonomy-policies` |
| `GET`, `PATCH`, `DELETE` | `/api/autonomy-policies/{id}` |

### Partners

| Method | Path |
|---|---|
| `GET`, `POST` | `/api/partners` |
| `GET`, `PATCH`, `DELETE` | `/api/partners/{id}` |
| `GET`, `POST` | `/api/partners/{partnerId}/contracts` |
| `GET`, `POST` | `/api/partners/{partnerId}/staff` |
| `GET`, `POST` | `/api/partners/{partnerId}/services` |
| `GET`, `POST` | `/api/partners/{partnerId}/slots` |
| `GET`, `PATCH`, `DELETE` | `/api/partner-contracts/{id}` |
| `GET`, `PATCH`, `DELETE` | `/api/partner-staff/{id}` |
| `GET`, `PATCH`, `DELETE` | `/api/partner-services/{id}` |
| `GET`, `PATCH`, `DELETE` | `/api/partner-slots/{id}` |

### Bookings

| Method | Path |
|---|---|
| `GET`, `POST` | `/api/bookings` |
| `GET`, `PATCH`, `DELETE` | `/api/bookings/{id}` |
| `GET`, `POST` | `/api/service-bookings` |
| `GET`, `PATCH`, `DELETE` | `/api/service-bookings/{id}` |

### Guests and Boarding Passes

| Method | Path |
|---|---|
| `GET`, `POST` | `/api/bookings/{bookingId}/guests` |
| `GET`, `POST` | `/api/bookings/{bookingId}/boarding-passes` |
| `GET`, `PATCH`, `DELETE` | `/api/guests/{id}` |
| `GET`, `POST` | `/api/boarding-passes/{id}` (detail and revoke) |

## Modules Without HTTP Routes

`ai`, `feedback`, and `sync` expose no public HTTP routes in this phase, matching the current
repository. No transport handler package or route is created for them.

## Authentication and Principal Contract

- `auth.Principal` remains the application-level caller value.
- `transport/http/middleware` owns the HTTP `Provider` interface and the access-token,
  development-header, and deny-all implementations.
- Bearer token parsing, `X-Dev-Principal` handling, and unauthenticated responses remain
  behaviorally identical.
- Access tokens, refresh-token rotation/reuse detection, logout-all, JWKS, Firebase exchange,
  rate-limit responses, and tenant/ship authorization remain unchanged.

## Error and Data Contracts

- Error envelope remains:
  `{"error":{"code":"...","message":"...","request_id":"..."}}`.
- Stable error codes and statuses from `apperrors` remain unchanged.
- Cursor pagination, `Idempotency-Key` handling, upload limits, and multipart behavior remain
  unchanged.
- Response field names, null/omitted behavior, timestamps, date formatting, and time-of-day
  formatting remain unchanged.
- Route-inventory and OpenAPI tests must pass without weakening assertions.
