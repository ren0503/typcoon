package voyage

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	appship "github.com/ren0503/tycoon/internal/application/ship"
	"github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

var portCallStatuses = map[string]bool{
	"scheduled": true, "arrived": true, "departed": true, "skipped": true,
}

type service struct {
	repo   Repository
	logger *slog.Logger
	ships  appship.Service
}

// New builds the voyage service.
func New(repo Repository, logger *slog.Logger, ships appship.Service) Service {
	return &service{repo: repo, logger: logger.With("module", "voyage"), ships: ships}
}

func (s *service) CreatePort(ctx context.Context, _ auth.Principal, input PortInput) (Port, error) {
	locode := strings.ToUpper(strings.TrimSpace(input.UnLocode))
	name := strings.TrimSpace(input.Name)
	country := strings.ToUpper(strings.TrimSpace(input.CountryCode))
	if locode == "" || name == "" || len(country) != 2 {
		s.logger.ErrorContext(ctx, "validation failed: port code, name, and country code are required", "input", input)
		return Port{}, validation("Port code, name, and a two-letter country code are required.")
	}
	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		s.logger.ErrorContext(ctx, "validation failed: invalid time zone", "input", input, "error", err)
		return Port{}, validation("Port time zone must be a valid IANA time zone.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create port", "input", input, "error", err)
		return Port{}, apperrors.Internal(err)
	}
	return s.repo.CreatePort(ctx, CreatePortParams{
		ID: id, UnLocode: locode, Name: name, CountryCode: country,
		Latitude: input.Latitude, Longitude: input.Longitude, Timezone: timezone,
	})
}

func (s *service) GetPort(ctx context.Context, _ auth.Principal, id uuid.UUID) (Port, error) {
	return s.repo.GetPort(ctx, id)
}

func (s *service) ListPorts(ctx context.Context, _ auth.Principal, rawCursor string, limit int32) (Page[Port], error) {
	parts := []string{"ports"}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[Port]{}, err
	}
	rows, err := s.repo.ListPorts(ctx, ListPortsParams{After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[Port]{}, err
	}
	page := Page[Port]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdatePort(ctx context.Context, p auth.Principal, id uuid.UUID, input PortUpdate) (Port, error) {
	current, err := s.GetPort(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "port not found or access denied", "principal", p, "port_id", id, "error", err)
		return Port{}, err
	}
	if input.Name != nil {
		if strings.TrimSpace(*input.Name) == "" {
			s.logger.ErrorContext(ctx, "validation failed: port name must not be empty", "principal", p, "input", input)
			return Port{}, validation("Port name must not be empty.")
		}
		current.Name = strings.TrimSpace(*input.Name)
	}
	if input.Timezone != nil {
		if _, err := time.LoadLocation(*input.Timezone); err != nil {
			s.logger.ErrorContext(ctx, "validation failed: invalid time zone", "principal", p, "input", input, "error", err)
			return Port{}, validation("Port time zone must be a valid IANA time zone.")
		}
		current.Timezone = strings.TrimSpace(*input.Timezone)
	}
	if input.Latitude != nil {
		current.Latitude = input.Latitude
	}
	if input.Longitude != nil {
		current.Longitude = input.Longitude
	}
	port, err := s.repo.UpdatePort(ctx, UpdatePortParams{
		ID: id, Name: current.Name, Latitude: current.Latitude, Longitude: current.Longitude, Timezone: current.Timezone,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update port", "principal", p, "input", input, "error", err)
		return Port{}, apperrors.Internal(err)
	}
	return port, nil
}

func (s *service) DeletePort(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetPort(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "port not found or access denied", "principal", p, "port_id", id, "error", err)
		return err
	}
	return s.repo.DeletePort(ctx, id)
}

