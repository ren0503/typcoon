# Tasks: Go Style Guide Compliance Refactor

**Input**: Design documents from `/specs/006-go-style-compliance/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: No new tests are generated. This is a behavior-preserving refactor; existing tests are preserved and may be refactored for style only. Test assertions, coverage, and test names must not be weakened.

**Organization**: Tasks are grouped by user story so each story remains independently verifiable. Because US1/US3/US4 edit overlapping source files, they run sequentially in priority order; verification-only tasks within US2 may run in parallel.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1–US5 map to the user stories in `spec.md`
- Every task names the exact repository paths it touches

## Path Conventions

Repository root: `/home/ren0503/tycoon/server`

- Commands: `cmd/server/`, `cmd/migrate/`
- Composition root: `internal/app/`
- Business modules: `internal/application/`
- Adapters: `internal/infrastructure/`
- Shared utilities: `internal/platform/`
- HTTP transport: `internal/transport/http/`
- Feature artifacts: `specs/006-go-style-compliance/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the behavioral baseline, immutable-artifact snapshot, compliance artifacts, and development-only style tooling.

- [X] T001 Capture the pre-refactor baseline in `specs/006-go-style-compliance/baseline.md`: current commit, `git status --short`, `go fmt ./...`, `go vet ./...`, `golangci-lint run ./...`, `go test ./...`, `go build -o bin/server ./cmd/server`, `go build -o bin/migrate ./cmd/migrate`, and `go test -tags=integration ./...` (or `go test -tags=integration -run '^$' ./...` when Docker is unavailable).
- [X] T002 Record SHA-256 hashes for `internal/infrastructure/persistence/postgres/sqlc/*.go`, `db/migrations/*`, `docs/schema.sql`, and `openapi/*` in `specs/006-go-style-compliance/baseline.md`.
- [X] T003 Create the compliance report skeleton at `specs/006-go-style-compliance/compliance-report.md` following `contracts/compliance-report-contract.md`, with empty exception and evidence tables.
- [X] T004 [P] Configure development-only formatter and linters in `.golangci.yml`: add `goimports` formatter and enable `revive`, `gocritic`, `misspell`, `prealloc`, `unparam`, `funcorder`, and `nolintlint` while keeping existing default linters and `depguard`.
- [X] T005 [P] Add a non-destructive formatter check target (for example `fmt-check` using `gofmt -l` and `golangci-lint fmt --diff`) and ensure the `lint` target runs the updated configuration in `Makefile`.
- [X] T006 Run the updated checks across `./...`, record initial finding counts by linter, and tune systematic false positives; document each tuning decision in `specs/006-go-style-compliance/research.md` and `specs/006-go-style-compliance/compliance-report.md`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Freeze the rule-coverage scope and behavior contracts that every story depends on.

**⚠️ CRITICAL**: No user story work begins until this phase is complete.

- [X] T007 Initialize the Rule Coverage Summary in `specs/006-go-style-compliance/compliance-report.md` from `contracts/style-rule-coverage.md`, marking every row `clear`, `partial`, or `missing` for the current tree.
- [X] T008 Freeze behavior-contract evidence in `specs/006-go-style-compliance/baseline.md`: route inventory output, HTTP contract test result, schema/migration parity result, trigger result, and generated-artifact hash comparison.
- [X] T009 Confirm the batch path map and exclusions in `contracts/style-rule-coverage.md`: `internal/infrastructure/persistence/postgres/sqlc/`, `db/migrations/`, `docs/schema.sql`, and `openapi/` are immutable and excluded from manual style edits.
- [X] T010 Dry-run the batch validation sequence from `contracts/behavior-preservation.md` once against the untouched tree and record the commands and outputs in `specs/006-go-style-compliance/baseline.md`.

**Checkpoint**: Baseline, tooling, coverage scope, and validation sequence are fixed. User stories may begin.

---

## Phase 3: User Story 1 - Reviewers Rely on a Single Enforced Style Baseline (Priority: P1) 🎯 MVP

**Goal**: Apply every applicable rule in `docs/style.md` to all hand-written source under `cmd/` and `internal/`, fixing violations or recording justified exceptions.

**Independent Test**: Sample files from every layer and verify each applicable rule is satisfied or recorded in `specs/006-go-style-compliance/compliance-report.md`; run the batch validation gate with zero unexplained findings.

