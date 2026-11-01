# Configuration Contract

**Feature**: `001-backend-foundation`

The application reads configuration **only from process environment variables**. It does not load a
`.env` file. Developer tooling (Makefile or Docker Compose) may load `.env` before starting the
application.

## Required vs Optional

- **Required** settings have no safe default. Missing or empty values cause startup to fail with a
  non-sensitive error naming the setting.
- **Optional** settings have defaults suitable for local development. Production environments MUST
  review and override them where noted.

## Environment Variables

| Variable | Required | Default | Allowed values | Notes |
|----------|----------|---------|----------------|-------|
| `APP_NAME` | No | `tycoon-backend` | any non-empty string | Used in logs |
| `APP_ENV` | No | `development` | `development`, `test`, `staging`, `production` | Controls log format and behavior |
| `LOG_LEVEL` | No | `info` | `debug`, `info`, `warn`, `error` | |
| `HTTP_HOST` | No | `0.0.0.0` | any host | |
| `HTTP_PORT` | No | `8080` | `1`–`65535` | |
| `HTTP_READ_TIMEOUT` | No | `5s` | positive duration | Go duration syntax |
| `HTTP_WRITE_TIMEOUT` | No | `10s` | positive duration | Go duration syntax |
| `HTTP_IDLE_TIMEOUT` | No | `60s` | positive duration | Go duration syntax |
| `HTTP_SHUTDOWN_TIMEOUT` | No | `10s` | positive duration | Bounds graceful shutdown |
| `DB_HOST` | **Yes** | — | any non-empty string | |
| `DB_PORT` | No | `5432` | `1`–`65535` | |
| `DB_NAME` | **Yes** | — | any non-empty string | |
| `DB_USER` | **Yes** | — | any non-empty string | |
| `DB_PASSWORD` | **Yes** | — | any non-empty string | **Secret**; never logged |
| `DB_SSLMODE` | No | `disable` | `disable`, `require`, `verify-ca`, `verify-full` | Production MUST use `require` or stricter |
| `DB_MAX_CONNS` | No | `10` | integer > 0 | |
| `DB_MIN_CONNS` | No | `0` | integer >= 0 and <= `DB_MAX_CONNS` | |
| `DB_MAX_CONN_LIFETIME` | No | `1h` | positive duration | |
| `DB_MAX_CONN_IDLE_TIME` | No | `30m` | positive duration | |
| `DB_CONNECT_TIMEOUT` | No | `5s` | positive duration | Bounds startup ping |
| `AWS_REGION` | No | `us-east-1` | any non-empty region | |
| `AWS_ENDPOINT_URL` | No | empty | absolute URL | Optional LocalStack or test endpoint override |
| `CORS_ALLOWED_ORIGINS` | No | empty | comma-separated absolute origins | Empty denies all cross-origin requests |

## AWS Credentials

- The application uses the AWS SDK default credential provider chain. It does not define custom
  credential variables.
- IAM roles and temporary credentials are supported through the default chain.
- No credentials are required to start the service. AWS clients are not created at startup.
- For local development without AWS, set `AWS_EC2_METADATA_DISABLED=true` to prevent instance
  metadata discovery.

## Example `.env` (local only)

`.env.example` is a template. Copy it to `.env` and let the Makefile or Docker Compose load it. Never
commit `.env`.

```dotenv
APP_NAME=tycoon-backend
APP_ENV=development
LOG_LEVEL=debug

HTTP_HOST=0.0.0.0
HTTP_PORT=8080
HTTP_READ_TIMEOUT=5s
HTTP_WRITE_TIMEOUT=10s
HTTP_IDLE_TIMEOUT=60s
HTTP_SHUTDOWN_TIMEOUT=10s

DB_HOST=localhost
DB_PORT=5432
DB_NAME=tycoon
DB_USER=tycoon
DB_PASSWORD=local-development-only
DB_SSLMODE=disable
DB_MAX_CONNS=10
DB_MIN_CONNS=0
DB_MAX_CONN_LIFETIME=1h
DB_MAX_CONN_IDLE_TIME=30m
DB_CONNECT_TIMEOUT=5s

AWS_REGION=us-east-1
AWS_ENDPOINT_URL=
AWS_EC2_METADATA_DISABLED=true

CORS_ALLOWED_ORIGINS=
```

## Failure Behavior

| Condition | Behavior |
|-----------|----------|
| Required variable missing or empty | Startup fails; error names the setting |
| Invalid enum, port, URL, or duration | Startup fails; error names the setting |
| `DB_MIN_CONNS > DB_MAX_CONNS` | Startup fails |
| Database unreachable at startup | Startup fails within `DB_CONNECT_TIMEOUT` |
| Database becomes unreachable after startup | Graceful shutdown, then non-zero exit |
| CORS origin not listed | No permissive cross-origin headers |

Errors MUST NOT include secret values. `DB_PASSWORD` MUST never appear in logs or responses.
