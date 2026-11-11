# Feature Specification: Ship Onboarding with Excel Structure Import

**Feature Branch**: `004-ship-structure-import`

**Created**: 2026-11-11

**Status**: Draft

**Input**: User description: "Refactor the existing Ship Management APIs to support a new ship onboarding workflow. Create a Ship independently through a standard form, then upload a multi-sheet Excel workbook containing the ship's Decks, Areas, and Rooms. Validate the workbook, show an import preview, and on confirmation create all related records transactionally while preserving the existing PostgreSQL schema, UUID v7 conventions, change-log triggers, and Cloud-to-Hub synchronization contract."

## Clarifications

### Session 2026-11-11

- Q: Should workbook validation run synchronously during the upload request, or be queued for asynchronous background processing? → A: Synchronous validation during upload, with lifecycle states still recorded.
- Q: After an import fails validation, should the administrator retry by uploading the corrected workbook as a new import, or by replacing and revalidating the existing import? → A: Upload the corrected workbook as a new import with a new `importId`; failed imports remain visible as history.
- Q: Should `Idempotency-Key` be enforced only where the existing schema supports persistent keys, or should this feature add a shared idempotency store for all writes? → A: Follow existing conventions; use the `importId` lifecycle for import idempotency instead of adding a shared store.
- Q: Should structure import be allowed only for ships with no existing structure, or may it add new non-conflicting structure to a partially populated ship? → A: Allow additive imports into any accessible ship; reject only records that conflict with existing non-deleted Decks or Rooms.
- Q: Should upload, validation, and confirmation create audit-log entries, or should the feature rely only on import-status records and structured logs? → A: Audit only confirmation, the data-changing step, with actor, import, and created-record counts.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create and Manage a Ship Independently (Priority: P1)

An administrator creates a vessel from a standard ship form without providing any deck, area, or room information. The system persists the vessel and returns a new `shipId` that can be used in the later structure-import steps. The administrator can also list, view, update, and soft-delete vessels according to existing access rules.

**Why this priority**: Vessel identity is the prerequisite for every later onboarding step. Separating vessel creation from structure import removes the need to know the entire physical layout before the vessel exists.

**Independent Test**: Create a ship with only vessel-level fields, verify a `shipId` is returned, then retrieve, update, list, and soft-delete it. Confirm that no Deck, Area, or Room is required or created.

**Acceptance Scenarios**:

1. **Given** an authorized administrator and valid vessel fields, **When** the administrator creates a ship, **Then** the ship is persisted with a newly generated unique identifier, a `shipId` is returned, and no structure records are required.
2. **Given** a ship payload with a missing or blank name, an invalid time zone, an unsupported status, an invalid IMO value, or a duplicate IMO number, **When** the ship is created or updated, **Then** the request is rejected with a consistent validation or conflict error and no incorrect record is persisted.
3. **Given** an existing ship with no decks, areas, or rooms, **When** the administrator retrieves or updates it, **Then** the operation succeeds and the vessel remains independent of its structure.
4. **Given** an existing ship, **When** the administrator soft-deletes it, **Then** the ship is hidden from normal reads and further structure imports, the stored record is retained, and existing soft-deletion semantics are preserved.

---

### User Story 2 - Upload a Structure Workbook and Receive a Validation Preview (Priority: P1)

An administrator downloads the provided Excel template, fills in the ship's Decks, Areas, and Rooms, and uploads the workbook for a previously created ship. The system validates the whole workbook and returns an `importId`, an import status, and a preview containing totals, valid and invalid counts, and row-level errors. No structure records are created at this stage.

**Why this priority**: Administrators must be able to detect and correct workbook errors before any vessel data is committed. A complete preview is the safety gate for onboarding a large vessel.

**Independent Test**: Upload a valid workbook and a set of deliberately invalid workbooks for an existing ship, inspect the preview responses, and verify that no Deck, Area, or Room record exists until confirmation.

