package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/audit"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

type fakeServiceStore struct {
	identity *sqlc.AccountIdentity
	account  sqlc.Account

	provisionErr error
	sessionErr   error

	provisionCalls      int
	sessionCalls        int
	provisionedAccount  sqlc.CreateAccountParams
	provisionedIdentity sqlc.LinkAccountIdentityParams
	sessions            []sqlc.CreateRefreshTokenParams
	provisionAudit      []audit.Entry
	sessionAudit        []audit.Entry

	refreshToken      *sqlc.RefreshToken
	refreshLookupErr  error
	rotateErr         error
	rotatedAccount    sqlc.Account
	rotatedToken      sqlc.RefreshToken
	refreshLookupHash []byte
	rotatedHash       []byte
	replacement       sqlc.CreateRefreshTokenParams
	rotateCalls       int
	rotatedAudit      []audit.Entry
	reuseAudit        []audit.Entry

	adminAuthority AdminAuthority
	adminErr       error
	adminCalls     int

	logoutErr        error
	logoutCalls      int
	logoutHash       []byte
	logoutEntry      audit.Entry
	logoutAllErr     error
	logoutAllCalls   int
	logoutAllAccount uuid.UUID
	logoutAllEntry   audit.Entry

	linkIdentity  sqlc.AccountIdentity
	linkErr       error
	linkCalls     int
	linkAccount   uuid.UUID
	linkParams    sqlc.LinkAccountIdentityParams
	linkEntry     audit.Entry
	unlinkErr     error
	unlinkCalls   int
	unlinkAccount uuid.UUID
	unlinkedID    uuid.UUID
	unlinkEntry   audit.Entry
}

func (f *fakeServiceStore) GetIdentityByProviderSubject(context.Context, string, string) (sqlc.AccountIdentity, error) {
	if f.identity == nil {
		return sqlc.AccountIdentity{}, ErrIdentityNotFound
	}
	return *f.identity, nil
}

func (f *fakeServiceStore) GetAccount(context.Context, uuid.UUID) (sqlc.Account, error) {
	if f.account.ID == uuid.Nil {
		return sqlc.Account{}, ErrIdentityNotFound
	}
	return f.account, nil
}

func (f *fakeServiceStore) ProvisionGuest(_ context.Context, account sqlc.CreateAccountParams, identity sqlc.LinkAccountIdentityParams, entry audit.Entry, _ audit.Appender) (sqlc.Account, sqlc.AccountIdentity, error) {
	f.provisionCalls++
	f.provisionedAccount = account
	f.provisionedIdentity = identity
	f.provisionAudit = append(f.provisionAudit, entry)
	if f.provisionErr != nil {
		return sqlc.Account{}, sqlc.AccountIdentity{}, f.provisionErr
	}
	created := sqlc.Account{
		ID: account.ID, Email: account.Email, DisplayName: account.DisplayName,
		Phone: account.Phone, Locale: account.Locale, Status: account.Status,
	}
	identity.AccountID = created.ID
	linked := sqlc.AccountIdentity{
		ID: identity.ID, AccountID: created.ID, Provider: identity.Provider,
		ProviderSub: identity.ProviderSub, Email: identity.Email,
	}
	f.account = created
	f.identity = &linked
	return created, linked, nil
}

func (f *fakeServiceStore) CreateSession(_ context.Context, token sqlc.CreateRefreshTokenParams, entry audit.Entry, _ audit.Appender) error {
	f.sessionCalls++
	f.sessions = append(f.sessions, token)
	f.sessionAudit = append(f.sessionAudit, entry)
	return f.sessionErr
}

func (f *fakeServiceStore) ResolveActiveAdmin(context.Context, uuid.UUID) (AdminAuthority, error) {
	f.adminCalls++
	if f.adminErr != nil {
		return AdminAuthority{}, f.adminErr
	}
	return f.adminAuthority, nil
}

func (f *fakeServiceStore) RevokeRefreshFamilyByHash(_ context.Context, hash []byte, entry audit.Entry, _ audit.Appender) error {
	f.logoutCalls++
	f.logoutHash = append([]byte(nil), hash...)
	f.logoutEntry = entry
	return f.logoutErr
}

func (f *fakeServiceStore) RevokeAccountSessions(_ context.Context, accountID uuid.UUID, entry audit.Entry, _ audit.Appender) error {
	f.logoutAllCalls++
	f.logoutAllAccount = accountID
	f.logoutAllEntry = entry
	return f.logoutAllErr
}

