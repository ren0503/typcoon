package tenant

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/database/sqlc"
	"github.com/ren0503/tycoon/internal/httpx"
	"github.com/ren0503/tycoon/internal/pagination"
)

type service struct {
	repo *repo
}

// New builds the tenant service.
func New(pool *pgxpool.Pool) TenantService {
	return &service{repo: newRepo(pool)}
}

func (s *service) Create(ctx context.Context, principal auth.Principal, input CreateInput) (sqlc.Tenant, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return sqlc.Tenant{}, apperrors.Validation(apperrors.CodeValidationFailed, "Tenant name is required.")
	}
	var id uuid.UUID
	if principal.ActorType != auth.ActorSystem && principal.TenantID != uuid.Nil {
		id = principal.TenantID
	} else {
		generated, err := uuid.NewV7()
		if err != nil {
			return sqlc.Tenant{}, apperrors.Internal(err)
		}
		id = generated
	}
	return s.repo.create(ctx, id, name)
}

func (s *service) Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.Tenant, error) {
	row, err := s.repo.get(ctx, id)
	if err != nil {
		return sqlc.Tenant{}, err
	}
	if !principal.CanAccessTenant(row.ID) {
		return sqlc.Tenant{}, apperrors.Forbidden("Tenant access is not permitted.")
	}
	return row, nil
}

func (s *service) GetActive(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.Tenant, error) {
	return s.Get(ctx, principal, id)
}

func (s *service) List(ctx context.Context, principal auth.Principal, cursor pagination.Cursor, limit int32) (Page, error) {
	onlyID := uuid.Nil
	if principal.ActorType != auth.ActorSystem {
		onlyID = principal.TenantID
	}
	rows, err := s.repo.list(ctx, onlyID, cursor.ID, limit+1)
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = httpx.NextID(len(rows), int(limit), filterParts(onlyID), page.Items[len(page.Items)-1].ID)
	}
	return page, nil
}

func (s *service) Update(ctx context.Context, principal auth.Principal, id uuid.UUID, input UpdateInput) (sqlc.Tenant, error) {
	if _, err := s.Get(ctx, principal, id); err != nil {
		return sqlc.Tenant{}, err
	}
	if input.Name == nil {
		return s.repo.get(ctx, id)
	}
	name := strings.TrimSpace(*input.Name)
	if name == "" {
		return sqlc.Tenant{}, apperrors.Validation(apperrors.CodeValidationFailed, "Tenant name must not be empty.")
	}
	return s.repo.update(ctx, id, &name)
}

func (s *service) Delete(ctx context.Context, principal auth.Principal, id uuid.UUID) error {
	if _, err := s.Get(ctx, principal, id); err != nil {
		return err
	}
	return s.repo.softDelete(ctx, id)
}

func filterParts(onlyID uuid.UUID) []string {
	return []string{"tenants", onlyID.String()}
}