**Acceptance Scenarios**:

1. **Given** an existing ship and a valid workbook containing the required sheets and columns, **When** the administrator uploads it, **Then** the system creates an import identified by a unique `importId`, associates it with that ship, validates the workbook, and returns a preview with totals and valid counts for Decks, Areas, and Rooms.
2. **Given** a workbook with a missing sheet, missing required column, missing required value, invalid data type, unsupported enum value, duplicate deck level, duplicate room number, invalid capacity, malformed layout JSON, or an Area or Room referencing an unknown deck level, **When** the workbook is uploaded, **Then** the preview reports each problem with the sheet name, row, column, and a safe message, and the import does not become confirmable.
3. **Given** a workbook whose Deck levels or Room numbers conflict with existing non-deleted records of the ship, **When** the workbook is uploaded, **Then** the conflict is reported as a validation error and confirmation is blocked.
4. **Given** a successful upload and preview, **When** the administrator lists the ship's decks, areas, and rooms, **Then** none of the workbook records have been persisted.
5. **Given** an `importId` that belongs to another ship or tenant, **When** the administrator requests its preview, **Then** access is denied without revealing whether the import exists.

---

### User Story 3 - Confirm the Import Atomically (Priority: P1)

The administrator reviews the preview and confirms a validated import. The system creates all Decks first, resolves their generated identifiers, creates all Areas and Rooms with the correct ship and deck relationships, and completes the import as one all-or-nothing operation.

**Why this priority**: Partial ship structures would be unsafe and difficult to repair. Atomic confirmation is what makes bulk onboarding trustworthy.

**Independent Test**: Confirm a ready import and verify every Deck, Area, and Room exists with correct relationships and ship ownership. Then force a persistence failure and verify that no imported structure remains and that the import reports a safe failure.

**Acceptance Scenarios**:

1. **Given** a ready import with only valid records, **When** the administrator confirms it, **Then** every Deck is created, every Area and Room is associated with the correct generated Deck, every imported record carries the correct ship identifier, and the import reaches a completed state.
2. **Given** a ready import, **When** any persistence operation fails, **Then** the entire structure import is rolled back, no partial Deck, Area, or Room records remain, and the import records a safe failure for the administrator.
3. **Given** an import that is still validated as failed, is not ready, or is already completed, **When** the administrator attempts confirmation, **Then** confirmation is rejected or returns the already-completed result without creating duplicate records.
4. **Given** repeated or concurrent confirmation requests for the same import, **When** they are processed, **Then** exactly one structure is persisted and every other request observes the current import state without duplicating records.
5. **Given** a validated import, **When** confirmation succeeds, **Then** the import summary reports the number of created Decks, Areas, and Rooms.

---

### User Story 4 - Track Import Status and Recover Safely (Priority: P2)

An administrator can check the current state of any structure import, see its counts and errors, and understand whether action is still required. Failed imports can be retried safely, while completed imports cannot be processed again.

**Why this priority**: Onboarding is a multi-step administrative task. Clear status and safe retries prevent duplicate work and give administrators a recovery path when a workbook or persistence step fails.

**Independent Test**: Drive an import through its lifecycle, query its status at each stage, retry a failed import, and re-confirm a completed import. Verify that no duplicate structure is created.

**Acceptance Scenarios**:

1. **Given** an import for an accessible ship, **When** the administrator requests its status, **Then** the system returns its current lifecycle state, file metadata, record counts, timestamps, and any safe error summary.
2. **Given** an import that failed validation, **When** the administrator uploads the corrected workbook as a new import, **Then** the new attempt is validated independently, failed imports remain visible as history, and no prior attempt creates duplicate structure records.
3. **Given** a completed import, **When** the administrator attempts to confirm or reprocess it, **Then** the system rejects reprocessing and returns the completed outcome without duplicate records.
4. **Given** an import that is interrupted during processing, **When** the system recovers, **Then** no partial structure is exposed and the import can be safely retried.

