# Implementation Plan: Firebase SSO with Backend-Owned Session Tokens

**Branch**: `003-firebase-sso-auth` | **Date**: 2026-11-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/003-firebase-sso-auth/spec.md`

## Summary

Replace direct guest OAuth for cloud clients with Firebase Authentication as the centralized identity provider, while keeping the Go backend as the sole issuer and authority for application sessions. Clients exchange a Firebase ID Token at `POST /v1/auth/firebase/exchange`; the backend verifies it with the Firebase Admin SDK, resolves the stable internal account through `accounts`/`account_identities`, and returns a short-lived RS256 access JWT plus an opaque rotating refresh token. Guest Mobile and Fleet Console receive separate, independently revocable sessions for the same internal account; Fleet Console authority is always resolved from a backend-owned `admin_users` grant linked to that account, never from Firebase claims. Refresh rotation and reuse detection are transactionally enforced in PostgreSQL, rate limits are shared through Redis in deployed environments, and signing keys come from AWS KMS with a public JWKS endpoint and documented rotation overlap. The Hub remains fully outside this authentication path.

Two specification conflicts are resolved explicitly in [research.md](./research.md): an unknown guest identity whose verified email already belongs to an account is **not** auto-provisioned (generic `409 identity_link_required`, then explicit authenticated linking per US5); and Fleet Console exchanges never auto-provision administrator access.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/ren0503/tycoon`)

**Primary Dependencies**: Firebase Admin SDK (`firebase.google.com/go/v4`), `github.com/golang-jwt/jwt/v5` (RS256 validation), `github.com/redis/go-redis/v9` (shared rate limits), AWS SDK for Go v2 KMS client (asymmetric signing), existing chi v5, pgx v5, sqlc, Goose, `google/uuid`, `go-playground/validator`, and OpenAPI tooling.

**Storage**: PostgreSQL 15+ for accounts, identities, refresh tokens, administrator grants, and audit rows. Redis/Valkey for shared authentication rate-limit counters in every environment, including development and test.

**Testing**: `go test ./...` (unit, config, contract spec lint) plus `go test -tags=integration ./...` (testcontainers PostgreSQL, `miniredis`, fakes for Firebase/KMS). No production Firebase, Google, Apple, KMS, or email credentials.

**Target Platform**: Linux server/containers (EC2 + Auto Scaling); local Docker Compose with PostgreSQL and Redis.

**Project Type**: Go modular-monolith REST web service; backend only. Guest Mobile and Fleet Console source code lives in separate repositories.

**Performance Goals**: Local access-token verification under 10 ms excluding database; exchange under 500 ms p95 excluding Firebase network latency; refresh rotation bounded by one PostgreSQL transaction; rate-limit checks under 10 ms p95 with Redis in-region. No unbounded external calls.

**Constraints**: Access-token lifetime must not exceed 15 minutes (default 15); refresh lifetime default 30 days; clock skew default 60 seconds; rate limits must not be weaker than the clarified defaults; all external calls have bounded timeouts; no token, hash, Firebase token, or signing secret in logs; production must fail fast on missing Firebase/KMS/Redis configuration and must reject emulator/local-key modes.

**Scale/Scope**: Seven authentication operations plus JWKS, one stable account per Firebase identity, multiple independent device/client sessions per account, both Guest Mobile and Fleet Console clients, existing 41-table baseline extended by one minimal migration. No client-app implementation, no administrator-management API, no Hub changes.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Post-design re-check (after research/data-model/contracts/quickstart): PASS — no violations, no Complexity Tracking entries.** The auth module remains inside the modular monolith, all schema changes are versioned and update the baseline, PostgreSQL remains the final authority for token/account state, Redis is only an optimization/abuse-control store already present in the approved stack, and no Hub dependency is introduced.

| Principle | Gate question | Status | Evidence/Notes |
|---|---|---|---|
| I. Modular Monolith | One package per module, public interfaces, acyclic dependencies, composition-root wiring | PASS | `internal/auth` owns auth tables and exposes `auth.Service`; Firebase/KMS/Redis/audit are injected interfaces; `internal/app` composes. No new module or service. |
| II. Database Integrity | Versioned migration, baseline updated, text+CHECK statuses, constraints authoritative | PASS | Migration `00003` extends `account_identities` provider, adds `admin_users.account_id`, and adds `refresh_tokens.client`; unique indexes and row locks enforce identity/session invariants. `docs/schema.sql` and `db/schema.sql` update together. |
| III. Offline-First | Hub remains independent; no cloud auth dependency | PASS | No Hub code changes; Firebase/JWT/KMS packages are not imported by sync code; no auth tokens enter snapshot/change-log payloads. |
| IV. Data Ownership | Auth module owns identity/session rows; no dual writes | PASS | All writes to `accounts`, `account_identities`, `refresh_tokens`, `admin_users`, and `admin_ship_access` stay in `internal/auth`; other modules use interfaces only. |
| V. AWS Readiness | AWS behind interfaces; local dev without AWS; fail-fast config | PASS | `Signer` interface with KMS and local implementations; local key/emulator only outside production; startup validates required config. |
| VI. Security | Short access tokens, hashed rotating refresh tokens, reuse revocation, least privilege, no secrets in logs | PASS | RS256 JWT with `kid`; SHA-256 refresh hashes; family revocation under row lock; backend-owned tenant/ship authorization; request/response redaction tests. |
| VII. Reliability | Bounded external calls, idempotency, graceful behavior | PASS | Firebase/KMS/Redis/DB timeouts; logout operations idempotent; rotation atomic; limiter/signing failures fail closed. |
| VIII. Testability | Business logic testable with fakes; real DB constraints covered; new tests fail without change | PASS | Verifier/signer/limiter/clock fakes plus real PostgreSQL concurrency tests; config and contract tests; no live cloud credentials. |
| IX. Specification-Driven | Spec, plan, tasks, docs, ADR updated | PASS | This plan, research, data model, contracts, quickstart; implementation updates architecture/schema/OpenAPI/env docs and records ADR-10. |
| X. Simplicity | No unnecessary distributed components; YAGNI; complexity justified | PASS | Redis is the architecture's existing rate-limit/cache store; no new service, broker, or microservice. New dependencies are narrowly justified in research R1/R5/R8. |