func (s *service) CreateVoyage(ctx context.Context, p auth.Principal, input VoyageInput) (Voyage, error) {
	if err := s.requireShip(ctx, p, input.ShipID); err != nil {
		s.logger.ErrorContext(ctx, "validation failed: ship not found or access denied", "principal", p, "ship_id", input.ShipID, "error", err)
		return Voyage{}, err
	}
	code := strings.TrimSpace(input.Code)
	if code == "" {
		s.logger.ErrorContext(ctx, "validation failed: voyage code is required", "principal", p, "input", input)
		return Voyage{}, validation("Voyage code is required.")
	}
	if !input.ArrivalAt.After(input.DepartureAt) {
		s.logger.ErrorContext(ctx, "validation failed: voyage arrival must be after departure", "principal", p, "input", input)
		return Voyage{}, validation("Voyage arrival must be after departure.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create voyage", "principal", p, "input", input, "error", err)
		return Voyage{}, apperrors.Internal(err)
	}
	voyage, err := s.repo.CreateVoyage(ctx, CreateVoyageParams{
		ID: id, ShipID: input.ShipID, Code: code, DepartureAt: input.DepartureAt, ArrivalAt: input.ArrivalAt,
		BoardingOpensAt: input.BoardingOpensAt, ManifestFrozenAt: input.ManifestFrozenAt,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create voyage", "principal", p, "input", input, "error", err)
		return Voyage{}, apperrors.Internal(err)
	}
	return voyage, nil
}

func (s *service) GetVoyage(ctx context.Context, p auth.Principal, id uuid.UUID) (Voyage, error) {
	row, err := s.repo.GetVoyage(ctx, id)
	if err != nil {
		return Voyage{}, err
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		return Voyage{}, err
	}
	return row, nil
}

func (s *service) Lookup(ctx context.Context, id uuid.UUID) (Voyage, error) {
	return s.repo.GetVoyage(ctx, id)
}

func (s *service) ListVoyages(ctx context.Context, p auth.Principal, shipID *uuid.UUID, rawCursor string, limit int32) (Page[Voyage], error) {
	var (
		rows  []Voyage
		parts []string
		err   error
	)
	shipPart := ""
	if shipID != nil {
		if err := s.requireShip(ctx, p, *shipID); err != nil {
			return Page[Voyage]{}, err
		}
		shipPart = shipID.String()
	} else if p.ActorType != auth.ActorSystem {
		shipPart = "tenant:" + p.TenantID.String()
	} else {
		shipPart = "all"
	}
	parts = []string{"voyages", shipPart}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[Voyage]{}, err
	}
	afterSort, err := parseSortTime(cursor.Sort)
	if err != nil {
		return Page[Voyage]{}, err
	}
	if shipID != nil {
		rows, err = s.repo.ListVoyages(ctx, ListVoyagesParams{
			ShipID: nullUUID(*shipID), AfterSort: afterSort, After: nullUUID(cursor.ID), PageLimit: limit + 1,
		})
	} else if p.ActorType == auth.ActorSystem {
		rows, err = s.repo.ListVoyages(ctx, ListVoyagesParams{
			AfterSort: afterSort, After: nullUUID(cursor.ID), PageLimit: limit + 1,
		})
	} else {
		rows, err = s.repo.ListVoyagesForTenant(ctx, ListVoyagesForTenantParams{
			TenantID: p.TenantID, AfterSort: afterSort, After: nullUUID(cursor.ID), PageLimit: limit + 1,
		})
	}
	if err != nil {
		return Page[Voyage]{}, err
	}
	page := Page[Voyage]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = pagination.Cursor{
			Filter: pagination.FilterHash(parts...),
			Sort:   last.DepartureAt.UTC().Format(time.RFC3339Nano),
			ID:     last.ID,
		}.Encode()
	}
	return page, nil
}

func (s *service) UpdateVoyage(ctx context.Context, p auth.Principal, id uuid.UUID, input VoyageUpdate) (Voyage, error) {
	current, err := s.GetVoyage(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "voyage not found or access denied", "principal", p, "voyage_id", id, "error", err)
		return Voyage{}, err
	}
	if input.Code != nil {
		if strings.TrimSpace(*input.Code) == "" {
			s.logger.ErrorContext(ctx, "validate failed: voyage code is required", "principal", p, "input", input)
			return Voyage{}, validation("Voyage code must not be empty.")
		}
		current.Code = strings.TrimSpace(*input.Code)
	}
	if input.DepartureAt != nil {
		current.DepartureAt = *input.DepartureAt
	}
	if input.ArrivalAt != nil {
		current.ArrivalAt = *input.ArrivalAt
	}
	if input.BoardingOpensAt != nil {
		current.BoardingOpensAt = input.BoardingOpensAt
	}
	if input.ManifestFrozenAt != nil {
		current.ManifestFrozenAt = input.ManifestFrozenAt
	}
	if !current.ArrivalAt.After(current.DepartureAt) {
		s.logger.ErrorContext(ctx, "validation failed: voyage arrival must be after departure", "principal", p, "input", input)
		return Voyage{}, validation("Voyage arrival must be after departure.")
	}
	voyage, err := s.repo.UpdateVoyage(ctx, UpdateVoyageParams{
		ID: id, Code: current.Code, DepartureAt: current.DepartureAt, ArrivalAt: current.ArrivalAt,
		BoardingOpensAt: current.BoardingOpensAt, ManifestFrozenAt: current.ManifestFrozenAt,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update voyage", "principal", p, "input", input, "error", err)
		return Voyage{}, apperrors.Internal(err)
	}
	return voyage, nil
}

func (s *service) DeleteVoyage(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetVoyage(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "voyage not found or access denied", "principal", p, "voyage_id", id, "error", err)
		return err
	}
	return s.repo.DeleteVoyage(ctx, id)
}

