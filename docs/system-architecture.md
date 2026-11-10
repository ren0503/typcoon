# System Architecture Document: Yacht Management Application

Version 0.1 (Draft) · Updated: 31 October 2026 · Status: Proposed, pending resolution of the items in Section 13

## 1. Overview

### 1.1 Objectives

The system supports yacht operations: managing vessels, voyages, staff (security, cleaning, and operators), passengers, and onboard business partners; automatically scheduling shifts using AI; and sending notifications to staff and guests. The system must continue operating when the vessel loses its satellite connection.

### 1.2 Design Principles

- **Offline-first:** All onboard business operations must work without internet access.
- The cloud is the source of truth for master data. The onboard hub is the source of truth for operational data while the vessel is underway.
- Only the hub communicates with the cloud. The hub initiates outbound connections; the cloud never connects inbound to the vessel.
- AI makes recommendations, the deterministic rule engine performs validation, and authorized personnel may override decisions. An AI kill switch must always be available.
- Each data item has only one write owner at any given time to prevent conflicts.
- Keep it simple initially: one hub per vessel, a modular-monolith backend, and EC2 + Auto Scaling instead of Kubernetes.

### 1.3 Out of Scope for This Version

Payments and folios (room accounts), accounting, third-party PMS integrations, and ticket sales to individual walk-in guests. These items are listed in Section 13.

## 2. System Components

| Component | Technology | Responsibility | Offline |
| --- | --- | --- | --- |
| Cloud backend (API + worker) | Go on EC2 Auto Scaling | Manage vessels, voyages, staff, guests, and bookings; publish data to hubs; receive operational data from hubs | Not applicable |
| Cloud database | RDS PostgreSQL | Source of truth for master data, bookings, and `change_log` | Not applicable |
| Fleet console | Web (React) | Manage the fleet, partners, core staff records, rule configuration, AI parameters, and reports | Not required |
| Ship hub | Go + SQLite on Nvidia Jetson (Ubuntu) | Store operational data, perform automatic scheduling, serve staff iPads and guest mobile apps, and synchronize with the cloud | Operates independently |
| Staff iPad app | iPad, local SQLite | Shift schedules, guest counts by deck, shift-change requests, emergency mode, and manual assignments | Yes |
| Guest mobile app | iOS/Android | Bookings, room information, notifications, and onboard services | Yes (cached data) |

## 3. Cloud Architecture (AWS)

### 3.1 Services Used

| Service | Responsibility |
| --- | --- |
| CloudFront + WAF + ALB | HTTPS entry point, attack protection, and mTLS for hubs |
| EC2 + Auto Scaling Group (API) | Stateless API servers behind the ALB |
| EC2 + Auto Scaling Group (worker) | Generate snapshots/deltas, run background jobs and notifications, and receive data from hubs |
| RDS PostgreSQL (Multi-AZ) | Source of truth, with RDS Proxy to prevent connection exhaustion during scaling |
| ElastiCache (Redis/Valkey) | Caching, rate limiting, distributed locks, and pub/sub change notifications |
| SQS + SNS | Internal cloud events, fan-out, job queues, and DLQs |
| EventBridge Scheduler | Schedule voyage-related jobs |
| S3 | Signed snapshot packages, images, contracts, logs, and post-voyage surveys |
| KMS, Private CA, Secrets Manager | Signing keys, device certificates, and secrets |
| CloudWatch | Logs, metrics, and alarms |

### 3.2 Backend

A single Go codebase structured as a modular monolith with twelve modules. Modules communicate with one another only through the public interfaces declared in their `module.go`; no module imports another module's repository or generated query layer directly. The same source code is deployed as two groups: API (handles requests) and workers (process SQS queues). Suggested libraries: chi or gin, pgx + sqlc, goose or golang-migrate, slog, and oapi-codegen.

| Module | Owned tables / responsibility | HTTP surface | Audit emission |
| --- | --- | --- | --- |
| `tenant` | `tenants`; root tenant lifecycle | `/api/tenants` | No (deferred) |
| `ship` | `ships`, `decks`, `areas`, `rooms`; ship master data and capacity | `/api/ships`, `/api/decks`, `/api/areas`, `/api/rooms` | No (deferred) |
| `voyage` | `ports`, `voyages`, `port_calls`; itinerary and voyage windows | `/api/ports`, `/api/voyages`, `/api/port-calls` | No (deferred) |
| `staff` | `staff`, `staff_leaves`, `staff_credentials`, `shift_templates`, `coverage_requirements`, `labor_rules`, `autonomy_policies` | `/api/staff`, `/api/shift-templates`, `/api/coverage-requirements`, `/api/labor-rules`, `/api/autonomy-policies` | Yes (**create/update/soft-delete**, `staff.*`) |
| `partner` | `partners`, `partner_contracts`, `partner_staff`, `partner_services`, `partner_slots` | `/api/partners`, `/api/partner-contracts`, `/api/partner-staff`, `/api/partner-services`, `/api/partner-slots` | No (deferred) |
| `booking` | `bookings`, `service_bookings`; reservation and idempotent booking writes | `/api/bookings`, `/api/service-bookings` | Yes (**create/update/soft-delete**, `booking.*`, `service_booking.*`) |
| `guest` | `guests`, `boarding_passes`; opaque encrypted documents | `/api/bookings/{id}/guests`, `/api/guests`, `/api/boarding-passes` | Yes (**create/update/soft-delete**, `guest.*`) |
| `auth` | `admin_users`, `admin_ship_access`, `accounts`, `account_identities`, `refresh_tokens`; Firebase SSO, backend token issuance, principal resolution, and token hashes | `/api/auth/*` | Yes (**session/identity events**, `auth.*`) |
| `sync` | `hub_devices`, `ship_devices`, `revocations`, `snapshots`, `job_runs`, `ops_events`; read-only `ship_seq`/`change_log` | none (internal ops only) | No (internal ops metadata) |
| `audit` | `audit_log`; shared append-only appender used inside caller transactions | none | Owns the append path |
| `ai` | `ai_params`; versioned metadata only, no signing workflow | none in this phase | No (deferred) |
| `feedback` | `feedback`, `feedback_insights`; submit/read only | none in this phase | No (deferred) |

