// Package partner implements partner persistence.
package partner

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apppartner "github.com/ren0503/tycoon/internal/application/partner"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Repo struct{ q *sqlc.Queries }

func New(pool *pgxpool.Pool) *Repo { return &Repo{q: sqlc.New(pool)} }

var _ apppartner.Repository = (*Repo)(nil)

func (r *Repo) CreatePartner(ctx context.Context, p apppartner.CreatePartnerParams) (apppartner.Partner, error) {
	row, err := r.q.CreatePartner(ctx, convert.To[sqlc.CreatePartnerParams](p))
	return convert.To[apppartner.Partner](row), postgres.MapPgError(err)
}
func (r *Repo) GetPartner(ctx context.Context, id uuid.UUID) (apppartner.Partner, error) {
	row, err := r.q.GetPartner(ctx, id)
	return convert.To[apppartner.Partner](row), postgres.MapPgError(err)
}
func (r *Repo) ListPartners(ctx context.Context, p apppartner.ListPartnersParams) ([]apppartner.Partner, error) {
	rows, err := r.q.ListPartners(ctx, convert.To[sqlc.ListPartnersParams](p))
	return convert.Slice[apppartner.Partner](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdatePartner(ctx context.Context, p apppartner.UpdatePartnerParams) (apppartner.Partner, error) {
	row, err := r.q.UpdatePartner(ctx, convert.To[sqlc.UpdatePartnerParams](p))
	return convert.To[apppartner.Partner](row), postgres.MapPgError(err)
}
func (r *Repo) DeletePartner(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeletePartner(ctx, id))
}
func (r *Repo) PartnerExists(ctx context.Context, p apppartner.PartnerExistsParams) (bool, error) {
	ok, err := r.q.PartnerExists(ctx, convert.To[sqlc.PartnerExistsParams](p))
	return ok, postgres.MapPgError(err)
}
func (r *Repo) PartnerServiceExists(ctx context.Context, id uuid.UUID) (bool, error) {
	ok, err := r.q.PartnerServiceExists(ctx, id)
	return ok, postgres.MapPgError(err)
}
func (r *Repo) CreateContract(ctx context.Context, p apppartner.CreatePartnerContractParams) (apppartner.PartnerContract, error) {
	row, err := r.q.CreatePartnerContract(ctx, convert.To[sqlc.CreatePartnerContractParams](p))
	return convert.To[apppartner.PartnerContract](row), postgres.MapPgError(err)
}
func (r *Repo) GetContract(ctx context.Context, id uuid.UUID) (apppartner.PartnerContract, error) {
	row, err := r.q.GetPartnerContract(ctx, id)
	return convert.To[apppartner.PartnerContract](row), postgres.MapPgError(err)
}
func (r *Repo) ListContracts(ctx context.Context, p apppartner.ListPartnerContractsParams) ([]apppartner.PartnerContract, error) {
	rows, err := r.q.ListPartnerContracts(ctx, convert.To[sqlc.ListPartnerContractsParams](p))
	return convert.Slice[apppartner.PartnerContract](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateContract(ctx context.Context, p apppartner.UpdatePartnerContractParams) (apppartner.PartnerContract, error) {
	row, err := r.q.UpdatePartnerContract(ctx, convert.To[sqlc.UpdatePartnerContractParams](p))
	return convert.To[apppartner.PartnerContract](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteContract(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeletePartnerContract(ctx, id))
}
func (r *Repo) CreateStaff(ctx context.Context, p apppartner.CreatePartnerStaffParams) (apppartner.PartnerStaff, error) {
	row, err := r.q.CreatePartnerStaff(ctx, convert.To[sqlc.CreatePartnerStaffParams](p))
	return convert.To[apppartner.PartnerStaff](row), postgres.MapPgError(err)
}
func (r *Repo) GetStaff(ctx context.Context, id uuid.UUID) (apppartner.PartnerStaff, error) {
	row, err := r.q.GetPartnerStaff(ctx, id)
	return convert.To[apppartner.PartnerStaff](row), postgres.MapPgError(err)
}
func (r *Repo) ListStaff(ctx context.Context, p apppartner.ListPartnerStaffParams) ([]apppartner.PartnerStaff, error) {
	rows, err := r.q.ListPartnerStaff(ctx, convert.To[sqlc.ListPartnerStaffParams](p))
	return convert.Slice[apppartner.PartnerStaff](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateStaff(ctx context.Context, p apppartner.UpdatePartnerStaffParams) (apppartner.PartnerStaff, error) {
	row, err := r.q.UpdatePartnerStaff(ctx, convert.To[sqlc.UpdatePartnerStaffParams](p))
	return convert.To[apppartner.PartnerStaff](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteStaff(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeletePartnerStaff(ctx, id))
}
func (r *Repo) CreateService(ctx context.Context, p apppartner.CreatePartnerServiceParams) (apppartner.PartnerService, error) {
	row, err := r.q.CreatePartnerService(ctx, convert.To[sqlc.CreatePartnerServiceParams](p))
	return convert.To[apppartner.PartnerService](row), postgres.MapPgError(err)
}
func (r *Repo) GetService(ctx context.Context, id uuid.UUID) (apppartner.PartnerService, error) {
	row, err := r.q.GetPartnerService(ctx, id)
	return convert.To[apppartner.PartnerService](row), postgres.MapPgError(err)
}
func (r *Repo) ListServices(ctx context.Context, p apppartner.ListPartnerServicesParams) ([]apppartner.PartnerService, error) {
	rows, err := r.q.ListPartnerServices(ctx, convert.To[sqlc.ListPartnerServicesParams](p))
	return convert.Slice[apppartner.PartnerService](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateService(ctx context.Context, p apppartner.UpdatePartnerServiceParams) (apppartner.PartnerService, error) {
	row, err := r.q.UpdatePartnerService(ctx, convert.To[sqlc.UpdatePartnerServiceParams](p))
	return convert.To[apppartner.PartnerService](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteService(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeletePartnerService(ctx, id))
}
func (r *Repo) CreateSlot(ctx context.Context, p apppartner.CreatePartnerSlotParams) (apppartner.PartnerSlot, error) {
	row, err := r.q.CreatePartnerSlot(ctx, convert.To[sqlc.CreatePartnerSlotParams](p))
	return convert.To[apppartner.PartnerSlot](row), postgres.MapPgError(err)
}
func (r *Repo) GetSlot(ctx context.Context, id uuid.UUID) (apppartner.PartnerSlot, error) {
	row, err := r.q.GetPartnerSlot(ctx, id)
	return convert.To[apppartner.PartnerSlot](row), postgres.MapPgError(err)
}
func (r *Repo) ListSlots(ctx context.Context, p apppartner.ListPartnerSlotsParams) ([]apppartner.PartnerSlot, error) {
	rows, err := r.q.ListPartnerSlots(ctx, convert.To[sqlc.ListPartnerSlotsParams](p))
	return convert.Slice[apppartner.PartnerSlot](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateSlot(ctx context.Context, p apppartner.UpdatePartnerSlotParams) (apppartner.PartnerSlot, error) {
	row, err := r.q.UpdatePartnerSlot(ctx, convert.To[sqlc.UpdatePartnerSlotParams](p))
	return convert.To[apppartner.PartnerSlot](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteSlot(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeletePartnerSlot(ctx, id))
}
func mapDelete(_ int64, err error) error {
	if err != nil {
		return postgres.MapPgError(err)
	}
	return nil
}
