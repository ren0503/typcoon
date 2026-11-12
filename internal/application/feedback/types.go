package feedback

import (
	"time"

	"github.com/google/uuid"
)

type FeedbackResponse struct {
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

type FeedbackInsight struct {
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

type CreateFeedbackResponseParams struct {
	ID             uuid.UUID `json:"id"`
	ShipID         uuid.UUID `json:"ship_id"`
	VoyageID       uuid.UUID `json:"voyage_id"`
	RespondentType string    `json:"respondent_type"`
	RespondentRef  *string   `json:"respondent_ref"`
	Language       *string   `json:"language"`
	TextRedacted   string    `json:"text_redacted"`
	SubmittedAt    time.Time `json:"submitted_at"`
}

type ListFeedbackInsightsParams struct {
	ShipID    uuid.UUID     `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateFeedbackInsightParams struct {
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
}
