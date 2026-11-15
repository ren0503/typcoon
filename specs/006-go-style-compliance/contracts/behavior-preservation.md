# Contract: Behavior Preservation

**Feature**: `006-go-style-compliance`
**Purpose**: Define the invariants and validation evidence that prove the refactor changes style only (FR-015, FR-019, FR-020; SC-002 through SC-010).

## Invariants

The refactor MUST NOT change any of the following:

| Area | Invariant | Evidence |
|------|-----------|----------|
| Business logic | Same decisions, calculations, ordering, and branch outcomes | Existing unit tests pass unchanged |
| Public Go API | Exported types, interfaces, functions, constants, and method sets remain source-compatible | `go build ./...` plus package consumers compile |
| HTTP surface | Same routes, methods, status codes, request shapes, response envelopes, error codes | Route inventory and HTTP contract tests pass unchanged |
| Authentication | Same token issuance, validation, refresh rotation, revocation, and rate limiting | Auth unit and integration tests pass unchanged |
| Authorization | Same tenant and ship-scope enforcement | Security integration tests pass unchanged |
| Sync behavior | Same idempotency, epoch, change-log, snapshot, and delta semantics | Sync and idempotency integration tests pass unchanged |
| Audit and redaction | Same audit events and sensitive-field redaction | Audit and redaction tests pass unchanged |
| Database | Same schema, migrations, triggers, constraints, and query behavior | Migration, trigger, and parity tests pass unchanged |
| Generated artifacts | Byte-identical generated SQLC files and migration artifacts | Hash comparison against baseline |
| Error messages | Same externally observable validation and error messages | Contract and handler tests pass unchanged |
| Time and concurrency | Same timeouts, cancellation, goroutine lifetime, and shutdown behavior | Race-enabled tests where available and integration tests |

## Forbidden Changes

- Editing `docs/style.md` (generated; source of truth).
- Editing generated SQLC code, migrations, `docs/schema.sql`, or `openapi/**` by hand.
- Renaming or removing exported identifiers.
- Changing database schema, constraints, triggers, or migration order.
- Adding a runtime dependency.
- Changing route registration, middleware order, or handler behavior.
- Changing test assertions, deleting tests, or skipping tests to make a batch pass.
- Adding `//nolint` without a specific linter and a reason.
- Introducing a new abstraction solely to satisfy a style preference.

## Baseline Capture

Before the first source edit, capture and store in the feature directory (or plan notes):

1. Current commit: `git rev-parse HEAD`.
2. `git status --short` (must be clean except for spec/plan artifacts).
3. `go fmt ./...` result.
4. `go vet ./...` result.
5. `golangci-lint run ./...` result.
6. `go test ./...` result.
7. `go build -o bin/server ./cmd/server` and `go build -o bin/migrate ./cmd/migrate`.
8. `go test -tags=integration ./...` when Docker is available; otherwise `go test -tags=integration -run '^$' ./...`.
9. Hashes of immutable artifacts:
   - `internal/infrastructure/persistence/postgres/sqlc/*.go`
   - `db/migrations/**`
   - `docs/schema.sql`
   - `openapi/**`

Record any pre-existing failure. A pre-existing failure is not caused by the refactor, but it MUST NOT be silently ignored; it is either fixed as a separate concern with approval or documented as a baseline caveat.

## Batch Validation Gate

Each batch MUST run, in order:

| Gate | Command | Pass Condition |
|------|---------|----------------|
| Format | `gofmt -l ./cmd ./internal` (and `golangci-lint fmt --diff ./...` once configured) | No files listed / no diff |
| Vet | `go vet ./...` | No findings |
| Lint | `golangci-lint run ./...` | Zero findings beyond recorded exceptions |
| Unit + contract | `go test ./...` | All pass |
| Build | `go build ./cmd/server ./cmd/migrate` | Success |
| Integration | `go test -tags=integration ./...` | All pass when Docker is available; compile-only otherwise |

Additionally, after each batch:

- Confirm `git diff --stat` touches only batch paths and the feature directory.
- Confirm no generated artifact changed (`git status --short` on immutable paths).
- Append the batch result to the compliance report / batch notes.
- If any gate fails, the batch is `blocked`; fix and rerun. A failure MUST NOT be waived to proceed.

## Conflict Protocol

When a style rule conflicts with behavior preservation:

1. Keep the behavior.
2. Record the location, rule, reason, risk, and mitigation in the compliance report.
3. Add an inline comment only when the deviation is locally surprising.
4. Do not rename or remove exported identifiers to satisfy a rule.
5. Do not edit generated output; change the source and regenerate, or record the conflict if regeneration is out of scope.
6. If the conflict reveals a spec ambiguity, update the spec/plan rather than making an undocumented decision.

## Final Acceptance

The feature is accepted only when:

- Every batch gate passes.
- The final full validation run passes with the same or better results than baseline.
- The route inventory, HTTP contract checks, and schema/migration parity checks are unchanged.
- Immutable artifact hashes match the baseline.
- No `open` compliance findings remain.
- The final diff contains only style refactors, dead-code removal, tooling configuration, and feature documentation artifacts.
