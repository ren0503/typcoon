package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/database/sqlc"
)

func newRefreshFixture(t *testing.T, row *sqlc.RefreshToken, account sqlc.Account) (authFixture, *fakeServiceStore) {
	t.Helper()
	store := &fakeServiceStore{
		account:        account,
		refreshToken:   row,
		rotatedAccount: account,
	}
	if row != nil {
		store.rotatedToken = sqlc.RefreshToken{
			ID:          uuid.New(),
			SubjectType: row.SubjectType,
			SubjectID:   row.SubjectID,
			FamilyID:    row.FamilyID,
			Client:      row.Client,
		}
	}
	f := newAuthFixture(t, &unitVerifier{}, store)
	if row != nil {
		if row.TokenHash == nil {
			row.TokenHash = HashRefreshToken("current-refresh-token")
		}
		store.refreshToken = row
	}
	return f, store
}

func TestRefreshHashesTokenForLookupAndSessionLimit(t *testing.T) {
	const rawToken = "current-refresh-token"
	account := sqlc.Account{ID: uuid.New(), Status: "active"}
	familyID := uuid.New()
	row := &sqlc.RefreshToken{
		ID:          uuid.New(),
		SubjectType: "account",
		SubjectID:   account.ID,
		FamilyID:    familyID,
		Client:      ClientGuestMobile,
		ExpiresAt:   time.Date(2026, time.November, 5, 13, 0, 0, 0, time.UTC),
	}
	f, store := newRefreshFixture(t, row, account)

	session, err := f.service.Refresh(context.Background(), RefreshInput{
		RefreshToken: rawToken,
		UserAgent:    "guest-mobile/test",
		IPAddress:    "198.51.100.10",
	})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if !bytes.Equal(store.refreshLookupHash, HashRefreshToken(rawToken)) {
		t.Fatal("refresh lookup did not use the token hash")
	}
	if bytes.Equal(store.refreshLookupHash, []byte(rawToken)) {
		t.Fatal("raw refresh token was used for lookup")
	}
	if len(f.limiter.calls) != 2 {
		t.Fatalf("rate-limit calls = %d, want 2", len(f.limiter.calls))
	}
	if f.limiter.calls[0].key != "refresh:ip:198.51.100.10" {
		t.Fatalf("IP limit key = %q", f.limiter.calls[0].key)
	}
	if !strings.HasPrefix(f.limiter.calls[1].key, "refresh:session:") {
		t.Fatalf("session limit key = %q", f.limiter.calls[1].key)
	}
	if strings.Contains(f.limiter.calls[1].key, rawToken) {
		t.Fatal("session limit key contains the raw refresh token")
	}
	if len(store.replacement.TokenHash) == 0 || bytes.Equal(store.replacement.TokenHash, []byte(session.RefreshToken)) {
		t.Fatal("replacement refresh token was not hashed")
	}
	if !bytes.Equal(store.replacement.TokenHash, HashRefreshToken(session.RefreshToken)) {
		t.Fatal("replacement hash does not match returned refresh token")
	}
	if store.replacement.ID == uuid.Nil {
		t.Fatal("replacement token id is missing")
	}
	if !store.replacement.ExpiresAt.Equal(f.clock.Now().Add(30 * 24 * time.Hour)) {
		t.Fatalf("replacement expiry = %s", store.replacement.ExpiresAt)
	}
	if store.replacement.UserAgent == nil || *store.replacement.UserAgent != "guest-mobile/test" {
		t.Fatalf("replacement user agent = %v", store.replacement.UserAgent)
	}
	if session.SessionID != familyID || session.AccountID != account.ID || session.Client != ClientGuestMobile {
		t.Fatalf("session identity = %+v", session)
	}
	if !session.AccessExpiresAt.Equal(f.clock.Now().Add(15 * time.Minute)) {
		t.Fatalf("access expiry = %s", session.AccessExpiresAt)
	}
	claims, err := f.validator.Validate(context.Background(), session.AccessToken)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.Subject != account.ID.String() || claims.Client != ClientGuestMobile {
		t.Fatalf("claims = subject %q client %q", claims.Subject, claims.Client)
	}
	if len(store.rotatedAudit) != 1 || store.rotatedAudit[0].Action != ActionSessionRefreshed {
		t.Fatalf("refresh audit = %+v", store.rotatedAudit)
	}
	if len(store.reuseAudit) != 1 || store.reuseAudit[0].Action != ActionSessionReuse {
		t.Fatalf("reuse audit = %+v", store.reuseAudit)
	}
	for _, entry := range append(store.rotatedAudit, store.reuseAudit...) {
		encoded, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("marshal audit: %v", err)
		}
		if bytes.Contains(encoded, []byte(rawToken)) || bytes.Contains(encoded, []byte(session.RefreshToken)) ||
			strings.Contains(strings.ToLower(string(encoded)), "hash") ||
			strings.Contains(strings.ToLower(string(encoded)), "family") {
			t.Fatalf("audit entry contains credential material: %s", encoded)
		}
	}
}