func (f *fakeServiceStore) LinkIdentityForAccount(_ context.Context, accountID uuid.UUID, identity sqlc.LinkAccountIdentityParams, entry audit.Entry, _ audit.Appender) (sqlc.AccountIdentity, error) {
	f.linkCalls++
	f.linkAccount = accountID
	f.linkParams = identity
	f.linkEntry = entry
	if f.linkErr != nil {
		return sqlc.AccountIdentity{}, f.linkErr
	}
	if f.linkIdentity.ID == uuid.Nil {
		f.linkIdentity = sqlc.AccountIdentity{
			ID: identity.ID, AccountID: accountID, Provider: identity.Provider,
			ProviderSub: identity.ProviderSub, Email: identity.Email,
		}
	}
	return f.linkIdentity, nil
}

func (f *fakeServiceStore) UnlinkIdentityForAccount(_ context.Context, accountID, identityID uuid.UUID, entry audit.Entry, _ audit.Appender) error {
	f.unlinkCalls++
	f.unlinkAccount = accountID
	f.unlinkedID = identityID
	f.unlinkEntry = entry
	return f.unlinkErr
}

func (f *fakeServiceStore) GetRefreshTokenByHash(_ context.Context, hash []byte) (sqlc.RefreshToken, error) {
	f.refreshLookupHash = append([]byte(nil), hash...)
	if f.refreshLookupErr != nil {
		return sqlc.RefreshToken{}, f.refreshLookupErr
	}
	if f.refreshToken == nil {
		return sqlc.RefreshToken{}, ErrRefreshTokenNotFound
	}
	return *f.refreshToken, nil
}

func (f *fakeServiceStore) RotateRefreshToken(
	_ context.Context,
	hash []byte,
	replacement sqlc.CreateRefreshTokenParams,
	refreshedEntry audit.Entry,
	reuseEntry audit.Entry,
	_ time.Time,
	_ audit.Appender,
) (sqlc.Account, sqlc.RefreshToken, error) {
	f.rotateCalls++
	f.rotatedHash = append([]byte(nil), hash...)
	f.replacement = replacement
	f.rotatedAudit = append(f.rotatedAudit, refreshedEntry)
	f.reuseAudit = append(f.reuseAudit, reuseEntry)
	if f.rotateErr != nil {
		return sqlc.Account{}, sqlc.RefreshToken{}, f.rotateErr
	}
	return f.rotatedAccount, f.rotatedToken, nil
}

type unitVerifier struct {
	identities map[string]VerifiedIdentity
	errors     map[string]error
	err        error
	calls      int
}

func (v *unitVerifier) Verify(_ context.Context, token string) (VerifiedIdentity, error) {
	v.calls++
	if v.err != nil {
		return VerifiedIdentity{}, v.err
	}
	if err, ok := v.errors[token]; ok {
		return VerifiedIdentity{}, err
	}
	identity, ok := v.identities[token]
	if !ok {
		return VerifiedIdentity{}, ErrInvalidFirebaseToken
	}
	return identity, nil
}

type unitClock struct {
	now time.Time
}

func (c *unitClock) Now() time.Time { return c.now }

type unitLimiterCall struct {
	key    string
	limit  int
	window time.Duration
}

type unitLimiter struct {
	decision RateLimitDecision
	err      error
	calls    []unitLimiterCall
}

