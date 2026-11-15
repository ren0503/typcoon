package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	apperrors "github.com/ren0503/tycoon/internal/platform/errors"
)

type service struct {
	store      Store
	verifier   Verifier
	keys       KeySet
	limiter    RateLimiter
	clock      Clock
	logger     *slog.Logger
	tokens     *AccessTokenIssuer
	refreshTTL time.Duration
	rateLimits RateLimits
}

func (s *service) Exchange(ctx context.Context, input ExchangeInput) (session IssuedSession, err error) {
	client := lo.CoalesceOrEmpty(input.Client, ClientGuestMobile)
	defer func() { s.logOperation(ctx, "exchange", client, err) }()

	// Validate input and rate-limit.
	if err := input.Valid(); err != nil {
		return IssuedSession{}, err
	}

	if err := s.checkRate(ctx, "exchange:ip:"+rateLimitIP(input.IPAddress), s.rateLimits.ExchangeIPPerMin); err != nil {
		return IssuedSession{}, err
	}

	verified, err := s.verifier.Verify(ctx, input.IDToken)
	if err != nil {
		return IssuedSession{}, mapVerifierError(err)
	}

	if err := s.checkRate(ctx, "exchange:identity:"+verified.UID, s.rateLimits.ExchangeIdentityPerMin); err != nil {
		return IssuedSession{}, err
	}

	// Resolve the account for the verified identity and requested client.
	account, _, err := s.resolveAccount(ctx, verified, client)
	if isAdminAction(client, err) {
		if auditErr := s.auditAdminDenied(ctx, uuid.Nil, err); auditErr != nil {
			return IssuedSession{}, auditErr
		}
		return IssuedSession{}, err
	}

	// Generate a new access token and refresh token for the account and client.
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

	entry, err := buildIssueAuditEntry(account.ID, client)
	if err != nil {
		return IssuedSession{}, err
	}

	familyID := NewRefreshFamilyID()
	err = s.store.CreateSession(ctx, CreateRefreshTokenParams{
		ID:          uuid.New(),
		SubjectType: "account",
		SubjectID:   account.ID,
		FamilyID:    familyID,
		Client:      client,
		TokenHash:   refreshHash,
		ExpiresAt:   refreshExpiresAt,
		UserAgent:   new(input.UserAgent),
	}, entry)
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
	var client string
	defer func() { s.logOperation(ctx, "refresh", client, err) }()

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

	storedToken, err := s.store.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return IssuedSession{}, ErrAuthenticationFailed
		}
		return IssuedSession{}, err
	}
	client = storedToken.Client
	now := s.clock.Now().UTC()
	if isRefreshTokenInvalid(storedToken, now) {
		return IssuedSession{}, ErrAuthenticationFailed
	}
	refreshedEntry, reuseEntry, err := buildRefreshAuditEntries(storedToken)
	if err != nil {
		return IssuedSession{}, err
	}

	if isRefreshTokenUsed(storedToken) {
		_, _, rotateErr := s.store.RotateRefreshToken(ctx, tokenHash, CreateRefreshTokenParams{}, refreshedEntry, reuseEntry, now)
		return IssuedSession{}, mapRefreshError(rotateErr)
	}

	account, err := s.loadActiveAccount(ctx, storedToken.SubjectID)
	if err != nil {
		return IssuedSession{}, err
	}
	if storedToken.Client == ClientFleetConsole {
		if _, err := s.store.ResolveActiveAdmin(ctx, account.ID); err != nil {
			if errors.Is(err, ErrAdminGrantNotFound) {
				if auditErr := s.auditAdminDenied(ctx, account.ID, nil); auditErr != nil {
					return IssuedSession{}, auditErr
				}
				return IssuedSession{}, ErrAuthenticationFailed
			}
			return IssuedSession{}, err
		}
	}

	accessToken, accessExpiresAt, err := s.tokens.Issue(ctx, account.ID, storedToken.Client)
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

	rotatedAccount, rotatedToken, err := s.store.RotateRefreshToken(ctx, tokenHash, CreateRefreshTokenParams{
		ID:        uuid.New(),
		TokenHash: refreshHash,
		ExpiresAt: refreshExpiresAt,
		UserAgent: new(input.UserAgent),
	}, refreshedEntry, reuseEntry, now)
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

func (s *service) Logout(ctx context.Context, refreshToken string) (err error) {
	var client string
	defer func() { s.logOperation(ctx, "logout", client, err) }()

	rawToken := strings.TrimSpace(refreshToken)
	if rawToken == "" {
		return nil
	}
	hash := HashRefreshToken(rawToken)
	storedToken, err := s.store.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil
		}
		return err
	}
	client = storedToken.Client
	actorType := ActorAccount
	if storedToken.SubjectType == "admin" {
		actorType = ActorAdmin
	}
	entry, err := buildLogoutAuditEntry(storedToken, actorType)
	if err != nil {
		return err
	}
	return s.store.RevokeRefreshFamilyByHash(ctx, hash, entry)
}

func (s *service) LogoutAll(ctx context.Context, accountID uuid.UUID) (err error) {
	defer func() { s.logOperation(ctx, "logout_all", "", err) }()

	if accountID == uuid.Nil {
		return ErrAuthenticationFailed
	}
	if err := s.checkRate(ctx, "logout-all:account:"+accountID.String(), s.rateLimits.LogoutAllAccountPerMin); err != nil {
		return err
	}
	entry, err := buildLogoutAllAuditEntry(accountID, ActorAccount)
	if err != nil {
		return err
	}
	return s.store.RevokeAccountSessions(ctx, accountID, entry)
}

