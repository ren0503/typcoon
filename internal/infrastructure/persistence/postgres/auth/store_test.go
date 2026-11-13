package auth

import (
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	appauth "github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/platform/errors"
)

func TestMapAuthWriteErrorClassifiesConstraints(t *testing.T) {
	identityErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: accountIdentityConflictConstraint}
	if !errors.Is(mapAuthWriteError(identityErr), appauth.ErrIdentityConflict) {
		t.Fatal("identity conflict was not classified")
	}
	emailErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: accountEmailConflictConstraint}
	if !errors.Is(mapAuthWriteError(emailErr), appauth.ErrEmailConflict) {
		t.Fatal("email conflict was not classified")
	}
	otherErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "other"}
	appErr, ok := apperrors.As(mapAuthWriteError(otherErr))
	if !ok || appErr.Code != apperrors.CodeConflict {
		t.Fatalf("other unique violation = %v", mapAuthWriteError(otherErr))
	}
}
