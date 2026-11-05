package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/audit"
	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

const (
	accountIdentityConflictConstraint = "account_identities_provider_provider_sub_key"
	accountEmailConflictConstraint    = "uq_accounts_email"
)

// AdminAuthority is the backend-resolved Fleet Console authorization context.
type AdminAuthority struct {
	AdminUserID uuid.UUID
	Role        string
	TenantID    uuid.UUID
	ShipIDs     []uuid.UUID
}

var (
	ErrIdentityNotFound     = errors.New("account identity not found")
	ErrIdentityConflict     = errors.New("account identity already exists")
	ErrEmailConflict        = errors.New("account email already exists")
	ErrAdminGrantNotFound   = errors.New("active administrator grant not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenInvalid  = errors.New("refresh token invalid")
	ErrRefreshTokenReused   = errors.New("refresh token reuse detected")
)

// Store owns transactional persistence for authentication state.
type Store struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

// NewStore builds the auth persistence layer over PostgreSQL.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: sqlc.New(pool)}
}

// InTx executes fn in a transaction, rolling back on error and committing on success.
func (s *Store) InTx(ctx context.Context, fn func(*Tx) error) error {
	transaction, err := s.pool.Begin(ctx)
	if err != nil {
		return database.MapPgError(err)
	}
	wrapped := &Tx{tx: transaction, q: sqlc.New(transaction)}
	if err := fn(wrapped); err != nil {
		_ = transaction.Rollback(ctx)
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return database.MapPgError(err)
	}
	return nil
}

// GetAccount returns a non-deleted account by ID.
func (s *Store) GetAccount(ctx context.Context, id uuid.UUID) (sqlc.Account, error) {
	row, err := s.q.GetAccount(ctx, id)
	return row, database.MapPgError(err)
}

// GetIdentityByProviderSubject resolves a linked identity.
func (s *Store) GetIdentityByProviderSubject(ctx context.Context, provider, subject string) (sqlc.AccountIdentity, error) {
	return getIdentityByProviderSubject(ctx, s.q, provider, subject)
}

// GetActiveAdminByAccountID resolves an active Fleet grant for an account.
func (s *Store) GetActiveAdminByAccountID(ctx context.Context, accountID uuid.UUID) (sqlc.AdminUser, error) {
	return getActiveAdminByAccountID(ctx, s.q, accountID)
}

// ListAdminShipIDs returns an administrator's ship scope.
func (s *Store) ListAdminShipIDs(ctx context.Context, adminUserID uuid.UUID) ([]uuid.UUID, error) {
	return listAdminShipIDs(ctx, s.q, adminUserID)
}

// ResolveActiveAdmin resolves the active grant and its backend-owned scope.
func (s *Store) ResolveActiveAdmin(ctx context.Context, accountID uuid.UUID) (AdminAuthority, error) {
	return resolveActiveAdmin(ctx, s.q, accountID)
}

// ProvisionGuest creates a guest account, its Firebase identity, and the
// provisioning audit row in one transaction.
func (s *Store) ProvisionGuest(
	ctx context.Context,
	account sqlc.CreateAccountParams,
	identity sqlc.LinkAccountIdentityParams,
	entry audit.Entry,
	appender audit.Appender,
) (sqlc.Account, sqlc.AccountIdentity, error) {
	if appender == nil {
		return sqlc.Account{}, sqlc.AccountIdentity{}, errors.New("audit appender is required")
	}
	var created sqlc.Account
	var linked sqlc.AccountIdentity
	err := s.InTx(ctx, func(tx *Tx) error {
		var err error
		created, linked, err = tx.CreateAccountWithIdentity(ctx, account, identity)
		if err != nil {
			return err
		}
		return appender.AppendTx(ctx, tx.Raw(), entry)
	})
	if err != nil {
		return sqlc.Account{}, sqlc.AccountIdentity{}, err
	}
	return created, linked, nil
}

