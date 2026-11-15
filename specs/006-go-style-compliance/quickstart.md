# Quickstart: Go Style Guide Compliance Refactor

**Feature**: `006-go-style-compliance`
**Purpose**: Validate that the refactor is complete, behavior-preserving, and compliant with `docs/style.md`.

This guide validates the feature; it does not define implementation steps. See [plan.md](./plan.md), [research.md](./research.md), and [contracts/](./contracts/) for decisions and invariants.

## Prerequisites

- Go 1.26 or newer (`go version`).
- `golangci-lint` v2.14 or newer (`golangci-lint version`).
- `gofmt` (ships with Go).
- Docker daemon for integration tests (optional; integration suites can be compile-checked without it).
- A clean working tree before capturing the baseline.
- Repository dependencies available: `go mod download`.

## 1. Capture the Behavioral Baseline

Run from the repository root:

```sh
git rev-parse HEAD
git status --short
go fmt ./...
go vet ./...
golangci-lint run ./...
go test ./...
go build -o bin/server ./cmd/server
go build -o bin/migrate ./cmd/migrate
go test -tags=integration ./...        # when Docker is available
# Otherwise compile-check the integration suites:
go test -tags=integration -run '^$' ./...
```

Record the results and the hashes of immutable artifacts:

```sh
sha256sum internal/infrastructure/persistence/postgres/sqlc/*.go
sha256sum db/migrations/*
sha256sum docs/schema.sql
sha256sum openapi/*
```

Expected baseline: `go fmt`, `go vet`, and `golangci-lint` are clean, and all Docker-independent tests pass. Any pre-existing failure is documented as a baseline caveat before editing.

## 2. Verify the Style Rule Coverage Matrix

Open [contracts/style-rule-coverage.md](./contracts/style-rule-coverage.md) and confirm:

- Every normative section of `docs/style.md` appears.
- Each rule has an enforcement method (`automated`, `manual`, or `both`).
- Each automated rule maps to a configured check.
- Each manual rule has a concrete review procedure.

When a rule is not applicable, the matrix states why.

## 3. Run a Batch Validation Gate

Apply the same sequence after each refactor batch:

```sh
gofmt -l ./cmd ./internal
golangci-lint fmt --diff ./...
go vet ./...
golangci-lint run ./...
go test ./...
go build ./cmd/server ./cmd/migrate
go test -tags=integration ./...        # when Docker is available
```

Expected outcomes:

- No files listed by the formatter.
- No `vet` findings.
- No lint findings except at locations recorded in the compliance report.
- All unit and contract tests pass.
- Both command entrypoints build.
- Integration tests pass with Docker, or compile successfully without it.

Also confirm the diff is scoped:

```sh
git diff --stat
git status --short internal/infrastructure/persistence/postgres/sqlc db/migrations docs/schema.sql openapi
```

Expected: only batch source files and `specs/006-go-style-compliance/` change; immutable artifacts show no changes.

## 4. Verify the Compliance Report

Open `specs/006-go-style-compliance/compliance-report.md` and confirm:

- It follows [contracts/compliance-report-contract.md](./contracts/compliance-report-contract.md).
- Each recorded exception has rule, location, reason, risk, and mitigation.
- Every inline exception comment references its exception id or rule.
- Open findings are zero.
- Immutable artifact hashes match the baseline.
- The validation evidence section records the final gate results.

## 5. Run the Final Acceptance Check

```sh
go vet ./...
golangci-lint run ./...
go test ./...
go build ./cmd/server ./cmd/migrate
go test -tags=integration ./...        # when Docker is available
git diff --stat
```

Confirm:

- All gates pass with results equal to or better than baseline.
- HTTP route inventory and contract tests are unchanged.
- Schema, migration, trigger, and parity tests are unchanged.
- No exported Go identifier was removed or had its signature changed in a breaking way.
- No new runtime dependency appears in `go.mod`.
- Covered by [contracts/behavior-preservation.md](./contracts/behavior-preservation.md).

## Conflict Handling

If a style rule cannot be satisfied without changing behavior or an immutable artifact:

1. Preserve behavior.
2. Record the deviation in the compliance report.
3. Add an inline comment only if the deviation is locally surprising.
4. Do not change generated output or exported contracts.
5. If the conflict changes the plan, update the spec/plan before proceeding.

## Troubleshooting

| Symptom | Likely Cause | Action |
|---------|--------------|--------|
| Formatter lists files | Import grouping or whitespace not applied | Run the configured formatter; verify the rule in the coverage matrix. |
| New lint findings appear in many files | A newly enabled check is too broad | Tune the check or record a justified exception; do not leave noise. |
| Integration test cannot start | Docker daemon unavailable | Use the compile-check command and record the caveat. |
| Immutable artifact hash changed | Generated file edited or regenerated with a different version | Revert the artifact; change only the source and regenerate deterministically. |
| Contract test fails | Observable behavior changed | Treat as a regression; revert or fix before proceeding. |
