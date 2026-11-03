package sync

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/database/sqlc"
)

type repo struct {
	q *sqlc.Queries
}

func newRepo(pool *pgxpool.Pool) *repo {
	return &repo{q: sqlc.New(pool)}
}
