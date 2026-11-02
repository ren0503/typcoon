package pagination

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/apperrors"
)

const (
	// DefaultLimit is the page size used when the client does not ask for one.
	DefaultLimit = 20
	// MaxLimit is the largest page a client may request; larger values are clamped.
	MaxLimit = 100
	// maxCursorLength bounds the opaque cursor input.
	maxCursorLength = 256

	cursorVersion = 1
)

// Cursor is the opaque keyset pagination state returned to clients.
type Cursor struct {
	Version int       `json:"v"`
	Filter  string    `json:"f"`
	Sort    string    `json:"s,omitempty"`
	ID      uuid.UUID `json:"id"`
}

// FilterHash derives a short, stable signature for the canonical filter set a
// cursor belongs to. Lists reject cursors that were issued for another filter.
func FilterHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:8])
}

// Limit clamps the requested page size to the documented default and maximum.
func Limit(requested int) int {
	if requested <= 0 {
		return DefaultLimit
	}
	if requested > MaxLimit {
		return MaxLimit
	}
	return requested
}

// Encode serializes the cursor using URL-safe base64 without padding.
func (c Cursor) Encode() string {
	c.Version = cursorVersion
	payload, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

// Decode parses an opaque cursor and verifies it belongs to the expected filter.
// An empty raw value means "first page" and returns a zero Cursor.
func Decode(raw, expectedFilter string) (Cursor, error) {
	if strings.TrimSpace(raw) == "" {
		return Cursor{}, nil
	}
	if len(raw) > maxCursorLength {
		return Cursor{}, invalidCursor()
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return Cursor{}, invalidCursor()
	}
	var cursor Cursor
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cursor); err != nil {
		return Cursor{}, invalidCursor()
	}
	if cursor.Version != cursorVersion || cursor.ID == uuid.Nil {
		return Cursor{}, invalidCursor()
	}
	if cursor.Filter != expectedFilter {
		return Cursor{}, invalidCursor()
	}
	return cursor, nil
}

// AfterID returns the keyset anchor identifier for an identifier-ordered list.
func AfterID(cursor Cursor) uuid.UUID {
	return cursor.ID
}

func invalidCursor() error {
	return apperrors.BadRequest(apperrors.CodeInvalidCursor, "Cursor is malformed or belongs to another filter.")
}

// Page is a cursor-paginated result set.
type Page[T any] struct {
	Items      []T
	NextCursor string
}