Ownership boundaries: the table owner is the only module allowed to write its rows. Cross-module reads go through interface methods (for example `ship.Service.TenantID`, `voyage.Service.Lookup`, `booking.Service.BookingExists`) and are re-checked for tenant/ship scope. Audit emission covers `staff`, `booking`, `guest`, and `auth` in this baseline; `audit.Appender.AppendTx` must be called inside the same transaction as the business write so a failed audit append rolls the write back. The `sync`, `ai`, and `feedback` modules are internal in this phase and register no HTTP routes; `auth` exposes public and protected `/api/auth/*` routes. `change_log` and `ship_seq` remain trigger-owned and are never written by application code.

### 3.3 Backend Responsibilities

- Manage vessels, voyages, and staff.
- Manage guest accounts (OAuth) and bookings.
- Publish data to hubs (snapshots and deltas).
- Schedule jobs to keep hub data up to date.
- Receive operational data from hubs (tasks, actual shifts, check-ins, incidents, and AI audit records).
- Analyze post-voyage feedback and publish new AI parameters (later phase).

### 3.4 EC2 + Auto Scaling Operations

| Area | Rule |
| --- | --- |
| State | APIs must not keep state on individual machines. Sessions are stored in JWTs or Redis; files are stored in S3 |
| Distribution | Spread across multiple AZs; ALB health check at `/readyz` |
| Deployment | AMIs or containers on EC2, with rolling updates (instance refresh or CodeDeploy). Database migrations are a separate step before deployment and must not run at instance startup |
| Scaling | Scale APIs based on request volume or CPU. Scale workers based on SQS queue depth |
| Instance shutdown | Use lifecycle hooks to let workers finish messages currently being processed; configure a sufficiently long visibility timeout |
| Administration | Use Systems Manager Session Manager instead of SSH; store secrets in Secrets Manager |
| Infrastructure | Terraform or CDK |

### 3.5 Cloud Database

Main table groups: `ships`, `decks`, `rooms`, `voyages`, `port_calls`, `staff`, `partners`, `partner_slots`, `bookings`, `guests`, `shift_templates` and shift plans, `accounts`, `refresh_tokens`, `hub_devices`, `change_log`, `ship_seq`, `job_runs`, and `audit_log`.

Migration `00002_business_schema.sql` installs the 41-table business baseline (including the `capture_change`/`touch_row` triggers and exclusion constraints) as a faithful copy of `docs/schema.sql`; `db/schema.sql` is kept in sync for sqlc generation.

Conventions: UUID v7 primary keys and common columns `id`, `ship_id`, `created_at`, `updated_at`, `version`, and `deleted_at`. Prevent overlapping slots using PostgreSQL exclusion constraints. Sensitive data (passports) is encrypted at the field level using KMS. Redis locks are only used to reduce load; the final decision is always enforced by database constraints.

## 4. Edge Architecture (Jetson Hub)

### 4.1 Hardware

- Nvidia Jetson Orin series (the exact model will be selected after real-world benchmarking), NVMe storage instead of an SD card, and a battery-backed RTC.
- Connect via Ethernet to the core switch. A UPS supplies power to the hub, switch, and access points.
- A second hub running in cold-standby or warm-standby mode is recommended.
- Full-disk encryption and secure boot enabled.

### 4.2 Software

Ubuntu, Go, and SQLite (modernc driver, WAL mode, single writer). Processes are separated and resource-limited using systemd and cgroups so that AI overload does not slow down the API:

| Process | Responsibility |
| --- | --- |
| `hub-api` | Two API listeners: one for staff and one for guests, each on a separate VLAN |
| `sync-agent` | Pull snapshots/deltas from the cloud and push the outbox to the cloud |
| `planner` | Solver (OR-Tools CP-SAT) and deterministic rule engine for shift scheduling |
| `llm-runtime` | Small locally running LLM, used only to convert free-form requests into structured data |
| `notify` | Deliver notifications over direct LAN connections; use push notifications when internet is available |

### 4.3 Hub Responsibilities

- Receive passenger and staff information from the cloud.
- Automatically schedule shifts and send notifications to staff through their iPads.
- Reassign staff when changes occur (shift swaps or emergencies).
- Serve staff iPads and guest mobile apps over the local network.
- Record operational events and push them to the cloud.

### 4.4 Degraded Operating Modes

| Failure | Behavior |
| --- | --- |
| LLM crashes or becomes too slow | Accept requests through structured forms in the iPad app |
| Solver times out | Use a simple heuristic to always produce a feasible option |
| Satellite connection is lost | Hub continues operating normally; outbox accumulates locally |
| Primary hub fails | Fail over to the standby hub using a virtual IP; data is replicated through Litestream |
| Software update fails | Use A/B updates and automatically roll back to the previous version if health checks fail |
| AI produces suspicious results | Captain activates the AI kill switch and switches to manual assignment |