## Project Structure

### Documentation (this feature)

```text
specs/003-firebase-sso-auth/
├── plan.md                       # This file
├── research.md                   # Phase 0 decisions
├── data-model.md                 # Phase 1 entities, invariants, migration
├── quickstart.md                 # Phase 1 validation guide
├── contracts/
│   ├── auth-api.yaml             # Authentication/identity OpenAPI fragment
│   └── client-integration.md     # Guest Mobile + Fleet Console integration contract
├── checklists/
│   └── requirements.md           # 16/16 requirement-quality checks
└── tasks.md                      # Phase 2 output (/speckit.tasks) — not created here
```

### Source Code (repository root)

```text
db/
├── schema.sql                                   # sqlc input; updated to match migration
├── migrations/
│   ├── 00002_business_schema.sql                # existing baseline
│   └── 00003_firebase_sso_auth.sql              # new minimal migration
└── queries/
    └── auth.sql                                 # extend with account/identity/session/admin-grant queries

internal/
├── app/
│   └── modules.go                               # compose auth service, public/protected route registration
├── auth/
│   ├── module.go                                # public Service interface + dependencies
│   ├── accounts.go                              # existing AccountService
│   ├── principal.go                             # existing Principal
│   ├── provider.go                              # existing dev/deny providers + JWT access-token provider
│   ├── service.go                               # exchange, refresh, logout, linking orchestration
│   ├── firebase.go                              # Firebase Admin SDK verifier + fake-compatible interface
│   ├── tokens.go                                # access-token issuance/validation, claims, clock skew
│   ├── signing.go                               # KMS/local crypto.Signer + JWKS key set
│   ├── refresh.go                               # opaque token generation, hashing, atomic rotation
│   ├── ratelimit.go                             # limiter interface, Redis and memory implementations
│   ├── store.go                                 # transactional SQL/sqlc persistence
│   ├── handlers.go                              # public and protected chi handlers
│   └── dto.go                                   # request/response types and validation
├── config/
│   └── config.go                                # auth/Firebase/signing/token/rate-limit config + fail-fast rules
├── httpserver/
│   ├── router.go                                # PublicAPIRoutes + protected /v1 tree
│   └── middleware/auth.go                       # bearer access-token provider integration
├── apperrors/errors.go                          # rate_limited/auth_unavailable codes
└── testutil/                                    # PostgreSQL harness; auth test fakes/helpers

cmd/server/main.go                                # build verifier/signer/limiter, inject auth deps
openapi/openapi.yaml                              # merge auth paths/components
docs/system-architecture.md                       # auth architecture, Redis limiter, Hub boundary, ADR-10
docs/schema.sql                                   # baseline documentation update
.env.example                                      # documented auth configuration
docker-compose.yml                                # Redis service for local auth rate limiting
```

**Structure Decision**: Extend the existing single-binary Go modular monolith. Authentication remains a single `internal/auth` module behind a public `Service` interface; HTTP handlers stay in the owning module, while `internal/httpserver` only mounts them and applies global middleware. The router is split into public auth routes and the existing authenticated `/v1` tree so exchange/refresh/logout/JWKS can be reached without a backend token.

## Implementation Strategy

1. **Schema and configuration first**: add migration `00003`, update both schema baselines, extend sqlc queries, and add fail-fast auth configuration with development-only local modes.
2. **Token and identity primitives**: implement the Firebase verifier interface, access-token signer/validator, JWKS, refresh generation/hashing, and rate limiter adapters with unit tests.
3. **Service orchestration**: implement exchange with race-safe provisioning/email-collision handling, client-scoped sessions, refresh rotation/reuse detection, logout/logout-all, link/unlink with last-credential protection, and transaction-aware audit.
4. **HTTP and wiring**: add public/protected handlers, JWT principal provider, router `PublicAPIRoutes`, `app.BuildModules` auth dependency injection, and update stale tests that call the old signature.
5. **Contract and documentation**: merge the auth OpenAPI paths, write the client integration contract, update architecture/schema/env docs, and add ADR-10.
6. **Verification**: run unit/config/boundary tests, testcontainers concurrency/constraint suites, contract tests, lint/vet/build, and schema-parity checks.

## Risks and Mitigations

- **Email-collision behavior is a deliberate FR-016 exception.** It is required to make US5 linking possible; the API uses a generic message and never merges accounts.
- **Existing administrator rows are unlinked.** The migration preserves them with `account_id = NULL`; deployment must explicitly link administrators. No email-based backfill is allowed.
- **KMS is unavailable locally.** Tests and local development use an in-memory/PEM RSA signer; production validation rejects that mode.
- **Redis is required in deployed environments.** Development/test may use the in-memory limiter; staging/production startup fails without Redis, and limiter errors fail closed.
- **The Hub is not in this repository.** Backend-side boundary tests prove no Hub dependency; the actual offline Hub suite remains in the Hub repository per the clarified scope.

## Complexity Tracking

No constitution violations. The new Firebase, JWT, Redis, and KMS dependencies and the Redis-backed limiter are justified in [research.md](./research.md); they use the architecture's approved stack and do not alter the modular-monolith or data-ownership model.
