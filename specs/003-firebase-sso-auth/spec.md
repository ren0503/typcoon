# Feature Specification: Firebase SSO with Backend-Owned Session Tokens

**Feature Branch**: `003-firebase-sso-auth`

**Created**: 2026-11-05

**Status**: Draft

**Input**: User description: "Implement Firebase Authentication as the centralized identity provider for the Yacht Management System while keeping the backend responsible for issuing and managing its own access and refresh tokens. Firebase ID tokens are verified with the Firebase Admin SDK, Firebase identities are mapped to internal accounts, and the backend issues short-lived access tokens plus rotating, revocable refresh tokens. The flow must support SSO across Guest Mobile and Fleet Console where appropriate. Hub authentication must remain separate and continue to work offline. Add secure identity mapping, token rotation and reuse detection, session revocation, rate limiting, configuration, API documentation, client integration, and tests without requiring production Firebase credentials."

## Clarifications

### Session 2026-11-05

- Q: How does Firebase relate to the OAuth and email-OTP mechanism described in the superseded guest OAuth feature and the current architecture document? → A: Firebase becomes the centralized identity provider for cloud Guest Mobile and Fleet Console sign-in. Direct Google/Apple OIDC and email-OTP sign-in described there are superseded for those clients where Firebase provides the equivalent capability. The backend remains the only issuer of application access and refresh tokens. The Hub keeps its own offline authentication mechanism.
- Q: Are unknown Firebase identities automatically provisioned? → A: A verified Firebase identity that is not linked to an internal account is provisioned as an active guest account according to the existing guest provisioning rule. Fleet Console administrator authority is never auto-provisioned; an administrator must already exist in the backend before a Firebase identity can receive Fleet Console access.
- Q: Can Firebase identities resolve to both guest and Fleet Console principals? → A: Yes, but only when backend records explicitly grant both. A verified Firebase identity always resolves to at most one internal account, and Fleet Console authority is a separate backend-owned grant with its own role, tenant, and ship scope.
- Q: Are Firebase custom claims authoritative for application authorization? → A: No. Firebase custom claims may be used only as non-authoritative context. The backend remains authoritative for account status, Guest Mobile access, Fleet Console access, roles, permissions, ship or fleet assignments, tenant boundaries, resource ownership, and suspension or revocation.
- Q: What happens when a different Firebase identity presents an email that already belongs to an existing account? → A: The accounts are not merged and the email is not used as the sole mapping key. The system resolves the identity only by its verified provider subject and requires an explicit, authenticated linking action to associate the identity with an existing account.
- Q: Does backend logout sign the user out of Firebase? → A: No. Backend logout revokes backend sessions only; Firebase sign-out remains under client control.
- Q: How should a Firebase identity that needs Fleet Console access map to internal principals? → A: Every Firebase identity maps to an internal account through `accounts` and `account_identities`. Fleet Console authority is a separate backend-owned administrator grant linked to that same account, and backend tokens use the internal account identifier as the subject. This preserves one SSO identity per person across Guest Mobile and Fleet Console while keeping authorization separate.
- Q: Should this feature expose authenticated endpoints for linking and unlinking Firebase identities? → A: Yes. The backend provides minimal authenticated Guest Mobile link and unlink endpoints. Linking requires a freshly verified Firebase ID Token for the identity being linked, rejects identities already linked elsewhere, is idempotent for the same account, and enforces last-credential protection on unlink. All link and unlink actions are audited.
- Q: Should client application changes be part of this feature, given that this repository contains only the backend? → A: No. This repository delivers the backend, database migration, OpenAPI contract, tests, and a documented client integration contract. Actual Guest Mobile and Fleet Console changes happen in their own repositories against that contract, covering Firebase sign-in, ID Token exchange, secure storage, backend access-token use, refresh, and logout.
- Q: What concrete rate-limit defaults should the specification require for the authentication endpoints? → A: Exchange: 10 requests per minute per IP address and 30 per minute per identity; refresh: 60 per minute per IP address and 30 per minute per session; link and unlink: 10 per minute per account; logout all sessions: 5 per minute per account. Exceeding a limit returns 429 with a stable `Retry-After` indication. Limits remain configurable but must not be relaxed below the documented defaults without review.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Sign In Through Firebase and Receive a Backend Session (Priority: P1)

A guest opens Guest Mobile, completes sign-in with Firebase, and the client exchanges the resulting Firebase ID Token for backend-owned tokens. The guest can then call protected backend APIs with the backend access token.

**Why this priority**: This is the entry point for every cloud-authenticated guest. Without a verified exchange there is no trustworthy backend session.

**Independent Test**: With a Firebase emulator or test double, sign in a test user, obtain a valid Firebase ID Token, exchange it at the backend, and verify that the response contains a backend access token, a backend refresh token, and an expiry. Repeat with the same Firebase identity and verify that the same internal account is returned. Register a new Firebase identity and verify that exactly one new guest account is created.

**Acceptance Scenarios**:

