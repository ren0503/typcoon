package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
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
		principal, id, err := parseDevEntry(entry)
		if err != nil {
			return nil, err
		}
		allowed[id] = principal
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