---

### User Story 5 - Download a Template Aligned with Validation (Priority: P2)

An administrator downloads an Excel template containing the Decks, Areas, and Rooms sheets, column headers and explanations, valid example rows, and the allowed enum values. The template explains how Areas and Rooms reference Decks through `deck_level`.

**Why this priority**: A correct template is the primary defense against invalid uploads and support requests, especially for vessels with many records.

**Independent Test**: Download the template, fill it with the provided examples, upload it for a ship, and verify that the preview accepts the examples. Verify that every documented enum value matches the validation rules.

**Acceptance Scenarios**:

1. **Given** an administrator requests the template, **When** the download completes, **Then** the workbook contains the required Decks, Areas, and Rooms sheets with all required column headers.
2. **Given** the template, **When** the administrator inspects it, **Then** it includes column descriptions, valid example rows, all allowed `area_type` and `room_type` values, and an explanation of `deck_level` references.
3. **Given** the template examples, **When** they are uploaded without modification, **Then** validation accepts them as valid records.
4. **Given** a change to an import validation rule, **When** the template is generated, **Then** its headers, examples, and documented values remain aligned with that rule.

---

### User Story 6 - Make Imported Structure Available for Offline Operations (Priority: P2)

After confirmation, the imported decks, areas, and rooms become part of the vessel's synchronized master data. The existing Cloud-to-Hub change-log and snapshot/delta protocol makes the new structure available to the vessel's hub with the same version and deletion semantics as records created individually.

**Why this priority**: Onboarded structure is only useful if the vessel can operate with it offline. Bypassing synchronization would make the import appear successful while leaving the hub incomplete.

**Independent Test**: Confirm an import, inspect the existing change-log stream for the ship, and verify that each imported record is present with correct ordering, version, and operation semantics. Apply the same checks after updating and soft-deleting an imported record.

**Acceptance Scenarios**:

1. **Given** a confirmed import, **When** the vessel's change-log stream is read, **Then** each imported Deck, Area, and Room is represented as a change for that ship.
2. **Given** an imported record, **When** it is later updated or soft-deleted, **Then** `updated_at`, `version`, and deletion semantics follow the existing database-managed behavior.
3. **Given** the structure import, **When** records are persisted, **Then** synchronization triggers and mechanisms are not bypassed or disabled.
4. **Given** a hub synchronized from the vessel's change log, **When** the import has completed, **Then** the hub receives the complete structure required for offline operations.

---

### Edge Cases

- The uploaded file is renamed with an `.xlsx` extension but its actual content is not a valid Excel workbook, is corrupted, is encrypted, or is a different spreadsheet format.
- The workbook exceeds the allowed file size or contains more data rows than the supported limit.
- The workbook is missing one or more required sheets, has duplicated sheet names, or uses renamed, reordered, or extra columns.
- A required cell is blank, contains only whitespace, or contains a value of the wrong type.
- A Deck level is not a non-negative whole number, is duplicated within the workbook, or conflicts with an existing non-deleted Deck of the same ship.
- A Room number is duplicated within the workbook or conflicts with an existing non-deleted Room of the same ship.
- An Area or Room references a `deck_level` that is absent from the Decks sheet, appears in a malformed row, or refers to a deck that failed validation.
- An Area has a negative capacity; a Room has a zero, negative, or non-numeric capacity.
- An optional `layout` value is present but is not syntactically valid JSON; a missing layout is accepted when the existing schema allows it.
- An `is_bookable` value is blank, is not a recognized boolean representation, or uses mixed case.
- The workbook has no Deck records but contains Areas or Rooms; or the workbook contains no importable records at all.
- The ship is soft-deleted, belongs to another tenant, or is not accessible to the caller before upload, during preview, or during confirmation.
- The `importId` does not exist, belongs to another ship, or is guessed by an unauthorized caller.
- Confirmation is attempted while validation is still running, after validation failed, or after a previous successful confirmation.
- Two administrators confirm the same import at the same time, or one administrator retries after a timeout.
- Persistence fails on a database constraint after some Decks have already been inserted.
- The system restarts during validation or confirmation, leaving an import in an in-progress state.
- Template examples no longer match the validation rules after an enum or column change.
- Imported records are created concurrently with an unrelated change to the same ship's structure.