// LinkIdentityForAccount inserts a Firebase identity for an existing active
// account. Repeating the same link is idempotent.
func (s *Store) LinkIdentityForAccount(
	ctx context.Context,
	accountID uuid.UUID,
	identity sqlc.LinkAccountIdentityParams,
	entry audit.Entry,
	appender audit.Appender,
) (sqlc.AccountIdentity, error) {
	if appender == nil {
		return sqlc.AccountIdentity{}, errors.New("audit appender is required")
	}
	var linked sqlc.AccountIdentity
	err := s.InTx(ctx, func(tx *Tx) error {
		if err := tx.LockAccount(ctx, accountID); err != nil {
			if appErr, ok := apperrors.As(err); ok && appErr.HTTPStatus == 404 {
				return ErrAuthenticationFailed
			}
			return err
		}
		account, err := tx.GetAccount(ctx, accountID)
		if err != nil {
			if appErr, ok := apperrors.As(err); ok && appErr.HTTPStatus == 404 {
				return ErrAuthenticationFailed
			}
			return err
		}
		if account.Status != "active" || account.DeletedAt != nil {
			return ErrAuthenticationFailed
		}
		existing, err := tx.GetIdentityByProviderSubject(ctx, identity.Provider, identity.ProviderSub)
		switch {
		case err == nil:
			if existing.AccountID != accountID {
				return ErrIdentityConflict
			}
			linked = existing
		case errors.Is(err, ErrIdentityNotFound):
			identity.AccountID = accountID
			linked, err = tx.CreateIdentity(ctx, identity)
			if err != nil {
				return err
			}
		default:
			return err
		}
		return appender.AppendTx(ctx, tx.Raw(), entry)
	})
	if err != nil {
		return sqlc.AccountIdentity{}, err
	}
	return linked, nil
}

// UnlinkIdentityForAccount deletes an identity only when owned by the account
// and at least one other identity remains.
func (s *Store) UnlinkIdentityForAccount(
	ctx context.Context,
	accountID, identityID uuid.UUID,
	entry audit.Entry,
	appender audit.Appender,
) error {
	if appender == nil {
		return errors.New("audit appender is required")
	}
	return s.InTx(ctx, func(tx *Tx) error {
		if err := tx.LockAccount(ctx, accountID); err != nil {
			if appErr, ok := apperrors.As(err); ok && appErr.HTTPStatus == 404 {
				return ErrIdentityNotFound
			}
			return err
		}
		identity, err := tx.q.GetAccountIdentity(ctx, identityID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrIdentityNotFound
		case err != nil:
			return database.MapPgError(err)
		case identity.AccountID != accountID:
			return ErrIdentityNotFound
		}
		count, err := tx.q.CountAccountIdentities(ctx, accountID)
		if err != nil {
			return database.MapPgError(err)
		}
		if count <= 1 {
			return ErrLastIdentity
		}
		deleted, err := tx.DeleteIdentityForAccount(ctx, identityID, accountID)
		if err != nil {
			return err
		}
		if deleted == 0 {
			return ErrIdentityNotFound
		}
		return appender.AppendTx(ctx, tx.Raw(), entry)
	})
}

// GetRefreshTokenByHash returns a refresh-token row by its storage hash.
func (s *Store) GetRefreshTokenByHash(ctx context.Context, hash []byte) (sqlc.RefreshToken, error) {
	row, err := s.q.GetRefreshTokenByHash(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.RefreshToken{}, ErrRefreshTokenNotFound
	}
	return row, database.MapPgError(err)
}

// RevokeRefreshFamilyByHash revokes the family for a presented refresh token.
// Unknown tokens are treated as an idempotent success.
func (s *Store) RevokeRefreshFamilyByHash(
	ctx context.Context,
	hash []byte,
	entry audit.Entry,
	appender audit.Appender,
) error {
	if appender == nil {
		return errors.New("audit appender is required")
	}
	return s.InTx(ctx, func(tx *Tx) error {
		stored, err := tx.GetRefreshTokenByHashForUpdate(ctx, hash)
		if err != nil {
			if errors.Is(err, ErrRefreshTokenNotFound) {
				return nil
			}
			return err
		}
		revoked, err := tx.RevokeRefreshTokenFamily(ctx, stored.FamilyID)
		if err != nil {
			return err
		}
		if revoked == 0 {
			return nil
		}
		return appender.AppendTx(ctx, tx.Raw(), entry)
	})
}