### Implementation for User Story 1

- [X] T011 [P] [US1] Refactor `internal/platform/config/config.go`, `internal/platform/config/config_test.go`, `internal/platform/errors/errors.go`, `internal/platform/errors/errors_test.go`, and `internal/platform/pagination/*.go` for naming, declaration grouping, error wrapping/naming, nesting, struct initialization, comments, and long lines.
- [X] T012 [P] [US1] Refactor `cmd/server/main.go`, `cmd/migrate/main.go`, `internal/app/api.go`, `internal/app/app.go`, `internal/app/migrate.go`, and `internal/app/modules.go` for `Exit in Main`, error handling, function ordering, declaration grouping, and comments; preserve runtime wiring and startup behavior.
- [X] T013 [P] [US1] Refactor `internal/application/ai/*.go` and `internal/application/audit/*.go` (non-test and test files) against the applicable rows of `contracts/style-rule-coverage.md`.
- [X] T014 [P] [US1] Refactor `internal/application/auth/*.go` (non-test and test files) against the applicable rows; preserve token, refresh, rate-limit, and audit behavior.
- [X] T015 [P] [US1] Refactor `internal/application/booking/*.go` (non-test and test files) against the applicable rows; preserve idempotency and status-transition behavior.
- [X] T016 [P] [US1] Refactor `internal/application/feedback/*.go` and `internal/application/guest/*.go` (non-test and test files) against the applicable rows; preserve guest-document secrecy behavior.
- [X] T017 [P] [US1] Refactor `internal/application/ownership/*.go` against the applicable rows while preserving `Module`, `Modules`, and `TableOwner` contracts; do not remove exported identifiers.
- [X] T018 [P] [US1] Refactor `internal/application/partner/*.go` (non-test and test files) against the applicable rows.
- [X] T019 [P] [US1] Refactor `internal/application/ship/*.go` (non-test and test files) against the applicable rows; preserve import-validation and template behavior.
- [X] T020 [P] [US1] Refactor `internal/application/staff/*.go` (non-test and test files) against the applicable rows; preserve attendance, PIN, and time behavior.
- [X] T021 [P] [US1] Refactor `internal/application/sync/*.go`, `internal/application/tenant/*.go`, and `internal/application/voyage/*.go` (non-test and test files) against the applicable rows; preserve sync and voyage behavior.
- [X] T022 [P] [US1] Refactor `internal/infrastructure/aws/*.go`, `internal/infrastructure/cache/redis/*.go`, `internal/infrastructure/health/*.go`, `internal/infrastructure/identity/firebase/*.go`, `internal/infrastructure/observability/logging/*.go`, and `internal/infrastructure/security/**/*.go` (non-test and test files) against the applicable rows.
- [X] T023 [P] [US1] Refactor non-generated files under `internal/infrastructure/persistence/postgres/` — `audit.go`, `migrate.go`, `pgerrors.go`, `postgres.go`, `convert/convert.go`, `auth/store.go`, `auth/accounts.go`, and each module `repo.go`/`tx.go` — against the applicable rows; exclude `internal/infrastructure/persistence/postgres/sqlc/*.go`.
- [X] T024 [P] [US1] Refactor `internal/transport/http/handlers/**/*.go` against the applicable rows; preserve request/response shapes, status codes, and error envelopes.
- [X] T025 [P] [US1] Refactor `internal/transport/http/middleware/*.go`, `internal/transport/http/httpx/*.go`, `internal/transport/http/router.go`, and `internal/transport/http/server.go` against the applicable rows; preserve middleware order and route registration.
- [X] T026 [US1] Update the Rule Coverage Summary and Justified Exceptions table in `specs/006-go-style-compliance/compliance-report.md` for all US1 changes, including `internal/application/auth/tokens.go:47` (`jwt.RegisteredClaims` embedding) if retained.

**Checkpoint**: All in-scope source has been reviewed and refactored by rule. US1 is independently verifiable through the coverage summary and a clean batch gate.

---

## Phase 4: User Story 2 - Existing Behavior and Contracts Remain Unchanged (Priority: P1)

**Goal**: Prove the refactor changed style only by comparing observable behavior and contracts to the frozen baseline.