## Requirements *(mandatory)*

### Functional Requirements

#### Ship Onboarding

- **FR-001**: The system MUST allow an authorized administrator to create a ship independently from its physical structure. Ship creation MUST NOT require any Deck, Area, or Room information.
- **FR-002**: Ship creation MUST accept only fields supported by the existing ship domain: name, IMO number, flag state, time zone, and status. No new ship fields may be introduced.
- **FR-003**: The system MUST require a non-blank ship name, assign the existing default status when no status is supplied, and accept only the statuses already supported by the ship domain.
- **FR-004**: The system MUST validate the time zone using the existing time-zone rules and default it consistently with current behavior when omitted.
- **FR-005**: The system MUST preserve the existing IMO-number rules. If the current domain enforces only uniqueness and optional presence, no new checksum or format rule may be introduced by this feature.
- **FR-006**: The system MUST generate a unique UUID v7 identifier for each new ship according to project conventions and MUST return the created ship information, including `shipId`.
- **FR-007**: The system MUST retain existing list, detail, update, and soft-delete behavior for ships, including tenant and ship access boundaries and consistent error responses.
- **FR-008**: Ship soft-deletion MUST remain a soft delete, MUST exclude the ship from normal reads and new structure imports, and MUST preserve existing version and change-log behavior.
- **FR-009**: Ship creation and management MUST reuse the existing ship APIs where equivalents already exist rather than introducing duplicate endpoints.
- **FR-010**: All new Ship Management endpoints MUST be exposed under `/api`, not `/v1`.

#### Workbook and Template

- **FR-011**: The system MUST provide a downloadable Excel template containing the Decks, Areas, and Rooms sheets.
- **FR-012**: The template MUST include column headers, a description of each column, valid example rows, the full list of allowed `area_type` and `room_type` values, and an explanation of how `deck_level` links Areas and Rooms to Decks.
- **FR-013**: The template MUST stay aligned with the actual import validation rules whenever those rules change.
- **FR-014**: The Decks sheet MUST define `level` and `name`. A valid Deck row MUST have a whole-number level that satisfies the existing deck-level rule and a non-blank name.
- **FR-015**: The Areas sheet MUST define `deck_level`, `name`, `area_type`, `capacity`, and `layout`. A valid Area row MUST have a non-blank name, one of the existing `area_type` values, a valid non-negative capacity when supplied, and a valid `deck_level` reference.
- **FR-016**: The Rooms sheet MUST define `deck_level`, `number`, `room_type`, `capacity`, `is_bookable`, and `layout`. A valid Room row MUST have a non-blank number and a `room_type` from the existing room-type values, and MUST have a capacity greater than zero.
- **FR-017**: `is_bookable` MUST accept recognized boolean representations and MUST default to the existing schema default when blank.
- **FR-018**: `layout` MUST be optional and, when present, MUST be syntactically valid JSON acceptable to the existing layout field. No new layout shape that is not supported by the existing domain may be required.
- **FR-019**: Required sheets MUST be matched by name. Required columns MUST be matched by name in a case-insensitive, whitespace-trimmed manner; extra columns MAY be ignored, but missing required columns MUST be reported as errors.
- **FR-020**: The system MUST reject workbooks that do not contain a usable Decks sheet or that contain no importable records.
- **FR-021**: Deck levels MUST be unique within the workbook and MUST NOT conflict with existing non-deleted Deck levels of the selected ship.
- **FR-022**: Room numbers MUST be unique within the workbook and MUST NOT conflict with existing non-deleted Room numbers of the selected ship.
- **FR-023**: Every Area and Room MUST reference a Deck level declared in the same workbook. References to decks that are missing from the workbook, invalid, or not importable MUST be rejected.
- **FR-024**: The import MUST NOT invent any workflow, partner, voyage, staff, guest, or booking fields beyond the existing Deck, Area, and Room domain.
- **FR-070**: The import MUST be allowed for any accessible, non-deleted ship, including a partially populated ship. Only records that conflict with existing non-deleted Deck levels or Room numbers may block the import; existing records MUST NOT be modified, replaced, or removed.

