# Implementation Plan: Go Style Guide Compliance Refactor

**Branch**: `006-go-style-compliance` | **Date**: 2026-11-15 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/006-go-style-compliance/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command; its definition describes the execution workflow.

## Summary

Bring every hand-written Go source file under `cmd/` and `internal/` into compliance with `docs/style.md` (the vendored Uber Go Style Guide) while preserving all runtime behavior, public contracts, HTTP surfaces, database schemas, migrations, and generated artifacts. The approach is a behavior-preserving, batch-by-batch refactor: first establish a rule-coverage baseline and strengthen lint tooling for durable enforcement, then refactor layer by layer (platform and commands, application modules, infrastructure, transport, tests), validating each batch with formatter, vet, lint, unit/contract tests, and build. Retained deviations are recorded in a single compliance report under the feature directory. No architectural change, feature change, or new runtime dependency is introduced.

## Technical Context

**Language/Version**: Go 1.26.0 (module `github.com/ren0503/tycoon`; local toolchain Go 1.26.8)

**Primary Dependencies**: go-chi/chi v5, jackc/pgx v5, sqlc-generated persistence, pressly/goose, redis/go-redis v9, Firebase Admin Go v4, AWS SDK for Go v2, getkin/kin-openapi, go-playground/validator, golang-jwt/jwt v5. Development tooling: golangci-lint v2.14.

**Storage**: PostgreSQL (cloud source of truth) and Redis/Valkey. No schema, migration, trigger, or query changes.

**Testing**: `go test ./...` for unit and OpenAPI spec lint; `go test -tags=integration ./...` for Testcontainers-backed PostgreSQL integration suites; miniredis for Redis tests; route-inventory and contract tests under `internal/transport/http` and `internal/app`.

**Target Platform**: Linux server (AWS EC2, Docker), Go modular monolith.

**Project Type**: web-service backend (modular monolith with application, infrastructure, platform, and transport layers).

**Performance Goals**: No performance regression and no new performance targets. Style-driven performance changes are limited to behavior-neutral improvements required by `docs/style.md` (for example, `strconv` over `fmt` or capacity hints) and must be justified by clarity or measured equivalence.

**Constraints**: Behavior-preserving only; no changes to business logic, public Go contracts, HTTP contracts, database schema, migrations, triggers, event payloads, or authentication/synchronization behavior; generated persistence code and migration artifacts must remain byte-identical; no new runtime dependency; integration tests require a container runtime and may be compile-checked when unavailable.

**Scale/Scope**: 203 Go files (141 non-test, 62 test), approximately 36.6k lines, across `cmd/` and `internal/{app,application,infrastructure,platform,transport}`. 54 Go packages total.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status | Notes |
|-----------|------|--------|-------|
| I. Modular Monolith | Preserve module boundaries; modules communicate through interfaces, not internal packages or shared mutable state | PASS | Refactor is file/package-local. Cross-package deduplication is limited to existing `internal/platform` packages; no new cross-module dependencies. |
| II. Database Integrity | No ad-hoc schema or migration edits | PASS | Schema, migrations, triggers, and generated persistence code are out of scope and must remain byte-identical. |
| III. Offline-First Architecture | Preserve sync, idempotency, epoch, and change-log behavior | PASS | No sync or transport behavior changes; existing synchronization tests must pass unchanged. |
| IV. Data Ownership | Preserve write-owner and ownership-status behavior | PASS | No ownership logic changes; `ownership` package data remains behavior-equivalent. |
| V. AWS Readiness | Keep AWS behind project interfaces; no new runtime dependency | PASS | Only development-time lint tooling may be added; runtime module graph is unchanged. |
| VI. Security | Preserve authentication, authorization, encryption, and secret handling | PASS | No security logic changes; auth, token, rate-limit, and secrecy tests must pass unchanged. |
| VII. Reliability | Preserve timeouts, graceful shutdown, idempotency, and goroutine lifecycle | PASS | Concurrency style changes are allowed only when lifecycle and timeout behavior is provably unchanged. |
| VIII. Testability | Tests must remain runnable and must not be weakened | PASS | Tests are refactored for style only; no assertions or coverage are removed. |
| IX. Specification-Driven Development | Work traces to spec, plan, and tasks | PASS | Spec and plan exist; a compliance report records justified deviations and every non-trivial change traces to a style rule. |
| X. Simplicity | No unjustified architectural complexity; YAGNI | PASS | No new abstractions solely for style. Lint tooling additions are development-only, rule-specific, and justified under FR-021. |

**Gate result**: PASS. No violations require an ADR. Complexity Tracking is empty except for the development-tooling justification recorded below.

### Post-Design Re-check

After Phase 1 design, the same gates were re-evaluated against `research.md`, `data-model.md`, `contracts/`, and `quickstart.md`. All remain PASS:

- The design adds no source packages, module dependencies, schema changes, or runtime behavior changes.
- The compliance report and coverage matrix are documentation/review artifacts under the feature directory.
- The validation strategy preserves testability and makes each batch independently verifiable.
- Development-only lint tooling remains the only addition and is justified under Principle X and FR-021.

## Project Structure

### Documentation (this feature)

```text
specs/006-go-style-compliance/
├── plan.md                      # This file (/speckit.plan command output)
├── research.md                  # Phase 0 output (/speckit.plan command)
├── data-model.md                # Phase 1 output (/speckit.plan command)
├── quickstart.md                # Phase 1 output (/speckit.plan command)
├── contracts/                   # Phase 1 output (/speckit.plan command)
│   ├── behavior-preservation.md
│   ├── compliance-report-contract.md
│   └── style-rule-coverage.md
├── compliance-report.md         # Refactor artifact: justified exceptions (FR-018)
├── checklists/
│   └── requirements.md
└── tasks.md                     # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
├── migrate/                     # Migration command entrypoint
└── server/                      # API server entrypoint

internal/
├── app/                         # Composition root and runtime wiring
├── application/                 # Business modules (ai, audit, auth, booking, feedback, guest,
│                                # ownership, partner, ship, staff, sync, tenant, voyage)
├── infrastructure/              # Adapters (aws, cache/redis, health, identity/firebase,
│                                # observability, persistence/postgres, security)
├── platform/                    # Shared utilities (config, errors, pagination, testutil)
└── transport/http/              # HTTP transport (handlers, middleware, router, openapi tests)

docs/
├── style.md                     # Authoritative style guide (generated; DO NOT EDIT)
├── schema.sql
└── system-architecture.md

db/
├── migrations/                  # Versioned migrations (out of scope)
└── queries/                     # SQLC query sources (out of scope)

openapi/                         # OpenAPI contract (must remain unchanged)
```

**Structure Decision**: Keep the existing modular-monolith layout. This feature adds no source directories or packages. All source edits occur in place under `cmd/` and `internal/`; new files are limited to feature documentation and the compliance report under `specs/006-go-style-compliance/`.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| Development-only lint tooling additions (no runtime dependency) | FR-021 requires durable enforcement of style rules not covered by the current lint configuration; manual review alone drifts. | Source-only cleanup without tooling leaves the repository compliant only until the next change and cannot enforce suppression justification (FR-022). |