func (s *service) LinkIdentity(ctx context.Context, accountID uuid.UUID, idToken string) (identity Identity, err error) {
	defer func() { s.logOperation(ctx, "identity_link", "", err) }()

	if isEmptyAccount(accountID, idToken) {
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
	params := LinkAccountIdentityParams{
		ID:          uuid.New(),
		Provider:    firebaseProvider,
		ProviderSub: verified.UID,
		Email:       verifiedEmail(verified),
	}
	entry, err := buildLinkIdentityAuditEntry(accountID, firebaseProvider)
	if err != nil {
		return Identity{}, err
	}

	linked, err := s.store.LinkIdentityForAccount(ctx, accountID, params, entry)
	if err == nil {
		return identityFromRow(linked), nil
	}

	switch {
	case errors.Is(err, ErrIdentityConflict):
		existing, lookupErr := s.store.GetIdentityByProviderSubject(ctx, firebaseProvider, verified.UID)
		if lookupErr == nil && existing.AccountID == accountID {
			if auditErr := s.store.AppendAudit(ctx, entry); auditErr != nil {
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
	defer func() { s.logOperation(ctx, "identity_unlink", "", err) }()

	if isEmptyIdentity(accountID, identityID) {
		return ErrAuthenticationFailed
	}
	if err := s.checkRate(ctx, "unlink:account:"+accountID.String(), s.rateLimits.LinkAccountPerMin); err != nil {
		return err
	}
	entry, err := buildUnlinkIdentityAuditEntry(accountID, firebaseProvider)
	if err != nil {
		return err
	}
	err = s.store.UnlinkIdentityForAccount(ctx, accountID, identityID, entry)
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
	entry, err := buildIdentityDeniedAuditEntry(action, accountID, firebaseProvider, reason)
	if err != nil {
		return err
	}
	return s.store.AppendAudit(ctx, entry)
}

func identityFromRow(row AccountIdentity) Identity {
	return Identity{
		ID:        row.ID,
		AccountID: row.AccountID,
		Provider:  row.Provider,
		Email:     row.Email,
		CreatedAt: row.CreatedAt,
	}
}

func (s *service) resolveAccount(ctx context.Context, verified VerifiedIdentity, client string) (Account, bool, error) {
	identity, err := s.store.GetIdentityByProviderSubject(ctx, firebaseProvider, verified.UID)
	switch {
	case err == nil:
		account, err := s.loadActiveAccount(ctx, identity.AccountID)
		if err != nil {
			return Account{}, false, err
		}
		if client == ClientFleetConsole {
			if err := s.requireAdminGrant(ctx, account.ID); err != nil {
				return Account{}, false, err
			}
		}
		return account, false, nil
	case errors.Is(err, ErrIdentityNotFound):
		if client != ClientGuestMobile {
			return Account{}, false, ErrClientForbidden
		}
		return s.provisionGuest(ctx, verified)
	default:
		return Account{}, false, err
	}
}

func (s *service) provisionGuest(ctx context.Context, verified VerifiedIdentity) (Account, bool, error) {
	accountID := uuid.New()
	email := verifiedEmail(verified)
	accountParams := CreateAccountParams{
		ID:          accountID,
		Email:       email,
		DisplayName: lo.ToPtr(strings.TrimSpace(verified.DisplayName)),
		Locale:      defaultLocale,
		Status:      "active",
	}
	identityParams := LinkAccountIdentityParams{
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
		return Account{}, false, err
	}
	account, _, err := s.store.ProvisionGuest(ctx, accountParams, identityParams, entry)
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
			return Account{}, false, ErrIdentityLinkRequired
		}
		return Account{}, false, lookupErr
	case errors.Is(err, ErrIdentityConflict):
		existing, lookupErr := s.store.GetIdentityByProviderSubject(ctx, firebaseProvider, verified.UID)
		if lookupErr != nil {
			return Account{}, false, lookupErr
		}
		account, err := s.loadActiveAccount(ctx, existing.AccountID)
		return account, false, err
	default:
		return Account{}, false, err
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

func (s *service) auditAdminDenied(ctx context.Context, accountID uuid.UUID, err error) error {
	reason := ReasonNoActiveGrant
	if err != nil && errors.Is(err, ErrAuthenticationFailed) {
		reason = ReasonAccountInactive
	}

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
	return s.store.AppendAudit(ctx, entry)
}

func (s *service) loadActiveAccount(ctx context.Context, accountID uuid.UUID) (Account, error) {
	account, err := s.store.GetAccount(ctx, accountID)
	if err != nil {
		if appErr, ok := apperrors.As(err); ok && appErr.HTTPStatus == 404 {
			return Account{}, ErrAuthenticationFailed
		}
		return Account{}, err
	}
	if account.Status != "active" || account.DeletedAt != nil {
		return Account{}, ErrAuthenticationFailed
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

func (s *service) logOperation(ctx context.Context, operation, client string, err error) {
	outcome := "success"
	if err != nil {
		outcome = authErrorCategory(err)
	}
	attrs := []any{"operation", operation, "outcome", outcome}
	if ValidClient(client) {
		attrs = append(attrs, "client", client)
	}
	if err != nil {
		s.logger.ErrorContext(ctx, "auth outcome", attrs...)
		return
	}
	s.logger.InfoContext(ctx, "auth outcome", attrs...)
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
