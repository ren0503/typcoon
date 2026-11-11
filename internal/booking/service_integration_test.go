//go:build integration

package booking_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/booking"
	"github.com/ren0503/tycoon/internal/partner"
	"github.com/ren0503/tycoon/internal/ship"
	"github.com/ren0503/tycoon/internal/testutil"
	"github.com/ren0503/tycoon/internal/voyage"
)

func TestBookingIdempotencyOverlapAndImmutability(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	shipSvc := ship.New(h.Pool, testutil.Logger(t), nil)
	shipRow, err := shipSvc.CreateShip(ctx, principal, ship.ShipInput{Name: "Booking Ship", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}
	voyageSvc := voyage.New(h.Pool, testutil.Logger(t), shipSvc)
	departure := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	v, err := voyageSvc.CreateVoyage(ctx, principal, voyage.VoyageInput{
		ShipID: shipRow.ID, Code: "BOOK-1", DepartureAt: departure, ArrivalAt: departure.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create voyage: %v", err)
	}
	deck, err := shipSvc.CreateDeck(ctx, principal, shipRow.ID, ship.DeckInput{Level: 1, Name: "Deck"})
	if err != nil {
		t.Fatalf("create deck: %v", err)
	}
	room, err := shipSvc.CreateRoom(ctx, principal, shipRow.ID, ship.RoomInput{
		DeckID: deck.ID, Number: "101", RoomType: "inside", Capacity: 2, IsBookable: true,
	})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	partnerSvc := partner.New(h.Pool, shipSvc)
	p, err := partnerSvc.CreatePartner(ctx, principal, partner.PartnerInput{Name: "Partner", BusinessType: "spa"})
	if err != nil {
		t.Fatalf("create partner: %v", err)
	}
	partnerServiceID := h.SeedPartnerService(t, shipRow.ID, p.ID)
	accountID := h.SeedAccount(t)
	accountSvc := auth.NewAccountService(h.Pool)

	svc := booking.New(h.Pool, shipSvc, voyageSvc, partnerSvc, accountSvc, nil)

	input := booking.BookingInput{
		ShipID: shipRow.ID, VoyageID: v.ID, AccountID: &accountID, RoomID: &room.ID,
		EmbarkSeq: 1, DisembarkSeq: 5, IdempotencyKey: "idem-booking-1",
	}
	first, err := svc.CreateBooking(ctx, principal, input)
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}
	replay, err := svc.CreateBooking(ctx, principal, input)
	if err != nil {
		t.Fatalf("replay booking: %v", err)
	}
	if replay.ID != first.ID {
		t.Fatalf("idempotent replay created a new booking %s, want %s", replay.ID, first.ID)
	}

	changed := input
	changed.DisembarkSeq = 6
	if _, err := svc.CreateBooking(ctx, principal, changed); err == nil {
		t.Fatal("expected idempotency conflict for changed payload")
	} else {
		testutil.RequireErrorCode(t, err, "idempotency_conflict")
	}

	overlapping := booking.BookingInput{
		ShipID: shipRow.ID, VoyageID: v.ID, AccountID: &accountID, RoomID: &room.ID,
		EmbarkSeq: 2, DisembarkSeq: 4, IdempotencyKey: "idem-booking-2",
	}
	if _, err := svc.CreateBooking(ctx, principal, overlapping); err == nil {
		t.Fatal("expected room overlap conflict")
	} else {
		testutil.RequireErrorCode(t, err, "overlap_conflict")
	}

	request := "quiet room"
	updated, err := svc.UpdateBooking(ctx, principal, first.ID, booking.BookingUpdate{SpecialRequests: &request})
	if err != nil {
		t.Fatalf("update booking: %v", err)
	}
	if updated.Status != first.Status {
		t.Fatalf("booking status changed from %q to %q", first.Status, updated.Status)
	}

	serviceInput := booking.ServiceBookingInput{
		BookingID: first.ID, PartnerServiceID: partnerServiceID, Quantity: 1, IdempotencyKey: "idem-service-1",
	}
	serviceFirst, err := svc.CreateServiceBooking(ctx, principal, serviceInput)
	if err != nil {
		t.Fatalf("create service booking: %v", err)
	}
	serviceReplay, err := svc.CreateServiceBooking(ctx, principal, serviceInput)
	if err != nil {
		t.Fatalf("replay service booking: %v", err)
	}
	if serviceReplay.ID != serviceFirst.ID {
		t.Fatalf("service booking replay created a new row")
	}
	serviceChanged := serviceInput
	serviceChanged.Quantity = 2
	if _, err := svc.CreateServiceBooking(ctx, principal, serviceChanged); err == nil {
		t.Fatal("expected service booking idempotency conflict")
	} else {
		testutil.RequireErrorCode(t, err, "idempotency_conflict")
	}

	var bookingCount, serviceCount int
	if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM bookings`).Scan(&bookingCount); err != nil {
		t.Fatalf("count bookings: %v", err)
	}
	if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM service_bookings`).Scan(&serviceCount); err != nil {
		t.Fatalf("count service bookings: %v", err)
	}
	if _, err := svc.CreateServiceBooking(ctx, principal, booking.ServiceBookingInput{
		BookingID: first.ID, PartnerServiceID: partnerServiceID, Quantity: 1, IdempotencyKey: "idem-service-fail",
	}); err != nil {
		t.Fatalf("expected initial create to succeed: %v", err)
	}
	if _, err := svc.CreateServiceBooking(ctx, principal, booking.ServiceBookingInput{
		BookingID: first.ID, PartnerServiceID: uuid.Nil, Quantity: 1, IdempotencyKey: "idem-service-rollback",
	}); err == nil {
		t.Fatal("expected invalid partner service reference")
	}
	var serviceAfter int
	if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM service_bookings`).Scan(&serviceAfter); err != nil {
		t.Fatalf("count service bookings after failure: %v", err)
	}
	if serviceAfter != serviceCount+1 {
		t.Fatalf("failed create left partial rows: before=%d after=%d", serviceCount, serviceAfter)
	}

	var auditRows int
	if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_log`).Scan(&auditRows); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if auditRows != 0 {
		t.Fatalf("audit rows = %d, want 0 when no appender is configured", auditRows)
	}
}
