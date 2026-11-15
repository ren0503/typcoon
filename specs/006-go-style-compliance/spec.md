# Feature Specification: Go Style Guide Compliance Refactor

**Feature Branch**: `006-go-style-compliance`

**Created**: 2026-11-15

**Status**: Draft

**Input**: User description: "Refactor the existing codebase to ensure that all source code strictly follows the coding conventions and best practices defined in docs/style.md. Improve consistency, readability, maintainability, and structural quality without changing existing business logic or application behavior."

## Clarifications

### Session 2026-11-15

- Q: Where should justified style exceptions be recorded so reviewers and later audits can find them? → A: A compliance report file in the feature directory listing rule, location, and reason; inline comment only when the deviation is locally surprising.
- Q: May this refactor change the project's lint configuration or add lint tooling to enforce more of the style guide, or must those remain unchanged? → A: Add new lint tooling or dependencies if needed to enforce the style guide.
- Q: How should existing lint-suppression comments (for example, `//nolint:errcheck` on deferred rollbacks) be handled during the refactor? → A: Review each suppression; keep it only with a documented reason, otherwise fix the underlying issue when behavior is unchanged.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reviewers Can Rely on a Single Enforced Style Baseline (Priority: P1)

A maintainer reviews a change and can confirm that the source code follows the project's authoritative Go style guide rather than reviewer preference. Style discussions focus on documented rules, and deviations are either fixed or explicitly justified.

**Why this priority**: Without a single enforced baseline, the codebase drifts, reviews become subjective, and the same issues recur across modules. This is the core outcome of the refactor.

**Independent Test**: Sample source files across every layer (command, composition root, application modules, infrastructure, transport, platform) and verify that each documented style category is either satisfied or has a recorded, justified exception.

**Acceptance Scenarios**:

1. **Given** the refactored repository, **When** a maintainer samples files from each architectural layer, **Then** naming, declaration grouping, import organization, error handling, struct initialization, and control-flow patterns consistently follow `docs/style.md`.
2. **Given** a style rule from `docs/style.md`, **When** a reviewer searches the codebase for violations of that rule, **Then** no unexplained violations remain in non-generated source files.
3. **Given** a rule that conflicts with required behavior or generated-code constraints, **When** the conflict is encountered, **Then** the behavior is preserved and the exception is documented rather than silently ignored.

---

### User Story 2 - Existing Behavior and Contracts Remain Unchanged (Priority: P1)

An API consumer, operator, or downstream runtime observes exactly the same behavior after the refactor: routes, request and response shapes, validation messages, authentication and authorization rules, pagination, idempotency, audit output, synchronization semantics, database schema, and migrations all remain unchanged.

**Why this priority**: This is a style-only refactor. Any behavioral change would be an unacceptable regression and would invalidate the purpose of the work.

**Independent Test**: Run the full existing test suite, contract tests, route inventory checks, and build before and after the refactor, and compare the observable outputs.

**Acceptance Scenarios**:

1. **Given** the pre-refactor test suite, **When** the refactored code is tested, **Then** all unit, integration, and contract tests pass without being weakened, skipped, or deleted.
2. **Given** the documented HTTP surface, **When** the route inventory and HTTP contract checks run, **Then** routes, methods, status codes, request shapes, and response envelopes are identical.
3. **Given** existing authentication, rate limiting, audit, booking idempotency, guest-document secrecy, and synchronization behavior, **When** their tests run, **Then** behavior is unchanged.
4. **Given** the database schema, migrations, triggers, and generated persistence code, **When** the migration and parity checks run, **Then** none of them has changed.
5. **Given** any public Go type, interface, function signature, or exported constant used by another package, **When** the refactor is complete, **Then** its contract remains source-compatible.

---

### User Story 3 - Developers See Consistent, Readable Code Across Modules (Priority: P2)

A developer moving between business modules, infrastructure adapters, and transport handlers encounters the same conventions for naming, grouping, ordering, error handling, and tests. Onboarding and code navigation require less context switching.

**Why this priority**: Consistency reduces cognitive load and defect risk, and it is the everyday benefit of compliance. It is secondary to establishing the baseline and preserving behavior.

**Independent Test**: Compare representative files from at least three business modules and three infrastructure areas and verify that analogous constructs use the same naming, ordering, and error-handling patterns.

**Acceptance Scenarios**:

1. **Given** two modules that implement similar use cases, **When** their service and repository files are compared, **Then** analogous functions, types, helpers, and error values follow the same conventions.
2. **Given** a file with mixed declarations, **When** it is reviewed, **Then** similar declarations are grouped, functions are ordered predictably, and imports follow the documented grouping and ordering rules.
3. **Given** tests in any package, **When** they are reviewed, **Then** table-driven tests, subtests, and parallel tests follow the documented patterns where applicable.

---

