package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/audit"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

const (
	firebaseProvider = "firebase"
	defaultLocale    = "en"
	rateLimitWindow  = time.Minute
)

var (
	// ErrAuthenticationFailed deliberately hides why an exchange credential failed.
	ErrAuthenticationFailed = errors.New("authentication failed")
	// ErrIdentityLinkRequired means a verified email already belongs to another account.
	ErrIdentityLinkRequired = errors.New("identity link required")
	// ErrClientForbidden means the requested session class is not authorized.
	ErrClientForbidden = errors.New("auth client forbidden")
	// ErrRateLimited indicates an authentication abuse limit was exceeded.
	ErrRateLimited = errors.New("auth rate limited")
	// ErrSessionUnavailable indicates signing or key material was unavailable.
	ErrSessionUnavailable = errors.New("auth session issuer unavailable")
	// ErrLastIdentity means unlinking would leave the account without a sign-in method.
	ErrLastIdentity = errors.New("last sign-in identity")
)

// RateLimitedError carries the stable retry indication for a 429 response.
type RateLimitedError struct {
	RetryAfter time.Duration
}

// Error implements error without exposing limiter internals.
func (e *RateLimitedError) Error() string {
	return "authentication rate limited"
}

// Is allows errors.Is(err, ErrRateLimited).
func (e *RateLimitedError) Is(target error) bool {
	return target == ErrRateLimited
}

type serviceStore interface {
	GetIdentityByProviderSubject(ctx context.Context, provider, subject string) (sqlc.AccountIdentity, error)
	GetAccount(ctx context.Context, id uuid.UUID) (sqlc.Account, error)
	ProvisionGuest(ctx context.Context, account sqlc.CreateAccountParams, identity sqlc.LinkAccountIdentityParams, entry audit.Entry, appender audit.Appender) (sqlc.Account, sqlc.AccountIdentity, error)
	CreateSession(ctx context.Context, token sqlc.CreateRefreshTokenParams, entry audit.Entry, appender audit.Appender) error
	GetRefreshTokenByHash(ctx context.Context, hash []byte) (sqlc.RefreshToken, error)
	ResolveActiveAdmin(ctx context.Context, accountID uuid.UUID) (AdminAuthority, error)
	RevokeRefreshFamilyByHash(ctx context.Context, hash []byte, entry audit.Entry, appender audit.Appender) error
	RevokeAccountSessions(ctx context.Context, accountID uuid.UUID, entry audit.Entry, appender audit.Appender) error
	LinkIdentityForAccount(ctx context.Context, accountID uuid.UUID, identity sqlc.LinkAccountIdentityParams, entry audit.Entry, appender audit.Appender) (sqlc.AccountIdentity, error)
	UnlinkIdentityForAccount(ctx context.Context, accountID, identityID uuid.UUID, entry audit.Entry, appender audit.Appender) error
	RotateRefreshToken(ctx context.Context, hash []byte, replacement sqlc.CreateRefreshTokenParams, refreshedEntry, reuseEntry audit.Entry, now time.Time, appender audit.Appender) (sqlc.Account, sqlc.RefreshToken, error)
}

