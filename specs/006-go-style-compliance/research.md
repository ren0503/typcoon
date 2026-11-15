# Phase 0 Research: Go Style Guide Compliance Refactor

**Feature**: `006-go-style-compliance`
**Date**: 2026-11-15
**Sources**: `docs/style.md`, `.golangci.yml`, `Makefile`, `go.mod`, repository source tree

## Context

`docs/style.md` is the vendored Uber Go Style Guide. It is a generated document (header states it is produced by `stitchmd` and must not be edited) and contains guidelines, style rules, performance rules, patterns, and a linting recommendation. The repository currently passes `gofmt`, `go vet`, and the configured `golangci-lint` run with zero issues, so the remaining gap is between the current lint coverage and the broader style guide. There are no `[NEEDS CLARIFICATION]` items in the plan's Technical Context; the decisions below resolve the open implementation-strategy questions.

## Decision 1: Treat the style guide as a rule inventory and classify each rule by enforcement method

**Decision**: Extract the section-level rules from `docs/style.md` into a coverage matrix that marks each rule as automated, manual, or both. Automated rules are enforced by formatter/linter configuration; manual rules are reviewed per batch using explicit search patterns and reviewer checklists. The matrix lives in `contracts/style-rule-coverage.md`.

**Rationale**: Many style guide sections are semantic and not fully lintable (copying slices at boundaries, choosing error wrapping depth, goroutine lifecycle, test table design, functional options). Treating only lint findings as compliance would leave the spec's SC-001 unmet. A matrix makes coverage auditable and prevents "linter clean" from being mistaken for "style compliant."

**Alternatives considered**:
- Rely on the existing linter only: rejected because it already passes and therefore cannot detect the target violations.
- Rewrite all files against the guide without a matrix: rejected as unreviewable and prone to behavior changes.
- Only review files with known violations: rejected because SC-001 requires 100% file coverage.

## Decision 2: Add development-only lint tooling aligned to the style guide

**Decision**: Extend the existing `golangci-lint` v2 configuration with the following, without adding runtime dependencies:
- Formatter: `goimports`, to enforce import grouping and ordering (style guide "Import Group Ordering").
- Linters: `revive` (naming, exported declarations, error strings, control-flow simplifications), `gocritic` (style diagnostics), `misspell` (comment/string spelling), `prealloc` (capacity hints), `unparam` (unused parameters), `funcorder` (function grouping/ordering), and `nolintlint` (requires specific, explained, used suppression directives).
- Keep the existing defaults (`errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`) and the `depguard` module-boundary rule.
- Wire formatting enforcement into the standard validation workflow (for example, a formatter-diff check alongside `make lint`), so `make fmt` remains usable but compliance is checked non-destructively.

**Rationale**: These tools map directly to documented guide sections and are already available in the installed `golangci-lint` v2.14 binary, so no new dependency is required. `nolintlint` directly enforces the clarified suppression policy (FR-022). `funcorder` covers the guide's function grouping/ordering rule. Tuning aggressive rules to avoid false positives is expected; retained exceptions are recorded.

**Tuning decision (revive)**: Disable the `revive` `exported` and `package-comments` rules. `docs/style.md` does not require doc comments on every exported identifier, and the existing configuration already disables `staticcheck` `ST1000` (package comments). Leaving the rules enabled produced 408 findings that were documentation requests rather than guide violations, conflicting with the change-scoped approach in Decision 7 and the "do not rewrite compliant files" rule. Comment accuracy remains covered by FR-013 and the manual review in US3. `revive` continues to enforce naming, error strings, context placement, and other guide-aligned rules.

**Alternatives considered**:
- Add a new standalone linter binary or a new module dependency: rejected because the installed runner already provides the needed checks and adds no runtime surface.
- Enable every available linter: rejected because it would create large, low-value churn and risk behavior changes.
- Enable `golines` for automatic line wrapping: rejected for the initial pass because it can reformat large areas unrelated to a rule and increases review risk; long-line handling is defined in Decision 5.

## Decision 3: Record justified exceptions in one compliance report

**Decision**: Create `specs/006-go-style-compliance/compliance-report.md` with one row per retained deviation containing: ID, style rule (guide section), location (`file:line`), reason, behavior or compatibility risk, and validation evidence. Inline comments are added only when a reader would otherwise be surprised at the deviation. Fixed findings are summarized by count, not listed individually. The report format is defined in `contracts/compliance-report-contract.md`.

**Rationale**: A single report gives reviewers and future maintainers one auditable location (clarification Q1), keeps production code free of explanatory noise, and makes SC-001/SC-011 mechanically checkable.

**Alternatives considered**:
- Inline comments for every deviation: rejected as noisy and easy to miss in review.
- Final summary only: rejected because it is not durable.

## Decision 4: Refactor in validation-gated batches ordered by layer

**Decision**: Use five batches, each ending in the full available validation gate:
1. Tooling and baseline: add lint configuration, capture pre-refactor validation baseline, and generate the rule-coverage matrix.
2. Platform and commands: `internal/platform/**`, `cmd/**`, and composition root `internal/app/**`.
3. Application modules: `internal/application/**` including test files.
4. Infrastructure: `internal/infrastructure/**` (excluding generated `sqlc/*.go`) including test files.
5. Transport and final sweep: `internal/transport/**`, remaining tests, cross-package consistency, and the final full validation run.

