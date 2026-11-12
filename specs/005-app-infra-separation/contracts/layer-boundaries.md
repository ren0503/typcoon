# Contract: Layer Boundaries and Dependency Direction

**Feature**: `005-app-infra-separation`
**Date**: 2026-11-12

This contract defines the allowed import directions. It is enforced by an automated
architecture test, not by convention alone.

## Layer Paths

| Layer | Path prefix | Responsibility |
|---|---|---|
| Application | `internal/application/` | Business types, rules, use cases, consumer-owned interfaces |
| Infrastructure | `internal/infrastructure/` | pgx/sqlc/Goose, AWS, KMS, Redis, Firebase, local security, logging, health |
| Transport | `internal/transport/` | Chi/HTTP server, handlers, DTOs, middleware, request/response mapping |
| Platform | `internal/platform/` | Config, shared errors, pagination, test-only support |
| Composition root | `internal/app` | Runtime assembly and lifecycle only |
| Commands | `cmd/` | Thin entrypoints; config/logger/signal and one composition-root call |

## Import Matrix

`Y` = allowed, `N` = forbidden, `*` = only through the documented exception.

| Importer \ Imported | application | infrastructure | transport | platform | app | cmd |
|---|---|---|---|---|---|---|
| application | Y | N | N | Y | N | N |
| infrastructure | Y | Y | N | Y | N | N |
| transport | Y | N | Y | Y | N | N |
| platform | N | N | N | Y | N | N |
| platform/testutil | Y* | Y* | N | Y | N | N |
| app | Y | Y | Y | Y | Y | N |
| cmd | N | N | N | Y | Y | N |

Additional rules:

- Application packages must not import `net/http`, `database/sql`, chi, pgx, pgtype,
  pgconn, pgerrcode, sqlc, Goose, AWS SDK, Redis, Firebase, or testcontainers.
- Application packages may import pure value/helper libraries such as `google/uuid`,
  `samber/lo`, `golang-jwt/jwt/v5`, and `golang.org/x/crypto`.
- Consumers may import another application module's public value type when a method result
  requires it. They must not import that module's repository, adapter, SQL, or transport.
- Infrastructure adapters import application interfaces; application never imports an
  adapter to obtain an interface.
- Transport handlers import application interfaces only; no handler opens a database
  connection or calls an SDK.
- Platform packages contain no business policy. `platform/testutil` is test-only and may
  import infrastructure to build PostgreSQL harnesses.
- Only `internal/app` imports application and infrastructure together.

## Interface Ownership

- The consuming application module declares capability interfaces: repository, transaction,
  audit append, signer/verifier, rate limiter, cipher, and cross-module lookup ports.
- The provider/application module may publish public value types and its own service
  interface. Consumers declare the narrower interface they use.
- `auth.Principal` is the explicitly allowed shared authenticated-caller value type.
- The HTTP `Provider` interface and `*http.Request` parsing live in
  `transport/http/middleware`; `auth.Principal` stays in `application/auth`.
- Development-only implementations (dev header principals, local signing, local document
  cipher) live outside application packages.

## Transaction and Audit Direction

- Application services own transaction boundaries through module-owned `Tx` interfaces.
- Infrastructure implements `Begin`/`Commit`/`Rollback` using one pgx transaction.
- Audit policy and `Entry` validation are application-owned; audit SQL is infrastructure-owned.
- `AppendAudit` on the application `Tx` must use the same driver transaction as the business
  writes.
- `pgx.Tx`, `pgconn.CommandTag`, `sqlc.*`, and `pgtype.*` must not appear in application
  package source.

## Architecture Test Rules

`internal/app/boundaries_test.go` (or its relocated equivalent) must:

1. Run `go list -json ./...` and build the complete import graph.
2. Classify every internal package by the prefixes above.
3. Fail when an application package imports infrastructure, transport, `internal/app`, or a
   forbidden external prefix.
4. Fail when infrastructure imports transport or `internal/app`.
5. Fail when transport imports infrastructure or `internal/app`.
6. Fail when platform (except testutil) imports a higher layer.
7. Fail on any import cycle across the repository.
8. Report both the offending package and imported package in failure messages.
9. Be mutation-tested: introducing a forbidden import must make the test fail; removing it
   must restore a pass.

Representative forbidden external prefixes:

```text
net/http
database/sql
github.com/go-chi/
github.com/jackc/pgx/
github.com/jackc/pgerrcode
github.com/aws/aws-sdk-go-v2
github.com/redis/go-redis
firebase.google.com/
github.com/testcontainers/
github.com/pressly/goose
```

## Enforcement Boundary

The test enforces production package rules. Test files may import infrastructure only when
they are infrastructure tests or `platform/testutil`. Production application files receive
no exceptions beyond the public value-type rule and `auth.Principal`.
