//go:build integration

package voyage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/ship"
	"github.com/ren0503/tycoon/internal/testutil"
	"github.com/ren0503/tycoon/internal/voyage"
)

func TestVoyageModuleConstraints(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	shipSvc := ship.New(h.Pool)
	shipRow, err := shipSvc.CreateShip(ctx, principal, ship.ShipInput{Name: "Voyage Ship", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}
	svc := voyage.New(h.Pool, shipSvc)

	departure := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	created, err := svc.CreateVoyage(ctx, principal, voyage.VoyageInput{
		ShipID: shipRow.ID, Code: "VOY-1", DepartureAt: departure, ArrivalAt: departure.Add(12 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create voyage: %v", err)
	}

	if _, err := svc.CreateVoyage(ctx, principal, voyage.VoyageInput{
		ShipID: shipRow.ID, Code: "VOY-OVERLAP", DepartureAt: departure.Add(time.Hour),
		ArrivalAt: departure.Add(6 * time.Hour),
	}); err == nil {
		t.Fatal("expected overlap conflict")
	} else {
		testutil.RequireErrorCode(t, err, "overlap_conflict")
	}

	if _, err := svc.CreateVoyage(ctx, principal, voyage.VoyageInput{
		ShipID: shipRow.ID, Code: "VOY-BAD", DepartureAt: departure.Add(48 * time.Hour),
		ArrivalAt: departure.Add(47 * time.Hour),
	}); err == nil {
		t.Fatal("expected reversed time validation")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}

	port, err := svc.CreatePort(ctx, principal, voyage.PortInput{
		UnLocode: "PTSIN", Name: "Sines", CountryCode: "PT", Timezone: "Europe/Lisbon",
	})
	if err != nil {
		t.Fatalf("create port: %v", err)
	}
	call, err := svc.CreatePortCall(ctx, principal, created.ID, voyage.PortCallInput{
		PortID: port.ID, Sequence: 1, AllowsEmbark: true, AllowsDisembark: true,
	})
	if err != nil {
		t.Fatalf("create port call: %v", err)
	}
	if _, err := svc.CreatePortCall(ctx, principal, created.ID, voyage.PortCallInput{
		PortID: port.ID, Sequence: 1,
	}); err == nil {
		t.Fatal("expected duplicate sequence conflict")
	} else {
		testutil.RequireErrorCode(t, err, "conflict")
	}
	if _, err := svc.CreatePortCall(ctx, principal, created.ID, voyage.PortCallInput{
		PortID: port.ID, Sequence: 2,
	}); err != nil {
		t.Fatalf("create second port call: %v", err)
	}
	if err := svc.DeletePortCall(ctx, principal, call.ID); err != nil {
		t.Fatalf("delete port call: %v", err)
	}
	if _, err := svc.CreatePortCall(ctx, principal, created.ID, voyage.PortCallInput{
		PortID: port.ID, Sequence: 1,
	}); err != nil {
		t.Fatalf("sequence should be reusable after soft delete: %v", err)
	}

	if err := svc.DeletePort(ctx, principal, port.ID); err != nil {
		t.Fatalf("delete port: %v", err)
	}
	if _, err := svc.CreatePortCall(ctx, principal, created.ID, voyage.PortCallInput{
		PortID: port.ID, Sequence: 3,
	}); err == nil {
		t.Fatal("expected invalid reference for soft-deleted port")
	} else {
		testutil.RequireErrorCode(t, err, "invalid_reference")
	}
}
