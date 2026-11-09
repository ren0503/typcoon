package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

type AccountService interface {
	Exists(ctx context.Context, accountID uuid.UUID) (bool, error)
}

type accountService struct {
	q *sqlc.Queries
}

func NewAccountService(pool *pgxpool.Pool) AccountService {
	return &accountService{q: sqlc.New(pool)}
}

func (s *accountService) Exists(ctx context.Context, accountID uuid.UUID) (bool, error) {
	ok, err := s.q.AccountExists(ctx, accountID)
	return ok, database.MapPgError(err)
}
