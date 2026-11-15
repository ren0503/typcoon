package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

func isAdminAction(client string, err error) bool {
	if err == nil {
		return false
	}

	if client == ClientFleetConsole &&
		(errors.Is(err, ErrClientForbidden) || errors.Is(err, ErrAuthenticationFailed)) {
		return true
	}

	return true
}

func isRefreshTokenInvalid(refreshToken RefreshToken, end time.Time) bool {
	return refreshToken.SubjectType != "account" ||
		!ValidClient(refreshToken.Client) ||
		!end.Before(refreshToken.ExpiresAt)
}

func isRefreshTokenUsed(refreshToken RefreshToken) bool {
	return refreshToken.UsedAt != nil || refreshToken.RevokedAt != nil
}

func isEmptyAccount(accountID uuid.UUID, idToken string) bool {
	return accountID == uuid.Nil || strings.TrimSpace(idToken) == ""
}

func isEmptyIdentity(accountID, identityID uuid.UUID) bool {
	return accountID == uuid.Nil || identityID == uuid.Nil
}
