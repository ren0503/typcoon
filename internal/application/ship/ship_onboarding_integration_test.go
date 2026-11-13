//go:build integration

package ship_test

import (
	"context"
	shiprepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/ship"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/application/ship"
	"github.com/ren0503/tycoon/internal/platform/testutil"
)

func TestShipOnboardingLifecycle(t *testing.T) {
	h := testutil.StartPostgres(t)
	t.Cleanup(func() { h.TruncateAll(t) })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	svc := ship.New(shiprepo.New(h.Pool), testutil.Logger(t))

	created, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "Onboarding", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}
	if created.Status != "active" {
		t.Fatalf("default status = %q, want active", created.Status)
	}
	for _, status := range []string{"active", "maintenance", "retired"} {
		if _, err := svc.UpdateShip(ctx, principal, created.ID, ship.ShipUpdate{Status: &status}); err != nil {
			t.Fatalf("set status %s: %v", status, err)
		}
	}
	if _, err := svc.UpdateShip(ctx, principal, created.ID, ship.ShipUpdate{Status: stringPtr("invalid")}); err == nil {
		t.Fatal("expected invalid status rejection")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}
	if _, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "  ", Timezone: "UTC"}); err == nil {
		t.Fatal("expected blank name rejection")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}
	if _, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "Bad TZ", Timezone: "Mars/Olympus"}); err == nil {
		t.Fatal("expected invalid timezone rejection")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}

	if _, err := svc.GetShip(ctx, principal, created.ID); err != nil {
		t.Fatalf("get ship: %v", err)
	}
	page, err := svc.ListShips(ctx, principal, "", 20)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("list ships = %d, err %v", len(page.Items), err)
	}
	name := "Renamed"
	updated, err := svc.UpdateShip(ctx, principal, created.ID, ship.ShipUpdate{Name: &name})
	if err != nil || updated.Name != name {
		t.Fatalf("update ship: %v, name %q", err, updated.Name)
	}
	assertCount(t, h, `SELECT count(*) FROM decks WHERE ship_id = $1`, created.ID, 0)
	assertCount(t, h, `SELECT count(*) FROM areas WHERE ship_id = $1`, created.ID, 0)
	assertCount(t, h, `SELECT count(*) FROM rooms WHERE ship_id = $1`, created.ID, 0)

	if err := svc.DeleteShip(ctx, principal, created.ID); err != nil {
		t.Fatalf("delete ship: %v", err)
	}
	if _, err := svc.GetShip(ctx, principal, created.ID); err == nil {
		t.Fatal("expected deleted ship to be hidden")
	} else {
		testutil.RequireErrorCode(t, err, "not_found")
	}
}

func stringPtr(value string) *string { return &value }
