# Client Integration Contract: Firebase SSO

This contract is the integration surface for Guest Mobile and Fleet Console. Client source changes live in their own repositories; this document and [auth-api.yaml](./auth-api.yaml) are the deliverables from the backend feature.

## 1. Non-negotiable rules

1. Sign in through Firebase Authentication. Never send Firebase passwords or provider credentials to the backend.
2. Exchange the Firebase ID Token only to establish or re-establish a backend session. Do not exchange it on every API request.
3. Send the **backend access token** as `Authorization: Bearer <access_token>` on protected APIs. Never send a Firebase ID Token as an access token.
4. Persist the backend **refresh token** only in platform-secure storage. Never put any token in a URL, query string, log, analytics event, crash report, or plain browser storage.
5. The backend access token is short-lived (at most 15 minutes). Refresh before expiry or after a `401`, using the single-use refresh token.
6. Never retry a refresh with the same token after an ambiguous network failure without first reconciling state. The client must serialize refresh per session.
7. Backend logout revokes backend sessions only. Firebase sign-out is a separate client action.
8. Fleet Console authority always comes from a pre-existing backend administrator grant. Firebase custom claims, an email match, or a Firebase sign-in never grant Fleet Console access.
9. Hub clients do not use this contract. Firebase ID Tokens and backend tokens are cloud-only and must not be synchronized to the Hub.

## 2. Session model

- One Firebase identity maps to one stable internal `account_id`.
- `client` selects the requested session class:
  - `guest_mobile` for Guest Mobile.
  - `fleet_console` for Fleet Console.
- Sessions for the same account and different devices/clients are independent refresh-token families. Revoking one does not revoke the others unless the user calls logout-all.
- Every successful refresh rotates the refresh token. The old token becomes unusable immediately.
- Reuse of an already-rotated refresh token revokes the entire session family. The client must then require a fresh Firebase sign-in.

## 3. Guest Mobile flow

### 3.1 Initial sign-in and exchange

1. Sign in with Firebase (Google, Apple, email, or another configured Firebase method).
2. Obtain a fresh Firebase ID Token from the Firebase SDK.
3. Call:

```http
POST /v1/auth/firebase/exchange
Content-Type: application/json
```

```json
{
  "firebase_id_token": "<firebase-id-token>",
  "client": "guest_mobile"
}
```

`client` may be omitted and defaults to `guest_mobile`, but sending it explicitly is recommended.

4. On `200`, store:
   - `access_token` in memory or the shortest-lived secure store available.
   - `refresh_token` in platform-secure storage.
   - `session.session_id`, `session.account_id`, `session.access_expires_at`, and `session.refresh_expires_at`.
5. Discard the Firebase ID Token after the exchange. Do not persist it as a session credential.
6. Call guest APIs with `Authorization: Bearer <access_token>`.

### 3.2 Email collision and explicit linking

If the exchange returns `409` with code `identity_link_required`, the signed-in Firebase identity is not attached to the intended existing account and the backend will not merge or attach it by email.

The user must:

1. Establish an authenticated session on the existing account (for example, an already-signed-in device or another linked sign-in method).
2. Obtain a **fresh** Firebase ID Token for the identity to link.
3. Call:

```http
POST /v1/auth/identities/link
Authorization: Bearer <existing-account-access-token>
Content-Type: application/json
```

```json
{"firebase_id_token": "<fresh-firebase-id-token>"}
```

4. On `200`, the identity is linked. The account ID remains unchanged. Reuse the existing session; do not create a new account.
5. Repeating the same link for the same account is idempotent (`200`). A `409 identity_conflict` means the identity is already linked elsewhere; do not retry or attempt to merge.

Unlinking:

```http
DELETE /v1/auth/identities/{identityId}
Authorization: Bearer <access-token>
```

- `204` means the identity was removed.
- `409 last_identity` means the account would have no usable sign-in identity; keep the identity and offer another sign-in method first.
- `404` means the identity is not owned by this account; do not disclose or retry.

### 3.3 Refresh

Refresh proactively when `access_expires_at` is near, or reactively on a protected-API `401`:

```http
POST /v1/auth/refresh
Content-Type: application/json
```

```json
{"refresh_token": "<current-refresh-token>"}
```

On success, replace **both** the access token and refresh token atomically in local state. Never continue using the old refresh token.

Refresh serialization:

