//go:build integration

package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/platform/testutil"
	"github.com/ren0503/tycoon/internal/transport/http/middleware"
)

type fleetGrant struct {
	AdminID  uuid.UUID
	TenantID uuid.UUID
	ShipIDs  []uuid.UUID
}

func seedLinkedAccount(t *testing.T, h *testutil.Harness, uid string) uuid.UUID {
	t.Helper()
	accountID := uuid.New()
	email := uid + "@example.com"
	if _, err := h.Pool.Exec(context.Background(),
		`INSERT INTO accounts (id, email, display_name, status) VALUES ($1, $2, $3, 'active')`,
		accountID, email, "Fleet User"); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := h.Pool.Exec(context.Background(),
		`INSERT INTO account_identities (id, account_id, provider, provider_sub, email)
		 VALUES ($1, $2, 'firebase', $3, $4)`,
		uuid.New(), accountID, uid, email); err != nil {
		t.Fatalf("seed identity: %v", err)
	}
	return accountID
}

func seedFleetGrant(t *testing.T, h *testutil.Harness, accountID uuid.UUID) fleetGrant {
	t.Helper()
	tenantID := h.SeedTenant(t)
	firstShip := h.SeedShip(t, tenantID)
	secondShip := h.SeedShip(t, tenantID)
	adminID := uuid.New()
	if _, err := h.Pool.Exec(context.Background(), `
		INSERT INTO admin_users (id, tenant_id, account_id, email, display_name, role, status)
		VALUES ($1, $2, $3, $4, $5, 'fleet_manager', 'active')`,
		adminID, tenantID, accountID, adminID.String()+"@example.com", "Fleet Manager"); err != nil {
		t.Fatalf("seed admin grant: %v", err)
	}
	for _, shipID := range []uuid.UUID{firstShip, secondShip} {
		if _, err := h.Pool.Exec(context.Background(),
			`INSERT INTO admin_ship_access (admin_user_id, ship_id) VALUES ($1, $2)`,
			adminID, shipID); err != nil {
			t.Fatalf("seed admin ship access: %v", err)
		}
	}
	return fleetGrant{AdminID: adminID, TenantID: tenantID, ShipIDs: []uuid.UUID{firstShip, secondShip}}
}

func exchangeAs(t *testing.T, f exchangeFixture, token, client string) (auth.IssuedSession, error) {
	t.Helper()
	return f.service.Exchange(context.Background(), auth.ExchangeInput{
		IDToken:   token,
		Client:    client,
		UserAgent: client + "/1.0",
		IPAddress: "192.0.2.30",
	})
}

