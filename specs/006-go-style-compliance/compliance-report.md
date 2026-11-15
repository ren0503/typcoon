# Compliance Report: Go Style Guide Refactor

- **Feature**: `006-go-style-compliance`
- **Baseline commit**: `18e99a458678aa0bd32e50818ef36f6867590afc`
- **Generated**: 2026-11-15
- **Environment**: Go 1.26.8; golangci-lint 2.14.0; Docker daemon unavailable (integration suites compile-checked).

## Summary

| Metric | Value |
|--------|-------|
| Hand-written Go files in scope | 187 |
| Findings fixed | All automated findings resolved: `gocritic` 2, `funcorder` 3, `unparam` 4, `nolintlint` 17; plus manual fixes (placeholders, mutable maps, `init`, global validator, long lines >160, `interface{}` → `any`) |
| False positives | 408 `revive` `exported`/`package-comments` findings (rules disabled; documented in `research.md`) |
| Recorded exceptions | 6 |
| Open findings | 0 |

## Rule Coverage Summary

| Guide Section | Enforcement | Outcome |
|---------------|-------------|---------|
| Pointers to Interfaces | manual | Reviewed; handlers and services accept interface values, no pointer-to-interface parameters found |
| Verify Interface Compliance | manual | Clean; compile-time assertions present for repositories, transactions, signers, limiters, and verifiers |
| Receivers and Interfaces | manual | Clean; receivers are consistent per type and zero-value mutexes are used |
| Zero-value Mutexes are Valid | manual | Clean; mutexes are declared in structs and never initialized in constructors |
| Copy Slices and Maps at Boundaries | manual | Reviewed; convert helpers allocate result slices rather than retain caller-owned backing arrays |
| Defer to Clean Up | manual | Clean; file, transaction, response, and key cleanup uses `defer` |
| Channel Size is One or None | manual | Reviewed; production channels are size 1; test-only result channels use size 2 (EX-004) |
| Start Enums at One | manual | Clean; the only `iota` block uses a deliberate zero sentinel in a test |
| Use `time` to Handle Time | both | Clean; `time.Time`/`time.Duration` used for instants and periods |
| Error Types | manual | Clean; `RateLimitedError` is exported because callers inspect `RetryAfter` |
| Error Wrapping | manual | Retained `%w: %v` sentinel pattern recorded in EX-003 |
| Error Naming | both | Clean; `Err` prefix for sentinels, `err` for locals |
| Handle Errors Once | manual | Retained contextual log-then-return pattern recorded in EX-002 |
| Handle Type Assertion Failures | both | Clean; no unguarded type assertions outside tests |
| Don't Panic | manual | Clean; no production `panic` calls |
| Use `go.uber.org/atomic` | manual | Clean; no raw `sync/atomic` usage |
| Avoid Mutable Globals | manual | Exported read-only ownership tables retained in EX-001; all validation maps replaced with switch-based functions |
| Avoid Embedding Types in Public Structs | manual | Clean; only a test fake embeds a handler interface |
| Avoid Using Built-In Names | both | Clean |
| Avoid `init()` | manual | Clean; config validator now uses an explicit constructor |
| Exit in Main / Exit Once | manual | Clean; `os.Exit` only in `cmd/server` and `cmd/migrate` `main` |
| Use Field Tags in Marshaled Structs | both | Clean; `govet` reports no issues |
| Don't Fire-and-Forget Goroutines | manual | Clean; server and monitor goroutines are joined through result channels |
| Wait for Goroutines to Exit | manual | Clean; `App.Run` selects on server/monitor errors and context cancellation |
| No Goroutines in `init()` | manual | Clean; no `init` functions remain |
| Prefer `strconv` over `fmt` | both | Reviewed; `fmt` use is message formatting, not primitive conversion |
| Avoid Repeated String-to-Byte Conversions | manual | Reviewed; no repeated conversions in loops |
| Prefer Specifying Container Capacity (Map) | manual | Reviewed; maps are small lookup tables or created with known sizes |
| Prefer Specifying Container Capacity (Slice) | both | Clean; `prealloc` reports no issues |
| Avoid Overly Long Lines | both | Clean for the 120-character target on touched lines; all production lines >160 wrapped, remaining 121–160 lines retained per Decision 5 |
| Be Consistent | manual | Reviewed; validation and conversion patterns are consistent across modules |
| Group Similar Declarations | manual | Clean |
| Import Group Ordering | automated | Clean; `goimports` formatting check passes |
| Package Names | both | Clean |
| Function Names | both | Exported `Get*` accessors retained for API compatibility in EX-005 |
| Import Aliasing | manual | Clean; aliases resolve conflicts and clarify module ownership |
| Function Grouping and Ordering | automated | Clean; `funcorder` reports no issues |
| Reduce Nesting | both | Clean |
| Unnecessary Else | both | Clean; sequences rewritten to `switch` where flagged |
| Top-level Variable Declarations | manual | Clean; grouped declarations are errors, interface assertions, or immutable values |
| Prefix Unexported Globals with `_` | manual | Clean; no mutable unexported globals remain |
| Embedding in Structs | manual | Clean |
| Local Variable Declarations | both | Clean |
| `nil` is a Valid Slice | manual | Reviewed; list helpers return `nil` on empty results where callers accept it |
| Reduce Scope of Variables | manual | Reviewed |
| Avoid Naked Parameters | manual | Reviewed; no ambiguous naked booleans/numbers in production call sites |
| Use Raw String Literals to Avoid Escaping | manual | Reviewed; no escaped regular expressions |
| Use Field Names to Initialize Structs | both | Clean |
| Omit Zero Value Fields in Structs | manual | Reviewed |
| Use `var` for Zero Value Structs | manual | Reviewed |
| Initializing Struct References | manual | Clean; pointer structs use `&T{...}` |
| Initializing Maps | both | Clean |
| Format Strings outside Printf | manual | Reviewed; log messages use structured key/value pairs |
| Naming Printf-style Functions | manual | Clean |
| Test Tables | manual | Reviewed; table-driven tests are used across contract and unit suites |
| Avoid Unnecessary Complexity in Table Tests | manual | Reviewed |
| Parallel Tests | manual | Reviewed; no parallelization added to suites with shared database fixtures |
| Functional Options | manual | Reviewed; existing exported constructors are retained unchanged for API compatibility |
| Lint Runners | automated | Clean; `golangci-lint` is wired into `make lint` and `make fmt-check` |

