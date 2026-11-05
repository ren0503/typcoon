# Tasks: Firebase SSO with Backend-Owned Session Tokens

**Input**: Design documents from `specs/003-firebase-sso-auth/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Required by FR-088 through FR-093 and SC-018 through SC-021. Write test tasks first and confirm they fail before the corresponding implementation.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested independently after the shared foundation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: User story label (`US1` through `US7`); omitted for Setup, Foundational, and Polish
- Every task includes an exact file path or directory

## Path Conventions

- Single Go module at repository root; no `src/` wrapper.
- Auth module: `internal/auth/`; config: `internal/config/`; router: `internal/httpserver/`; composition: `internal/app/`; migrations: `db/migrations/`; OpenAPI: `openapi/openapi.yaml`; feature artifacts: `specs/003-firebase-sso-auth/`.
- Integration tests use the Go build tag `integration` and the existing testcontainers harness.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the dependencies and shared error codes required by all auth work.

- [X] T001 Add Firebase Admin SDK, `golang-jwt/jwt/v5`, `redis/go-redis/v9`, AWS KMS SDK, and `alicebob/miniredis/v2` test dependency to `go.mod`, then run `go mod tidy` so `go.sum` is consistent.
- [X] T002 [P] Add a Redis service with healthcheck to `docker-compose.yml` and document `AUTH_RATE_LIMIT_REDIS_URL=redis://localhost:6379/0` in the local service configuration.
- [X] T003 [P] Add `CodeRateLimited = "rate_limited"` and `CodeAuthUnavailable = "auth_unavailable"` plus helper constructors for 429/503 responses in `internal/apperrors/errors.go`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared schema, configuration, crypto, persistence, and routing primitives that MUST exist before any user story can be completed.

**Critical**: No user story implementation starts until this phase is complete.

