package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
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
	return ok, postgres.MapPgError(err)
}
