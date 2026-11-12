package sync

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	appship "github.com/ren0503/tycoon/internal/application/ship"
	appvoyage "github.com/ren0503/tycoon/internal/application/voyage"
	"github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

var (
	hubRoles         = map[string]bool{"primary": true, "standby": true}
	hubStatuses      = map[string]bool{"provisioning": true, "active": true, "retired": true, "revoked": true}
	revocationTypes  = map[string]bool{"device": true, "staff": true, "boarding_pass": true, "partner_staff": true}
	snapshotPhases   = map[string]bool{"prepare": true, "open_boarding": true, "finalize": true, "manual": true}
	snapshotStatuses = map[string]bool{"building": true, "ready": true, "failed": true, "superseded": true}
	jobTypes         = map[string]bool{"prepare_snapshot": true, "open_boarding": true, "finalize_manifest": true, "cutoff_check": true}
	jobStatuses      = map[string]bool{"succeeded": true, "failed": true, "skipped": true}
)

type service struct {
	repo   Repository
	ships  appship.Service
	voyage appvoyage.Service
}

// New builds the internal sync service.
func New(repo Repository, ships appship.Service, voyages appvoyage.Service) Service {
	return &service{repo: repo, ships: ships, voyage: voyages}
}

func (s *service) requireShip(ctx context.Context, p auth.Principal, shipID uuid.UUID) error {
	tenantID, err := s.ships.TenantID(ctx, shipID)
	if err != nil {
		return err
	}
	if !p.CanAccessTenant(tenantID) || !p.CanAccessShip(shipID) {
		return apperrors.Forbidden("Ship is not accessible.")
	}
	return nil
}

func (s *service) ProvisionHubDevice(ctx context.Context, p auth.Principal, input HubDeviceInput) (HubDeviceView, error) {
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return HubDeviceView{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || !hubRoles[input.Role] {
		return HubDeviceView{}, validation("Hub device name and a recognised role are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return HubDeviceView{}, apperrors.Internal(err)
	}
	row, err := s.repo.CreateHubDevice(ctx, CreateHubDeviceParams{
		ID: id, ShipID: input.ShipID, Name: name, Role: input.Role, CertSerial: trimPtr(input.CertSerial),
	})
	return hubView(row), err
}

func (s *service) GetHubDevice(ctx context.Context, p auth.Principal, id uuid.UUID) (HubDeviceView, error) {
	row, err := s.repo.GetHubDevice(ctx, id)
	if err != nil {
		return HubDeviceView{}, err
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		return HubDeviceView{}, err
	}
	return hubView(row), nil
}

func (s *service) ListHubDevices(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[HubDeviceView], error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return Page[HubDeviceView]{}, err
	}
	parts := []string{"hub_devices", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[HubDeviceView]{}, err
	}
	rows, err := s.repo.ListHubDevices(ctx, ListHubDevicesParams{
		ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[HubDeviceView]{}, err
	}
	page := Page[HubDeviceView]{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: rows[len(rows)-1].ID}.Encode()
	}
	for _, row := range rows {
		page.Items = append(page.Items, hubView(row))
	}
	return page, nil
}

func (s *service) SetHubDeviceStatus(ctx context.Context, p auth.Principal, id uuid.UUID, status string, epoch int32) (HubDeviceView, error) {
	current, err := s.GetHubDevice(ctx, p, id)
	if err != nil {
		return HubDeviceView{}, err
	}
	if !hubStatuses[status] || epoch < 1 {
		return HubDeviceView{}, validation("Hub status and epoch are invalid.")
	}
	row, err := s.repo.UpdateHubDeviceStatus(ctx, UpdateHubDeviceStatusParams{
		Status: status, Epoch: epoch, ID: current.ID,
	})
	return hubView(row), err
}

func (s *service) RetireHubDevice(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetHubDevice(ctx, p, id); err != nil {
		return err
	}
	return s.repo.RetireHubDevice(ctx, id)
}

func (s *service) RegisterShipDevice(ctx context.Context, p auth.Principal, input ShipDeviceInput) (ShipDeviceView, error) {
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return ShipDeviceView{}, err
	}
	name := strings.TrimSpace(input.DeviceName)
	deviceType := input.DeviceType
	if deviceType == "" {
		deviceType = "ipad"
	}
	if name == "" || deviceType != "ipad" || len(input.TokenHash) == 0 {
		return ShipDeviceView{}, validation("Device name, ipad device type, and a token hash are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return ShipDeviceView{}, apperrors.Internal(err)
	}
	row, err := s.repo.CreateShipDevice(ctx, CreateShipDeviceParams{
		ID: id, ShipID: input.ShipID, DeviceName: name, DeviceType: deviceType, TokenHash: input.TokenHash,
	})
	return shipDeviceView(row), err
}

func (s *service) ListShipDevices(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[ShipDeviceView], error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return Page[ShipDeviceView]{}, err
	}
	parts := []string{"ship_devices", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[ShipDeviceView]{}, err
	}
	rows, err := s.repo.ListShipDevices(ctx, ListShipDevicesParams{
		ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[ShipDeviceView]{}, err
	}
	page := Page[ShipDeviceView]{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: rows[len(rows)-1].ID}.Encode()
	}
	for _, row := range rows {
		page.Items = append(page.Items, shipDeviceView(row))
	}
	return page, nil
}

