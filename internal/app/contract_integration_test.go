//go:build integration

package app

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

	airepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/ai"
	authrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/auth"
	bookingrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/booking"
	feedbackrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/feedback"
	guestrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/guest"
	partnerrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/partner"
	shiprepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/ship"
	staffrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/staff"
	syncrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sync"
	tenantrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/tenant"
	voyagerepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/voyage"
	"github.com/ren0503/tycoon/internal/platform/testutil"
	httpserver "github.com/ren0503/tycoon/internal/transport/http"
	"github.com/ren0503/tycoon/internal/transport/http/middleware"
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

	modules := BuildModules(ModuleDeps{Logger: slog.New(slog.DiscardHandler)})
	router := httpserver.NewRouter(httpserver.RouterConfig{
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
			key := strings.ToUpper(method) + " " + normalizeRoute("/api"+path)
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
	provider, err := middleware.NewDevHeaderProvider([]string{token})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	logger := slog.New(slog.DiscardHandler)
	modules := BuildModules(ModuleDeps{
		Logger:       logger,
		Accounts:     authrepo.NewAccountService(h.Pool),
		TenantRepo:   tenantrepo.New(h.Pool),
		ShipRepo:     shiprepo.New(h.Pool),
		VoyageRepo:   voyagerepo.New(h.Pool),
		StaffRepo:    staffrepo.New(h.Pool),
		PartnerRepo:  partnerrepo.New(h.Pool),
		BookingRepo:  bookingrepo.New(h.Pool),
		GuestRepo:    guestrepo.New(h.Pool),
		AIRepo:       airepo.New(h.Pool),
		FeedbackRepo: feedbackrepo.New(h.Pool),
		SyncRepo:     syncrepo.New(h.Pool),
	})
	router := httpserver.NewRouter(httpserver.RouterConfig{
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
		specReq.URL.Path = req.URL.Path
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

	exercise("/api/ships")
	exercise("/api/ships/" + shipID.String())
	exercise("/api/tenants")
}
