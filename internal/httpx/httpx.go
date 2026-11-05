// Package httpx contains HTTP helpers shared by the business modules' handlers.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/pagination"
)

const maxBodyBytes = 1 << 20

var validate = validator.New(validator.WithRequiredStructEnabled())

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type pageEnvelope[T any] struct {
	Data []T  `json:"data"`
	Meta meta `json:"meta"`
}

type meta struct {
	NextCursor *string `json:"next_cursor"`
}

// WriteJSON writes a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// WriteError maps an error to the shared envelope. Unknown errors become 500s and
// their details are never exposed to the client.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := apperrors.As(err)
	if !ok {
		appErr = apperrors.Internal(err)
	}
	WriteJSON(w, appErr.HTTPStatus, errorEnvelope{Error: errorBody{
		Code:      appErr.Code,
		Message:   appErr.Message,
		RequestID: middleware.GetReqID(r.Context()),
	}})
}

// Decode parses and validates a JSON request body.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return apperrors.BadRequest(apperrors.CodeValidationFailed, "Request body is not valid JSON.")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apperrors.BadRequest(apperrors.CodeValidationFailed, "Request body must contain a single JSON object.")
	}
	if err := validate.Struct(dst); err != nil {
		return apperrors.Validation(apperrors.CodeValidationFailed, validationMessage(err))
	}
	return nil
}

func validationMessage(err error) string {
	var fieldErrs validator.ValidationErrors
	if errors.As(err, &fieldErrs) && len(fieldErrs) > 0 {
		return "Validation failed for field " + fieldErrs[0].Field() + "."
	}
	return "Request validation failed."
}

// Page writes a cursor-paginated collection.
func Page[T any](w http.ResponseWriter, items []T, nextCursor string) {
	env := pageEnvelope[T]{Data: items}
	if env.Data == nil {
		env.Data = []T{}
	}
	if nextCursor != "" {
		env.Meta.NextCursor = &nextCursor
	}
	WriteJSON(w, http.StatusOK, env)
}

// PathID parses a UUID path parameter.
func PathID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		return uuid.Nil, apperrors.BadRequest(apperrors.CodeValidationFailed, "Path parameter "+name+" must be a UUID.")
	}
	return id, nil
}

// ParseUUID parses a UUID value that is known to be present.
func ParseUUID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.BadRequest(apperrors.CodeValidationFailed, "Identifier must be a UUID.")
	}
	return id, nil
}

// QueryUUID parses an optional UUID query parameter.
func QueryUUID(r *http.Request, name string) (*uuid.UUID, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, apperrors.BadRequest(apperrors.CodeValidationFailed, "Query parameter "+name+" must be a UUID.")
	}
	return &id, nil
}

// ListParams holds the decoded pagination state for a list request.
type ListParams struct {
	Cursor pagination.Cursor
	Limit  int32
}

// ParseList decodes limit and cursor for the given filter set.
func ParseList(r *http.Request, filterParts ...string) (ListParams, error) {
	requested := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return ListParams{}, apperrors.BadRequest(apperrors.CodeValidationFailed, "Query parameter limit must be a non-negative integer.")
		}
		requested = n
	}
	cursor, err := pagination.Decode(r.URL.Query().Get("cursor"), pagination.FilterHash(filterParts...))
	if err != nil {
		return ListParams{}, err
	}
	return ListParams{Cursor: cursor, Limit: int32(pagination.Limit(requested))}, nil
}

// NextID derives the next cursor for an identifier-ordered list. It returns an
// empty string when there are no more rows.
func NextID(rows, pageLen int, filterParts []string, last uuid.UUID) string {
	if rows <= pageLen {
		return ""
	}
	return pagination.Cursor{Filter: pagination.FilterHash(filterParts...), ID: last}.Encode()
}

// NextSorted derives the next cursor for a list ordered by a sort key and id.
func NextSorted(rows, pageLen int, filterParts []string, sortKey string, last uuid.UUID) string {
	if rows <= pageLen {
		return ""
	}
	return pagination.Cursor{Filter: pagination.FilterHash(filterParts...), Sort: sortKey, ID: last}.Encode()
}

// NoContent writes an empty 204 response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// ParseLimit clamps the requested page size for services that decode their own cursor.
func ParseLimit(r *http.Request) (int32, error) {
	requested := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return 0, apperrors.BadRequest(apperrors.CodeValidationFailed, "Query parameter limit must be a non-negative integer.")
		}
		requested = n
	}
	return int32(pagination.Limit(requested)), nil
}

// RawCursor returns the raw cursor query parameter.
func RawCursor(r *http.Request) string {
	return r.URL.Query().Get("cursor")
}

// Header returns the Idempotency-Key header.
func Header(r *http.Request) string {
	return r.Header.Get("Idempotency-Key")
}

// SetNoStore prevents token-bearing auth responses from being cached.
func SetNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}
