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
	"github.com/getkin/kin-openapi/routers"

	"github.com/ren0503/tycoon/internal/app"
	"github.com/ren0503/tycoon/internal/auth"
)

func TestStructureImportTemplateMatchesSpec(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatalf("load openapi: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("openapi is invalid: %v", err)
	}

	token := "admin:template:00000000-0000-0000-0000-000000000001"
	provider, err := auth.NewDevHeaderProvider([]string{token})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	modules := app.BuildModules(nil, slog.New(slog.DiscardHandler), nil)
	router := NewRouter(RouterConfig{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:        contractPinger{},
		Provider:  provider,
		APIRoutes: modules.RegisterRoutes,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ships/structure/import-template", nil)
	req.Header.Set("X-Dev-Principal", token)
	specPath := strings.TrimPrefix(req.URL.Path, "/api")
	pathItem := doc.Paths.Find(specPath)
	if pathItem == nil || pathItem.Get == nil {
		t.Fatalf("template path not in spec: %s", specPath)
	}
	route := &routers.Route{
		Spec: doc, Method: http.MethodGet, Path: specPath, PathItem: pathItem, Operation: pathItem.Get,
	}
	specReq := req.Clone(req.Context())
	specReq.URL.Path = specPath
	requestInput := &openapi3filter.RequestValidationInput{
		Request: specReq, PathParams: map[string]string{}, Route: route,
		Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
	}
	if err := openapi3filter.ValidateRequest(context.Background(), requestInput); err != nil {
		t.Fatalf("template request violates spec: %v", err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("template status = %d, body %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("content type = %q", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="ship-structure-import-template.xlsx"` {
		t.Fatalf("content disposition = %q", got)
	}
	responseInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestInput,
		Status:                 rec.Code,
		Header:                 rec.Header(),
		Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
	}
	if err := openapi3filter.ValidateResponse(context.Background(), responseInput); err != nil {
		t.Fatalf("template response violates spec: %v", err)
	}
}
