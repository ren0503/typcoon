package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/apperrors"
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

// Execer is the subset of a transaction the appender needs.
type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// Appender appends audit entries inside the caller's transaction.
type Appender interface {
	AppendTx(ctx context.Context, tx Execer, entry Entry) error
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

// Store persists audit records in the audit_log table.
type Store struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewStore creates an audit store backed by the given pool.
func NewStore(pool *pgxpool.Pool, logger *slog.Logger) *Store {
	return &Store{pool: pool, logger: logger.With("component", "audit_store")}
}

// AppendTx validates and inserts an audit entry using the caller's transaction.
func (s *Store) AppendTx(ctx context.Context, tx Execer, entry Entry) error {
	if err := entry.validate(); err != nil {
		s.logger.ErrorContext(ctx, "error validating audit entry", "error", err, "entry", entry)
		return err
	}
	occurredAt := entry.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	details := entry.Details
	if len(details) == 0 {
		details = json.RawMessage(`{}`)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_log
			(occurred_at, source, actor_type, actor_id, action, entity, entity_id, ship_id, voyage_id, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		occurredAt, entry.Source, entry.ActorType, entry.ActorID, entry.Action,
		entry.Entity, entry.EntityID, entry.ShipID, entry.VoyageID, details,
	)
	if err != nil {
		s.logger.ErrorContext(ctx, "error inserting audit entry", "error", err, "entry", entry)
		return apperrors.Internal(err)
	}
	return nil
}

var _ Appender = (*Store)(nil)

// List returns audit records matching the filter, newest first.
func (s *Store) List(ctx context.Context, filter Filter) ([]Record, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, occurred_at, source, actor_type, actor_id, action, entity, entity_id, ship_id, voyage_id, details
		FROM audit_log
		WHERE ($1::uuid IS NULL OR ship_id = $1)
		  AND ($2 = '' OR entity = $2)
		  AND ($3 = '' OR entity_id = $3)
		ORDER BY id DESC
		LIMIT 500`,
		filter.ShipID, filter.Entity, filter.EntityID,
	)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var record Record
		if err := rows.Scan(&record.ID, &record.OccurredAt, &record.Source, &record.ActorType,
			&record.ActorID, &record.Action, &record.Entity, &record.EntityID,
			&record.ShipID, &record.VoyageID, &record.Details); err != nil {
			return nil, apperrors.Internal(err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal(err)
	}
	return records, nil
}

func (e Entry) validate() error {
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