func (l *unitLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (RateLimitDecision, error) {
	l.calls = append(l.calls, unitLimiterCall{key: key, limit: limit, window: window})
	if l.err != nil {
		return RateLimitDecision{}, l.err
	}
	if !l.decision.Allowed && l.decision.RetryAfter == 0 {
		return RateLimitDecision{Allowed: true}, nil
	}
	return l.decision, nil
}

type unitAudit struct{}

func (unitAudit) AppendTx(context.Context, audit.Execer, audit.Entry) error { return nil }

type authFixture struct {
	service   Service
	store     *fakeServiceStore
	verifier  *unitVerifier
	limiter   *unitLimiter
	clock     *unitClock
	validator *AccessTokenIssuer
}

func newAuthFixture(t *testing.T, verifier *unitVerifier, store *fakeServiceStore) authFixture {
	t.Helper()
	clock := &unitClock{now: time.Date(2026, time.November, 5, 12, 0, 0, 0, time.UTC)}
	signer, err := NewRandomLocalSigner("test-kid")
	if err != nil {
		t.Fatalf("NewRandomLocalSigner: %v", err)
	}
	keys, err := NewStaticKeySet("test-kid", signer)
	if err != nil {
		t.Fatalf("NewStaticKeySet: %v", err)
	}
	limiter := &unitLimiter{}
	if store == nil {
		store = &fakeServiceStore{}
	}
	service, err := newService(ServiceOptions{
		Verifier:   verifier,
		Keys:       keys,
		Limiter:    limiter,
		Clock:      clock,
		Audit:      unitAudit{},
		Issuer:     "test-issuer",
		Audience:   "test-audience",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 30 * 24 * time.Hour,
		ClockSkew:  time.Minute,
	}, store)
	if err != nil {
		t.Fatalf("newService: %v", err)
	}
	validator, err := NewAccessTokenIssuer(keys, "test-issuer", "test-audience", 15*time.Minute, time.Minute, clock)
	if err != nil {
		t.Fatalf("NewAccessTokenIssuer: %v", err)
	}
	return authFixture{
		service: service, store: store, verifier: verifier, limiter: limiter,
		clock: clock, validator: validator,
	}
}

func runExchange(t *testing.T, f authFixture, input ExchangeInput) (IssuedSession, error) {
	t.Helper()
	if input.IPAddress == "" {
		input.IPAddress = "192.0.2.20"
	}
	if input.UserAgent == "" {
		input.UserAgent = "guest-mobile/test"
	}
	return f.service.Exchange(context.Background(), input)
}

func TestExchangeDefaultsToGuestMobileAndIssuesOneSession(t *testing.T) {
	f := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{
			"valid-token": {UID: "firebase-uid", DisplayName: "Test Guest"},
		},
	}, nil)

	session, err := runExchange(t, f, ExchangeInput{IDToken: "valid-token"})
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if session.Client != ClientGuestMobile {
		t.Fatalf("session client = %q, want %q", session.Client, ClientGuestMobile)
	}
	if f.store.provisionCalls != 1 || f.store.sessionCalls != 1 {
		t.Fatalf("store calls = provision %d session %d, want 1 and 1", f.store.provisionCalls, f.store.sessionCalls)
	}
	if len(f.store.sessions) != 1 || f.store.sessions[0].Client != ClientGuestMobile {
		t.Fatalf("refresh rows = %+v", f.store.sessions)
	}
	if f.store.provisionedAccount.Locale != "en" || f.store.provisionedAccount.Status != "active" {
		t.Fatalf("provisioned account = %+v", f.store.provisionedAccount)
	}
	if f.store.provisionedAccount.Email != nil || f.store.provisionedIdentity.Email != nil {
		t.Fatal("unverified email was persisted")
	}
	if len(f.store.provisionAudit) != 1 || f.store.provisionAudit[0].Action != ActionIdentityProvisioned {
		t.Fatalf("provisioning audit = %+v", f.store.provisionAudit)
	}
	if len(f.store.sessionAudit) != 1 || f.store.sessionAudit[0].Action != ActionSessionIssued {
		t.Fatalf("session audit = %+v", f.store.sessionAudit)
	}

	claims, err := f.validator.Validate(context.Background(), session.AccessToken)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.Subject != session.AccountID.String() || claims.Client != ClientGuestMobile {
		t.Fatalf("claims = subject %q client %q", claims.Subject, claims.Client)
	}
}

func TestExchangePersistsOnlyVerifiedProfileFields(t *testing.T) {
	f := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{
			"verified":   {UID: "uid-verified", Email: "guest@example.com", EmailVerified: true, DisplayName: " Guest "},
			"unverified": {UID: "uid-unverified", Email: "unverified@example.com", DisplayName: "Guest"},
		},
	}, nil)

	if _, err := runExchange(t, f, ExchangeInput{IDToken: "verified"}); err != nil {
		t.Fatalf("verified exchange: %v", err)
	}
	if f.store.provisionedAccount.Email == nil || *f.store.provisionedAccount.Email != "guest@example.com" {
		t.Fatalf("account email = %v", f.store.provisionedAccount.Email)
	}
	if f.store.provisionedAccount.DisplayName == nil || *f.store.provisionedAccount.DisplayName != "Guest" {
		t.Fatalf("display name = %v", f.store.provisionedAccount.DisplayName)
	}
	if f.store.provisionedIdentity.Email == nil || *f.store.provisionedIdentity.Email != "guest@example.com" {
		t.Fatalf("identity email = %v", f.store.provisionedIdentity.Email)
	}

	other := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{
			"unverified": {UID: "uid-unverified", Email: "unverified@example.com", DisplayName: "Guest"},
		},
	}, nil)
	if _, err := runExchange(t, other, ExchangeInput{IDToken: "unverified"}); err != nil {
		t.Fatalf("unverified exchange: %v", err)
	}
	if other.store.provisionedAccount.Email != nil || other.store.provisionedIdentity.Email != nil {
		t.Fatalf("unverified email persisted: account=%v identity=%v",
			other.store.provisionedAccount.Email, other.store.provisionedIdentity.Email)
	}
}

