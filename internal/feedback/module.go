// Package feedback owns redacted feedback responses and record-only insights.
package feedback

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
)

// Page is a cursor-paginated collection.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// SubmitInput records a redacted feedback response.
type SubmitInput struct {
	VoyageID       uuid.UUID
	RespondentType string
	RespondentRef  *string
	Language       *string
	TextRedacted   string
}

// InsightInput records a computed feedback insight. Insights are immutable.
type InsightInput struct {
	ResponseID   uuid.UUID
	ShipID       uuid.UUID
	VoyageID     uuid.UUID
	Topic        string
	AreaType     *string
	DeckLevel    *int32
	ServiceCode  *string
	Severity     *int16
	Sentiment    *int16
	ModelVersion string
}

// FeedbackService submits and reads feedback; it never updates or deletes.
type FeedbackService interface {
	Submit(ctx context.Context, principal auth.Principal, input SubmitInput) (ResponseView, error)
	Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (ResponseView, error)
	ListInsights(ctx context.Context, principal auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[InsightView], error)
	RecordInsight(ctx context.Context, principal auth.Principal, input InsightInput) (InsightView, error)
}

// ResponseView is the feedback response representation.
type ResponseView struct {
	ID             uuid.UUID  `json:"id"`
	ShipID         uuid.UUID  `json:"ship_id"`
	VoyageID       uuid.UUID  `json:"voyage_id"`
	RespondentType string     `json:"respondent_type"`
	RespondentRef  *string    `json:"respondent_ref"`
	Language       *string    `json:"language"`
	TextRedacted   string     `json:"text_redacted"`
	SubmittedAt    time.Time  `json:"submitted_at"`
	AnalyzedAt     *time.Time `json:"analyzed_at"`
}

// InsightView is the feedback insight representation.
type InsightView struct {
	ID           uuid.UUID `json:"id"`
	ResponseID   uuid.UUID `json:"response_id"`
	ShipID       uuid.UUID `json:"ship_id"`
	VoyageID     uuid.UUID `json:"voyage_id"`
	Topic        string    `json:"topic"`
	AreaType     *string   `json:"area_type"`
	DeckLevel    *int32    `json:"deck_level"`
	ServiceCode  *string   `json:"service_code"`
	Severity     *int16    `json:"severity"`
	Sentiment    *int16    `json:"sentiment"`
	ModelVersion string    `json:"model_version"`
	CreatedAt    time.Time `json:"created_at"`
}