## Justified Exceptions

| ID | Style Rule | Location | Reason | Risk | Mitigation | Inline Comment |
|----|------------|----------|--------|------|------------|----------------|
| EX-001 | Avoid Mutable Globals | `internal/application/ownership/ownership.go:24`, `:40` | `Modules` and `TableOwner` are exported, read-only ownership contracts used by boundary tests and persistence checks; removing or hiding them would break the public Go surface and the ownership map | A future change could mutate the tables | No writes exist; treat them as read-only and keep mutation out of callers | No |
| EX-002 | Handle Errors Once | Application services (`internal/application/**`, 165 `logger.Error`/`Warn` sites) | Services intentionally log domain failures with principal and input context before returning the error to the transport layer; removing the logs would change observability behavior | Duplicate log entries if callers also log | Retain the existing pattern for behavior preservation; new code should prefer a single handling site | No |
| EX-003 | Error Wrapping | `internal/application/auth/tokens.go:151`; `internal/application/auth/service.go:91,96,101,195,199,203,579`; `internal/application/auth/module.go:195`; `internal/infrastructure/cache/redis/limiter.go:48`; `internal/infrastructure/identity/firebase/verifier.go:78` | `fmt.Errorf("%w: %v", sentinel, err)` wraps the stable sentinel while keeping the underlying driver/verification error out of the chain, preserving error classification | `errors.Is` cannot match the underlying error | Deliberate; revisit only with a classification change and contract tests | No |
| EX-004 | Channel Size is One or None | `internal/application/ship/import_integration_test.go:305`; `internal/application/auth/exchange_integration_test.go:221`; `identity_integration_test.go:186`; `refresh_integration_test.go:159` | Test-only result channels use capacity 2 so two concurrent workers can report without blocking | None; test-only | Keep; production channels remain size 1 | No |
| EX-005 | Function Names | Exported `Get*` service and repository accessors across `internal/application/**` and `internal/infrastructure/persistence/**` | Renaming exported methods would break the public Go API and the module interfaces; the guide recommends avoiding `Get` by default, not as a hard rule | API inconsistency if new code copies the pattern | Retain for compatibility; new unexported helpers avoid the `Get` prefix | No |
| EX-006 | Functional Options / Simplicity | Duplicated `nullUUID`, `validation`, and `trimPtr` helpers in `internal/application/{staff,partner,sync,guest,booking}/service.go` | Consolidating these 3–7 line helpers would require a new shared application package or cross-module dependency, increasing coupling for negligible duplication | Small repeated helpers | Keep local helpers until a shared platform package exists; do not add one solely for these helpers | No |

