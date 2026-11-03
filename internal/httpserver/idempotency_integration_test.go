//go:build integration

package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/app"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/testutil"
)

func TestIdempotencyHeadersAreHonoured(t *testing.T) {
	h := testutil.StartPostgres(t)
	tenantID := h.SeedTenant(t)
	shipID := h.SeedShip(t, tenantID)
	departure := time.Now().UTC().Add(48 * time.Hour)
	voyageID := h.SeedVoyage(t, shipID, "IDEM-1", departure)
	accountID := h.SeedAccount(t)
	partnerID := h.SeedPartner(t, tenantID)
	partnerServiceID := h.SeedPartnerService(t, shipID, partnerID)

	token := "admin:idem:" + tenantID.String()
	provider, err := auth.NewDevHeaderProvider([]string{token})
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
	do := func(path, key string, body any) httpResult {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Dev-Principal", token)
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return httpResult{Status: rec.Code, Body: rec.Body.Bytes()}
	}

	bookingBody := map[string]any{
		"ship_id":       shipID.String(),
		"voyage_id":     voyageID.String(),
		"account_id":    accountID.String(),
		"embark_seq":    1,
		"disembark_seq": 5,
	}
	first := do("/v1/bookings", "http-booking-1", bookingBody)
	if first.Status != http.StatusCreated {
		t.Fatalf("first booking status = %d body=%s", first.Status, first.Body)
	}
	replay := do("/v1/bookings", "http-booking-1", bookingBody)
	if replay.Status != http.StatusCreated {
		t.Fatalf("replay booking status = %d body=%s", replay.Status, replay.Body)
	}
	firstID := first.decode(t)["id"]
	replayID := replay.decode(t)["id"]
	if firstID != replayID {
		t.Fatalf("idempotent booking ids differ: %v vs %v", firstID, replayID)
	}

	changedBody := map[string]any{
		"ship_id":       shipID.String(),
		"voyage_id":     voyageID.String(),
		"account_id":    accountID.String(),
		"embark_seq":    1,
		"disembark_seq": 6,
	}
	conflict := do("/v1/bookings", "http-booking-1", changedBody)
	if conflict.Status != http.StatusConflict {
		t.Fatalf("changed payload status = %d body=%s", conflict.Status, conflict.Body)
	}
	if code := errorCode(conflict.decode(t)); code != "idempotency_conflict" {
		t.Fatalf("changed payload code = %q", code)
	}

	serviceBody := map[string]any{
		"booking_id":         firstID,
		"partner_service_id": partnerServiceID.String(),
		"quantity":           1,
	}
	serviceFirst := do("/v1/service-bookings", "http-service-1", serviceBody)
	if serviceFirst.Status != http.StatusCreated {
		t.Fatalf("first service booking status = %d body=%s", serviceFirst.Status, serviceFirst.Body)
	}
	serviceReplay := do("/v1/service-bookings", "http-service-1", serviceBody)
	if serviceReplay.Status != http.StatusCreated {
		t.Fatalf("service replay status = %d body=%s", serviceReplay.Status, serviceReplay.Body)
	}
	if serviceFirst.decode(t)["id"] != serviceReplay.decode(t)["id"] {
		t.Fatal("idempotent service booking ids differ")
	}
	serviceBody["quantity"] = 2
	serviceConflict := do("/v1/service-bookings", "http-service-1", serviceBody)
	if serviceConflict.Status != http.StatusConflict {
		t.Fatalf("service changed payload status = %d body=%s", serviceConflict.Status, serviceConflict.Body)
	}
	if code := errorCode(serviceConflict.decode(t)); code != "idempotency_conflict" {
		t.Fatalf("service changed payload code = %q", code)
	}
}