1. **Given** a user completes Firebase sign-in with a valid, unexpired ID Token, **When** the token is exchanged, **Then** the backend verifies the token and returns a backend access token, backend refresh token, access-token expiry, and session information.
2. **Given** a Firebase identity already linked to an internal account, **When** it is exchanged, **Then** the existing account is resolved and no duplicate account or identity is created.
3. **Given** a Firebase identity that is not linked to any internal account, **When** it is exchanged, **Then** exactly one active guest account and one Firebase identity link are created atomically.
4. **Given** a malformed, expired, not-yet-valid, wrong-audience, wrong-issuer, invalid-signature, or revoked Firebase ID Token, **When** it is exchanged, **Then** no backend tokens, account, or session are created.
5. **Given** a client has a backend access token, **When** it calls a protected API, **Then** the backend accepts the token as its own credential and does not treat the Firebase ID Token as an access credential.
6. **Given** a successful exchange, **When** the response and logs are inspected, **Then** the Firebase ID Token is not persisted as a long-term session credential and no secret or raw token is logged.

---

### User Story 2 - Maintain and Refresh a Backend Session Safely (Priority: P1)

An authenticated guest continues working across access-token expiry. The client refreshes transparently, receiving a new backend access token and a new refresh token. If an old refresh token is replayed, the affected session family is revoked.

**Why this priority**: Short-lived access tokens are only practical with reliable, secure refresh. Refresh-token reuse is a direct account-takeover risk.

**Independent Test**: Obtain a session, refresh it, and verify that a new access token and new refresh token are issued while the previous refresh token no longer works. Replay the previous refresh token and verify that every token in the family is rejected. Attempt two refreshes concurrently with the same token and verify that at most one succeeds without leaving an inconsistent state.

**Acceptance Scenarios**:

1. **Given** a valid unexpired refresh token, **When** it is presented, **Then** a new backend access token and a new backend refresh token are issued, and the presented token becomes unusable.
2. **Given** an already-rotated refresh token, **When** it is presented again, **Then** the entire token family and its session are revoked and no new session is issued.
3. **Given** two concurrent refresh requests using the same token, **When** both are processed, **Then** at most one succeeds and the family is never left in an ambiguous or partially rotated state.
4. **Given** an expired, revoked, malformed, or unknown refresh token, **When** it is presented, **Then** the request is rejected without revealing whether a session ever existed.
5. **Given** an account or administrator is suspended or deleted, **When** refresh is attempted, **Then** no new access token is issued.
6. **Given** a refresh response, **When** it is delivered, **Then** it is marked non-cacheable and contains no refresh token in a URL, log, or error message.

---

### User Story 3 - Fleet Console SSO With Backend-Controlled Authorization (Priority: P1)

A Fleet Console administrator signs in through Firebase and receives a backend session tied to the same internal account used by Guest Mobile. Access to Fleet Console APIs is granted only by a backend-owned administrator grant linked to that account, including its role, tenant, and ship assignments. A guest Firebase identity cannot obtain Fleet Console access merely by presenting a valid Firebase token or custom claim.

**Why this priority**: Centralized authentication must not collapse authorization boundaries. Fleet Console exposes fleet-wide and tenant-scoped operations, so backend authority is mandatory.

**Independent Test**: Create an active internal account with a Firebase identity link and an active administrator grant, exchange a valid token, and verify that Fleet Console APIs are authorized within the assigned tenant and ships but not outside them. Exchange a guest Firebase identity with no administrator grant and verify that the same Fleet Console APIs are denied. Add a Firebase custom claim such as an administrator role and verify that it does not grant access without a backend grant.

**Acceptance Scenarios**:

1. **Given** a verified Firebase identity linked to an internal account with an active administrator grant, **When** it is exchanged, **Then** a backend session is issued with the account's internal identifier and the administrator grant's backend-owned role, tenant, and ship scope.
2. **Given** a verified Firebase identity with no administrator grant, **When** it calls a Fleet Console API, **Then** the request is denied even if the Firebase token contains administrator-like custom claims.
3. **Given** a guest session, **When** it calls a Fleet Console API, **Then** the request is denied by backend policy.
4. **Given** an administrator whose linked internal account or administrator grant is suspended or deleted, **When** it attempts to exchange or refresh, **Then** the request is rejected.
5. **Given** a Fleet Console request, **When** the principal is resolved, **Then** tenant and ship boundaries are enforced from backend records on every request.
6. **Given** a verified Firebase identity authorized for both guest and Fleet Console use, **When** it uses either client, **Then** each client receives only the authority explicitly granted to it by backend records.

---

### User Story 4 - End Sessions and Revoke Access (Priority: P2)

An authenticated user signs out of the current device or, after a suspected compromise, signs out of all devices. Backend sessions are revoked without assuming that Firebase sign-out occurred.

**Why this priority**: Users and operators need a reliable way to terminate backend access. Revocation is also required when credentials or devices are lost.

**Independent Test**: Sign in on two sessions, log out one, and verify that only that session's refresh family is revoked. Sign out all sessions and verify that every refresh token for the internal account is rejected while other accounts remain unaffected. Repeat both operations and verify that they are idempotent.

**Acceptance Scenarios**:

