package auth

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLogoutHashesTokenAndAuditsFamilyRevocation(t *testing.T) {
	accountID := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	store := &fakeServiceStore{refreshToken: &RefreshToken{
		SubjectID:   accountID,
		SubjectType: "account",
		Client:      ClientGuestMobile,
	}}
	f := newAuthFixture(t, &unitVerifier{}, store)
	token := "presented-refresh-token"

	if err := f.service.Logout(context.Background(), token); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if store.logoutCalls != 1 {
		t.Fatalf("revoke calls = %d, want 1", store.logoutCalls)
	}
	if string(store.logoutHash) != string(HashRefreshToken(token)) {
		t.Fatal("logout did not hash the presented refresh token")
	}
	entry := store.logoutEntry
	if entry.Action != ActionSessionRevoked || entry.ActorType != string(ActorAccount) {
		t.Fatalf("logout audit = %+v", entry)
	}
	if entry.ActorID == nil || *entry.ActorID != accountID.String() ||
		entry.EntityID == nil || *entry.EntityID != accountID.String() {
		t.Fatalf("logout audit identity = %+v", entry)
	}
	if entry.Entity == nil || *entry.Entity != "session" {
		t.Fatalf("logout audit entity = %v", entry.Entity)
	}
	var details authAuditDetails
	if err := json.Unmarshal(entry.Details, &details); err != nil {
		t.Fatalf("decode audit details: %v", err)
	}
	if details.Client != ClientGuestMobile {
		t.Fatalf("logout audit client = %q", details.Client)
	}
	if f.verifier.calls != 0 {
		t.Fatalf("logout called Firebase verifier %d times", f.verifier.calls)
	}
}

func TestLogoutTreatsUnknownTokenAsIdempotentSuccess(t *testing.T) {
	store := &fakeServiceStore{refreshLookupErr: ErrRefreshTokenNotFound}
	f := newAuthFixture(t, &unitVerifier{}, store)

	if err := f.service.Logout(context.Background(), "unknown-token"); err != nil {
		t.Fatalf("Logout unknown token: %v", err)
	}
	if store.logoutCalls != 0 {
		t.Fatalf("revoke calls = %d, want 0", store.logoutCalls)
	}
}

func TestLogoutPropagatesLookupFailure(t *testing.T) {
	lookupErr := errors.New("database unavailable")
	store := &fakeServiceStore{refreshLookupErr: lookupErr}
	f := newAuthFixture(t, &unitVerifier{}, store)

	if err := f.service.Logout(context.Background(), "token"); !errors.Is(err, lookupErr) {
		t.Fatalf("Logout error = %v, want lookup failure", err)
	}
}

func TestLogoutAllRateLimitsAccountAndAuditsRevocation(t *testing.T) {
	accountID := uuid.MustParse("55555555-5555-4555-8555-555555555555")
	store := &fakeServiceStore{}
	f := newAuthFixture(t, &unitVerifier{}, store)
	f.limiter.decision = RateLimitDecision{Allowed: true}

	if err := f.service.LogoutAll(context.Background(), accountID); err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}
	if len(f.limiter.calls) != 1 {
		t.Fatalf("limiter calls = %+v", f.limiter.calls)
	}
	call := f.limiter.calls[0]
	if call.key != "logout-all:account:"+accountID.String() || call.limit != defaultLogoutAllAccountPerMin || call.window != rateLimitWindow {
		t.Fatalf("limiter call = %+v", call)
	}
	if store.logoutAllCalls != 1 || store.logoutAllAccount != accountID {
		t.Fatalf("logout-all store call = %d account %s", store.logoutAllCalls, store.logoutAllAccount)
	}
	entry := store.logoutAllEntry
	if entry.Action != ActionSessionRevokedAll || entry.ActorType != string(ActorAccount) {
		t.Fatalf("logout-all audit = %+v", entry)
	}
	if entry.ActorID == nil || *entry.ActorID != accountID.String() ||
		entry.EntityID == nil || *entry.EntityID != accountID.String() {
		t.Fatalf("logout-all audit identity = %+v", entry)
	}

	store.logoutAllCalls = 0
	f.limiter.calls = nil
	f.limiter.decision = RateLimitDecision{Allowed: false, RetryAfter: 30 * time.Second}
	if err := f.service.LogoutAll(context.Background(), accountID); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("rate-limited LogoutAll error = %v", err)
	}
	if store.logoutAllCalls != 0 {
		t.Fatal("rate-limited logout-all reached the store")
	}
}

func TestLogoutAllRejectsMissingAccountAndLimiterFailure(t *testing.T) {
	f := newAuthFixture(t, &unitVerifier{}, &fakeServiceStore{})
	if err := f.service.LogoutAll(context.Background(), uuid.Nil); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("nil account error = %v", err)
	}
	if len(f.limiter.calls) != 0 {
		t.Fatal("nil account reached the limiter")
	}

	accountID := uuid.New()
	f.limiter.err = ErrRateLimiterUnavailable
	if err := f.service.LogoutAll(context.Background(), accountID); !errors.Is(err, ErrRateLimiterUnavailable) {
		t.Fatalf("limiter failure error = %v", err)
	}
}
