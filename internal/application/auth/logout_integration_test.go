//go:build integration

package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/platform/testutil"
)

func countAuditAction(t *testing.T, f exchangeFixture, action string) int {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_log WHERE action = $1`, action).Scan(&count); err != nil {
		t.Fatalf("query audit action %s: %v", action, err)
	}
	return count
}

func TestLogoutRevokesOnlyPresentedFamilyAndIsIdempotent(t *testing.T) {
	h := testutil.StartPostgres(t)
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{
			"valid-token": {UID: "uid-logout", EmailVerified: true},
		},
	})
	first, err := exchange(t, f, "valid-token")
	if err != nil {
		t.Fatalf("first exchange: %v", err)
	}
	second, err := exchange(t, f, "valid-token")
	if err != nil {
		t.Fatalf("second exchange: %v", err)
	}
	expiring, err := exchange(t, f, "valid-token")
	if err != nil {
		t.Fatalf("expiring exchange: %v", err)
	}
	verifierCalls := f.verifier.Calls()

	if err := f.service.Logout(context.Background(), first.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if f.verifier.Calls() != verifierCalls {
		t.Fatal("logout attempted a Firebase sign-out; Firebase was called")
	}
	assertFamilyRevoked(t, h, first.SessionID)
	if secondRow := readRefreshToken(t, h, second.RefreshToken); secondRow.RevokedAt != nil {
		t.Fatal("logout revoked another refresh family")
	}
	assertAuditAction(t, f, auth.ActionSessionRevoked)

	if _, err := refreshSession(t, f, first.RefreshToken); !errors.Is(err, auth.ErrAuthenticationFailed) {
		t.Fatalf("refresh after logout error = %v, want generic authentication failure", err)
	}
	if _, err := refreshSession(t, f, second.RefreshToken); err != nil {
		t.Fatalf("other family refresh: %v", err)
	}

	if err := f.service.Logout(context.Background(), first.RefreshToken); err != nil {
		t.Fatalf("repeated logout: %v", err)
	}
	if err := f.service.Logout(context.Background(), "unknown-or-malformed-token"); err != nil {
		t.Fatalf("unknown token logout: %v", err)
	}
	if f.verifier.Calls() != verifierCalls {
		t.Fatal("idempotent logout called Firebase")
	}

	f.clock.Advance(31 * 24 * time.Hour)
	if err := f.service.Logout(context.Background(), expiring.RefreshToken); err != nil {
		t.Fatalf("expired token logout: %v", err)
	}
	if row := readRefreshToken(t, h, expiring.RefreshToken); row.RevokedAt == nil {
		t.Fatal("expired token logout did not revoke the family")
	}
	assertFamilyRevoked(t, h, expiring.SessionID)
}

func TestLogoutAllRevokesGuestFleetAndOnlyThatAccount(t *testing.T) {
	h := testutil.StartPostgres(t)
	accountID := seedLinkedAccount(t, h, "logout-all")
	seedFleetGrant(t, h, accountID)
	otherAccountID := seedLinkedAccount(t, h, "logout-all-other")
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{
			"session-token": {UID: "logout-all", EmailVerified: true},
			"other-token":   {UID: "logout-all-other", EmailVerified: true},
		},
	})

	guestSession, err := exchangeAs(t, f, "session-token", auth.ClientGuestMobile)
	if err != nil {
		t.Fatalf("guest exchange: %v", err)
	}
	fleetSession, err := exchangeAs(t, f, "session-token", auth.ClientFleetConsole)
	if err != nil {
		t.Fatalf("Fleet exchange: %v", err)
	}
	otherSession, err := exchangeAs(t, f, "other-token", auth.ClientGuestMobile)
	if err != nil {
		t.Fatalf("other account exchange: %v", err)
	}
	if guestSession.AccountID != accountID || fleetSession.AccountID != accountID || otherSession.AccountID != otherAccountID {
		t.Fatalf("unexpected account ids: guest=%s fleet=%s other=%s", guestSession.AccountID, fleetSession.AccountID, otherSession.AccountID)
	}
	verifierCalls := f.verifier.Calls()

	if err := f.service.LogoutAll(context.Background(), accountID); err != nil {
		t.Fatalf("logout-all: %v", err)
	}
	if f.verifier.Calls() != verifierCalls {
		t.Fatal("logout-all attempted a Firebase sign-out; Firebase was called")
	}
	assertFamilyRevoked(t, h, guestSession.SessionID)
	assertFamilyRevoked(t, h, fleetSession.SessionID)
	if otherRow := readRefreshToken(t, h, otherSession.RefreshToken); otherRow.RevokedAt != nil {
		t.Fatal("logout-all revoked another account's family")
	}
	if got := countAuditAction(t, f, auth.ActionSessionRevokedAll); got != 1 {
		t.Fatalf("logout-all audit entries = %d, want 1", got)
	}

	for name, token := range map[string]string{
		"guest": guestSession.RefreshToken,
		"fleet": fleetSession.RefreshToken,
	} {
		if _, err := refreshSession(t, f, token); !errors.Is(err, auth.ErrAuthenticationFailed) {
			t.Fatalf("%s refresh after logout-all error = %v, want generic authentication failure", name, err)
		}
	}
	if _, err := refreshSession(t, f, otherSession.RefreshToken); err != nil {
		t.Fatalf("other account refresh after logout-all: %v", err)
	}

	if err := f.service.LogoutAll(context.Background(), accountID); err != nil {
		t.Fatalf("repeated logout-all: %v", err)
	}
	if got := countAuditAction(t, f, auth.ActionSessionRevokedAll); got != 1 {
		t.Fatalf("repeated logout-all audit entries = %d, want 1", got)
	}
}
