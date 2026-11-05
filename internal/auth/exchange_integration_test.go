//go:build integration

package auth_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/testutil"
)

type exchangeFixture struct {
	service   auth.Service
	store     *auth.Store
	verifier  *testutil.FakeVerifier
	clock     *testutil.FakeClock
	audit     *testutil.AuditRecorder
	validator *auth.AccessTokenIssuer
}

func newExchangeFixture(t *testing.T, h *testutil.Harness, verifier *testutil.FakeVerifier) exchangeFixture {
	t.Helper()
	clock := testutil.NewFakeClock(time.Date(2026, time.November, 5, 12, 0, 0, 0, time.UTC))
	_, keys := testutil.NewTestKeySet(t, "test-kid")
	store := auth.NewStore(h.Pool)
	auditRecorder := &testutil.AuditRecorder{}
	service, err := auth.NewService(auth.ServiceOptions{
		Store:      store,
		Verifier:   verifier,
		Keys:       keys,
		Limiter:    &testutil.FakeRateLimiter{},
		Clock:      clock,
		Audit:      auditRecorder,
		Issuer:     "test-issuer",
		Audience:   "test-audience",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 30 * 24 * time.Hour,
		ClockSkew:  time.Minute,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	validator, err := auth.NewAccessTokenIssuer(keys, "test-issuer", "test-audience", 15*time.Minute, time.Minute, clock)
	if err != nil {
		t.Fatalf("NewAccessTokenIssuer: %v", err)
	}
	return exchangeFixture{
		service:   service,
		store:     store,
		verifier:  verifier,
		clock:     clock,
		audit:     auditRecorder,
		validator: validator,
	}
}

func exchange(t *testing.T, f exchangeFixture, token string) (auth.IssuedSession, error) {
	t.Helper()
	return f.service.Exchange(context.Background(), auth.ExchangeInput{
		IDToken:   token,
		UserAgent: "guest-mobile/1.0",
		IPAddress: "192.0.2.10",
	})
}

func assertAuthRowCounts(t *testing.T, h *testutil.Harness, accounts, identities, sessions int) {
	t.Helper()
	for query, want := range map[string]int{
		"SELECT count(*) FROM accounts":           accounts,
		"SELECT count(*) FROM account_identities": identities,
		"SELECT count(*) FROM refresh_tokens":     sessions,
	} {
		var got int
		if err := h.Pool.QueryRow(context.Background(), query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		if got != want {
			t.Fatalf("%s = %d, want %d", query, got, want)
		}
	}
}

func TestExchangeRejectsInvalidFirebaseTokensWithoutWrites(t *testing.T) {
	h := testutil.StartPostgres(t)
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{},
		Errors: map[string]error{
			"expired":        auth.ErrInvalidFirebaseToken,
			"wrong-audience": auth.ErrInvalidFirebaseToken,
			"wrong-issuer":   auth.ErrInvalidFirebaseToken,
			"revoked":        auth.ErrInvalidFirebaseToken,
		},
	})

	for _, token := range []string{"malformed", "expired", "wrong-audience", "wrong-issuer", "revoked"} {
		if _, err := exchange(t, f, token); !errors.Is(err, auth.ErrAuthenticationFailed) {
			t.Fatalf("%s token error = %v, want generic authentication failure", token, err)
		}
	}
	assertAuthRowCounts(t, h, 0, 0, 0)
}

func TestExchangeProvisionsReturningIdentityAndSession(t *testing.T) {
	h := testutil.StartPostgres(t)
	identity := auth.VerifiedIdentity{
		UID:           "firebase-uid-1",
		Email:         "Guest@Example.com",
		EmailVerified: true,
		DisplayName:   " Guest One ",
	}
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{"valid-token": identity},
	})

	session, err := exchange(t, f, "valid-token")
	if err != nil {
		t.Fatalf("first exchange: %v", err)
	}
	if session.Client != auth.ClientGuestMobile {
		t.Fatalf("client = %q, want %q", session.Client, auth.ClientGuestMobile)
	}
	if session.AccountID == uuid.Nil || session.SessionID == uuid.Nil {
		t.Fatalf("missing identifiers: %+v", session)
	}
	if !session.RefreshExpiresAt.Equal(f.clock.Now().Add(30 * 24 * time.Hour)) {
		t.Fatalf("refresh expiry = %s, want clock-based 30 days", session.RefreshExpiresAt)
	}

	account, err := f.store.GetAccount(context.Background(), session.AccountID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if account.Email == nil || *account.Email != identity.Email {
		t.Fatalf("account email = %v, want %q", account.Email, identity.Email)
	}
	if account.DisplayName == nil || *account.DisplayName != "Guest One" {
		t.Fatalf("account display name = %v, want trimmed profile name", account.DisplayName)
	}
	if account.Status != "active" {
		t.Fatalf("account status = %q, want active", account.Status)
	}

	linked, err := f.store.GetIdentityByProviderSubject(context.Background(), "firebase", identity.UID)
	if err != nil {
		t.Fatalf("GetIdentityByProviderSubject: %v", err)
	}
	if linked.AccountID != session.AccountID {
		t.Fatalf("identity account = %s, want %s", linked.AccountID, session.AccountID)
	}
	if linked.Email == nil || *linked.Email != identity.Email {
		t.Fatalf("identity email = %v, want %q", linked.Email, identity.Email)
	}

	claims, err := f.validator.Validate(context.Background(), session.AccessToken)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.Subject != session.AccountID.String() || claims.Client != auth.ClientGuestMobile {
		t.Fatalf("access claims = subject %q client %q", claims.Subject, claims.Client)
	}
	if !claims.ExpiresAt.Time.Equal(session.AccessExpiresAt) {
		t.Fatalf("access expiry = %s, session expiry = %s", claims.ExpiresAt.Time, session.AccessExpiresAt)
	}

	var client, subjectType string
	var subjectID uuid.UUID
	var tokenHash []byte
	var userAgent *string
	err = h.Pool.QueryRow(context.Background(),
		`SELECT client, subject_type, subject_id, token_hash, user_agent FROM refresh_tokens WHERE family_id = $1`,
		session.SessionID).Scan(&client, &subjectType, &subjectID, &tokenHash, &userAgent)
	if err != nil {
		t.Fatalf("read refresh token: %v", err)
	}
	if client != auth.ClientGuestMobile || subjectType != "account" || subjectID != session.AccountID {
		t.Fatalf("refresh row = client %q subject %q/%s", client, subjectType, subjectID)
	}
	if !bytes.Equal(tokenHash, auth.HashRefreshToken(session.RefreshToken)) {
		t.Fatal("stored refresh hash does not match returned refresh token")
	}
	if bytes.Equal(tokenHash, []byte(session.RefreshToken)) {
		t.Fatal("raw refresh token was stored")
	}
	if userAgent == nil || *userAgent != "guest-mobile/1.0" {
		t.Fatalf("user agent = %v", userAgent)
	}

	second, err := exchange(t, f, "valid-token")
	if err != nil {
		t.Fatalf("returning exchange: %v", err)
	}
	if second.AccountID != session.AccountID {
		t.Fatalf("returning account = %s, want %s", second.AccountID, session.AccountID)
	}
	if second.SessionID == session.SessionID {
		t.Fatal("returning exchange reused the previous refresh family")
	}
	assertAuthRowCounts(t, h, 1, 1, 2)
}

func TestConcurrentFirstExchangeCreatesOneAccountAndIdentity(t *testing.T) {
	h := testutil.StartPostgres(t)
	identity := auth.VerifiedIdentity{
		UID:           "firebase-uid-concurrent",
		Email:         "concurrent@example.com",
		EmailVerified: true,
		DisplayName:   "Concurrent Guest",
	}
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{"valid-token": identity},
	})

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
			session, err := exchange(t, f, "valid-token")
			results <- struct {
				session auth.IssuedSession
				err     error
			}{session: session, err: err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var accountID uuid.UUID
	successes := 0
	for result := range results {
		if result.err != nil {
			if !errors.Is(result.err, auth.ErrAuthenticationFailed) {
				t.Fatalf("concurrent exchange failed unexpectedly: %v", result.err)
			}
			continue
		}
		successes++
		if accountID == uuid.Nil {
			accountID = result.session.AccountID
		}
		if result.session.AccountID != accountID {
			t.Fatalf("concurrent exchanges resolved different accounts: %s and %s", accountID, result.session.AccountID)
		}
	}
	if successes == 0 {
		t.Fatal("both concurrent first exchanges failed")
	}
	assertAuthRowCounts(t, h, 1, 1, successes)
}