### User Story 4 - Dead Code and Avoidable Patterns Are Removed (Priority: P2)

A maintainer can remove placeholder declarations, unused imports, redundant logic, and avoidable global initialization patterns so that the code reflects only what is required.

**Why this priority**: Dead declarations and avoidable patterns obscure intent and can mask unused dependencies. Cleaning them improves maintainability without changing behavior.

**Independent Test**: Search for placeholder declarations and unused dependencies, and verify that removal does not alter compilation, tests, or runtime behavior.

**Acceptance Scenarios**:

1. **Given** package-level placeholder declarations that exist only to keep an import alive, **When** the refactor is complete, **Then** they and their unused imports are removed.
2. **Given** package-level mutable state that the style guide discourages, **When** it is not required for behavior, **Then** it is replaced with a non-mutable or lazily initialized equivalent, or documented as a justified exception.
3. **Given** duplicated trivial helpers across packages, **When** consolidation does not introduce new coupling or change behavior, **Then** the duplication is reduced using the existing shared platform packages.

---

### User Story 5 - Validation Gates Prevent Regression (Priority: P3)

A maintainer can run the project's formatting, static analysis, linting, tests, contract checks, and build in a single, repeatable workflow that confirms style compliance and behavioral safety.

**Why this priority**: Guardrails keep the repository compliant after the refactor. They make the improvement durable rather than a one-time cleanup.

**Independent Test**: Run the documented validation commands on a clean checkout and confirm that each passes and reports no style-related findings introduced by the refactor.

**Acceptance Scenarios**:

1. **Given** a clean checkout of the refactored repository, **When** the formatter check is run, **Then** it reports no files requiring formatting.
2. **Given** the refactored repository, **When** static analysis and linting are run, **Then** they complete with no findings.
3. **Given** the refactored repository, **When** the unit test target and the build target are run, **Then** both succeed.
4. **Given** a development environment with the required container runtime, **When** the integration test target is run, **Then** it succeeds; when that runtime is unavailable, the integration suites still compile successfully.

---

### Edge Cases

- A documented style rule conflicts with required serialization, framework integration, or generated-code behavior. The refactor must preserve behavior and record the exception instead of forcing the rule.
- Generated persistence code or migration artifacts cannot be manually edited. Changes must be applied only to their sources, and regenerated output must remain byte-identical to the committed artifacts.
- A rule would require renaming an exported identifier consumed outside the package. Backward compatibility takes precedence, so the identifier is retained and the conflict is reported.
- A rule suggests removing package-level state that is intentionally shared for correctness, such as a single validation registry. The behavior must be preserved through an equivalent safe pattern or a documented exception.
- A style-only change alters test assertions or golden files. The change must be reverted or adjusted so the test still verifies the original behavior.
- Two modules contain near-identical helpers. Consolidation is allowed only when it does not create a new cross-module dependency or change exported behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The refactor MUST treat `docs/style.md` as the single authoritative source for coding style and conventions.
- **FR-002**: The refactor MUST cover all hand-written Go source under `cmd/` and `internal/`, including tests, while excluding generated code, vendored code, and migration artifacts.
- **FR-003**: Naming MUST follow the documented conventions for packages, functions, receivers, errors, constants, and unexported globals.
- **FR-004**: Declarations MUST be organized according to the documented rules for grouping similar declarations, import order, function grouping and ordering, and top-level variable declarations.
- **FR-005**: Error handling MUST follow the documented patterns for error types, wrapping, naming, handling errors once, and type-assertion safety.
- **FR-006**: Control flow MUST be simplified according to the documented rules for reducing nesting, avoiding unnecessary `else`, and reducing variable scope.
- **FR-007**: Struct and map initialization MUST follow the documented rules, including named fields, omission of zero-value fields, `var` for zero-value structs, and map initialization.
- **FR-008**: Package-level initialization MUST avoid `init()` and mutable globals where a behavior-equivalent alternative exists; any retained use MUST be documented as a justified exception.
- **FR-009**: Public structs and interfaces MUST follow the documented rules for interface compliance, interface pointers, embedding, and field tags.
- **FR-010**: Concurrency-related code MUST follow the documented rules for channels, goroutine lifecycle, and waiting for goroutines to exit.
- **FR-011**: Performance-related rules that are applicable without changing behavior, such as container capacity hints and avoiding repeated conversions, MUST be applied where they improve clarity without over-engineering.
- **FR-012**: Placeholder declarations, unused imports, unreachable code, and redundant logic MUST be removed.
- **FR-013**: Comments and documentation MUST be accurate, useful, and consistent with the documented conventions; comments that merely restate code SHOULD be removed.
- **FR-014**: Test code MUST follow the documented patterns for table tests, subtest complexity, and parallel tests.
- **FR-015**: No business logic, functional requirement, feature, public API contract, HTTP contract, database schema, migration, trigger, event payload, or authentication/synchronization behavior may change.
- **FR-016**: No new runtime dependency may be introduced solely for the refactor; development-time lint or static-analysis tooling MAY be added when it is needed to enforce a specific rule in `docs/style.md`, provided it does not enter the shipped runtime dependency graph.
- **FR-017**: No architectural change may be made unless `docs/style.md` explicitly requires it; stylistic preference alone is not sufficient justification.
- **FR-018**: Every conflict between a style rule and existing behavior or generated-code constraints MUST be recorded in a compliance report file under the feature directory, listing the rule, the affected location, and the reason it was not changed; an inline code comment MUST be added only when the deviation would be locally surprising to a reader.
- **FR-019**: The refactor MUST be applied incrementally in reviewable batches, and each batch MUST pass the available validation gates before the next batch begins.
- **FR-020**: The final result MUST pass the project formatter, static analysis, linter, unit tests, and build; integration tests MUST be run when the required runtime is available.
- **FR-021**: Any lint or static-analysis check added to enforce `docs/style.md` MUST be documented with the rule it protects, MUST be wired into the project's standard validation workflow, and MUST report zero findings on the refactored source except for recorded Justified Exceptions.
- **FR-022**: Existing lint-suppression comments MUST be reviewed individually; a suppression MAY be retained only when it carries a documented reason and no behavior-preserving fix exists, and otherwise the underlying finding MUST be fixed.

