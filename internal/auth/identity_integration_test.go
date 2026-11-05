//go:build integration

package auth_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/testutil"
)

func countIdentities(t *testing.T, h *testutil.Harness) int {
	t.Helper()
	var count int
	if err := h.Pool.QueryRow(context.Background(), `SELECT count(*) FROM account_identities`).Scan(&count); err != nil {
		t.Fatalf("count identities: %v", err)
	}
	return count
}

func identityAccountForUID(t *testing.T, h *testutil.Harness, uid string) uuid.UUID {
	t.Helper()
	var accountID uuid.UUID
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT account_id FROM account_identities WHERE provider = 'firebase' AND provider_sub = $1`, uid).
		Scan(&accountID); err != nil {
		t.Fatalf("read identity account for %s: %v", uid, err)
	}
	return accountID
}

func identityIDForUID(t *testing.T, h *testutil.Harness, uid string) uuid.UUID {
	t.Helper()
	var identityID uuid.UUID
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT id FROM account_identities WHERE provider = 'firebase' AND provider_sub = $1`, uid).
		Scan(&identityID); err != nil {
		t.Fatalf("read identity id for %s: %v", uid, err)
	}
	return identityID
}

func TestIdentityEmailCollisionDoesNotMergeAndExplicitLinkKeepsAccount(t *testing.T) {
	h := testutil.StartPostgres(t)
	existingAccountID := seedLinkedAccount(t, h, "identity-existing")
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{
			"existing-token": {UID: "identity-existing", Email: "identity-existing@example.com", EmailVerified: true},
			"new-token":      {UID: "identity-new", Email: "identity-existing@example.com", EmailVerified: true},
		},
	})

	if _, err := exchange(t, f, "new-token"); !errors.Is(err, auth.ErrIdentityLinkRequired) {
		t.Fatalf("collision exchange error = %v, want identity link required", err)
	}
	assertAuthRowCounts(t, h, 1, 1, 0)

	session, err := exchange(t, f, "existing-token")
	if err != nil {
		t.Fatalf("existing identity exchange: %v", err)
	}
	if session.AccountID != existingAccountID {
		t.Fatalf("existing exchange account = %s, want %s", session.AccountID, existingAccountID)
	}

	linked, err := f.service.LinkIdentity(context.Background(), existingAccountID, "new-token")
	if err != nil {
		t.Fatalf("link new identity: %v", err)
	}
	if linked.AccountID != existingAccountID || linked.Provider != "firebase" || linked.Email == nil ||
		*linked.Email != "identity-existing@example.com" {
		t.Fatalf("linked identity = %+v", linked)
	}
	if countIdentities(t, h) != 2 {
		t.Fatalf("identity count after link = %d, want 2", countIdentities(t, h))
	}
	if got := identityAccountForUID(t, h, "identity-new"); got != existingAccountID {
		t.Fatalf("new identity linked to %s, want %s", got, existingAccountID)
	}
	assertAuditAction(t, f, auth.ActionIdentityLinked)

	again, err := f.service.LinkIdentity(context.Background(), existingAccountID, "new-token")
	if err != nil {
		t.Fatalf("idempotent link: %v", err)
	}
	if again.ID != linked.ID || countIdentities(t, h) != 2 {
		t.Fatalf("idempotent link changed identity: first=%s second=%s count=%d", linked.ID, again.ID, countIdentities(t, h))
	}

	newSession, err := exchange(t, f, "new-token")
	if err != nil {
		t.Fatalf("exchange freshly linked identity: %v", err)
	}
	if newSession.AccountID != existingAccountID {
		t.Fatalf("freshly linked identity resolved to %s, want %s", newSession.AccountID, existingAccountID)
	}
}

func TestDuplicateIdentityLinkConflictDoesNotRepointOwnership(t *testing.T) {
	h := testutil.StartPostgres(t)
	firstAccountID := seedLinkedAccount(t, h, "identity-first")
	secondAccountID := seedLinkedAccount(t, h, "identity-second")
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{
			"first-token": {UID: "identity-first", EmailVerified: true},
		},
	})

	if _, err := f.service.LinkIdentity(context.Background(), secondAccountID, "first-token"); !errors.Is(err, auth.ErrIdentityConflict) {
		t.Fatalf("duplicate identity link error = %v", err)
	}
	if got := identityAccountForUID(t, h, "identity-first"); got != firstAccountID {
		t.Fatalf("duplicate identity was repointed to %s, want %s", got, firstAccountID)
	}
	if countIdentities(t, h) != 2 {
		t.Fatalf("identity count = %d, want 2", countIdentities(t, h))
	}
	if got := countAuditAction(f, auth.ActionIdentityLinkDenied); got != 1 {
		t.Fatalf("link denied audit entries = %d, want 1", got)
	}
}

