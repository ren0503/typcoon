// Package voyage implements voyage persistence.
package voyage

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	appvoyage "github.com/ren0503/tycoon/internal/application/voyage"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Repo struct{ q *sqlc.Queries }

func New(pool *pgxpool.Pool) *Repo { return &Repo{q: sqlc.New(pool)} }

var _ appvoyage.Repository = (*Repo)(nil)

func (r *Repo) CreatePort(ctx context.Context, p appvoyage.CreatePortParams) (appvoyage.Port, error) {
	row, err := r.q.CreatePort(ctx, sqlc.CreatePortParams{ID: p.ID, UnLocode: p.UnLocode, Name: p.Name, CountryCode: p.CountryCode, Latitude: decimalValue(p.Latitude), Longitude: decimalValue(p.Longitude), Timezone: p.Timezone})
	return convertPort(row), postgres.MapPgError(err)
}
func (r *Repo) GetPort(ctx context.Context, id uuid.UUID) (appvoyage.Port, error) {
	row, err := r.q.GetPort(ctx, id)
	return convertPort(row), postgres.MapPgError(err)
}
func (r *Repo) ListPorts(ctx context.Context, p appvoyage.ListPortsParams) ([]appvoyage.Port, error) {
	rows, err := r.q.ListPorts(ctx, sqlc.ListPortsParams{After: p.After, PageLimit: p.PageLimit})
	out := make([]appvoyage.Port, len(rows))
	for i, row := range rows {
		out[i] = convertPort(row)
	}
	return out, postgres.MapPgError(err)
}
func (r *Repo) UpdatePort(ctx context.Context, p appvoyage.UpdatePortParams) (appvoyage.Port, error) {
	row, err := r.q.UpdatePort(ctx, sqlc.UpdatePortParams{ID: p.ID, Name: p.Name, Latitude: decimalValue(p.Latitude), Longitude: decimalValue(p.Longitude), Timezone: p.Timezone})
	return convertPort(row), postgres.MapPgError(err)
}
func (r *Repo) DeletePort(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeletePort(ctx, id))
}
func (r *Repo) PortExists(ctx context.Context, id uuid.UUID) (bool, error) {
	ok, err := r.q.PortExists(ctx, id)
	return ok, postgres.MapPgError(err)
}

func (r *Repo) CreateVoyage(ctx context.Context, p appvoyage.CreateVoyageParams) (appvoyage.Voyage, error) {
	row, err := r.q.CreateVoyage(ctx, sqlc.CreateVoyageParams{ID: p.ID, ShipID: p.ShipID, Code: p.Code, DepartureAt: p.DepartureAt, ArrivalAt: p.ArrivalAt, BoardingOpensAt: p.BoardingOpensAt, ManifestFrozenAt: p.ManifestFrozenAt})
	return convertVoyage(row), postgres.MapPgError(err)
}
func (r *Repo) GetVoyage(ctx context.Context, id uuid.UUID) (appvoyage.Voyage, error) {
	row, err := r.q.GetVoyage(ctx, id)
	return convertVoyage(row), postgres.MapPgError(err)
}
func (r *Repo) ListVoyages(ctx context.Context, p appvoyage.ListVoyagesParams) ([]appvoyage.Voyage, error) {
	rows, err := r.q.ListVoyages(ctx, sqlc.ListVoyagesParams{ShipID: p.ShipID, AfterSort: p.AfterSort, After: p.After, PageLimit: p.PageLimit})
	out := make([]appvoyage.Voyage, len(rows))
	for i, row := range rows {
		out[i] = convertVoyage(row)
	}
	return out, postgres.MapPgError(err)
}
func (r *Repo) ListVoyagesForTenant(ctx context.Context, p appvoyage.ListVoyagesForTenantParams) ([]appvoyage.Voyage, error) {
	rows, err := r.q.ListVoyagesForTenant(ctx, sqlc.ListVoyagesForTenantParams{TenantID: p.TenantID, AfterSort: p.AfterSort, After: p.After, PageLimit: p.PageLimit})
	out := make([]appvoyage.Voyage, len(rows))
	for i, row := range rows {
		out[i] = convertVoyage(row)
	}
	return out, postgres.MapPgError(err)
}
func (r *Repo) UpdateVoyage(ctx context.Context, p appvoyage.UpdateVoyageParams) (appvoyage.Voyage, error) {
	row, err := r.q.UpdateVoyage(ctx, sqlc.UpdateVoyageParams{Code: p.Code, DepartureAt: p.DepartureAt, ArrivalAt: p.ArrivalAt, BoardingOpensAt: p.BoardingOpensAt, ManifestFrozenAt: p.ManifestFrozenAt, ID: p.ID})
	return convertVoyage(row), postgres.MapPgError(err)
}
func (r *Repo) DeleteVoyage(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeleteVoyage(ctx, id))
}