#### Upload, Validation, and Preview

- **FR-025**: The system MUST provide a workbook-upload operation for an existing, accessible ship and MUST create a unique import identified by an `importId` associated with that ship. Upload MUST perform validation synchronously and return only after validation completes, while still recording the import lifecycle states.
- **FR-026**: The system MUST validate the actual content of the uploaded file, not only its file extension.
- **FR-027**: The system MUST accept only the supported workbook format and MUST enforce a maximum file size and maximum data-row count per sheet.
- **FR-028**: The system MUST reject corrupted, encrypted, macro-enabled, unreadable, or otherwise malformed workbooks with a safe validation error.
- **FR-029**: Validation MUST cover workbook structure, required sheets and columns, missing required values, invalid data types, unsupported enum values, duplicate Deck levels, duplicate Room numbers, nonexistent Deck references, invalid capacity values, invalid layout JSON, and conflicts with existing database records.
- **FR-030**: Validation MUST report every detected problem with the sheet, row, column, and a safe, actionable message whenever those coordinates are available.
- **FR-031**: The preview MUST report total records, valid records, and invalid records for each of Decks, Areas, and Rooms.
- **FR-032**: The preview MUST include row-level validation errors and a summary of the records that would be created on confirmation.
- **FR-033**: The system MUST NOT persist Deck, Area, or Room records during upload, validation, or preview.
- **FR-034**: The system MUST NOT silently skip invalid records. If any record is invalid or conflicts with existing data, the import MUST NOT become confirmable.
- **FR-035**: The system MUST provide an import-status operation that returns the import's current lifecycle state, file metadata, record counts, timestamps, and safe error summary.
- **FR-036**: Validation results and preview data MUST remain available after the upload request completes so the administrator can review and confirm the import later.
- **FR-037**: The system MUST provide a consistent error envelope and stable error codes for upload, validation, status, preview, and confirmation failures.

#### Import Lifecycle and Confirmation

- **FR-038**: Each structure import MUST have a lifecycle with the states `uploaded`, `validating`, `ready`, `failed`, `processing`, and `completed`.
- **FR-039**: The system MUST enforce valid lifecycle transitions: an uploaded import may become validating; a validating import may become ready or failed; only a ready import may become processing; a processing import may become completed or failed; completed imports MUST NOT be reprocessed.
- **FR-040**: The system MUST prevent confirmation of an import that has not completed successful validation.
- **FR-041**: Confirmation MUST create Decks first, resolve their generated identifiers, then create Areas and Rooms with the correct Deck identifiers and the correct ship identifier for every record.
- **FR-042**: Confirmation MUST maintain referential integrity for every imported record and MUST preserve the existing ship-to-deck, ship-to-area, ship-to-room, deck-to-area, and deck-to-room relationships.
- **FR-043**: Confirmation MUST be an all-or-nothing operation. If any record cannot be persisted, the entire structure import MUST be rolled back and no partial structure may remain.
- **FR-044**: The system MUST prevent duplicate confirmation and concurrent processing of the same import. Repeated requests MUST observe the existing outcome without creating duplicate records.
- **FR-045**: An import that failed validation MUST be retried by uploading the corrected workbook as a new import with a new `importId`; failed imports MUST remain visible as history. A processing failure may be retried safely, and retries MUST NOT duplicate structure created by an earlier attempt.
- **FR-046**: An import that is interrupted during processing MUST be recoverable without exposing partial structure or requiring duplicate data entry.
- **FR-047**: The import lifecycle and its outcomes MUST be persisted so they survive process restarts.
- **FR-048**: If import tracking requires additional database storage, its purpose MUST be documented and delivered through a versioned migration. No staging table or other abstraction may be added without a concrete requirement.

