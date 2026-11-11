package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/database/sqlc"
)

// ErrUnauthenticated indicates that no valid principal could be resolved.
var ErrUnauthenticated = errors.New("unauthenticated")

// Provider resolves the caller's Principal from an HTTP request.
type Provider interface {
	Authenticate(r *http.Request) (Principal, error)
}

// DenyAllProvider rejects every request. It is the safe default until the
// token-issuing workflows from a later feature exist.
type DenyAllProvider struct{}

// Authenticate always fails.
func (DenyAllProvider) Authenticate(*http.Request) (Principal, error) {
	return Principal{}, ErrUnauthenticated
}

// DevHeaderProvider resolves principals from explicit development headers. It
// only accepts principals present in the configured allowlist; the config
// layer prevents it from activating in production.
type DevHeaderProvider struct {
	allowed map[string]Principal
}

// NewDevHeaderProvider builds a provider from an allowlist of principal
// descriptors. Each descriptor uses the form
// "<actorType>:<actorID>:<tenantID>" with optional ":<shipID>,<shipID>".
func NewDevHeaderProvider(entries []string) (*DevHeaderProvider, error) {
	allowed := make(map[string]Principal, len(entries))
	for _, entry := range entries {
		principal, _, err := parseDevEntry(entry)
		if err != nil {
			return nil, err
		}
		allowed[strings.TrimSpace(entry)] = principal
	}
	return &DevHeaderProvider{allowed: allowed}, nil
}

func (p *DevHeaderProvider) Authenticate(r *http.Request) (Principal, error) {
	id := strings.TrimSpace(r.Header.Get("X-Dev-Principal"))
	if id == "" {
		return Principal{}, ErrUnauthenticated
	}
	principal, ok := p.allowed[id]
	if !ok {
		return Principal{}, ErrUnauthenticated
	}
	principal.Dev = true
	return principal, nil
}

func parseDevEntry(entry string) (Principal, string, error) {
	parts := strings.Split(entry, ":")
	if len(parts) < 3 || len(parts) > 4 {
		return Principal{}, "", errors.New("invalid dev principal entry")
	}
	actorType := ActorType(strings.TrimSpace(parts[0]))
	switch actorType {
	case ActorAdmin, ActorAccount, ActorStaff, ActorSystem:
	default:
		return Principal{}, "", errors.New("invalid dev principal actor type")
	}
	actorID := strings.TrimSpace(parts[1])
	if actorID == "" {
		return Principal{}, "", errors.New("invalid dev principal actor id")
	}
	tenantID, err := uuid.Parse(strings.TrimSpace(parts[2]))
	if err != nil {
		return Principal{}, "", errors.New("invalid dev principal tenant id")
	}
	principal := Principal{ActorType: actorType, ActorID: actorID, TenantID: tenantID}
	if len(parts) == 4 && strings.TrimSpace(parts[3]) != "" {
		for _, raw := range strings.Split(parts[3], ",") {
			shipID, err := uuid.Parse(strings.TrimSpace(raw))
			if err != nil {
				return Principal{}, "", errors.New("invalid dev principal ship id")
			}
			principal.ShipIDs = append(principal.ShipIDs, shipID)
		}
	}
	return principal, actorID, nil
}

type accountLookup interface {
	GetAccount(ctx context.Context, id uuid.UUID) (sqlc.Account, error)
	ResolveActiveAdmin(ctx context.Context, accountID uuid.UUID) (AdminAuthority, error)
}

// AccessTokenProvider resolves principals from backend-issued access tokens.
type AccessTokenProvider struct {
	tokens   *AccessTokenIssuer
	accounts accountLookup
}

// NewAccessTokenProvider builds the bearer access-token provider.
func NewAccessTokenProvider(tokens *AccessTokenIssuer, accounts accountLookup) (*AccessTokenProvider, error) {
	if tokens == nil {
		return nil, errors.New("access token issuer is required")
	}
	if accounts == nil {
		return nil, errors.New("account lookup is required")
	}
	return &AccessTokenProvider{tokens: tokens, accounts: accounts}, nil
}

// Authenticate validates the bearer token and resolves the active account.
func (p *AccessTokenProvider) Authenticate(r *http.Request) (Principal, error) {
	raw, ok := bearerToken(r)
	if !ok {
		return Principal{}, ErrUnauthenticated
	}
	claims, err := p.tokens.Validate(r.Context(), raw)
	if err != nil {
		return Principal{}, ErrUnauthenticated
	}
	accountID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return Principal{}, ErrUnauthenticated
	}
	account, err := p.accounts.GetAccount(r.Context(), accountID)
	if err != nil || account.Status != "active" || account.DeletedAt != nil {
		return Principal{}, ErrUnauthenticated
	}
	switch claims.Client {
	case ClientGuestMobile:
		return Principal{ActorType: ActorAccount, ActorID: account.ID.String(), AccountID: account.ID}, nil
	case ClientFleetConsole:
		authority, err := p.accounts.ResolveActiveAdmin(r.Context(), accountID)
		if err != nil {
			return Principal{}, ErrUnauthenticated
		}
		return Principal{
			ActorType: ActorAdmin,
			ActorID:   authority.AdminUserID.String(),
			AccountID: accountID,
			TenantID:  authority.TenantID,
			Role:      authority.Role,
			ShipIDs:   authority.ShipIDs,
		}, nil
	default:
		return Principal{}, ErrUnauthenticated
	}
}

func bearerToken(r *http.Request) (string, bool) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
