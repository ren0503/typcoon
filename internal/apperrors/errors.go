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
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
