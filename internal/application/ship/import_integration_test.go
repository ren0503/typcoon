//go:build integration

package ship_test

import (
	"context"
	"errors"
	shiprepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/ship"
	syncrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sync"
	"testing"
	"time"

	appaudit "github.com/ren0503/tycoon/internal/application/audit"
	"github.com/ren0503/tycoon/internal/application/ship"
	syncservice "github.com/ren0503/tycoon/internal/application/sync"
	"github.com/ren0503/tycoon/internal/platform/testutil"
)

type failingAuditRepo struct{ ship.Repository }

func (r failingAuditRepo) Begin(ctx context.Context) (ship.Tx, error) {
	tx, err := r.Repository.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return failingAuditTx{tx}, nil
}

type failingAuditTx struct{ ship.Tx }

func (failingAuditTx) AppendAudit(context.Context, appaudit.Entry) error {
	return errors.New("audit unavailable")
}

func TestStructureImportLifecycleAndIdempotency(t *testing.T) {
	h := testutil.StartPostgres(t)
	t.Cleanup(func() { h.TruncateAll(t) })
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	svc := ship.New(shiprepo.New(h.Pool), testutil.Logger(t))

	created, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "Imported", Timezone: "UTC", Status: "maintenance"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}
	if created.Status != "maintenance" {
		t.Fatalf("status = %q, want maintenance", created.Status)
	}
	template, err := svc.StructureImportTemplate(ctx)
	if err != nil {
		t.Fatalf("template: %v", err)
	}

	preview, err := svc.UploadStructureImport(ctx, principal, created.ID, "template.xlsx", template, "upload-1")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if preview.Import.Status != "ready" || !preview.Import.CanConfirm {
		t.Fatalf("unexpected import status: %+v", preview.Import)
	}
	if preview.Import.Counts.Decks.Total != 2 || preview.Import.Counts.Areas.Total != 2 || preview.Import.Counts.Rooms.Total != 2 {
		t.Fatalf("unexpected counts: %+v", preview.Import.Counts)
	}
	if len(preview.Records.Decks) != 2 || len(preview.Records.Areas) != 2 || len(preview.Records.Rooms) != 2 {
		t.Fatalf("unexpected records: %+v", preview.Records)
	}
	assertCount(t, h, `SELECT count(*) FROM decks WHERE ship_id = $1`, created.ID, 0)
	assertCount(t, h, `SELECT count(*) FROM areas WHERE ship_id = $1`, created.ID, 0)
	assertCount(t, h, `SELECT count(*) FROM rooms WHERE ship_id = $1`, created.ID, 0)

	replay, err := svc.UploadStructureImport(ctx, principal, created.ID, "template.xlsx", template, "upload-1")
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if replay.Import.ID != preview.Import.ID {
		t.Fatalf("replay import id = %s, want %s", replay.Import.ID, preview.Import.ID)
	}
	if _, err := svc.UploadStructureImport(ctx, principal, created.ID, "template.xlsx", []byte("different"), "upload-1"); err == nil {
		t.Fatal("expected idempotency conflict")
	} else {
		testutil.RequireErrorCode(t, err, "idempotency_conflict")
	}

	confirmed, err := svc.ConfirmStructureImport(ctx, principal, created.ID, preview.Import.ID)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.Status != "completed" {
		t.Fatalf("confirmed status = %q", confirmed.Status)
	}
	if _, err := svc.ConfirmStructureImport(ctx, principal, created.ID, preview.Import.ID); err != nil {
		t.Fatalf("repeat confirm: %v", err)
	}

	assertCount(t, h, `SELECT count(*) FROM decks WHERE ship_id = $1`, created.ID, 2)
	assertCount(t, h, `SELECT count(*) FROM areas WHERE ship_id = $1`, created.ID, 2)
	assertCount(t, h, `SELECT count(*) FROM rooms WHERE ship_id = $1`, created.ID, 2)
	assertCount(t, h, `SELECT count(*) FROM change_log WHERE ship_id = $1`, created.ID, 7)
	assertCount(t, h, `SELECT count(*) FROM audit_log WHERE action = 'ship.structure_import.confirmed'`, 1)

	syncSvc := syncservice.New(syncrepo.New(h.Pool), svc, nil)
	changes, err := syncSvc.ChangeLog(ctx, principal, created.ID, 0, 100)
	if err != nil {
		t.Fatalf("change log: %v", err)
	}
	entityCounts := map[string]int{}
	var lastSeq int64
	for _, change := range changes {
		if change.Seq <= lastSeq {
			t.Fatalf("change log seq is not increasing: %d after %d", change.Seq, lastSeq)
		}
		lastSeq = change.Seq
		if change.Op != "upsert" {
			t.Fatalf("change log op = %q, want upsert", change.Op)
		}
		entityCounts[change.Entity]++
	}
	if entityCounts["decks"] != 2 || entityCounts["areas"] != 2 || entityCounts["rooms"] != 2 {
		t.Fatalf("change log entity counts = %#v", entityCounts)
	}

	decks, err := svc.ListDecks(ctx, principal, created.ID, "", 20)
	if err != nil || len(decks.Items) == 0 {
		t.Fatalf("list imported decks: %v", err)
	}
	name := "Renamed Deck"
	if _, err := svc.UpdateDeck(ctx, principal, decks.Items[0].ID, ship.DeckUpdate{Name: &name}); err != nil {
		t.Fatalf("update imported deck: %v", err)
	}
	rooms, err := svc.ListRooms(ctx, principal, created.ID, "", 20)
	if err != nil || len(rooms.Items) == 0 {
		t.Fatalf("list imported rooms: %v", err)
	}
	if err := svc.DeleteRoom(ctx, principal, rooms.Items[0].ID); err != nil {
		t.Fatalf("delete imported room: %v", err)
	}
	changes, err = syncSvc.ChangeLog(ctx, principal, created.ID, lastSeq, 100)
	if err != nil {
		t.Fatalf("change log after update: %v", err)
	}
	var sawDeckUpdate, sawRoomDelete bool
	for _, change := range changes {
		if change.Entity == "decks" && change.Op == "upsert" && change.Version > 1 {
			sawDeckUpdate = true
		}
		if change.Entity == "rooms" && change.Op == "delete" && change.Version > 1 {
			sawRoomDelete = true
		}
	}
	if !sawDeckUpdate || !sawRoomDelete {
		t.Fatalf("post-import changes = %#v", changes)
	}

	conflict, err := svc.UploadStructureImport(ctx, principal, created.ID, "template.xlsx", template, "upload-2")
	if err != nil {
		t.Fatalf("conflicting upload: %v", err)
	}
	if conflict.Import.Status != "failed" || conflict.Import.CanConfirm {
		t.Fatalf("conflict upload should fail validation: %+v", conflict.Import)
	}
	if _, err := svc.ConfirmStructureImport(ctx, principal, created.ID, conflict.Import.ID); err == nil {
		t.Fatal("expected non-confirmable error")
	} else {
		testutil.RequireErrorCode(t, err, "conflict")
	}

	otherTenant := h.SeedTenant(t)
	otherPrincipal := testutil.Principal(otherTenant)
	if _, err := svc.GetStructureImport(ctx, otherPrincipal, created.ID, preview.Import.ID); err == nil {
		t.Fatal("expected cross-tenant not found")
	} else {
		testutil.RequireErrorCode(t, err, "not_found")
	}
}