1. **Given** a valid session, **When** the user logs out, **Then** the current refresh-token family and session are revoked and further refreshes fail.
2. **Given** the user logs out, **When** the operation completes, **Then** the client is not automatically signed out of Firebase and no Firebase credential is revoked by the backend.
3. **Given** multiple active sessions for one internal account, **When** the user signs out all sessions, **Then** every backend refresh-token family for that account is revoked and no other account is affected.
4. **Given** logout or logout-all is repeated, **When** it is processed, **Then** the result is idempotent and no previously revoked session becomes usable.
5. **Given** a session has already expired or been revoked, **When** logout is requested, **Then** the response does not reveal session existence or account details.

---

### User Story 5 - Link Existing Accounts and Handle Identity Collisions Safely (Priority: P2)

An existing guest signs in with a Firebase identity that is not yet linked to their account. The system does not merge accounts because of a matching email address; instead, an explicit authenticated linking action associates the verified identity with the intended account. Duplicate or conflicting identities are handled without corrupting data.

**Why this priority**: Email collisions and duplicate identities are common with social sign-in. Automatic merging can expose another person's data.

**Independent Test**: Create an account with an email address, then exchange a new Firebase identity asserting the same email but with a different subject. Verify that no merge occurs. Call the authenticated link endpoint for the intended account with a freshly verified Firebase ID Token and verify that the identity is attached and the account identifier is unchanged. Call the unlink endpoint and verify that the identity is removed only when another usable sign-in identity remains. Attempt to link the same identity to a second account and verify that the duplicate is rejected.

**Acceptance Scenarios**:

1. **Given** an existing account and a new Firebase identity with a matching email, **When** the identity is exchanged without an explicit link, **Then** no account is merged and no identity is attached solely by email.
2. **Given** an authenticated account owner completes an explicit linking action with a verified Firebase identity, **When** linking succeeds, **Then** the identity is associated with that account and the account identifier remains stable.
3. **Given** a Firebase identity already linked to another account, **When** linking to a different account is attempted, **Then** the operation is rejected and no existing mapping is changed.
4. **Given** concurrent exchanges for the same previously unknown Firebase identity, **When** they race, **Then** exactly one account and one identity link are created, and both requests resolve to the same account or one fails safely.
5. **Given** an account has only one usable sign-in identity, **When** unlinking that identity is attempted, **Then** the operation is rejected to avoid locking the account out.
6. **Given** an account is suspended or deleted, **When** an identity is linked or exchanged, **Then** the restriction is enforced and preserved.

---

### User Story 6 - Operate Authentication Securely and Predictably (Priority: P2)

An operator can deploy the service with environment-specific Firebase and signing configuration, observe authentication outcomes, rotate signing keys, and limit abuse without exposing credentials or personal data.

**Why this priority**: Firebase credentials, signing keys, replay attacks, and unobservable failures are production risks that must be handled before public exposure.

**Independent Test**: Start the service without required production Firebase or signing configuration and verify that it refuses to start. Retrieve the public signing key set and verify a token against it. Rotate the signing key and verify that new tokens use the new key while unexpired old tokens still verify during the overlap period. Trigger rate limits and verify a stable throttling response.

**Acceptance Scenarios**:

1. **Given** a production deployment missing a required Firebase project, credential, issuer, audience, or signing-key source, **When** the service starts, **Then** it fails fast with a clear configuration error.
2. **Given** a backend access token, **When** a relying party retrieves the public key set, **Then** the token can be verified using the matching key identifier without access to private key material.
3. **Given** a signing-key rotation, **When** new tokens are issued and old tokens are verified, **Then** new tokens use the active key and unexpired old tokens remain verifiable during a documented overlap period.
4. **Given** repeated invalid exchange or refresh attempts, **When** the configured threshold is exceeded, **Then** the caller receives a throttling response with a stable retry indication.
5. **Given** any authentication outcome, **When** logs and metrics are inspected, **Then** the outcome and failure category are observable without tokens, credential material, or unnecessary personal data.
6. **Given** a local development or test environment, **When** it uses the Firebase emulator or a test double, **Then** production credentials are neither required nor used.

---

### User Story 7 - Keep Hub Offline Authentication Independent (Priority: P3)

An onboard Hub continues to verify signed boarding passes, create local sessions, and enforce local authorization without Firebase or cloud connectivity.

**Why this priority**: The vessel must operate when the satellite link is unavailable. Firebase SSO must not create a hidden cloud dependency for the Hub.

**Independent Test**: Run the Hub offline authentication tests with Firebase and cloud services unavailable. Verify that boarding-pass verification, local session creation, and local authorization still succeed. Verify that no Firebase token or cloud refresh token is required by the Hub.

**Acceptance Scenarios**:

1. **Given** the Hub has no internet or Firebase connectivity, **When** a guest presents a valid signed boarding pass, **Then** the Hub verifies it and creates a local session.
2. **Given** the Hub is offline, **When** staff use an existing local session, **Then** local authorization still works.
3. **Given** cloud Firebase configuration changes, **When** the Hub operates, **Then** the Hub is not required to call Firebase or acquire a Firebase token.
4. **Given** cloud account data is synchronized to the Hub, **When** the payload is inspected, **Then** no Firebase ID Token, backend access token, or backend refresh token is included.