func (s *service) RevokeShipDevice(ctx context.Context, p auth.Principal, id uuid.UUID) (ShipDeviceView, error) {
	row, err := s.repo.RevokeShipDevice(ctx, id)
	if err != nil {
		return ShipDeviceView{}, err
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		return ShipDeviceView{}, err
	}
	return shipDeviceView(row), nil
}

func (s *service) CreateRevocation(ctx context.Context, p auth.Principal, input RevocationInput) (Revocation, error) {
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return Revocation{}, err
	}
	if !revocationTypes[input.SubjectType] || strings.TrimSpace(input.SubjectID) == "" {
		return Revocation{}, validation("Revocation subject type and subject id are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Revocation{}, apperrors.Internal(err)
	}
	row, err := s.repo.CreateRevocation(ctx, CreateRevocationParams{
		ID: id, ShipID: input.ShipID, SubjectType: input.SubjectType,
		SubjectID: strings.TrimSpace(input.SubjectID), Reason: trimPtr(input.Reason), ExpiresAt: input.ExpiresAt,
	})
	return row, err
}

func (s *service) GetRevocation(ctx context.Context, p auth.Principal, id uuid.UUID) (Revocation, error) {
	row, err := s.repo.GetRevocation(ctx, id)
	if err != nil {
		return Revocation{}, err
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		return Revocation{}, err
	}
	return row, nil
}

func (s *service) ListRevocations(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[Revocation], error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return Page[Revocation]{}, err
	}
	parts := []string{"revocations", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[Revocation]{}, err
	}
	rows, err := s.repo.ListRevocations(ctx, ListRevocationsParams{
		ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[Revocation]{}, err
	}
	page := Page[Revocation]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) RecordSnapshot(ctx context.Context, p auth.Principal, input SnapshotInput) (SnapshotView, error) {
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return SnapshotView{}, err
	}
	if !snapshotPhases[input.Phase] {
		return SnapshotView{}, validation("Snapshot phase is not recognised.")
	}
	status := input.Status
	if status == "" {
		status = "building"
	}
	if !snapshotStatuses[status] {
		return SnapshotView{}, validation("Snapshot status is not recognised.")
	}
	if strings.TrimSpace(input.StorageKey) == "" || strings.TrimSpace(input.Sha256) == "" ||
		strings.TrimSpace(input.Kid) == "" || strings.TrimSpace(input.Signature) == "" {
		return SnapshotView{}, validation("Snapshot storage key, sha256, kid, and signature are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return SnapshotView{}, apperrors.Internal(err)
	}
	row, err := s.repo.CreateSnapshot(ctx, CreateSnapshotParams{
		ID: id, ShipID: input.ShipID, VoyageID: optionalUUID(input.VoyageID), Phase: input.Phase,
		SeqBase: input.SeqBase, StorageKey: strings.TrimSpace(input.StorageKey), Sha256: strings.TrimSpace(input.Sha256),
		SizeBytes: input.SizeBytes, Kid: strings.TrimSpace(input.Kid), Signature: strings.TrimSpace(input.Signature), Status: status,
	})
	return snapshotView(row), err
}

func (s *service) GetSnapshot(ctx context.Context, p auth.Principal, id uuid.UUID) (SnapshotView, error) {
	row, err := s.repo.GetSnapshot(ctx, id)
	if err != nil {
		return SnapshotView{}, err
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		return SnapshotView{}, err
	}
	return snapshotView(row), nil
}

func (s *service) ListSnapshots(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[SnapshotView], error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return Page[SnapshotView]{}, err
	}
	parts := []string{"snapshots", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[SnapshotView]{}, err
	}
	rows, err := s.repo.ListSnapshots(ctx, ListSnapshotsParams{
		ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[SnapshotView]{}, err
	}
	page := Page[SnapshotView]{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: rows[len(rows)-1].ID}.Encode()
	}
	for _, row := range rows {
		page.Items = append(page.Items, snapshotView(row))
	}
	return page, nil
}

