package auth

import (
	"context"

	"github.com/google/uuid"
)

// ActorType identifies who performed an action for audit purposes.
type ActorType string

const (
	ActorAdmin   ActorType = "admin"
	ActorAccount ActorType = "account"
	ActorStaff   ActorType = "staff"
	ActorHub     ActorType = "hub"
	ActorSystem  ActorType = "system"
	ActorAI      ActorType = "ai"
)

// Principal is the authenticated caller resolved by a Provider.
type Principal struct {
	ActorType ActorType
	ActorID   string
	AccountID uuid.UUID
	TenantID  uuid.UUID
	Role      string
	ShipIDs   []uuid.UUID
	Dev       bool
}

// CanAccessTenant reports whether the principal may act within the tenant.
func (p Principal) CanAccessTenant(tenantID uuid.UUID) bool {
	if p.ActorType == ActorSystem {
		return true
	}
	if p.TenantID == uuid.Nil {
		return false
	}
	return p.TenantID == tenantID
}

// CanAccessShip reports whether the principal may act on the ship. A principal
// with no explicit ship list is tenant-wide (subject to the tenant check).
func (p Principal) CanAccessShip(shipID uuid.UUID) bool {
	if p.ActorType == ActorSystem {
		return true
	}
	if p.TenantID == uuid.Nil {
		return false
	}
	if len(p.ShipIDs) == 0 {
		return true
	}
	for _, allowed := range p.ShipIDs {
		if allowed == shipID {
			return true
		}
	}
	return false
}

type contextKey struct{}

// WithPrincipal stores the authenticated principal in the context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, p)
}

// PrincipalFrom returns the principal stored in the context, if any.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(contextKey{}).(Principal)
	return p, ok
}
