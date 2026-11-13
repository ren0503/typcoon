//go:build integration

package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ren0503/tycoon/internal/application/auth"
	authrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/auth"
	"github.com/ren0503/tycoon/internal/platform/testutil"
	httpserver "github.com/ren0503/tycoon/internal/transport/http"
	authhandlers "github.com/ren0503/tycoon/internal/transport/http/handlers/auth"
	"github.com/ren0503/tycoon/internal/transport/http/middleware"
)

func TestSensitiveValuesNeverAppearInResponsesOrLogs(t *testing.T) {
	h := testutil.StartPostgres(t)
	tenantID := h.SeedTenant(t)
	shipID := h.SeedShip(t, tenantID)
	departure := time.Now().UTC().Add(24 * time.Hour)
	voyageID := h.SeedVoyage(t, shipID, "REDACT-1", departure)
	bookingID := h.SeedBooking(t, shipID, voyageID)
	staffID := h.SeedStaff(t, tenantID, shipID, "EMP-1")

	token := "admin:redact:" + tenantID.String()
	provider, err := middleware.NewDevHeaderProvider([]string{token})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var logs bytes.Buffer
	modules := BuildModules(integrationModuleDeps(h, slog.New(slog.DiscardHandler)))
	router := httpserver.NewRouter(httpserver.RouterConfig{
		Logger:    slog.New(slog.NewTextHandler(&logs, nil)),
		DB:        contractPinger{},
		Provider:  provider,
		APIRoutes: modules.RegisterRoutes,
	})
	do := func(method, path string, body any) httpResult {
		var reader *bytes.Reader
		if body != nil {
			payload, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			reader = bytes.NewReader(payload)
		} else {
			reader = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("X-Dev-Principal", token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return httpResult{Status: rec.Code, Body: rec.Body.Bytes()}
	}

	ciphertext := append([]byte("TYC1"), bytes.Repeat([]byte{0x42}, 28)...)
	documentEnc := base64.StdEncoding.EncodeToString(ciphertext)
	create := do(http.MethodPost, "/api/bookings/"+bookingID.String()+"/guests", map[string]any{
		"full_name":             "Redacted Guest",
		"document_enc":          documentEnc,
		"emergency_contact_enc": documentEnc,
	})
	if create.Status != http.StatusCreated {
		t.Fatalf("create guest status = %d body=%s", create.Status, create.Body)
	}
	guestID, _ := create.decode(t)["id"].(string)
	list := do(http.MethodGet, "/api/bookings/"+bookingID.String()+"/guests", nil)
	if list.Status != http.StatusOK {
		t.Fatalf("list guests status = %d", list.Status)
	}

	pin := "918273"
	credential := do(http.MethodPut, "/api/staff/"+staffID.String()+"/credential", map[string]any{"pin": pin})
	if credential.Status != http.StatusOK {
		t.Fatalf("set credential status = %d body=%s", credential.Status, credential.Body)
	}
	metadata := do(http.MethodGet, "/api/staff/"+staffID.String()+"/credential", nil)
	if metadata.Status != http.StatusOK {
		t.Fatalf("get credential status = %d body=%s", metadata.Status, metadata.Body)
	}

	for _, result := range []httpResult{create, list, credential, metadata} {
		for _, forbidden := range []string{"document_enc", "emergency_contact_enc", documentEnc, pin, "pin_hash"} {
			if strings.Contains(string(result.Body), forbidden) {
				t.Fatalf("response leaked %q: %s", forbidden, result.Body)
			}
		}
	}
	for _, forbidden := range []string{documentEnc, pin, "pin_hash", "TYC1"} {
		if strings.Contains(logs.String(), forbidden) {
			t.Fatalf("log output leaked %q: %s", forbidden, logs.String())
		}
	}

	var storedHash string
	if err := h.Pool.QueryRow(t.Context(), `SELECT pin_hash FROM staff_credentials WHERE staff_id = $1`, staffID).Scan(&storedHash); err != nil {
		t.Fatalf("read pin hash: %v", err)
	}
	if storedHash == pin || strings.Contains(storedHash, pin) {
		t.Fatalf("pin stored without hashing: %q", storedHash)
	}
	_ = guestID
}

func TestAuthTokensNeverAppearInResponsesOrLogs(t *testing.T) {
	h := testutil.StartPostgres(t)
	const firebaseToken = "firebase-id-token-integration-secret"
	const email = "redact-auth@example.com"
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	clock := testutil.NewFakeClock(time.Now().UTC())
	_, keys := testutil.NewTestKeySet(t, "redact-kid")
	service, err := auth.NewService(auth.ServiceOptions{
		Store:      authrepo.NewStore(h.Pool),
		Verifier:   &testutil.FakeVerifier{Identities: map[string]auth.VerifiedIdentity{firebaseToken: {UID: "redact-auth-uid", Email: email, EmailVerified: true}}},
		Keys:       keys,
		Limiter:    &testutil.FakeRateLimiter{},
		Clock:      clock,
		Logger:     logger,
		Issuer:     "redact-issuer",
		Audience:   "redact-audience",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 720 * time.Hour,
		ClockSkew:  time.Minute,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	router := httpserver.NewRouter(httpserver.RouterConfig{
		Logger: logger,
		DB:     contractPinger{},
		PublicAPIRoutes: func(r chi.Router) {
			authhandlers.RegisterPublicRoutes(r, service)
		},
	})

	payload, err := json.Marshal(map[string]string{"firebase_id_token": firebaseToken, "client": "guest_mobile"})
	if err != nil {
		t.Fatalf("marshal exchange: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/firebase/exchange", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("exchange status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("exchange Cache-Control = %q", got)
	}
	if strings.Contains(rec.Body.String(), firebaseToken) {
		t.Fatal("exchange response leaked the Firebase ID Token")
	}
	var session struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	for _, secret := range []string{firebaseToken, session.AccessToken, session.RefreshToken, email} {
		if secret == "" {
			continue
		}
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("logs leaked %q: %s", secret, logs.String())
		}
	}

	jwksReq := httptest.NewRequest(http.MethodGet, "/api/auth/jwks", nil)
	jwksRec := httptest.NewRecorder()
	router.ServeHTTP(jwksRec, jwksReq)
	if jwksRec.Code != http.StatusOK {
		t.Fatalf("jwks status = %d body=%s", jwksRec.Code, jwksRec.Body.String())
	}
	if strings.Contains(jwksRec.Body.String(), `"d"`) || strings.Contains(jwksRec.Body.String(), `"p"`) {
		t.Fatalf("JWKS leaked private material: %s", jwksRec.Body.String())
	}
}
