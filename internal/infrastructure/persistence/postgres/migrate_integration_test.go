//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/platform/testutil"
)

func TestMigrationsCreateBusinessSchema(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var tableCount int
	if err := h.Pool.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		  AND table_name <> 'goose_db_version'`).Scan(&tableCount); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if tableCount != 41 {
		t.Fatalf("business table count = %d, want 41", tableCount)
	}

	for _, constraint := range []string{"ex_voyages_no_overlap", "ex_partner_slots_no_overlap", "ex_bookings_room_no_overlap"} {
		var found bool
		if err := h.Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = $1 AND contype = 'x'
			)`, constraint).Scan(&found); err != nil {
			t.Fatalf("check exclusion constraint %s: %v", constraint, err)
		}
		if !found {
			t.Fatalf("exclusion constraint %s is missing", constraint)
		}
	}

	for _, function := range []string{"capture_change", "touch_row"} {
		var found bool
		if err := h.Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_proc WHERE proname = $1
			)`, function).Scan(&found); err != nil {
			t.Fatalf("check function %s: %v", function, err)
		}
		if !found {
			t.Fatalf("function %s is missing", function)
		}
	}
}

func TestMigrationsRerunIsHarmless(t *testing.T) {
	h := testutil.StartPostgres(t)

	result, err := testutil.MigrateUp(t, h.DSN)
	if err != nil {
		t.Fatalf("re-run migrations: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected no pending migrations on re-run, applied %d", len(result))
	}
}