**Independent Test**: Re-run the baseline commands and contract checks; confirm identical route inventory, contract output, schema/migration parity, test results, and immutable artifact hashes.

- [X] T027 [P] [US2] Compare post-US1 `go test ./...`, route-inventory output, and HTTP contract results against `specs/006-go-style-compliance/baseline.md`; record the comparison in `specs/006-go-style-compliance/compliance-report.md`.
- [X] T028 [P] [US2] Run `go test -tags=integration ./...` (or `go test -tags=integration -run '^$' ./...` without Docker) and compare auth, security, idempotency, audit, redaction, and sync results against the baseline; record evidence.
- [X] T029 [P] [US2] Verify public Go source compatibility: run `go build ./...`, inspect `git diff` for changes to exported declarations (`^func [A-Z]`, `^type [A-Z]`, exported constants/variables), and record that no exported identifier was removed or signature-changed.
- [X] T030 [P] [US2] Verify database integrity: compare schema, migration, trigger, and parity test results, and re-compute hashes for `internal/infrastructure/persistence/postgres/sqlc/*.go`, `db/migrations/*`, `docs/schema.sql`, and `openapi/*` against `specs/006-go-style-compliance/baseline.md`.
- [X] T031 [US2] Write the Behavior Preservation Evidence section in `specs/006-go-style-compliance/compliance-report.md`, including commands, results, and any baseline caveats.

**Checkpoint**: US2 can be demonstrated independently: behavior and contracts match the baseline even if later style passes are not complete.

---

## Phase 5: User Story 3 - Developers See Consistent, Readable Code Across Modules (Priority: P2)

**Goal**: Make analogous constructs across modules and adapters follow the same conventions.

**Independent Test**: Compare representative files from at least three business modules and three infrastructure areas and confirm matching naming, ordering, error, and test patterns.

- [X] T032 [US3] Normalize sentinel error naming and placement across `internal/application/**/module.go`, `internal/application/auth/tokens.go`, `internal/application/auth/ratelimit.go`, and `internal/transport/http/middleware/provider.go`; keep `Err` prefixes and preserve error messages.
- [X] T033 [US3] Normalize constructor and method ordering plus receiver names across `internal/application/**/service.go`, `internal/application/**/module.go`, and `internal/infrastructure/persistence/postgres/**/repo.go`; verify with `funcorder` and manual review.
- [X] T034 [US3] Normalize import aliasing for cross-module application packages (for example `appship`, `appvoyage`, `apppartner`) across `internal/application/**/*.go`; remove aliases that do not resolve a conflict or add clarity.
- [X] T035 [US3] Normalize declaration grouping and top-level `const`/`var` ordering in `internal/application/*/module.go`, `internal/application/*/types.go`, and `internal/infrastructure/persistence/postgres/pgerrors.go`; do not reorder in ways that change initialization semantics.
- [X] T036 [US3] Normalize test structure in `internal/application/**/*_test.go`, `internal/infrastructure/**/*_test.go`, and `internal/transport/http/**/*_test.go` for table naming, subtest structure, and safe `t.Parallel()` use; do not change assertions, fixtures, or coverage.
- [X] T037 [US3] Review exported and unexported comments across `internal/application/**`, `internal/infrastructure/**`, `internal/platform/**`, and `internal/transport/**`; fix inaccurate comments, remove code-restating comments, and preserve behavior-significant comments.
- [X] T038 [US3] Record the cross-module consistency review in `specs/006-go-style-compliance/compliance-report.md`, naming the representative modules compared and any retained inconsistencies.

**Checkpoint**: US3 is independently verifiable by comparing representative modules and confirming the recorded consistency review.

---

## Phase 6: User Story 4 - Dead Code and Avoidable Patterns Are Removed (Priority: P2)

**Goal**: Remove placeholders, avoidable mutable globals, redundant code, and undocumented suppressions.

**Independent Test**: Search for placeholder declarations, mutable globals, and `//nolint` directives; confirm each is removed or justified, and that removal does not change compilation, tests, or runtime behavior.

