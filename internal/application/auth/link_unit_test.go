package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestExchangeClassifiesVerifiedEmailCollisionWithoutLinking(t *testing.T) {
	store := &fakeServiceStore{provisionErr: ErrEmailConflict}
	f := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{
			"collision-token": {UID: "new-firebase-uid", Email: "same@example.com", EmailVerified: true},
		},
	}, store)

	if _, err := runExchange(t, f, ExchangeInput{IDToken: "collision-token"}); !errors.Is(err, ErrIdentityLinkRequired) {
		t.Fatalf("collision exchange error = %v, want identity link required", err)
	}
	if store.provisionCalls != 1 {
		t.Fatalf("provision calls = %d, want 1", store.provisionCalls)
	}
	if store.sessionCalls != 0 || store.linkCalls != 0 {
		t.Fatalf("collision created a session or link: session=%d link=%d", store.sessionCalls, store.linkCalls)
	}
}

func TestLinkIdentityLinksFreshFirebaseIdentityToAuthenticatedAccount(t *testing.T) {
	accountID := uuid.MustParse("66666666-6666-4666-8666-666666666666")
	store := &fakeServiceStore{}
	f := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{
			"link-token": {UID: "fresh-uid", Email: "fresh@example.com", EmailVerified: true},
		},
	}, store)

	identity, err := f.service.LinkIdentity(context.Background(), accountID, "link-token")
	if err != nil {
		t.Fatalf("LinkIdentity: %v", err)
	}
	if store.linkCalls != 1 || store.linkAccount != accountID {
		t.Fatalf("link store call = %d account %s", store.linkCalls, store.linkAccount)
	}
	if store.linkParams.Provider != firebaseProvider || store.linkParams.ProviderSub != "fresh-uid" ||
		store.linkParams.Email == nil || *store.linkParams.Email != "fresh@example.com" {
		t.Fatalf("link params = %+v", store.linkParams)
	}
	if identity.ID == uuid.Nil || identity.AccountID != accountID || identity.Provider != firebaseProvider {
		t.Fatalf("linked identity = %+v", identity)
	}
	entry := store.linkEntry
	if entry.Action != ActionIdentityLinked || entry.ActorType != string(ActorAccount) ||
		entry.ActorID == nil || *entry.ActorID != accountID.String() {
		t.Fatalf("link audit = %+v", entry)
	}
	if len(f.limiter.calls) != 1 || f.limiter.calls[0].key != "link:account:"+accountID.String() ||
		f.limiter.calls[0].limit != defaultLinkAccountPerMin {
		t.Fatalf("link limiter calls = %+v", f.limiter.calls)
	}
}

func TestLinkIdentityTreatsSameAccountRaceAsIdempotent(t *testing.T) {
	accountID := uuid.New()
	existingID := uuid.New()
	store := &fakeServiceStore{
		linkErr: ErrIdentityConflict,
		identity: &AccountIdentity{
			ID: existingID, AccountID: accountID, Provider: firebaseProvider,
			ProviderSub: "raced-uid",
		},
	}
	f := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{"link-token": {UID: "raced-uid"}},
	}, store)

	identity, err := f.service.LinkIdentity(context.Background(), accountID, "link-token")
	if err != nil {
		t.Fatalf("same-account link race: %v", err)
	}
	if identity.ID != existingID || identity.AccountID != accountID {
		t.Fatalf("idempotent identity = %+v", identity)
	}
}

func TestLinkIdentityRejectsDuplicateOwnershipAndInactiveAccount(t *testing.T) {
	accountID := uuid.New()
	otherAccountID := uuid.New()
	store := &fakeServiceStore{
		linkErr: ErrIdentityConflict,
		identity: &AccountIdentity{
			ID: uuid.New(), AccountID: otherAccountID, Provider: firebaseProvider,
			ProviderSub: "owned-elsewhere",
		},
	}
	f := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{"link-token": {UID: "owned-elsewhere"}},
	}, store)
	if _, err := f.service.LinkIdentity(context.Background(), accountID, "link-token"); !errors.Is(err, ErrIdentityConflict) {
		t.Fatalf("duplicate ownership error = %v", err)
	}

	inactive := &fakeServiceStore{linkErr: ErrAuthenticationFailed}
	inactiveFixture := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{"link-token": {UID: "inactive-uid"}},
	}, inactive)
	if _, err := inactiveFixture.service.LinkIdentity(context.Background(), accountID, "link-token"); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("inactive account link error = %v", err)
	}
}

func TestLinkIdentityRateLimitAndEmptyInput(t *testing.T) {
	f := newAuthFixture(t, &unitVerifier{}, &fakeServiceStore{})
	if _, err := f.service.LinkIdentity(context.Background(), uuid.Nil, "token"); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("nil account link error = %v", err)
	}
	if _, err := f.service.LinkIdentity(context.Background(), uuid.New(), " "); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("empty token link error = %v", err)
	}
	f.limiter.decision = RateLimitDecision{Allowed: false, RetryAfter: 30}
	if _, err := f.service.LinkIdentity(context.Background(), uuid.New(), "token"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("rate-limited link error = %v", err)
	}
	if f.verifier.calls != 0 {
		t.Fatal("rate-limited link reached Firebase verification")
	}
}

func TestUnlinkIdentityMapsLastCredentialAndOwnershipFailures(t *testing.T) {
	accountID := uuid.New()
	identityID := uuid.New()
	store := &fakeServiceStore{unlinkErr: ErrLastIdentity}
	f := newAuthFixture(t, &unitVerifier{}, store)
	if err := f.service.UnlinkIdentity(context.Background(), accountID, identityID); !errors.Is(err, ErrLastIdentity) {
		t.Fatalf("last identity unlink error = %v", err)
	}
	if store.unlinkCalls != 1 || store.unlinkAccount != accountID || store.unlinkedID != identityID {
		t.Fatalf("unlink store call = %d account %s identity %s", store.unlinkCalls, store.unlinkAccount, store.unlinkedID)
	}
	if len(f.limiter.calls) != 1 || f.limiter.calls[0].key != "unlink:account:"+accountID.String() ||
		f.limiter.calls[0].limit != defaultLinkAccountPerMin {
		t.Fatalf("unlink limiter calls = %+v", f.limiter.calls)
	}

	store.unlinkErr = ErrIdentityNotFound
	if err := f.service.UnlinkIdentity(context.Background(), accountID, identityID); !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("not-owned identity unlink error = %v", err)
	}
}

func TestUnlinkIdentitySucceedsForOwnedNonLastIdentity(t *testing.T) {
	accountID := uuid.New()
	identityID := uuid.New()
	store := &fakeServiceStore{}
	f := newAuthFixture(t, &unitVerifier{}, store)
	if err := f.service.UnlinkIdentity(context.Background(), accountID, identityID); err != nil {
		t.Fatalf("UnlinkIdentity: %v", err)
	}
	if store.unlinkEntry.Action != ActionIdentityUnlinked || store.unlinkEntry.ActorID == nil ||
		*store.unlinkEntry.ActorID != accountID.String() {
		t.Fatalf("unlink audit = %+v", store.unlinkEntry)
	}
}