---

### Edge Cases

- The Firebase ID Token is malformed, uses the wrong algorithm, has an invalid signature, wrong issuer, wrong audience, expired `exp`, future `iat` or `nbf`, missing required claims, or an unknown key identifier.
- The Firebase user has been disabled or tokens have been revoked after issuance.
- A Firebase identity is unknown, already linked, linked to a suspended account, or concurrently presented by two first-time exchanges.
- The same Firebase `uid` later presents a different email, display name, or provider metadata.
- A new Firebase identity presents an email already used by another account.
- A user has identities from multiple Firebase providers (for example Google and Apple) under different Firebase `uid` values for the same person.
- The internal account is suspended, deleted, or later restored while sessions exist.
- An administrator is suspended or has ship assignments revoked while a session is active.
- A client attempts to use a Firebase ID Token as a backend access token, or a backend refresh token as an access token.
- Two refresh requests race with the same token; one rotates the family and the other triggers reuse handling.
- A refresh token is stolen and replayed after the legitimate client has rotated it.
- Logout is attempted with an unknown, expired, already-revoked, or malformed refresh token.
- Logout-all is attempted after some sessions have already expired or been revoked.
- The Firebase Admin SDK is unavailable, times out, or returns an unexpected error.
- The signing key source or KMS is unavailable or misconfigured.
- The signing key is rotated while unexpired access tokens remain in use.
- The system clock is skewed relative to Firebase or the backend token issuer.
- A production deployment is accidentally configured with a development Firebase project, emulator, or development signing key.
- A client cannot reach the refresh endpoint, receives a refresh failure, or restarts with securely stored tokens.
- The Hub is fully offline while cloud authentication is deployed.

## Requirements *(mandatory)*

### Firebase Authentication and Token Verification

- **FR-001**: Clients MUST authenticate users through Firebase Authentication and MUST send the resulting Firebase ID Token to the backend exchange endpoint to establish a backend session.
- **FR-002**: The backend MUST verify Firebase ID Tokens using the official Firebase Admin SDK. Manual Firebase JWT verification MUST NOT be implemented unless a documented technical requirement is reviewed and approved.
- **FR-003**: Firebase ID Token verification MUST validate the token signature, issuer, audience, expiration, issued-at time, not-before time, and all other claims required by the Firebase Admin SDK.
- **FR-004**: The backend MUST reject malformed, expired, not-yet-valid, wrong-project, wrong-audience, invalid-signature, and revoked Firebase ID Tokens.
- **FR-005**: The backend MUST check Firebase token revocation for exchange requests where the Firebase Admin SDK supports it. If revocation checking is unavailable, the backend MUST fail closed for the affected exchange rather than issue a session.
- **FR-006**: The verified Firebase `uid` MUST be the external identity subject. Client-supplied `uid`, email, provider, role, tenant, ship, or account identifiers MUST NOT be trusted as identity or authorization claims.
- **FR-007**: An unverified Firebase ID Token MUST NOT create a backend account, identity link, session, access token, or refresh token.
- **FR-008**: Firebase project configuration and service credentials MUST be supplied through environment-specific secure configuration and MUST NOT be committed to source control.
- **FR-009**: Production MUST use the production Firebase project and MUST fail fast when Firebase configuration is missing, invalid, or inconsistent with the configured issuer and audience.
- **FR-010**: Local development and automated tests MUST be able to use the Firebase emulator or a replaceable verifier double without production Firebase credentials. Emulator or test mode MUST be selectable only in an explicitly non-production configuration.
- **FR-011**: Firebase-asserted email MUST be used only as profile information or a linking hint when Firebase marks it verified. Email alone MUST NOT resolve or merge accounts.
- **FR-012**: Firebase custom claims MUST be treated as non-authoritative context. Authorization decisions MUST come from backend-owned account, role, tenant, ship, and ownership records.

### Identity Mapping and Account Lifecycle