func TestUnlinkEnforcesOwnershipAndLastCredential(t *testing.T) {
	h := testutil.StartPostgres(t)
	accountID := seedLinkedAccount(t, h, "identity-unlink")
	seedLinkedAccount(t, h, "identity-other")
	firstIdentityID := identityIDForUID(t, h, "identity-unlink")
	otherIdentityID := identityIDForUID(t, h, "identity-other")
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{
			"second-token": {UID: "identity-unlink-second", EmailVerified: true},
		},
	})

	if err := f.service.UnlinkIdentity(context.Background(), accountID, firstIdentityID); !errors.Is(err, auth.ErrLastIdentity) {
		t.Fatalf("last identity unlink error = %v", err)
	}
	if countIdentities(t, h) != 2 {
		t.Fatal("last identity unlink deleted a row")
	}
	if err := f.service.UnlinkIdentity(context.Background(), accountID, otherIdentityID); !errors.Is(err, auth.ErrIdentityNotFound) {
		t.Fatalf("not-owned unlink error = %v", err)
	}
	if countIdentities(t, h) != 2 {
		t.Fatal("not-owned unlink deleted a row")
	}
	if got := countAuditAction(f, auth.ActionIdentityUnlinkDenied); got != 2 {
		t.Fatalf("unlink denied audit entries = %d, want 2", got)
	}

	second, err := f.service.LinkIdentity(context.Background(), accountID, "second-token")
	if err != nil {
		t.Fatalf("link second identity: %v", err)
	}
	if err := f.service.UnlinkIdentity(context.Background(), accountID, firstIdentityID); err != nil {
		t.Fatalf("unlink owned non-last identity: %v", err)
	}
	if countIdentities(t, h) != 2 {
		t.Fatalf("identity count after unlink = %d, want 2", countIdentities(t, h))
	}
	if err := f.service.UnlinkIdentity(context.Background(), accountID, second.ID); !errors.Is(err, auth.ErrLastIdentity) {
		t.Fatalf("unlink remaining identity error = %v", err)
	}
	if identityAccountForUID(t, h, "identity-unlink-second") != accountID {
		t.Fatal("remaining identity changed owner")
	}
	assertAuditAction(t, f, auth.ActionIdentityUnlinked)
}

func TestConcurrentLinkRacesAllowOneOwner(t *testing.T) {
	h := testutil.StartPostgres(t)
	firstAccountID := seedLinkedAccount(t, h, "identity-race-first")
	secondAccountID := seedLinkedAccount(t, h, "identity-race-second")
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{
			"race-token": {UID: "identity-race", EmailVerified: true},
		},
	})

	start := make(chan struct{})
	results := make(chan struct {
		accountID uuid.UUID
		err       error
	}, 2)
	var wg sync.WaitGroup
	for _, accountID := range []uuid.UUID{firstAccountID, secondAccountID} {
		wg.Add(1)
		go func(accountID uuid.UUID) {
			defer wg.Done()
			<-start
			_, err := f.service.LinkIdentity(context.Background(), accountID, "race-token")
			results <- struct {
				accountID uuid.UUID
				err       error
			}{accountID: accountID, err: err}
		}(accountID)
	}
	close(start)
	wg.Wait()
	close(results)

	var winner uuid.UUID
	successes, conflicts := 0, 0
	for result := range results {
		switch {
		case result.err == nil:
			successes++
			winner = result.accountID
		case errors.Is(result.err, auth.ErrIdentityConflict):
			conflicts++
		default:
			t.Fatalf("concurrent link unexpected error: %v", result.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent link results: successes=%d conflicts=%d", successes, conflicts)
	}
	if got := identityAccountForUID(t, h, "identity-race"); got != winner {
		t.Fatalf("race identity owner = %s, want winner %s", got, winner)
	}
	if got := countAuditAction(f, auth.ActionIdentityLinked); got != 1 {
		t.Fatalf("link success audit entries = %d, want 1", got)
	}
	if got := countAuditAction(f, auth.ActionIdentityLinkDenied); got != 1 {
		t.Fatalf("link denied audit entries = %d, want 1", got)
	}
}
