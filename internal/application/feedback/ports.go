package feedback

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	CreateFeedbackResponse(context.Context, CreateFeedbackResponseParams) (FeedbackResponse, error)
	GetFeedbackResponse(context.Context, uuid.UUID) (FeedbackResponse, error)
	ListFeedbackInsights(context.Context, ListFeedbackInsightsParams) ([]FeedbackInsight, error)
	CreateFeedbackInsight(context.Context, CreateFeedbackInsightParams) (FeedbackInsight, error)
}