### Key Entities

- **Style Rule**: A documented convention in `docs/style.md`, identified by its section, that defines an expected coding pattern.
- **Compliance Finding**: An observed source location that does not satisfy a Style Rule, including whether it is fixable without behavior change or requires a documented exception.
- **Source Batch**: A reviewable group of related files or packages refactored together and validated before proceeding.
- **Validation Gate**: A project command or check (formatting, static analysis, linting, unit tests, integration tests, contract checks, build) that must pass for a batch or the final result.
- **Justified Exception**: A retained deviation from a Style Rule where changing it would alter behavior, break compatibility, or modify generated artifacts, recorded in the compliance report with its reason.
- **Compliance Report**: A review artifact in the feature directory listing every Justified Exception with its Style Rule, affected location, and reason.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of hand-written Go source files under `cmd/` and `internal/` are reviewed against every applicable section of `docs/style.md`, and each finding is either fixed or recorded as a justified exception in the compliance report.
- **SC-002**: The formatter reports zero files requiring changes.
- **SC-003**: Static analysis and linting complete with zero findings.
- **SC-004**: 100% of existing unit and contract tests pass without deletion, skipping, or weakened assertions.
- **SC-005**: The build completes successfully for every command entrypoint.
- **SC-006**: The HTTP route inventory, HTTP contract validation, and database schema/migration parity checks remain unchanged.
- **SC-007**: Zero exported identifiers are removed or have their signatures changed in a way that breaks existing package consumers.
- **SC-008**: Zero new runtime dependencies appear in the module manifest.
- **SC-009**: Every placeholder declaration that existed only to keep an import alive is removed, and no unused imports remain.
- **SC-010**: Integration test suites pass when the required container runtime is available and compile successfully when it is not.
- **SC-011**: Reviewers can trace each non-trivial change in the final diff to a specific Style Rule or to dead-code removal, and every retained deviation appears in the compliance report.
- **SC-012**: No unrelated files, generated artifacts, or documentation outside the refactor scope are modified.
- **SC-013**: Every lint or static-analysis check added during the refactor reports zero findings on hand-written source, except where a Justified Exception is recorded, and no added tooling enters the shipped runtime dependency graph.
- **SC-014**: Every lint-suppression comment remaining after the refactor is intentional and documents its reason, and every suppression without such justification has been removed along with its underlying finding.

## Assumptions

- `docs/style.md` is the Uber Go Style Guide as vendored in this repository and is authoritative over general Go community preference.
- The refactor's "users" are maintainers and developers; benefits are expressed in review speed, consistency, and reduced regression risk rather than end-user features.
- Generated persistence code, migrations, and other tool-generated artifacts are out of scope for manual edits; their sources remain the only editable inputs.
- The repository already passes its current lint configuration, so the work targets rules in `docs/style.md` that are not fully covered by the existing linter settings, and may extend the lint tooling to enforce them durably.
- The project's canonical validation entry points are the `fmt`, `vet`, `lint`, `test`, `test-integration`, and `build` targets defined in the `Makefile`.
- Integration tests require a container runtime; when unavailable, compile-checking the integration suites is an acceptable substitute for running them.
- Existing public Go identifiers are treated as contracts and are not renamed or removed even when a style rule suggests otherwise.
- Where two style rules or a rule and a behavior constraint conflict, behavior preservation and backward compatibility win, and the conflict is reported.