## 5. Client Applications

### 5.1 Staff iPad App

- Features: shift schedules, tasks, guest counts by deck, shift-change requests, issue reporting, and notifications.
- Supervisor role: approval queue, emergency assistance mode, manual assignment, enable/disable automation, and visibility into AI decision rationale.
- Offline-first: local SQLite and an operation queue with idempotency keys. Actions requiring an immediate decision (approval or emergency actions) require a connection to the hub and must not be silently queued.
- Display connection status and the time when data was last updated. Operational time comes from the hub; the iPad's clock must not be trusted.
- Deploy through MDM, using Single App Mode, with the internal CA certificate and Local Network permission provisioned in advance.

### 5.2 Guest Mobile App

- Before boarding: register an account and book a room and vessel through the cloud.
- Onboard: receive room information and notifications, and manage services through the hub on the local network.
- When internet access is available, the app may call the cloud according to the rules in Section 6.3.
- Use a signed boarding pass (QR code) to enter a local session on the hub.

### 5.3 Fleet Console (Web)

Approximately eight pages: vessel list, vessel details (decks and rooms), port library, users and roles, partners and contracts, core staff records, rule and AI parameter configuration, and reports and audit logs.

### 5.4 Hub Diagnostics Page

A minimal page accessible through the management LAN for technicians, showing sync status, storage capacity, version, and logs. It contains no business functionality.

## 6. Data Ownership

### 6.1 By Data Type

| Type | Examples | Authority | Sync Direction |
| --- | --- | --- | --- |
| Master data | Vessels, decks, rooms, partners, staff profiles, guest manifest | Cloud | Cloud → hub |
| Operational data | Actual shift plans, tasks, shift changes, check-ins, incidents, AI audits | Hub while the vessel is underway | Hub → cloud |
| AI parameters | Weights, autonomy thresholds, forecasting models | Cloud | Cloud → hub, versioned and signed |
| Accounts and payments | Guest profiles, payment methods | Cloud | Hub keeps a read-only copy |

### 6.2 Booking Ownership by Phase

| Type | Before manifest finalization | After finalization |
| --- | --- | --- |
| Rooms | Cloud | Hub (room changes onboard only), then synchronized back to the cloud |
| Services and partner slots | Cloud (for advance booking) | Hub |

Prevent duplicate bookings using exclusion constraints or unique constraints in the database. Every write request must include an `Idempotency-Key`. During payment, reservations are held temporarily (10–15 minutes).

### 6.3 Two Connection Paths for the Guest Mobile App

| Scenario | Read | Write |
| --- | --- | --- |
| Not yet onboard, internet available | Cloud | Cloud |
| Onboard, hub reachable | Hub | Hub |
| Onboard, hub unreachable, internet available | Cloud (may be stale) | Accounts and payments only. Service requests are queued in the cloud for hub processing; the app displays “awaiting confirmation” |
| No hub and no internet | Local cache | Queue in the app and retry when connectivity returns |

The cloud returns an `ownership` status (`cloud` or `ship`) for each voyage so the app knows where writes should be directed.

## 7. Cloud ↔ Hub Synchronization

### 7.1 Underlying Approach

Synchronization is based on a change log. The hub actively pulls and pushes data over HTTPS with mTLS. Database-level replication is not used because the two sides use PostgreSQL and SQLite, and the satellite link is frequently interrupted.

| Channel | Direction | Purpose |
| --- | --- | --- |
| Snapshot | Cloud → hub | Initial setup for a voyage, a new hub, or recovery. Compressed file (zstd) on S3, downloaded through a presigned URL with resumable downloads |
| Delta | Cloud → hub | Changes to master data, manifest, and AI parameters. API returns changes after a cursor |
| Outbox | Hub → cloud | Operational data, sent in batches with idempotency keys |
| Change notification | Cloud → hub | Only indicates that a new package is available; carries no data |

### 7.2 Downstream Synchronization (Cloud to Hub)

- The cloud writes to `change_log` in the same transaction as the data change. Each vessel has a `ship_seq` counter row updated in that same transaction. The row lock is held until commit, so `seq` order matches commit order and the hub cannot miss records.
- The hub calls `GET /api/sync/changes?after=<last_seq>`. Deletions use versioned tombstones.
- The hub writes each batch of changes and updates `last_seq` in the same SQLite transaction. Reapplying an old batch is harmless because of `version`.
- Each batch includes `schema_version`. A hub that has not been upgraded rejects the batch and reports the issue.
- A hub receives data only for its own vessel and the current voyage.
- Change notification: a long-poll endpoint, `GET /api/sync/wait?after=<seq>`, keeps the connection open for up to 30 seconds. API instances notify one another through ElastiCache pub/sub. If this channel fails, the hub polls every 60 seconds. Redis is only an optimization; RDS remains the source of truth.
- The hub does not subscribe to SNS/SQS and does not hold AWS credentials.

### 7.3 Upstream Synchronization (Hub to Cloud)

- Every operational change is written to the `outbox` table in the same transaction as the data change, with a local sequence number and `hub_id`.
- The hub sends small compressed batches. Each batch contains `(hub_id, from_seq, to_seq)` and `epoch`. The cloud deduplicates using `(hub_id, seq)` and returns an `ack`. This provides effectively-once processing.
- Two priority queues: incidents, muster events, and check-ins are sent first; logs and large reports are sent later.
- The hub's `epoch` is incremented when the hardware is replaced. The cloud rejects data from older epochs.

