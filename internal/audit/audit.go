// Package audit is a temporary compatibility facade. It is removed once all
// call sites use application/audit and module transaction ports.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	appaudit "github.com/ren0503/tycoon/internal/application/audit"
	"github.com/ren0503/tycoon/internal/platform/errors"
)

type Entry = appaudit.Entry

type Filter = appaudit.Filter

type Record = appaudit.Record

// Execer is the subset of a transaction the appender needs.
type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// Appender appends audit entries inside the caller's transaction.
type Appender interface {
	AppendTx(ctx context.Context, tx Execer, entry Entry) error
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
	if err := appaudit.Validate(entry); err != nil {
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
	execer := tx
	if execer == nil {
		execer = s.pool
	}
	_, err := execer.Exec(ctx, `
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