- [X] T004 Create `db/migrations/00003_firebase_sso_auth.sql` with: `account_identities.provider` CHECK extended to `('google','apple','email_otp','firebase')`; nullable `admin_users.account_id uuid REFERENCES accounts(id)`; partial unique index `uq_admin_users_account` on `admin_users(account_id)` where `account_id IS NOT NULL AND deleted_at IS NULL`; and `refresh_tokens.client text NOT NULL DEFAULT 'guest_mobile' CHECK (client IN ('guest_mobile','fleet_console'))`. Include a reversible `+goose Down` section, then update `docs/schema.sql` and `db/schema.sql` identically.
- [X] T005 Extend `db/queries/auth.sql` with queries for identity lookup by `(provider, provider_sub)`, account lookup/status by ID, active admin grant by `account_id`, admin ship scope, refresh-token insert/lookup with `FOR UPDATE`, conditional mark-used, replacement linking, family revoke, account-wide revoke, and identity insert/delete by account. Run `make sqlc` and verify `internal/database/sqlc/` compiles.
- [X] T006 [P] Add `FirebaseConfig`, `TokenConfig`, `SigningConfig`, and `RateLimitConfig` fields to `internal/config/config.go` with bindings/defaults for project ID/credentials/emulator, issuer/audience, access TTL `15m`, refresh TTL `720h`, clock skew `60s`, signing `local|kms`, active KID, KMS key map, local key file/KID, limiter `memory|redis`, Redis URL/prefix, trusted-proxy flag, and default rate-limit values. Keep production-only fail-fast validation for T064.
- [X] T007 [P] Implement `Signer`, `KeySet`, local PEM/random RSA signing, and an AWS KMS `crypto.Signer` adapter in `internal/auth/signing.go`, including RS256 JWK marshaling without private material. Add `internal/auth/signing_test.go`.
- [X] T008 [P] Implement the project-owned `Verifier` interface and Firebase Admin SDK adapter in `internal/auth/firebase.go`, using `VerifyIDTokenAndCheckRevoked`, bounded timeouts, project ID, credentials source, and non-production emulator selection. Add `internal/auth/firebase_test.go`.
- [X] T009 [P] Implement access-token issuance and validation in `internal/auth/tokens.go` with claims `iss`, `aud`, `sub` (account UUID), `iat`, `exp`, `jti`, `typ="access"`, and `client`; enforce RS256 only, required issuer/audience/expiry, issued-at validation, configurable leeway defaulting to 60 seconds, and a maximum 15-minute lifetime. Add `internal/auth/tokens_test.go`.
- [X] T010 [P] Implement opaque refresh-token generation (32 random bytes, base64url), SHA-256 hashing, family helpers, and expiry helpers in `internal/auth/refresh.go`. Add `internal/auth/refresh_test.go` asserting raw tokens are never stored by the helper.
- [X] T011 [P] Implement the `RateLimiter` interface with in-memory and Redis fixed-window adapters in `internal/auth/ratelimit.go`, atomic increment/TTL behavior, `Retry-After` calculation, and fail-closed errors. Add `internal/auth/ratelimit_test.go` using `miniredis`.
- [X] T012 [P] Add reusable auth test fakes for Firebase verification, clock, RSA signing, rate limiting, and audit capture in `internal/testutil/auth.go`, usable by unit and integration tests without production credentials.
- [X] T013 Implement the transactional auth store in `internal/auth/store.go` over the generated sqlc queries, including account/identity creation, unique-conflict classification, refresh row locking, family revocation, account-wide revocation, and admin grant/ship lookup. Integration helpers may be added to `internal/testutil/auth.go`.
- [X] T014 Define the public `auth.Service` interface, dependency options, and constructor in `internal/auth/module.go`; inject verifier, signer/key set, rate limiter, clock, and audit appender. Start with compile-safe method stubs only.
- [X] T015 Implement the backend access-token `Provider` in `internal/auth/provider.go` so it validates the JWT, loads the active account, rejects suspended/deleted accounts, and returns `Principal{ ActorType: account }` for guest tokens. Keep the existing dev/deny providers unchanged.
- [X] T016 [P] Add auth audit action constants and a redaction-safe audit helper for `auth.identity.*`, `auth.session.*`, and `auth.admin.access_denied` in `internal/auth/audit.go`; never write tokens, hashes, Firebase tokens, email secrets, or signing material.
- [X] T017 Add `PublicAPIRoutes` to `RouterConfig` and mount public auth routes outside the global `Authenticate` middleware in `internal/httpserver/router.go`; add `Auth`/`RegisterPublicRoutes` to `internal/app/modules.go`; pass signer/verifier/limiter dependencies from `cmd/server/main.go`. Update production `BuildModules` callers only.
- [X] T018 Add public/protected auth handler skeletons and a shared `no-store` response helper in `internal/auth/handlers.go`, mounted under `/v1/auth`; protected handlers remain behind the existing bearer middleware.

**Checkpoint**: Schema, config, crypto, store, service interface, provider, and router seams compile; user stories can now be implemented in priority order.

---

## Phase 3: User Story 1 - Sign In Through Firebase and Receive a Backend Session (Priority: P1) — MVP

**Goal**: A guest exchanges a valid Firebase ID Token for a backend access token and rotating refresh token; returning identities resolve to the same account; first-time identities provision exactly one active guest account and one Firebase identity link.

**Independent Test**: With a fake or emulator verifier, exchange a valid token twice and confirm the same `account_id`; exchange a new `uid` and confirm exactly one account/link; submit invalid, expired, wrong-audience, wrong-issuer, and revoked tokens and confirm no rows or tokens are created.

### Tests for User Story 1

- [X] T019 [P] [US1] Add the exchange contract test in `internal/httpserver/auth_contract_integration_test.go` asserting `POST /v1/auth/firebase/exchange` is public, validates `FirebaseExchangeRequest`, returns `TokenResponse`, and sends `Cache-Control: no-store`.
- [X] T020 [P] [US1] Add exchange integration tests in `internal/auth/exchange_integration_test.go` for invalid/expired/wrong-audience/wrong-issuer/revoked tokens, first-time provisioning, returning-identity idempotency, and concurrent first exchanges creating one account/link.
- [X] T021 [P] [US1] Add exchange unit tests in `internal/auth/service_test.go` for default `client=guest_mobile`, verified email/profile handling, exactly one session issuance, no Firebase ID Token persistence, and generic authentication errors.

