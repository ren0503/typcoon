package apperrors

import (
	"net/http"
	"testing"
)

func TestRateLimited(t *testing.T) {
	err := RateLimited("Too many requests.")

	if err.Code != CodeRateLimited {
		t.Fatalf("Code = %q, want %q", err.Code, CodeRateLimited)
	}
	if err.HTTPStatus != http.StatusTooManyRequests {
		t.Fatalf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusTooManyRequests)
	}
}

func TestAuthUnavailable(t *testing.T) {
	err := AuthUnavailable("Authentication is temporarily unavailable.")

	if err.Code != CodeAuthUnavailable {
		t.Fatalf("Code = %q, want %q", err.Code, CodeAuthUnavailable)
	}
	if err.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus = %d, want %d", err.HTTPStatus, http.StatusServiceUnavailable)
	}
}
