// Package apperrors defines the shared public error envelope and error codes.
package apperrors

import (
	"errors"
	"net/http"
)

// Error is a shared application error with a safe public message and an internal cause.
type Error struct {
	Code       string
	Message    string
	HTTPStatus int
	cause      error
}

// New creates a new application error.
func New(code, message string, status int) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: status}
}

// Wrap creates an application error that retains an internal cause.
func Wrap(cause error, code, message string, status int) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: status, cause: cause}
}

// Error implements the error interface using the safe public message.
func (e *Error) Error() string {
	return e.Message
}

// Unwrap returns the internal cause.
func (e *Error) Unwrap() error {
	return e.cause
}

// Cause returns the internal cause.
func (e *Error) Cause() error {
	return e.cause
}

// Internal creates a generic internal error whose cause is never exposed to clients.
func Internal(cause error) *Error {
	return Wrap(cause, "internal_error", "Internal server error.", http.StatusInternalServerError)
}

// As extracts an application error from an error chain.
func As(err error) (*Error, bool) {
	if appErr, ok := errors.AsType[*Error](err); ok {
		return appErr, true
	}
	return nil, false
}

// Error codes shared across modules and the HTTP layer.
const (
	CodeUnauthorized        = "unauthorized"
	CodeForbidden           = "forbidden"
	CodeNotFound            = "not_found"
	CodeValidationFailed    = "validation_failed"
	CodeInvalidReference    = "invalid_reference"
	CodeInvalidCursor       = "invalid_cursor"
	CodeConflict            = "conflict"
	CodeOverlapConflict     = "overlap_conflict"
	CodeIdempotencyConflict = "idempotency_conflict"
	CodeBusinessRule        = "business_rule_violation"
	CodeRetryable           = "retryable"
	CodeRateLimited         = "rate_limited"
	CodeAuthUnavailable     = "auth_unavailable"
)

// NotFound creates a not-found application error.
func NotFound(code, message string) *Error {
	return New(code, message, http.StatusNotFound)
}

// Conflict creates a conflict application error.
func Conflict(code, message string) *Error {
	return New(code, message, http.StatusConflict)
}

// Validation creates a validation application error.
func Validation(code, message string) *Error {
	return New(code, message, http.StatusUnprocessableEntity)
}

// BadRequest creates a malformed-request application error.
func BadRequest(code, message string) *Error {
	return New(code, message, http.StatusBadRequest)
}

// Unauthorized creates an authentication-required application error.
func Unauthorized(message string) *Error {
	return New(CodeUnauthorized, message, http.StatusUnauthorized)
}

// Forbidden creates an authorization application error.
func Forbidden(message string) *Error {
	return New(CodeForbidden, message, http.StatusForbidden)
}

// RateLimited creates an authentication rate-limit application error.
func RateLimited(message string) *Error {
	return New(CodeRateLimited, message, http.StatusTooManyRequests)
}

// AuthUnavailable creates a fail-closed authentication dependency error.
func AuthUnavailable(message string) *Error {
	return New(CodeAuthUnavailable, message, http.StatusServiceUnavailable)
}