### Implementation for User Story 1

- [X] T022 [US1] Add `POST /auth/firebase/exchange`, `FirebaseExchangeRequest`, `TokenResponse`, `Session`, and shared auth error responses to `openapi/openapi.yaml`, including `401`, `409`, `422`, `429`, and `503`.
- [X] T023 [US1] Implement `Service.Exchange` in `internal/auth/service.go`: apply exchange IP/identity limits, verify the Firebase token, extract verified `uid`/email/display name, resolve `(firebase, uid)`, provision account plus identity atomically when unknown and conflict-free, reject suspended/deleted accounts, create a refresh family, issue an access token with the account UUID subject, and return session metadata.
- [X] T024 [US1] Implement exchange request validation and the HTTP handler in `internal/auth/dto.go` and `internal/auth/handlers.go`, returning the stable response fields and generic errors without token leakage.
- [X] T025 [US1] Mount the exchange route as a public `/v1/auth` route in `internal/auth/handlers.go` and `internal/httpserver/router.go`, and apply the no-store header to token responses.
- [X] T026 [US1] Audit successful guest provisioning and session issuance in `internal/auth/service.go` using the T016 helper, with actor/account context but no token or hash material.
- [X] T027 [US1] Add exchange response/error mapping in `internal/auth/handlers.go` so malformed/expired/revoked/unknown credentials return the same generic `401`, and add/verify no-secret logging in the exchange path.

**Checkpoint**: US1 is independently functional with a fake/emulator verifier; the MVP can be validated before refresh or Fleet work.

---

## Phase 4: User Story 2 - Maintain and Refresh a Backend Session Safely (Priority: P1)

**Goal**: A valid refresh token rotates exactly once into a new access/refresh pair; a replayed token revokes the entire family; concurrent refreshes never leave a partial rotation.

**Independent Test**: Exchange a session, refresh it, confirm the old refresh token fails, replay it, confirm the family is revoked, and run two concurrent refreshes to confirm at most one succeeds.

### Tests for User Story 2

- [X] T028 [P] [US2] Add the refresh contract test in `internal/httpserver/auth_contract_integration_test.go` for `POST /v1/auth/refresh`, `RefreshRequest`, `TokenResponse`, generic `401`, `429`, `503`, and no-store headers.
- [X] T029 [P] [US2] Add refresh integration tests in `internal/auth/refresh_integration_test.go` for successful rotation, old-token rejection, family revocation on replay, concurrent refresh with at most one success, expiry, suspended/deleted account rejection, and unknown-token generic errors.
- [X] T030 [P] [US2] Add refresh unit tests in `internal/auth/refresh_test.go` or `internal/auth/service_test.go` for hash lookup, used/revoked state transitions, replacement linking, and fixed-clock expiry behavior.

### Implementation for User Story 2

- [X] T031 [US2] Add `POST /auth/refresh` and its request/response/error documentation to `openapi/openapi.yaml`.
- [X] T032 [US2] Implement `Service.Refresh` in `internal/auth/service.go`: apply refresh IP/session limits, hash and lock the row, reject expired/revoked/unknown tokens generically, revoke the family on reuse, mark the presented row used with `replaced_by`, insert one replacement in the same family, re-check account status, and issue a new access token.
- [X] T033 [US2] Implement the refresh DTO and handler in `internal/auth/dto.go` and `internal/auth/handlers.go`, mounted as a public `/v1/auth/refresh` route.
- [X] T034 [US2] Wire refresh rate limits in `internal/auth/handlers.go` using IP and SHA-256(session token) keys; return `429 Retry-After` and `503 auth_unavailable` on limiter failure without issuing tokens.
- [X] T035 [US2] Audit successful refreshes and reuse detection in `internal/auth/service.go` without storing refresh hashes, families, or tokens.

