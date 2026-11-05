# Research: 003-firebase-sso-auth

Phase 0 output. Every decision below resolves a plan or specification ambiguity; implementation must follow these decisions unless the spec is amended.

## R1. Firebase ID Token verification

**Decision**: Use the official Firebase Admin SDK (`firebase.google.com/go/v4`) behind a project-owned `Verifier` interface. Production and staging call `VerifyIDTokenAndCheckRevoked`; any verification, revocation-check, timeout, or SDK-unavailable error fails closed. Tests use a deterministic fake verifier; the Firebase Auth emulator is selectable only when `APP_ENV` is `development` or `test`.

**Rationale**: FR-002 mandates the official SDK, and FR-005 requires revocation checking with fail-closed behavior. The interface keeps unit tests free of Firebase and prevents provider-specific types from leaking into the auth service.

**Alternatives considered**: manual Firebase JWT/JWKS verification (explicitly forbidden by FR-002); treating revocation-check unavailability as success (violates FR-005); live Firebase credentials in tests (violates FR-010/FR-091).

## R2. Identity resolution, guest provisioning, and email collisions

**Decision**:
- Resolve only by `(provider='firebase', provider_sub=verified uid)`. Email is never a mapping key.
- A guest-client exchange for an unknown identity with no email collision creates exactly one `accounts` row and one `account_identities` row in one transaction.
- If the verified Firebase email matches a non-deleted account's email, the exchange does **not** auto-provision or auto-link. It returns a generic `409 identity_link_required`; the user must authenticate to the intended account and call the link endpoint. This is the narrow, documented exception to FR-016 required by FR-019/FR-020 and US5.
- Fleet Console exchanges never auto-provision. They require an existing identity link and an active administrator grant.
- Unique constraints `account_identities(provider, provider_sub)` and the lower-case account email index are the final race authority. On identity-insert `23505`, roll back and re-read the identity; on account-email `23505`, retry provisioning with `accounts.email = NULL`.
- A verified email is stored on the identity link only when Firebase marks it verified. The account email is set only for a conflict-free guest provisioning.

**Rationale**: This is the only interpretation that lets US5 link a colliding identity to the existing account without merging or attaching by email. Provisioning a shell account first would permanently consume the identity link and make the required explicit link impossible.

**Alternatives considered**: auto-provision with `email=NULL` on collision (creates an unusable duplicate and blocks later linking); merge by email (forbidden); reject all first-time identities (violates SC-001/SC-004).

## R3. Client-scoped sessions and authorization context

**Decision**: Add `refresh_tokens.client` with values `guest_mobile` and `fleet_console` (default `guest_mobile`). The exchange request accepts an optional `client`, defaulting to `guest_mobile`. Access tokens carry `client` and `sub` (internal account UUID), but no role or tenant authorization decisions. Every request re-resolves the account, and for `fleet_console` re-resolves the active administrator grant with role, tenant, and ship scope.

**Rationale**: FR-043 requires Guest and Fleet sessions for one account to be independently revocable; FR-049/FR-069/FR-070 require client-specific authorization. A `fleet_console` session without an active grant is rejected at exchange and refresh, while a guest session remains usable if an administrator grant is suspended. Re-resolving authorization from the database also satisfies FR-072 immediately, not merely at token expiry.

**Alternatives considered**: infer the client from `User-Agent` (untrusted and ambiguous); embed role/tenant/ship in the JWT only (stale authorization); one shared session for both clients (cannot revoke independently).

## R4. Administrator-to-account link

**Decision**: Migration `00003_firebase_sso_auth.sql` adds nullable `admin_users.account_id uuid REFERENCES accounts(id)` and a partial unique index on `account_id` where the row is not soft-deleted. Existing rows remain `NULL`. Existing administrators must be linked explicitly by an operational seed/runbook; the feature does not infer a link from email or `idp_subject`, and it adds no administrator-management API.

**Rationale**: FR-027/FR-028 require the same stable account ID but forbid automatic administrator access. A nullable column preserves all existing data and allows a future admin-management feature to populate the link safely.

**Alternatives considered**: backfilling by email (violates FR-019/FR-027); a separate join table (unnecessary for a one-to-one grant); making the column non-null in this migration (breaks existing rows).

## R5. Backend access tokens

**Decision**: Issue RS256 JWTs with claims `iss`, `aud`, `sub` (account UUID), `iat`, `exp`, `jti`, `typ="access"`, and `client`. Default lifetime is 15 minutes; configuration must reject values above 15 minutes. Validation uses `github.com/golang-jwt/jwt/v5` with RS256 as the only allowed algorithm, required issuer, required audience, required expiry, issued-at validation, and a configurable leeway defaulting to 60 seconds. Validation then loads account status and, for fleet sessions, the active administrator grant.

**Rationale**: FR-031 through FR-040 and FR-070/FR-072 are covered without trusting Firebase or client claims. Database-backed authorization on each request is slightly stronger than the spec's token-expiry bound.