### 7.4 Conflicts and Ordering

Ownership by table means conflicts should be rare. The cloud wins for master data; the hub wins for operational data. Event ordering is based on sequence numbers assigned by the hub, not timestamps alone, because clocks may drift. The hub needs a battery-backed RTC and time synchronization from the vessel's GPS or NTP.

### 7.5 Bandwidth and Reliability

Use persistent HTTP/2 connections, zstd compression, size-limited batches, and retries with backoff and jitter. The hub reports `sync_lag` so the Fleet Console can raise alerts. CRDT, AppSync, Kafka, and MQTT are not used as the primary data transport.

## 8. Voyage Lifecycle and Scheduler

### 8.1 Three Synchronization Phases

| Phase | Timing | Contents |
| --- | --- | --- |
| Preparation | Approximately 24–48 hours before departure | Initial snapshot: vessel, staff, partners, expected manifest, and AI parameters. The hub applies it and runs the initial shift scheduling |
| Boarding opens | When embarkation begins | Latest manifest delta for check-in |
| Finalization | 1 hour to 30 minutes before departure | Finalize the manifest and activate the official plan |

During the voyage, changes are delivered through small deltas. After the voyage, the hub pushes any remaining data, collects surveys, and deletes guest data from the hub.

### 8.2 Scheduler Mechanism

- Each voyage has fixed, named schedules (`voyage-{id}-prepare`, `-open-boarding`, `-finalize`, `-cutoff-check`) in EventBridge Scheduler. When `departure_at` changes, the backend updates them using an upsert by name.
- Messages carry only `voyage_id`, job type, and `schedule_version`. When a worker runs, it reads the actual time from the database and skips the job if the version is outdated.
- Prevent duplicate execution using a unique constraint on `job_runs(voyage_id, job_type, schedule_version)` and an advisory lock per voyage.
- Failed jobs are retried by SQS. Jobs exceeding the retry threshold go to a DLQ and trigger an alert.
- A monitoring job runs every five minutes. If a hub's synchronization lag exceeds the threshold, it alerts the Captain and Fleet Console.
- If synchronization fails during the finalization window: raise an alert, retry, and ultimately import a signed package through USB. The hub can operate with stale data but must clearly display “manifest not finalized”.
- Store times in UTC together with the port time zone. Multi-port itineraries require manifest updates for each leg.

## 9. AI-Based Shift Scheduling and Autonomy Levels

### 9.1 Three AI Tools

| Tool | Purpose |
| --- | --- |
| Solver (OR-Tools CP-SAT) | Generate optimized shift schedules and reassignments |
| Small forecasting model or statistical rules | Predict staffing demand based on guest counts, check-out dates, and peak hours |
| Local LLM | Convert free-form requests into structured data; it does not write to the database directly |

The LLM and solver only make recommendations. A deterministic rule engine (not AI) validates hard constraints: minimum rest periods, total weekly hours, required certifications, and minimum security staffing levels. Maritime rest-hour regulations depend on international rules and the vessel's flag state; a legal expert must confirm the specific limits.

### 9.2 Initial Shift Scheduling

After the snapshot is applied, the hub automatically generates a shift schedule based on staff data, passenger data, and records from previous voyages. For a vessel with no history, cloud-provided shift templates are used as the starting point. The result is a versioned plan with a score and rationale. Supervisors can review and modify it.

### 9.3 Shift Changes

Workflow: iPad submits a request → LLM converts it into an intent (if it is free-form text) → rule engine validates it → solver finds a replacement arrangement affecting the fewest people → the change is executed or sent to a supervisor depending on the autonomy level → notifications are sent and an audit record is written.

Limits: cap the number of people affected per request (for example, three), and prioritize minimal changes and fairness. If no valid solution exists, clearly explain what is missing and where, then escalate to a supervisor.

### 9.4 Supervisor Emergency Mode

The supervisor selects an area (for example, Deck 3), a role (cleaning), a time, and a minimum staffing level, then asks AI to reassign staff. Within seconds, the system returns a change table showing who moves from where to where, which areas become understaffed, and travel time. The supervisor can accept the proposal, edit individual changes, or assign staff manually.

Manual assignments are still checked by the rule engine: hard violations are blocked; soft violations generate warnings and may be overridden with a reason. Only the Captain may override rest-period rules, and such overrides are logged. All changes pass through a planner queue with a single worker. Plans are versioned; if a conflict occurs, the plan is recalculated rather than blindly overwritten.

### 9.5 Autonomy Levels

| Level | Example | Handling |
| --- | --- | --- |
| Fully automatic | Assign rooms for cleaning, shift reminders, routine notifications | Execute automatically and log |
| Conditional automation | Shift swap between two employees with the same job title and no rule violations | Automatically approve after passing the rule engine; notify the supervisor |
| Requires approval | Change a security position, make up hours, reduce staffing below the safe minimum | Supervisor confirms on the iPad |
| Never automated | Emergency notifications, muster, security incidents | Initiated by a person; system provides support |

These levels are configured in the cloud, adjustable for each yacht operator, and published to the hub with safe default values.

### 9.6 Post-Voyage Feedback Learning Loop

1. Collect text surveys from guests and staff, including while offline.
2. Anonymize the data and apply GDPR retention policies.
3. A cloud LLM analyzes the feedback into JSON conforming to a fixed schema (topic, area, service, severity, sentiment). Results that do not conform to the schema are rejected.
4. Aggregate by vessel and vessel class, normalized by guest count and voyage type.
5. Convert findings into controlled parameter changes; an analyst reviews significant changes.
6. Backtest against historical voyages.
7. Publish a versioned and signed parameter release. The hub runs it in shadow mode before switching over, with a rollback path always available.

