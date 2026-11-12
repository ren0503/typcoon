package guest

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ren0503/tycoon/internal/application/audit"
	appguest "github.com/ren0503/tycoon/internal/application/guest"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Tx struct {
	tx pgx.Tx
	q  *sqlc.Queries
}

var _ appguest.Tx = (*Tx)(nil)

func (t *Tx) CreateGuest(ctx context.Context, p appguest.CreateGuestParams) (appguest.Guest, error) {
	row, err := t.q.CreateGuest(ctx, convert.To[sqlc.CreateGuestParams](p))
	return convert.To[appguest.Guest](row), postgres.MapPgError(err)
}
func (t *Tx) UpdateGuest(ctx context.Context, p appguest.UpdateGuestParams) (appguest.Guest, error) {
	row, err := t.q.UpdateGuest(ctx, convert.To[sqlc.UpdateGuestParams](p))
	return convert.To[appguest.Guest](row), postgres.MapPgError(err)
}
func (t *Tx) SoftDeleteGuest(ctx context.Context, id uuid.UUID) error {
	return deleteResult(t.q.SoftDeleteGuest(ctx, id))
}
func (t *Tx) CreateBoardingPass(ctx context.Context, p appguest.CreateBoardingPassParams) (appguest.BoardingPass, error) {
	row, err := t.q.CreateBoardingPass(ctx, convert.To[sqlc.CreateBoardingPassParams](p))
	return convert.To[appguest.BoardingPass](row), postgres.MapPgError(err)
}
func (t *Tx) RevokeBoardingPass(ctx context.Context, id uuid.UUID) (appguest.BoardingPass, error) {
	row, err := t.q.RevokeBoardingPass(ctx, id)
	return convert.To[appguest.BoardingPass](row), postgres.MapPgError(err)
}
func (t *Tx) AppendAudit(ctx context.Context, e audit.Entry) error {
	return postgres.AppendAuditTx(ctx, t.tx, e)
}
func (t *Tx) Commit(ctx context.Context) error   { return postgres.MapPgError(t.tx.Commit(ctx)) }
func (t *Tx) Rollback(ctx context.Context) error { return postgres.MapPgError(t.tx.Rollback(ctx)) }
func deleteResult(n int64, err error) error {
	if err != nil {
		return postgres.MapPgError(err)
	}
	if n == 0 {
		return postgres.MapPgError(pgx.ErrNoRows)
	}
	return nil
}
