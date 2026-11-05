//go:build integration

package auth_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/testutil"
)

func refreshSession(t *testing.T, f exchangeFixture, token string) (auth.IssuedSession, error) {
	t.Helper()
	return f.service.Refresh(context.Background(), auth.RefreshInput{
		RefreshToken: token,
		UserAgent:    "guest-mobile/1.1",
		IPAddress:    "198.51.100.20",
	})
}

type storedRefreshToken struct {
	ID         uuid.UUID
	FamilyID   uuid.UUID
	SubjectID  uuid.UUID
	Client     string
	ExpiresAt  time.Time
	UsedAt     *time.Time
	RevokedAt  *time.Time
	ReplacedBy uuid.NullUUID
	UserAgent  *string
}

func readRefreshToken(t *testing.T, h *testutil.Harness, token string) storedRefreshToken {
	t.Helper()
	var row storedRefreshToken
	err := h.Pool.QueryRow(context.Background(),
		`SELECT id, family_id, subject_id, client, expires_at, used_at, revoked_at, replaced_by, user_agent
		 FROM refresh_tokens WHERE token_hash = $1`,
		auth.HashRefreshToken(token),
	).Scan(&row.ID, &row.FamilyID, &row.SubjectID, &row.Client, &row.ExpiresAt,
		&row.UsedAt, &row.RevokedAt, &row.ReplacedBy, &row.UserAgent)
	if err != nil {
		t.Fatalf("read refresh token: %v", err)
	}
	return row
}

func assertFamilyRevoked(t *testing.T, h *testutil.Harness, familyID uuid.UUID) {
	t.Helper()
	var total, revoked int
	err := h.Pool.QueryRow(context.Background(),
		`SELECT count(*), count(revoked_at) FROM refresh_tokens WHERE family_id = $1`,
		familyID).Scan(&total, &revoked)
	if err != nil {
		t.Fatalf("count family tokens: %v", err)
	}
	if total == 0 || revoked != total {
		t.Fatalf("family %s revoked %d/%d tokens", familyID, revoked, total)
	}
}

func assertAuditAction(t *testing.T, f exchangeFixture, action string) {
	t.Helper()
	for _, got := range f.audit.Actions() {
		if got == action {
			return
		}
	}
	t.Fatalf("audit actions %v do not contain %s", f.audit.Actions(), action)
}

func TestRefreshRotatesAndReplayRevokesFamily(t *testing.T) {
	h := testutil.StartPostgres(t)
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{
			"valid-token": {UID: "uid-refresh", Email: "refresh@example.com", EmailVerified: true},
		},
	})
	first, err := exchange(t, f, "valid-token")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	oldRow := readRefreshToken(t, h, first.RefreshToken)
	f.clock.Advance(5 * time.Minute)

	second, err := refreshSession(t, f, first.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if second.SessionID != first.SessionID || second.AccountID != first.AccountID || second.Client != first.Client {
		t.Fatalf("rotated session identity = %+v, want family %s account %s client %s", second, first.SessionID, first.AccountID, first.Client)
	}
	if second.RefreshToken == first.RefreshToken || second.AccessToken == first.AccessToken {
		t.Fatal("refresh did not rotate both credentials")
	}
	if !second.RefreshExpiresAt.Equal(f.clock.Now().Add(30 * 24 * time.Hour)) {
		t.Fatalf("rotated refresh expiry = %s", second.RefreshExpiresAt)
	}
	claims, err := f.validator.Validate(context.Background(), second.AccessToken)
	if err != nil {
		t.Fatalf("validate rotated access token: %v", err)
	}
	if claims.Subject != first.AccountID.String() || claims.Client != auth.ClientGuestMobile {
		t.Fatalf("rotated claims = subject %q client %q", claims.Subject, claims.Client)
	}

	rotatedRow := readRefreshToken(t, h, second.RefreshToken)
	oldRow = readRefreshToken(t, h, first.RefreshToken)
	if oldRow.UsedAt == nil {
		t.Fatal("presented refresh token was not marked used")
	}
	if !oldRow.ReplacedBy.Valid || oldRow.ReplacedBy.UUID != rotatedRow.ID {
		t.Fatalf("replaced_by = %+v, want %s", oldRow.ReplacedBy, rotatedRow.ID)
	}
	if oldRow.RevokedAt != nil {
		t.Fatal("presented refresh token was revoked during successful rotation")
	}
	if rotatedRow.FamilyID != first.SessionID || rotatedRow.SubjectID != first.AccountID || rotatedRow.Client != first.Client {
		t.Fatalf("replacement row = %+v", rotatedRow)
	}
	if rotatedRow.UsedAt != nil || rotatedRow.RevokedAt != nil {
		t.Fatalf("replacement row is already unusable: used=%v revoked=%v", rotatedRow.UsedAt, rotatedRow.RevokedAt)
	}
	assertAuthRowCounts(t, h, 1, 1, 2)
	assertAuditAction(t, f, auth.ActionSessionRefreshed)

	if _, err := refreshSession(t, f, first.RefreshToken); !errors.Is(err, auth.ErrAuthenticationFailed) {
		t.Fatalf("replay error = %v, want generic authentication failure", err)
	}
	assertFamilyRevoked(t, h, first.SessionID)
	assertAuditAction(t, f, auth.ActionSessionReuse)

	if _, err := refreshSession(t, f, second.RefreshToken); !errors.Is(err, auth.ErrAuthenticationFailed) {
		t.Fatalf("replacement refresh after reuse error = %v, want generic authentication failure", err)
	}
}

