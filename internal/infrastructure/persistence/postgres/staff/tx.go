package staff

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ren0503/tycoon/internal/application/audit"
	appstaff "github.com/ren0503/tycoon/internal/application/staff"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Tx struct {
	tx pgx.Tx
	q  *sqlc.Queries
}

var _ appstaff.Tx = (*Tx)(nil)

func (t *Tx) CreateStaff(ctx context.Context, p appstaff.CreateStaffParams) (appstaff.Staff, error) {
	row, err := t.q.CreateStaff(ctx, convert.To[sqlc.CreateStaffParams](p))
	return convert.To[appstaff.Staff](row), postgres.MapPgError(err)
}
func (t *Tx) UpdateStaff(ctx context.Context, p appstaff.UpdateStaffParams) (appstaff.Staff, error) {
	row, err := t.q.UpdateStaff(ctx, convert.To[sqlc.UpdateStaffParams](p))
	return convert.To[appstaff.Staff](row), postgres.MapPgError(err)
}
func (t *Tx) SoftDeleteStaff(ctx context.Context, id uuid.UUID) error {
	n, err := t.q.SoftDeleteStaff(ctx, id)
	if err != nil {
		return postgres.MapPgError(err)
	}
	if n == 0 {
		return postgres.MapPgError(pgx.ErrNoRows)
	}
	return nil
}
func (t *Tx) AppendAudit(ctx context.Context, e audit.Entry) error {
	return postgres.AppendAuditTx(ctx, t.tx, e)
}
func (t *Tx) Commit(ctx context.Context) error   { return postgres.MapPgError(t.tx.Commit(ctx)) }
func (t *Tx) Rollback(ctx context.Context) error { return postgres.MapPgError(t.tx.Rollback(ctx)) }