- **FR-013**: Each verified Firebase identity MUST be represented by a provider identifier of `firebase` and the verified Firebase `uid` as the provider subject.
- **FR-014**: The combination of provider and provider subject MUST be unique and MUST map to at most one internal account.
- **FR-015**: A returning Firebase identity MUST resolve to its existing internal account without creating a duplicate account or identity.
- **FR-016**: A verified Firebase identity that is not linked to any internal account MUST be provisioned as exactly one active guest account according to the existing guest provisioning rule, atomically with its identity link.
- **FR-017**: Concurrent first-time exchanges for the same Firebase identity MUST create at most one account and one identity link.
- **FR-018**: Accounts MUST retain a stable internal identifier that does not change when identities are added, linked, or removed.
- **FR-019**: The system MUST NOT automatically merge accounts solely because their email addresses match. Account resolution MUST use the verified provider subject and explicit backend links.
- **FR-020**: When a new Firebase identity presents an email already associated with another account, the system MUST NOT attach it automatically and MUST provide a safe outcome that does not disclose whether the other account exists.
- **FR-021**: Linking a verified Firebase identity to an existing account MUST require an explicit authenticated action by a party authorized to act on that account.
- **FR-022**: Linking a Firebase identity that is already linked to another account MUST be rejected without changing the existing link.
- **FR-023**: Unlinking an identity MUST be rejected when it would leave the account with no usable sign-in method.
- **FR-024**: Identity linking, unlinking, provisioning, suspension, and deletion MUST be recorded in the audit log with the acting principal and affected account.
- **FR-025**: Accounts MUST preserve the existing lifecycle values `active`, `suspended`, and `deleted`. Authentication and refresh MUST reject any account that is not active.
- **FR-026**: The system MUST preserve existing account data, identities, relationships, and soft-delete semantics.
- **FR-027**: Every verified Firebase identity MUST map to an internal account through `accounts` and `account_identities`. Fleet Console authority MUST NOT be inferred from the Firebase identity or auto-provisioned. It MUST require a pre-existing active backend administrator grant linked to that internal account, with backend-owned role, tenant, and ship assignments.
- **FR-028**: The administrator grant MUST reference the same stable internal account identifier used by Guest Mobile, so one person can use both clients without a duplicate identity. The smallest necessary schema migration MUST be used to link administrator authorization records to internal accounts while preserving existing `admin_users` and `admin_ship_access` data.
- **FR-029**: The existing `accounts`, `account_identities`, and `refresh_tokens` records MUST be reused where they can represent the requirement. Any schema change MUST be delivered as a versioned migration and MUST update the schema baseline documentation.

### Backend Access Tokens

- **FR-030**: The backend MUST issue its own short-lived access token only after successful Firebase ID Token verification and internal account resolution.
- **FR-031**: Access tokens MUST have a configurable lifetime that MUST NOT exceed 15 minutes.
- **FR-032**: Access tokens MUST include the internal account identifier as the subject, a token issuer, an audience, an issued-at time, an expiration time, a unique token identifier, and a token type or use claim.
- **FR-033**: Access tokens MUST include only the minimum authorization context needed by backend policies. They MUST NOT include passwords, Firebase ID Tokens, provider tokens, refresh tokens, signing secrets, or unnecessary personal information.
- **FR-034**: Access tokens MUST be signed with an approved signing mechanism. The implementation MUST use an asymmetric algorithm with managed private-key storage and a public-key discovery mechanism such as JWKS unless the existing architecture explicitly defines another approved mechanism.
- **FR-035**: Production private signing keys MUST come from a secure managed source. A local development key MUST be allowed only in an explicitly non-production configuration and MUST NOT be used as a production fallback.
- **FR-036**: Access token validation MUST reject invalid signatures, disallowed algorithms, wrong issuers, wrong audiences, expired tokens, not-yet-valid tokens, future issued-at times, missing required claims, and wrong token types.
- **FR-037**: Access token validation MUST apply a documented configurable clock-skew allowance, defaulting to 60 seconds.
- **FR-038**: Public signing keys MUST be published with key identifiers and MUST NOT contain private key material.
- **FR-039**: Signing keys MUST be rotatable. New tokens MUST use the active key, and unexpired tokens signed by a retired key MUST remain verifiable during a documented overlap period.
- **FR-040**: A Firebase ID Token MUST NOT be accepted as a backend access token, and a refresh token MUST NOT be accepted as an access token.

### Backend Refresh Tokens and Sessions

- **FR-041**: Refresh tokens MUST be opaque, cryptographically secure, high-entropy values generated from a proven random source. They MUST NOT be derived from predictable account or device data.
- **FR-042**: Only a one-way hash of each refresh token MUST be stored. Raw refresh tokens MUST be returned only in the token response body.
- **FR-043**: Each refresh token MUST be associated with an internal account and a session or token family. Guest and Fleet Console sessions for the same account MUST remain independently revocable while sharing the same account subject.
- **FR-044**: Refresh tokens MUST have a configurable lifetime, defaulting to 30 days, and MUST be rejected after expiry.
- **FR-045**: Every successful refresh MUST rotate the refresh token. The presented token MUST become unusable, and the replacement MUST be linked to it within the same family.
- **FR-046**: Reuse of an already-rotated refresh token MUST revoke the entire token family and invalidate every active refresh token in that family.
- **FR-047**: Refresh rotation and reuse detection MUST be atomic and safe under concurrency. At most one concurrent refresh using the same token may succeed.
- **FR-048**: Refresh MUST be rejected for expired, revoked, malformed, unknown, or already-used tokens, and for principals that are not active. The response MUST NOT reveal which condition failed.
- **FR-049**: Suspended or deleted guest accounts and suspended or deleted Fleet Console administrators MUST be unable to refresh.
- **FR-050**: Logout MUST revoke the current refresh-token family and session and MUST be idempotent.
- **FR-051**: Logout all sessions MUST revoke every backend refresh-token family for the authenticated internal account, including Guest Mobile and Fleet Console sessions, and MUST be idempotent.
- **FR-052**: Logout and logout all sessions MUST NOT assume or perform Firebase sign-out.
- **FR-053**: Session and device metadata MUST be minimal and limited to what is operationally necessary. Refresh-token hashes, raw tokens, token families, and token identifiers MUST NOT be written to logs.
- **FR-054**: Refresh tokens MUST use the existing refresh-token storage model where possible. The token family MUST represent a session whose revocation state can be determined atomically.
- **FR-055**: Access to protected APIs MUST fail once a session is revoked and the access token expires, and refresh MUST fail immediately after revocation.