func TestConcurrentRefreshAllowsAtMostOneRotation(t *testing.T) {
	h := testutil.StartPostgres(t)
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{
			"valid-token": {UID: "uid-refresh-concurrent", EmailVerified: true},
		},
	})
	first, err := exchange(t, f, "valid-token")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}

	start := make(chan struct{})
	results := make(chan struct {
		session auth.IssuedSession
		err     error
	}, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			session, err := refreshSession(t, f, first.RefreshToken)
			results <- struct {
				session auth.IssuedSession
				err     error
			}{session: session, err: err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	for result := range results {
		if result.err != nil {
			if !errors.Is(result.err, auth.ErrAuthenticationFailed) {
				t.Fatalf("concurrent refresh failed unexpectedly: %v", result.err)
			}
			continue
		}
		successes++
		if result.session.SessionID != first.SessionID || result.session.AccountID != first.AccountID {
			t.Fatalf("concurrent refresh returned unexpected session: %+v", result.session)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent refresh successes = %d, want exactly 1", successes)
	}
	assertAuthRowCounts(t, h, 1, 1, 2)
	assertFamilyRevoked(t, h, first.SessionID)
	assertAuditAction(t, f, auth.ActionSessionRefreshed)
	assertAuditAction(t, f, auth.ActionSessionReuse)
}

func TestRefreshRejectsExpiredInactiveAndUnknownTokens(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		h := testutil.StartPostgres(t)
		f := newExchangeFixture(t, h, &testutil.FakeVerifier{
			Identities: map[string]auth.VerifiedIdentity{"valid-token": {UID: "uid-refresh-expired"}},
		})
		first, err := exchange(t, f, "valid-token")
		if err != nil {
			t.Fatalf("exchange: %v", err)
		}
		f.clock.Advance(31 * 24 * time.Hour)

		if _, err := refreshSession(t, f, first.RefreshToken); !errors.Is(err, auth.ErrAuthenticationFailed) {
			t.Fatalf("expired refresh error = %v", err)
		}
		row := readRefreshToken(t, h, first.RefreshToken)
		if row.UsedAt != nil || row.RevokedAt != nil {
			t.Fatalf("expired token was mutated: used=%v revoked=%v", row.UsedAt, row.RevokedAt)
		}
		assertAuthRowCounts(t, h, 1, 1, 1)
	})

	for _, status := range []string{"suspended", "deleted"} {
		t.Run(status, func(t *testing.T) {
			h := testutil.StartPostgres(t)
			f := newExchangeFixture(t, h, &testutil.FakeVerifier{
				Identities: map[string]auth.VerifiedIdentity{"valid-token": {UID: "uid-refresh-" + status}},
			})
			first, err := exchange(t, f, "valid-token")
			if err != nil {
				t.Fatalf("exchange: %v", err)
			}
			if _, err := h.Pool.Exec(context.Background(),
				`UPDATE accounts SET status = $2, deleted_at = CASE WHEN $2 = 'deleted' THEN now() ELSE NULL END WHERE id = $1`,
				first.AccountID, status); err != nil {
				t.Fatalf("update account: %v", err)
			}
			if _, err := refreshSession(t, f, first.RefreshToken); !errors.Is(err, auth.ErrAuthenticationFailed) {
				t.Fatalf("%s refresh error = %v", status, err)
			}
			row := readRefreshToken(t, h, first.RefreshToken)
			if row.UsedAt != nil || row.RevokedAt != nil {
				t.Fatalf("inactive account token was mutated: used=%v revoked=%v", row.UsedAt, row.RevokedAt)
			}
			assertAuthRowCounts(t, h, 1, 1, 1)
		})
	}

	t.Run("unknown", func(t *testing.T) {
		h := testutil.StartPostgres(t)
		f := newExchangeFixture(t, h, &testutil.FakeVerifier{
			Identities: map[string]auth.VerifiedIdentity{"valid-token": {UID: "uid-refresh-unknown"}},
		})
		if _, err := exchange(t, f, "valid-token"); err != nil {
			t.Fatalf("exchange: %v", err)
		}
		for _, token := range []string{"unknown-refresh-token", "malformed"} {
			if _, err := refreshSession(t, f, token); !errors.Is(err, auth.ErrAuthenticationFailed) {
				t.Fatalf("%s refresh error = %v", token, err)
			}
		}
		assertAuthRowCounts(t, h, 1, 1, 1)
	})
}
