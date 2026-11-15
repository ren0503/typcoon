# Phase 1 Data Model: Go Style Guide Compliance Refactor

**Feature**: `006-go-style-compliance`
**Date**: 2026-11-15
**Scope**: This feature has no database or runtime data model changes. The entities below model the refactor's working artifacts and review evidence so completeness and traceability are verifiable. Physical persistence is limited to Markdown files under the feature directory.

## Entity Overview

```text
StyleRule 1 ─── * ComplianceFinding * ─── 0..1 JustifiedException
      │                     │
      │                     └── belongs to ── SourceBatch 1 ─── * ValidationGate
      │
      └── summarized in ── StyleRuleCoverage (contract matrix)

ComplianceReport 1 ─── * JustifiedException
BaselineSnapshot 1 ─── * ValidationGate (pre-refactor results)
```

## StyleRule

A documented convention from `docs/style.md`.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Stable identifier derived from the guide section, e.g. `import-group-ordering`. |
| `section` | string | yes | Heading path in `docs/style.md`, e.g. `Style > Import Group Ordering`. |
| `summary` | string | yes | One-sentence statement of the expected pattern. |
| `enforcement` | enum | yes | `automated`, `manual`, or `both`. |
| `automated_checks` | list[string] | when automated | Formatter or linter names that detect the rule. |
| `manual_review` | string | when manual | Search pattern or reviewer instruction used to verify the rule. |
| `batch` | SourceBatch ref | yes | The batch primarily responsible for the rule. |
| `applicable` | bool | yes | False only when the rule cannot apply to this repository (record why). |

**Validation rules**:
- Every guide section in the coverage matrix MUST map to at least one `StyleRule`.
- `automated_checks` MUST reference a check that is actually configured and runnable.
- `enforcement=manual` rules MUST have a concrete `manual_review` procedure.
- A rule may appear in more than one batch only for the final cross-layer sweep; a primary batch MUST exist.

**Identity/uniqueness**: `id` is unique across the matrix.

## ComplianceFinding

One observed source location that may violate a StyleRule.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Unique finding identifier, e.g. `F-APPL-014`. |
| `rule_id` | StyleRule ref | yes | The rule the location may violate. |
| `location` | string | yes | Repository-relative `path:line` or `path:line-line`. |
| `classification` | enum | yes | `fixable`, `exception_required`, or `false_positive`. |
| `status` | enum | yes | `open`, `fixed`, `recorded`, or `verified`. |
| `resolution` | string | when closed | What changed, or the exception id when recorded. |
| `evidence` | string | when fixed/recorded | Validation command result or rule quote supporting closure. |

**Validation rules**:
- `fixable` findings MUST end as `fixed` or be reclassified with a reason.
- `exception_required` findings MUST link to a JustifiedException before becoming `recorded`.
- `false_positive` findings MUST state why the rule does not apply and require reviewer confirmation.
- A finding cannot be `verified` without recorded `evidence`.

**State transitions**:
```text
open → fixed → verified
open → recorded (requires JustifiedException)
open → false_positive → verified
fixed → open (validation regression)
```

## SourceBatch

A reviewable unit of refactoring that is validated before the next batch begins.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Batch identifier, e.g. `B2-platform-commands`. |
| `name` | string | yes | Human-readable batch name. |
| `paths` | list[string] | yes | Repository paths in scope. |
| `excluded_paths` | list[string] | yes | Generated or out-of-scope paths explicitly excluded. |
| `status` | enum | yes | `planned`, `in_progress`, `validated`, `accepted`, or `blocked`. |
| `gate_results` | list[ValidationGate] | yes | Results for the batch validation run. |
| `findings` | list[ComplianceFinding] | yes | Findings assigned to the batch. |
| `diff_summary` | string | when validated | Counts by rule and by dead-code removal. |

**Validation rules**:
- `accepted` requires every required gate to be `passed` or explicitly `not_applicable` with a reason.
- A batch MUST NOT start while a prior batch is `blocked`.
- `excluded_paths` MUST include generated SQLC output, migrations, and other immutable artifacts.

**State transitions**:
```text
planned → in_progress → validated → accepted
                    └── blocked → in_progress (after resolving failure)
```

## ValidationGate

