package health_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ren0503/tycoon/internal/transport/http/handlers/health"
)

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error {
	return f.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	health.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("body = %q", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q", got)
	}
}

func TestReadyWhenDatabaseIsAvailable(t *testing.T) {
	handler := health.Ready(fakePinger{}, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != `{"status":"ready"}` {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestReadyWhenDatabaseIsUnavailable(t *testing.T) {
	handler := health.Ready(fakePinger{err: errors.New("connection refused")}, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != `{"status":"not_ready"}` {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHealthResponsesDoNotLeakInternalDetails(t *testing.T) {
	sensitive := errors.New("password=super-secret host=db.internal user=admin")
	handler := health.Ready(fakePinger{err: sensitive}, testLogger())
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	body := rec.Body.String()
	for _, forbidden := range []string{"password", "super-secret", "db.internal", "admin"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, body)
		}
	}
}