func TestRefreshRejectsExpiredAndUnknownTokens(t *testing.T) {
	account := sqlc.Account{ID: uuid.New(), Status: "active"}
	row := &sqlc.RefreshToken{
		ID:          uuid.New(),
		SubjectType: "account",
		SubjectID:   account.ID,
		FamilyID:    uuid.New(),
		Client:      ClientGuestMobile,
		ExpiresAt:   time.Date(2026, time.November, 5, 12, 0, 0, 0, time.UTC),
	}
	f, store := newRefreshFixture(t, row, account)

	if _, err := f.service.Refresh(context.Background(), RefreshInput{RefreshToken: "current-refresh-token"}); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("expired error = %v", err)
	}
	if store.rotateCalls != 0 {
		t.Fatalf("expired token reached rotation %d times", store.rotateCalls)
	}

	unknown := &fakeServiceStore{}
	unknownFixture := newAuthFixture(t, &unitVerifier{}, unknown)
	if _, err := unknownFixture.service.Refresh(context.Background(), RefreshInput{RefreshToken: "unknown"}); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("unknown error = %v", err)
	}
	if unknown.rotateCalls != 0 {
		t.Fatalf("unknown token reached rotation %d times", unknown.rotateCalls)
	}

	if _, err := f.service.Refresh(context.Background(), RefreshInput{RefreshToken: "   "}); !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("empty error = %v", err)
	}
}

func TestRefreshReuseReturnsGenericErrorAndDoesNotIssueToken(t *testing.T) {
	const rawToken = "used-refresh-token"
	familyID := uuid.New()
	account := sqlc.Account{ID: uuid.New(), Status: "active"}
	usedAt := time.Date(2026, time.November, 5, 12, 1, 0, 0, time.UTC)
	row := &sqlc.RefreshToken{
		ID:          uuid.New(),
		SubjectType: "account",
		SubjectID:   account.ID,
		FamilyID:    familyID,
		Client:      ClientGuestMobile,
		ExpiresAt:   time.Date(2026, time.November, 5, 13, 0, 0, 0, time.UTC),
		UsedAt:      &usedAt,
	}
	f, store := newRefreshFixture(t, row, account)
	store.rotateErr = ErrRefreshTokenReused

	_, err := f.service.Refresh(context.Background(), RefreshInput{RefreshToken: rawToken})
	if !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("reuse error = %v, want generic authentication failure", err)
	}
	if store.rotateCalls != 1 {
		t.Fatalf("reuse rotation calls = %d, want 1", store.rotateCalls)
	}
	if len(store.replacement.TokenHash) != 0 {
		t.Fatal("reuse path generated a replacement token")
	}
	if len(store.reuseAudit) != 1 || store.reuseAudit[0].Action != ActionSessionReuse {
		t.Fatalf("reuse audit = %+v", store.reuseAudit)
	}
}

func TestRefreshRejectsInactiveAccountWithoutRotation(t *testing.T) {
	for _, status := range []string{"suspended", "deleted"} {
		t.Run(status, func(t *testing.T) {
			account := sqlc.Account{ID: uuid.New(), Status: status}
			row := &sqlc.RefreshToken{
				ID:          uuid.New(),
				SubjectType: "account",
				SubjectID:   account.ID,
				FamilyID:    uuid.New(),
				Client:      ClientGuestMobile,
				ExpiresAt:   time.Date(2026, time.November, 5, 13, 0, 0, 0, time.UTC),
			}
			f, store := newRefreshFixture(t, row, account)
			if _, err := f.service.Refresh(context.Background(), RefreshInput{RefreshToken: "current-refresh-token"}); !errors.Is(err, ErrAuthenticationFailed) {
				t.Fatalf("inactive account error = %v", err)
			}
			if store.rotateCalls != 0 {
				t.Fatalf("inactive account reached rotation %d times", store.rotateCalls)
			}
		})
	}
}

func TestRefreshRejectsNonAccountOrInvalidClientRows(t *testing.T) {
	for _, row := range []*sqlc.RefreshToken{
		{ID: uuid.New(), SubjectType: "admin", SubjectID: uuid.New(), FamilyID: uuid.New(), Client: ClientGuestMobile, ExpiresAt: time.Now().Add(time.Hour)},
		{ID: uuid.New(), SubjectType: "account", SubjectID: uuid.New(), FamilyID: uuid.New(), Client: "unknown", ExpiresAt: time.Now().Add(time.Hour)},
	} {
		store := &fakeServiceStore{refreshToken: row}
		f := newAuthFixture(t, &unitVerifier{}, store)
		if _, err := f.service.Refresh(context.Background(), RefreshInput{RefreshToken: "current-refresh-token"}); !errors.Is(err, ErrAuthenticationFailed) {
			t.Fatalf("row %+v error = %v", row, err)
		}
		if store.rotateCalls != 0 {
			t.Fatalf("invalid row reached rotation %d times", store.rotateCalls)
		}
	}
}
