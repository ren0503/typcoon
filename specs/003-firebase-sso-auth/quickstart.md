# Quickstart: Firebase SSO with Backend-Owned Session Tokens

Validation guide for feature `003-firebase-sso-auth`. Implementation details live in `tasks.md` (created later); entity rules are in [data-model.md](./data-model.md); the API is in [contracts/auth-api.yaml](./contracts/auth-api.yaml); client behavior is in [contracts/client-integration.md](./contracts/client-integration.md).

## Prerequisites

- Go 1.26 (`go version`)
- Docker running (testcontainers + Docker Compose PostgreSQL/Redis)
- `jq` and `curl` for the manual smoke test
- Optional: Firebase CLI for the Auth emulator
- No AWS or production Firebase credentials are required for local development or tests

## Build and test commands

```sh
make fmt vet lint test        # unit, config, boundary, OpenAPI lint; no infrastructure
make test-integration         # testcontainers PostgreSQL + Redis/miniredis auth suites
make build                    # server + migrate binaries
```

The integration suite must apply migrations from an empty database and re-run them harmlessly. It uses fakes for Firebase/KMS and must never contact live Google, Apple, Firebase, or KMS services.

## Local run with the Firebase emulator and a local signing key

```sh
# 1. Start dependencies (Redis service is added for auth rate limits)
docker compose up -d postgres redis

# 2. Generate a development-only RSA signing key; never commit it
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 \
  -out /tmp/tycoon-auth-dev-key.pem
openssl pkey -in /tmp/tycoon-auth-dev-key.pem -pubout \
  -out /tmp/tycoon-auth-dev-key.pub.pem

# 3. Configure the service
cp .env.example .env
# Edit .env; minimum additions for local SSO:
#   APP_ENV=development
#   AUTH_MODE=jwt
#   AUTH_FIREBASE_PROJECT_ID=demo-tycoon
#   FIREBASE_AUTH_EMULATOR_HOST=localhost:9099
#   AUTH_TOKEN_ISSUER=http://localhost:8080
#   AUTH_TOKEN_AUDIENCE=tycoon-api
#   AUTH_SIGNING_MODE=local
#   AUTH_SIGNING_ACTIVE_KID=local-dev-1
#   AUTH_SIGNING_LOCAL_KID=local-dev-1
#   AUTH_SIGNING_LOCAL_KEY_FILE=/tmp/tycoon-auth-dev-key.pem
#   AUTH_RATE_LIMIT_REDIS_URL=redis://localhost:6379/0
#   AUTH_ACCESS_TTL=15m
#   AUTH_REFRESH_TTL=720h
#   AUTH_CLOCK_SKEW=60s
#   AUTH_TRUST_PROXY=false

# 4. Apply the migration (separate deployment step; never at instance startup)
make migrate-up

# 5. Start the API
make run
```

Expected startup behavior:

- Every environment rejects missing Redis configuration with a clear error before serving.
- Production/staging rejects local signing, emulator mode, missing Firebase credentials, or missing KMS configuration before serving.
- Development/test accepts the emulator and local key only when `APP_ENV` is `development` or `test`.
- `GET /healthz` still works; `/v1/auth/jwks` returns public keys; protected `/v1` routes still reject requests without a backend access token.

## Manual smoke test (emulator)

```sh
# Terminal A: Firebase Auth emulator
firebase emulators:start --only auth --project demo-tycoon

# Terminal B: create a test Firebase user and capture its ID Token
ID_TOKEN=$(curl -s -X POST \
  'http://localhost:9099/identitytoolkit.googleapis.com/v1/accounts:signUp?key=fake-api-key' \
  -H 'Content-Type: application/json' \
  -d '{"email":"guest@example.com","password":"local-development-only","returnSecureToken":true}' \
  | jq -r '.idToken')

# Exchange it for a backend session
SESSION=$(curl -s -X POST 'http://localhost:8080/v1/auth/firebase/exchange' \
  -H 'Content-Type: application/json' \
  -d "{\"firebase_id_token\":\"$ID_TOKEN\",\"client\":\"guest_mobile\"}")

ACCESS=$(printf '%s' "$SESSION" | jq -r '.access_token')
REFRESH=$(printf '%s' "$SESSION" | jq -r '.refresh_token')
printf '%s\n' "$SESSION" | jq '{token_type, expires_in, session}'

# Use only the backend access token on protected APIs
curl -s -H "Authorization: Bearer $ACCESS" 'http://localhost:8080/v1/ships'

# Public JWKS (must contain no private material)
curl -s 'http://localhost:8080/v1/auth/jwks' | jq '.keys[0] | {kty, kid, use, alg, n, e}'

# Rotate the refresh token
ROTATED=$(curl -s -X POST 'http://localhost:8080/v1/auth/refresh' \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$REFRESH\"}")
NEW_REFRESH=$(printf '%s' "$ROTATED" | jq -r '.refresh_token')

# Replaying the old token must fail generically and revoke the family
curl -s -o /dev/null -w '%{http_code}\n' -X POST 'http://localhost:8080/v1/auth/refresh' \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$REFRESH\"}"   # expect 401

# The replacement must now also be unusable because the family was revoked
curl -s -o /dev/null -w '%{http_code}\n' -X POST 'http://localhost:8080/v1/auth/refresh' \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$NEW_REFRESH\"}"  # expect 401
```