**Checkpoint**: US1 and US2 together provide a complete guest sign-in and session-maintenance MVP.

---

## Phase 5: User Story 3 - Fleet Console SSO With Backend-Controlled Authorization (Priority: P1)

**Goal**: A Fleet Console user gets a backend session only through an active `admin_users` grant linked to the same internal account; role, tenant, and ship scope come from PostgreSQL and cannot be elevated by Firebase claims.

**Independent Test**: Seed an active account/identity/admin grant and confirm Fleet exchange and authorization respect tenant/ship scope; use a guest identity without a grant and confirm `403`; add administrator-like Firebase claims and confirm they are ignored.

### Tests for User Story 3

- [X] T036 [P] [US3] Add Fleet integration tests in `internal/auth/fleet_integration_test.go` for `client=fleet_console` with/without an active grant, guest-token denial on Fleet APIs, ignored Firebase custom claims, tenant/ship enforcement, and suspended/deleted account or grant denial on exchange and refresh.
- [X] T037 [P] [US3] Add provider/principal unit tests in `internal/auth/provider_test.go` for guest `ActorAccount`, Fleet `ActorAdmin`, role/tenant/ship mapping, missing/suspended grant rejection, and no privilege elevation from client input.

### Implementation for User Story 3

- [X] T038 [US3] Document Fleet Console `403`, `client=fleet_console`, and backend-grant-only authorization in `openapi/openapi.yaml` and `contracts/client-integration.md`.
- [X] T039 [US3] Implement active administrator-grant and ship-scope resolution helpers in `internal/auth/store.go`, returning role, tenant, and ship IDs and treating missing/suspended/deleted grants as unauthorized.
- [X] T040 [US3] Extend `Service.Exchange` in `internal/auth/service.go` for `client=fleet_console`: never auto-provision, require an existing active linked account/grant, return generic `403` without a grant, and set the session client to `fleet_console`.
- [X] T041 [US3] Extend `internal/auth/provider.go` so a Fleet access token resolves to `ActorAdmin` with backend role/tenant/ships on every request; guest tokens resolve only to `ActorAccount` and cannot call Fleet-authorized routes.
- [X] T042 [US3] Add Fleet grant checks to `Service.Refresh` in `internal/auth/service.go` so suspended/deleted grants cannot refresh, while guest sessions for the same account remain independently usable.
- [X] T043 [US3] Audit `auth.admin.access_denied` and Fleet session issuance in `internal/auth/service.go` without Firebase claim values or token material.

**Checkpoint**: US1-US3 provide Guest Mobile plus safe Fleet Console SSO with backend authorization.

---

## Phase 6: User Story 4 - End Sessions and Revoke Access (Priority: P2)

**Goal**: Logout revokes only the current refresh family; logout-all revokes every family for the authenticated account; both are idempotent and never call Firebase.

**Independent Test**: Create two sessions, log out one and confirm only it fails refresh; log out all and confirm every session for the account fails while another account remains unaffected; repeat both operations.

### Tests for User Story 4

- [X] T044 [P] [US4] Add logout/logout-all integration tests in `internal/auth/logout_integration_test.go` for family-only revocation, all-session revocation, other-account isolation, idempotency, unknown/expired token behavior, and no Firebase sign-out.
- [X] T045 [P] [US4] Add contract tests in `internal/httpserver/auth_contract_integration_test.go` for `POST /v1/auth/logout` (public, `204`) and `POST /v1/auth/logout-all` (bearer-protected, `204`).

### Implementation for User Story 4