### Authentication API

- **FR-056**: The backend MUST provide a Firebase ID Token exchange operation at `POST /v1/auth/firebase/exchange`.
- **FR-057**: The exchange request MUST accept a Firebase ID Token and MUST verify it before issuing any backend token, creating any session, or granting any account access.
- **FR-058**: The exchange response MUST include a backend access token, a backend refresh token, the access-token expiration or equivalent expiry value, and relevant session information such as a session or family identifier. It MUST NOT include the Firebase ID Token, Firebase refresh token, or any provider secret.
- **FR-059**: The backend MUST provide a token refresh operation at `POST /v1/auth/refresh` that validates the refresh token, rotates it, and returns a new backend access token and refresh token.
- **FR-060**: The backend MUST provide a logout operation at `POST /v1/auth/logout` that revokes the current backend refresh token or session.
- **FR-061**: The backend MUST provide a logout-all operation at `POST /v1/auth/logout-all` that revokes all backend sessions for the authenticated internal account.
- **FR-062**: Authentication endpoints MUST follow the existing `/v1` and `/v1/auth` API conventions, the shared error envelope, documented status codes, and the project's OpenAPI contract.
- **FR-063**: Authentication responses MUST be marked non-cacheable and MUST NOT place tokens, Firebase ID Tokens, or session secrets in URLs.
- **FR-064**: Authentication errors MUST use safe, consistent messages. Unknown, expired, revoked, and malformed refresh tokens MUST NOT be distinguishable to the caller.
- **FR-065**: Exchange, refresh, and logout-all endpoints MUST use the existing authenticated request model where applicable and MUST NOT require Firebase tokens for subsequent protected API calls.
- **FR-066**: Authentication endpoints and their success and error responses MUST be documented in the project OpenAPI contract, including content types, authentication requirements, rate-limit responses, and non-cacheable responses.

### Identity Linking API

- **FR-095**: The backend MUST provide an authenticated link operation at `POST /v1/auth/identities/link` that accepts a freshly verified Firebase ID Token and associates that verified identity with the authenticated internal account.
- **FR-096**: The backend MUST provide an authenticated unlink operation at `DELETE /v1/auth/identities/{identityId}` that removes an identity only when the account retains at least one usable sign-in identity.
- **FR-097**: Link and unlink operations MUST reject identities already linked to another account, MUST be idempotent when the same identity is linked to the same account, and MUST NOT resolve accounts by email.
- **FR-098**: Link and unlink operations MUST validate the authenticated principal and the Firebase identity independently, MUST NOT trust client-supplied account identifiers, and MUST record the acting principal and affected account in the audit log.
- **FR-099**: Link and unlink responses MUST follow the existing error envelope, MUST be non-cacheable, and MUST NOT reveal whether another account exists.
- **FR-100**: Link and unlink operations MUST be documented in the OpenAPI contract and covered by authorization, duplicate-identity, last-credential, audit, and concurrency tests.

### Authorization and Roles

- **FR-067**: The backend MUST remain authoritative for internal account status, Guest Mobile access, Fleet Console access, roles, permissions, ship and fleet assignments, tenant boundaries, resource ownership, and access revocation.
- **FR-068**: Firebase custom claims MUST NOT be the sole source of application authorization.
- **FR-069**: Guest Mobile principals MUST NOT access Fleet Console APIs unless a backend policy explicitly authorizes the same principal for Fleet Console use.
- **FR-070**: Fleet Console principals MUST carry a backend-owned role, tenant scope, and ship scope, and every Fleet Console request MUST enforce those boundaries.
- **FR-071**: Backend authorization MUST NOT trust client-supplied account identifiers, roles, tenants, ship identifiers, or authorization claims.
- **FR-072**: Suspension, deletion, role removal, and ship-assignment revocation MUST take effect no later than the next refresh or access-token expiry consistent with the short access-token lifetime.
- **FR-073**: Security-relevant authentication, identity, session, suspension, and revocation events MUST be fully recorded in the audit log.

### Hub Isolation

- **FR-074**: Hub authentication MUST remain separate from Firebase SSO and the backend session model.
- **FR-075**: The Hub MUST continue to support offline boarding-pass verification, local session creation, local authorization, and operation without a live Firebase or cloud connection.
- **FR-076**: The Hub MUST NOT be required to call Firebase or obtain a Firebase ID Token to authenticate guests or staff.
- **FR-077**: Firebase ID Tokens, backend access tokens, and backend refresh tokens MUST NOT be synchronized to the Hub.
- **FR-078**: Changes to cloud authentication MUST NOT break or weaken existing Hub offline authentication behavior.

### Security, Configuration, and Operations

