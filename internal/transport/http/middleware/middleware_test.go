package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/ren0503/tycoon/internal/transport/http/middleware"
)

func TestRequestLoggerEmitsStructuredFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := chimiddleware.RequestID(middleware.RequestLogger(logger)(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("ok"))
		},
	)))

	req := httptest.NewRequest(http.MethodGet, "/test?token=secret", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logged := buf.String()
	for _, want := range []string{
		"request_id=",
		"method=GET",
		"path=/test",
		"status=201",
		"duration_ms=",
		"remote_ip=192.0.2.1:1234",
		"user_agent=test-agent",
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("log does not contain %q: %s", want, logged)
		}
	}
	if strings.Contains(logged, "secret") {
		t.Errorf("log leaked the query string: %s", logged)
	}
}

func TestRequestLoggerDefaultsStatusToOK(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := middleware.RequestLogger(logger)(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		},
	))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !strings.Contains(buf.String(), "status=200") {
		t.Errorf("log does not contain status=200: %s", buf.String())
	}
}

func TestRecovererReturnsGenericError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := middleware.Recoverer(logger)(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {
			panic("sensitive panic value")
		},
	))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "internal_error") {
		t.Fatalf("body = %q", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "sensitive panic value") {
		t.Fatalf("response leaked panic value: %q", rec.Body.String())
	}
	if !strings.Contains(buf.String(), "panic recovered") {
		t.Fatalf("log = %q", buf.String())
	}
}

func TestCORSAllowsListedOrigin(t *testing.T) {
	handler := middleware.CORS([]string{"https://console.example.com"})(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://console.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://console.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestCORSRejectsUnlistedOrigin(t *testing.T) {
	handler := middleware.CORS(nil)(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}
