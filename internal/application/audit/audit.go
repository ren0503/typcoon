// Package audit defines audit value types and validation policy.
package audit

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/platform/errors"
)

// Entry is one append-only audit record.
type Entry struct {
	OccurredAt time.Time
	Source     string
	ActorType  string
	ActorID    *string
	Action     string
	Entity     *string
	EntityID   *string
	ShipID     *uuid.UUID
	VoyageID   *uuid.UUID
	Details    json.RawMessage
}

// Filter selects audit records for querying.
type Filter struct {
	ShipID   *uuid.UUID
	Entity   string
	EntityID string
}

// Record is a stored audit row.
type Record struct {
	ID         int64           `json:"id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Source     string          `json:"source"`
	ActorType  string          `json:"actor_type"`
	ActorID    *string         `json:"actor_id,omitempty"`
	Action     string          `json:"action"`
	Entity     *string         `json:"entity,omitempty"`
	EntityID   *string         `json:"entity_id,omitempty"`
	ShipID     *uuid.UUID      `json:"ship_id,omitempty"`
	VoyageID   *uuid.UUID      `json:"voyage_id,omitempty"`
	Details    json.RawMessage `json:"details,omitempty"`
}

// Validate checks the audit entry policy.
func Validate(e Entry) error {
	switch e.Source {
	case "cloud", "hub":
	default:
		return apperrors.Validation(apperrors.CodeValidationFailed, "Audit source must be cloud or hub.")
	}
	switch e.ActorType {
	case "admin", "account", "staff", "hub", "system", "ai":
	default:
		return apperrors.Validation(apperrors.CodeValidationFailed, "Audit actor type is invalid.")
	}
	if e.Action == "" {
		return apperrors.Validation(apperrors.CodeValidationFailed, "Audit action is required.")
	}
	return nil
}
