//go:build integration

package guest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/audit"
	"github.com/ren0503/tycoon/internal/booking"
	"github.com/ren0503/tycoon/internal/guest"
	"github.com/ren0503/tycoon/internal/platform/testutil"
	"github.com/ren0503/tycoon/internal/ship"
)

type failingAppender struct{}

func (failingAppender) AppendTx(context.Context, audit.Execer, audit.Entry) error {
	return errors.New("forced audit failure")
}

func TestGuestCiphertextSecrecyAndAudit(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	shipID := h.SeedShip(t, tenantID)
	departure := time.Now().UTC().Add(24 * time.Hour)
	voyageID := h.SeedVoyage(t, shipID, "GUEST-1", departure)
	bookingID := h.SeedBooking(t, shipID, voyageID)

	bookingSvc := booking.New(h.Pool, ship.New(h.Pool, testutil.Logger(t), nil), nil, nil, auth.NewAccountService(h.Pool), nil)
	auditStore := audit.NewStore(h.Pool, testutil.Logger(t))
	cipher, err := guest.NewDevDocumentCipher()
	if err != nil {
		t.Fatalf("build cipher: %v", err)
	}
	svc := guest.New(h.Pool, bookingSvc, auditStore, cipher)

	document, keyID, err := cipher.Encrypt([]byte("passport-secret"))
	if err != nil {
		t.Fatalf("encrypt document: %v", err)
	}
	emergency, _, err := cipher.Encrypt([]byte("emergency-secret"))
	if err != nil {
		t.Fatalf("encrypt emergency contact: %v", err)
	}

	created, err := svc.Create(ctx, principal, bookingID, guest.Input{
		FullName: "Guest One", DocumentEnc: document, DocumentKeyID: &keyID, EmergencyContactEnc: emergency,
	})
	if err != nil {
		t.Fatalf("create guest: %v", err)
	}
	if !created.HasDocument || !created.HasEmergencyContact {
		t.Fatalf("guest view did not report opaque document presence")
	}

	var auditCount int
	if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE entity = 'guest' AND entity_id = $1`, created.ID.String()).Scan(&auditCount); err != nil {
		t.Fatalf("count audit entries: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("audit entries = %d, want 1 written in the same transaction", auditCount)
	}

	if _, err := svc.Create(ctx, principal, bookingID, guest.Input{
		FullName: "Plaintext Guest", DocumentEnc: []byte("plaintext-passport"),
	}); err == nil {
		t.Fatal("expected plaintext document rejection")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}

	var plaintextRows int
	if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM guests WHERE full_name = 'Plaintext Guest'`).Scan(&plaintextRows); err != nil {
		t.Fatalf("count plaintext guests: %v", err)
	}
	if plaintextRows != 0 {
		t.Fatalf("plaintext guest was persisted")
	}

	expires := time.Now().Add(time.Hour)
	pass, err := svc.IssueBoardingPass(ctx, principal, bookingID, &created.ID, &expires)
	if err != nil {
		t.Fatalf("issue boarding pass: %v", err)
	}
	listed, err := svc.ListBoardingPasses(ctx, principal, bookingID, "", 20)
	if err != nil {
		t.Fatalf("list boarding passes: %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].ID != pass.ID {
		t.Fatalf("boarding pass list = %+v", listed.Items)
	}
	revoked, err := svc.RevokeBoardingPass(ctx, principal, pass.ID)
	if err != nil {
		t.Fatalf("revoke boarding pass: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("revoked_at was not set")
	}
	again, err := svc.RevokeBoardingPass(ctx, principal, pass.ID)
	if err != nil {
		t.Fatalf("idempotent revoke: %v", err)
	}
	if again.RevokedAt == nil {
		t.Fatal("idempotent revoke returned no revoked_at")
	}

	validDocument, keyID2, err := cipher.Encrypt([]byte("second-passport"))
	if err != nil {
		t.Fatalf("encrypt second document: %v", err)
	}
	failing := guest.New(h.Pool, bookingSvc, failingAppender{}, cipher)
	if _, err := failing.Create(ctx, principal, bookingID, guest.Input{
		FullName: "No Audit Guest", DocumentEnc: validDocument, DocumentKeyID: &keyID2,
	}); err == nil {
		t.Fatal("expected forced audit failure")
	}
	var noAuditRows int
	if err := h.Pool.QueryRow(ctx, `SELECT count(*) FROM guests WHERE full_name = 'No Audit Guest'`).Scan(&noAuditRows); err != nil {
		t.Fatalf("count no-audit guests: %v", err)
	}
	if noAuditRows != 0 {
		t.Fatalf("guest row persisted without an audit entry")
	}
}