- **FR-079**: Firebase credentials, signing keys, and related secrets MUST be supplied through secure configuration and MUST NOT be committed to source control or written to logs.
- **FR-080**: Production startup MUST fail fast when required Firebase configuration, signing-key configuration, issuer, audience, or rate-limit configuration is missing or invalid.
- **FR-081**: Authentication, identity-linking, and refresh endpoints MUST be rate-limited with the documented defaults: exchange at 10 requests per minute per IP address and 30 per minute per identity; refresh at 60 per minute per IP address and 30 per minute per session; link and unlink at 10 per minute per account; and logout all sessions at 5 per minute per account. Exceeding a limit MUST return 429 with a stable `Retry-After` indication. Limits MUST apply across service instances where the deployment supports multiple instances.
- **FR-082**: Rate-limit or shared-store unavailability MUST fail safely and MUST NOT silently disable protection.
- **FR-083**: Authentication logs and metrics MUST record outcomes and failure categories without exposing Firebase ID Tokens, backend tokens, refresh-token hashes, authorization codes, credential material, or unnecessary personal data.
- **FR-084**: All authentication transport MUST use TLS in deployed environments.
- **FR-085**: Calls to Firebase, key services, rate-limit stores, and the database MUST have bounded timeouts and MUST NOT hang indefinitely.
- **FR-086**: Error responses MUST NOT reveal implementation details, token contents, account existence, or whether a Firebase user exists.
- **FR-087**: Authentication configuration, required environment variables, and safe local-development setup MUST be documented.

### Testing, Documentation, and Compatibility

- **FR-088**: Unit tests MUST cover Firebase token verification outcomes, claim validation, identity resolution, access-token validation, refresh rotation, refresh reuse detection, configuration fail-fast behavior, and authorization-claim handling.
- **FR-089**: Integration tests MUST cover account provisioning, returning identity resolution, identity link and unlink, duplicate identity handling, email collisions, concurrent exchanges, concurrent refreshes, token-family revocation, logout, logout all sessions, suspension, and Guest Mobile versus Fleet Console authorization boundaries against a real PostgreSQL database.
- **FR-090**: Contract tests MUST verify that the documented authentication operations are mounted and that success and error responses match the OpenAPI contract.
- **FR-091**: Tests MUST use Firebase emulator tooling, mocks, fake clocks, and test doubles. They MUST NOT require production Firebase credentials or live Google, Apple, KMS, or email credentials.
- **FR-092**: Regression tests MUST prove that Hub offline authentication remains functional when Firebase and cloud services are unavailable.
- **FR-093**: Existing tests, linting, build verification, and schema-parity checks MUST continue to pass.
- **FR-094**: The project architecture documentation, schema baseline, OpenAPI contract, and environment-variable documentation MUST be updated to reflect Firebase SSO, identity mapping, token behavior, and the Hub boundary.
- **FR-101**: The feature MUST provide a documented client integration contract covering Firebase sign-in, Firebase ID Token exchange only when establishing or re-establishing a backend session, secure storage of backend tokens, backend access-token usage, refresh on expiry, refresh-failure handling, logout, and the rule that Firebase ID Tokens are not persisted as long-term backend session credentials.
- **FR-102**: Guest Mobile and Fleet Console client source changes are outside this repository's deliverables. The contract MUST be sufficient for those clients to integrate without relying on undocumented backend behavior.

### Key Entities

- **Firebase Identity**: A verified external identity identified by Firebase `uid`, optionally carrying verified email and display metadata. It is the only trusted external identity key.
- **Internal Account**: A stable guest identity with lifecycle status, locale, optional profile data, and relationships to bookings and other business records.
- **Administrator Authorization Record**: A backend-owned Fleet Console grant with role, tenant, ship scope, status, and lifecycle. It references the same stable internal account as Guest Mobile, is separate from Firebase claims, and is explicit rather than inferred.
- **Account Identity Link**: A unique association between an internal account or administrator and one Firebase identity, identified by provider plus provider subject.
- **Backend Session**: A login session represented by a refresh-token family, with expiry, revocation, and minimal device metadata.
- **Refresh Token**: A hashed, opaque, single-use credential linked to an account and session, with issued, expiry, used, revoked, and replacement state.
- **Access Token**: A short-lived backend-signed credential carrying the internal subject, issuer, audience, timestamps, unique identifier, and minimum authorization context.
- **Signing Key**: A managed asymmetric key with an identifier, active or retired state, and a public verification representation.
- **Authentication Audit Event**: An append-only record of security-relevant identity, session, authorization, suspension, and revocation actions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A first-time guest can complete Firebase sign-in and receive a usable backend session without manual account setup.
- **SC-002**: 100% of invalid, expired, not-yet-valid, wrong-project, wrong-audience, invalid-signature, and revoked Firebase ID Tokens are rejected in automated tests, with zero backend sessions created.
- **SC-003**: Returning users always resolve to the same internal account for the same Firebase identity, with zero duplicate accounts in repeated sign-in tests.
- **SC-004**: A previously unknown Firebase identity creates exactly one guest account and one identity link, including under concurrent first-time exchanges.
- **SC-005**: Accounts with matching emails are never merged automatically, and automated tests confirm zero automatic merges.
- **SC-006**: 100% of successful refreshes rotate the refresh token, and every previous refresh token becomes unusable.
- **SC-007**: 100% of replayed refresh tokens revoke their token family, and every token in that family is rejected afterward.
- **SC-008**: Concurrent refresh requests using the same token result in at most one success and never leave a partially rotated or ambiguous session state.
- **SC-009**: Logout revokes the current session, and logout-all revokes every session for the account, with no effect on other accounts.
- **SC-010**: 100% of automated authorization tests prevent Guest Mobile principals from accessing Fleet Console APIs without an explicit backend grant.
- **SC-011**: Access tokens never exceed 15 minutes of validity, and expired or mis-signed tokens are rejected.
- **SC-012**: Suspended or deleted accounts and administrators cannot exchange or refresh, and access ends no later than access-token expiry.
- **SC-013**: No raw refresh token, Firebase ID Token, signing secret, or credential material is stored in the database or emitted in logs during automated tests.
- **SC-014**: Authentication, identity-linking, and refresh rate limits use the documented default thresholds, produce a 429 response with `Retry-After` when exceeded, and remain effective across instances in the tested deployment configuration.
- **SC-015**: Hub offline authentication tests pass with Firebase and cloud services unavailable, with no Firebase dependency introduced.
- **SC-016**: Missing or invalid production Firebase or signing configuration prevents startup rather than silently enabling an insecure fallback.
- **SC-017**: Signing-key rotation publishes the new public key before use, and unexpired tokens from the previous key remain verifiable during the documented overlap period.
- **SC-018**: All existing and new build, test, lint, and schema-parity checks pass.
- **SC-019**: The OpenAPI contract documents every authentication success and error response, and contract tests confirm the documented operations are mounted.
- **SC-020**: Identity linking and unlinking work through the authenticated API, reject duplicate mappings and last-credential removal, and produce audit records for every successful and rejected action.
- **SC-021**: The client integration contract documents every step from Firebase sign-in through logout, and review confirms that no undocumented backend behavior is required for Guest Mobile or Fleet Console integration.