#### Database and Synchronization

- **FR-049**: The existing PostgreSQL schema for ships, decks, areas, and rooms MUST remain the source of truth. No existing field or relationship may be redesigned without a verified implementation gap.
- **FR-050**: Imported records MUST use the existing UUID v7 generation conventions and MUST preserve the existing `version`, `updated_at`, and `deleted_at` behavior.
- **FR-051**: Imported records MUST be created through normal persistence paths so the existing change-capture triggers continue to generate change-log records and per-ship sequence numbers.
- **FR-052**: Imported structure MUST be available to the existing Cloud-to-Hub change-log and snapshot/delta synchronization mechanism with the same semantics as individually created records.
- **FR-053**: The import MUST NOT bypass, disable, or defer required database triggers or synchronization mechanisms during bulk insertion.
- **FR-054**: The Hub schema MUST NOT be changed unless the Cloud schema or synchronization contract actually changes.
- **FR-055**: Imported records MUST remain subject to existing soft-deletion, versioning, and conflict-constraint behavior after confirmation.

#### Security, Access, and Operational Safety

- **FR-056**: The system MUST verify that the ship referenced by an upload exists, is not soft-deleted, and is accessible to the caller.
- **FR-057**: The system MUST verify that an `importId` belongs to the ship in the request path and that the caller is authorized for that ship and tenant.
- **FR-058**: The system MUST enforce the existing authorization rules on every Ship Management and structure-import endpoint.
- **FR-059**: The system MUST not trust client-provided `shipId` or `importId` values without ownership and access verification.
- **FR-060**: The system MUST NOT expose internal file paths, storage locations, stack traces, SQL details, or sensitive processing information in API responses or logs.
- **FR-061**: Temporary or uploaded files MUST be cleaned up according to the project's storage lifecycle.
- **FR-062**: Write operations MUST follow the project's `Idempotency-Key` convention, including persistent enforcement only where the underlying schema supports it. Repeated identical requests MUST NOT duplicate effects, and reuse of a key with a different payload MUST be rejected consistently with existing conventions. This feature MUST NOT introduce a shared idempotency store.
- **FR-063**: The import lifecycle itself MUST provide idempotent upload and confirmation behavior where persistent idempotency is not otherwise available for the underlying records.
- **FR-064**: The feature MUST NOT change unrelated APIs, introduce a new microservice, or add distributed infrastructure outside the existing modular monolith.
- **FR-071**: Confirmation MUST emit an audit entry through the existing audit interface in the same transaction as the structure persistence. The entry MUST record the acting principal, ship, import identifier, imported record counts, and outcome, and MUST NOT include raw workbook content or sensitive file details. If the audit entry cannot be persisted, the confirmation MUST fail and roll back.

#### Verification Requirements

- **FR-065**: Tests MUST cover successful ship creation, rejection of invalid ship data, ship retrieval, update, and soft-deletion.
- **FR-066**: Tests MUST cover valid workbook upload, invalid file format, missing sheets or columns, invalid data types and enum values, duplicate Deck levels, duplicate Room numbers, unknown Deck references, row-level validation errors, and correct preview generation.
- **FR-067**: Tests MUST cover successful import of Decks, Areas, and Rooms, preservation of all foreign-key relationships, rollback on persistence failure, duplicate and concurrent confirmation, and expected change-log records from database triggers.
- **FR-068**: Tests MUST verify that imported records are available to the existing synchronization mechanism and retain correct version and deletion semantics.
- **FR-069**: Tests MUST run through the project's existing test commands and MUST use the project's real database test infrastructure where database behavior is involved.

