# Data Model: Initialize Backend Foundation

**Feature**: `001-backend-foundation` | **Date**: 2026-11-01

## Scope Note

This feature introduces **no business entities, aggregates, or business database tables**. The
entities below are technical configuration and interface structures required to run the service.
They are documented so implementation and tests share a single contract.

## Technical Entities

### ApplicationConfig

Represents the complete runtime configuration assembled from process environment variables.

| Field | Type | Required | Default | Validation |
|-------|------|----------|---------|------------|
| `AppName` | string | No | `tycoon-backend` | non-empty |
| `AppEnv` | enum string | No | `development` | `development`, `test`, `staging`, `production` |
| `LogLevel` | enum string | No | `info` | `debug`, `info`, `warn`, `error` |
| `HTTP.Host` | string | No | `0.0.0.0` | non-empty |
| `HTTP.Port` | int | No | `8080` | 1–65535 |
| `HTTP.ReadTimeout` | duration | No | `5s` | > 0 |
| `HTTP.WriteTimeout` | duration | No | `10s` | > 0 |
| `HTTP.IdleTimeout` | duration | No | `60s` | > 0 |
| `HTTP.ShutdownTimeout` | duration | No | `10s` | > 0 |
| `Database.Host` | string | Yes | — | non-empty |
| `Database.Port` | int | No | `5432` | 1–65535 |
| `Database.Name` | string | Yes | — | non-empty |
| `Database.User` | string | Yes | — | non-empty |
| `Database.Password` | string | Yes | — | non-empty; secret; never logged |
| `Database.SSLMode` | enum string | No | `disable` | `disable`, `require`, `verify-ca`, `verify-full` |
| `Database.MaxConns` | int32 | No | `10` | > 0 |
| `Database.MinConns` | int32 | No | `0` | >= 0 and <= `MaxConns` |
| `Database.MaxConnLifetime` | duration | No | `1h` | > 0 |
| `Database.MaxConnIdleTime` | duration | No | `30m` | > 0 |
| `Database.ConnectTimeout` | duration | No | `5s` | > 0 |
| `AWS.Region` | string | No | `us-east-1` | non-empty |
| `AWS.EndpointURL` | string | No | empty | valid absolute URL when set |
| `CORS.AllowedOrigins` | []string | No | empty | each item an absolute origin; empty denies all |

**Relationships**: `ApplicationConfig` is owned by the bootstrap layer and passed by value or
read-only pointer to the HTTP, database, AWS, and logging packages. It has no persistence.

**Invariants**:

- The password is never included in logs, error messages, or health responses.
- An empty `CORS.AllowedOrigins` list means no origin is permitted.
- `MinConns` never exceeds `MaxConns`.

### DatabasePool

Represents the live PostgreSQL connection pool.

| Field | Type | Notes |
|-------|------|-------|
| `Pool` | `*pgxpool.Pool` | Created at startup from `ApplicationConfig.Database` |
| `ConnectTimeout` | duration | Bounds the startup ping |
| `HealthInterval` | duration | Cadence of the background health monitor |
| `HealthTimeout` | duration | Per-check timeout |

**Lifecycle**:

```text
created → pinged (startup) → healthy ⇄ unhealthy → closed
```

- Startup ping failure causes process exit.
- Any later monitor failure triggers graceful shutdown and process exit.
- `Close` is idempotent and runs after the HTTP server stops.

### HealthResponse

Returned by `GET /health`.

| Field | Type | Value |
|-------|------|-------|
| `status` | string | `ok` |

**Invariant**: Contains no dependency, version, host, or error details.

### ReadinessResponse

Returned by `GET /ready`.

| Field | Type | Value |
|-------|------|-------|
| `status` | string | `ready` or `not_ready` |

HTTP status is `200` when ready and `503` when not ready. No error text, database address, or
internal detail is included.

### ErrorResponse

Returned for HTTP errors raised by the foundation (for example, method-not-allowed or not-found).

| Field | Type | Notes |
|-------|------|-------|
| `error.code` | string | Stable machine-readable code |
| `error.message` | string | Safe, human-readable message |
| `error.request_id` | string | Correlation identifier when available |

**Invariant**: The wrapped internal cause is logged but never serialized.

### ApplicationError

Shared error type used internally.

| Field | Type | Notes |
|-------|------|-------|
| `Code` | string | Stable code such as `internal_error` |
| `Message` | string | Safe public message |
| `Cause` | error | Internal cause; logged only |
| `HTTPStatus` | int | Status used when rendered by an HTTP handler |

### RequestLogRecord

Structured fields emitted for each HTTP request.

| Field | Type | Notes |
|-------|------|-------|
| `request_id` | string | Generated when absent |
| `method` | string | HTTP method |
| `path` | string | Request path without query string |
| `status` | int | Response status |
| `duration_ms` | int64 | Handler duration |
| `remote_ip` | string | Client address |
| `user_agent` | string | Optional; truncated |

**Invariant**: Query strings, authorization headers, cookies, and request bodies are never logged.

### AWSRuntimeConfig

Reusable AWS SDK configuration prepared at startup.

| Field | Type | Notes |
|-------|------|-------|
| `Config` | `aws.Config` | Loaded with default credential chain |
| `Region` | string | From application configuration |
| `EndpointURL` | string | Optional override for local emulation |

**Invariant**: No service client is constructed and no AWS API call is made at startup. Credentials
are resolved lazily by the SDK only when a future client actually makes a call.

## State Transitions

### Service Lifecycle

```text
starting → running → shutting_down → stopped
    │                                  ▲
    └──────── startup failure ─────────┘ (exit non-zero)
```

- `starting`: configuration loaded and validated, logger initialized, pool created and pinged, AWS
  config loaded, router built.
- `running`: HTTP server accepting requests; database monitor active.
- `shutting_down`: triggered by a termination signal or database monitor failure; HTTP server stops
  accepting new requests, in-flight requests drain within the shutdown timeout, pool closes.
- `stopped`: process exits; zero on clean signal shutdown, non-zero on startup or runtime failure.

## Validation Rules

- Required fields missing or empty → startup fails with a non-sensitive message naming the setting.
- Invalid enum, port, URL, or duration → startup fails naming the setting.
- `MinConns > MaxConns` → startup fails.
- Database ping failure within `ConnectTimeout` → startup fails.
- Database monitor failure after startup → graceful shutdown then non-zero exit.
- CORS origin not in the allowlist → no permissive cross-origin headers are returned.
