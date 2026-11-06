//go:build integration

package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/auth"
)

type exchangeContractService struct {
	auth.Service
	exchangeCalls    int
	refreshCalls     int
	logoutCalls      int
	logoutToken      string
	logoutAllCalls   int
	logoutAllAccount uuid.UUID
	linkCalls        int
	linkAccount      uuid.UUID
	linkToken        string
	linkIdentity     auth.Identity
	unlinkCalls      int
	unlinkAccount    uuid.UUID
	unlinkIdentityID uuid.UUID
	err              error
	refreshErr       error
	logoutErr        error
	logoutAllErr     error
	linkErr          error
	unlinkErr        error
	jwksKeys         []auth.JWK
	jwksErr          error
}

func (s *exchangeContractService) Exchange(context.Context, auth.ExchangeInput) (auth.IssuedSession, error) {
	s.exchangeCalls++
	if s.err != nil {
		return auth.IssuedSession{}, s.err
	}
	return contractSession(), nil
}

func (s *exchangeContractService) Refresh(context.Context, auth.RefreshInput) (auth.IssuedSession, error) {
	s.refreshCalls++
	if s.refreshErr != nil {
		return auth.IssuedSession{}, s.refreshErr
	}
	return contractSession(), nil
}

func (s *exchangeContractService) Logout(_ context.Context, refreshToken string) error {
	s.logoutCalls++
	s.logoutToken = refreshToken
	return s.logoutErr
}

func (s *exchangeContractService) LogoutAll(_ context.Context, accountID uuid.UUID) error {
	s.logoutAllCalls++
	s.logoutAllAccount = accountID
	return s.logoutAllErr
}

func (s *exchangeContractService) LinkIdentity(_ context.Context, accountID uuid.UUID, idToken string) (auth.Identity, error) {
	s.linkCalls++
	s.linkAccount = accountID
	s.linkToken = idToken
	if s.linkErr != nil {
		return auth.Identity{}, s.linkErr
	}
	if s.linkIdentity.ID == uuid.Nil {
		s.linkIdentity = auth.Identity{
			ID:        uuid.MustParse("66666666-6666-4666-8666-666666666666"),
			AccountID: accountID,
			Provider:  "firebase",
			CreatedAt: time.Now().UTC(),
		}
	}
	return s.linkIdentity, nil
}

func (s *exchangeContractService) UnlinkIdentity(_ context.Context, accountID, identityID uuid.UUID) error {
	s.unlinkCalls++
	s.unlinkAccount = accountID
	s.unlinkIdentityID = identityID
	return s.unlinkErr
}

func (s *exchangeContractService) JWKS(context.Context) ([]auth.JWK, error) {
	if s.jwksErr != nil {
		return nil, s.jwksErr
	}
	return s.jwksKeys, nil
}

func contractSession() auth.IssuedSession {
	now := time.Now().UTC()
	return auth.IssuedSession{
		SessionID:        uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		AccountID:        uuid.MustParse("22222222-2222-4222-8222-222222222222"),
		Client:           auth.ClientGuestMobile,
		AccessToken:      "backend-access-token",
		AccessExpiresAt:  now.Add(15 * time.Minute),
		RefreshToken:     "backend-refresh-token",
		RefreshExpiresAt: now.Add(720 * time.Hour),
	}
}

