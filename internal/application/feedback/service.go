package feedback

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	appship "github.com/ren0503/tycoon/internal/application/ship"
	appvoyage "github.com/ren0503/tycoon/internal/application/voyage"
	"github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

type service struct {
	repo    Repository
	ships   appship.Service
	voyages appvoyage.Service
}

// New builds the feedback service.
func New(repo Repository, ships appship.Service, voyages appvoyage.Service) FeedbackService {
	return &service{repo: repo, ships: ships, voyages: voyages}
}

func (s *service) Submit(ctx context.Context, p auth.Principal, input SubmitInput) (ResponseView, error) {
	v, err := s.voyages.Lookup(ctx, input.VoyageID)
	if err != nil {
		return ResponseView{}, err
	}
	tenantID, err := s.ships.TenantID(ctx, v.ShipID)
	if err != nil {
		return ResponseView{}, err
	}
	if !p.CanAccessTenant(tenantID) || !p.CanAccessShip(v.ShipID) {
		return ResponseView{}, apperrors.Forbidden("Feedback submission is not permitted.")
	}
	if strings.TrimSpace(input.TextRedacted) == "" {
		return ResponseView{}, validation("text_redacted is required.")
	}
	if !validRespondentType(input.RespondentType) {
		return ResponseView{}, validation("respondent_type is not recognised.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return ResponseView{}, apperrors.Internal(err)
	}
	row, err := s.repo.CreateFeedbackResponse(ctx, CreateFeedbackResponseParams{
		ID: id, ShipID: v.ShipID, VoyageID: input.VoyageID, RespondentType: input.RespondentType,
		RespondentRef: input.RespondentRef, Language: input.Language,
		TextRedacted: strings.TrimSpace(input.TextRedacted), SubmittedAt: time.Now(),
	})
	if err != nil {
		return ResponseView{}, err
	}
	return toResponseView(row), nil
}

func (s *service) Get(ctx context.Context, p auth.Principal, id uuid.UUID) (ResponseView, error) {
	row, err := s.repo.GetFeedbackResponse(ctx, id)
	if err != nil {
		return ResponseView{}, err
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		return ResponseView{}, err
	}
	return toResponseView(row), nil
}

func (s *service) ListInsights(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[InsightView], error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return Page[InsightView]{}, err
	}
	parts := []string{"feedback_insights", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[InsightView]{}, err
	}
	rows, err := s.repo.ListFeedbackInsights(ctx, ListFeedbackInsightsParams{
		ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[InsightView]{}, err
	}
	page := Page[InsightView]{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: rows[len(rows)-1].ID}.Encode()
	}
	for _, row := range rows {
		page.Items = append(page.Items, toInsightView(row))
	}
	return page, nil
}

func (s *service) RecordInsight(ctx context.Context, p auth.Principal, input InsightInput) (InsightView, error) {
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		return InsightView{}, err
	}
	if _, err := s.voyages.Lookup(ctx, input.VoyageID); err != nil {
		return InsightView{}, err
	}
	if strings.TrimSpace(input.Topic) == "" {
		return InsightView{}, validation("Insight topic is required.")
	}
	if input.Severity != nil && (*input.Severity < 1 || *input.Severity > 5) {
		return InsightView{}, validation("Insight severity must be between 1 and 5.")
	}
	if input.Sentiment != nil && (*input.Sentiment < -2 || *input.Sentiment > 2) {
		return InsightView{}, validation("Insight sentiment must be between -2 and 2.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return InsightView{}, apperrors.Internal(err)
	}
	row, err := s.repo.CreateFeedbackInsight(ctx, CreateFeedbackInsightParams{
		ID: id, ResponseID: input.ResponseID, ShipID: input.ShipID, VoyageID: input.VoyageID,
		Topic: input.Topic, AreaType: input.AreaType, DeckLevel: input.DeckLevel,
		ServiceCode: input.ServiceCode, Severity: input.Severity, Sentiment: input.Sentiment,
		ModelVersion: input.ModelVersion,
	})
	if err != nil {
		return InsightView{}, err
	}
	return toInsightView(row), nil
}

func (s *service) requireShip(ctx context.Context, p auth.Principal, shipID uuid.UUID) error {
	tenantID, err := s.ships.TenantID(ctx, shipID)
	if err != nil {
		return err
	}
	if !p.CanAccessTenant(tenantID) || !p.CanAccessShip(shipID) {
		return apperrors.Forbidden("Feedback access is not permitted.")
	}
	return nil
}

func validRespondentType(value string) bool {
	switch value {
	case "guest", "staff":
		return true
	}
	return false
}

func toResponseView(row FeedbackResponse) ResponseView {
	return ResponseView{
		ID: row.ID, ShipID: row.ShipID, VoyageID: row.VoyageID, RespondentType: row.RespondentType,
		RespondentRef: row.RespondentRef, Language: row.Language, TextRedacted: row.TextRedacted,
		SubmittedAt: row.SubmittedAt, AnalyzedAt: row.AnalyzedAt,
	}
}

func toInsightView(row FeedbackInsight) InsightView {
	return InsightView{
		ID: row.ID, ResponseID: row.ResponseID, ShipID: row.ShipID, VoyageID: row.VoyageID,
		Topic: row.Topic, AreaType: row.AreaType, DeckLevel: row.DeckLevel, ServiceCode: row.ServiceCode,
		Severity: row.Severity, Sentiment: row.Sentiment, ModelVersion: row.ModelVersion, CreatedAt: row.CreatedAt,
	}
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}

func validation(message string) error {
	return apperrors.Validation(apperrors.CodeValidationFailed, message)
}
