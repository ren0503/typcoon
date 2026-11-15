package auth

import (
	"errors"

	"github.com/google/uuid"
	"github.com/ren0503/tycoon/internal/application/audit"
)

func buildIssueAuditEntry(accountID uuid.UUID, client string) (audit.Entry, error) {
	return buildAuthAuditEntry(authAuditEntry{
		Action:    ActionSessionIssued,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "session",
		Details:   authAuditDetails{Client: client},
	})
}

func buildRefreshAuditEntries(stored RefreshToken) (audit.Entry, audit.Entry, error) {
	refreshed, err := buildAuthAuditEntry(authAuditEntry{
		Action:    ActionSessionRefreshed,
		ActorType: ActorAccount,
		ActorID:   stored.SubjectID.String(),
		AccountID: stored.SubjectID,
		Entity:    "session",
		Details:   authAuditDetails{Client: stored.Client},
	})
	if err != nil {
		return audit.Entry{}, audit.Entry{}, err
	}
	reuse, err := buildAuthAuditEntry(authAuditEntry{
		Action:    ActionSessionReuse,
		ActorType: ActorAccount,
		ActorID:   stored.SubjectID.String(),
		AccountID: stored.SubjectID,
		Entity:    "session",
		Details:   authAuditDetails{Client: stored.Client},
	})
	if err != nil {
		return audit.Entry{}, audit.Entry{}, err
	}
	return refreshed, reuse, nil
}

func buildLogoutAuditEntry(stored RefreshToken, actorType ActorType) (audit.Entry, error) {
	return buildAuthAuditEntry(authAuditEntry{
		Action:    ActionSessionRevoked,
		ActorType: actorType,
		ActorID:   stored.SubjectID.String(),
		AccountID: stored.SubjectID,
		Entity:    "session",
		Details:   authAuditDetails{Client: stored.Client},
	})
}

func buildLogoutAllAuditEntry(accountID uuid.UUID, actorType ActorType) (audit.Entry, error) {
	return buildAuthAuditEntry(authAuditEntry{
		Action:    ActionSessionRevokedAll,
		ActorType: actorType,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "session",
	})
}

func buildLinkIdentityAuditEntry(accountID uuid.UUID, provider string) (audit.Entry, error) {
	return buildAuthAuditEntry(authAuditEntry{
		Action:    ActionIdentityLinked,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "identity",
		Details:   authAuditDetails{Provider: provider},
	})
}

func buildIdentityDeniedAuditEntry(action string, accountID uuid.UUID, provider string, reason AuthAuditReason) (audit.Entry, error) {
	return buildAuthAuditEntry(authAuditEntry{
		Action:    action,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "identity",
		Details:   authAuditDetails{Provider: provider, Reason: reason},
	})
}

func buildUnlinkIdentityAuditEntry(accountID uuid.UUID, provider string) (audit.Entry, error) {
	return buildAuthAuditEntry(authAuditEntry{
		Action:    ActionIdentityUnlinked,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "identity",
		Details:   authAuditDetails{Provider: provider},
	})
}

func mapRefreshError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrRefreshTokenReused) ||
		errors.Is(err, ErrRefreshTokenNotFound) ||
		errors.Is(err, ErrRefreshTokenInvalid) {
		return ErrAuthenticationFailed
	}
	return err
}
