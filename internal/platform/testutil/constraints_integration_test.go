//go:build integration

package testutil_test

import (
	"context"
	authrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/auth"
	bookingrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/booking"
	partnerrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/partner"
	shiprepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/ship"
	staffrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/staff"
	voyagerepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/voyage"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/booking"
	"github.com/ren0503/tycoon/internal/application/partner"
	"github.com/ren0503/tycoon/internal/application/ship"
	"github.com/ren0503/tycoon/internal/application/staff"
	"github.com/ren0503/tycoon/internal/application/voyage"
	"github.com/ren0503/tycoon/internal/platform/testutil"
)

func TestConstraintMatrixSurfacesMappedErrors(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	shipSvc := ship.New(shiprepo.New(h.Pool), testutil.Logger(t))
	baseShip := h.SeedShip(t, tenantID)
	deckID := h.SeedDeck(t, baseShip, 1)
	voyageSvc := voyage.New(voyagerepo.New(h.Pool), testutil.Logger(t), shipSvc)
	staffSvc := staff.New(staffrepo.New(h.Pool), testutil.Logger(t), shipSvc)
	partnerSvc := partner.New(partnerrepo.New(h.Pool), shipSvc)
	bookingSvc := booking.New(bookingrepo.New(h.Pool), shipSvc, voyageSvc, partnerSvc, authrepo.NewAccountService(h.Pool))

	departure := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	baseVoyage := h.SeedVoyage(t, baseShip, "MATRIX-1", departure)
	roomID := h.SeedRoom(t, baseShip, deckID, "M-1")
	partnerID := h.SeedPartner(t, tenantID)
	areaID := h.SeedArea(t, baseShip, deckID, "Matrix Area")
	h.SeedPartnerService(t, baseShip, partnerID)
	accountID := h.SeedAccount(t)

	type testCase struct {
		name string
		run  func() error
		code string
	}
	imo := "IMO1234567"
	cases := []testCase{
		{
			name: "invalid room type",
			run: func() error {
				_, err := shipSvc.CreateRoom(ctx, principal, baseShip, ship.RoomInput{
					DeckID: deckID, Number: "BAD-1", RoomType: "penthouse", Capacity: 2,
				})
				return err
			},
			code: "validation_failed",
		},
		{
			name: "invalid area type",
			run: func() error {
				_, err := shipSvc.CreateArea(ctx, principal, baseShip, ship.AreaInput{
					DeckID: deckID, Name: "Bad Area", AreaType: "casino",
				})
				return err
			},
			code: "validation_failed",
		},
		{
			name: "invalid staff department",
			run: func() error {
				_, err := staffSvc.CreateStaff(ctx, principal, staff.StaffInput{
					ShipID: &baseShip, EmployeeCode: "BAD-EMP", FullName: "Bad Dept", Department: "bridge",
				})
				return err
			},
			code: "validation_failed",
		},
		{
			name: "invalid port call sequence",
			run: func() error {
				port, err := voyageSvc.CreatePort(ctx, principal, voyage.PortInput{
					UnLocode: "ZZBAD", Name: "Bad Port", CountryCode: "ZZ",
				})
				if err != nil {
					return err
				}
				_, err = voyageSvc.CreatePortCall(ctx, principal, baseVoyage, voyage.PortCallInput{
					PortID: port.ID, Sequence: 0,
				})
				return err
			},
			code: "validation_failed",
		},
		{
			name: "unique ship IMO",
			run: func() error {
				if _, err := shipSvc.CreateShip(ctx, principal, ship.ShipInput{
					Name: "IMO One", ImoNumber: &imo, Timezone: "UTC",
				}); err != nil {
					return err
				}
				_, err := shipSvc.CreateShip(ctx, principal, ship.ShipInput{
					Name: "IMO Two", ImoNumber: &imo, Timezone: "UTC",
				})
				return err
			},
			code: "conflict",
		},
		{
			name: "unique voyage code per ship",
			run: func() error {
				_, err := voyageSvc.CreateVoyage(ctx, principal, voyage.VoyageInput{
					ShipID: baseShip, Code: "MATRIX-1", DepartureAt: departure.Add(2000 * time.Hour), ArrivalAt: departure.Add(2024 * time.Hour),
				})
				return err
			},
			code: "conflict",
		},
		{
			name: "non-partial port locode blocks reuse after soft delete",
			run: func() error {
				port, err := voyageSvc.CreatePort(ctx, principal, voyage.PortInput{
					UnLocode: "USNYC", Name: "New York", CountryCode: "US",
				})
				if err != nil {
					return err
				}
				if err := voyageSvc.DeletePort(ctx, principal, port.ID); err != nil {
					return err
				}
				_, err = voyageSvc.CreatePort(ctx, principal, voyage.PortInput{
					UnLocode: "USNYC", Name: "New York Again", CountryCode: "US",
				})
				return err
			},
			code: "conflict",
		},
		{
			name: "foreign key room deck",
			run: func() error {
				_, err := shipSvc.CreateRoom(ctx, principal, baseShip, ship.RoomInput{
					DeckID: uuid.New(), Number: "FK-1", RoomType: "inside", Capacity: 2,
				})
				return err
			},
			code: "invalid_reference",
		},
		{
			name: "foreign key port call port",
			run: func() error {
				_, err := voyageSvc.CreatePortCall(ctx, principal, baseVoyage, voyage.PortCallInput{
					PortID: uuid.New(), Sequence: 1,
				})
				return err
			},
			code: "invalid_reference",
		},
		{
			name: "voyage exclusion overlap",
			run: func() error {
				_, err := voyageSvc.CreateVoyage(ctx, principal, voyage.VoyageInput{
					ShipID: baseShip, Code: "MATRIX-2", DepartureAt: departure.Add(2 * time.Hour), ArrivalAt: departure.Add(26 * time.Hour),
				})
				return err
			},
			code: "overlap_conflict",
		},
		{
			name: "partner slot exclusion overlap",
			run: func() error {
				slotStart := departure.Add(500 * time.Hour)
				first, err := partnerSvc.CreateSlot(ctx, principal, partnerID, partner.SlotInput{
					ShipID: baseShip, AreaID: areaID, StartAt: slotStart, EndAt: slotStart.Add(2 * time.Hour),
				})
				if err != nil {
					return err
				}
				approved := "approved"
				if _, err := partnerSvc.UpdateSlot(ctx, principal, first.ID, partner.SlotUpdate{Status: &approved}); err != nil {
					return err
				}
				second, err := partnerSvc.CreateSlot(ctx, principal, partnerID, partner.SlotInput{
					ShipID: baseShip, AreaID: areaID, StartAt: slotStart.Add(time.Hour), EndAt: slotStart.Add(3 * time.Hour),
				})
				if err != nil {
					return err
				}
				_, err = partnerSvc.UpdateSlot(ctx, principal, second.ID, partner.SlotUpdate{Status: &approved})
				return err
			},
			code: "overlap_conflict",
		},
		{
			name: "booking room exclusion overlap",
			run: func() error {
				if _, err := bookingSvc.CreateBooking(ctx, principal, booking.BookingInput{
					ShipID: baseShip, VoyageID: baseVoyage, AccountID: &accountID, RoomID: &roomID,
					EmbarkSeq: 1, DisembarkSeq: 5, IdempotencyKey: "matrix-booking-1",
				}); err != nil {
					return err
				}
				_, err := bookingSvc.CreateBooking(ctx, principal, booking.BookingInput{
					ShipID: baseShip, VoyageID: baseVoyage, AccountID: &accountID, RoomID: &roomID,
					EmbarkSeq: 2, DisembarkSeq: 4, IdempotencyKey: "matrix-booking-2",
				})
				return err
			},
			code: "overlap_conflict",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatalf("expected mapped error %q", tc.code)
			}
			testutil.RequireErrorCode(t, err, tc.code)
		})
	}
}
