package ai

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	apperrors "github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/platform/helpers"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

// ParamView is the public AI parameter metadata representation.
type ParamView struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Kind         string     `json:"kind"`
	ParamVersion int32      `json:"param_version"`
	StorageKey   string     `json:"storage_key"`
	Sha256       string     `json:"sha256"`
	Kid          string     `json:"kid"`
	Signature    string     `json:"signature"`
	Status       string     `json:"status"`
	ReleasedAt   *time.Time `json:"released_at"`
	Notes        *string    `json:"notes"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Version      int64      `json:"version"`
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

// New builds the AI parameter metadata service.
func New(repo Repository, logger *slog.Logger) AIParamService {
	return &service{repo: repo, logger: logger}
}

func (s *service) Create(ctx context.Context, p auth.Principal, input ParamInput) (view ParamView, err error) {
	defer func() { s.logOperation(ctx, "create", err) }()

	err = s.validateBeforeCreate(p, input)
	if err != nil {
		return ParamView{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return ParamView{}, apperrors.Internal(err)
	}

	row, err := s.repo.CreateAIParam(ctx, input.toCreateAPIParam(id))
	if err != nil {
		return ParamView{}, err
	}

	return row.toView(), nil
}

func (s *service) Get(ctx context.Context, p auth.Principal, id uuid.UUID) (view ParamView, err error) {
	defer func() { s.logOperation(ctx, "get", err) }()

	row, err := s.repo.GetAIParam(ctx, id)
	if err != nil {
		return ParamView{}, err
	}

	if !p.CanAccessTenant(row.TenantID) {
		return ParamView{}, apperrors.NotFound(apperrors.CodeNotFound, "Resource not found.")
	}

	return row.toView(), nil
}

func (s *service) List(ctx context.Context, p auth.Principal, tenantID uuid.UUID, kind *string, rawCursor string, limit int32) (data pagination.Page[ParamView], err error) {
	defer func() { s.logOperation(ctx, "list", err) }()

	if !p.CanAccessTenant(tenantID) {
		return pagination.Page[ParamView]{}, apperrors.Forbidden("AI parameter access is not permitted.")
	}
	parts := []string{"ai_params", tenantID.String()}
	if kind != nil {
		parts = append(parts, *kind)
	}

	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return pagination.Page[ParamView]{}, err
	}

	var kindParam *string
	if kind != nil {
		kindParam = kind
	}

	rows, err := s.repo.ListAIParams(ctx, ListAIParamsParams{
		TenantID: tenantID, Kind: kindParam, After: helpers.NullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return pagination.Page[ParamView]{}, err
	}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		data.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: rows[len(rows)-1].ID}.Encode()
	}
	for _, row := range rows {
		data.Items = append(data.Items, row.toView())
	}
	return data, nil
}

func (s *service) Update(ctx context.Context, p auth.Principal, id uuid.UUID, input ParamUpdate) (view ParamView, err error) {
	defer func() { s.logOperation(ctx, "update", err) }()

	current, err := s.Get(ctx, p, id)
	if err != nil {
		return ParamView{}, err
	}
	status := input.Status
	if status == "" {
		status = current.Status
	}
	if !validStatus(status) {
		return ParamView{}, validation("AI parameter status is not recognised.")
	}
	row, err := s.repo.UpdateAIParamMetadata(ctx, input.toUpdateAIParamMetadataParams(id))
	if err != nil {
		return ParamView{}, err
	}
	return row.toView(), nil
}

func (s *service) Delete(ctx context.Context, p auth.Principal, id uuid.UUID) (err error) {
	defer func() { s.logOperation(ctx, "delete", err) }()

	if _, err := s.Get(ctx, p, id); err != nil {
		return err
	}
	return s.repo.SoftDeleteAIParam(ctx, id)
}

func (s *service) validateBeforeCreate(p auth.Principal, input ParamInput) error {
	if !p.CanAccessTenant(input.TenantID) {
		return apperrors.Forbidden("AI parameter access is not permitted.")
	}
	if strings.TrimSpace(input.Kind) == "" || input.ParamVersion < 1 {
		return validation("kind and param_version are required.")
	}
	if input.Status == "" {
		input.Status = "draft"
	}
	if !validStatus(input.Status) {
		return validation("AI parameter status is not recognised.")
	}
	return nil
}

func (s *service) logOperation(ctx context.Context, operation string, err error) {
	outcome := "success"
	if err != nil {
		outcome = "failure"
	}
	attrs := []any{"operation", operation, "outcome", outcome}
	if err != nil {
		attrs = append(attrs, "error", err)
		s.logger.ErrorContext(ctx, "AI parameter operation", attrs...)
		return
	}
	s.logger.InfoContext(ctx, "AI parameter operation", attrs...)
}

func validStatus(status string) bool {
	switch status {
	case "draft", "approved", "released", "retired":
		return true
	}
	return false
}

func validation(message string) error {
	return apperrors.Validation(apperrors.CodeValidationFailed, message)
}
