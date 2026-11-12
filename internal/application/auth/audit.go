package auth

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/audit"
	"github.com/ren0503/tycoon/internal/platform/errors"
)

const (
	ActionIdentityProvisioned  = "auth.identity.provisioned"
	ActionIdentityLinked       = "auth.identity.linked"
	ActionIdentityUnlinked     = "auth.identity.unlinked"
	ActionIdentityLinkDenied   = "auth.identity.link_rejected"
	ActionIdentityUnlinkDenied = "auth.identity.unlink_rejected"
	ActionSessionIssued        = "auth.session.issued"
	ActionSessionRefreshed     = "auth.session.refreshed"
	ActionSessionReuse         = "auth.session.reuse_detected"
	ActionSessionRevoked       = "auth.session.revoked"
	ActionSessionRevokedAll    = "auth.session.revoked_all"
	ActionAdminAccessDenied    = "auth.admin.access_denied"
)

// AuthAuditReason is a closed set of non-sensitive rejection reasons.
type AuthAuditReason string

const (
	ReasonConflict        AuthAuditReason = "conflict"
	ReasonLastIdentity    AuthAuditReason = "last_identity"
	ReasonAccountInactive AuthAuditReason = "account_inactive"
	ReasonNotOwner        AuthAuditReason = "not_owner"
	ReasonNoActiveGrant   AuthAuditReason = "no_active_grant"
)

// authAuditDetails is deliberately limited to non-secret metadata.
type authAuditDetails struct {
	Client   string          `json:"client,omitempty"`
	Provider string          `json:"provider,omitempty"`
	Reason   AuthAuditReason `json:"reason,omitempty"`
}

type authAuditEntry struct {
	Action    string
	ActorType ActorType
	ActorID   string
	AccountID uuid.UUID
	Entity    string
	Details   authAuditDetails
}

func buildAuthAuditEntry(event authAuditEntry) (audit.Entry, error) {
	details, err := json.Marshal(event.Details)
	if err != nil {
		return audit.Entry{}, apperrors.Internal(err)
	}
	entry := audit.Entry{
		Source:    "cloud",
		ActorType: string(event.ActorType),
		Action:    event.Action,
		Details:   details,
	}
	if event.ActorID != "" {
		actorID := event.ActorID
		entry.ActorID = &actorID
	}
	if event.Entity != "" {
		entity := event.Entity
		entry.Entity = &entity
	}
	if event.AccountID != uuid.Nil {
		entityID := event.AccountID.String()
		entry.EntityID = &entityID
	}
	return entry, nil
}