**Alternatives considered**: hand-written JWT parsing (forbidden by constitution VI); HS256 (does not support public-key discovery); embedding role/tenant/ship and trusting them until expiry (stale after revocation); accepting any asymmetric algorithm (algorithm-confusion risk).

## R6. Signing keys, KMS, JWKS, and rotation

**Decision**: Put signing behind a `Signer`/`KeySet` interface. Production/staging use AWS KMS asymmetric keys (`RSASSA_PKCS1_V1_5_SHA_256`) through a `crypto.Signer` adapter; development/tests use an in-memory or PEM-backed RSA key. Local signing is rejected when `APP_ENV=production`. Publish public keys at `GET /v1/auth/jwks` with `kid`, `kty=RSA`, `alg=RS256`, `use=sig`, `n`, and `e`; never expose private material. Configure a `kid -> KMS key ARN` map plus an active `kid`. Rotation procedure: publish/add the new key, deploy it as non-active, then make it active; retain the old public key for at least the access-token lifetime (recommended 24-hour overlap) before removal.

**Rationale**: FR-034/FR-035/FR-038/FR-039 and SC-017 require asymmetric signing, managed private keys, public discovery, and overlap during rotation. KMS is already the approved AWS mechanism and is represented through an interface per constitution V.

**Alternatives considered**: local private key in production (forbidden); symmetric JWT signing (cannot publish safe verification keys); DB-stored private keys (expand compromise blast radius); no rotation overlap (breaks unexpired tokens).

## R7. Refresh tokens, rotation, and reuse detection

**Decision**: Generate 32 random bytes from `crypto/rand`, encode base64url (256-bit opaque credential), and store only SHA-256(token) in `refresh_tokens.token_hash`. Refresh tokens expire after 30 days by default. Rotate inside one transaction: lock the presented row with `FOR UPDATE`, reject expiry/revocation/account-inactive, conditionally set `used_at` and `replaced_by`, insert the replacement in the same `family_id`, and commit. A presented token with `used_at` or `revoked_at` set revokes the entire family and returns a generic 401. Concurrent refreshes serialize on the row lock; at most one rotates, and the later request is treated as reuse. Logout revokes the presented token's family; logout-all revokes every family for the account. Both are idempotent and return 204.

**Rationale**: FR-041 through FR-055, SC-006 through SC-009. SHA-256 is appropriate because the token already has 256 bits of entropy; password-style hashing is unnecessary and would add latency. The database row lock is the final concurrency authority.

**Alternatives considered**: storing raw tokens (forbidden); rotating outside a transaction (partial rotation); returning distinct errors for expired/revoked/unknown (FR-048/FR-064); enforcing concurrency with Redis locks only (not authoritative).

## R8. Rate limiting

**Decision**: Implement a shared rate limiter behind an interface with Redis and in-memory implementations. `AUTH_RATE_LIMIT_BACKEND=memory` is allowed only for development/test; staging/production require `redis`. Use Redis atomic fixed-window counters per scope/key with a TTL equal to the window plus one second. Defaults from the clarification are enforced as minimums: exchange 10/min/IP and 30/min/identity; refresh 60/min/IP and 30/min/session; link/unlink 10/min/account; logout-all 5/min/account. Return `429` with `Retry-After` seconds. A limiter error returns `503 auth_unavailable` and no tokens (fail closed). Identity/session keys use verified `uid` or SHA-256(refresh token), never raw tokens. Client IP comes from `RemoteAddr` unless a trusted-proxy mode is explicitly configured.

**Rationale**: FR-081/FR-082 and SC-014 require cross-instance protection on an Auto Scaling deployment. Redis is the architecture-approved shared store. Fixed windows are simple, atomic, and adequate for abuse control; YAGNI rules out a more complex distributed token bucket.

**Alternatives considered**: per-instance memory in production (does not satisfy cross-instance limits); PostgreSQL counters (adds write load and a new table); failing open on Redis errors (forbidden); logging or keying by raw tokens (forbidden).

## R9. Configuration and fail-fast rules

**Decision**: Extend `AuthConfig` with Firebase, token, signing, trusted-proxy, and rate-limit settings. Production/staging require a Firebase project and credentials source, KMS signing, an active key ID, issuer/audience, a Redis URL, and rate-limit defaults no weaker than the spec. Production rejects emulator mode and local key mode. Development/test may use emulator, a local PEM/random key, and the in-memory limiter. Durations and limits are validated at startup; secrets are never included in validation errors or logs. All Firebase, KMS, Redis, and database calls receive bounded timeouts.

**Rationale**: FR-008/FR-009/FR-080 and SC-016, while preserving FR-010/FR-091 local testability. It also enforces constitution V and VII.

**Alternatives considered**: lazy validation at first request (fails after serving starts); implicit development fallbacks in production (security risk); committing a shared dev key (secret leakage).

## R10. HTTP contract, route mounting, and errors

