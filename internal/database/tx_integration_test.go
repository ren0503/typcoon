//go:build integration

package database_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/audit"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/booking"
	"github.com/ren0503/tycoon/internal/ship"
	"github.com/ren0503/tycoon/internal/testutil"
	"github.com/ren0503/tycoon/internal/voyage"
)

type failingAppender struct{}

func (failingAppender) AppendTx(context.Context, audit.Execer, audit.Entry) error {
	return errors.New("forced audit failure")
}

func TestFailedMultiWriteRollsBackAndLeavesNoChangeLog(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	shipID := h.SeedShip(t, tenantID)
	departure := time.Now().UTC().Add(48 * time.Hour)
	voyageID := h.SeedVoyage(t, shipID, "TX-1", departure)
	accountID := h.SeedAccount(t)

	shipSvc := ship.New(h.Pool, testutil.Logger(t), nil)
	voyageSvc := voyage.New(h.Pool, testutil.Logger(t), shipSvc)
	accountSvc := auth.NewAccountService(h.Pool)
	input := booking.BookingInput{
		ShipID: shipID, VoyageID: voyageID, AccountID: &accountID,
		EmbarkSeq: 1, DisembarkSeq: 5, IdempotencyKey: "tx-rollback",
	}

	failing := booking.New(h.Pool, shipSvc, voyageSvc, nil, accountSvc, failingAppender{})
	if _, err := failing.CreateBooking(ctx, principal, input); err == nil {
		t.Fatal("expected forced audit failure")
	}

	assertCount(t, h, `SELECT count(*) FROM bookings`, 0)
	assertCount(t, h, `SELECT count(*) FROM audit_log`, 0)
	assertCount(t, h, `SELECT count(*) FROM change_log WHERE ship_id = $1`, 0, shipID)

	success := booking.New(h.Pool, shipSvc, voyageSvc, nil, accountSvc, audit.NewStore(h.Pool, testutil.Logger(t)))
	row, err := success.CreateBooking(ctx, principal, input)
	if err != nil {
		t.Fatalf("create booking with audit: %v", err)
	}
	if row.ID.String() == "" {
		t.Fatal("created booking has no id")
	}
	assertCount(t, h, `SELECT count(*) FROM bookings`, 1)
	assertCount(t, h, `SELECT count(*) FROM audit_log WHERE entity_id = $1`, 1, row.ID)
	assertCount(t, h, `SELECT count(*) FROM change_log WHERE ship_id = $1 AND entity_id = $2`, 1, shipID, row.ID)
}

func assertCount(t *testing.T, h *testutil.Harness, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := h.Pool.QueryRow(context.Background(), query, args...).Scan(&got); err != nil {
		t.Fatalf("count query %q: %v", query, err)
	}
	if got != want {
		t.Fatalf("count query %q = %d, want %d", query, got, want)
	}
}