func TestFirebaseExchangeContract(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatalf("load openapi: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("openapi is invalid: %v", err)
	}
	router2, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("spec router: %v", err)
	}

	service := &exchangeContractService{}
	router := NewRouter(RouterConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:     contractPinger{},
		PublicAPIRoutes: func(r chi.Router) {
			auth.RegisterPublicRoutes(r, service)
		},
	})

	exchange := func(body, contentType string, validateRequest bool) (*httptest.ResponseRecorder, *openapi3filter.RequestValidationInput) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/firebase/exchange", strings.NewReader(body))
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		specReq := req.Clone(req.Context())
		specReq.URL.Path = strings.TrimPrefix(req.URL.Path, "/v1")
		route, pathParams, err := router2.FindRoute(specReq)
		if err != nil {
			t.Fatalf("exchange route not found in spec: %v", err)
		}
		requestInput := &openapi3filter.RequestValidationInput{
			Request: specReq, PathParams: pathParams, Route: route,
		}
		if validateRequest {
			if err := openapi3filter.ValidateRequest(context.Background(), requestInput); err != nil {
				t.Fatalf("valid exchange request violates spec: %v", err)
			}
		}
		req.Body = io.NopCloser(strings.NewReader(body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec, requestInput
	}

	rec, requestInput := exchange(`{"firebase_id_token":"verified-firebase-token","client":"guest_mobile"}`, "application/json", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("exchange status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := rec.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("Pragma = %q, want no-cache", got)
	}
	if service.exchangeCalls != 1 {
		t.Fatalf("exchange calls = %d, want 1", service.exchangeCalls)
	}
	responseInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestInput,
		Status:                 rec.Code,
		Header:                 rec.Header(),
		Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
	}
	if err := openapi3filter.ValidateResponse(context.Background(), responseInput); err != nil {
		t.Fatalf("exchange response violates spec: %v", err)
	}

	for _, body := range []string{
		`{}`,
		`{"firebase_id_token":"verified-firebase-token","client":"unknown_client"}`,
	} {
		rec, requestInput := exchange(body, "application/json", false)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("invalid body %s status = %d, want 422", body, rec.Code)
		}
		responseInput := &openapi3filter.ResponseValidationInput{
			RequestValidationInput: requestInput,
			Status:                 rec.Code,
			Header:                 rec.Header(),
			Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		}
		if err := openapi3filter.ValidateResponse(context.Background(), responseInput); err != nil {
			t.Fatalf("validation error response violates spec: %v", err)
		}
	}
	if service.exchangeCalls != 1 {
		t.Fatalf("invalid exchange reached service; calls = %d", service.exchangeCalls)
	}
}

func TestFirebaseExchangeErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantRetry  string
	}{
		{name: "generic auth", err: auth.ErrAuthenticationFailed, wantStatus: http.StatusUnauthorized, wantCode: "unauthorized"},
		{name: "link required", err: auth.ErrIdentityLinkRequired, wantStatus: http.StatusConflict, wantCode: "identity_link_required"},
		{name: "client forbidden", err: auth.ErrClientForbidden, wantStatus: http.StatusForbidden, wantCode: "forbidden"},
		{name: "firebase unavailable", err: auth.ErrFirebaseUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "auth_unavailable", wantRetry: "30"},
		{name: "limiter unavailable", err: auth.ErrRateLimiterUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "auth_unavailable", wantRetry: "30"},
		{name: "signing unavailable", err: auth.ErrSessionUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "auth_unavailable", wantRetry: "30"},
		{name: "rate limited", err: &auth.RateLimitedError{RetryAfter: 5 * time.Second}, wantStatus: http.StatusTooManyRequests, wantCode: "rate_limited", wantRetry: "5"},
		{name: "internal", err: errors.New("database details"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &exchangeContractService{err: test.err}
			router := NewRouter(RouterConfig{
				Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
				DB:     contractPinger{},
				PublicAPIRoutes: func(r chi.Router) {
					auth.RegisterPublicRoutes(r, service)
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/v1/auth/firebase/exchange",
				strings.NewReader(`{"firebase_id_token":"verified-firebase-token"}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, test.wantStatus, rec.Body.String())
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Error.Code != test.wantCode {
				t.Fatalf("error code = %q, want %q", body.Error.Code, test.wantCode)
			}
			if got := rec.Header().Get("Retry-After"); got != test.wantRetry {
				t.Fatalf("Retry-After = %q, want %q", got, test.wantRetry)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q, want no-store", got)
			}
			if strings.Contains(rec.Body.String(), "database details") {
				t.Fatal("internal error details leaked to the client")
			}
		})
	}
}

type fixedContractProvider struct {
	principal auth.Principal
	err       error
}

func (p fixedContractProvider) Authenticate(*http.Request) (auth.Principal, error) {
	if p.err != nil {
		return auth.Principal{}, p.err
	}
	return p.principal, nil
}

func TestLogoutContract(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatalf("load openapi: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("openapi is invalid: %v", err)
	}
	specRouter, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("spec router: %v", err)
	}
	service := &exchangeContractService{}
	publicRouter := NewRouter(RouterConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:     contractPinger{},
		PublicAPIRoutes: func(r chi.Router) {
			auth.RegisterPublicRoutes(r, service)
		},
	})
	logoutRequest := func(body string) (*httptest.ResponseRecorder, *openapi3filter.RequestValidationInput) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		specReq := req.Clone(req.Context())
		specReq.URL.Path = strings.TrimPrefix(req.URL.Path, "/v1")
		route, pathParams, err := specRouter.FindRoute(specReq)
		if err != nil {
			t.Fatalf("logout route not found in spec: %v", err)
		}
		req.Body = io.NopCloser(strings.NewReader(body))
		rec := httptest.NewRecorder()
		publicRouter.ServeHTTP(rec, req)
		return rec, &openapi3filter.RequestValidationInput{Request: specReq, PathParams: pathParams, Route: route}
	}
	validateResponse := func(rec *httptest.ResponseRecorder, requestInput *openapi3filter.RequestValidationInput) {
		t.Helper()
		if err := openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{
			RequestValidationInput: requestInput,
			Status:                 rec.Code,
			Header:                 rec.Header(),
			Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		}); err != nil {
			t.Fatalf("logout response violates spec: %v", err)
		}
	}

	rec, requestInput := logoutRequest(`{"refresh_token":"current-refresh-token"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("logout Cache-Control = %q", got)
	}
	if service.logoutCalls != 1 || service.logoutToken != "current-refresh-token" {
		t.Fatalf("logout call = %d token %q", service.logoutCalls, service.logoutToken)
	}
	validateResponse(rec, requestInput)

	rec, requestInput = logoutRequest(`{}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid logout status = %d, want 422", rec.Code)
	}
	validateResponse(rec, requestInput)

	accountID := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	protectedRouter := NewRouter(RouterConfig{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:       contractPinger{},
		Provider: fixedContractProvider{principal: auth.Principal{ActorType: auth.ActorAccount, ActorID: accountID.String(), AccountID: accountID}},
		APIRoutes: func(r chi.Router) {
			auth.RegisterProtectedRoutes(r, service)
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout-all", nil)
	specReq := req.Clone(req.Context())
	specReq.URL.Path = strings.TrimPrefix(req.URL.Path, "/v1")
	route, pathParams, err := specRouter.FindRoute(specReq)
	if err != nil {
		t.Fatalf("logout-all route not found in spec: %v", err)
	}
	rec = httptest.NewRecorder()
	protectedRouter.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout-all status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("logout-all Cache-Control = %q", got)
	}
	if service.logoutAllCalls != 1 || service.logoutAllAccount != accountID {
		t.Fatalf("logout-all call = %d account %s", service.logoutAllCalls, service.logoutAllAccount)
	}
	validateResponse(rec, &openapi3filter.RequestValidationInput{Request: specReq, PathParams: pathParams, Route: route})
}

func TestIdentityLinkContract(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatalf("load openapi: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("openapi is invalid: %v", err)
	}
	specRouter, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("spec router: %v", err)
	}
	accountID := uuid.MustParse("77777777-7777-4777-8777-777777777777")
	service := &exchangeContractService{}
	router := NewRouter(RouterConfig{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:       contractPinger{},
		Provider: fixedContractProvider{principal: auth.Principal{ActorType: auth.ActorAccount, ActorID: accountID.String(), AccountID: accountID}},
		APIRoutes: func(r chi.Router) {
			auth.RegisterProtectedRoutes(r, service)
		},
	})
	validateResponse := func(rec *httptest.ResponseRecorder, requestInput *openapi3filter.RequestValidationInput) {
		t.Helper()
		if err := openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{
			RequestValidationInput: requestInput,
			Status:                 rec.Code,
			Header:                 rec.Header(),
			Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		}); err != nil {
			t.Fatalf("identity response violates spec: %v", err)
		}
	}
	requestInputFor := func(req *http.Request) *openapi3filter.RequestValidationInput {
		t.Helper()
		specReq := req.Clone(req.Context())
		specReq.URL.Path = strings.TrimPrefix(req.URL.Path, "/v1")
		route, pathParams, err := specRouter.FindRoute(specReq)
		if err != nil {
			t.Fatalf("route %s not found in spec: %v", specReq.URL.Path, err)
		}
		return &openapi3filter.RequestValidationInput{Request: specReq, PathParams: pathParams, Route: route}
	}
	linkRequest := func(body string) (*httptest.ResponseRecorder, *openapi3filter.RequestValidationInput) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/identities/link", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec, requestInputFor(req)
	}

	rec, requestInput := linkRequest(`{"firebase_id_token":"fresh-firebase-token"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("link status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("link Cache-Control = %q", got)
	}
	if service.linkCalls != 1 || service.linkAccount != accountID || service.linkToken != "fresh-firebase-token" {
		t.Fatalf("link call = %d account %s token %q", service.linkCalls, service.linkAccount, service.linkToken)
	}
	validateResponse(rec, requestInput)

	rec, requestInput = linkRequest(`{}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid link status = %d, want 422", rec.Code)
	}
	validateResponse(rec, requestInput)

	service.linkErr = auth.ErrIdentityConflict
	rec, requestInput = linkRequest(`{"firebase_id_token":"duplicate"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate link status = %d, want 409", rec.Code)
	}
	validateResponse(rec, requestInput)

	service.linkErr = &auth.RateLimitedError{RetryAfter: 2 * time.Second}
	rec, requestInput = linkRequest(`{"firebase_id_token":"too-often"}`)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("rate-limited link status = %d retry = %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	validateResponse(rec, requestInput)

	identityID := uuid.MustParse("88888888-8888-4888-8888-888888888888")
	service.linkErr = nil
	unlinkRequest := func(path string) (*httptest.ResponseRecorder, *openapi3filter.RequestValidationInput) {
		t.Helper()
		req := httptest.NewRequest(http.MethodDelete, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec, requestInputFor(req)
	}
	rec, requestInput = unlinkRequest("/v1/auth/identities/" + identityID.String())
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unlink status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("unlink Cache-Control = %q", got)
	}
	if service.unlinkCalls != 1 || service.unlinkAccount != accountID || service.unlinkIdentityID != identityID {
		t.Fatalf("unlink call = %d account %s identity %s", service.unlinkCalls, service.unlinkAccount, service.unlinkIdentityID)
	}
	validateResponse(rec, requestInput)

	service.unlinkErr = auth.ErrIdentityNotFound
	rec, requestInput = unlinkRequest("/v1/auth/identities/" + identityID.String())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown identity unlink status = %d, want 404", rec.Code)
	}
	validateResponse(rec, requestInput)

	service.unlinkErr = auth.ErrLastIdentity
	rec, requestInput = unlinkRequest("/v1/auth/identities/" + identityID.String())
	if rec.Code != http.StatusConflict {
		t.Fatalf("last identity unlink status = %d, want 409", rec.Code)
	}
	validateResponse(rec, requestInput)

	rec, requestInput = unlinkRequest("/v1/auth/identities/not-a-uuid")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed identity unlink status = %d, want 400", rec.Code)
	}
	validateResponse(rec, requestInput)

	service.unlinkErr = &auth.RateLimitedError{RetryAfter: time.Second}
	rec, requestInput = unlinkRequest("/v1/auth/identities/" + identityID.String())
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("rate-limited unlink status = %d retry = %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	validateResponse(rec, requestInput)
}

func TestRefreshContract(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatalf("load openapi: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("openapi is invalid: %v", err)
	}
	specRouter, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("spec router: %v", err)
	}

	service := &exchangeContractService{}
	router := NewRouter(RouterConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:     contractPinger{},
		PublicAPIRoutes: func(r chi.Router) {
			auth.RegisterPublicRoutes(r, service)
		},
	})
	validateRefresh := func(body, contentType string, validateRequest bool) (*httptest.ResponseRecorder, *openapi3filter.RequestValidationInput) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", strings.NewReader(body))
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		specReq := req.Clone(req.Context())
		specReq.URL.Path = strings.TrimPrefix(req.URL.Path, "/v1")
		route, pathParams, err := specRouter.FindRoute(specReq)
		if err != nil {
			t.Fatalf("refresh route not found in spec: %v", err)
		}
		requestInput := &openapi3filter.RequestValidationInput{
			Request: specReq, PathParams: pathParams, Route: route,
		}
		if validateRequest {
			if err := openapi3filter.ValidateRequest(context.Background(), requestInput); err != nil {
				t.Fatalf("valid refresh request violates spec: %v", err)
			}
		}
		req.Body = io.NopCloser(strings.NewReader(body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec, requestInput
	}
	validateResponse := func(rec *httptest.ResponseRecorder, requestInput *openapi3filter.RequestValidationInput) {
		t.Helper()
		responseInput := &openapi3filter.ResponseValidationInput{
			RequestValidationInput: requestInput,
			Status:                 rec.Code,
			Header:                 rec.Header(),
			Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		}
		if err := openapi3filter.ValidateResponse(context.Background(), responseInput); err != nil {
			t.Fatalf("refresh response violates spec: %v", err)
		}
	}

	rec, requestInput := validateRefresh(`{"refresh_token":"current-refresh-token"}`, "application/json", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := rec.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("Pragma = %q, want no-cache", got)
	}
	if service.refreshCalls != 1 {
		t.Fatalf("refresh calls = %d, want 1", service.refreshCalls)
	}
	validateResponse(rec, requestInput)

	for _, body := range []string{`{}`, `{"refresh_token":""}`} {
		rec, requestInput := validateRefresh(body, "application/json", false)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("invalid body %s status = %d, want 422", body, rec.Code)
		}
		validateResponse(rec, requestInput)
	}
	if service.refreshCalls != 1 {
		t.Fatalf("invalid refresh reached service; calls = %d", service.refreshCalls)
	}

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantRetry  string
	}{
		{name: "generic auth", err: auth.ErrAuthenticationFailed, wantStatus: http.StatusUnauthorized, wantCode: "unauthorized"},
		{name: "rate limited", err: &auth.RateLimitedError{RetryAfter: 5 * time.Second}, wantStatus: http.StatusTooManyRequests, wantCode: "rate_limited", wantRetry: "5"},
		{name: "limiter unavailable", err: auth.ErrRateLimiterUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "auth_unavailable", wantRetry: "30"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &exchangeContractService{refreshErr: test.err}
			router := NewRouter(RouterConfig{
				Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
				DB:     contractPinger{},
				PublicAPIRoutes: func(r chi.Router) {
					auth.RegisterPublicRoutes(r, service)
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh",
				strings.NewReader(`{"refresh_token":"current-refresh-token"}`))
			req.Header.Set("Content-Type", "application/json")
			specReq := req.Clone(req.Context())
			specReq.URL.Path = strings.TrimPrefix(req.URL.Path, "/v1")
			route, pathParams, err := specRouter.FindRoute(specReq)
			if err != nil {
				t.Fatalf("refresh route not found in spec: %v", err)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, test.wantStatus, rec.Body.String())
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Error.Code != test.wantCode {
				t.Fatalf("error code = %q, want %q", body.Error.Code, test.wantCode)
			}
			if got := rec.Header().Get("Retry-After"); got != test.wantRetry {
				t.Fatalf("Retry-After = %q, want %q", got, test.wantRetry)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q, want no-store", got)
			}
			validateResponse(rec, &openapi3filter.RequestValidationInput{
				Request: specReq, PathParams: pathParams, Route: route,
			})
		})
	}
}

func TestJWKSContract(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatalf("load openapi: %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("openapi is invalid: %v", err)
	}
	specRouter, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("spec router: %v", err)
	}

	service := &exchangeContractService{jwksKeys: []auth.JWK{{
		Kty: "RSA",
		KID: "active-key",
		Use: "sig",
		Alg: "RS256",
		N:   "cHVibGljLW1vZHVsdXM",
		E:   "AQAB",
	}}}
	router := NewRouter(RouterConfig{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:     contractPinger{},
		PublicAPIRoutes: func(r chi.Router) {
			auth.RegisterPublicRoutes(r, service)
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/jwks", nil)
	specReq := req.Clone(req.Context())
	specReq.URL.Path = strings.TrimPrefix(req.URL.Path, "/v1")
	route, pathParams, err := specRouter.FindRoute(specReq)
	if err != nil {
		t.Fatalf("jwks route not found in spec: %v", err)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("jwks status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "public") {
		t.Fatalf("jwks Cache-Control = %q, want public cache", got)
	}
	if err := openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: specReq, PathParams: pathParams, Route: route},
		Status:                 rec.Code,
		Header:                 rec.Header(),
		Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
	}); err != nil {
		t.Fatalf("jwks response violates spec: %v", err)
	}

	service.jwksErr = auth.ErrSessionUnavailable
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable jwks status = %d, want 503", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got == "" {
		t.Fatal("unavailable jwks missing Retry-After")
	}
	if err := openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: specReq, PathParams: pathParams, Route: route},
		Status:                 rec.Code,
		Header:                 rec.Header(),
		Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
	}); err != nil {
		t.Fatalf("unavailable jwks response violates spec: %v", err)
	}
}
