package partner

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

type repo struct {
	q *sqlc.Queries
}

func newRepo(pool *pgxpool.Pool) *repo {
	return &repo{q: sqlc.New(pool)}
}

func (r *repo) createPartner(ctx context.Context, p sqlc.CreatePartnerParams) (sqlc.Partner, error) {
	row, err := r.q.CreatePartner(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getPartner(ctx context.Context, id uuid.UUID) (sqlc.Partner, error) {
	row, err := r.q.GetPartner(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listPartners(ctx context.Context, p sqlc.ListPartnersParams) ([]sqlc.Partner, error) {
	rows, err := r.q.ListPartners(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updatePartner(ctx context.Context, p sqlc.UpdatePartnerParams) (sqlc.Partner, error) {
	row, err := r.q.UpdatePartner(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deletePartner(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeletePartner(ctx, id)
	return execResult(n, err)
}

func (r *repo) createContract(ctx context.Context, p sqlc.CreatePartnerContractParams) (sqlc.PartnerContract, error) {
	row, err := r.q.CreatePartnerContract(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getContract(ctx context.Context, id uuid.UUID) (sqlc.PartnerContract, error) {
	row, err := r.q.GetPartnerContract(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listContracts(ctx context.Context, p sqlc.ListPartnerContractsParams) ([]sqlc.PartnerContract, error) {
	rows, err := r.q.ListPartnerContracts(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateContract(ctx context.Context, p sqlc.UpdatePartnerContractParams) (sqlc.PartnerContract, error) {
	row, err := r.q.UpdatePartnerContract(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteContract(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeletePartnerContract(ctx, id)
	return execResult(n, err)
}

func (r *repo) createStaff(ctx context.Context, p sqlc.CreatePartnerStaffParams) (sqlc.PartnerStaff, error) {
	row, err := r.q.CreatePartnerStaff(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getStaff(ctx context.Context, id uuid.UUID) (sqlc.PartnerStaff, error) {
	row, err := r.q.GetPartnerStaff(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listStaff(ctx context.Context, p sqlc.ListPartnerStaffParams) ([]sqlc.PartnerStaff, error) {
	rows, err := r.q.ListPartnerStaff(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateStaff(ctx context.Context, p sqlc.UpdatePartnerStaffParams) (sqlc.PartnerStaff, error) {
	row, err := r.q.UpdatePartnerStaff(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteStaff(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeletePartnerStaff(ctx, id)
	return execResult(n, err)
}

func (r *repo) createService(ctx context.Context, p sqlc.CreatePartnerServiceParams) (sqlc.PartnerService, error) {
	row, err := r.q.CreatePartnerService(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getService(ctx context.Context, id uuid.UUID) (sqlc.PartnerService, error) {
	row, err := r.q.GetPartnerService(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listServices(ctx context.Context, p sqlc.ListPartnerServicesParams) ([]sqlc.PartnerService, error) {
	rows, err := r.q.ListPartnerServices(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateService(ctx context.Context, p sqlc.UpdatePartnerServiceParams) (sqlc.PartnerService, error) {
	row, err := r.q.UpdatePartnerService(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteService(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeletePartnerService(ctx, id)
	return execResult(n, err)
}

func (r *repo) createSlot(ctx context.Context, p sqlc.CreatePartnerSlotParams) (sqlc.PartnerSlot, error) {
	row, err := r.q.CreatePartnerSlot(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getSlot(ctx context.Context, id uuid.UUID) (sqlc.PartnerSlot, error) {
	row, err := r.q.GetPartnerSlot(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listSlots(ctx context.Context, p sqlc.ListPartnerSlotsParams) ([]sqlc.PartnerSlot, error) {
	rows, err := r.q.ListPartnerSlots(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateSlot(ctx context.Context, p sqlc.UpdatePartnerSlotParams) (sqlc.PartnerSlot, error) {
	row, err := r.q.UpdatePartnerSlot(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteSlot(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeletePartnerSlot(ctx, id)
	return execResult(n, err)
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}