func (s *service) CreatePortCall(ctx context.Context, p auth.Principal, voyageID uuid.UUID, input PortCallInput) (PortCall, error) {
	v, err := s.GetVoyage(ctx, p, voyageID)
	if err != nil {
		s.logger.ErrorContext(ctx, "voyage not found or access denied", "principal", p, "voyage_id", voyageID, "error", err)
		return PortCall{}, err
	}
	ok, err := s.repo.PortExists(ctx, input.PortID)
	if err != nil {
		s.logger.ErrorContext(ctx, "error checking port existence", "principal", p, "port_id", input.PortID, "error", err)
		return PortCall{}, err
	}
	if !ok {
		s.logger.ErrorContext(ctx, "validation failed: port does not exist", "principal", p, "port_id", input.PortID)
		return PortCall{}, apperrors.Validation(apperrors.CodeInvalidReference, "Port does not exist.")
	}
	if input.Sequence < 1 {
		s.logger.ErrorContext(ctx, "validation failed: port call sequence must be at least 1", "principal", p, "input", input)
		return PortCall{}, validation("Port call sequence must be at least 1.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return PortCall{}, apperrors.Internal(err)
	}
	portCall, err := s.repo.CreatePortCall(ctx, CreatePortCallParams{
		ID: id, ShipID: v.ShipID, VoyageID: voyageID, PortID: input.PortID, Sequence: input.Sequence,
		Eta: input.Eta, Etd: input.Etd, AllowsEmbark: input.AllowsEmbark, AllowsDisembark: input.AllowsDisembark,
		Status: "scheduled",
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create port call", "principal", p, "input", input, "error", err)
		return PortCall{}, apperrors.Internal(err)
	}
	return portCall, nil
}

func (s *service) GetPortCall(ctx context.Context, p auth.Principal, id uuid.UUID) (PortCall, error) {
	row, err := s.repo.GetPortCall(ctx, id)
	if err != nil {
		return PortCall{}, err
	}
	if _, err := s.GetVoyage(ctx, p, row.VoyageID); err != nil {
		return PortCall{}, err
	}
	return row, nil
}

func (s *service) ListPortCalls(ctx context.Context, p auth.Principal, voyageID uuid.UUID, rawCursor string, limit int32) (Page[PortCall], error) {
	if _, err := s.GetVoyage(ctx, p, voyageID); err != nil {
		return Page[PortCall]{}, err
	}
	parts := []string{"port_calls", voyageID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[PortCall]{}, err
	}
	afterSeq := int64(0)
	if cursor.Sort != "" {
		parsed, err := strconv.ParseInt(cursor.Sort, 10, 64)
		if err != nil {
			return Page[PortCall]{}, apperrors.BadRequest(apperrors.CodeInvalidCursor, "Cursor is malformed or belongs to another filter.")
		}
		afterSeq = parsed
	}
	rows, err := s.repo.ListPortCalls(ctx, ListPortCallsParams{
		VoyageID: voyageID, AfterSeq: &afterSeq, After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[PortCall]{}, err
	}
	page := Page[PortCall]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = pagination.Cursor{
			Filter: pagination.FilterHash(parts...), Sort: strconv.FormatInt(int64(last.Sequence), 10), ID: last.ID,
		}.Encode()
	}
	return page, nil
}

func (s *service) UpdatePortCall(ctx context.Context, p auth.Principal, id uuid.UUID, input PortCallUpdate) (PortCall, error) {
	current, err := s.GetPortCall(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "port call not found or access denied", "principal", p, "port_call_id", id, "error", err)
		return PortCall{}, err
	}
	if input.Eta != nil {
		current.Eta = input.Eta
	}
	if input.Etd != nil {
		current.Etd = input.Etd
	}
	if input.Ata != nil {
		current.Ata = input.Ata
	}
	if input.Atd != nil {
		current.Atd = input.Atd
	}
	if input.AllowsEmbark != nil {
		current.AllowsEmbark = *input.AllowsEmbark
	}
	if input.AllowsDisembark != nil {
		current.AllowsDisembark = *input.AllowsDisembark
	}
	if input.Status != nil {
		if !portCallStatuses[*input.Status] {
			s.logger.ErrorContext(ctx, "validation failed: unrecognized port call status", "principal", p, "input", input)
			return PortCall{}, validation("Port call status is not recognised.")
		}
		current.Status = *input.Status
	}
	portCall, err := s.repo.UpdatePortCall(ctx, UpdatePortCallParams{
		ID: id, Sequence: current.Sequence, Eta: current.Eta, Etd: current.Etd, Ata: current.Ata, Atd: current.Atd,
		AllowsEmbark: current.AllowsEmbark, AllowsDisembark: current.AllowsDisembark, Status: current.Status,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update port call", "principal", p, "input", input, "error", err)
		return PortCall{}, apperrors.Internal(err)
	}
	return portCall, nil
}

func (s *service) DeletePortCall(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetPortCall(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "port call not found or access denied", "principal", p, "port_call_id", id, "error", err)
		return err
	}
	return s.repo.DeletePortCall(ctx, id)
}

func (s *service) requireShip(ctx context.Context, p auth.Principal, shipID uuid.UUID) error {
	tenantID, err := s.ships.TenantID(ctx, shipID)
	if err != nil {
		return err
	}
	if !p.CanAccessTenant(tenantID) || !p.CanAccessShip(shipID) {
		return apperrors.Forbidden("Voyage access is not permitted.")
	}
	return nil
}

func parseSortTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, apperrors.BadRequest(apperrors.CodeInvalidCursor, "Cursor is malformed or belongs to another filter.")
	}
	return &parsed, nil
}

func validation(message string) error {
	return apperrors.Validation(apperrors.CodeValidationFailed, message)
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}
