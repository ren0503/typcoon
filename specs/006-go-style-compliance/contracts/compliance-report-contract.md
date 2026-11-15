# Contract: Compliance Report

**Feature**: `006-go-style-compliance`
**Artifact path**: `specs/006-go-style-compliance/compliance-report.md`
**Purpose**: Define the single durable record of justified style deviations (FR-018, SC-001, SC-011).

## Required Structure

The report MUST contain these sections in this order:

1. **Header**: feature id, generation date, baseline commit, validation environment note (for example, Docker availability).
2. **Summary**: total files reviewed, findings fixed, false positives, recorded exceptions, open findings (MUST be zero at completion).
3. **Rule Coverage Summary**: one line per guide section with enforcement method and outcome (`clear`, `fixed`, `exception`, `not_applicable`).
4. **Justified Exceptions**: a table with exactly one row per recorded exception.
5. **Immutable Artifact Verification**: hashes or diff status for generated SQLC files, migrations, `docs/schema.sql`, and `openapi/**`.
6. **Validation Evidence**: the final results of each validation gate.

## Justified Exceptions Table

| Column | Required | Content |
|--------|----------|---------|
| ID | yes | Stable identifier, e.g. `E-001`. |
| Style Rule | yes | Guide section and rule summary, e.g. `Avoid Embedding Types in Public Structs`. |
| Location | yes | Repository-relative `path:line`. |
| Reason | yes | Why no behavior-preserving fix exists. Must be specific, not "legacy". |
| Risk | yes | What could break if changed (serialization, public API, generated output, concurrency). |
| Mitigation | yes | Existing test, contract, or hash that protects the behavior. |
| Inline Comment | yes | `yes` or `no`; `yes` requires the comment to name the exception id or rule. |

**Example row**:

| E-001 | Avoid Embedding Types in Public Structs | internal/application/auth/tokens.go:47 | Access token payload embeds `jwt.RegisteredClaims` to inherit registered-claim serialization required by the token contract. | Flattening the struct changes JSON field handling and token compatibility. | JWT token round-trip tests and auth contract tests. | yes |

## Rules

- Exactly one row per recorded exception. No duplicates.
- Fixed findings are summarized by count and batch; they are not listed individually.
- A row whose `Inline Comment` is `yes` MUST correspond to a real inline comment naming the exception id or style rule.
- The report MUST state that open findings are zero before completion.
- The report MUST NOT include secrets, tokens, personal data, or environment-specific credentials.
- The report MUST NOT be used to justify renaming or removing an exported identifier; those changes are prohibited (FR-015, SC-007).
- The report MUST be updated after each batch so an interrupted refactor remains auditable.

## Validation Protocol

1. Compare the report's exception count with a repository search for inline exception references; they MUST agree.
2. Confirm every exception location still exists and still matches the described construct.
3. Confirm every exception has a linked mitigation test or contract result.
4. Confirm the immutable-artifact verification section shows no changes.
5. Confirm open findings are zero in the final batch.

## Non-Compliance

The feature MUST NOT be reported complete when any of the following is true:
- A deviation is retained without a report row.
- A report row lacks a reason, risk, or mitigation.
- Open findings remain.
- An immutable artifact hash changed.
- A recorded exception no longer reflects the code it references.
