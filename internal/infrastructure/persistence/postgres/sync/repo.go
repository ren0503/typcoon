// Package sync implements sync persistence.
package sync

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appsync "github.com/ren0503/tycoon/internal/application/sync"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Repo struct{ q *sqlc.Queries }

func New(pool *pgxpool.Pool) *Repo { return &Repo{q: sqlc.New(pool)} }

var _ appsync.Repository = (*Repo)(nil)

func (r *Repo) CreateHubDevice(ctx context.Context, p appsync.CreateHubDeviceParams) (appsync.HubDevice, error) {
	row, err := r.q.CreateHubDevice(ctx, convert.To[sqlc.CreateHubDeviceParams](p))
	return convert.To[appsync.HubDevice](row), postgres.MapPgError(err)
}
func (r *Repo) GetHubDevice(ctx context.Context, id uuid.UUID) (appsync.HubDevice, error) {
	row, err := r.q.GetHubDevice(ctx, id)
	return convert.To[appsync.HubDevice](row), postgres.MapPgError(err)
}
func (r *Repo) ListHubDevices(ctx context.Context, p appsync.ListHubDevicesParams) ([]appsync.HubDevice, error) {
	rows, err := r.q.ListHubDevices(ctx, convert.To[sqlc.ListHubDevicesParams](p))
	return convert.Slice[appsync.HubDevice](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateHubDeviceStatus(ctx context.Context, p appsync.UpdateHubDeviceStatusParams) (appsync.HubDevice, error) {
	row, err := r.q.UpdateHubDeviceStatus(ctx, convert.To[sqlc.UpdateHubDeviceStatusParams](p))
	return convert.To[appsync.HubDevice](row), postgres.MapPgError(err)
}
func (r *Repo) RetireHubDevice(ctx context.Context, id uuid.UUID) error {
	return deleteResult(r.q.RetireHubDevice(ctx, id))
}
func (r *Repo) CreateShipDevice(ctx context.Context, p appsync.CreateShipDeviceParams) (appsync.ShipDevice, error) {
	row, err := r.q.CreateShipDevice(ctx, convert.To[sqlc.CreateShipDeviceParams](p))
	return convert.To[appsync.ShipDevice](row), postgres.MapPgError(err)
}
func (r *Repo) ListShipDevices(ctx context.Context, p appsync.ListShipDevicesParams) ([]appsync.ShipDevice, error) {
	rows, err := r.q.ListShipDevices(ctx, convert.To[sqlc.ListShipDevicesParams](p))
	return convert.Slice[appsync.ShipDevice](rows), postgres.MapPgError(err)
}
func (r *Repo) RevokeShipDevice(ctx context.Context, id uuid.UUID) (appsync.ShipDevice, error) {
	row, err := r.q.RevokeShipDevice(ctx, id)
	return convert.To[appsync.ShipDevice](row), postgres.MapPgError(err)
}
func (r *Repo) CreateRevocation(ctx context.Context, p appsync.CreateRevocationParams) (appsync.Revocation, error) {
	row, err := r.q.CreateRevocation(ctx, convert.To[sqlc.CreateRevocationParams](p))
	return convert.To[appsync.Revocation](row), postgres.MapPgError(err)
}
func (r *Repo) GetRevocation(ctx context.Context, id uuid.UUID) (appsync.Revocation, error) {
	row, err := r.q.GetRevocation(ctx, id)
	return convert.To[appsync.Revocation](row), postgres.MapPgError(err)
}
func (r *Repo) ListRevocations(ctx context.Context, p appsync.ListRevocationsParams) ([]appsync.Revocation, error) {
	rows, err := r.q.ListRevocations(ctx, convert.To[sqlc.ListRevocationsParams](p))
	return convert.Slice[appsync.Revocation](rows), postgres.MapPgError(err)
}
func (r *Repo) CreateSnapshot(ctx context.Context, p appsync.CreateSnapshotParams) (appsync.Snapshot, error) {
	row, err := r.q.CreateSnapshot(ctx, convert.To[sqlc.CreateSnapshotParams](p))
	return convert.To[appsync.Snapshot](row), postgres.MapPgError(err)
}
func (r *Repo) GetSnapshot(ctx context.Context, id uuid.UUID) (appsync.Snapshot, error) {
	row, err := r.q.GetSnapshot(ctx, id)
	return convert.To[appsync.Snapshot](row), postgres.MapPgError(err)
}
func (r *Repo) ListSnapshots(ctx context.Context, p appsync.ListSnapshotsParams) ([]appsync.Snapshot, error) {
	rows, err := r.q.ListSnapshots(ctx, convert.To[sqlc.ListSnapshotsParams](p))
	return convert.Slice[appsync.Snapshot](rows), postgres.MapPgError(err)
}
func (r *Repo) CreateJobRun(ctx context.Context, p appsync.CreateJobRunParams) (appsync.JobRun, error) {
	row, err := r.q.CreateJobRun(ctx, convert.To[sqlc.CreateJobRunParams](p))
	return convert.To[appsync.JobRun](row), postgres.MapPgError(err)
}
func (r *Repo) FinishJobRun(ctx context.Context, p appsync.FinishJobRunParams) (appsync.JobRun, error) {
	row, err := r.q.FinishJobRun(ctx, convert.To[sqlc.FinishJobRunParams](p))
	return convert.To[appsync.JobRun](row), postgres.MapPgError(err)
}
func (r *Repo) ListJobRuns(ctx context.Context, p appsync.ListJobRunsParams) ([]appsync.JobRun, error) {
	rows, err := r.q.ListJobRuns(ctx, convert.To[sqlc.ListJobRunsParams](p))
	return convert.Slice[appsync.JobRun](rows), postgres.MapPgError(err)
}
func (r *Repo) UpsertOpsEvent(ctx context.Context, p appsync.UpsertOpsEventParams) (bool, error) {
	n, err := r.q.UpsertOpsEvent(ctx, convert.To[sqlc.UpsertOpsEventParams](p))
	return n > 0, postgres.MapPgError(err)
}
func (r *Repo) ListOpsEvents(ctx context.Context, p appsync.ListOpsEventsParams) ([]appsync.OpsEvent, error) {
	rows, err := r.q.ListOpsEvents(ctx, convert.To[sqlc.ListOpsEventsParams](p))
	return convert.Slice[appsync.OpsEvent](rows), postgres.MapPgError(err)
}
func (r *Repo) ListOpsEventsAfter(ctx context.Context, shipID uuid.UUID, after time.Time, limit int32) ([]appsync.OpsEvent, error) {
	rows, err := r.q.ListOpsEvents(ctx, sqlc.ListOpsEventsParams{ShipID: shipID, AfterReceived: after, PageLimit: limit})
	return convert.Slice[appsync.OpsEvent](rows), postgres.MapPgError(err)
}
func (r *Repo) GetShipSeq(ctx context.Context, id uuid.UUID) (appsync.ShipSeq, error) {
	row, err := r.q.GetShipSeq(ctx, id)
	return convert.To[appsync.ShipSeq](row), postgres.MapPgError(err)
}
func (r *Repo) ListChangeLog(ctx context.Context, p appsync.ListChangeLogParams) ([]appsync.ChangeLog, error) {
	rows, err := r.q.ListChangeLog(ctx, convert.To[sqlc.ListChangeLogParams](p))
	return convert.Slice[appsync.ChangeLog](rows), postgres.MapPgError(err)
}
func deleteResult(n int64, err error) error {
	if err != nil {
		return postgres.MapPgError(err)
	}
	if n == 0 {
		return postgres.MapPgError(pgx.ErrNoRows)
	}
	return nil
}