## Assumptions

- The existing `accounts`, `account_identities`, and `refresh_tokens` tables remain the baseline. The current `account_identities` provider check does not include `firebase`, so the smallest necessary migration will extend provider support and preserve existing rows. Any additional session or administrator-link fields will also be delivered as a versioned migration.
- The superseded guest OAuth feature established guest account, identity, access-token, refresh-token, and configuration rules. This feature reuses those rules and replaces the direct guest OAuth sign-in mechanism with Firebase for Guest Mobile and Fleet Console where Firebase provides the same capability.
- Every cloud identity uses one internal account. The existing `admin_users` and `admin_ship_access` records remain the authoritative Fleet Console authorization model, but they will be linked to that internal account by a minimal migration. A verified Firebase identity receives Fleet Console access only through an explicit backend administrator grant linked to its account; no email-only or automatic administrator provisioning occurs. Backend tokens use the internal account identifier as the subject, and `refresh_tokens.subject_type = 'account'` is used for cloud sessions.
- The backend is implemented as a Go modular monolith using the existing Chi, pgx, sqlc, goose, PostgreSQL, and OpenAPI conventions. The Firebase Admin SDK and an approved JWT/JWKS library will be used rather than hand-rolled cryptography.
- Firebase handles credential collection, provider sign-in, password recovery, email verification, multi-factor authentication where configured, and Firebase account lifecycle. The backend does not store Firebase passwords or provider credentials.
- Firebase project identifiers, service-account credentials, signing keys, and issuer/audience values are supplied by the deployment environment. This feature does not create or manage external Firebase projects.
- The documented defaults are 15-minute access tokens, 30-day refresh tokens, and a 60-second clock-skew allowance. The documented rate-limit defaults are exchange 10 per minute per IP and 30 per minute per identity, refresh 60 per minute per IP and 30 per minute per session, link and unlink 10 per minute per account, and logout-all 5 per minute per account. These values may be configured, but rate limits must not be relaxed below the documented defaults without review.
- A Firebase identity that is unknown to the backend is provisioned as a guest account. Fleet Console access is never provisioned automatically and requires an existing administrator grant linked to that account.
- Guest Mobile and Fleet Console are separate clients with separate backend authorization policies even when they share one Firebase identity. A shared Firebase identity does not by itself grant access to either client. Sessions remain independently revocable per device and client even though they share the same account subject.
- The Hub continues to use its own offline boarding-pass, local-session, and local-authorization mechanisms. Firebase and backend session tokens are cloud-only and are not synchronized to the Hub.
- Firebase custom claims are not a substitute for backend authorization. The backend remains the source of truth for roles, permissions, tenant scope, ship assignments, and resource ownership.
- Refresh tokens are single-use and clients serialize refresh attempts. Concurrent reuse is treated as a security event that revokes the token family.
- Identity linking and unlinking are in scope for this feature through minimal authenticated Guest Mobile endpoints. Administrator-only identity administration and bulk account migration are out of scope.
- This repository contains the backend only; Guest Mobile and Fleet Console source code is not present. This feature delivers the backend, migration, OpenAPI contract, tests, and client integration contract. Client application changes are implemented in their respective repositories.
- Existing architecture language that describes direct guest OAuth, or automatic account linking based on matching email addresses, conflicts with this feature and will be updated as part of the implementation documentation.