A command or check that proves formatting, static analysis, tests, or build health.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Gate identifier, e.g. `fmt`, `lint`, `unit`, `build`, `integration`. |
| `command` | string | yes | Exact command from the `Makefile` or formatter check. |
| `required` | bool | yes | Whether the gate blocks acceptance. |
| `applicability` | enum | yes | `always`, `docker_required`, or `compile_check_only`. |
| `state` | enum | yes | `not_run`, `passed`, `failed`, `not_applicable`. |
| `output_summary` | string | when run | Finding counts, failing packages, or artifact paths. |
| `run_at` | timestamp | when run | When the gate was executed. |
| `batch_id` | SourceBatch ref | yes | Batch or `final` sweep the result belongs to. |

**Validation rules**:
- `passed` requires zero formatter changes, zero lint findings beyond recorded exceptions, and zero test/build failures.
- `integration` is `docker_required`; when Docker is unavailable it becomes `compile_check_only` using `go test -tags=integration -run '^$' ./...`.
- A final sweep MUST include every required gate.

## JustifiedException

A retained, intentional deviation from a StyleRule.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Exception identifier, e.g. `E-003`. |
| `rule_id` | StyleRule ref | yes | The rule being deviated from. |
| `location` | string | yes | Repository-relative `path:line`. |
| `reason` | string | yes | Why no behavior-preserving fix exists. |
| `risk` | string | yes | Behavior, compatibility, or generated-artifact risk if changed. |
| `mitigation` | string | yes | Test or contract that protects the behavior. |
| `inline_comment` | bool | yes | Whether a locally surprising deviation also has an inline comment. |
| `status` | enum | yes | `proposed`, `recorded`, or `rejected`. |

**Validation rules**:
- Every `recorded` exception MUST have a non-empty `reason`, `risk`, and `mitigation`.
- Renaming or removing exported identifiers is not a valid exception reason; such changes are prohibited instead.
- `inline_comment=true` MUST correspond to an actual comment naming the exception id or rule.
- Rejected exceptions MUST be resolved by a fix or reclassified finding.

**State transitions**:
```text
proposed → recorded (reviewer accepts)
proposed → rejected (behavior-preserving fix exists)
recorded → rejected (new evidence enables a fix)
```

## ComplianceReport

The single durable artifact listing every recorded exception.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `feature` | string | yes | `006-go-style-compliance`. |
| `generated_at` | date | yes | Report generation date. |
| `rule_coverage` | list[StyleRule] | yes | Summary of rules reviewed and enforcement method. |
| `findings_fixed` | int | yes | Count of fixed findings. |
| `findings_false_positive` | int | yes | Count of confirmed false positives. |
| `exceptions` | list[JustifiedException] | yes | Every recorded exception. |
| `remaining_violations` | list[ComplianceFinding] | yes | Open findings; MUST be empty at completion. |

**Validation rules**:
- MUST be located at `specs/006-go-style-compliance/compliance-report.md`.
- MUST contain exactly one row per recorded exception.
- MUST state `remaining_violations` is empty before the feature is reported complete.
- MUST NOT list fixed findings individually; counts and batch summaries are sufficient.

## StyleRuleCoverage (contract view)

The review matrix defined in `contracts/style-rule-coverage.md`.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `guide_section` | string | yes | Heading path in `docs/style.md`. |
| `rule_ids` | list[StyleRule] | yes | Rules extracted from that section. |
| `enforcement` | enum | yes | `automated`, `manual`, or `both`. |
| `checks` | list[string] | yes | Linter/formatter or manual procedure. |
| `batch` | SourceBatch ref | yes | Primary batch. |
| `status` | enum | yes | `clear`, `partial`, or `missing` at review start. |

**Validation rules**:
- Every top-level section and sub-section that contains normative guidance MUST appear.
- `status` MUST be updated as findings are resolved.
- Sections that do not apply MUST be marked `clear` with a short reason.

## BaselineSnapshot

The pre-refactor validation state used for regression comparison.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `commit` | string | yes | Git commit at baseline capture. |
| `captured_at` | timestamp | yes | Capture time. |
| `gates` | list[ValidationGate] | yes | Baseline results for every validation gate. |
| `artifact_hashes` | map[string]string | yes | Hashes for generated and immutable artifacts. |
| `notes` | string | no | Environment caveats, e.g. Docker availability. |

**Validation rules**:
- Baseline MUST be captured before any source edit.
- `artifact_hashes` MUST cover generated SQLC files, migrations, `docs/schema.sql`, and `openapi/**`.
- Final comparison MUST show identical hashes for immutable artifacts.
