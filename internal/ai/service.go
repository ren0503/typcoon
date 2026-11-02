package ai

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
	"github.com/ren0503/tycoon/internal/pagination"
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
	q *sqlc.Queries
}

// New builds the AI parameter metadata service.
func New(pool *pgxpool.Pool) AIParamService {
	return &service{q: sqlc.New(pool)}
}

func (s *service) Create(ctx context.Context, p auth.Principal, input ParamInput) (ParamView, error) {
	if !p.CanAccessTenant(input.TenantID) {
		return ParamView{}, apperrors.Forbidden("AI parameter access is not permitted.")
	}
	if strings.TrimSpace(input.Kind) == "" || input.ParamVersion < 1 {
		return ParamView{}, validation("kind and param_version are required.")
	}
	if input.Status == "" {
		input.Status = "draft"
	}
	if !validStatus(input.Status) {
		return ParamView{}, validation("AI parameter status is not recognised.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return ParamView{}, apperrors.Internal(err)
	}
	row, err := s.q.CreateAIParam(ctx, sqlc.CreateAIParamParams{
		ID: id, TenantID: input.TenantID, Kind: input.Kind, ParamVersion: input.ParamVersion,
		StorageKey: input.StorageKey, Sha256: input.Sha256, Kid: input.Kid, Signature: input.Signature,
		Status: input.Status, ReleasedAt: input.ReleasedAt, Notes: input.Notes,
	})
	if err != nil {
		return ParamView{}, database.MapPgError(err)
	}
	return toView(row), nil
}

func (s *service) Get(ctx context.Context, p auth.Principal, id uuid.UUID) (ParamView, error) {
	row, err := s.q.GetAIParam(ctx, id)
	if err != nil {
		return ParamView{}, database.MapPgError(err)
	}
	if !p.CanAccessTenant(row.TenantID) {
		return ParamView{}, apperrors.NotFound(apperrors.CodeNotFound, "Resource not found.")
	}
	return toView(row), nil
}

func (s *service) List(ctx context.Context, p auth.Principal, tenantID uuid.UUID, kind *string, rawCursor string, limit int32) (Page[ParamView], error) {
	if !p.CanAccessTenant(tenantID) {
		return Page[ParamView]{}, apperrors.Forbidden("AI parameter access is not permitted.")
	}
	parts := []string{"ai_params", tenantID.String()}
	if kind != nil {
		parts = append(parts, *kind)
	}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[ParamView]{}, err
	}
	var kindParam *string
	if kind != nil {
		kindParam = kind
	}
	rows, err := s.q.ListAIParams(ctx, sqlc.ListAIParamsParams{
		TenantID: tenantID, Kind: kindParam, After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[ParamView]{}, database.MapPgError(err)
	}
	page := Page[ParamView]{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: rows[len(rows)-1].ID}.Encode()
	}
	for _, row := range rows {
		page.Items = append(page.Items, toView(row))
	}
	return page, nil
}

func (s *service) Update(ctx context.Context, p auth.Principal, id uuid.UUID, input ParamUpdate) (ParamView, error) {
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
	row, err := s.q.UpdateAIParamMetadata(ctx, sqlc.UpdateAIParamMetadataParams{
		StorageKey: input.StorageKey, Sha256: input.Sha256, Kid: input.Kid, Signature: input.Signature,
		Status: status, ReleasedAt: input.ReleasedAt, Notes: input.Notes, ID: id,
	})
	if err != nil {
		return ParamView{}, database.MapPgError(err)
	}
	return toView(row), nil
}

func (s *service) Delete(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.Get(ctx, p, id); err != nil {
		return err
	}
	n, err := s.q.SoftDeleteAIParam(ctx, id)
	if err != nil {
		return database.MapPgError(err)
	}
	if n == 0 {
		return database.MapPgError(pgx.ErrNoRows)
	}
	return nil
}

func validStatus(status string) bool {
	switch status {
	case "draft", "approved", "released", "retired":
		return true
	}
	return false
}

func toView(row sqlc.AiParam) ParamView {
	return ParamView{
		ID: row.ID, TenantID: row.TenantID, Kind: row.Kind, ParamVersion: row.ParamVersion,
		StorageKey: row.StorageKey, Sha256: row.Sha256, Kid: row.Kid, Signature: row.Signature,
		Status: row.Status, ReleasedAt: row.ReleasedAt, Notes: row.Notes,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version,
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
