//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/platform/testutil"
)

func TestChangeLogAndTouchTriggers(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := uuid.New()
	if _, err := h.Pool.Exec(ctx, `INSERT INTO tenants (id, name) VALUES ($1, 'trigger-tenant')`, tenantID); err != nil {
		t.Fatalf("insert tenant: %v", err)
	}
	shipID := uuid.New()
	if _, err := h.Pool.Exec(ctx, `
		INSERT INTO ships (id, tenant_id, name, imo_number, timezone)
		VALUES ($1, $2, 'Trigger Ship', $3, 'UTC')`, shipID, tenantID, "IMO"+shipID.String()[:9]); err != nil {
		t.Fatalf("insert ship: %v", err)
	}

	var version int64
	var insertedAt time.Time
	if err := h.Pool.QueryRow(ctx, `SELECT version, updated_at FROM ships WHERE id = $1`, shipID).Scan(&version, &insertedAt); err != nil {
		t.Fatalf("read ship: %v", err)
	}
	if version != 1 {
		t.Fatalf("insert version = %d, want 1", version)
	}

	time.Sleep(10 * time.Millisecond)
	if _, err := h.Pool.Exec(ctx, `UPDATE ships SET name = 'Renamed Ship' WHERE id = $1`, shipID); err != nil {
		t.Fatalf("update ship: %v", err)
	}

	var updatedVersion int64
	var updatedAt time.Time
	if err := h.Pool.QueryRow(ctx, `SELECT version, updated_at FROM ships WHERE id = $1`, shipID).Scan(&updatedVersion, &updatedAt); err != nil {
		t.Fatalf("read updated ship: %v", err)
	}
	if updatedVersion != 2 {
		t.Fatalf("update version = %d, want 2 (trigger-maintained)", updatedVersion)
	}
	if !updatedAt.After(insertedAt) {
		t.Fatalf("updated_at was not advanced by the trigger")
	}

	var lastSeq int64
	if err := h.Pool.QueryRow(ctx, `SELECT last_seq FROM ship_seq WHERE ship_id = $1`, shipID).Scan(&lastSeq); err != nil {
		t.Fatalf("read ship_seq: %v", err)
	}
	if lastSeq != 2 {
		t.Fatalf("ship_seq.last_seq = %d, want 2", lastSeq)
	}

	rows, err := h.Pool.Query(ctx, `
		SELECT seq, entity, version FROM change_log WHERE ship_id = $1 ORDER BY seq`, shipID)
	if err != nil {
		t.Fatalf("read change_log: %v", err)
	}
	defer rows.Close()
	var seqs []int64
	for rows.Next() {
		var seq, rowVersion int64
		var entity string
		if err := rows.Scan(&seq, &entity, &rowVersion); err != nil {
			t.Fatalf("scan change_log: %v", err)
		}
		if entity != "ships" {
			t.Fatalf("change_log entity = %q, want ships", entity)
		}
		if rowVersion != seq {
			t.Fatalf("change_log version = %d, want matching seq %d", rowVersion, seq)
		}
		seqs = append(seqs, seq)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("change_log rows: %v", err)
	}
	if len(seqs) != 2 || seqs[0] != 1 || seqs[1] != 2 {
		t.Fatalf("change_log seqs = %v, want [1 2]", seqs)
	}
}
