# Implementation Plan: Initialize Backend Foundation

**Branch**: `001-backend-foundation` | **Date**: 2026-11-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-backend-foundation/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command; its definition describes the
execution workflow.

## Summary

Establish a production-ready but deliberately minimal Go backend foundation: a single module with
clear internal package boundaries, environment-only configuration with fail-fast validation, a pgx
connection pool with lifecycle management, versioned Goose migrations, sqlc-generated database
access, reusable AWS SDK v2 configuration that requires no credentials at startup, a Chi HTTP server
with health/readiness endpoints and structured request logging, graceful shutdown, container and
local orchestration files, a minimal Terraform scaffold, and reproducible developer commands.

No business logic, domain entities, business tables, business APIs, authentication flows, or feature
modules are introduced.

## Technical Context

**Language/Version**: Go 1.26 (toolchain currently installed: `go1.26.8`)

**Primary Dependencies**: `github.com/go-chi/chi/v5`, `github.com/go-chi/cors`,
`github.com/jackc/pgx/v5` (pgxpool), `github.com/pressly/goose/v3`, `github.com/spf13/viper`,
`github.com/go-playground/validator/v10`, `github.com/aws/aws-sdk-go-v2/config`,
`github.com/testcontainers/testcontainers-go`

**Storage**: PostgreSQL 16 (local via Docker Compose); no business schema, only Goose's own version
table and a no-op technical migration

**Testing**: Go standard `testing`; unit tests run without infrastructure; integration tests use
`testcontainers-go` behind the `integration` build tag; no AWS account required

**Target Platform**: Linux server container (amd64/arm64); AWS EC2 in a future phase

**Project Type**: Web service — single Go module, modular monolith

**Performance Goals**: Startup under 5 seconds with a reachable database; `/health` and `/ready`
respond in under 100 ms p95 locally; graceful shutdown within the configured timeout (default 10 s)

**Constraints**: Must start with no AWS credentials or AWS network access; must fail fast on invalid
configuration or unavailable database; no metrics, tracing, or profiling endpoints; no business
logic or schema; no Kubernetes, EKS, or serverless

**Scale/Scope**: One backend service, foundation only, zero feature modules, zero business tables

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status |
|-----------|------|--------|
| I. Modular Monolith | Single Go module; internal packages with explicit boundaries; no microservices | PASS |
| II. Database Integrity | Versioned Goose migrations; pgx/sqlc; no business tables; constraints reserved for future business schema | PASS |
| III. Offline-First Architecture | Foundation introduces no cloud-inbound connection or sync contract; hub-initiated model remains untouched | PASS |
| IV. Data Ownership | No business entities or operational data introduced; no ownership ambiguity possible | PASS |
| V. AWS Readiness | AWS SDK behind an internal package with DI; local development needs no credentials; Terraform scaffold included | PASS |
| VI. Security | Secrets only from environment; no hardcoded credentials; CORS allowlist denies by default; health output leaks nothing | PASS |
| VII. Reliability | Timeouts on external dependencies; graceful shutdown; migrations run as a separate command, never at startup | PASS |
| VIII. Testability | Business-independent unit tests; integration tests against real PostgreSQL; no AWS account needed | PASS |
| IX. Specification-Driven Development | Spec, clarifications, plan, research, contracts, and quickstart produced before implementation | PASS |
| X. Simplicity | Minimal dependency set; no Kubernetes, Kafka, CRDT, or speculative abstractions | PASS |

**Post-design re-check**: No violations identified after Phase 1. The Complexity Tracking table is
intentionally empty.

## Project Structure

### Documentation (this feature)

```text
specs/001-backend-foundation/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── openapi.yaml
│   └── configuration.md
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created here)
```

### Source Code (repository root)

```text
cmd/
├── server/
│   └── main.go              # Application entry point
└── migrate/
    └── main.go              # Migration entry point (up/down/status/version)

internal/
├── app/
│   └── app.go               # Bootstrap, lifecycle orchestration, graceful shutdown
├── config/
│   ├── config.go            # Config structs, env mapping, defaults, validation
│   └── config_test.go
├── logging/
│   ├── logging.go           # slog initialization and context helpers
│   └── logging_test.go
├── database/
│   ├── postgres.go          # pgxpool initialization, ping, lifecycle
│   ├── migrate.go           # Embedded Goose migration runner
│   ├── postgres_test.go     # Integration test (build tag: integration)
│   └── sqlc/                # sqlc-generated code (never hand-edited)
├── awsconfig/
│   ├── config.go            # AWS SDK v2 config, region, endpoint override
│   └── config_test.go
├── httpserver/
│   ├── server.go            # http.Server construction and shutdown
│   ├── router.go            # Chi router and middleware wiring
│   ├── handlers/
│   │   ├── health.go        # GET /health and GET /ready
│   │   └── health_test.go
│   └── middleware/
│       ├── request_logger.go
│       ├── request_logger_test.go
│       ├── recoverer.go
│       ├── recoverer_test.go
│       └── cors.go
├── health/
│   └── checker.go           # Database health monitor that triggers shutdown
├── apperrors/
│   └── errors.go            # Shared error types, codes, safe messages
└── testutil/
    └── postgres.go          # Testcontainers PostgreSQL helper (build tag: integration)

db/
├── migrations/
│   └── 00001_init.sql       # No-op technical migration (proves the pipeline)
└── queries/
    └── health.sql           # Minimal sqlc example query (SELECT 1)

terraform/
├── versions.tf              # Terraform and provider version constraints
├── providers.tf             # AWS provider configuration
├── variables.tf             # Input variables
├── outputs.tf               # Naming and tag outputs
├── main.tf                  # Locals for naming/tagging; no resources by default
├── environments/
│   ├── dev/terraform.tfvars
│   └── prod/terraform.tfvars
└── README.md

Dockerfile
docker-compose.yml
sqlc.yaml
Makefile
.env.example
.dockerignore
.gitignore
README.md
```

**Structure Decision**: A single Go module at the repository root using the standard `cmd/` entry
points plus `internal/` packages. `internal/` prevents accidental external imports and enforces the
modular-monolith boundary while keeping the option to add feature packages later. No `pkg/` or
`modules/` tree is created because the foundation exposes no public library and no feature modules
exist yet. Generated sqlc code lives under `internal/database/sqlc` and is isolated from handwritten
code.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations. No added complexity requires justification.