- [X] T039 [P] [US4] Remove `var _ = time.Now` and any now-unused `time` import from `internal/application/staff/service.go` and `internal/application/partner/service.go`.
- [X] T040 [P] [US4] Remove `var _ = pgx.ErrNoRows` and any now-unused `pgx` import from `internal/infrastructure/persistence/postgres/ship/repo.go`.
- [X] T041 [P] [US4] Replace unexported mutable maps with functions or switch-based lookups in `internal/application/booking/service.go` (`serviceBookingStatuses`), `internal/application/voyage/service.go` (`portCallStatuses`), and `internal/infrastructure/persistence/postgres/pgerrors.go` (`constraintFields`); preserve membership results exactly.
- [X] T042 [P] [US4] Replace `init()` validator registration with behavior-equivalent lazy initialization in `internal/platform/config/config.go` and `internal/transport/http/httpx/httpx.go`; preserve tag-name and validation behavior.
- [X] T043 [P] [US4] Evaluate exported globals in `internal/application/ownership/ownership.go` (`Modules`, `TableOwner`); because they are exported contracts, retain them and record a Justified Exception, or add compatible read-only accessors without removing the existing identifiers.
- [X] T044 [US4] Evaluate duplicated `nullUUID`/`validation` helpers in `internal/application/staff/service.go` and `internal/application/partner/service.go`; consolidate only within an existing shared platform package, or record an exception when consolidation would add cross-module coupling.
- [X] T045 [US4] Remove redundant or unreachable code and unused parameters reported by `unparam`/`gocritic` across `internal/**`; confirm each removal preserves behavior and record false positives.
- [X] T046 [US4] Review all 14 `//nolint` directives in `internal/application/booking/service.go`, `internal/application/guest/service.go`, and `internal/application/staff/service.go`; keep only those that name a specific linter and include a reason, remove avoidable ones, and verify `nolintlint` passes.
- [X] T047 [US4] Update the Dead Code and Suppression sections of `specs/006-go-style-compliance/compliance-report.md` with removed items, retained exceptions, and false positives.

**Checkpoint**: US4 is independently verifiable by searching for placeholders, mutable globals, and suppressions and confirming each is removed or recorded.

---

## Phase 7: User Story 5 - Validation Gates Prevent Regression (Priority: P3)

**Goal**: Make the refactor durable by wiring all checks into the canonical workflow and demonstrating a clean final run.

**Independent Test**: From a clean checkout, run the documented validation workflow and confirm formatter, vet, lint, unit/contract tests, build, and integration (or compile-check) all pass with no unexplained findings.

- [X] T048 [US5] Ensure the canonical validation workflow in `Makefile` runs the formatter check, `vet`, `lint`, `test`, `build`, and the integration or compile-check path; document the commands in the `Makefile` comments.
- [X] T049 [US5] Run the full validation suite (`make fmt-check` or equivalent, `make vet`, `make lint`, `make test`, `make build`, `make test-integration` or compile-check) and record every result in `specs/006-go-style-compliance/compliance-report.md`.
- [X] T050 [US5] Verify `go.mod` and `go.sum` contain no new runtime dependencies; confirm any tooling addition is development-only and does not enter the shipped binary.
- [X] T051 [US5] Execute `specs/006-go-style-compliance/quickstart.md` end to end, including the compliance report validation and immutable-artifact checks; record evidence.
- [X] T052 [US5] Review the final `git diff --stat` and `git diff` for unintended edits; confirm only `cmd/`, `internal/`, `Makefile`, `.golangci.yml`, and `specs/006-go-style-compliance/` changed, and update `plan.md`/`spec.md` if implementation diverged.

**Checkpoint**: US5 is independently verifiable by running the documented workflow from a clean checkout and confirming a clean result.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final audit, traceability, and completion evidence.

- [X] T053 [P] Re-run the complete Rule Coverage Matrix audit against `docs/style.md` and update every status in `specs/006-go-style-compliance/compliance-report.md`; confirm zero open findings.
- [X] T054 [P] Confirm `docs/style.md` is unchanged and that no unrelated documentation, generated artifact, schema, migration, or OpenAPI file was modified.
- [X] T055 Verify the constitution gates in `specs/006-go-style-compliance/plan.md` still pass after the full refactor (module boundaries, database integrity, offline-first, data ownership, AWS readiness, security, reliability, testability, spec-driven, simplicity) and record the result.
- [X] T056 Produce the final compliance report evidence: exception count matches inline references, every exception has rule/location/reason/risk/mitigation, immutable hashes match baseline, and final validation results are recorded.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; starts immediately.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories.
- **US1 (Phase 3, P1)**: Depends on Foundational; must complete before US2/US3/US4 because they verify or edit the refactored files.
- **US2 (Phase 4, P1)**: Depends on US1; verification-only tasks T027–T030 may run in parallel.
- **US3 (Phase 5, P2)**: Depends on US1; runs after US2 so behavior evidence covers the first refactor pass.
- **US4 (Phase 6, P2)**: Depends on US1 and US3 because it edits files those phases touched; parallel tasks are limited to non-overlapping files.
- **US5 (Phase 7, P3)**: Depends on all prior stories; final workflow and validation.
- **Polish (Phase 8)**: Depends on US5.

