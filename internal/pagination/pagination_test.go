package pagination

import (
	"testing"

	"github.com/google/uuid"
)

func TestLimitClamping(t *testing.T) {
	cases := map[int]int{0: DefaultLimit, -5: DefaultLimit, 1: 1, 20: 20, 100: 100, 101: MaxLimit, 5000: MaxLimit}
	for requested, want := range cases {
		if got := Limit(requested); got != want {
			t.Fatalf("Limit(%d) = %d, want %d", requested, got, want)
		}
	}
}

func TestCursorRoundTrip(t *testing.T) {
	filter := FilterHash("voyage_id=abc")
	original := Cursor{Filter: filter, Sort: "2026-01-01T00:00:00Z", ID: uuid.MustParse("018f0000-0000-7000-8000-000000000001")}

	decoded, err := Decode(original.Encode(), filter)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.ID != original.ID || decoded.Sort != original.Sort {
		t.Fatalf("round trip mismatch: %+v", decoded)
	}
}

func TestCursorRejectsForeignFilter(t *testing.T) {
	encoded := Cursor{Filter: FilterHash("ship_id=a"), ID: uuid.New()}.Encode()
	if _, err := Decode(encoded, FilterHash("ship_id=b")); err == nil {
		t.Fatal("expected error for cursor issued under a different filter")
	}
}

func TestCursorRejectsGarbage(t *testing.T) {
	for _, raw := range []string{"not-base64!!", "e30", "eyJ2Ijo5OSwiaWQiOiIifQ"} {
		if _, err := Decode(raw, ""); err == nil {
			t.Fatalf("expected error for cursor %q", raw)
		}
	}
}