func (s *service) Exchange(ctx context.Context, input ExchangeInput) (session IssuedSession, err error) {
	ctx, cancel := s.operationContext(ctx)
	defer cancel()
	client := input.Client
	defer func() { s.logOperation("exchange", client, err) }()
	client, err = normalizeClient(input.Client)
	if err != nil {
		return IssuedSession{}, err
	}
	if strings.TrimSpace(input.IDToken) == "" {
		return IssuedSession{}, ErrAuthenticationFailed
	}
	if err := s.checkRate(ctx, "exchange:ip:"+rateLimitIP(input.IPAddress), s.rateLimits.ExchangeIPPerMin); err != nil {
		return IssuedSession{}, err
	}

	verified, err := s.verifier.Verify(ctx, input.IDToken)
	if err != nil {
		return IssuedSession{}, mapVerifierError(err)
	}
	if strings.TrimSpace(verified.UID) == "" {
		return IssuedSession{}, ErrAuthenticationFailed
	}
	if err := s.checkRate(ctx, "exchange:identity:"+verified.UID, s.rateLimits.ExchangeIdentityPerMin); err != nil {
		return IssuedSession{}, err
	}

	account, _, err := s.resolveAccount(ctx, verified, client)
	if err != nil {
		if client == ClientFleetConsole &&
			(errors.Is(err, ErrClientForbidden) || errors.Is(err, ErrAuthenticationFailed)) {
			reason := ReasonNoActiveGrant
			if errors.Is(err, ErrAuthenticationFailed) {
				reason = ReasonAccountInactive
			}
			if auditErr := s.auditAdminDenied(ctx, uuid.Nil, reason); auditErr != nil {
				return IssuedSession{}, auditErr
			}
		}
		return IssuedSession{}, err
	}

	accessToken, accessExpiresAt, err := s.tokens.Issue(ctx, account.ID, client)
	if err != nil {
		return IssuedSession{}, fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	refreshToken, refreshHash, err := NewOpaqueRefreshToken()
	if err != nil {
		return IssuedSession{}, fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	refreshExpiresAt, err := RefreshTokenExpiry(s.clock.Now(), s.refreshTTL)
	if err != nil {
		return IssuedSession{}, fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	familyID := NewRefreshFamilyID()
	entry, err := buildAuthAuditEntry(authAuditEntry{
		Action:    ActionSessionIssued,
		ActorType: ActorAccount,
		ActorID:   account.ID.String(),
		AccountID: account.ID,
		Entity:    "session",
		Details:   authAuditDetails{Client: client},
	})
	if err != nil {
		return IssuedSession{}, err
	}
	err = s.store.CreateSession(ctx, sqlc.CreateRefreshTokenParams{
		ID:          uuid.New(),
		SubjectType: "account",
		SubjectID:   account.ID,
		FamilyID:    familyID,
		Client:      client,
		TokenHash:   refreshHash,
		ExpiresAt:   refreshExpiresAt,
		UserAgent:   optionalString(input.UserAgent),
	}, entry, s.audit)
	if err != nil {
		return IssuedSession{}, err
	}

	return IssuedSession{
		SessionID:        familyID,
		AccountID:        account.ID,
		Client:           client,
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (s *service) Refresh(ctx context.Context, input RefreshInput) (session IssuedSession, err error) {
	ctx, cancel := s.operationContext(ctx)
	defer cancel()
	client := ""
	defer func() { s.logOperation("refresh", client, err) }()
	rawToken := strings.TrimSpace(input.RefreshToken)
	if rawToken == "" {
		return IssuedSession{}, ErrAuthenticationFailed
	}
	if err := s.checkRate(ctx, "refresh:ip:"+rateLimitIP(input.IPAddress), s.rateLimits.RefreshIPPerMin); err != nil {
		return IssuedSession{}, err
	}
	tokenHash := HashRefreshToken(rawToken)
	if err := s.checkRate(ctx, "refresh:session:"+hex.EncodeToString(tokenHash), s.rateLimits.RefreshSessionPerMin); err != nil {
		return IssuedSession{}, err
	}

	stored, err := s.store.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return IssuedSession{}, ErrAuthenticationFailed
		}
		return IssuedSession{}, err
	}
	client = stored.Client
	if stored.SubjectType != "account" || !validClient(stored.Client) {
		return IssuedSession{}, ErrAuthenticationFailed
	}
	now := s.clock.Now().UTC()
	refreshedEntry, reuseEntry, err := buildRefreshAuditEntries(stored)
	if err != nil {
		return IssuedSession{}, err
	}

	if stored.UsedAt != nil || stored.RevokedAt != nil {
		_, _, rotateErr := s.store.RotateRefreshToken(ctx, tokenHash, sqlc.CreateRefreshTokenParams{}, refreshedEntry, reuseEntry, now, s.audit)
		return IssuedSession{}, mapRefreshError(rotateErr)
	}

	if !now.Before(stored.ExpiresAt) {
		return IssuedSession{}, ErrAuthenticationFailed
	}
	account, err := s.loadActiveAccount(ctx, stored.SubjectID)
	if err != nil {
		return IssuedSession{}, err
	}
	if stored.Client == ClientFleetConsole {
		if _, err := s.store.ResolveActiveAdmin(ctx, account.ID); err != nil {
			if errors.Is(err, ErrAdminGrantNotFound) {
				if auditErr := s.auditAdminDenied(ctx, account.ID, ReasonNoActiveGrant); auditErr != nil {
					return IssuedSession{}, auditErr
				}
				return IssuedSession{}, ErrAuthenticationFailed
			}
			return IssuedSession{}, err
		}
	}
	accessToken, accessExpiresAt, err := s.tokens.Issue(ctx, account.ID, stored.Client)
	if err != nil {
		return IssuedSession{}, fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	refreshToken, refreshHash, err := NewOpaqueRefreshToken()
	if err != nil {
		return IssuedSession{}, fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	refreshExpiresAt, err := RefreshTokenExpiry(now, s.refreshTTL)
	if err != nil {
		return IssuedSession{}, fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	rotatedAccount, rotatedToken, err := s.store.RotateRefreshToken(ctx, tokenHash, sqlc.CreateRefreshTokenParams{
		ID:        uuid.New(),
		TokenHash: refreshHash,
		ExpiresAt: refreshExpiresAt,
		UserAgent: optionalString(input.UserAgent),
	}, refreshedEntry, reuseEntry, now, s.audit)
	if err != nil {
		return IssuedSession{}, mapRefreshError(err)
	}

	return IssuedSession{
		SessionID:        rotatedToken.FamilyID,
		AccountID:        rotatedAccount.ID,
		Client:           rotatedToken.Client,
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func buildRefreshAuditEntries(stored sqlc.RefreshToken) (audit.Entry, audit.Entry, error) {
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

func (s *service) Logout(ctx context.Context, refreshToken string) (err error) {
	ctx, cancel := s.operationContext(ctx)
	defer cancel()
	client := ""
	defer func() { s.logOperation("logout", client, err) }()
	rawToken := strings.TrimSpace(refreshToken)
	if rawToken == "" {
		return nil
	}
	hash := HashRefreshToken(rawToken)
	stored, err := s.store.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil
		}
		return err
	}
	client = stored.Client
	actorType := ActorAccount
	if stored.SubjectType == "admin" {
		actorType = ActorAdmin
	}
	entry, err := buildAuthAuditEntry(authAuditEntry{
		Action:    ActionSessionRevoked,
		ActorType: actorType,
		ActorID:   stored.SubjectID.String(),
		AccountID: stored.SubjectID,
		Entity:    "session",
		Details:   authAuditDetails{Client: stored.Client},
	})
	if err != nil {
		return err
	}
	return s.store.RevokeRefreshFamilyByHash(ctx, hash, entry, s.audit)
}

func (s *service) LogoutAll(ctx context.Context, accountID uuid.UUID) (err error) {
	ctx, cancel := s.operationContext(ctx)
	defer cancel()
	defer func() { s.logOperation("logout_all", "", err) }()
	if accountID == uuid.Nil {
		return ErrAuthenticationFailed
	}
	if err := s.checkRate(ctx, "logout-all:account:"+accountID.String(), s.rateLimits.LogoutAllAccountPerMin); err != nil {
		return err
	}
	entry, err := buildAuthAuditEntry(authAuditEntry{
		Action:    ActionSessionRevokedAll,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "account",
	})
	if err != nil {
		return err
	}
	return s.store.RevokeAccountSessions(ctx, accountID, entry, s.audit)
}

func (s *service) LinkIdentity(ctx context.Context, accountID uuid.UUID, idToken string) (identity Identity, err error) {
	ctx, cancel := s.operationContext(ctx)
	defer cancel()
	defer func() { s.logOperation("identity_link", "", err) }()
	if accountID == uuid.Nil || strings.TrimSpace(idToken) == "" {
		return Identity{}, ErrAuthenticationFailed
	}
	if err := s.checkRate(ctx, "link:account:"+accountID.String(), s.rateLimits.LinkAccountPerMin); err != nil {
		return Identity{}, err
	}
	verified, err := s.verifier.Verify(ctx, idToken)
	if err != nil {
		return Identity{}, mapVerifierError(err)
	}
	if strings.TrimSpace(verified.UID) == "" {
		return Identity{}, ErrAuthenticationFailed
	}
	params := sqlc.LinkAccountIdentityParams{
		ID:          uuid.New(),
		Provider:    firebaseProvider,
		ProviderSub: verified.UID,
		Email:       verifiedEmail(verified),
	}
	entry, err := buildAuthAuditEntry(authAuditEntry{
		Action:    ActionIdentityLinked,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "identity",
		Details:   authAuditDetails{Provider: firebaseProvider},
	})
	if err != nil {
		return Identity{}, err
	}
	linked, err := s.store.LinkIdentityForAccount(ctx, accountID, params, entry, s.audit)
	if err == nil {
		return identityFromRow(linked), nil
	}
	switch {
	case errors.Is(err, ErrIdentityConflict):
		existing, lookupErr := s.store.GetIdentityByProviderSubject(ctx, firebaseProvider, verified.UID)
		if lookupErr == nil && existing.AccountID == accountID {
			if auditErr := s.audit.AppendTx(ctx, nil, entry); auditErr != nil {
				return Identity{}, auditErr
			}
			return identityFromRow(existing), nil
		}
		if lookupErr != nil && !errors.Is(lookupErr, ErrIdentityNotFound) {
			return Identity{}, lookupErr
		}
		if auditErr := s.auditIdentityDenied(ctx, ActionIdentityLinkDenied, accountID, ReasonConflict); auditErr != nil {
			return Identity{}, auditErr
		}
		return Identity{}, ErrIdentityConflict
	case errors.Is(err, ErrAuthenticationFailed):
		if auditErr := s.auditIdentityDenied(ctx, ActionIdentityLinkDenied, accountID, ReasonAccountInactive); auditErr != nil {
			return Identity{}, auditErr
		}
		return Identity{}, ErrAuthenticationFailed
	default:
		return Identity{}, err
	}
}

func (s *service) UnlinkIdentity(ctx context.Context, accountID, identityID uuid.UUID) (err error) {
	ctx, cancel := s.operationContext(ctx)
	defer cancel()
	defer func() { s.logOperation("identity_unlink", "", err) }()
	if accountID == uuid.Nil || identityID == uuid.Nil {
		return ErrAuthenticationFailed
	}
	if err := s.checkRate(ctx, "unlink:account:"+accountID.String(), s.rateLimits.LinkAccountPerMin); err != nil {
		return err
	}
	entry, err := buildAuthAuditEntry(authAuditEntry{
		Action:    ActionIdentityUnlinked,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "identity",
		Details:   authAuditDetails{Provider: firebaseProvider},
	})
	if err != nil {
		return err
	}
	err = s.store.UnlinkIdentityForAccount(ctx, accountID, identityID, entry, s.audit)
	switch {
	case errors.Is(err, ErrLastIdentity):
		if auditErr := s.auditIdentityDenied(ctx, ActionIdentityUnlinkDenied, accountID, ReasonLastIdentity); auditErr != nil {
			return auditErr
		}
		return ErrLastIdentity
	case errors.Is(err, ErrIdentityNotFound):
		if auditErr := s.auditIdentityDenied(ctx, ActionIdentityUnlinkDenied, accountID, ReasonNotOwner); auditErr != nil {
			return auditErr
		}
		return ErrIdentityNotFound
	default:
		return err
	}
}

func (s *service) auditIdentityDenied(ctx context.Context, action string, accountID uuid.UUID, reason AuthAuditReason) error {
	entry, err := buildAuthAuditEntry(authAuditEntry{
		Action:    action,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "identity",
		Details:   authAuditDetails{Provider: firebaseProvider, Reason: reason},
	})
	if err != nil {
		return err
	}
	return s.audit.AppendTx(ctx, nil, entry)
}

func identityFromRow(row sqlc.AccountIdentity) Identity {
	return Identity{
		ID:        row.ID,
		AccountID: row.AccountID,
		Provider:  row.Provider,
		Email:     row.Email,
		CreatedAt: row.CreatedAt,
	}
}

func (s *service) resolveAccount(ctx context.Context, verified VerifiedIdentity, client string) (sqlc.Account, bool, error) {
	identity, err := s.store.GetIdentityByProviderSubject(ctx, firebaseProvider, verified.UID)
	switch {
	case err == nil:
		account, err := s.loadActiveAccount(ctx, identity.AccountID)
		if err != nil {
			return sqlc.Account{}, false, err
		}
		if client == ClientFleetConsole {
			if err := s.requireAdminGrant(ctx, account.ID); err != nil {
				return sqlc.Account{}, false, err
			}
		}
		return account, false, nil
	case errors.Is(err, ErrIdentityNotFound):
		if client != ClientGuestMobile {
			return sqlc.Account{}, false, ErrClientForbidden
		}
		return s.provisionGuest(ctx, verified)
	default:
		return sqlc.Account{}, false, err
	}
}

func (s *service) provisionGuest(ctx context.Context, verified VerifiedIdentity) (sqlc.Account, bool, error) {
	accountID := uuid.New()
	email := verifiedEmail(verified)
	accountParams := sqlc.CreateAccountParams{
		ID:          accountID,
		Email:       email,
		DisplayName: optionalString(strings.TrimSpace(verified.DisplayName)),
		Locale:      defaultLocale,
		Status:      "active",
	}
	identityParams := sqlc.LinkAccountIdentityParams{
		ID:          uuid.New(),
		Provider:    firebaseProvider,
		ProviderSub: verified.UID,
		Email:       email,
	}
	entry, err := buildAuthAuditEntry(authAuditEntry{
		Action:    ActionIdentityProvisioned,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "account",
		Details:   authAuditDetails{Provider: firebaseProvider},
	})
	if err != nil {
		return sqlc.Account{}, false, err
	}
	account, _, err := s.store.ProvisionGuest(ctx, accountParams, identityParams, entry, s.audit)
	if err == nil {
		return account, true, nil
	}

	switch {
	case errors.Is(err, ErrEmailConflict):
		// A concurrent exchange for the same uid may have won after the email
		// unique check; resolve that winner before treating the email as a
		// collision with a different account.
		existing, lookupErr := s.store.GetIdentityByProviderSubject(ctx, firebaseProvider, verified.UID)
		if lookupErr == nil {
			account, err := s.loadActiveAccount(ctx, existing.AccountID)
			return account, false, err
		}
		if errors.Is(lookupErr, ErrIdentityNotFound) {
			return sqlc.Account{}, false, ErrIdentityLinkRequired
		}
		return sqlc.Account{}, false, lookupErr
	case errors.Is(err, ErrIdentityConflict):
		existing, lookupErr := s.store.GetIdentityByProviderSubject(ctx, firebaseProvider, verified.UID)
		if lookupErr != nil {
			return sqlc.Account{}, false, lookupErr
		}
		account, err := s.loadActiveAccount(ctx, existing.AccountID)
		return account, false, err
	default:
		return sqlc.Account{}, false, err
	}
}

func (s *service) requireAdminGrant(ctx context.Context, accountID uuid.UUID) error {
	if _, err := s.store.ResolveActiveAdmin(ctx, accountID); err != nil {
		if errors.Is(err, ErrAdminGrantNotFound) {
			return ErrClientForbidden
		}
		return err
	}
	return nil
}

func (s *service) auditAdminDenied(ctx context.Context, accountID uuid.UUID, reason AuthAuditReason) error {
	actorID := ""
	if accountID != uuid.Nil {
		actorID = accountID.String()
	}
	entry, err := buildAuthAuditEntry(authAuditEntry{
		Action:    ActionAdminAccessDenied,
		ActorType: ActorAccount,
		ActorID:   actorID,
		AccountID: accountID,
		Entity:    "account",
		Details:   authAuditDetails{Provider: firebaseProvider, Reason: reason},
	})
	if err != nil {
		return err
	}
	return s.audit.AppendTx(ctx, nil, entry)
}

func (s *service) loadActiveAccount(ctx context.Context, accountID uuid.UUID) (sqlc.Account, error) {
	account, err := s.store.GetAccount(ctx, accountID)
	if err != nil {
		if appErr, ok := apperrors.As(err); ok && appErr.HTTPStatus == 404 {
			return sqlc.Account{}, ErrAuthenticationFailed
		}
		return sqlc.Account{}, err
	}
	if account.Status != "active" || account.DeletedAt != nil {
		return sqlc.Account{}, ErrAuthenticationFailed
	}
	return account, nil
}

func (s *service) checkRate(ctx context.Context, key string, limit int) error {
	decision, err := s.limiter.Allow(ctx, key, limit, rateLimitWindow)
	if err != nil {
		if errors.Is(err, ErrRateLimiterUnavailable) {
			return err
		}
		return fmt.Errorf("%w: %v", ErrRateLimiterUnavailable, err)
	}
	if decision.Allowed {
		return nil
	}
	retryAfter := decision.RetryAfter
	if retryAfter <= 0 {
		retryAfter = rateLimitWindow
	}
	return &RateLimitedError{RetryAfter: retryAfter}
}

func (s *service) operationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.timeout)
}

func (s *service) logOperation(operation, client string, err error) {
	outcome := "success"
	if err != nil {
		outcome = authErrorCategory(err)
	}
	attrs := []any{"operation", operation, "outcome", outcome}
	if validClient(client) {
		attrs = append(attrs, "client", client)
	}
	if err != nil {
		s.logger.Warn("auth outcome", attrs...)
		return
	}
	s.logger.Info("auth outcome", attrs...)
}

func authErrorCategory(err error) string {
	switch {
	case errors.Is(err, ErrAuthenticationFailed):
		return "authentication_failed"
	case errors.Is(err, ErrClientForbidden):
		return "forbidden"
	case errors.Is(err, ErrRateLimited):
		return "rate_limited"
	case errors.Is(err, ErrRateLimiterUnavailable),
		errors.Is(err, ErrFirebaseUnavailable),
		errors.Is(err, ErrSessionUnavailable):
		return "dependency_unavailable"
	case errors.Is(err, ErrIdentityLinkRequired):
		return "identity_link_required"
	case errors.Is(err, ErrIdentityConflict):
		return "identity_conflict"
	case errors.Is(err, ErrLastIdentity):
		return "last_identity"
	case errors.Is(err, ErrIdentityNotFound):
		return "identity_not_found"
	default:
		return "error"
	}
}

func normalizeClient(client string) (string, error) {
	client = strings.TrimSpace(client)
	if client == "" {
		return ClientGuestMobile, nil
	}
	if !validClient(client) {
		return "", ErrClientForbidden
	}
	return client, nil
}

func verifiedEmail(identity VerifiedIdentity) *string {
	if !identity.EmailVerified {
		return nil
	}
	email := strings.TrimSpace(identity.Email)
	if email == "" {
		return nil
	}
	return &email
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func rateLimitIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "unknown"
	}
	return ip
}

func mapVerifierError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidFirebaseToken):
		return ErrAuthenticationFailed
	case errors.Is(err, ErrFirebaseUnavailable):
		return ErrFirebaseUnavailable
	default:
		return ErrFirebaseUnavailable
	}
}