### User Story Dependencies

- **US1 (P1)**: No story dependency after Foundational. MVP scope.
- **US2 (P1)**: Depends on US1 for the refactor to verify.
- **US3 (P2)**: Depends on US1; should not run before US2 to preserve attributable behavior evidence.
- **US4 (P2)**: Depends on US1 and US3 for file stability.
- **US5 (P3)**: Depends on US1–US4.

### Within Each User Story

- US1: module and layer tasks are file-disjoint and parallelizable; the compliance-report task (T026) runs last.
- US2: T027–T030 are read-only comparisons and parallelizable; T031 depends on all four.
- US3: T032–T037 edit overlapping file sets, so run sequentially; T038 depends on all.
- US4: T039–T043 are mostly file-disjoint; T044 follows T039; T045–T047 are sequential.
- US5: T048–T052 are strictly ordered.

### Parallel Opportunities

- Setup: T004 and T005 can run in parallel; T001–T003 prepare independent artifacts.
- US1: T011–T025 target disjoint paths and can be split across developers after T001–T010.
- US2: T027–T030 can run in parallel.
- US4: T039–T043 can run in parallel when assigned non-overlapping files.
- Polish: T053 and T054 are independent audits.

---

## Parallel Example: User Story 1

```text
Developer A: T011 platform + T012 commands/composition root
Developer B: T013 ai/audit + T014 auth
Developer C: T015 booking + T016 feedback/guest + T017 ownership
Developer D: T018 partner + T019 ship + T020 staff + T021 sync/tenant/voyage
Developer E: T022 infrastructure adapters + T023 persistence (non-generated)
Developer F: T024 handlers + T025 middleware/httpx/router/server
```

Each developer runs the batch validation gate before handing off; T026 is completed once after all US1 edits land.

## Parallel Example: User Story 2

```text
Task: T027 unit + contract + route inventory comparison
Task: T028 integration behavior comparison
Task: T029 public Go API compatibility check
Task: T030 schema/migration/parity and immutable hash check
```

T031 records the combined evidence after these complete.

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 Setup and Phase 2 Foundational.
2. Complete US1 (Phase 3), validating each module/layer batch.
3. **STOP and VALIDATE**: confirm the coverage summary and clean batch gate.
4. This delivers the core value: a documented, enforced style baseline across the codebase.

### Incremental Delivery

1. Setup + Foundational → baseline, tooling, coverage scope, and validation sequence ready.
2. US1 → applied style baseline with recorded exceptions (MVP).
3. US2 → proof of behavior and contract preservation.
4. US3 → cross-module consistency and readability.
5. US4 → dead code, mutable globals, and suppression cleanup.
6. US5 → durable validation gates and final evidence.
7. Polish → full coverage audit and completion report.

### Parallel Team Strategy

1. One owner completes Setup and Foundational because they gate every story.
2. After Foundational, US1 modules/layers are split across developers by disjoint paths.
3. US2 verification tasks run in parallel once US1 lands.
4. US3 and US4 are sequential-style passes with limited parallel file ownership.
5. One owner runs US5 and Polish to keep final evidence consistent.

---

## Notes

- Every task preserves behavior; when a style rule conflicts with behavior or an immutable artifact, retain the behavior and record a Justified Exception in `specs/006-go-style-compliance/compliance-report.md`.
- Do not edit `docs/style.md`, generated SQLC code, migrations, `docs/schema.sql`, or `openapi/**`.
- Do not weaken, skip, or delete tests. Existing tests may be refactored for style only.
- Run the batch gate after each logical group; a failed gate blocks progress until resolved.
- Commit only when explicitly requested; this task list does not include commit steps.
