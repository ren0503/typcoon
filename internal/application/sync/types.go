package sync

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CreateHubDeviceParams struct {
	ID         uuid.UUID `json:"id"`
	ShipID     uuid.UUID `json:"ship_id"`
	Name       string    `json:"name"`
	Role       string    `json:"role"`
	CertSerial *string   `json:"cert_serial"`
}

type ListHubDevicesParams struct {
	ShipID    uuid.UUID     `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type UpdateHubDeviceStatusParams struct {
	Status string    `json:"status"`
	Epoch  int32     `json:"epoch"`
	ID     uuid.UUID `json:"id"`
}

type CreateShipDeviceParams struct {
	ID         uuid.UUID `json:"id"`
	ShipID     uuid.UUID `json:"ship_id"`
	DeviceName string    `json:"device_name"`
	DeviceType string    `json:"device_type"`
	TokenHash  []byte    `json:"token_hash"`
}

type ListShipDevicesParams struct {
	ShipID    uuid.UUID     `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateRevocationParams struct {
	ID          uuid.UUID  `json:"id"`
	ShipID      uuid.UUID  `json:"ship_id"`
	SubjectType string     `json:"subject_type"`
	SubjectID   string     `json:"subject_id"`
	Reason      *string    `json:"reason"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type ListRevocationsParams struct {
	ShipID    uuid.UUID     `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateSnapshotParams struct {
	ID         uuid.UUID     `json:"id"`
	ShipID     uuid.UUID     `json:"ship_id"`
	VoyageID   uuid.NullUUID `json:"voyage_id"`
	Phase      string        `json:"phase"`
	SeqBase    int64         `json:"seq_base"`
	StorageKey string        `json:"storage_key"`
	Sha256     string        `json:"sha256"`
	SizeBytes  *int64        `json:"size_bytes"`
	Kid        string        `json:"kid"`
	Signature  string        `json:"signature"`
	Status     string        `json:"status"`
}

type ListSnapshotsParams struct {
	ShipID    uuid.UUID     `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateJobRunParams struct {
	ID              uuid.UUID `json:"id"`
	VoyageID        uuid.UUID `json:"voyage_id"`
	JobType         string    `json:"job_type"`
	ScheduleVersion int32     `json:"schedule_version"`
}

type FinishJobRunParams struct {
	Status string    `json:"status"`
	Error  *string   `json:"error"`
	ID     uuid.UUID `json:"id"`
}

type ListJobRunsParams struct {
	VoyageID  uuid.UUID     `json:"voyage_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type UpsertOpsEventParams struct {
	HubID      uuid.UUID       `json:"hub_id"`
	LocalSeq   int64           `json:"local_seq"`
	ShipID     uuid.UUID       `json:"ship_id"`
	VoyageID   uuid.NullUUID   `json:"voyage_id"`
	Epoch      int32           `json:"epoch"`
	Entity     string          `json:"entity"`
	EntityID   string          `json:"entity_id"`
	Op         string          `json:"op"`
	Payload    json.RawMessage `json:"payload"`
	OccurredAt time.Time       `json:"occurred_at"`
}

type ListOpsEventsParams struct {
	ShipID        uuid.UUID `json:"ship_id"`
	AfterReceived time.Time `json:"after_received"`
	PageLimit     int32     `json:"page_limit"`
}

type ListChangeLogParams struct {
	ShipID    uuid.UUID `json:"ship_id"`
	AfterSeq  int64     `json:"after_seq"`
	PageLimit int32     `json:"page_limit"`
}

type HubDevice struct {
	ID                uuid.UUID  `json:"id"`
	ShipID            uuid.UUID  `json:"ship_id"`
	Name              string     `json:"name"`
	Role              string     `json:"role"`
	Epoch             int32      `json:"epoch"`
	CertSerial        *string    `json:"cert_serial"`
	CertExpiresAt     *time.Time `json:"cert_expires_at"`
	Status            string     `json:"status"`
	SoftwareVersion   *string    `json:"software_version"`
	LastSeenAt        *time.Time `json:"last_seen_at"`
	LastSeqApplied    int64      `json:"last_seq_applied"`
	LastAckedLocalSeq int64      `json:"last_acked_local_seq"`
	SyncLagSeconds    *int32     `json:"sync_lag_seconds"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Version           int64      `json:"version"`
	DeletedAt         *time.Time `json:"deleted_at"`
}

type ShipDevice struct {
	ID           uuid.UUID  `json:"id"`
	ShipID       uuid.UUID  `json:"ship_id"`
	DeviceName   string     `json:"device_name"`
	DeviceType   string     `json:"device_type"`
	TokenHash    []byte     `json:"token_hash"`
	Status       string     `json:"status"`
	RegisteredAt time.Time  `json:"registered_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Version      int64      `json:"version"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

type Revocation struct {
	ID          uuid.UUID  `json:"id"`
	ShipID      uuid.UUID  `json:"ship_id"`
	SubjectType string     `json:"subject_type"`
	SubjectID   string     `json:"subject_id"`
	Reason      *string    `json:"reason"`
	RevokedAt   time.Time  `json:"revoked_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Version     int64      `json:"version"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type Snapshot struct {
	ID            uuid.UUID     `json:"id"`
	ShipID        uuid.UUID     `json:"ship_id"`
	VoyageID      uuid.NullUUID `json:"voyage_id"`
	Phase         string        `json:"phase"`
	SeqBase       int64         `json:"seq_base"`
	SchemaVersion int32         `json:"schema_version"`
	StorageKey    string        `json:"storage_key"`
	Sha256        string        `json:"sha256"`
	SizeBytes     *int64        `json:"size_bytes"`
	Kid           string        `json:"kid"`
	Signature     string        `json:"signature"`
	Status        string        `json:"status"`
	CreatedAt     time.Time     `json:"created_at"`
}

type JobRun struct {
	ID              uuid.UUID  `json:"id"`
	VoyageID        uuid.UUID  `json:"voyage_id"`
	JobType         string     `json:"job_type"`
	ScheduleVersion int32      `json:"schedule_version"`
	Status          string     `json:"status"`
	Attempt         int32      `json:"attempt"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
	Error           *string    `json:"error"`
}

type OpsEvent struct {
	HubID      uuid.UUID       `json:"hub_id"`
	LocalSeq   int64           `json:"local_seq"`
	ShipID     uuid.UUID       `json:"ship_id"`
	VoyageID   uuid.NullUUID   `json:"voyage_id"`
	Epoch      int32           `json:"epoch"`
	Entity     string          `json:"entity"`
	EntityID   string          `json:"entity_id"`
	Op         string          `json:"op"`
	Payload    json.RawMessage `json:"payload"`
	OccurredAt time.Time       `json:"occurred_at"`
	ReceivedAt time.Time       `json:"received_at"`
}

type ShipSeq struct {
	ShipID  uuid.UUID `json:"ship_id"`
	LastSeq int64     `json:"last_seq"`
}

type ChangeLog struct {
	ShipID        uuid.UUID       `json:"ship_id"`
	Seq           int64           `json:"seq"`
	Entity        string          `json:"entity"`
	EntityID      uuid.UUID       `json:"entity_id"`
	Op            string          `json:"op"`
	Version       int64           `json:"version"`
	SchemaVersion int32           `json:"schema_version"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
}