- [X] T046 [US4] Add logout and logout-all paths, request bodies, `204`, and error responses to `openapi/openapi.yaml`.
- [X] T047 [US4] Implement `Service.Logout` in `internal/auth/service.go` in `internal/auth/service.go`: hash the refresh token, revoke its family, return success for well-formed unknown/expired/already-revoked tokens, and never reveal session existence.
- [X] T048 [US4] Implement `Service.LogoutAll` in `internal/auth/service.go` using the authenticated account subject to revoke every refresh family for that account idempotently.
- [X] T049 [US4] Implement logout/logout-all DTOs and handlers in `internal/auth/dto.go` and `internal/auth/handlers.go`, mounting logout publicly and logout-all behind the bearer middleware.
- [X] T050 [US4] Add logout/logout-all rate limits and audit actions in `internal/auth/handlers.go` and `internal/auth/service.go`; ensure no tokens or hashes appear in audit details.

**Checkpoint**: Session termination works independently and does not affect other accounts or clients beyond the intended scope.

---

## Phase 7: User Story 5 - Link Existing Accounts and Handle Identity Collisions Safely (Priority: P2)

**Goal**: A new Firebase identity with an email matching an existing account is never merged or auto-linked; an authenticated owner can explicitly link a freshly verified identity; unlink protects the last sign-in method.

**Independent Test**: Create an account with an email, exchange a new Firebase identity with the same email, confirm no merge/provision/link; link the identity through the authenticated endpoint and confirm the stable account ID; unlink the only identity and confirm rejection; attempt duplicate linking and confirm no mapping changes.

### Tests for User Story 5

- [X] T051 [P] [US5] Add identity integration tests in `internal/auth/identity_integration_test.go` for email collision (no merge, no account/link created, generic `409 identity_link_required`), explicit link, link idempotency, duplicate identity conflict, last-identity unlink rejection, concurrent link races, and audit rows for successful/rejected actions.
- [X] T052 [P] [US5] Add contract tests in `internal/httpserver/auth_contract_integration_test.go` for `POST /v1/auth/identities/link` and `DELETE /v1/auth/identities/{identityId}`, including `200`, `204`, `404`, `409`, `429`, and no-store behavior.
- [X] T053 [P] [US5] Add unit tests in `internal/auth/service_test.go` for collision classification, same-account idempotency, duplicate ownership, and last-credential counting using fakes.

### Implementation for User Story 5

- [X] T054 [US5] Add link/unlink paths, `IdentityLinkRequest`, `Identity`, `404`, and `409` documentation to `openapi/openapi.yaml`.
- [X] T055 [US5] Add the email-collision branch to `Service.Exchange` in `internal/auth/service.go`: only for `guest_mobile`, when the verified email matches a non-deleted account, create nothing and return generic `409 identity_link_required`; never resolve or merge by email.
- [X] T056 [US5] Implement `Service.LinkIdentity` in `internal/auth/service.go`: require the authenticated account, freshly verify the Firebase ID Token and revocation, insert `(firebase, uid)` atomically, return `200` for same-account repeat, return generic `409 identity_conflict` for another account, reject inactive accounts, and audit the acting principal/affected account.
- [X] T057 [US5] Implement `Service.UnlinkIdentity` in `internal/auth/service.go`: require an identity owned by the authenticated account, reject `last_identity` when no other usable identity remains, delete only the owned row, and audit success/rejection separately from rolled-back business writes.
- [X] T058 [US5] Implement link/unlink DTOs and protected handlers in `internal/auth/dto.go` and `internal/auth/handlers.go`, mounting `POST /v1/auth/identities/link` and `DELETE /v1/auth/identities/{identityId}` behind the bearer middleware.
- [X] T059 [US5] Enforce link/unlink limits of 10 requests/minute/account and add the required audit actions in `internal/auth/handlers.go` and `internal/auth/service.go`; ensure rejected-action audit never leaks whether another account exists.

**Checkpoint**: Collision handling and identity lifecycle are complete without automatic merges or account-takeover paths.

---

## Phase 8: User Story 6 - Operate Authentication Securely and Predictably (Priority: P2)

**Goal**: Production fails fast on missing Firebase/KMS/Redis configuration, signing keys rotate with JWKS overlap, abuse is limited across instances, and logs reveal no secrets.

