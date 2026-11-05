//go:build integration

package httpserver

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

	"github.com/ren0503/tycoon/internal/app"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/testutil"
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
	provider, err := auth.NewDevHeaderProvider([]string{token})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var logs bytes.Buffer
	modules := app.BuildModules(h.Pool, slog.New(slog.DiscardHandler), nil)
	router := NewRouter(RouterConfig{
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
	create := do(http.MethodPost, "/v1/bookings/"+bookingID.String()+"/guests", map[string]any{
		"full_name":             "Redacted Guest",
		"document_enc":          documentEnc,
		"emergency_contact_enc": documentEnc,
	})
	if create.Status != http.StatusCreated {
		t.Fatalf("create guest status = %d body=%s", create.Status, create.Body)
	}
	guestID, _ := create.decode(t)["id"].(string)
	list := do(http.MethodGet, "/v1/bookings/"+bookingID.String()+"/guests", nil)
	if list.Status != http.StatusOK {
		t.Fatalf("list guests status = %d", list.Status)
	}

	pin := "918273"
	credential := do(http.MethodPut, "/v1/staff/"+staffID.String()+"/credential", map[string]any{"pin": pin})
	if credential.Status != http.StatusOK {
		t.Fatalf("set credential status = %d body=%s", credential.Status, credential.Body)
	}
	metadata := do(http.MethodGet, "/v1/staff/"+staffID.String()+"/credential", nil)
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