- Keep one refresh operation in flight per session/device.
- If a second caller needs a token while refresh is in flight, wait for the result.
- On `401`, do not retry the same refresh token; require Firebase sign-in and a new exchange.
- On `429`, wait at least `Retry-After` seconds before retrying.
- On `503`, fail safely: show a temporary unavailable state, keep secure tokens only if policy allows, and retry with backoff. Do not fall back to an unauthenticated session.

### 3.4 Logout

```http
POST /v1/auth/logout
Content-Type: application/json
```

```json
{"refresh_token": "<current-refresh-token>"}
```

Always clear local tokens after the attempt, regardless of whether the token was already expired or revoked. The endpoint is idempotent (`204`). Separately call the Firebase SDK sign-out if the user chose to sign out of Firebase; the backend does not do it.

### 3.5 Logout all devices

```http
POST /v1/auth/logout-all
Authorization: Bearer <access-token>
```

On `204`, clear local tokens and require a new Firebase exchange on every device. The user may still be signed in to Firebase until the client calls Firebase sign-out.

## 4. Fleet Console flow

1. Sign in through Firebase.
2. Exchange with the Fleet Console session class:

```json
{
  "firebase_id_token": "<firebase-id-token>",
  "client": "fleet_console"
}
```

3. Fleet Console access is granted only when the internal account has an active backend administrator grant with role, tenant, and ship scope.
4. If exchange returns `403`, do not retry with a guest session and do not attempt to elevate using Firebase custom claims. Show an access-denied message and direct the user to an administrator.
5. Use the returned access token for Fleet Console APIs. The backend re-resolves role, tenant, and ship scope on every request; a change or suspension takes effect without waiting for token expiry.
6. Fleet Console and Guest Mobile sessions are separate refresh families. Logging out of one does not revoke the other.

## 5. Error handling summary

| Status | Code | Client behavior |
|---|---|---|
| `400` | shared validation codes | Fix the request; do not retry unchanged |
| `401` | `unauthorized` | For exchange: restart Firebase sign-in. For refresh: require a new exchange/sign-in; never replay the refresh token. |
| `403` | `forbidden` | No authority for this client/operation; do not retry or use custom claims |
| `404` | `not_found` | Identity not owned or not found; do not disclose |
| `409` | `identity_link_required` | Prompt explicit link from an authenticated existing account |
| `409` | `identity_conflict` | Identity already linked elsewhere; do not merge or retry |
| `409` | `last_identity` | Keep the identity; add another sign-in method first |
| `422` | validation | Fix input |
| `429` | `rate_limited` | Wait at least `Retry-After` seconds, then retry with jitter |
| `503` | `auth_unavailable` | Fail closed; show temporary unavailable and retry with backoff |

Do not infer account existence from `403`, `404`, or `409` messages. UI copy must be generic.

## 6. Secure storage guidance

- iOS: Keychain with device-only accessibility appropriate to the app's background needs.
- Android: Android Keystore-backed encrypted storage.
- Web/Fleet Console: do not store refresh tokens in `localStorage` or `sessionStorage`. Prefer a trusted backend-for-frontend with a Secure, HttpOnly, SameSite cookie, or an equivalent security-reviewed mechanism. The access token should remain in memory.
- Clear refresh tokens on explicit logout, logout-all, refresh family revocation, account suspension, and app uninstall.
- Never log or attach token values to telemetry. Redact `Authorization`, `refresh_token`, and `firebase_id_token` fields.

## 7. Client acceptance checklist

- [ ] Firebase sign-in succeeds and produces an ID Token without backend credentials.
- [ ] First exchange returns a backend access token, refresh token, expiry, and session ID.
- [ ] Returning exchange resolves the same `account_id`.
- [ ] Protected API calls use only the backend access token.
- [ ] Refresh rotates both tokens and never reuses the old refresh token.
- [ ] A `401` refresh clears the session and returns the user to Firebase sign-in.
- [ ] `409 identity_link_required` opens the explicit link flow for an authenticated existing account.
- [ ] Duplicate link is idempotent; duplicate identity conflict is not retried.
- [ ] Last-identity unlink is blocked and explained.
- [ ] Logout revokes the current session; logout-all revokes all sessions; Firebase sign-out is separate.
- [ ] Fleet Console without an active grant receives `403` and cannot elevate via claims.
- [ ] No token appears in URLs, logs, analytics, crash reports, or insecure storage.
- [ ] Hub/offline flows do not request or store Firebase/backend tokens.
