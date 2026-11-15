package sync

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CreateHubDevice(context.Context, CreateHubDeviceParams) (HubDevice, error)
	GetHubDevice(context.Context, uuid.UUID) (HubDevice, error)
	ListHubDevices(context.Context, ListHubDevicesParams) ([]HubDevice, error)
	UpdateHubDeviceStatus(context.Context, UpdateHubDeviceStatusParams) (HubDevice, error)
	RetireHubDevice(context.Context, uuid.UUID) error
	CreateShipDevice(context.Context, CreateShipDeviceParams) (ShipDevice, error)
	ListShipDevices(context.Context, ListShipDevicesParams) ([]ShipDevice, error)
	RevokeShipDevice(context.Context, uuid.UUID) (ShipDevice, error)
	CreateRevocation(context.Context, CreateRevocationParams) (Revocation, error)
	GetRevocation(context.Context, uuid.UUID) (Revocation, error)
	ListRevocations(context.Context, ListRevocationsParams) ([]Revocation, error)
	CreateSnapshot(context.Context, CreateSnapshotParams) (Snapshot, error)
	GetSnapshot(context.Context, uuid.UUID) (Snapshot, error)
	ListSnapshots(context.Context, ListSnapshotsParams) ([]Snapshot, error)
	CreateJobRun(context.Context, CreateJobRunParams) (JobRun, error)
	FinishJobRun(context.Context, FinishJobRunParams) (JobRun, error)
	ListJobRuns(context.Context, ListJobRunsParams) ([]JobRun, error)
	UpsertOpsEvent(context.Context, UpsertOpsEventParams) (bool, error)
	ListOpsEvents(context.Context, ListOpsEventsParams) ([]OpsEvent, error)
	GetShipSeq(context.Context, uuid.UUID) (ShipSeq, error)
	ListChangeLog(context.Context, ListChangeLogParams) ([]ChangeLog, error)
	ListOpsEventsAfter(context.Context, uuid.UUID, time.Time, int32) ([]OpsEvent, error)
}