## Validation scenarios

### 1. Firebase exchange and guest provisioning (US1, SC-001–SC-006)

- Invalid signature, wrong issuer, wrong audience, expired, not-yet-valid, and revoked Firebase tokens are rejected with zero accounts, identities, sessions, or tokens created.
- A first-time valid guest identity creates exactly one active account and one Firebase identity link.
- Repeating the exchange returns the same `account_id` with no duplicate rows.
- Raw Firebase ID Tokens, backend tokens, and hashes never appear in responses, database columns other than the permitted hash, or logs.
- Concurrent first-time exchanges for one `uid` create one account/link; both requests resolve to it or one fails safely.

### 2. Refresh rotation and reuse (US2, SC-006–SC-009)

- Every refresh returns a new access token and refresh token; the prior refresh token becomes unusable.
- Replaying a rotated token returns a generic `401` and revokes the whole family.
- Two concurrent refreshes with the same token: at most one succeeds; the family is never partially rotated.
- Expired, revoked, malformed, and unknown tokens return the same safe `401` shape.
- Suspended/deleted accounts cannot refresh; tokens never appear in logs/errors.

### 3. Fleet Console authorization (US3, SC-010–SC-012)

Integration tests seed an account, identity, active `admin_users.account_id` grant, role, tenant, and ship access.

- A `fleet_console` exchange requires the active grant and returns backend-owned role/tenant/ship scope in the principal.
- A guest Firebase identity without a grant receives `403` and cannot gain access via Firebase custom claims.
- A `guest_mobile` token cannot call Fleet Console APIs even if the same account has an admin grant.
- Suspending/deleting the account or grant immediately denies exchange/refresh/authorization; no access survives beyond the bounded access-token lifetime.
- Tenant and ship boundaries are enforced on every Fleet request.

### 4. Logout and revocation (US4, SC-009)

- Logout revokes only the current family; a second device remains usable.
- Logout-all revokes every family for the account and no other account.
- Repeating either operation is idempotent and does not make a revoked token usable.
- Backend logout does not call Firebase; Firebase sign-out remains a client action.

### 5. Linking, collisions, and last-credential protection (US5, SC-005/SC-020)

- A new Firebase identity with an email matching an existing account is not merged or auto-linked; exchange returns `409 identity_link_required` and creates nothing.
- Authenticated link with a fresh Firebase ID Token attaches the identity to the existing account and keeps `account_id` stable.
- Linking an identity already linked elsewhere returns `409 identity_conflict` and changes nothing.
- Repeating the same link for the same account is idempotent (`200`).
- Unlinking the only identity returns `409 last_identity`; unlinking with another identity succeeds.
- Every successful and rejected link/unlink has an `audit_log` row with actor/account context and no token material.

### 6. Security, configuration, and operations (US6, SC-013–SC-018)

- `APP_ENV=production` fails fast when Firebase project/credentials, KMS signing, issuer/audience, or Redis limiter configuration is missing.
- `APP_ENV=production` rejects `FIREBASE_AUTH_EMULATOR_HOST`, `AUTH_SIGNING_MODE=local`, and access TTLs above 15 minutes.
- JWKS exposes only public `kid`, `n`, `e`; a token verifies with the published key and fails after tampering.
- Rotation: publish the new public key, activate it, verify new tokens use the new `kid`, and old unexpired tokens remain valid during the documented overlap.
- Rate limits return `429` with `Retry-After`; tests cover both IP and identity/session/account scopes and a shared Redis backend across two service instances.
- Redis/limiter unavailability fails closed with `503 auth_unavailable`; no token is issued.
- All auth responses are `Cache-Control: no-store`; JWKS is public-cacheable.
- Logs contain outcome categories only, never tokens, hashes, Firebase credentials, signing keys, or unnecessary personal data.

### 7. Hub isolation (US7, SC-015)

- A dependency-boundary test confirms the sync module does not import Firebase, JWT, KMS, or auth-token packages.
- No auth-token fields are added to snapshot, change-log, or sync payload definitions.
- The backend contains no Hub implementation; the actual offline boarding-pass/local-session suite remains in the Hub repository per FR-102. Document this boundary in the architecture update.

### 8. Contracts and documentation (SC-019–SC-021)

- `openapi/openapi.yaml` documents every auth operation, request/response, error, rate-limit response, non-cacheable header, and JWKS.
- Contract tests assert each documented operation is mounted with the correct public/protected middleware.
- The client integration contract covers Firebase sign-in, exchange, secure storage, access-token use, refresh/failure handling, link/unlink, logout, and the Hub exclusion; review confirms no undocumented backend behavior is required.

## Definition of done

- `make fmt vet lint test` and `make test-integration` pass locally and in CI.
- Migration applies cleanly to an empty database and re-runs harmlessly; `docs/schema.sql` and `db/schema.sql` match `00003_firebase_sso_auth.sql`.
- No production Firebase, Google, Apple, KMS, or email credentials are required by any test.
- OpenAPI contract tests pass and the generated/checked spec contains no token leakage.
- Architecture, schema, environment, and ADR-10 documentation are updated; no Guest Mobile, Fleet Console, or Hub source changes are made in this repository.