// RevokeAccountSessions revokes every refresh family for an account.
func (s *Store) RevokeAccountSessions(
	ctx context.Context,
	accountID uuid.UUID,
	entry audit.Entry,
	appender audit.Appender,
) error {
	if appender == nil {
		return errors.New("audit appender is required")
	}
	return s.InTx(ctx, func(tx *Tx) error {
		revoked, err := tx.RevokeAccountRefreshTokens(ctx, accountID)
		if err != nil {
			return err
		}
		if revoked == 0 {
			return nil
		}
		return appender.AppendTx(ctx, tx.Raw(), entry)
	})
}

// RotateRefreshToken validates and rotates a refresh token atomically. A
// presented token that was already used or revoked revokes the family and is
// reported as ErrRefreshTokenReused after the revocation commits.
func (s *Store) RotateRefreshToken(
	ctx context.Context,
	hash []byte,
	replacement sqlc.CreateRefreshTokenParams,
	refreshedEntry audit.Entry,
	reuseEntry audit.Entry,
	now time.Time,
	appender audit.Appender,
) (sqlc.Account, sqlc.RefreshToken, error) {
	if appender == nil {
		return sqlc.Account{}, sqlc.RefreshToken{}, errors.New("audit appender is required")
	}
	var account sqlc.Account
	var rotated sqlc.RefreshToken
	reused := false
	err := s.InTx(ctx, func(tx *Tx) error {
		stored, err := tx.GetRefreshTokenByHashForUpdate(ctx, hash)
		if err != nil {
			return err
		}
		if stored.UsedAt != nil || stored.RevokedAt != nil {
			if _, err := tx.RevokeRefreshTokenFamily(ctx, stored.FamilyID); err != nil {
				return err
			}
			if err := appender.AppendTx(ctx, tx.Raw(), reuseEntry); err != nil {
				return err
			}
			reused = true
			return nil
		}
		if !now.UTC().Before(stored.ExpiresAt) ||
			stored.SubjectType != "account" ||
			!validClient(stored.Client) {
			return ErrRefreshTokenInvalid
		}
		if replacement.ID == uuid.Nil || len(replacement.TokenHash) == 0 || replacement.ExpiresAt.IsZero() {
			return errors.New("refresh token replacement is incomplete")
		}
		account, err = tx.GetAccount(ctx, stored.SubjectID)
		if err != nil {
			if appErr, ok := apperrors.As(err); ok && appErr.HTTPStatus == 404 {
				return ErrRefreshTokenInvalid
			}
			return err
		}
		if account.Status != "active" || account.DeletedAt != nil {
			return ErrRefreshTokenInvalid
		}
		if stored.Client == ClientFleetConsole {
			if _, err := tx.ResolveActiveAdmin(ctx, stored.SubjectID); err != nil {
				if errors.Is(err, ErrAdminGrantNotFound) {
					return ErrRefreshTokenInvalid
				}
				return err
			}
		}
		replacement.SubjectType = stored.SubjectType
		replacement.SubjectID = stored.SubjectID
		replacement.FamilyID = stored.FamilyID
		replacement.Client = stored.Client
		rotated, err = tx.CreateRefreshToken(ctx, replacement)
		if err != nil {
			return err
		}
		if _, err := tx.MarkRefreshTokenUsed(ctx, stored.ID, &rotated.ID); err != nil {
			return err
		}
		return appender.AppendTx(ctx, tx.Raw(), refreshedEntry)
	})
	if err != nil {
		return sqlc.Account{}, sqlc.RefreshToken{}, err
	}
	if reused {
		return sqlc.Account{}, sqlc.RefreshToken{}, ErrRefreshTokenReused
	}
	return account, rotated, nil
}

// CreateSession inserts a refresh token and its session-issued audit row in
// one transaction.
func (s *Store) CreateSession(
	ctx context.Context,
	token sqlc.CreateRefreshTokenParams,
	entry audit.Entry,
	appender audit.Appender,
) error {
	if appender == nil {
		return errors.New("audit appender is required")
	}
	return s.InTx(ctx, func(tx *Tx) error {
		if token.Client == ClientFleetConsole {
			if _, err := tx.ResolveActiveAdmin(ctx, token.SubjectID); err != nil {
				if errors.Is(err, ErrAdminGrantNotFound) {
					return ErrClientForbidden
				}
				return err
			}
		}
		if _, err := tx.CreateRefreshToken(ctx, token); err != nil {
			return err
		}
		return appender.AppendTx(ctx, tx.Raw(), entry)
	})
}