Actual operational data (room-cleaning times and the rate at which supervisors modify schedules) is more reliable than survey text. Direct model training on the Jetson is not planned for the initial phase.

## 10. Authentication and Security

### 10.1 Authentication Scenarios

| Actor | Mechanism | Cloud Required for Authentication |
| --- | --- | --- |
| Hub ↔ cloud | X.509 device certificate (mTLS) issued by Private CA, plus a short-lived access token containing `hub_id`, `ship_id`, and `epoch` | Yes |
| iPad ↔ hub | No cloud authentication. The hub validates a registered device token; staff select their name and enter a short PIN so audit logs identify the responsible person | No |
| Guest ↔ cloud | Firebase SSO: Guest Mobile exchanges a Firebase ID Token for backend access and refresh tokens | Yes |
| Guest ↔ hub | Cloud-signed boarding pass (QR); hub issues a local session associated with the booking, room, and device | No |
| Fleet Console administrator | Firebase SSO plus a backend administrator grant; Firebase claims never grant fleet authority | Yes |
| Technician ↔ hub | Short-lived SSH certificates, password authentication disabled, remote access through a hub-initiated channel and subject to approval | Usually |

### 10.2 Firebase SSO

- Firebase is the identity provider for Guest Mobile and Fleet Console. The backend verifies Firebase ID Tokens with the Firebase Admin SDK, including revocation checks, and maps the verified `uid` to an internal account through `account_identities` with `provider = 'firebase'`.
- The backend issues short-lived RS256 access JWTs (at most 15 minutes) and rotates opaque refresh tokens on every use. Only the SHA-256 hash is stored in RDS. Reuse of a used or revoked token revokes the entire token family.
- Staging and production sign tokens with KMS keys, publish public keys at `/api/auth/jwks`, and use `kid` for key rotation. Redis is required in every environment for shared authentication rate limits.
- Unknown guest identities are provisioned on exchange. Fleet Console access requires an existing active administrator grant and is never auto-provisioned. Accounts are never merged by email; a verified email collision requires an explicit identity link.
- Authentication events are audited, rate limits fail closed, and Firebase tokens are never stored or logged. See `docs/auth.md` for the full flow and client rules.
- The onboard hub does not use Firebase or backend session tokens; its local offline authentication remains independent.

### 10.3 Keys and Certificates

| Key | Storage | Purpose |
| --- | --- | --- |
| Package and AI-parameter signing key | KMS | Sign snapshots, deltas, and models. The hub stores public keys indexed by `kid` |
| Boarding-pass signing key | KMS | Sign boarding passes; separate from the JWT signing key |
| JWT signing key | KMS | Guest and administrator access tokens |
| Device CA | Private CA | Issue hub certificates |
| Hub private key | Generated on the device, in a hardware security area if supported | mTLS with cloud and signing local session tokens |

The hub is provisioned at the factory using a CSR and one-time code. Certificates renew automatically before expiry. A manual USB-based renewal procedure is available for extended periods without connectivity.

For key rotation: send the new key first, retain both keys during the transition period, and then revoke the old key. OP-TEE/fTPM support depends on the Jetson Linux version and must be verified against the documentation before finalizing the hardware choice.

### 10.4 Data Protection

- Encrypt hub disks, encrypt passport fields, use KMS for cloud at-rest encryption, and use TLS everywhere.
- The hub stores guest data only for the current voyage and deletes it after the voyage.
- Device and account revocation lists are included in sync packages and checked by the hub offline. There will be a delay between revocation and the hub receiving it, so local tokens must have short lifetimes.
- Maintain audit logs for sensitive operations and all AI decisions.
- Comply with GDPR and the data-protection regulations of customer countries.

## 11. Onboard Local Network