func (s *service) ClaimJobRun(ctx context.Context, p auth.Principal, input JobRunInput) (JobRun, error) {
	v, err := s.voyage.Lookup(ctx, input.VoyageID)
	if err != nil {
		return JobRun{}, err
	}
	if err := s.requireShip(ctx, p, v.ShipID); err != nil {
		return JobRun{}, err
	}
	if !jobTypes[input.JobType] || input.ScheduleVersion < 1 {
		return JobRun{}, validation("Job type and a positive schedule version are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return JobRun{}, apperrors.Internal(err)
	}
	row, err := s.repo.CreateJobRun(ctx, CreateJobRunParams{
		ID: id, VoyageID: input.VoyageID, JobType: input.JobType, ScheduleVersion: input.ScheduleVersion,
	})
	return row, err
}

func (s *service) FinishJobRun(ctx context.Context, p auth.Principal, id uuid.UUID, status string, jobError *string) (JobRun, error) {
	if !jobStatuses[status] {
		return JobRun{}, validation("Job status is not recognised.")
	}
	row, err := s.repo.FinishJobRun(ctx, FinishJobRunParams{Status: status, Error: trimPtr(jobError), ID: id})
	if err != nil {
		return JobRun{}, err
	}
	v, err := s.voyage.Lookup(ctx, row.VoyageID)
	if err != nil {
		return JobRun{}, err
	}
	if err := s.requireShip(ctx, p, v.ShipID); err != nil {
		return JobRun{}, err
	}
	return row, nil
}

func (s *service) ListJobRuns(ctx context.Context, p auth.Principal, voyageID uuid.UUID, rawCursor string, limit int32) (Page[JobRun], error) {
	v, err := s.voyage.Lookup(ctx, voyageID)
	if err != nil {
		return Page[JobRun]{}, err
	}
	if err := s.requireShip(ctx, p, v.ShipID); err != nil {
		return Page[JobRun]{}, err
	}
	parts := []string{"job_runs", voyageID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[JobRun]{}, err
	}
	rows, err := s.repo.ListJobRuns(ctx, ListJobRunsParams{
		VoyageID: voyageID, After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[JobRun]{}, err
	}
	page := Page[JobRun]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) AppendOpsEvent(ctx context.Context, p auth.Principal, input OpsEventInput) (bool, error) {
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return false, err
	}
	if input.LocalSeq < 1 || input.Epoch < 1 || strings.TrimSpace(input.Entity) == "" ||
		strings.TrimSpace(input.EntityID) == "" || (input.Op != "upsert" && input.Op != "delete") {
		return false, validation("Ops event sequence, epoch, entity, entity id, and op are required.")
	}
	payload := input.Payload
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	if !json.Valid(payload) {
		return false, validation("Ops event payload must be valid JSON.")
	}
	ok, err := s.repo.UpsertOpsEvent(ctx, UpsertOpsEventParams{
		HubID: input.HubID, LocalSeq: input.LocalSeq, ShipID: input.ShipID,
		VoyageID: optionalUUID(input.VoyageID), Epoch: input.Epoch, Entity: strings.TrimSpace(input.Entity),
		EntityID: strings.TrimSpace(input.EntityID), Op: input.Op, Payload: payload, OccurredAt: input.OccurredAt,
	})
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (s *service) ListOpsEvents(ctx context.Context, p auth.Principal, shipID uuid.UUID, after time.Time, limit int32) ([]OpsEvent, error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > pagination.MaxLimit {
		limit = pagination.MaxLimit
	}
	rows, err := s.repo.ListOpsEventsAfter(ctx, shipID, after, limit)
	return rows, err
}

func (s *service) ShipSeq(ctx context.Context, p auth.Principal, shipID uuid.UUID) (ShipSeq, error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return ShipSeq{}, err
	}
	row, err := s.repo.GetShipSeq(ctx, shipID)
	return row, err
}

func (s *service) ChangeLog(ctx context.Context, p auth.Principal, shipID uuid.UUID, afterSeq int64, limit int32) ([]ChangeLog, error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > pagination.MaxLimit {
		limit = pagination.MaxLimit
	}
	rows, err := s.repo.ListChangeLog(ctx, ListChangeLogParams{ShipID: shipID, AfterSeq: afterSeq, PageLimit: limit})
	return rows, err
}

func hubView(row HubDevice) HubDeviceView {
	return HubDeviceView{
		ID: row.ID, ShipID: row.ShipID, Name: row.Name, Role: row.Role, Epoch: row.Epoch,
		Status: row.Status, SoftwareVersion: row.SoftwareVersion, LastSeenAt: row.LastSeenAt,
		LastSeqApplied: row.LastSeqApplied, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version,
	}
}

func shipDeviceView(row ShipDevice) ShipDeviceView {
	return ShipDeviceView{
		ID: row.ID, ShipID: row.ShipID, DeviceName: row.DeviceName, DeviceType: row.DeviceType,
		Status: row.Status, RegisteredAt: row.RegisteredAt, CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt, Version: row.Version,
	}
}

func snapshotView(row Snapshot) SnapshotView {
	view := SnapshotView{
		ID: row.ID, ShipID: row.ShipID, Phase: row.Phase, SeqBase: row.SeqBase,
		SchemaVersion: row.SchemaVersion, StorageKey: row.StorageKey, Sha256: row.Sha256,
		SizeBytes: row.SizeBytes, Kid: row.Kid, Signature: row.Signature, Status: row.Status, CreatedAt: row.CreatedAt,
	}
	if row.VoyageID.Valid {
		voyageID := row.VoyageID.UUID
		view.VoyageID = &voyageID
	}
	return view
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}

func optionalUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil || *id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func trimPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func validation(message string) error {
	return apperrors.Validation(apperrors.CodeValidationFailed, message)
}
