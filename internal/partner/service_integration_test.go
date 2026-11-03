//go:build integration

package partner_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ren0503/tycoon/internal/partner"
	"github.com/ren0503/tycoon/internal/ship"
	"github.com/ren0503/tycoon/internal/testutil"
)

func TestPartnerContractAndSlotRules(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	shipSvc := ship.New(h.Pool)
	shipRow, err := shipSvc.CreateShip(ctx, principal, ship.ShipInput{Name: "Partner Ship", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}
	svc := partner.New(h.Pool, shipSvc)

	p, err := svc.CreatePartner(ctx, principal, partner.PartnerInput{Name: "Vendor", BusinessType: "spa"})
	if err != nil {
		t.Fatalf("create partner: %v", err)
	}

	start := pgtype.Date{Time: time.Now().UTC().Truncate(24 * time.Hour), Valid: true}
	invalidEnd := pgtype.Date{Time: start.Time.Add(-24 * time.Hour), Valid: true}
	if _, err := svc.CreateContract(ctx, principal, p.ID, partner.ContractInput{
		ShipID: shipRow.ID, StartsOn: start, EndsOn: invalidEnd,
	}); err == nil {
		t.Fatal("expected contract date validation")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}

	deck := h.SeedDeck(t, shipRow.ID, 1)
	area := h.SeedArea(t, shipRow.ID, deck, "Spa")
	slotStart := time.Now().UTC().Add(24 * time.Hour)
	first, err := svc.CreateSlot(ctx, principal, p.ID, partner.SlotInput{
		ShipID: shipRow.ID, AreaID: area, StartAt: slotStart, EndAt: slotStart.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create requested slot: %v", err)
	}
	if first.Status != "requested" {
		t.Fatalf("new slot status = %q, want requested", first.Status)
	}
	second, err := svc.CreateSlot(ctx, principal, p.ID, partner.SlotInput{
		ShipID: shipRow.ID, AreaID: area, StartAt: slotStart.Add(30 * time.Minute), EndAt: slotStart.Add(90 * time.Minute),
	})
	if err != nil {
		t.Fatalf("overlapping requested slot should be allowed: %v", err)
	}
	if _, err := svc.UpdateSlot(ctx, principal, first.ID, partner.SlotUpdate{Status: ptr("approved")}); err != nil {
		t.Fatalf("approve first slot: %v", err)
	}
	if _, err := svc.UpdateSlot(ctx, principal, second.ID, partner.SlotUpdate{Status: ptr("approved")}); err == nil {
		t.Fatal("expected overlap conflict when approving overlapping slot")
	} else {
		testutil.RequireErrorCode(t, err, "overlap_conflict")
	}

	if _, err := svc.CreateSlot(ctx, principal, p.ID, partner.SlotInput{
		ShipID: shipRow.ID, AreaID: area, StartAt: slotStart.Add(5 * time.Hour), EndAt: slotStart.Add(4 * time.Hour),
	}); err == nil {
		t.Fatal("expected slot time validation")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}
}

func ptr[T any](v T) *T { return &v }
