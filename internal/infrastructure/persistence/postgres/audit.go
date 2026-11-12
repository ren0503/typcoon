package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ren0503/tycoon/internal/application/audit"
)

// AppendAuditTx validates and inserts an audit entry using the caller's transaction.
func AppendAuditTx(ctx context.Context, tx pgx.Tx, entry audit.Entry) error {
	if err := audit.Validate(entry); err != nil {
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
	return MapPgError(err)
}