func TestStructureImportConfirmationRollsBackAuditFailure(t *testing.T) {
	h := testutil.StartPostgres(t)
	t.Cleanup(func() { h.TruncateAll(t) })
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	svc := ship.New(failingAuditRepo{shiprepo.New(h.Pool)}, testutil.Logger(t))

	created, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "Rollback", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}
	template, err := svc.StructureImportTemplate(ctx)
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	preview, err := svc.UploadStructureImport(ctx, principal, created.ID, "template.xlsx", template, "")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	if _, err := svc.ConfirmStructureImport(ctx, principal, created.ID, preview.Import.ID); err == nil {
		t.Fatal("expected confirmation failure")
	} else {
		testutil.RequireErrorCode(t, err, "internal_error")
	}
	assertCount(t, h, `SELECT count(*) FROM decks WHERE ship_id = $1`, created.ID, 0)
	row, err := svc.GetStructureImport(ctx, principal, created.ID, preview.Import.ID)
	if err != nil {
		t.Fatalf("get after rollback: %v", err)
	}
	if row.Status != "ready" || row.ErrorSummary == nil {
		t.Fatalf("import after rollback = %+v", row)
	}

	retry := ship.New(shiprepo.New(h.Pool), testutil.Logger(t))
	if _, err := retry.ConfirmStructureImport(ctx, principal, created.ID, preview.Import.ID); err != nil {
		t.Fatalf("retry confirm: %v", err)
	}
}

func assertCount(t *testing.T, h *testutil.Harness, query string, args ...any) {
	t.Helper()
	var count int
	if err := h.Pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
}