**Rationale**: Layer ordering respects dependency direction from the constitution, keeps batches reviewable, and allows early detection of tooling false positives before broad edits. Tests travel with their batch so each layer is validated in place; the final sweep catches cross-layer consistency.

**Alternatives considered**:
- Refactor by style rule across the whole repository: rejected because each batch would touch every layer and be hard to review or bisect.
- One single batch: rejected as unreviewable and high risk.
- Refactor by module: rejected because shared platform and transport code would be deferred too long, delaying tooling feedback.

## Decision 5: Handle non-generated long lines with an explicit threshold

**Decision**: Apply the guide's "avoid overly long lines" rule to hand-written, non-generated Go files using a 120-character target for lines added or touched during the refactor. Existing lines longer than 120 that are not otherwise changed may remain when wrapping would reduce readability or churn unrelated code; those retained cases are recorded only if they exceed 160 characters and are not justified by a single unavoidable literal or signature. Generated `internal/infrastructure/persistence/postgres/sqlc/*.go` files are excluded.

**Rationale**: The guide gives no numeric limit. A 120-character target matches common Go practice, while the change-scoped rule prevents a repository-wide reformat that would obscure behavioral review. The 160-character escalation keeps a bounded, auditable exception set.

**Alternatives considered**:
- Wrap every line over 80/100/120 characters: rejected as a large, risky, low-value diff (552 lines exceed 120 and 98 exceed 160, many in generated code).
- Ignore long lines entirely: rejected because the guide explicitly names the rule.
- Adopt an auto-wrapping formatter immediately: rejected for review risk; may be revisited after the main refactor.

## Decision 6: Apply the clarified suppression policy mechanically

**Decision**: Review all 14 existing `//nolint` directives (mostly `//nolint:errcheck` on deferred rollbacks). Retain a suppression only when the behavior-preserving alternative is unsafe or unavailable, and require the directive to name the linter and include a reason. Enable `nolintlint` to prevent unexplained or unused directives. Replace avoidable suppressions with explicit error handling only when behavior is provably unchanged.

**Rationale**: Matches the accepted clarification (Q3) and the guide's "Handle Errors Once" rule. `nolintlint` makes the policy durable.

**Alternatives considered**:
- Remove all suppressions: rejected because deferred transaction rollback errors are intentionally ignored and handling them would change control flow or logging.
- Leave all suppressions: rejected because it preserves undocumented deviations.

## Decision 7: Keep generated and migration artifacts immutable

**Decision**: Do not edit `internal/infrastructure/persistence/postgres/sqlc/*.go`, `db/migrations/**`, `docs/schema.sql`, `openapi/**`, or any file marked as generated. If a change appears necessary, change only the source query or migration and regenerate, then confirm the regenerated output is byte-identical to the committed artifacts. Any mismatch is treated as a conflict to investigate, not a style fix.

**Rationale**: Preserves database-integrity and API contracts (constitution principles II and III) and the spec's generated-artifact edge case.

**Alternatives considered**:
- Reformat generated files directly: rejected because regeneration would overwrite the change and the door would open to schema-contract drift.
- Regenerate without checking the diff: rejected because unrelated generator changes could alter behavior or artifacts.

## Decision 8: Establish a behavior baseline before editing

**Decision**: Before the first source edit, record the baseline results of `make fmt` (or formatter check), `make vet`, `make lint`, `make test`, `make build`, and `make test-integration` when Docker is available. Compare each batch and the final state against this baseline. Treat any new failure, changed route inventory, changed contract output, changed schema/migration parity result, or changed generated artifact as a regression to resolve before proceeding.

**Rationale**: The refactor must be behavior-preserving. A recorded baseline makes regressions attributable to a batch and prevents "it passed on my machine" ambiguity. Integration suites compile-check with `go test -tags=integration -run '^$' ./...` when Docker is unavailable, as documented in the `Makefile`.

**Alternatives considered**:
- Rely on the final test run only: rejected because a failure would not identify the responsible batch.
- Add new behavioral tests: rejected as out of scope; existing tests are the contract.

## Decision 9: Change-scoped refactoring for already-compliant files

**Decision**: Do not rewrite files that already comply. For each file, record the applicable rule scan result; only edit the specific constructs that violate a rule. Avoid mass renames of unexported identifiers unless the guide requires the change and the identifier is local. Keep diffs mechanically traceable to a rule or dead-code removal.

**Rationale**: Matches the task's "minimal, focused refactoring" instruction and SC-011/SC-012 (traceable changes, no unrelated modifications).

**Alternatives considered**:
- Full-file reformatting while touching a file: rejected because it hides behavioral review and violates the minimal-diff constraint.

## Resolved Unknowns

| Unknown | Resolution |
|---------|------------|
| Whether lint tooling may change | Yes; development-only additions are allowed (clarification Q2, FR-016/FR-021). |
| How to record exceptions | Single compliance report with inline comments only when surprising (clarification Q1, FR-018). |
| How to treat suppressions | Review individually; require specific linter and reason; fix when safe (clarification Q3, FR-022). |
| How to handle long lines | 120-character target for changed hand-written lines; bounded exceptions (Decision 5). |
| How to validate behavior | Pre-change baseline plus per-batch and final gates (Decision 8). |
| How to handle generated code | Immutable; regenerate from source only and confirm byte-identical output (Decision 7). |