func authenticate(t *testing.T, f exchangeFixture, token string) (auth.Principal, error) {
	t.Helper()
	provider, err := middleware.NewAccessTokenProvider(f.validator, f.store)
	if err != nil {
		t.Fatalf("NewAccessTokenProvider: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/ships", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	return provider.Authenticate(request)
}

func TestFleetExchangeWithoutGrantIsDenied(t *testing.T) {
	h := testutil.StartPostgres(t)
	accountID := seedLinkedAccount(t, h, "fleet-no-grant")
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{
			"fleet-token": {UID: "fleet-no-grant"},
			"unknown":     {UID: "fleet-unknown"},
		},
	})

	if _, err := exchangeAs(t, f, "fleet-token", auth.ClientFleetConsole); !errors.Is(err, auth.ErrClientForbidden) {
		t.Fatalf("Fleet exchange without grant error = %v", err)
	}
	if _, err := exchangeAs(t, f, "unknown", auth.ClientFleetConsole); !errors.Is(err, auth.ErrClientForbidden) {
		t.Fatalf("unknown Fleet exchange error = %v", err)
	}
	assertAuthRowCounts(t, h, 1, 1, 0)
	assertAuditAction(t, f, auth.ActionAdminAccessDenied)

	var linkedAccount uuid.UUID
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT account_id FROM account_identities WHERE provider = 'firebase' AND provider_sub = 'fleet-no-grant'`).
		Scan(&linkedAccount); err != nil {
		t.Fatalf("read linked account: %v", err)
	}
	if linkedAccount != accountID {
		t.Fatalf("existing identity was relinked to %s, want %s", linkedAccount, accountID)
	}
}

func TestFleetExchangeRefreshAndPrincipalUseBackendGrant(t *testing.T) {
	h := testutil.StartPostgres(t)
	accountID := seedLinkedAccount(t, h, "fleet-granted")
	grant := seedFleetGrant(t, h, accountID)
	f := newExchangeFixture(t, h, &testutil.FakeVerifier{
		Identities: map[string]auth.VerifiedIdentity{
			"fleet-token": {UID: "fleet-granted"},
			"guest-token": {UID: "fleet-granted"},
		},
	})

	fleetSession, err := exchangeAs(t, f, "fleet-token", auth.ClientFleetConsole)
	if err != nil {
		t.Fatalf("Fleet exchange: %v", err)
	}
	if fleetSession.AccountID != accountID || fleetSession.Client != auth.ClientFleetConsole {
		t.Fatalf("Fleet session = %+v", fleetSession)
	}
	claims, err := f.validator.Validate(context.Background(), fleetSession.AccessToken)
	if err != nil {
		t.Fatalf("validate Fleet access token: %v", err)
	}
	if claims.Client != auth.ClientFleetConsole || claims.Subject != accountID.String() {
		t.Fatalf("Fleet claims = client %q subject %q", claims.Client, claims.Subject)
	}
	principal, err := authenticate(t, f, fleetSession.AccessToken)
	if err != nil {
		t.Fatalf("authenticate Fleet token: %v", err)
	}
	if principal.ActorType != auth.ActorAdmin || principal.ActorID != grant.AdminID.String() ||
		principal.AccountID != accountID || principal.Role != "fleet_manager" || principal.TenantID != grant.TenantID {
		t.Fatalf("Fleet principal = %+v", principal)
	}
	if len(principal.ShipIDs) != 2 || principal.ShipIDs[0] != grant.ShipIDs[0] || principal.ShipIDs[1] != grant.ShipIDs[1] {
		t.Fatalf("Fleet ship scope = %v", principal.ShipIDs)
	}

	guestSession, err := exchangeAs(t, f, "guest-token", auth.ClientGuestMobile)
	if err != nil {
		t.Fatalf("guest exchange: %v", err)
	}
	guestPrincipal, err := authenticate(t, f, guestSession.AccessToken)
	if err != nil {
		t.Fatalf("authenticate guest token: %v", err)
	}
	if guestPrincipal.ActorType != auth.ActorAccount || guestPrincipal.ActorID != accountID.String() || guestPrincipal.AccountID != accountID {
		t.Fatalf("guest principal = %+v", guestPrincipal)
	}
	if guestPrincipal.TenantID != uuid.Nil || guestPrincipal.Role != "" || len(guestPrincipal.ShipIDs) != 0 {
		t.Fatalf("guest token received Fleet authority: %+v", guestPrincipal)
	}
	if guestPrincipal.CanAccessTenant(grant.TenantID) {
		t.Fatal("guest token accessed the Fleet tenant")
	}

	f.clock.Advance(time.Minute)
	rotated, err := refreshSession(t, f, fleetSession.RefreshToken)
	if err != nil {
		t.Fatalf("Fleet refresh: %v", err)
	}
	if rotated.Client != auth.ClientFleetConsole || rotated.SessionID != fleetSession.SessionID || rotated.AccountID != accountID {
		t.Fatalf("rotated Fleet session = %+v", rotated)
	}
	var client string
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT client FROM refresh_tokens WHERE family_id = $1 AND used_at IS NULL`,
		rotated.SessionID).Scan(&client); err != nil {
		t.Fatalf("read Fleet refresh family: %v", err)
	}
	if client != auth.ClientFleetConsole {
		t.Fatalf("replacement refresh client = %q", client)
	}
}