func TestStructureImportStaleRecoveryAndStatus(t *testing.T) {
	h := testutil.StartPostgres(t)
	t.Cleanup(func() { h.TruncateAll(t) })
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	svc := ship.New(shiprepo.New(h.Pool), testutil.Logger(t))

	created, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "Stale", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}
	template, err := svc.StructureImportTemplate(ctx)
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	preview, err := svc.UploadStructureImport(ctx, principal, created.ID, "template.xlsx", template, "")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if preview.Import.Status != "ready" {
		t.Fatalf("status = %q, want ready", preview.Import.Status)
	}
	if _, err := h.Pool.Exec(ctx, `UPDATE ship_structure_imports SET status = 'validating', updated_at = now() - interval '11 minutes' WHERE id = $1`, preview.Import.ID); err != nil {
		t.Fatalf("mark stale: %v", err)
	}
	recovered, err := svc.GetStructureImport(ctx, principal, created.ID, preview.Import.ID)
	if err != nil {
		t.Fatalf("get stale import: %v", err)
	}
	if recovered.Status != "failed" || recovered.CanConfirm || recovered.ErrorSummary == nil {
		t.Fatalf("recovered import = %+v", recovered)
	}
	if _, err := svc.ConfirmStructureImport(ctx, principal, created.ID, preview.Import.ID); err == nil {
		t.Fatal("expected failed import to be non-confirmable")
	} else {
		testutil.RequireErrorCode(t, err, "conflict")
	}
	assertCount(t, h, `SELECT count(*) FROM decks WHERE ship_id = $1`, created.ID, 0)

	preview2, err := svc.PreviewStructureImport(ctx, principal, created.ID, preview.Import.ID)
	if err != nil {
		t.Fatalf("preview failed history: %v", err)
	}
	if preview2.Import.Status != "failed" {
		t.Fatalf("preview status = %q", preview2.Import.Status)
	}
}

func TestStructureImportConcurrentConfirmation(t *testing.T) {
	h := testutil.StartPostgres(t)
	t.Cleanup(func() { h.TruncateAll(t) })
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	svc := ship.New(shiprepo.New(h.Pool), testutil.Logger(t))

	created, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "Concurrent", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}
	template, err := svc.StructureImportTemplate(ctx)
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	preview, err := svc.UploadStructureImport(ctx, principal, created.ID, "template.xlsx", template, "")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	errs := make(chan error, 2)
	for range 2 {
		go func() {
			row, confirmErr := svc.ConfirmStructureImport(ctx, principal, created.ID, preview.Import.ID)
			if confirmErr == nil && row.Status != "completed" {
				confirmErr = errors.New("confirmation did not complete")
			}
			errs <- confirmErr
		}()
	}
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent confirm: %v", err)
		}
	}
	assertCount(t, h, `SELECT count(*) FROM decks WHERE ship_id = $1`, created.ID, 2)
	assertCount(t, h, `SELECT count(*) FROM areas WHERE ship_id = $1`, created.ID, 2)
	assertCount(t, h, `SELECT count(*) FROM rooms WHERE ship_id = $1`, created.ID, 2)
}

func TestStructureImportConfirmationRechecksConflicts(t *testing.T) {
	h := testutil.StartPostgres(t)
	t.Cleanup(func() { h.TruncateAll(t) })
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	svc := ship.New(shiprepo.New(h.Pool), testutil.Logger(t))

	created, err := svc.CreateShip(ctx, principal, ship.ShipInput{Name: "Recheck", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}
	template, err := svc.StructureImportTemplate(ctx)
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	preview, err := svc.UploadStructureImport(ctx, principal, created.ID, "template.xlsx", template, "")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	conflicting, err := svc.CreateDeck(ctx, principal, created.ID, ship.DeckInput{Level: 1, Name: "Added after validation"})
	if err != nil {
		t.Fatalf("create conflicting deck: %v", err)
	}
	if _, err := svc.ConfirmStructureImport(ctx, principal, created.ID, preview.Import.ID); err == nil {
		t.Fatal("expected conflict re-check failure")
	} else {
		testutil.RequireErrorCode(t, err, "conflict")
	}
	assertCount(t, h, `SELECT count(*) FROM areas WHERE ship_id = $1`, created.ID, 0)
	assertCount(t, h, `SELECT count(*) FROM rooms WHERE ship_id = $1`, created.ID, 0)
	row, err := svc.GetStructureImport(ctx, principal, created.ID, preview.Import.ID)
	if err != nil {
		t.Fatalf("get import after conflict: %v", err)
	}
	if row.Status != "ready" || row.ErrorSummary == nil {
		t.Fatalf("import after conflict = %+v", row)
	}
	if err := svc.DeleteDeck(ctx, principal, conflicting.ID); err != nil {
		t.Fatalf("delete conflicting deck: %v", err)
	}
	if _, err := svc.ConfirmStructureImport(ctx, principal, created.ID, preview.Import.ID); err != nil {
		t.Fatalf("retry confirm: %v", err)
	}
	assertCount(t, h, `SELECT count(*) FROM decks WHERE ship_id = $1`, created.ID, 2)
}
