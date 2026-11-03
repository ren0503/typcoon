//go:build integration

package httpserver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/app"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/pagination"
	"github.com/ren0503/tycoon/internal/testutil"
)

type httpResult struct {
	Status int
	Body   []byte
}

func (r httpResult) decode(t *testing.T) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(r.Body, &out); err != nil {
		t.Fatalf("decode response %q: %v", string(r.Body), err)
	}
	return out
}

func errorCode(body map[string]any) string {
	raw, ok := body["error"].(map[string]any)
	if !ok {
		return ""
	}
	code, _ := raw["code"].(string)
	return code
}

func TestAuthenticationTenantIsolationAndSensitiveFields(t *testing.T) {
	h := testutil.StartPostgres(t)
	tenantA := h.SeedTenant(t)
	tenantB := h.SeedTenant(t)
	shipA := h.SeedShip(t, tenantA)
	shipB := h.SeedShip(t, tenantB)
	departure := time.Now().UTC().Add(24 * time.Hour)
	voyageA := h.SeedVoyage(t, shipA, "SEC-1", departure)
	voyageB := h.SeedVoyage(t, shipB, "SEC-2", departure)
	bookingA := h.SeedBooking(t, shipA, voyageA)

	tokenA := "admin:sec-a:" + tenantA.String()
	tokenB := "admin:sec-b:" + tenantB.String()
	shipToken := "admin:sec-ship:" + tenantA.String() + ":" + shipA.String()
	provider, err := auth.NewDevHeaderProvider([]string{tokenA, tokenB, shipToken})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	modules := app.BuildModules(h.Pool)
	router := NewRouter(RouterConfig{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:        contractPinger{},
		Provider:  provider,
		APIRoutes: modules.RegisterRoutes,
	})
	do := func(method, path, token string, body []byte) httpResult {
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			req.Header.Set("X-Dev-Principal", token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return httpResult{Status: rec.Code, Body: rec.Body.Bytes()}
	}

	if res := do(http.MethodGet, "/v1/ships", "", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", res.Status)
	} else if code := errorCode(res.decode(t)); code != "unauthorized" {
		t.Fatalf("unauthenticated code = %q", code)
	}
	if res := do(http.MethodGet, "/v1/ships", "admin:unknown:"+tenantA.String(), nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("unknown principal status = %d, want 401", res.Status)
	}

	if res := do(http.MethodGet, "/v1/ships/"+shipB.String(), tokenA, nil); res.Status != http.StatusForbidden {
		t.Fatalf("cross-tenant read status = %d, want 403", res.Status)
	} else if code := errorCode(res.decode(t)); code != "forbidden" {
		t.Fatalf("cross-tenant code = %q", code)
	}
	if res := do(http.MethodGet, "/v1/ships/"+shipA.String(), tokenA, nil); res.Status != http.StatusOK {
		t.Fatalf("own-tenant read status = %d, want 200", res.Status)
	}

	res := do(http.MethodGet, "/v1/ships", tokenA, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("list ships status = %d", res.Status)
	}
	body := res.decode(t)
	items, _ := body["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("tenant A sees %d ships, want 1", len(items))
	}
	if id, _ := items[0].(map[string]any)["id"].(string); id != shipA.String() {
		t.Fatalf("tenant A saw ship %s, want %s", id, shipA)
	}

	if res := do(http.MethodGet, "/v1/voyages/"+voyageB.String(), shipToken, nil); res.Status != http.StatusForbidden {
		t.Fatalf("ship-scoped foreign voyage status = %d, want 403", res.Status)
	}
	if res := do(http.MethodGet, "/v1/voyages/"+voyageA.String(), shipToken, nil); res.Status != http.StatusOK {
		t.Fatalf("ship-scoped own voyage status = %d, want 200", res.Status)
	}

	if res := do(http.MethodGet, "/v1/ships?cursor=not-a-cursor", tokenA, nil); res.Status != http.StatusBadRequest {
		t.Fatalf("malformed cursor status = %d, want 400", res.Status)
	} else if code := errorCode(res.decode(t)); code != "invalid_cursor" {
		t.Fatalf("malformed cursor code = %q", code)
	}
	foreignCursor := pagination.Cursor{Filter: "other-filter", ID: uuid.New()}.Encode()
	if res := do(http.MethodGet, "/v1/ships?cursor="+foreignCursor, tokenA, nil); res.Status != http.StatusBadRequest {
		t.Fatalf("foreign-filter cursor status = %d, want 400", res.Status)
	} else if code := errorCode(res.decode(t)); code != "invalid_cursor" {
		t.Fatalf("foreign-filter cursor code = %q", code)
	}
	if res := do(http.MethodGet, "/v1/ships?limit=500", tokenA, nil); res.Status != http.StatusOK {
		t.Fatalf("limit 500 status = %d, want 200", res.Status)
	}

	ciphertext := append([]byte("TYC1"), make([]byte, 28)...)
	documentEnc := base64.StdEncoding.EncodeToString(ciphertext)
	guestBody, _ := json.Marshal(map[string]any{
		"full_name":    "Secret Guest",
		"document_enc": documentEnc,
	})
	res = do(http.MethodPost, "/v1/bookings/"+bookingA.String()+"/guests", tokenA, guestBody)
	if res.Status != http.StatusCreated {
		t.Fatalf("create guest status = %d body=%s", res.Status, res.Body)
	}
	if strings.Contains(string(res.Body), "document_enc") || strings.Contains(string(res.Body), documentEnc) {
		t.Fatalf("guest response leaked ciphertext: %s", res.Body)
	}
	created := res.decode(t)
	guestID, _ := created["id"].(string)
	res = do(http.MethodGet, "/v1/guests/"+guestID, tokenA, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("read guest status = %d", res.Status)
	}
	if strings.Contains(string(res.Body), "document_enc") || strings.Contains(string(res.Body), "passport") {
		t.Fatalf("guest read leaked sensitive fields: %s", res.Body)
	}

	plaintextBody, _ := json.Marshal(map[string]any{
		"full_name": "Plaintext Guest",
		"document":  "passport-secret",
	})
	res = do(http.MethodPost, "/v1/bookings/"+bookingA.String()+"/guests", tokenA, plaintextBody)
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("plaintext document status = %d, want 422 body=%s", res.Status, res.Body)
	}
	if code := errorCode(res.decode(t)); code != "business_rule_violation" {
		t.Fatalf("plaintext document code = %q", code)
	}
}