func TestFleetGrantSuspensionBlocksFleetButNotGuestRefresh(t *testing.T) {
	for _, state := range []string{"suspended", "deleted"} {
		t.Run(state, func(t *testing.T) {
			h := testutil.StartPostgres(t)
			accountID := seedLinkedAccount(t, h, "fleet-grant-"+state)
			grant := seedFleetGrant(t, h, accountID)
			f := newExchangeFixture(t, h, &testutil.FakeVerifier{
				Identities: map[string]auth.VerifiedIdentity{"token": {UID: "fleet-grant-" + state}},
			})
			fleetSession, err := exchangeAs(t, f, "token", auth.ClientFleetConsole)
			if err != nil {
				t.Fatalf("Fleet exchange: %v", err)
			}
			guestSession, err := exchangeAs(t, f, "token", auth.ClientGuestMobile)
			if err != nil {
				t.Fatalf("guest exchange: %v", err)
			}
			if state == "suspended" {
				_, err = h.Pool.Exec(context.Background(), `UPDATE admin_users SET status = 'suspended' WHERE id = $1`, grant.AdminID)
			} else {
				_, err = h.Pool.Exec(context.Background(), `UPDATE admin_users SET deleted_at = now() WHERE id = $1`, grant.AdminID)
			}
			if err != nil {
				t.Fatalf("update grant: %v", err)
			}

			if _, err := exchangeAs(t, f, "token", auth.ClientFleetConsole); !errors.Is(err, auth.ErrClientForbidden) {
				t.Fatalf("Fleet exchange after %s grant error = %v", state, err)
			}
			if _, err := authenticate(t, f, fleetSession.AccessToken); !errors.Is(err, middleware.ErrUnauthenticated) {
				t.Fatalf("Fleet token after %s grant authenticate error = %v", state, err)
			}
			if _, err := refreshSession(t, f, fleetSession.RefreshToken); !errors.Is(err, auth.ErrAuthenticationFailed) {
				t.Fatalf("Fleet refresh after %s grant error = %v", state, err)
			}
			if _, err := refreshSession(t, f, guestSession.RefreshToken); err != nil {
				t.Fatalf("guest refresh after %s Fleet grant error = %v", state, err)
			}
		})
	}
}

func TestFleetAccountSuspensionBlocksExchangeAndRefresh(t *testing.T) {
	for _, state := range []string{"suspended", "deleted"} {
		t.Run(state, func(t *testing.T) {
			h := testutil.StartPostgres(t)
			accountID := seedLinkedAccount(t, h, "fleet-account-"+state)
			seedFleetGrant(t, h, accountID)
			f := newExchangeFixture(t, h, &testutil.FakeVerifier{
				Identities: map[string]auth.VerifiedIdentity{"token": {UID: "fleet-account-" + state}},
			})
			fleetSession, err := exchangeAs(t, f, "token", auth.ClientFleetConsole)
			if err != nil {
				t.Fatalf("Fleet exchange: %v", err)
			}
			if state == "suspended" {
				_, err = h.Pool.Exec(context.Background(), `UPDATE accounts SET status = 'suspended' WHERE id = $1`, accountID)
			} else {
				_, err = h.Pool.Exec(context.Background(), `UPDATE accounts SET status = 'deleted', deleted_at = now() WHERE id = $1`, accountID)
			}
			if err != nil {
				t.Fatalf("update account: %v", err)
			}

			if _, err := exchangeAs(t, f, "token", auth.ClientFleetConsole); !errors.Is(err, auth.ErrAuthenticationFailed) {
				t.Fatalf("Fleet exchange after %s account error = %v", state, err)
			}
			if _, err := refreshSession(t, f, fleetSession.RefreshToken); !errors.Is(err, auth.ErrAuthenticationFailed) {
				t.Fatalf("Fleet refresh after %s account error = %v", state, err)
			}
		})
	}
}