**Decision**: Mount `POST /v1/auth/firebase/exchange`, `POST /v1/auth/refresh`, `POST /v1/auth/logout`, and `GET /v1/auth/jwks` as public routes outside the global `/v1` authentication middleware. Mount `POST /v1/auth/logout-all`, `POST /v1/auth/identities/link`, and `DELETE /v1/auth/identities/{identityId}` as protected routes. Return `Cache-Control: no-store` on all token-bearing/auth responses; never put tokens in URLs. Use the shared error envelope. Invalid exchange/refresh credentials return one generic `401`; link conflicts return a non-disclosing `409`; throttling returns `429` with `Retry-After`; limiter/signing/Firebase unavailability returns `503 auth_unavailable`. Add `rate_limited` and `auth_unavailable` error codes.

**Rationale**: FR-056 through FR-066, FR-095 through FR-100, SC-019. Separating public and protected route registration is the smallest change to the current global-auth router.

**Alternatives considered**: matching paths inside the auth middleware allowlist (easy to misconfigure); putting the Firebase token in a query parameter (leaks through logs/caches); exposing internal SQL tables (forbidden by FR-044).

## R11. Audit events

**Decision**: Emit these audit actions: `auth.identity.provisioned`, `auth.identity.linked`, `auth.identity.unlinked`, `auth.identity.link_rejected`, `auth.identity.unlink_rejected`, `auth.session.issued`, `auth.session.refreshed`, `auth.session.reuse_detected`, `auth.session.revoked`, `auth.session.revoked_all`, `auth.admin.access_denied`. Successful mutations audit inside the business transaction; rejected link/unlink actions audit in a short separate transaction after the conflict is determined. Never include tokens, hashes, Firebase ID Tokens, passwords, or provider secrets. Invalid-token floods are observed through metrics/log categories rather than per-request audit rows.

**Rationale**: FR-024/FR-073/FR-083 and SC-020. Same-transaction audit preserves the constitution's "missing audit means no write" rule; separate rejected-action audit ensures security-relevant denials are visible without rolling back the user-visible error.

**Alternatives considered**: auditing every invalid Firebase token (audit-store DoS); logging token fragments (leakage); best-effort audit for successful writes (silent loss).

## R12. Module boundary and composition

**Decision**: Keep `internal/auth` as the owning module and add `service.go`, `firebase.go`, `tokens.go`, `signing.go`, `refresh.go`, `ratelimit.go`, `handlers.go`, and `dto.go` there. `auth.Service` is the public interface; verifier, signer, limiter, clock, and audit appender are injected through `auth.Options`. `internal/app` composes them from config. `Modules` gains `Auth auth.Service`, `RegisterPublicRoutes`, and protected auth routes via `RegisterRoutes`. `RouterConfig` gains `PublicAPIRoutes`. Existing dev-header provider remains for non-production development and tests.

**Rationale**: Constitution I and the existing module pattern. No new module or service boundary is introduced; auth simply becomes an HTTP-exposing module.

**Alternatives considered**: a separate `identity` module (splits owned tables across modules); auth handlers in `httpserver` (breaks table-route ownership); global mutable singletons (untestable).

## R13. Test strategy

**Decision**: Use fakes for Firebase verification, a fixed clock, an in-memory RSA signer, an in-memory/`miniredis` limiter, and real PostgreSQL testcontainers. Cover: invalid/expired/wrong-audience/wrong-issuer/revoked Firebase tokens; returning identity; first-time provisioning; concurrent first exchanges; email collision no-merge/no-provision; fleet no-grant denial; admin grant authorization; access-token claims/validation; concurrent refresh with exactly one rotation; replay family revocation; logout/logout-all idempotency; suspended account/admin; link/unlink duplicate/last-credential/audit; rate limits and fail-closed behavior; config fail-fast; migration apply/re-run; mounted contract operations and response validation. No production Firebase, Google, Apple, KMS, or email credentials.

**Rationale**: FR-088 through FR-093 and SC-018/SC-019. Real PostgreSQL is required for unique indexes, row locks, and concurrency; fakes are required for deterministic external behavior.

**Alternatives considered**: mocking the database (misses race semantics); hitting live Firebase/KMS (non-hermetic); testing only handlers (misses service invariants).

## R14. Hub isolation and documentation

**Decision**: This repository contains no Hub implementation; the Hub's offline auth suite remains in the Hub repository. In this repository, add a dependency-boundary test proving that `internal/sync` and shared sync payload/query code do not import Firebase, JWT, KMS, or auth token packages, and verify that no auth-token fields are added to snapshot/change-log payloads. Update `docs/system-architecture.md` (auth ownership, sessions, Redis rate limiting, Hub boundary, ADR-10 superseding ADR-05), `docs/schema.sql`, `db/schema.sql`, `openapi/openapi.yaml`, `.env.example`, and the client integration contract. Client app source changes remain out of scope per FR-102.

**Rationale**: FR-074 through FR-078, FR-094, FR-101/FR-102, and SC-015/SC-021. The backend can prove it adds no Hub dependency; the actual offline Hub test can only run in the Hub repository.

**Alternatives considered**: claiming a Hub test exists in this repo (false); omitting documentation updates (violates constitution IX); modifying client repositories (out of scope).
