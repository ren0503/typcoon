//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/go-chi/chi/v5"

	"github.com/ren0503/tycoon/internal/app"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/testutil"
)

func TestDeclaredOperationsAreMounted(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatalf("load openapi: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("openapi is invalid: %v", err)
	}

	modules := app.BuildModules(nil, slog.New(slog.DiscardHandler), nil)
	router := NewRouter(RouterConfig{
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:              contractPinger{},
		PublicAPIRoutes: modules.RegisterPublicRoutes,
		APIRoutes:       modules.RegisterRoutes,
	})
	walker, ok := router.(chi.Routes)
	if !ok {
		t.Fatal("router does not implement chi.Routes")
	}

	mounted := make(map[string]bool)
	if err := chi.Walk(walker, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		mounted[method+" "+normalizeRoute(route)] = true
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			key := strings.ToUpper(method) + " " + normalizeRoute("/v1"+path)
			if !mounted[key] {
				t.Errorf("openapi declares %s but the router does not mount it", key)
			}
		}
	}
}

type contractPinger struct{}

func (contractPinger) Ping(context.Context) error { return nil }

func normalizeRoute(route string) string {
	trimmed := strings.TrimSuffix(route, "/")
	if trimmed == "" {
		return "/"
	}
	return trimmed
}

func TestExercisedRequestsAndResponsesMatchSpec(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatalf("load openapi: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("openapi is invalid: %v", err)
	}

	h := testutil.StartPostgres(t)
	tenantID := h.SeedTenant(t)
	shipID := h.SeedShip(t, tenantID)
	token := "admin:contract:" + tenantID.String()
	provider, err := auth.NewDevHeaderProvider([]string{token})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	modules := app.BuildModules(h.Pool, slog.New(slog.DiscardHandler), nil)
	router := NewRouter(RouterConfig{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:        contractPinger{},
		Provider:  provider,
		APIRoutes: modules.RegisterRoutes,
	})

	router2, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("spec router: %v", err)
	}
	exercise := func(path string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Dev-Principal", token)
		specReq := req.Clone(req.Context())
		specReq.URL.Path = strings.TrimPrefix(req.URL.Path, "/v1")
		route, pathParams, err := router2.FindRoute(specReq)
		if err != nil {
			t.Fatalf("%s not found in spec: %v", path, err)
		}
		requestInput := &openapi3filter.RequestValidationInput{
			Request: specReq, PathParams: pathParams, Route: route,
		}
		if err := openapi3filter.ValidateRequest(context.Background(), requestInput); err != nil {
			t.Fatalf("request %s violates spec: %v", path, err)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		responseInput := &openapi3filter.ResponseValidationInput{
			RequestValidationInput: requestInput,
			Status:                 rec.Code,
			Header:                 rec.Header(),
			Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		}
		if err := openapi3filter.ValidateResponse(context.Background(), responseInput); err != nil {
			t.Fatalf("response for %s (status %d) violates spec: %v", path, rec.Code, err)
		}
	}

	exercise("/v1/ships")
	exercise("/v1/ships/" + shipID.String())
	exercise("/v1/tenants")
}