// Tx is a transactional auth store handle.
type Tx struct {
	tx pgx.Tx
	q  *sqlc.Queries
}

// LockAccount serializes identity modifications for one account.
func (t *Tx) LockAccount(ctx context.Context, accountID uuid.UUID) error {
	var locked uuid.UUID
	err := t.tx.QueryRow(ctx, `SELECT id FROM accounts WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, accountID).Scan(&locked)
	return database.MapPgError(err)
}

// Raw exposes the underlying transaction for transaction-aware audit writes.
func (t *Tx) Raw() pgx.Tx {
	return t.tx
}

// Queries exposes generated queries scoped to this transaction.
func (t *Tx) Queries() *sqlc.Queries {
	return t.q
}

// GetAccount returns a non-deleted account by ID inside the transaction.
func (t *Tx) GetAccount(ctx context.Context, id uuid.UUID) (sqlc.Account, error) {
	row, err := t.q.GetAccount(ctx, id)
	return row, database.MapPgError(err)
}

// GetIdentityByProviderSubject resolves a linked identity inside the transaction.
func (t *Tx) GetIdentityByProviderSubject(ctx context.Context, provider, subject string) (sqlc.AccountIdentity, error) {
	return getIdentityByProviderSubject(ctx, t.q, provider, subject)
}

// CreateAccountWithIdentity creates the account and Firebase identity atomically.
func (t *Tx) CreateAccountWithIdentity(
	ctx context.Context,
	account sqlc.CreateAccountParams,
	identity sqlc.LinkAccountIdentityParams,
) (sqlc.Account, sqlc.AccountIdentity, error) {
	created, err := t.q.CreateAccount(ctx, account)
	if err != nil {
		return sqlc.Account{}, sqlc.AccountIdentity{}, mapAuthWriteError(err)
	}
	identity.AccountID = created.ID
	linked, err := t.q.LinkAccountIdentity(ctx, identity)
	if err != nil {
		return sqlc.Account{}, sqlc.AccountIdentity{}, mapAuthWriteError(err)
	}
	return created, linked, nil
}

// CreateIdentity links one provider identity to an existing account.
func (t *Tx) CreateIdentity(ctx context.Context, identity sqlc.LinkAccountIdentityParams) (sqlc.AccountIdentity, error) {
	linked, err := t.q.LinkAccountIdentity(ctx, identity)
	if err != nil {
		return sqlc.AccountIdentity{}, mapAuthWriteError(err)
	}
	return linked, nil
}

// DeleteIdentityForAccount deletes an identity only when owned by the account.
func (t *Tx) DeleteIdentityForAccount(ctx context.Context, identityID, accountID uuid.UUID) (int64, error) {
	deleted, err := t.q.DeleteAccountIdentityForAccount(ctx, sqlc.DeleteAccountIdentityForAccountParams{
		ID:        identityID,
		AccountID: accountID,
	})
	return deleted, database.MapPgError(err)
}

// GetActiveAdminByAccountID resolves an active Fleet grant inside the transaction.
func (t *Tx) GetActiveAdminByAccountID(ctx context.Context, accountID uuid.UUID) (sqlc.AdminUser, error) {
	return getActiveAdminByAccountID(ctx, t.q, accountID)
}

// ListAdminShipIDs returns an administrator's ship scope inside the transaction.
func (t *Tx) ListAdminShipIDs(ctx context.Context, adminUserID uuid.UUID) ([]uuid.UUID, error) {
	return listAdminShipIDs(ctx, t.q, adminUserID)
}

// ResolveActiveAdmin resolves the active grant and its scope inside the transaction.
func (t *Tx) ResolveActiveAdmin(ctx context.Context, accountID uuid.UUID) (AdminAuthority, error) {
	return resolveActiveAdmin(ctx, t.q, accountID)
}

// CreateRefreshToken inserts one refresh-token row.
func (t *Tx) CreateRefreshToken(ctx context.Context, params sqlc.CreateRefreshTokenParams) (sqlc.RefreshToken, error) {
	row, err := t.q.CreateRefreshToken(ctx, params)
	return row, database.MapPgError(err)
}

// GetRefreshTokenByHashForUpdate locks a refresh-token row by hash.
func (t *Tx) GetRefreshTokenByHashForUpdate(ctx context.Context, hash []byte) (sqlc.RefreshToken, error) {
	row, err := t.q.GetRefreshTokenByHashForUpdate(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.RefreshToken{}, ErrRefreshTokenNotFound
	}
	return row, database.MapPgError(err)
}

// MarkRefreshTokenUsed marks an unused row used and links its replacement.
func (t *Tx) MarkRefreshTokenUsed(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID) (sqlc.RefreshToken, error) {
	var replacement uuid.NullUUID
	if replacedBy != nil {
		replacement = uuid.NullUUID{UUID: *replacedBy, Valid: true}
	}
	row, err := t.q.MarkRefreshTokenUsed(ctx, sqlc.MarkRefreshTokenUsedParams{
		ID:         id,
		ReplacedBy: replacement,
	})
	return row, database.MapPgError(err)
}

// RevokeRefreshTokenFamily revokes every token in a session family.
func (t *Tx) RevokeRefreshTokenFamily(ctx context.Context, familyID uuid.UUID) (int64, error) {
	revoked, err := t.q.RevokeRefreshTokenFamily(ctx, familyID)
	return revoked, database.MapPgError(err)
}

// RevokeAccountRefreshTokens revokes every account refresh family.
func (t *Tx) RevokeAccountRefreshTokens(ctx context.Context, accountID uuid.UUID) (int64, error) {
	revoked, err := t.q.RevokeAccountRefreshTokens(ctx, accountID)
	return revoked, database.MapPgError(err)
}

func getIdentityByProviderSubject(ctx context.Context, q *sqlc.Queries, provider, subject string) (sqlc.AccountIdentity, error) {
	row, err := q.GetAccountIdentityByProviderSubject(ctx, sqlc.GetAccountIdentityByProviderSubjectParams{
		Provider:    provider,
		ProviderSub: subject,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.AccountIdentity{}, ErrIdentityNotFound
	}
	return row, database.MapPgError(err)
}

func getActiveAdminByAccountID(ctx context.Context, q *sqlc.Queries, accountID uuid.UUID) (sqlc.AdminUser, error) {
	row, err := q.GetActiveAdminUserByAccountID(ctx, uuid.NullUUID{UUID: accountID, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.AdminUser{}, ErrAdminGrantNotFound
	}
	return row, database.MapPgError(err)
}

func resolveActiveAdmin(ctx context.Context, q *sqlc.Queries, accountID uuid.UUID) (AdminAuthority, error) {
	admin, err := getActiveAdminByAccountID(ctx, q, accountID)
	if err != nil {
		return AdminAuthority{}, err
	}
	shipIDs, err := listAdminShipIDs(ctx, q, admin.ID)
	if err != nil {
		return AdminAuthority{}, err
	}
	return AdminAuthority{
		AdminUserID: admin.ID,
		Role:        admin.Role,
		TenantID:    admin.TenantID,
		ShipIDs:     shipIDs,
	}, nil
}

func listAdminShipIDs(ctx context.Context, q *sqlc.Queries, adminUserID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.ListAdminShipAccess(ctx, adminUserID)
	if err != nil {
		return nil, database.MapPgError(err)
	}
	shipIDs := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		shipIDs[i] = row.ShipID
	}
	return shipIDs, nil
}

func mapAuthWriteError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case hasConstraint(err, accountIdentityConflictConstraint):
		return ErrIdentityConflict
	case hasConstraint(err, accountEmailConflictConstraint):
		return ErrEmailConflict
	default:
		return database.MapPgError(err)
	}
}

func hasConstraint(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == pgerrcode.UniqueViolation &&
		pgErr.ConstraintName == constraint
}