func (r *Repo) CreatePortCall(ctx context.Context, p appvoyage.CreatePortCallParams) (appvoyage.PortCall, error) {
	row, err := r.q.CreatePortCall(ctx, sqlc.CreatePortCallParams{ID: p.ID, ShipID: p.ShipID, VoyageID: p.VoyageID, PortID: p.PortID, Sequence: p.Sequence, Eta: p.Eta, Etd: p.Etd, Ata: p.Ata, Atd: p.Atd, AllowsEmbark: p.AllowsEmbark, AllowsDisembark: p.AllowsDisembark, Status: p.Status})
	return convertPortCall(row), postgres.MapPgError(err)
}
func (r *Repo) GetPortCall(ctx context.Context, id uuid.UUID) (appvoyage.PortCall, error) {
	row, err := r.q.GetPortCall(ctx, id)
	return convertPortCall(row), postgres.MapPgError(err)
}
func (r *Repo) ListPortCalls(ctx context.Context, p appvoyage.ListPortCallsParams) ([]appvoyage.PortCall, error) {
	rows, err := r.q.ListPortCalls(ctx, sqlc.ListPortCallsParams{VoyageID: p.VoyageID, AfterSeq: p.AfterSeq, After: p.After, PageLimit: p.PageLimit})
	out := make([]appvoyage.PortCall, len(rows))
	for i, row := range rows {
		out[i] = convertPortCall(row)
	}
	return out, postgres.MapPgError(err)
}
func (r *Repo) UpdatePortCall(ctx context.Context, p appvoyage.UpdatePortCallParams) (appvoyage.PortCall, error) {
	row, err := r.q.UpdatePortCall(ctx, sqlc.UpdatePortCallParams{Sequence: p.Sequence, Eta: p.Eta, Etd: p.Etd, Ata: p.Ata, Atd: p.Atd, AllowsEmbark: p.AllowsEmbark, AllowsDisembark: p.AllowsDisembark, Status: p.Status, ID: p.ID})
	return convertPortCall(row), postgres.MapPgError(err)
}
func (r *Repo) DeletePortCall(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeletePortCall(ctx, id))
}

func mapDelete(_ int64, err error) error {
	if err != nil {
		return postgres.MapPgError(err)
	}
	return nil
}

func decimalValue(v *float64) decimal.NullDecimal {
	if v == nil {
		return decimal.NullDecimal{}
	}
	return decimal.NullDecimal{Decimal: decimal.NewFromFloat(*v), Valid: true}
}
func decimalPtr(v decimal.NullDecimal) *float64 {
	if !v.Valid {
		return nil
	}
	f := v.Decimal.InexactFloat64()
	return &f
}
func convertPort(row sqlc.Port) appvoyage.Port {
	return appvoyage.Port{ID: row.ID, UnLocode: row.UnLocode, Name: row.Name, CountryCode: row.CountryCode, Latitude: decimalPtr(row.Latitude), Longitude: decimalPtr(row.Longitude), Timezone: row.Timezone, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version, DeletedAt: row.DeletedAt}
}
func convertVoyage(row sqlc.Voyage) appvoyage.Voyage {
	return appvoyage.Voyage{ID: row.ID, ShipID: row.ShipID, Code: row.Code, Status: row.Status, DepartureAt: row.DepartureAt, ArrivalAt: row.ArrivalAt, BoardingOpensAt: row.BoardingOpensAt, ManifestFrozenAt: row.ManifestFrozenAt, Ownership: row.Ownership, ScheduleVersion: row.ScheduleVersion, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version, DeletedAt: row.DeletedAt}
}
func convertPortCall(row sqlc.PortCall) appvoyage.PortCall {
	return appvoyage.PortCall{ID: row.ID, ShipID: row.ShipID, VoyageID: row.VoyageID, PortID: row.PortID, Sequence: row.Sequence, Eta: row.Eta, Etd: row.Etd, Ata: row.Ata, Atd: row.Atd, AllowsEmbark: row.AllowsEmbark, AllowsDisembark: row.AllowsDisembark, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version, DeletedAt: row.DeletedAt}
}