**Independent Test**: Start with missing production Firebase/signing/Redis configuration and confirm startup failure; retrieve JWKS and verify a token; rotate the active key and confirm new tokens use it while old unexpired tokens verify; exceed limits and confirm `429 Retry-After`; simulate Redis failure and confirm fail-closed `503`.

### Tests for User Story 6

- [ ] T060 [P] [US6] Extend `internal/config/config_test.go` for production missing Firebase/KMS/Redis, emulator/local-key rejection, access TTL above 15 minutes, clock skew default `60s`, rate-limit values below documented minimums, and valid development/test defaults.
- [ ] T061 [P] [US6] Extend `internal/auth/ratelimit_test.go` for Redis/memory parity, exact default thresholds, shared counters across two limiter instances, `Retry-After`, and fail-closed errors when Redis is unavailable.
- [ ] T062 [P] [US6] Add signing/JWKS tests in `internal/auth/signing_test.go` and `internal/auth/handlers_test.go` for public-only JWK fields, active/retired key sets, new-token `kid`, and old-token verification during overlap.
- [ ] T063 [P] [US6] Add response/log redaction tests in `internal/httpserver/sensitive_redaction_integration_test.go` and `internal/auth/service_test.go` asserting no Firebase ID Token, access token, refresh token, hash, signing secret, or unnecessary personal data appears in logs/errors.

### Implementation for User Story 6

- [ ] T064 [US6] Complete production/staging validation in `internal/config/config.go`: require Firebase project/credentials, KMS signing and active KID, issuer/audience, Redis limiter, and defaults no weaker than exchange 10/min/IP + 30/min/identity, refresh 60/min/IP + 30/min/session, link/unlink 10/min/account, logout-all 5/min/account; reject emulator/local key/access TTL >15m.
- [ ] T065 [US6] Implement the public `GET /v1/auth/jwks` handler and route in `internal/auth/handlers.go` with public cache headers and `KeySet` output containing only RSA public parameters.
- [ ] T066 [US6] Implement configured active/retired KMS key handling and JWKS rotation behavior in `internal/auth/signing.go`; document the minimum overlap as the access-token lifetime and recommend 24 hours.
- [ ] T067 [US6] Wire Redis limiter selection, bounded timeouts, and fail-closed `503 auth_unavailable` behavior in `cmd/server/main.go`, `internal/auth/ratelimit.go`, and `internal/auth/handlers.go`.
- [ ] T068 [US6] Add bounded Firebase/KMS/Redis/DB timeouts and outcome-category logging/metrics in `internal/auth/service.go`, `internal/auth/firebase.go`, `internal/auth/signing.go`, and `internal/httpserver/middleware/auth.go` without token or credential values.
- [ ] T069 [US6] Add JWKS and complete `429`/`503` documentation to `openapi/openapi.yaml`, including `Retry-After` on throttling/unavailability.

**Checkpoint**: Auth is deployable with production configuration, rotation, redis-backed throttling, and safe observability.

---

## Phase 9: User Story 7 - Keep Hub Offline Authentication Independent (Priority: P3)

**Goal**: Cloud Firebase SSO introduces no Firebase, JWT, KMS, or backend-session dependency into the Hub/sync path and no auth tokens enter sync payloads.

**Independent Test**: Run the dependency-boundary and sync-payload tests; confirm `internal/sync` and shared sync queries do not import auth/Firebase/JWT/KMS packages and no snapshot/change-log payload contains auth token fields.

### Tests and Implementation for User Story 7

- [ ] T070 [P] [US7] Extend `internal/app/boundaries_test.go` to assert `internal/sync` and shared sync payload code do not import Firebase, JWT, KMS, or auth-token packages, and add a sync-payload assertion that no Firebase/access/refresh token fields are introduced.
- [ ] T071 [US7] Update `docs/system-architecture.md` and `specs/003-firebase-sso-auth/contracts/client-integration.md` to state that the Hub remains offline-only, no Hub auth code changes occur in this repository, and the actual offline Hub suite remains in the Hub repository per FR-074 through FR-078 and FR-102.

