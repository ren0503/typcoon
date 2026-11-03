// Package sync owns device, credential, snapshot, and job lifecycle state.
// It is an internal module: it exposes no public HTTP routes in this phase.
package sync

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

// Page is a cursor-paginated collection.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// HubDeviceView is the sanitized hub device representation (no cert serial).
type HubDeviceView struct {
	ID              uuid.UUID  `json:"id"`
	ShipID          uuid.UUID  `json:"ship_id"`
	Name            string     `json:"name"`
	Role            string     `json:"role"`
	Epoch           int32      `json:"epoch"`
	Status          string     `json:"status"`
	SoftwareVersion *string    `json:"software_version"`
	LastSeenAt      *time.Time `json:"last_seen_at"`
	LastSeqApplied  int64      `json:"last_seq_applied"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Version         int64      `json:"version"`
}

// ShipDeviceView is the sanitized iPad registry representation (no token hash).
type ShipDeviceView struct {
	ID           uuid.UUID `json:"id"`
	ShipID       uuid.UUID `json:"ship_id"`
	DeviceName   string    `json:"device_name"`
	DeviceType   string    `json:"device_type"`
	Status       string    `json:"status"`
	RegisteredAt time.Time `json:"registered_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Version      int64     `json:"version"`
}

// SnapshotView is the snapshot package metadata representation.
type SnapshotView struct {
	ID            uuid.UUID  `json:"id"`
	ShipID        uuid.UUID  `json:"ship_id"`
	VoyageID      *uuid.UUID `json:"voyage_id"`
	Phase         string     `json:"phase"`
	SeqBase       int64      `json:"seq_base"`
	SchemaVersion int32      `json:"schema_version"`
	StorageKey    string     `json:"storage_key"`
	Sha256        string     `json:"sha256"`
	SizeBytes     *int64     `json:"size_bytes"`
	Kid           string     `json:"kid"`
	Signature     string     `json:"signature"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
}

// HubDeviceInput is the hub provisioning payload.
type HubDeviceInput struct {
	ShipID     uuid.UUID
	Name       string
	Role       string
	CertSerial *string
}

// ShipDeviceInput is the iPad registration payload.
type ShipDeviceInput struct {
	ShipID     uuid.UUID
	DeviceName string
	DeviceType string
	TokenHash  []byte
}

// RevocationInput is the revocation creation payload.
type RevocationInput struct {
	ShipID      uuid.UUID
	SubjectType string
	SubjectID   string
	Reason      *string
	ExpiresAt   *time.Time
}

// SnapshotInput is the snapshot record payload.
type SnapshotInput struct {
	ShipID     uuid.UUID
	VoyageID   *uuid.UUID
	Phase      string
	SeqBase    int64
	StorageKey string
	Sha256     string
	SizeBytes  *int64
	Kid        string
	Signature  string
	Status     string
}

// JobRunInput is the job claim payload.
type JobRunInput struct {
	VoyageID        uuid.UUID
	JobType         string
	ScheduleVersion int32
}

// OpsEventInput is the hub outbox ingestion payload.
type OpsEventInput struct {
	HubID      uuid.UUID
	LocalSeq   int64
	ShipID     uuid.UUID
	VoyageID   *uuid.UUID
	Epoch      int32
	Entity     string
	EntityID   string
	Op         string
	Payload    []byte
	OccurredAt time.Time
}

// Service is the internal sync surface. All mutating operations are
// system-managed; read operations on trigger-owned tables are exposed without
// write paths.
type Service interface {
	ProvisionHubDevice(ctx context.Context, p auth.Principal, input HubDeviceInput) (HubDeviceView, error)
	GetHubDevice(ctx context.Context, p auth.Principal, id uuid.UUID) (HubDeviceView, error)
	ListHubDevices(ctx context.Context, p auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[HubDeviceView], error)
	SetHubDeviceStatus(ctx context.Context, p auth.Principal, id uuid.UUID, status string, epoch int32) (HubDeviceView, error)
	RetireHubDevice(ctx context.Context, p auth.Principal, id uuid.UUID) error

	RegisterShipDevice(ctx context.Context, p auth.Principal, input ShipDeviceInput) (ShipDeviceView, error)
	ListShipDevices(ctx context.Context, p auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[ShipDeviceView], error)
	RevokeShipDevice(ctx context.Context, p auth.Principal, id uuid.UUID) (ShipDeviceView, error)

	CreateRevocation(ctx context.Context, p auth.Principal, input RevocationInput) (sqlc.Revocation, error)
	GetRevocation(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.Revocation, error)
	ListRevocations(ctx context.Context, p auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[sqlc.Revocation], error)

	RecordSnapshot(ctx context.Context, p auth.Principal, input SnapshotInput) (SnapshotView, error)
	GetSnapshot(ctx context.Context, p auth.Principal, id uuid.UUID) (SnapshotView, error)
	ListSnapshots(ctx context.Context, p auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[SnapshotView], error)

	ClaimJobRun(ctx context.Context, p auth.Principal, input JobRunInput) (sqlc.JobRun, error)
	FinishJobRun(ctx context.Context, p auth.Principal, id uuid.UUID, status string, jobError *string) (sqlc.JobRun, error)
	ListJobRuns(ctx context.Context, p auth.Principal, voyageID uuid.UUID, cursor string, limit int32) (Page[sqlc.JobRun], error)

	AppendOpsEvent(ctx context.Context, p auth.Principal, input OpsEventInput) (bool, error)
	ListOpsEvents(ctx context.Context, p auth.Principal, shipID uuid.UUID, after time.Time, limit int32) ([]sqlc.OpsEvent, error)

	ShipSeq(ctx context.Context, p auth.Principal, shipID uuid.UUID) (sqlc.ShipSeq, error)
	ChangeLog(ctx context.Context, p auth.Principal, shipID uuid.UUID, afterSeq int64, limit int32) ([]sqlc.ChangeLog, error)
}
