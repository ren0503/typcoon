package tenant

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

type service struct {
	repo Repository
}

// New builds the tenant service from an injected repository.
func New(repo Repository) TenantService {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, principal auth.Principal, input CreateInput) (Tenant, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Tenant{}, apperrors.Validation(apperrors.CodeValidationFailed, "Tenant name is required.")
	}
	var id uuid.UUID
	if principal.ActorType != auth.ActorSystem && principal.TenantID != uuid.Nil {
		id = principal.TenantID
	} else {
		generated, err := uuid.NewV7()
		if err != nil {
			return Tenant{}, apperrors.Internal(err)
		}
		id = generated
	}
	return s.repo.Create(ctx, CreateParams{ID: id, Name: name})
}

func (s *service) Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (Tenant, error) {
	row, err := s.repo.Get(ctx, id)
	if err != nil {
		return Tenant{}, err
	}
	if !principal.CanAccessTenant(row.ID) {
		return Tenant{}, apperrors.Forbidden("Tenant access is not permitted.")
	}
	return row, nil
}

func (s *service) GetActive(ctx context.Context, principal auth.Principal, id uuid.UUID) (Tenant, error) {
	return s.Get(ctx, principal, id)
}

func (s *service) List(ctx context.Context, principal auth.Principal, cursor pagination.Cursor, limit int32) (Page, error) {
	onlyID := uuid.Nil
	if principal.ActorType != auth.ActorSystem {
		onlyID = principal.TenantID
	}
	rows, err := s.repo.List(ctx, ListParams{OnlyID: nullUUID(onlyID), After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = nextID(len(rows), int(limit), filterParts(onlyID), page.Items[len(page.Items)-1].ID)
	}
	return page, nil
}

func (s *service) Update(ctx context.Context, principal auth.Principal, id uuid.UUID, input UpdateInput) (Tenant, error) {
	if _, err := s.Get(ctx, principal, id); err != nil {
		return Tenant{}, err
	}
	if input.Name == nil {
		return s.repo.Get(ctx, id)
	}
	name := strings.TrimSpace(*input.Name)
	if name == "" {
		return Tenant{}, apperrors.Validation(apperrors.CodeValidationFailed, "Tenant name must not be empty.")
	}
	return s.repo.Update(ctx, UpdateParams{ID: id, Name: &name})
}

func (s *service) Delete(ctx context.Context, principal auth.Principal, id uuid.UUID) error {
	if _, err := s.Get(ctx, principal, id); err != nil {
		return err
	}
	return s.repo.SoftDelete(ctx, id)
}

func filterParts(onlyID uuid.UUID) []string {
	return []string{"tenants", onlyID.String()}
}

func nextID(rows, pageLen int, filterParts []string, last uuid.UUID) string {
	if rows <= pageLen {
		return ""
	}
	return pagination.Cursor{Filter: pagination.FilterHash(filterParts...), ID: last}.Encode()
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}
