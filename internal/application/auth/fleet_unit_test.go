package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestExchangeFleetRequiresExistingGrantAndNeverProvisions(t *testing.T) {
	account := Account{ID: uuid.New(), Status: "active"}
	identity := &AccountIdentity{
		ID: uuid.New(), AccountID: account.ID, Provider: firebaseProvider, ProviderSub: "fleet-uid",
	}
	store := &fakeServiceStore{
		identity: identity,
		account:  account,
		adminAuthority: AdminAuthority{
			AdminUserID: uuid.New(),
			Role:        "fleet_manager",
			TenantID:    uuid.New(),
		},
	}
	f := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{"fleet": {UID: "fleet-uid"}},
	}, store)

	session, err := f.service.Exchange(context.Background(), ExchangeInput{
		IDToken: "fleet", Client: ClientFleetConsole, IPAddress: "192.0.2.40",
	})
	if err != nil {
		t.Fatalf("Fleet exchange: %v", err)
	}
	if session.Client != ClientFleetConsole || session.AccountID != account.ID {
		t.Fatalf("Fleet session = %+v", session)
	}
	if len(store.sessions) != 1 || store.sessions[0].Client != ClientFleetConsole {
		t.Fatalf("Fleet refresh rows = %+v", store.sessions)
	}
	if store.provisionCalls != 0 || store.adminCalls != 1 {
		t.Fatalf("store calls = provision %d admin %d", store.provisionCalls, store.adminCalls)
	}

	noGrant := &fakeServiceStore{
		identity: identity,
		account:  account,
		adminErr: ErrAdminGrantNotFound,
	}
	noGrantFixture := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{"fleet": {UID: "fleet-uid"}},
	}, noGrant)
	if _, err := noGrantFixture.service.Exchange(context.Background(), ExchangeInput{
		IDToken: "fleet", Client: ClientFleetConsole, IPAddress: "192.0.2.41",
	}); !errors.Is(err, ErrClientForbidden) {
		t.Fatalf("Fleet exchange without grant error = %v", err)
	}
	if noGrant.sessionCalls != 0 || noGrant.provisionCalls != 0 {
		t.Fatalf("no-grant exchange wrote data: sessions=%d provision=%d", noGrant.sessionCalls, noGrant.provisionCalls)
	}

	unknown := &fakeServiceStore{}
	unknownFixture := newAuthFixture(t, &unitVerifier{
		identities: map[string]VerifiedIdentity{"fleet": {UID: "unlinked-fleet-uid"}},
	}, unknown)
	if _, err := unknownFixture.service.Exchange(context.Background(), ExchangeInput{
		IDToken: "fleet", Client: ClientFleetConsole, IPAddress: "192.0.2.42",
	}); !errors.Is(err, ErrClientForbidden) {
		t.Fatalf("unlinked Fleet exchange error = %v", err)
	}
	if unknown.provisionCalls != 0 || unknown.sessionCalls != 0 || unknown.adminCalls != 0 {
		t.Fatalf("unlinked Fleet exchange wrote data: %+v", unknown)
	}
}

func TestRefreshFleetRequiresActiveGrant(t *testing.T) {
	const rawToken = "fleet-refresh-token"
	account := Account{ID: uuid.New(), Status: "active"}
	row := &RefreshToken{
		ID:          uuid.New(),
		SubjectType: "account",
		SubjectID:   account.ID,
		FamilyID:    uuid.New(),
		Client:      ClientFleetConsole,
		ExpiresAt:   time.Date(2026, time.November, 5, 13, 0, 0, 0, time.UTC),
		TokenHash:   HashRefreshToken(rawToken),
	}
	f, store := newRefreshFixture(t, row, account)
	store.adminAuthority = AdminAuthority{AdminUserID: uuid.New(), Role: "fleet_manager", TenantID: uuid.New()}

	if _, err := f.service.Refresh(context.Background(), RefreshInput{RefreshToken: rawToken}); err != nil {
		t.Fatalf("Fleet refresh: %v", err)
	}
	if store.adminCalls != 1 || store.rotateCalls != 1 {
		t.Fatalf("Fleet refresh calls = admin %d rotate %d", store.adminCalls, store.rotateCalls)
	}

	suspendedStore := &fakeServiceStore{
		account:        account,
		refreshToken:   row,
		rotatedAccount: account,
		adminErr:       ErrAdminGrantNotFound,
	}
	suspendedFixture := newAuthFixture(t, &unitVerifier{}, suspendedStore)
	if _, err := suspendedFixture.service.Refresh(context.Background(), RefreshInput{RefreshToken: rawToken}); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("Fleet refresh without active grant error = %v", err)
	}
	if suspendedStore.rotateCalls != 0 {
		t.Fatalf("Fleet refresh without active grant reached rotation %d times", suspendedStore.rotateCalls)
	}
}