**Checkpoint**: The backend proves it adds no Hub dependency; the Hub's own offline suite remains external by scope.

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Complete contracts, documentation, stale-test updates, and full verification.

- [ ] T072 Update `internal/httpserver/route_inventory_test.go`, `internal/httpserver/contract_integration_test.go`, `internal/httpserver/sensitive_redaction_integration_test.go`, `internal/httpserver/idempotency_integration_test.go`, and `internal/httpserver/security_integration_test.go` for the new `BuildModules` signature, public/protected auth routes, and fake auth dependencies.
- [ ] T073 Merge all auth paths/components into `openapi/openapi.yaml` and validate with the existing kin-openapi contract tests; ensure every documented operation is mounted with the correct middleware.
- [ ] T074 Update `docs/system-architecture.md` for Firebase as the centralized cloud IdP, auth module HTTP surface, account/admin link, RSA/KMS/JWKS, Redis rate limits, and the replacement of direct OAuth/ADR-05.
- [ ] T075 Add ADR-10 to `docs/system-architecture.md` Section 21 documenting Firebase SSO superseding ADR-05 while preserving Hub offline auth.
- [ ] T076 Verify `docs/schema.sql` and `db/schema.sql` match migration `00003_firebase_sso_auth.sql`; run migration apply and re-run checks in integration tests and keep the 41-table ownership map unchanged.
- [ ] T077 Update `.env.example` with all Firebase, token, signing, KMS, rate-limit, Redis, and trusted-proxy variables plus development-only warnings.
- [ ] T078 Reconcile `specs/003-firebase-sso-auth/quickstart.md` with the final environment variable names and commands; keep the emulator/local-key flow free of production credentials.
- [ ] T079 Run `make fmt`, `make vet`, `make lint`, and `make test`; fix all failures including stale `BuildModules` call sites and ownership/boundary checks.
- [ ] T080 Run `make test-integration` and fix migration, concurrency, rotation, linking, Fleet authorization, rate-limit, JWKS, and contract failures.
- [ ] T081 Run `make sqlc` twice and confirm generated output is idempotent and matches the updated schema/query files.
- [ ] T082 Perform a final security review of `internal/auth/`, `internal/httpserver/middleware/auth.go`, and auth handlers: no tokens/hashes/credentials in logs, no token in URLs, no Firebase ID Token accepted as access token, and no `no-store` omission on token responses.
- [ ] T083 Execute the `quickstart.md` validation scenarios and record any deviations in the plan or quickstart before declaring the feature complete.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; can start immediately.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories.
- **US1 (Phase 3)**: Depends on Foundational; no dependency on other stories.
- **US2 (Phase 4)**: Depends on US1 because refresh consumes the session created by exchange.
- **US3 (Phase 5)**: Depends on US1 (exchange/session) and US2 (refresh grant checks); provider work can start after Foundational.
- **US4 (Phase 6)**: Depends on US1/Foundational; logout-all needs the authenticated principal from T015/T041.
- **US5 (Phase 7)**: Depends on US1; collision handling modifies the exchange flow, so sequence after US1 (and US3 for Fleet behavior stability).
- **US6 (Phase 8)**: Depends on Foundational and US1; can proceed in parallel with US2/US3 once signing and handlers exist.
- **US7 (Phase 9)**: Depends on Foundational; can proceed in parallel with US1-US6.
- **Polish (Phase 10)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **US1 (P1)**: Foundational only; MVP scope.
- **US2 (P1)**: US1 for session issuance; independently testable with a seeded refresh family.
- **US3 (P1)**: US1 for Fleet exchange, US2 for grant-aware refresh; independently testable with seeded admin rows.
- **US4 (P2)**: US1 for sessions; independently testable with seeded refresh families.
- **US5 (P2)**: US1 for exchange; independently testable with seeded accounts/identities.
- **US6 (P2)**: Foundational/US1 for signing and config; independently testable with fake Firebase/KMS/Redis.
- **US7 (P3)**: Foundational; no dependency on token behavior.