### Required API Surface

The following external operations are required. Existing equivalent operations MUST be reused and refactored rather than duplicated.

| Method | Endpoint | Responsibility |
|---|---|---|
| POST | `/api/ships` | Create a ship |
| GET | `/api/ships` | List ships |
| GET | `/api/ships/{shipId}` | Get ship details |
| PUT/PATCH | `/api/ships/{shipId}` | Update ship information |
| DELETE | `/api/ships/{shipId}` | Soft-delete a ship according to existing rules |
| POST | `/api/ships/{shipId}/structure/imports` | Upload a structure workbook |
| GET | `/api/ships/{shipId}/structure/imports/{importId}` | Get import status |
| GET | `/api/ships/{shipId}/structure/imports/{importId}/preview` | Get the validation preview |
| POST | `/api/ships/{shipId}/structure/imports/{importId}/confirm` | Confirm and persist the import |
| GET | `/api/ships/structure/import-template` | Download the Excel template |

### Key Entities *(include if feature involves data)*

- **Ship**: A vessel owned by a tenant. It holds vessel identity and operational settings and can exist before any physical structure is defined.
- **Deck**: A level of the ship's physical structure. A Deck belongs to exactly one ship and has a level and name. Deck levels are unique per ship among non-deleted records.
- **Area**: A common space on a deck, such as a restaurant, lobby, or pool. An Area belongs to one ship and one Deck and may carry a capacity and layout data.
- **Room**: A bookable or operational room on a deck. A Room belongs to one ship and one Deck, has a room number unique per ship among non-deleted records, and has a type and positive capacity.
- **Ship Structure Import**: A tracked import attempt for one ship. It holds the unique import identifier, owning ship, lifecycle state, file metadata, per-entity counts, validation findings, timestamps, and the outcome of confirmation.
- **Import Validation Finding**: A row-level problem associated with an import, identifying the sheet, row, column, and a safe explanation of the rule that failed.
- **Excel Template**: The downloadable workbook that documents the supported sheets, columns, examples, enum values, and Deck-reference rules.
- **Change Log Entry**: The existing synchronization record generated when a ship, deck, area, or room is inserted or changed, used to deliver imported structure to the vessel hub.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An administrator can create a vessel with vessel-level information only, and the vessel is available for structure import, in under 1 minute.
- **SC-002**: An administrator can onboard a vessel containing 20 decks, 100 areas, and 500 rooms using the template, upload, preview, and confirmation workflow in under 5 minutes, excluding workbook-filling time.
- **SC-003**: At least 95% of valid workbooks within the supported size limits return a complete preview within 30 seconds, and confirmation of the same structure completes within 60 seconds.
- **SC-004**: 100% of validation findings include the affected sheet, row, and column when those coordinates are available, and the preview reports totals and valid/invalid counts for every entity.
- **SC-005**: 100% of workbooks containing any invalid or conflicting record are blocked from confirmation, with no invalid record silently skipped.
- **SC-006**: Zero partially imported structures remain after any failed confirmation.
- **SC-007**: Repeated or concurrent confirmation attempts produce exactly one persisted structure for an import.
- **SC-008**: 100% of imported Deck, Area, and Room records appear in the existing change-log/synchronization stream with the same version and deletion semantics as individually created records.
- **SC-009**: 100% of attempts to access another tenant's or another ship's import are denied without revealing protected data.
- **SC-010**: Administrators no longer need to create Decks, Areas, or Rooms individually to onboard a vessel with the supported target volume.

## Assumptions

