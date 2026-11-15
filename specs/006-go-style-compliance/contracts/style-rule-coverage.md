# Contract: Style Rule Coverage

**Feature**: `006-go-style-compliance`
**Purpose**: Define how every normative section of `docs/style.md` is verified, so "linter clean" cannot be mistaken for "style compliant." This is the review contract for FR-001 through FR-014 and SC-001.

## Automated Check Configuration

The following development-only checks are added to the existing `golangci-lint` v2 configuration. They MUST NOT enter the shipped runtime dependency graph. Existing checks (`errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `depguard`, `staticcheck` settings) remain enabled.

| Check | Type | Rules it enforces |
|-------|------|-------------------|
| `gofmt` / `golangci-lint fmt --diff` | formatter | Base formatting and indentation |
| `goimports` | formatter | Import grouping and ordering |
| `revive` | linter | Naming, exported declarations, error strings, control-flow simplifications, context placement |
| `gocritic` | linter | Style diagnostics and simplifications |
| `misspell` | linter | Spelling in comments and strings |
| `prealloc` | linter | Slice capacity hints |
| `unparam` | linter | Unused parameters |
| `funcorder` | linter | Function grouping and ordering |
| `nolintlint` | linter | Specific, explained, used suppressions |

**Configuration contract**:
- Every added check MUST be wired into the standard validation workflow so it runs in local and CI validation.
- A check that produces systematic false positives MUST be tuned or disabled with a recorded rationale in `research.md` or the compliance report; it MUST NOT be left noisy.
- No added check may require a runtime package already absent from the module graph.

## Rule Coverage Matrix

**Batch key**: B1 = tooling/baseline, B2 = platform and commands, B3 = application modules, B4 = infrastructure, B5 = transport and final sweep.

### Guidelines

| Guide Section | Rule Focus | Enforcement | Check / Manual Procedure | Primary Batch |
|---------------|------------|-------------|--------------------------|---------------|
| Pointers to Interfaces | Pass interfaces by value unless mutation is required | Manual | Search for `*<InterfaceType>` parameters and returns; confirm pointer is required | B5 |
| Verify Interface Compliance | Compile-time compliance assertions where appropriate | Manual | Search for `var _ Interface = (*Impl)(nil)`; add only where it clarifies intent | B3, B4 |
| Receivers and Interfaces | Value receivers for value semantics; nil-safe methods | Manual | Review receivers per type; confirm consistent receiver name and kind | B2–B5 |
| Zero-value Mutexes are Valid | Use zero-value `sync.Mutex`/`RWMutex` | Manual | Search for mutexes initialized in constructors; simplify to zero value when behavior is unchanged | B4, B5 |
| Copy Slices and Maps at Boundaries (Receiving) | Copy caller-owned slices/maps when retaining or mutating | Manual | Review constructors and setters that store slice/map arguments | B2–B5 |
| Copy Slices and Maps at Boundaries (Returning) | Copy internal slices/maps before returning | Manual | Review getters that expose internal collections | B2–B5 |
| Defer to Clean Up | Use `defer` for cleanup; avoid deferred calls in loops | Manual | Search for manual cleanup and `defer` inside loops | B2–B5 |
| Channel Size is One or None | Unbuffered or size-one channels unless justified | Manual | Search for `make(chan` with capacity > 1; justify or change | B2–B5 |
| Start Enums at One | Non-zero enum sentinel where zero is invalid | Manual | Review const blocks with iota; confirm zero value is intentional | B2–B5 |
| Use `time` to Handle Time | `time.Time` for instants, `time.Duration` for periods; no bare integers for time | Both | Review time fields/params; static analysis catches some misuse | B2–B5 |
| Error Types | Prefer opaque errors; custom types only when callers inspect them | Manual | Review custom error types and exported error behavior | B3, B4 |
| Error Wrapping | Wrap with context using `%w`; do not wrap without adding context | Manual | Review `fmt.Errorf` calls and error returns | B2–B5 |
| Error Naming | `Err` prefix for sentinel errors; `err` for locals | Both | `revive` plus manual search for error variables | B2–B5 |
| Handle Errors Once | Do not log and return; handle or propagate, not both | Manual | Review error branches for duplicate logging/handling | B2–B5 |
| Handle Type Assertion Failures | Use comma-ok assertions; never panic on assertion | Both | `govet`/`gocritic` plus manual review | B2–B5 |
| Don't Panic | No panic for expected failures | Manual | Search for `panic(` outside `main`/tests; justify or replace | B2–B5 |
| Use `go.uber.org/atomic` | Use typed atomics; avoid raw `sync/atomic` misuse | Manual | Search for `sync/atomic`; confirm typed or documented alternative | B4, B5 |
| Avoid Mutable Globals | Prefer functions, constants, or immutable values | Manual | Search for package-level `var` maps/slices/pointers; classify | B2–B5 |
| Avoid Embedding Types in Public Structs | Do not leak embedded type APIs from exported structs | Manual | Review exported structs with embedded fields; document justified cases (for example, JWT claims) | B3, B5 |
| Avoid Using Built-In Names | Do not shadow `len`, `cap`, `new`, `copy`, `error`, etc. | Both | `revive`/`gocritic` plus manual search | B2–B5 |
| Avoid `init()` | No `init()`; use explicit initialization or lazy sync | Manual | Search for `func init()`; replace or record exception | B2 |
| Exit in Main / Exit Once | Call `os.Exit` only from `main`; exit at most once | Manual | Search for `os.Exit`; confirm only entrypoints exit | B2 |
| Use Field Tags in Marshaled Structs | Explicit JSON/encoding tags | Both | `govet` plus search for marshaled structs without tags | B3, B5 |
| Don't Fire-and-Forget Goroutines | Every goroutine has an owner and a completion path | Manual | Search for `go func`; confirm lifecycle, cancellation, and wait | B2–B5 |
| Wait for Goroutines to Exit | Use `WaitGroup`/`errgroup`/channels to join | Manual | Review concurrent owners for join logic | B2–B5 |
| No Goroutines in `init()` | No concurrency started during package init | Manual | Search for `go ` in init functions | B2 |

### Performance

| Guide Section | Rule Focus | Enforcement | Check / Manual Procedure | Primary Batch |
|---------------|------------|-------------|--------------------------|---------------|
| Prefer `strconv` over `fmt` | Use `strconv` for primitive conversions | Both | `gocritic`/manual search for `fmt.Sprintf`/`Sscanf` on primitives | B2–B5 |
| Avoid Repeated String-to-Byte Conversions | Convert once per scope | Manual | Review loops and repeated `[]byte(...)`/`string(...)` conversions | B3, B4 |
| Prefer Specifying Container Capacity (Map) | Capacity hints where size is known | Manual | Review map creation in hot paths | B3, B4 |
| Prefer Specifying Container Capacity (Slice) | Capacity hints where size is known | Both | `prealloc` plus manual review | B2–B5 |

### Style

| Guide Section | Rule Focus | Enforcement | Check / Manual Procedure | Primary Batch |
|---------------|------------|-------------|--------------------------|---------------|
| Avoid Overly Long Lines | Readable line length | Both | Formatter check plus the threshold in `research.md` Decision 5 | B2–B5 |
| Be Consistent | Consistent patterns within and across files | Manual | Compare analogous constructs across modules | B5 |
| Group Similar Declarations | Group related `const`/`var`/`type` declarations | Manual | Review declaration blocks per file | B2–B5 |
| Import Group Ordering | Standard, third-party, project groups in order | Automated | `goimports` | B2–B5 |
| Package Names | Short, lower-case, no underscores; avoid generic names | Both | `revive` plus manual review of package names | B2–B5 |
| Function Names | Descriptive names; no `Get` prefix by default | Both | `revive` plus manual review | B2–B5 |
| Import Aliasing | Alias only to resolve conflicts or improve clarity | Manual | Review import aliases | B2–B5 |
| Function Grouping and Ordering | Constructors before methods; exported before unexported where practical | Automated | `funcorder` plus manual review | B2–B5 |
| Reduce Nesting | Early returns over deep nesting | Both | `gocritic`/`revive` plus manual review | B2–B5 |
| Unnecessary Else | Drop `else` after a terminating branch | Both | `revive`/`gocritic` plus manual review | B2–B5 |
| Top-level Variable Declarations | Standard grouping and ordering | Manual | Review package-level `var` blocks | B2–B5 |
| Prefix Unexported Globals with `_` | Prefix package-level unexported mutable globals | Manual | Search for unexported package-level `var`; rename or record | B2–B5 |
| Embedding in Structs | Embed for interface implementation, not convenience | Manual | Review embedded fields; document public-struct exceptions | B3–B5 |
| Local Variable Declarations | Use `:=` where clear; `var` for zero values | Both | `gocritic` plus manual review | B2–B5 |
| `nil` is a Valid Slice | Return `nil` slices rather than empty allocations when equivalent | Manual | Review slice returns and comparisons | B2–B5 |
| Reduce Scope of Variables | Declare variables in the narrowest scope | Manual | Review declarations before `if`/`for` that can move inside | B2–B5 |
| Avoid Naked Parameters | Name ambiguous boolean/numeric parameters with a comment or options struct | Manual | Review calls with bare `true`/`false`/numeric literals where meaning is unclear | B2–B5 |
| Use Raw String Literals to Avoid Escaping | Use backtick strings for regex/escaped content | Manual | Search for strings with heavy escaping | B2–B5 |
| Use Field Names to Initialize Structs | No positional struct literals | Both | `govet` (composites) plus manual review | B2–B5 |
| Omit Zero Value Fields in Structs | Omit zero values from struct literals | Manual | Review struct literals for redundant zero fields | B2–B5 |
| Use `var` for Zero Value Structs | `var s T` for empty structs | Manual | Review `T{}` usage | B2–B5 |
| Initializing Struct References | Use `&T{...}` consistently | Manual | Review pointer struct initialization | B2–B5 |
| Initializing Maps | Use `make`/literals appropriately; avoid nil-map writes | Both | `gocritic` plus manual review | B2–B5 |
| Format Strings outside Printf | Use constants for non-Printf format strings | Manual | Search for format strings passed to logging/format helpers | B2–B5 |
| Naming Printf-style Functions | `f` suffix for Printf-style helpers | Manual | Review helper names | B2–B5 |

### Patterns

| Guide Section | Rule Focus | Enforcement | Check / Manual Procedure | Primary Batch |
|---------------|------------|-------------|--------------------------|---------------|
| Test Tables | Table-driven tests with clear case names | Manual | Review test files; confirm cases are independently named and asserted | B2–B5 |
| Avoid Unnecessary Complexity in Table Tests | Keep setup out of table literals | Manual | Review large inline table entries | B2–B5 |
| Parallel Tests | Use `t.Parallel()` where safe | Manual | Review independent tests; add only when shared state is absent | B2–B5 |
| Functional Options | Use functional options for extensible constructors | Manual | Review constructors with many optional parameters; apply only when it simplifies without new abstraction | B3, B4 |

### Linting

| Guide Section | Rule Focus | Enforcement | Check / Manual Procedure | Primary Batch |
|---------------|------------|-------------|--------------------------|---------------|
| Lint Runners | Consistent lint runner and base checks | Automated | `golangci-lint` configuration and validation workflow | B1 |

## Manual Review Protocol

For each batch:
1. List every file in scope and mark the guide sections that apply.
2. Run the automated checks and collect findings.
3. For each manual rule, run the documented search procedure or inspect the relevant constructs.
4. Classify each finding as `fixable`, `exception_required`, or `false_positive`.
5. Fix `fixable` findings; record `exception_required` findings in the compliance report; justify `false_positive` findings in the batch notes.
6. Re-run the batch validation gate before marking the batch accepted.

## Completion Contract

The feature is complete only when:
- Every row above has been reviewed for every in-scope file.
- All automated checks report zero findings, except locations covered by a recorded exception.
- No `ComplianceFinding` remains in `open` state.
- The compliance report lists every recorded exception with rule, location, reason, risk, and mitigation.
