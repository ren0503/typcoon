# Data Model: Firebase SSO with Backend-Owned Session Tokens

This document records the business view of authentication entities, the minimal schema migration, and the invariants the service layer and PostgreSQL must enforce. Column-level truth lives in `docs/schema.sql` and `db/schema.sql`; migration `00003_firebase_sso_auth.sql` delivers the changes.

## Conventions

- The feature adds **no new tables**. The existing 41-table baseline and ownership map remain valid; only `account_identities`, `admin_users`, and `refresh_tokens` change.
- UUID primary keys remain application-generated UUID v7 where rows are created by the app; DB defaults remain fallbacks.
- Existing text + `CHECK` status conventions apply. New values are added by dropping and recreating the named CHECK constraint in the migration, never by weakening validation.
- `accounts`, `account_identities`, `admin_users`, `admin_ship_access`, and `refresh_tokens` remain owned by module `auth`. No other module writes them.
- Raw Firebase ID Tokens, backend access tokens, refresh tokens, token hashes, signing keys, and provider secrets are never persisted in business tables or emitted to logs.
- Times are UTC timestamps. Authentication compares against the database clock for persisted expiry and against a configurable clock-skew allowance for JWT claims.

## Entities

### Firebase Identity (`account_identities`)

| Field | Meaning | Rules |
|---|---|---|
| `id` | Stable identity-link ID | UUID; application-generated |
| `account_id` | Owning internal account | FK `accounts(id)`; never changed by exchange |
| `provider` | External provider | `firebase` is added to the CHECK values; existing `google`, `apple`, `email_otp` rows remain valid |
| `provider_sub` | Verified Firebase `uid` | Must be the verified token subject; unique with `provider` |
| `email` | Latest verified Firebase email, if verified | Nullable; never used to resolve or merge accounts |
| `created_at` | Link creation time | Immutable |

**Invariants**

- `(provider, provider_sub)` maps to at most one internal account (FR-013/FR-014).
- A returning identity resolves to the same account without creating another row (FR-015).
- Identity linking is explicit; no email-based resolution (FR-011/FR-019).
- Unlinking physically deletes only an `account_identities` row owned by the authenticated account, after last-credential protection (FR-023/FR-096). This table is not a synced business table.

### Internal Account (`accounts`)

| Field | Meaning | Rules |
|---|---|---|
| `id` | Stable internal subject | Used as access-token `sub`; never changes when identities are added or removed |
| `email` | Profile email | Case-insensitively unique among non-deleted rows; may be `NULL` |
| `display_name` | Profile display name | Optional; may be populated from verified Firebase profile data |
| `locale`, `phone` | Existing profile fields | Preserved; not authentication keys |
| `status` | Lifecycle | `active`, `suspended`, `deleted`; only `active` with `deleted_at IS NULL` may exchange, refresh, link, or unlink |
| `deleted_at` | Soft delete | Soft-deleted accounts are treated as absent to callers |

**Invariants**

- Every Firebase identity belongs to exactly one account (FR-027).
- Guest provisioning creates exactly one active guest account plus one identity link atomically when there is no email collision (FR-016).
- An unknown guest identity whose verified email matches another account is not provisioned or linked; the exchange returns a generic link-required outcome (FR-019/FR-020, US5).
- Account identifiers remain stable across link/unlink (FR-018).

### Administrator Authorization Grant (`admin_users`, `admin_ship_access`)

| Field | Meaning | Rules |
|---|---|---|
| `admin_users.account_id` | Same internal account used by Guest Mobile | Nullable FK `accounts(id)`; added by migration |
| `admin_users.role` | Fleet Console role | Existing CHECK values; backend-owned |
| `admin_users.tenant_id` | Tenant boundary | Existing FK; enforced on every request |
| `admin_users.status` | Grant status | `active` or `suspended`; only active grants authorize Fleet Console |
| `admin_users.deleted_at` | Grant soft delete | Soft-deleted grants do not authorize |
| `admin_ship_access` | Ship scope join | Existing rows; no ship rows means tenant-wide per existing `Principal` semantics |

**Invariants**

