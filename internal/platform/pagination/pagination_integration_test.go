//go:build integration

package pagination_test

import (
	"context"
	authrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/auth"
	bookingrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/booking"
	shiprepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/ship"
	tenantrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/tenant"
	voyagerepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/voyage"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/booking"
	"github.com/ren0503/tycoon/internal/application/ship"
	"github.com/ren0503/tycoon/internal/application/tenant"
	"github.com/ren0503/tycoon/internal/application/voyage"
	"github.com/ren0503/tycoon/internal/platform/pagination"
	"github.com/ren0503/tycoon/internal/platform/testutil"
)

func TestCursorPaginationHasNoDuplicatesOrSkips(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const total = 150
	seeded := make(map[uuid.UUID]bool, total)
	for i := 0; i < total; i++ {
		id := uuid.New()
		seeded[id] = true
		if _, err := h.Pool.Exec(ctx, `INSERT INTO tenants (id, name) VALUES ($1, $2)`, id, "tenant-page"); err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
	}

	tenantSvc := tenant.New(tenantrepo.New(h.Pool))
	system := testutil.SystemPrincipal()
	tenantFilter := pagination.FilterHash("tenants", uuid.Nil.String())
	assertWalk(t, total, func(cursor string, limit int32) ([]uuid.UUID, string, error) {
		decoded, err := pagination.Decode(cursor, tenantFilter)
		if err != nil {
			return nil, "", err
		}
		page, err := tenantSvc.List(ctx, system, decoded, limit)
		ids := make([]uuid.UUID, len(page.Items))
		for i, row := range page.Items {
			ids[i] = row.ID
		}
		return ids, page.NextCursor, err
	}, seeded)

	principal := testutil.Principal(h.SeedTenant(t))
	shipSvc := ship.New(shiprepo.New(h.Pool), testutil.Logger(t))
	shipRow := h.SeedShip(t, principal.TenantID)
	voyageSvc := voyage.New(voyagerepo.New(h.Pool), testutil.Logger(t), shipSvc)
	departure := time.Now().UTC().Add(24 * time.Hour)
	voyageID := h.SeedVoyage(t, shipRow, "PAGE-1", departure)
	bookingIDs := make(map[uuid.UUID]bool, total)
	for i := 0; i < total; i++ {
		id := uuid.New()
		bookingIDs[id] = true
		if _, err := h.Pool.Exec(ctx, `
			INSERT INTO bookings (id, ship_id, voyage_id, embark_seq, disembark_seq)
			VALUES ($1, $2, $3, 1, 5)`, id, shipRow, voyageID); err != nil {
			t.Fatalf("seed booking: %v", err)
		}
	}
	bookingSvc := booking.New(bookingrepo.New(h.Pool), shipSvc, voyageSvc, nil, authrepo.NewAccountService(h.Pool))
	assertWalk(t, total, func(cursor string, limit int32) ([]uuid.UUID, string, error) {
		page, err := bookingSvc.ListBookings(ctx, principal, &shipRow, cursor, limit)
		ids := make([]uuid.UUID, len(page.Items))
		for i, row := range page.Items {
			ids[i] = row.ID
		}
		return ids, page.NextCursor, err
	}, bookingIDs)

	firstPage, err := tenantSvc.List(ctx, system, pagination.Cursor{}, 20)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if firstPage.NextCursor == "" {
		t.Fatal("expected a next cursor for the first page")
	}
	concurrentID := uuid.New()
	if _, err := h.Pool.Exec(ctx, `INSERT INTO tenants (id, name) VALUES ($1, 'tenant-concurrent')`, concurrentID); err != nil {
		t.Fatalf("concurrent insert: %v", err)
	}
	assertWalk(t, total+1, func(cursor string, limit int32) ([]uuid.UUID, string, error) {
		decoded, err := pagination.Decode(cursor, tenantFilter)
		if err != nil {
			return nil, "", err
		}
		page, err := tenantSvc.List(ctx, system, decoded, limit)
		ids := make([]uuid.UUID, len(page.Items))
		for i, row := range page.Items {
			ids[i] = row.ID
		}
		return ids, page.NextCursor, err
	}, nil)
}

func assertWalk(t *testing.T, expected int, fetch func(cursor string, limit int32) ([]uuid.UUID, string, error), mustInclude map[uuid.UUID]bool) {
	t.Helper()
	for _, limit := range []int32{20, 100} {
		seen := make(map[uuid.UUID]bool)
		cursor := ""
		for pages := 0; ; pages++ {
			ids, next, err := fetch(cursor, limit)
			if err != nil {
				t.Fatalf("limit %d: fetch cursor %q: %v", limit, cursor, err)
			}
			if len(ids) == 0 {
				t.Fatalf("limit %d: empty page before end", limit)
			}
			if int32(len(ids)) > limit {
				t.Fatalf("limit %d: page returned %d rows", limit, len(ids))
			}
			for _, id := range ids {
				if seen[id] {
					t.Fatalf("limit %d: duplicate id %s", limit, id)
				}
				seen[id] = true
			}
			if next == "" {
				break
			}
			cursor = next
			if pages > expected {
				t.Fatalf("limit %d: pagination did not terminate", limit)
			}
		}
		if len(seen) != expected {
			t.Fatalf("limit %d: walked %d rows, want %d", limit, len(seen), expected)
		}
		for id := range mustInclude {
			if !seen[id] {
				t.Fatalf("limit %d: skipped seeded id %s", limit, id)
			}
		}
	}
}
