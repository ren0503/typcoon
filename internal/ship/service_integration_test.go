//go:build integration

package ship_test

import (
	"context"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/ship"
	"github.com/ren0503/tycoon/internal/testutil"
)

func TestShipModuleCRUDAndConstraints(t *testing.T) {
	h := testutil.StartPostgres(t)
	t.Cleanup(func() { h.TruncateAll(t) })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	svc := ship.New(h.Pool, testutil.Logger(t), nil)

	imo := "IMO1234567"
	created, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "Oceanic", ImoNumber: &imo, Timezone: "Europe/Lisbon"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}

	other, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "Other", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create second ship: %v", err)
	}

	if _, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "Duplicate", ImoNumber: &imo, Timezone: "UTC"}); err == nil {
		t.Fatal("expected duplicate IMO conflict")
	} else {
		testutil.RequireErrorCode(t, err, "conflict")
	}

	deck, err := svc.CreateDeck(ctx, principal, created.ID, ship.DeckInput{Level: 1, Name: "Main"})
	if err != nil {
		t.Fatalf("create deck: %v", err)
	}
	if _, err := svc.CreateDeck(ctx, principal, created.ID, ship.DeckInput{Level: 1, Name: "Duplicate"}); err == nil {
		t.Fatal("expected duplicate deck level conflict")
	} else {
		testutil.RequireErrorCode(t, err, "conflict")
	}

	room, err := svc.CreateRoom(ctx, principal, created.ID, ship.RoomInput{
		DeckID: deck.ID, Number: "101", RoomType: "inside", Capacity: 2, IsBookable: true,
	})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if _, err := svc.CreateRoom(ctx, principal, created.ID, ship.RoomInput{
		DeckID: deck.ID, Number: "101", RoomType: "inside", Capacity: 2,
	}); err == nil {
		t.Fatal("expected duplicate room number conflict")
	} else {
		testutil.RequireErrorCode(t, err, "conflict")
	}
	if _, err := svc.CreateRoom(ctx, principal, created.ID, ship.RoomInput{
		DeckID: deck.ID, Number: "102", RoomType: "inside", Capacity: 0,
	}); err == nil {
		t.Fatal("expected capacity validation error")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}

	foreignDeck := h.SeedDeck(t, other.ID, 1)
	if _, err := svc.CreateArea(ctx, principal, created.ID, ship.AreaInput{
		DeckID: foreignDeck, Name: "Foreign", AreaType: "restaurant",
	}); err == nil {
		t.Fatal("expected invalid reference for foreign deck")
	} else {
		testutil.RequireErrorCode(t, err, "invalid_reference")
	}

	if err := svc.DeleteDeck(ctx, principal, deck.ID); err != nil {
		t.Fatalf("delete deck: %v", err)
	}
	if _, err := svc.CreateArea(ctx, principal, created.ID, ship.AreaInput{
		DeckID: deck.ID, Name: "After Delete", AreaType: "restaurant",
	}); err == nil {
		t.Fatal("expected invalid reference for soft-deleted deck")
	} else {
		testutil.RequireErrorCode(t, err, "invalid_reference")
	}

	if _, err := svc.GetRoom(ctx, principal, room.ID); err != nil {
		t.Fatalf("room should remain readable after parent deck soft delete: %v", err)
	}
}