- At most one non-deleted administrator grant may reference the same account (partial unique index on `account_id`).
- A Fleet Console session requires an existing active grant; it is never auto-provisioned from Firebase (FR-027/FR-028).
- Grant role, tenant, and ship scope are read from PostgreSQL on each request (FR-070/FR-072).
- Existing administrator rows keep `account_id = NULL`; linking them is an explicit operational action, not an email-based backfill.
- Suspending/deleting the grant blocks new Fleet exchanges/refreshes and removes Fleet authority on the next request.

### Backend Session / Refresh Token Family (`refresh_tokens`)

| Field | Meaning | Rules |
|---|---|---|
| `id` | Individual refresh-token row | UUID; application-generated |
| `subject_type` | Subject kind | Always `account` for Firebase SSO |
| `subject_id` | Internal account UUID | Same stable account for both clients |
| `family_id` | Session identifier | Shared by every rotation in one session; independently revocable |
| `client` | Session class | New column: `guest_mobile` or `fleet_console`; default `guest_mobile` |
| `token_hash` | SHA-256 of the opaque refresh token | Unique; raw token never stored |
| `issued_at`, `expires_at` | Lifetime | Default 30 days; expired tokens are rejected |
| `used_at` | Rotation marker | Non-null means the token has been used and must not be accepted again |
| `replaced_by` | Next token ID | Links a rotated token to its replacement in the same family |
| `revoked_at` | Revocation marker | Set on logout, logout-all, reuse detection, or security revocation |
| `user_agent` | Minimal device metadata | Truncated; no IP, token, or unnecessary personal data |

**Invariants**

- Only the hash is stored; the raw refresh token is returned only in the response body (FR-042).
- A session is a token family; Guest and Fleet sessions for one account may be revoked independently (FR-043).
- Each successful refresh marks the presented row used and inserts exactly one replacement in the same family, in one transaction (FR-045/FR-047).
- Presenting an already-used or revoked token revokes the entire family and returns a generic authentication failure (FR-046).
- Expired, revoked, malformed, unknown, or inactive-subject tokens never issue a new access token (FR-048/FR-049).
- Logout revokes one family; logout-all revokes all families for the account; both are idempotent (FR-050/FR-051).
- Fleet sessions additionally require an active administrator grant at exchange and refresh.

### Access Token (ephemeral)

| Claim | Meaning | Rules |
|---|---|---|
| `iss` | Backend issuer | Required, configured |
| `aud` | Backend API audience | Required, configured |
| `sub` | Internal account UUID | Stable; never a Firebase `uid` |
| `iat` | Issued-at | Required; future values rejected with leeway |
| `exp` | Expiry | Required; at most 15 minutes after `iat` |
| `jti` | Unique token ID | Required; never used as a session identifier |
| `typ` | Token use | Must equal `access` |
| `client` | Session class | `guest_mobile` or `fleet_console`; selects the authorization path |

The token contains no role, tenant, ship assignment, Firebase token, provider token, refresh token, password, or personal data. Authorization context is resolved from backend records on each request.

### Signing Key (external configuration, not a table)

| Attribute | Meaning | Rules |
|---|---|---|
| `kid` | Key identifier | Unique; published in JWT header and JWKS |
| Private key | KMS key ARN in staging/production | Never leaves KMS; local PEM/in-memory only outside production |
| Public key | Published JWK | RSA `n`/`e`; no private material |
| Active flag | Current signing key | New tokens use the active `kid` |
| Retired keys | Previous public keys | Retained for at least the access-token lifetime during rotation |

No private key is stored in PostgreSQL, source control, or logs. JWKS is served from configured key material at `GET /v1/auth/jwks`.

### Rate-Limit Counter (ephemeral, Redis or memory)

| Attribute | Meaning | Rules |
|---|---|---|
| Scope/key | IP, identity, session hash, or account | Never a raw token or unnecessary personal data |
| Window | One-minute bucket | TTL expires automatically |
| Count | Requests in the window | Atomic increment; shared in staging/production |
| Retry-After | Seconds until reset | Returned with `429` |

Rate-limit state is intentionally not part of the database schema. Redis is the shared implementation for deployed environments; the in-memory implementation is limited to development/test.

## Migration `00003_firebase_sso_auth.sql`

The migration is intentionally minimal and reversible:

