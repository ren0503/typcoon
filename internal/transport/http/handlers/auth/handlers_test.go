package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ren0503/tycoon/internal/infrastructure/security/local"
)

type jwksOnlyService struct {
	Service
	keys []JWK
	err  error
}

func (s jwksOnlyService) JWKS(context.Context) ([]JWK, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.keys, nil
}

func TestJWKSHandlerPublishesOnlyPublicKeysWithPublicCache(t *testing.T) {
	signer, err := local.NewRandomLocalSigner("active-key")
	if err != nil {
		t.Fatalf("NewRandomLocalSigner: %v", err)
	}
	keySet, err := local.NewStaticKeySet("active-key", signer)
	if err != nil {
		t.Fatalf("NewStaticKeySet: %v", err)
	}
	keys, err := keySet.PublicJWKS(context.Background())
	if err != nil {
		t.Fatalf("PublicJWKS: %v", err)
	}
	service := jwksOnlyService{keys: keys}
	router := chi.NewRouter()
	RegisterPublicRoutes(router, service)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/jwks", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q", got)
	}
	cacheControl := rec.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "public") || !strings.Contains(cacheControl, "max-age") {
		t.Fatalf("Cache-Control = %q, want public max-age", cacheControl)
	}
	if strings.Contains(rec.Body.String(), "no-store") {
		t.Fatal("JWKS response must be publicly cacheable")
	}

	var body struct {
		Keys []JWK `json:"keys"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode JWKS: %v", err)
	}
	if len(body.Keys) != 1 {
		t.Fatalf("keys = %d, want 1", len(body.Keys))
	}
	key := body.Keys[0]
	if key.KID != "active-key" || key.Kty != "RSA" || key.Use != "sig" || key.Alg != "RS256" {
		t.Fatalf("unexpected JWK: %+v", key)
	}
	if key.N == "" || key.E == "" {
		t.Fatalf("JWK missing public parameters: %+v", key)
	}
	for _, forbidden := range []string{`"d"`, `"p"`, `"q"`, `"dp"`, `"dq"`, `"qi"`} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("JWKS leaked private field %s: %s", forbidden, rec.Body.String())
		}
	}
}

func TestJWKSHandlerFailsClosedWhenKeysUnavailable(t *testing.T) {
	service := jwksOnlyService{err: ErrSessionUnavailable}
	router := chi.NewRouter()
	RegisterPublicRoutes(router, service)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/jwks", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body %s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Retry-After"); got == "" {
		t.Fatal("missing Retry-After on unavailable response")
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}
