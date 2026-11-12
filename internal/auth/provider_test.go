package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type fakeAccountLookup struct {
	account   sqlc.Account
	err       error
	authority AdminAuthority
	adminErr  error
}

func (f *fakeAccountLookup) GetAccount(context.Context, uuid.UUID) (sqlc.Account, error) {
	return f.account, f.err
}

func (f *fakeAccountLookup) ResolveActiveAdmin(context.Context, uuid.UUID) (AdminAuthority, error) {
	if f.adminErr != nil {
		return AdminAuthority{}, f.adminErr
	}
	return f.authority, nil
}

func TestAccessTokenProviderAuthenticatesActiveAccount(t *testing.T) {
	issuer, _ := newTokenIssuer(t)
	accountID := uuid.New()
	raw, _, err := issuer.Issue(context.Background(), accountID, ClientGuestMobile)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	provider, err := NewAccessTokenProvider(issuer, &fakeAccountLookup{account: sqlc.Account{ID: accountID, Status: "active"}})
	if err != nil {
		t.Fatalf("NewAccessTokenProvider: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/bookings", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	principal, err := provider.Authenticate(request)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if principal.ActorType != ActorAccount || principal.ActorID != accountID.String() || principal.AccountID != accountID {
		t.Fatalf("principal = %+v", principal)
	}
}

func TestAccessTokenProviderRejectsMissingInvalidAndInactive(t *testing.T) {
	issuer, _ := newTokenIssuer(t)
	accountID := uuid.New()
	raw, _, err := issuer.Issue(context.Background(), accountID, ClientGuestMobile)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	provider, err := NewAccessTokenProvider(issuer, &fakeAccountLookup{account: sqlc.Account{ID: accountID, Status: "suspended"}})
	if err != nil {
		t.Fatalf("NewAccessTokenProvider: %v", err)
	}
	cases := []struct {
		name   string
		header string
	}{
		{name: "missing"},
		{name: "malformed", header: "Token " + raw},
		{name: "invalid", header: "Bearer not-a-token"},
		{name: "suspended", header: "Bearer " + raw},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/bookings", nil)
			if tc.header != "" {
				request.Header.Set("Authorization", tc.header)
			}
			if _, err := provider.Authenticate(request); !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestAccessTokenProviderMapsFleetGrantFromBackend(t *testing.T) {
	issuer, _ := newTokenIssuer(t)
	accountID := uuid.New()
	adminID := uuid.New()
	tenantID := uuid.New()
	firstShip := uuid.New()
	secondShip := uuid.New()
	raw, _, err := issuer.Issue(context.Background(), accountID, ClientFleetConsole)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	provider, err := NewAccessTokenProvider(issuer, &fakeAccountLookup{
		account: sqlc.Account{ID: accountID, Status: "active"},
		authority: AdminAuthority{
			AdminUserID: adminID,
			Role:        "fleet_manager",
			TenantID:    tenantID,
			ShipIDs:     []uuid.UUID{firstShip, secondShip},
		},
	})
	if err != nil {
		t.Fatalf("NewAccessTokenProvider: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/ships", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	principal, err := provider.Authenticate(request)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if principal.ActorType != ActorAdmin || principal.ActorID != adminID.String() || principal.AccountID != accountID || principal.Dev {
		t.Fatalf("principal actor = %+v", principal)
	}
	if principal.Role != "fleet_manager" || principal.TenantID != tenantID {
		t.Fatalf("principal role/tenant = %q/%s", principal.Role, principal.TenantID)
	}
	if len(principal.ShipIDs) != 2 || principal.ShipIDs[0] != firstShip || principal.ShipIDs[1] != secondShip {
		t.Fatalf("principal ships = %v", principal.ShipIDs)
	}
	if !principal.CanAccessTenant(tenantID) || !principal.CanAccessShip(firstShip) {
		t.Fatal("Fleet principal cannot access its backend scope")
	}
	if principal.CanAccessTenant(uuid.New()) || principal.CanAccessShip(uuid.New()) {
		t.Fatal("Fleet principal escaped its backend scope")
	}
}

func TestAccessTokenProviderGuestCannotElevateToFleet(t *testing.T) {
	issuer, _ := newTokenIssuer(t)
	accountID := uuid.New()
	raw, _, err := issuer.Issue(context.Background(), accountID, ClientGuestMobile)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	provider, err := NewAccessTokenProvider(issuer, &fakeAccountLookup{
		account: sqlc.Account{ID: accountID, Status: "active"},
		authority: AdminAuthority{
			AdminUserID: uuid.New(),
			Role:        "super_admin",
			TenantID:    uuid.New(),
			ShipIDs:     []uuid.UUID{uuid.New()},
		},
	})
	if err != nil {
		t.Fatalf("NewAccessTokenProvider: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/ships", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	principal, err := provider.Authenticate(request)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if principal.ActorType != ActorAccount || principal.ActorID != accountID.String() || principal.AccountID != accountID {
		t.Fatalf("principal = %+v", principal)
	}
	if principal.TenantID != uuid.Nil || principal.Role != "" || len(principal.ShipIDs) != 0 {
		t.Fatalf("guest principal received Fleet context: %+v", principal)
	}
	if principal.CanAccessTenant(uuid.New()) {
		t.Fatal("guest principal gained tenant access")
	}
}

func TestAccessTokenProviderFleetRequiresActiveGrant(t *testing.T) {
	for _, adminErr := range []error{ErrAdminGrantNotFound, errors.New("admin lookup failed")} {
		issuer, _ := newTokenIssuer(t)
		accountID := uuid.New()
		raw, _, err := issuer.Issue(context.Background(), accountID, ClientFleetConsole)
		if err != nil {
			t.Fatalf("Issue: %v", err)
		}
		provider, err := NewAccessTokenProvider(issuer, &fakeAccountLookup{
			account:  sqlc.Account{ID: accountID, Status: "active"},
			adminErr: adminErr,
		})
		if err != nil {
			t.Fatalf("NewAccessTokenProvider: %v", err)
		}
		request := httptest.NewRequest(http.MethodGet, "/v1/ships", nil)
		request.Header.Set("Authorization", "Bearer "+raw)
		if _, err := provider.Authenticate(request); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("grant error %v produced authenticate error %v", adminErr, err)
		}
	}
}

func TestDevHeaderProviderAcceptsConfiguredDescriptor(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	descriptor := "admin:dev-1:" + tenantID.String()
	provider, err := NewDevHeaderProvider([]string{descriptor})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Dev-Principal", descriptor)
	principal, err := provider.Authenticate(req)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if principal.ActorType != ActorAdmin || principal.ActorID != "dev-1" || principal.TenantID != tenantID || !principal.Dev {
		t.Fatalf("principal = %+v", principal)
	}
}