## Behavior Preservation Evidence

- Unit and contract suites (`go test ./...`) pass, including `TestOpenAPISpecIsValid`, `TestRouteInventoryExcludesInternalTables`, and `TestBusinessRouteTreeBuilds`.
- `go build ./...` succeeds for every package; no exported declaration name was removed or renamed in modified `.go` files (checked by comparing exported `func`/`type`/`var`/`const` names against `HEAD`).
- Immutable artifacts are byte-identical to `baseline.md`; the SQLC, migration, schema, and OpenAPI hashes match.
- Integration suites compile with `go test -tags=integration -run '^$' ./...`; Docker was unavailable, so runtime integration behavior was not re-executed.
- No generated artifact, database object, route shape, request/response field, or error envelope was edited.

## Cross-Module Consistency Review

Representative modules compared: `auth`, `ship`, `staff`, `partner`, `voyage`, and their persistence adapters.

- Constructors use `New` and appear before methods; `funcorder` reports zero findings after moving `appendAudit`, `requireShip`, and `loadPublicKey`.
- Validation lookups use switch-based predicates (`isShipStatus`, `isHubRole`, `isDepartment`, `isPartnerStatus`, `isServiceBookingStatus`, `isPortCallStatus`) instead of package-level maps.
- Sentinel errors remain grouped in `module.go`/`interface.go` with `Err` prefixes; error messages are unchanged.
- Import aliases (`appship`, `appvoyage`, `apppartner`, `apperrors`) are used only to resolve package-name conflicts or clarify ownership.
- Retained inconsistencies are the exceptions above (`Get*` accessors, ownership tables, log-then-return, `%w: %v` sentinel wrapping, test channel sizes, local helper duplication).

## Dead Code and Suppressions

- Removed placeholders: `var _ = time.Now` in `staff`/`partner` services and `var _ = pgx.ErrNoRows` in the ship repository; the now-unused `pgx` import was removed.
- Removed `init()` from `internal/platform/config/config.go`; validation now uses an explicit `newValidator` constructor.
- Replaced the package-level `httpx` validator with `sync.OnceValue(newValidator)` lazy initialization.
- Replaced validation maps (`serviceBookingStatuses`, `portCallStatuses`, `constraintFields`, ship/sync/staff/partner status maps) with switch-based functions; preserved membership results.
- Replaced the 14 repeated `//nolint:errcheck` transaction-rollback suppressions with a package-local `rollback(ctx, tx)` helper that explicitly discards the rollback error; no `//nolint` directives remain and `nolintlint` passes.
- No unused parameters remain in modified files; `unparam` and `gocritic` report zero findings.

## Immutable Artifact Verification

Baseline hashes are recorded in `baseline.md`. Final `sha256sum` output for `internal/infrastructure/persistence/postgres/sqlc/*.go`, `db/migrations/*.sql`, `docs/schema.sql`, and `openapi/openapi.yaml` is byte-identical to the baseline.

## Validation Evidence

| Gate | Command | Result |
|------|---------|--------|
| Format | `gofmt -l ./cmd ./internal` | No files listed |
| Formatter diff | `golangci-lint fmt --diff ./...` | No diff |
| Vet | `go vet ./...` | Pass |
| Lint | `golangci-lint run ./...` | 0 issues |
| Unit/contract tests | `go test ./...` | All packages pass |
| Build | `go build ./cmd/server ./cmd/migrate` | Success |
| Integration | `go test -tags=integration ./...` | Not run (Docker unavailable) |
| Integration compile-check | `go test -tags=integration -run '^$' ./...` | Pass |
| Contract tests | `TestOpenAPISpecIsValid`, `TestRouteInventoryExcludesInternalTables`, `TestBusinessRouteTreeBuilds` | Pass |
| Suppressions | `grep -rn '//nolint:'` | 0 directives remain |