```sql
-- +goose Up
ALTER TABLE account_identities
  DROP CONSTRAINT account_identities_provider_check;
ALTER TABLE account_identities
  ADD CONSTRAINT account_identities_provider_check
  CHECK (provider IN ('google','apple','email_otp','firebase'));

ALTER TABLE admin_users
  ADD COLUMN account_id uuid REFERENCES accounts(id);
CREATE UNIQUE INDEX uq_admin_users_account
  ON admin_users (account_id)
  WHERE account_id IS NOT NULL AND deleted_at IS NULL;

ALTER TABLE refresh_tokens
  ADD COLUMN client text NOT NULL DEFAULT 'guest_mobile'
  CHECK (client IN ('guest_mobile','fleet_console'));

-- +goose Down
ALTER TABLE refresh_tokens DROP COLUMN client;
DROP INDEX IF EXISTS uq_admin_users_account;
ALTER TABLE admin_users DROP COLUMN account_id;
ALTER TABLE account_identities
  DROP CONSTRAINT account_identities_provider_check;
ALTER TABLE account_identities
  ADD CONSTRAINT account_identities_provider_check
  CHECK (provider IN ('google','apple','email_otp'));
```

Implementation must verify the actual generated constraint name in PostgreSQL and keep `docs/schema.sql` plus `db/schema.sql` in sync with this migration. No new table is added, so the 41-table ownership map and schema-parity expectations remain unchanged.

## Operations and State Transitions

### Exchange (`POST /v1/auth/firebase/exchange`)

1. Apply IP rate limit; decode the request; verify the Firebase ID Token, including revocation check; apply identity rate limit.
2. Extract verified `uid`, verified email (only if `email_verified`), and display name. Never trust client-supplied identity fields.
3. Resolve `(firebase, uid)`.
   - Found: load account; require `active` and not deleted. For `fleet_console`, require an active administrator grant.
   - Not found, `guest_mobile`, no email collision: provision account + identity atomically; set account email only when verified and conflict-free; audit provisioning.
   - Not found, `guest_mobile`, email collision: return generic `409 identity_link_required`; create nothing.
   - Not found or unlinked, `fleet_console`: return generic `403`; create no account.
4. Create a new refresh family (`client` + `user_agent`) and issue an access token with `sub = account_id` and `client`.
5. Return tokens and session metadata with `Cache-Control: no-store`. Audit session issuance without token material.

### Refresh (`POST /v1/auth/refresh`)

1. Apply IP rate limit; hash the presented token; look up the row.
2. If unknown, expired, revoked, or already used, return a generic `401`. If used/revoked, atomically revoke the whole family first.
3. Lock the row, re-check lifecycle and account/grant status, mark it used with `replaced_by`, insert exactly one replacement in the same family, and commit.
4. Issue a new access token for the same account and client. A second concurrent request sees `used_at` and triggers family revocation.

Session state: `active` → `rotated` (used and replaced) → `revoked` (family); any reuse of a rotated token moves the family to `revoked`.

### Logout and Logout-All

- Logout accepts a refresh token, revokes its family, and returns `204`; unknown/expired/already-revoked tokens still return `204` for a well-formed request.
- Logout-all requires a valid access token, revokes every refresh family for `sub`, and returns `204`; repeating it has no further effect.
- Neither operation calls Firebase or revokes Firebase credentials (FR-052).

### Link and Unlink

- Link requires a valid backend access token and a freshly verified Firebase ID Token.
  - Same identity already linked to the same account: `200` idempotent.
  - Identity linked to another account: generic `409`, no change, rejected action audited.
  - Account suspended/deleted: authentication/authorization failure.
  - Success: insert `(firebase, uid)` for `sub`, audit the acting principal and account.
- Unlink requires a valid backend access token and an identity owned by that account.
  - Not owned: generic `404`.
  - Last usable identity: `409 last_identity`, no change, rejected action audited.
  - Success: delete the identity row, audit the acting principal and account.

## Authorization Model

- `guest_mobile` tokens resolve to `Principal{ActorType: account, ActorID: account_id}` with no tenant authority.
- `fleet_console` tokens resolve to `Principal{ActorType: admin, ActorID: admin_users.id, TenantID, Role, ShipIDs}` only when the linked account and grant are active.
- Every request re-reads account/grant state, so suspension, deletion, role change, or ship-scope removal is enforced immediately (stronger than FR-072's token-expiry bound).
- Firebase custom claims are never copied into `Principal` and cannot grant tenant, role, or ship authority.
- A guest token cannot become a Fleet token by changing client input; the client field selects a requested session class, and Fleet authorization still requires the backend grant.
