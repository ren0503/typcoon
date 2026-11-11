//go:build integration

package testutil

import (
	"log/slog"
	"testing"
)

// Logger returns a discard logger suitable for services under test.
func Logger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.DiscardHandler)
}
