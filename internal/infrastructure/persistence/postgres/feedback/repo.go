// Package feedback implements feedback persistence.
package feedback

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appfeedback "github.com/ren0503/tycoon/internal/application/feedback"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Repo struct{ q *sqlc.Queries }

func New(pool *pgxpool.Pool) *Repo { return &Repo{q: sqlc.New(pool)} }

var _ appfeedback.Repository = (*Repo)(nil)

func (r *Repo) CreateFeedbackResponse(ctx context.Context, p appfeedback.CreateFeedbackResponseParams) (appfeedback.FeedbackResponse, error) {
	row, err := r.q.CreateFeedbackResponse(ctx, convert.To[sqlc.CreateFeedbackResponseParams](p))
	return convert.To[appfeedback.FeedbackResponse](row), postgres.MapPgError(err)
}
func (r *Repo) GetFeedbackResponse(ctx context.Context, id uuid.UUID) (appfeedback.FeedbackResponse, error) {
	row, err := r.q.GetFeedbackResponse(ctx, id)
	return convert.To[appfeedback.FeedbackResponse](row), postgres.MapPgError(err)
}
func (r *Repo) ListFeedbackInsights(ctx context.Context, p appfeedback.ListFeedbackInsightsParams) ([]appfeedback.FeedbackInsight, error) {
	rows, err := r.q.ListFeedbackInsights(ctx, convert.To[sqlc.ListFeedbackInsightsParams](p))
	return convert.Slice[appfeedback.FeedbackInsight](rows), postgres.MapPgError(err)
}
func (r *Repo) CreateFeedbackInsight(ctx context.Context, p appfeedback.CreateFeedbackInsightParams) (appfeedback.FeedbackInsight, error) {
	row, err := r.q.CreateFeedbackInsight(ctx, convert.To[sqlc.CreateFeedbackInsightParams](p))
	return convert.To[appfeedback.FeedbackInsight](row), postgres.MapPgError(err)
}