func TestExchangeDoesNotPersistFirebaseIDToken(t *testing.T) {
	const idToken = "firebase-id-token-that-must-not-be-persisted"
	f := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{
			idToken: {UID: "uid-no-persist", Email: "safe@example.com", EmailVerified: true},
		},
	}, nil)

	session, err := runExchange(t, f, ExchangeInput{IDToken: idToken})
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if session.AccessToken == idToken || session.RefreshToken == idToken {
		t.Fatal("backend returned the Firebase ID Token as a session credential")
	}
	if len(f.store.sessions) != 1 {
		t.Fatalf("session rows = %d, want 1", len(f.store.sessions))
	}
	if !bytes.Equal(f.store.sessions[0].TokenHash, HashRefreshToken(session.RefreshToken)) {
		t.Fatal("refresh token hash was not what was stored")
	}
	for _, entry := range append(f.store.provisionAudit, f.store.sessionAudit...) {
		encoded, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("marshal audit: %v", err)
		}
		if strings.Contains(string(encoded), idToken) {
			t.Fatalf("audit entry contains Firebase ID Token: %s", encoded)
		}
	}
}

func TestExchangeReturnsGenericAuthenticationErrors(t *testing.T) {
	invalid := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{
			"revoked": {UID: "uid-revoked", EmailVerified: true},
		},
		errors: map[string]error{"revoked": ErrInvalidFirebaseToken},
	}, nil)
	for _, token := range []string{"malformed", "revoked"} {
		_, err := runExchange(t, invalid, ExchangeInput{IDToken: token})
		if !errors.Is(err, ErrAuthenticationFailed) {
			t.Fatalf("%s error = %v, want ErrAuthenticationFailed", token, err)
		}
	}

	inactive := &fakeServiceStore{
		identity: &sqlc.AccountIdentity{ID: uuid.New(), AccountID: uuid.New(), Provider: "firebase", ProviderSub: "uid-inactive"},
		account:  sqlc.Account{ID: uuid.New(), Status: "suspended"},
	}
	inactive.account.ID = inactive.identity.AccountID
	f := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{"inactive": {UID: "uid-inactive"}},
	}, inactive)
	if _, err := runExchange(t, f, ExchangeInput{IDToken: "inactive"}); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("inactive account error = %v, want ErrAuthenticationFailed", err)
	}

	unavailable := newAuthFixture(t, &unitVerifier{err: ErrFirebaseUnavailable}, nil)
	if _, err := runExchange(t, unavailable, ExchangeInput{IDToken: "any"}); !errors.Is(err, ErrFirebaseUnavailable) {
		t.Fatalf("unavailable verifier error = %v", err)
	}
}

func TestExchangeAppliesIPLimitBeforeVerification(t *testing.T) {
	f := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{"valid": {UID: "uid-limited"}},
	}, nil)
	f.limiter.decision = RateLimitDecision{Allowed: false, RetryAfter: 42 * time.Second}

	_, err := runExchange(t, f, ExchangeInput{IDToken: "valid"})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("error = %v, want ErrRateLimited", err)
	}
	var limited *RateLimitedError
	if !errors.As(err, &limited) || limited.RetryAfter != 42*time.Second {
		t.Fatalf("rate-limit error = %#v", err)
	}
	if f.verifier.calls != 0 {
		t.Fatalf("verifier calls = %d, want 0", f.verifier.calls)
	}
	if f.store.provisionCalls != 0 || f.store.sessionCalls != 0 {
		t.Fatalf("store was called after rate limit: provision=%d session=%d", f.store.provisionCalls, f.store.sessionCalls)
	}
}