- The feature description names `yacht_management_system_architecture_en.md`, `cloud_schema_postgres.sql`, and `hub_schema_sqlite.sql`, but those files are not present in the repository. The repository's authoritative equivalents are `docs/system-architecture.md` and `docs/schema.sql`, consistent with the project constitution. No separate Hub SQLite schema file was available for review.
- The existing PostgreSQL schema for `ships`, `decks`, `areas`, and `rooms` is authoritative and will not be redesigned. The existing relationships, constraints, UUID v7 convention, versioning, soft deletion, and trigger behavior are retained.
- The import is additive and creation-only. It may add new non-conflicting structure to a partially populated ship, but it does not merge, rename, update, or replace existing decks, areas, or rooms.
- Areas and Rooms in a workbook reference only Deck levels declared in the same workbook. Importing new Areas or Rooms into pre-existing decks is not part of this feature, and Deck-level conflicts with existing records are rejected.
- Soft-deleted Deck levels and Room numbers may be reused according to the existing partial-unique-index behavior. Only conflicts with non-deleted records block an import.
- Validation is synchronous during the upload request because no existing worker or queue infrastructure was identified for this workflow. The lifecycle states are still tracked after validation completes so the workflow can be moved to asynchronous processing later without changing the business contract.
- A new import-tracking record is necessary to persist status, counts, validation findings, outcomes, and the normalized validated rows required to render a later preview and perform confirmation across requests and restarts. Its purpose is limited to import orchestration: it is not a staging copy of business records, is never used as a source of truth for ship structure, is excluded from synchronization, and is superseded by the records created at confirmation. It will require a versioned migration.
- The supported upload format is `.xlsx`. Legacy `.xls`, CSV, password-protected workbooks, and macro-enabled workbooks are out of scope.
- Reasonable protective defaults are assumed: a maximum upload size of 10 MiB and a maximum of 5,000 data rows per sheet. These limits may be tightened during planning but must be documented and enforced consistently.
- Required workbook sheets are `Decks`, `Areas`, and `Rooms`. Headers are matched case-insensitively with surrounding whitespace trimmed; extra columns are ignored.
- Area capacity is optional and must be non-negative when supplied. Room capacity is required and must be greater than zero, matching the existing domain behavior.
- The current IMO rule is uniqueness with optional presence; no checksum or additional format validation is introduced unless one already exists in the implementation.
- Existing business endpoints are mounted under `/api` at runtime, while some current OpenAPI material and tests still describe `/v1`. This feature treats `/api` as canonical for new endpoints and requires reconciliation of documentation and contract tests during planning.
- Idempotency for ship writes follows the existing platform convention. Where persistent idempotency is not schema-backed, the import identifier and lifecycle state provide equivalent protection against duplicate uploads and confirmations.
- Audit coverage for this feature is limited to confirmation through the existing audit module. Upload and preview remain observable through the persisted import-status record and structured logs, not through new audit entries.
- The existing authorization model, error envelope, logging, and storage lifecycle are reused. No new authentication or authorization mechanism is introduced.

## Dependencies

- The existing Ship Management module, its Deck, Area, and Room operations, and the PostgreSQL schema.
- The existing authentication and tenant/ship authorization middleware.
- The existing database trigger and change-log synchronization mechanism.
- The existing testing infrastructure for unit and PostgreSQL-backed integration tests.
- The repository's authoritative architecture and schema documents: `docs/system-architecture.md` and `docs/schema.sql`.

## Out of Scope

- Updating, merging, or deleting existing structure records through an import.
- Importing voyages, ports, staff, partners, guests, bookings, or any entity outside Decks, Areas, and Rooms.
- Importing ship-level fields through the workbook or changing a ship's lifecycle state through an import.
- CSV, legacy `.xls`, password-protected, or macro-enabled workbook support.
- Semantic validation of layout coordinates beyond the existing layout field rules.
- A new queue, worker service, microservice, or distributed processing system.
- Changes to the Hub SQLite schema or synchronization contract that are not required by a verified Cloud schema change.
- Manual per-record UI workflows for creating decks, areas, or rooms.
- Tenant provisioning, fleet-console administration, and any feature outside ship onboarding.
- An IMO checksum rule that does not already exist in the current implementation.