- Connect the hub to the core switch over Ethernet. Run the AP controller locally so loss of satellite connectivity does not disrupt onboard Wi-Fi.
- Use separate SSIDs and VLANs for staff and guests, with client isolation and a firewall between the networks. The hub listens on both VLANs using separate listeners. Guests must never be able to access staff APIs.
- Local DHCP and DNS must not depend on the internet. The hub uses a static IP or DHCP reservation. If a standby hub is available, use a virtual IP (VRRP).
- Conduct a wireless site survey before installation because the vessel's steel structure significantly attenuates radio signals. Enable fast roaming (802.11k/v/r).
- iPads discover the hub in this order: fixed hostname `hub.ship.local` through local DNS, mDNS, the primary hub's static IP, then the standby hub's IP.
- iOS considerations: grant Local Network permission; use TLS with a private CA installed through MDM (Let's Encrypt requires internet access); handle Wi-Fi networks without internet; disable private MAC addresses if DHCP reservations are MAC-based; use Single App Mode and keep the screen awake during shifts; do not rely on APNs while offline.
- Guest notifications: the primary channel is a direct connection to the hub while the app is open. Apple/Google push notifications are secondary and available when internet access exists. Accept that when the app is in the background without internet, notifications may not arrive until the app is opened.
- The hub records each iPad's `last_seen` so supervisors and technicians can identify devices that have been disconnected for an extended period.

## 12. Operations, Monitoring, and Backups

| Area | Approach |
| --- | --- |
| Cloud monitoring | CloudWatch: API latency, 5xx errors, queue depth and DLQs, database connections, and hub sync lag. Structured logs include `request_id`; use OpenTelemetry and X-Ray |
| Hub monitoring | When connected, the hub sends heartbeats and metrics (CPU, RAM, temperature, storage, and `sync_lag`) to the cloud |
| Cloud backups | Automated RDS backups, PITR, cross-region snapshots, and S3 versioning |
| Hub backups | Litestream replicates the SQLite file to the standby hub over LAN and to S3 when connected |
| Hub updates | A/B-style OTA updates, digitally signed, tested on one vessel before wider rollout |
| CI/CD | Linting, tests (real PostgreSQL through testcontainers), security scanning, staging deployment followed by production, and a rollback plan |
| Environments | dev, staging, prod, with separate AWS accounts for each environment |
| Testing | Disconnect tests (unplug the hub, turn off APs, restart the hub while iPads are in use), wireless site surveys, load tests using realistic device counts, and sync/offline tests |

## 13. Risks and Open Questions

### 13.1 Risks

| Risk | Mitigation |
| --- | --- |
| Hub is a single point of failure | Standby hub, Litestream, UPS, A/B updates, and failover drills |
| Hub clock drifts during extended network outages | Battery-backed RTC, vessel GPS/NTP, and ordering based on hub sequence numbers |
| AI generates an incorrect plan or violates regulations | Deterministic rule engine, autonomy levels, AI kill switch, and audit logs |
| Unstable local network | Wireless site survey, offline-first iPad app, and clear connection status |
| Guest data exposure if a device is lost | Disk encryption, retain only current-voyage data, revoke certificates, and increment epoch |
| Sync fails during the finalization window | Multiple sync phases, alerts, and signed USB import |
| Concurrent workload overloads Jetson | Separate processes, resource limits, and real-world benchmarking before selecting the model |

### 13.2 Open Questions

1. What does a guest booking slot include: rooms, services, and partner slots? (This document assumes all three.)
2. Which Jetson model and LLM should be used, after measuring RAM usage and latency together with the solver?
3. One hub or two hubs in the initial pilot?
4. What autonomy level should AI have for each type of change? This must be agreed with operations representatives.
5. Which maritime rest-hour regulations apply to each vessel flag? Legal expertise is required.
6. Who manages and supplies the vessel's Wi-Fi and switch infrastructure?
7. What is the scope of onboard payments and service folios?
8. How will partners register slots and staff: through the Fleet Console or a separate interface?
9. Will iPads be managed through MDM, and how many iPads and concurrent guest connections are expected?

## 14. Suggested Implementation Roadmap

| Phase | Scope |
| --- | --- |
| 1. Foundation | Terraform/CDK, CI/CD, RDS, ALB, ASG, OAuth, and factory hub provisioning with Private CA |
| 2. Master data and synchronization | Basic Fleet Console, `change_log`, snapshots and deltas, outbox, three-phase scheduler, and hub snapshot application |
| 3. Onboard operations | iPad app (shift schedules, tasks, notifications), rule engine, solver for initial shift scheduling, shift changes, emergency mode, and manual assignment |
| 4. Guests | Mobile app (bookings, boarding pass, room information, notifications), service bookings, and partner slots |
| 5. Advanced AI | Local LLM for free-form requests, forecasting model, and post-voyage feedback learning loop |
| 6. Reliability | Standby hub, device hardening, incident drills, and approved remote access |

Recommended early technical spikes: benchmark the solver and LLM on actual Jetson hardware; test synchronization over a satellite link with simulated latency; survey Wi-Fi on a real vessel; and test failover from the primary hub to the standby hub.

## 15. Detailed Data Model

### 15.1 Cloud (PostgreSQL)

| Group | Tables | Main Columns | Notes |
| --- | --- | --- | --- |
| Vessel | `ships`, `decks`, `rooms` | `ship_id`, `level`, `number`, `type`, `capacity`, `status` | Room statuses: vacant, occupied, needs cleaning, maintenance |
| Voyage | `voyages`, `port_calls`, `ports` | `departure_at`, `status`, `ownership`, `schedule_version`, `eta`, `etd`, `sequence` | `ownership` is `cloud` or `ship`; changes when the manifest is finalized |
| Staff | `staff`, `staff_availability` | `department`, `role`, `certifications`, `languages`, `status` | Security, cleaning, and operator groups |
| Partners | `partners`, `partner_staff`, `partner_slots` | `area_id`, `start_at`, `end_at`, `capacity`, `status` | Exclusion constraint prevents overlapping slots by area and time window |
| Accounts | `accounts`, `account_identities`, `refresh_tokens` | `(provider, sub)`, `token_hash`, `family_id`, `revoked_at` | Refresh tokens are stored as hashes and tracked by token family |
| Bookings | `bookings`, `guests`, `service_bookings` | `voyage_id`, `room_id`, `status`, `expires_at`, `idempotency_key` | Temporary holds. Passports and emergency contacts are encrypted at the field level |
| Devices | `hub_devices` | `hub_id`, `ship_id`, `epoch`, `cert_serial`, `status` | Epoch increments when a hub is replaced |
| Synchronization | `change_log`, `ship_seq`, `job_runs` | `seq`, `entity`, `op`, `version`, `payload`; `last_seq`; `(voyage_id, job_type, schedule_version)` | `ship_seq` assigns sequence numbers in the same transaction as the data |
| Received operations | `ops_events`, `audit_log` | `hub_id`, `local_seq`, `entity`, `payload` | Data from the hub outbox, deduplicated by `(hub_id, local_seq)` |
| AI | `ai_params` | `version`, `signature`, `status` | Versioned and digitally signed parameters and models |

Common columns: `id` (UUID v7), `ship_id`, `created_at`, `updated_at`, `version`, and `deleted_at`.

### 15.2 Hub (SQLite)

| Type | Tables | Notes |
| --- | --- | --- |
| Read-only replicas from cloud | `ships`, `decks`, `rooms`, current `voyage`, voyage staff, guests and bookings, `partner_slots`, `ai_params`, `revocations` | Only `sync-agent` may write |
| Hub-owned | `plan_versions`, `assignments`, `tasks`, `attendance`, `check_ins`, `service_orders`, `incidents`, `change_requests`, `ai_proposals`, `notifications` | Every change is recorded together with an outbox entry |
| Sessions and devices | `device_registry`, `staff_sessions`, `guest_sessions` | Local session tokens with short expiration |
| System | `sync_state` (`last_seq`, `schema_version`, `epoch`, `hub_id`), `outbox` (`local_seq`, `entity`, `payload`, `acked_at`), `audit_log` | Single writer, WAL |

### 15.3 Differences Between PostgreSQL and SQLite

- Store UUIDs as text, timestamps as UTC integers (milliseconds) or ISO text, and JSON as text.
- SQLite has neither exclusion constraints nor Row-Level Security. Prevent overlapping slots in the application layer using a `BEGIN IMMEDIATE` transaction.
- Maintain separate migration sets, but share a synchronization contract (table list, columns, and rules) and automated tests to prevent schema drift.
- The repository layer is an interface with two implementations (Postgres and SQLite); business logic is shared.

## 16. Main API Specifications

### 16.1 General Conventions

REST following OpenAPI, `/api` prefix, cursor-based pagination, a consistent error format, and an `Idempotency-Key` header for write operations. The hub also sends `X-Hub-Epoch` and `X-Schema-Version`.

Important error codes: 409 for an outdated epoch, 426 when the hub needs a schema upgrade, and 429 when the rate limit is exceeded.

### 16.2 Hub Calls to Cloud (mTLS)

| Endpoint | Method | Description |
| --- | --- | --- |
| `/api/hub/enroll` | POST | At the factory: submit a CSR with a one-time code and receive a certificate |
| `/api/hub/token` | POST | Exchange a certificate for a short-lived access token |
| `/api/sync/snapshot?voyage_id=` | GET | Return a presigned URL, base `seq`, checksum, and signature |
| `/api/sync/changes?after=&limit=` | GET | Return a batch of changes ordered by `seq` |
| `/api/sync/wait?after=` | GET | Long-poll for up to 30 seconds; return when a new `seq` is available |
| `/api/sync/outbox` | POST | Push a batch of operational data and return an `ack` |
| `/api/hub/heartbeat` | POST | Send health metrics and `sync_lag` |

### 16.3 iPad Calls to Hub

| Endpoint | Description |
| --- | --- |
| `POST /api/auth/device-login` | Registered device token plus staff name and PIN; returns a local session |
| `GET /api/me/shifts` | My shift schedule |
| `GET /api/tasks`, `POST /api/tasks/{id}/complete` | List tasks and mark a task as complete |
| `POST /api/shift-requests` | Submit a shift-change request using a structured form or free-form text |
| `GET /api/floors/{id}/guests/summary` | Guest count by deck |
| `POST /api/emergency/reassign` | Supervisor: request reassignment by area and role |
| `POST /api/assignments` | Supervisor: manual assignment, validated by the rule engine |
| `GET /api/events` | SSE: notifications and shift/task updates |

### 16.4 Guest Mobile App

| Location | Endpoint | Description |
| --- | --- | --- |
| Cloud | `/api/auth/oauth/{provider}` | Authorization Code + PKCE; backend issues JWT and refresh token |
| Cloud | `/api/bookings`, `/api/boarding-pass` | Create bookings and receive a signed boarding pass |
| Hub | `POST /api/guest/session` | Submit boarding pass and receive a local session |
| Hub | `GET /api/guest/room`, `GET /api/guest/notifications` | Room information and notifications |
| Hub | `GET/POST /api/guest/services/bookings` | View and book services, with idempotency |

## 17. Detailed Synchronization Protocol

### 17.1 Snapshot Package

A compressed file containing `manifest.json` (`voyage_id`, `seq_base`, `schema_version`, file list, SHA-256 hashes, signing-key `kid`, and signature) and zstd-compressed NDJSON files, one per table.

The hub applies the package atomically: write to a temporary database, verify checksums, signature, and row counts, then atomically rename it to the primary database. Retain several previous database versions for rollback.

### 17.2 Delta and Outbox Records

- Delta: `{seq, entity, id, op, version, data}`, where `op` is `upsert` or `delete` (tombstone).
- Outbox: `{hub_id, epoch, from_seq, to_seq, records[]}`. Each record contains `local_seq`, entity, and payload.
- The cloud deduplicates by `(hub_id, local_seq)` and returns an `ack` containing the highest contiguous sequence number received.

### 17.3 sync-agent State Machine

| State | Action | Transition |
| --- | --- | --- |
| Idle | Wait for `sync/wait` or the 60-second polling interval | Pulling |
| Pulling | Fetch a delta or snapshot | Applying |
| Applying | Apply changes in one transaction and update `last_seq` | Pushing |
| Pushing | Send outbox batches, processing priority items first | Idle |
| Backoff | On network errors or 5xx responses, retry with exponential backoff from 2 seconds up to 5 minutes, with jitter | Pulling or Pushing |

## 18. Shift Scheduling Data Model

| Concept | Description |
| --- | --- |
| `shift_templates` | Shift templates (morning, afternoon, night) by department, defined in the cloud |
| `demands` | Per-shift staffing demand by position and area, forecast from guest counts and historical voyage data |
| `plan_versions` | Versioned shift plans with scores and rationale. Each change creates a new version |
| `assignments` | Staff assigned to a shift and position within a plan version |
| `change_requests` | Shift-change or reassignment requests |
| `ai_proposals` | Solver-generated proposals, including affected staff and rationale |
| `rule_violations` | Hard or soft violations detected by the rule engine, including any overrides |

`change_request` lifecycle: `submitted → parsed → validated → proposed → auto_approved or pending_approval → applied`, or `rejected` and `expired`.

Limit the maximum number of people affected by each change (for example, three). At commit time, if `plan_versions` has changed, recalculate the plan.

## 19. Non-Functional Requirements

The figures below are initial proposed targets. They must be confirmed with operations and validated through real-world measurements before being committed.

| Area | Proposed Target |
| --- | --- |
| API latency on the hub LAN | Under 500 ms for normal operations |
| Automatic shift change | Produce a proposal within seconds, subject to the solver's time limit |
| Emergency reassignment | Return a proposal within seconds, always with a heuristic fallback |
| Cloud API latency | p95 under 300 ms |
| Hub restart | Tens of seconds; the app remains usable through offline-first behavior |
| Hub recovery | Fail over to the standby hub within a few minutes |
| Hub data loss in case of hardware failure | A few seconds (Litestream) |
| Cloud | Multi-AZ, automated backups, PITR |
| Delayed-sync alerts | Threshold configurable per vessel |
| Multilingual support | i18n from the outset for the iPad and guest mobile apps |
| Accessibility | Large text, high contrast, and fast interactions on iPad |

Before finalizing, estimate the workload: number of vessels, staff members, maximum guests, iPads, and concurrently connected guest devices per vessel.

## 20. Cost Factors

| Area | Main Cost Items |
| --- | --- |
| Cloud | EC2 (API and workers), RDS Multi-AZ, ElastiCache, NAT or VPC endpoints, data transfer, S3, KMS, and Private CA |
| Per vessel | Jetson (including standby hub), NVMe, UPS, switch and APs, iPads, and MDM |
| Connectivity | Satellite plan; therefore, synchronization data volume must be optimized |
| Operations | Monitoring, on-call team, and OTA update processes |

Development and staging environments can be shut down outside working hours and use smaller configurations to reduce costs.

## 21. Architecture Decision Records (ADR)

| ID | Decision | Rationale | Consequence |
| --- | --- | --- | --- |
| ADR-01 | EC2 + Auto Scaling instead of Kubernetes | Small team and fewer components | AMIs and rolling deployments must be managed |
| ADR-02 | Hub pulls data from cloud | Avoids NAT and eliminates the need to give AWS credentials to the hub | Long polling is needed to reduce latency |
| ADR-03 | SQLite on the hub | One vessel's data, one writer, and easy backups | Two migration sets; no exclusion constraints |
| ADR-04 | Change log and outbox instead of database replication | Different database engines and unreliable connectivity | Ordering and deduplication must be managed by the application |
| ADR-05 | Custom OAuth instead of Cognito | Desire for identity-system autonomy | Responsibility for token security is self-managed |
| ADR-06 | iPad does not authenticate with cloud | Only the hub communicates with the cloud | Local device tokens and PINs are required |
| ADR-07 | AI recommends; rule engine decides | Safety and compliance with rest-hour regulations | The independent rule engine must be maintained |
| ADR-08 | One hub plus a standby hub instead of leader election | Simpler and easier to verify | Initial failover requires a manual operation |
| ADR-09 | Booking ownership changes by phase | Each data item has one write owner | The app must understand `ownership` |

## 22. Test Plan

| Type | Scope |
| --- | --- |
| Unit and integration | Real PostgreSQL (testcontainers), real SQLite, and OpenAPI contract tests |
| Synchronization contract | Automated tests ensure PostgreSQL and SQLite schemas match and that replaying an old batch is harmless |
| Offline and networking | Disconnect mid-operation, restart the hub while iPads are in use, and simulate satellite latency |
| AI | Test the rule engine against edge cases, compare solver results with heuristics, and run new parameters in shadow mode |
| Load | Realistic numbers of iPads and guest devices, concurrent SSE connections, and cloud load based on fleet size |
| Security | Dependency and image scanning, OAuth authentication tests, device revocation, and VLAN isolation checks |
| Field testing | Wireless site survey and pilot on one vessel before scaling out |

## 23. Glossary

| Term | Meaning |
| --- | --- |
| Hub | Small Jetson server onboard; source of operational data while offline |
| Master data | Core data managed by the cloud: vessels, rooms, staff, partners, and guests |
| Manifest | Guest and room list for a voyage |
| Snapshot and delta | Full data package and incremental changes sent from cloud to hub |
| Outbox | Queue of operational changes on the hub waiting to be pushed to the cloud |
| Epoch | Hub generation number, incremented when hardware is replaced to reject data from the old device |
| Idempotency | Repeating the same request multiple times does not create duplicate results |
| Rule engine | Deterministic rule-validation component; not AI |
| Muster | Gathering and accounting for people during an emergency |
| MDM | Mobile Device Management system used to provision and manage iPads |
