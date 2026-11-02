# Quickstart: 002-business-modules

Validation guide proving the business-layer foundation works end-to-end. Implementation details live in `tasks.md`; model/entity detail in [data-model.md](./data-model.md); API shape in [contracts/openapi.yaml](./contracts/openapi.yaml).

## Prerequisites

- Go 1.26 (`go version`)
- Docker running (testcontainers + docker-compose Postgres)
- No AWS credentials required (FR-055/TC-007)

## Setup & run

```sh
make sqlc                       # regenerate models from db/schema.sql + db/queries (zero errors, SC-003)
docker compose up -d postgres   # local database
cp .env.example .env            # if present; ensure AUTH_MODE=dev + DEV_PRINCIPALS non-empty
make migrate-up                 # applies migrations to an empty DB (migration 00001 + 00002)
make run                        # serve API; app fails fast if required config missing (V/VI gates)
```

Expected: server boots, logs a `dev-principal` marker line, `/v1` routes return 401 without principal headers, and the dev principal allowlist works with the configured dev header.

## Commands under test

```sh
make fmt vet lint test          # unit + boundary + config guard tests
make test-integration           # testcontainers Postgres: migration, constraints, CRUD, pagination
curl -s localhost:8080/healthz  # unchanged foundation health route
```

## Validation scenarios (traceable to spec)

1. **Ownership completeness (US1, SC-001)** — run the boundary/ownership test in `make test`; it fails if any of the 41 tables has zero/multiple owners or a module imports another's internals. Expect: PASS.
2. **Schema fidelity (US2, SC-004)** — `make test-integration` applies migrations to an empty DB and asserts table/column/index/trigger parity. Re-running migrations is harmless (FR-053). Expect: PASS with 41 tables.
3. **Basic CRUD + soft delete (US3)** — via dev principal: create tenant → ship → deck → room; update a room's capacity; soft-delete it. Expect: room disappears from list; GET by id returns 404; re-creating the same room number succeeds (partial unique index ignores deleted rows) except for the three noted non-partial unique tables (`voyages code`, `ships.imo`, `ports.un_locode`) where reuse returns 409.
4. **Parent reference rule (Clarification 2026-11-02)** — create an area referencing a soft-deleted deck → expect 422 `invalid_reference`/`not_found`, no row written (FR-032).
5. **API foundation safety (US4)** —
   - Unauthenticated GET `/v1/ships` → 401 `unauthorized`, same body shape for all protected routes (FR-043).
   - Principal scoped to ship A lists bookings of voyage in ship B → 403/404-deny (SC-008).
   - `GET /v1/guests/{id}` shows `has_document`/`has_emergency_contact` booleans but never ciphertext or plaintext (FR-046).
   - Create booking with `Idempotency-Key: k1` → 201; repeat same body/key → same booking, no duplicate; same key different body → 409 `idempotency_conflict` (FR-042).
   - Direct URL guessing of `change_log`/`refresh_tokens`/`audit_log`… → 401/404; no CRUD routes for internal tables (FR-044).
6. **Constraint behavior (US5, SC-007)** — integration tests assert: voyage overlap → 409 (EXCLUDE constraint), partner slot approved-overlap → 409, room booking overlap → 409, `days_of_week` value 9 → 422, malformed UUID → 400, `certifications: {}` (non-array) → 422, transaction rollback on chained failure, cursor pagination walk has no dup/skip pages, cursor from another filter → 400 `invalid_cursor`.
7. **Audit emission (FR-057)** — create a staff member with `auditable` path; assert `audit_log` contains the append within the same transaction (missing audit ⇒ no staff row, tested via forced failure injection).
8. **Config fail-fast (II/V)** — `AUTH_MODE=dev` + `ENV=production` at boot → process exits with config error before serving (cannot activate dev principal in production).

## Definition of done signals

- `make sqlc` idempotent, zero errors.
- Both test suites green locally **and** in CI with the same commands (FR-052).
- No advanced workflows implemented (SC-010) — reviewing diffs confirms none of FR-036's exclusions appear.
