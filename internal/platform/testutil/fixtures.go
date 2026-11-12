//go:build integration

package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/platform/errors"
)

// Principal builds a tenant-wide principal with optional ship scope.
func Principal(tenantID uuid.UUID, shipIDs ...uuid.UUID) auth.Principal {
	return auth.Principal{
		ActorType: auth.ActorAdmin,
		ActorID:   "test-admin",
		TenantID:  tenantID,
		Role:      "fleet_manager",
		ShipIDs:   shipIDs,
	}
}

// SystemPrincipal builds a system actor principal.
func SystemPrincipal() auth.Principal {
	return auth.Principal{ActorType: auth.ActorSystem, ActorID: "test-system"}
}

// RequireErrorCode fails the test unless err is an application error with code.
func RequireErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %q, got nil", code)
	}
	appErr, ok := apperrors.As(err)
	if !ok {
		t.Fatalf("expected application error %q, got %T: %v", code, err, err)
	}
	if appErr.Code != code {
		t.Fatalf("error code = %q, want %q (%v)", appErr.Code, code, err)
	}
}

// SeedTenant inserts a tenant and returns its id.
func (h *Harness) SeedTenant(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.Pool.Exec(context.Background(), `INSERT INTO tenants (id, name) VALUES ($1, $2)`, id, "test-tenant")
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return id
}

// SeedShip inserts a ship for the tenant and returns its id.
func (h *Harness) SeedShip(t *testing.T, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	imo := "IMO" + id.String()[:9]
	_, err := h.Pool.Exec(context.Background(),
		`INSERT INTO ships (id, tenant_id, name, imo_number, timezone) VALUES ($1, $2, $3, $4, 'UTC')`,
		id, tenantID, "Test Ship", imo)
	if err != nil {
		t.Fatalf("seed ship: %v", err)
	}
	return id
}

// SeedVoyage inserts a non-overlapping voyage for the ship.
func (h *Harness) SeedVoyage(t *testing.T, shipID uuid.UUID, code string, departure time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.Pool.Exec(context.Background(), `
		INSERT INTO voyages (id, ship_id, code, departure_at, arrival_at)
		VALUES ($1, $2, $3, $4, $5)`,
		id, shipID, code, departure, departure.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("seed voyage: %v", err)
	}
	return id
}

// SeedBooking inserts a minimal pending booking for the ship and voyage.
func (h *Harness) SeedBooking(t *testing.T, shipID, voyageID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.Pool.Exec(context.Background(), `
		INSERT INTO bookings (id, ship_id, voyage_id, embark_seq, disembark_seq)
		VALUES ($1, $2, $3, 1, 5)`, id, shipID, voyageID)
	if err != nil {
		t.Fatalf("seed booking: %v", err)
	}
	return id
}

// SeedStaff inserts an active staff member for the tenant and ship.
func (h *Harness) SeedStaff(t *testing.T, tenantID, shipID uuid.UUID, code string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.Pool.Exec(context.Background(), `
		INSERT INTO staff (id, tenant_id, ship_id, employee_code, full_name, department)
		VALUES ($1, $2, $3, $4, $5, 'security')`, id, tenantID, shipID, code, "Test Staff "+code)
	if err != nil {
		t.Fatalf("seed staff: %v", err)
	}
	return id
}

// SeedDeck inserts a deck for the ship.
func (h *Harness) SeedDeck(t *testing.T, shipID uuid.UUID, level int) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.Pool.Exec(context.Background(),
		`INSERT INTO decks (id, ship_id, level, name) VALUES ($1, $2, $3, $4)`,
		id, shipID, level, "Deck")
	if err != nil {
		t.Fatalf("seed deck: %v", err)
	}
	return id
}

// SeedRoom inserts a room on the ship and deck.
func (h *Harness) SeedRoom(t *testing.T, shipID, deckID uuid.UUID, number string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.Pool.Exec(context.Background(), `
		INSERT INTO rooms (id, ship_id, deck_id, number, room_type, capacity)
		VALUES ($1, $2, $3, $4, 'inside', 2)`, id, shipID, deckID, number)
	if err != nil {
		t.Fatalf("seed room: %v", err)
	}
	return id
}

// SeedArea inserts a common area on the ship and deck.
func (h *Harness) SeedArea(t *testing.T, shipID, deckID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.Pool.Exec(context.Background(), `
		INSERT INTO areas (id, ship_id, deck_id, name, area_type)
		VALUES ($1, $2, $3, $4, 'restaurant')`, id, shipID, deckID, name)
	if err != nil {
		t.Fatalf("seed area: %v", err)
	}
	return id
}

// SeedAccount inserts a guest account and returns its id.
func (h *Harness) SeedAccount(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.Pool.Exec(context.Background(),
		`INSERT INTO accounts (id, email, display_name) VALUES ($1, $2, $3)`,
		id, id.String()+"@example.com", "Test Guest")
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}
	return id
}

// SeedPartner inserts a partner for the tenant.
func (h *Harness) SeedPartner(t *testing.T, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.Pool.Exec(context.Background(), `
		INSERT INTO partners (id, tenant_id, name, business_type)
		VALUES ($1, $2, $3, 'entertainment')`, id, tenantID, "Test Partner")
	if err != nil {
		t.Fatalf("seed partner: %v", err)
	}
	return id
}

// SeedPartnerService inserts a partner service on the ship and partner.
func (h *Harness) SeedPartnerService(t *testing.T, shipID, partnerID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := h.Pool.Exec(context.Background(), `
		INSERT INTO partner_services (id, ship_id, partner_id, name, category)
		VALUES ($1, $2, $3, $4, 'entertainment')`, id, shipID, partnerID, "Test Service")
	if err != nil {
		t.Fatalf("seed partner service: %v", err)
	}
	return id
}