### Within Each User Story

- Write tests first; confirm they fail.
- Add OpenAPI/contract surface before or with handlers.
- Persistence helpers before service orchestration.
- Service orchestration before handlers/routes.
- Audit and rate limiting before declaring the story complete.
- Validate the story's independent test before moving to the next priority.

### Parallel Opportunities

- Setup: T002 and T003 can run in parallel after T001 starts.
- Foundational: T006, T007, T008, T009, T010, T011, T012, and T016 touch independent files and can run in parallel; T004 → T005 → T013 → T014 → T017 → T018 is the sequential spine.
- US1: T019, T020, T021 can run in parallel; then T022 → T023 → T024/T025 → T026/T027.
- US2: T028, T029, T030 can run in parallel; then T031 → T032 → T033/T034 → T035.
- US3: T036 and T037 can run in parallel; then T038 → T039/T040/T041/T042 → T043.
- US4: T044 and T045 can run in parallel; then T046 → T047/T048 → T049/T050.
- US5: T051, T052, T053 can run in parallel; then T054 → T055/T056/T057 → T058/T059.
- US6: T060, T061, T062, T063 can run in parallel; then T064/T065/T066/T067/T068/T069.
- US7: T070 can run while US1-US6 are in progress; T071 follows the final auth shape.
- Polish: T072/T073/T074/T075/T076/T077 can run in parallel after implementation; T078-T083 are verification-gated.

---

## Parallel Examples

### User Story 1

```text
T019 Contract test for exchange
T020 Integration tests for exchange/provisioning/concurrency
T021 Unit tests for exchange orchestration
```

### User Story 2

```text
T028 Refresh contract test
T029 Refresh integration tests
T030 Refresh unit tests
```

### User Story 3

```text
T036 Fleet authorization integration tests
T037 JWT provider/principal unit tests
```

### User Story 5

```text
T051 Identity collision/link/unlink integration tests
T052 Link/unlink contract tests
T053 Collision/link-state unit tests
```

### User Story 6

```text
T060 Production config tests
T061 Rate limiter tests
T062 JWKS/signing rotation tests
T063 Redaction/security tests
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 (Setup).
2. Complete Phase 2 (Foundational).
3. Complete Phase 3 (US1).
4. Stop and validate US1 independently with a fake/emulator verifier: first-time and returning exchange, no duplicate rows, invalid tokens create nothing.
5. Deploy/demo the guest exchange MVP.

### Incremental Delivery

1. Setup + Foundational → shared foundation ready.
2. US1 → guest exchange MVP.
3. US2 → safe refresh and reuse detection.
4. US3 → Fleet Console SSO with backend authorization.
5. US4 → logout/logout-all.
6. US5 → identity linking, collisions, last-credential protection.
7. US6 → production configuration, JWKS rotation, rate limits, observability.
8. US7 → Hub isolation proof.
9. Polish → contracts, docs, full test suite, quickstart validation.

### Parallel Team Strategy

After Foundational and US1:

- Developer A: US2 refresh and session rotation.
- Developer B: US3 Fleet Console authorization.
- Developer C: US4 logout and US5 identity lifecycle.
- Developer D: US6 operations/signing/rate limits.
- Developer E: US7 Hub boundary and documentation.

Stories must still pass their own independent tests before integration.

---

## Notes

- Tests are required, not optional, because FR-088 through FR-093 explicitly demand unit, integration, contract, concurrency, and regression coverage.
- [P] tasks touch different files and have no dependency on an incomplete task in the same phase.
- Every task references a concrete existing or planned path.
- Do not add client-app source, administrator-management APIs, or Hub code; those are out of scope per FR-102.
- Keep tokens, hashes, Firebase ID Tokens, signing secrets, and unnecessary personal data out of logs, audit details, errors, and URLs.
- Stop at each checkpoint to validate the user story independently.
