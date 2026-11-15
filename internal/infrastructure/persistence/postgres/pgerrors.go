package postgres

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apperrors "github.com/ren0503/tycoon/internal/platform/errors"
)

// constraintField maps well-known database constraint names to the request
// field they protect so clients receive a useful validation message.
func constraintField(name string) (string, bool) {
	switch name {
	case "uq_admin_users_email", "uq_accounts_email":
		return "email", true
	case "uq_decks_level":
		return "level", true
	case "uq_rooms_number":
		return "number", true
	case "uq_staff_code":
		return "employee_code", true
	case "uq_shift_templates_code":
		return "code", true
	case "uq_port_calls_seq":
		return "sequence", true
	case "uq_bookings_idem", "uq_service_bookings_idem", "uq_ship_structure_imports_idem":
		return "idempotency_key", true
	case "uq_autonomy_scope":
		return "change_type", true
	case "uq_hub_primary":
		return "role", true
	case "admin_ship_access_pkey":
		return "ship_id", true
	case "staff_credentials_staff_id_key":
		return "staff_id", true
	case "account_identities_provider_provider_sub_key":
		return "provider_sub", true
	case "ai_params_tenant_id_kind_param_version_key":
		return "param_version", true
	case "job_runs_voyage_id_job_type_schedule_version_key":
		return "schedule_version", true
	case "ships_imo_number_key":
		return "imo_number", true
	case "ports_un_locode_key":
		return "un_locode", true
	case "voyages_ship_id_code_key":
		return "code", true
	case "refresh_tokens_token_hash_key", "ship_devices_token_hash_key":
		return "token_hash", true
	case "hub_devices_cert_serial_key":
		return "cert_serial", true
	case "boarding_passes_jti_key":
		return "jti", true
	default:
		return "", false
	}
}

// MapPgError converts PostgreSQL driver errors into shared application errors.
// It never exposes SQL text or driver details to callers.
func MapPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NotFound(apperrors.CodeNotFound, "Resource not found.")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return apperrors.Internal(err)
	}

	field := pgErr.ConstraintName
	if hint, ok := constraintField(pgErr.ConstraintName); ok {
		field = hint
	}

	switch pgErr.Code {
	case pgerrcode.UniqueViolation:
		if field == "idempotency_key" {
			return apperrors.Wrap(err, apperrors.CodeIdempotencyConflict,
				"Idempotency key was already used with a different payload.", 409)
		}
		return apperrors.Wrap(err, apperrors.CodeConflict,
			"Resource conflicts with an existing record ("+field+").", 409)
	case pgerrcode.ExclusionViolation:
		return apperrors.Wrap(err, apperrors.CodeOverlapConflict,
			"Resource overlaps an existing record.", 409)
	case pgerrcode.ForeignKeyViolation:
		return apperrors.Wrap(err, apperrors.CodeInvalidReference,
			"Referenced resource does not exist or was deleted.", 422)
	case pgerrcode.CheckViolation:
		return apperrors.Wrap(err, apperrors.CodeValidationFailed,
			"Value violates a database constraint ("+field+").", 422)
	case pgerrcode.InvalidTextRepresentation:
		return apperrors.Wrap(err, apperrors.CodeValidationFailed,
			"Invalid value format.", 400)
	case pgerrcode.InvalidJSONText:
		return apperrors.Wrap(err, apperrors.CodeValidationFailed,
			"Invalid JSON value.", 422)
	case pgerrcode.SerializationFailure, pgerrcode.DeadlockDetected:
		return apperrors.Wrap(err, apperrors.CodeRetryable,
			"Transaction conflict; retry the request.", 503)
	default:
		if len(pgErr.Code) < 2 {
			return apperrors.Internal(err)
		}
		switch pgErr.Code[:2] {
		case "23":
			return apperrors.Wrap(err, apperrors.CodeConflict,
				"Resource conflicts with an existing record.", 409)
		case "22":
			return apperrors.Wrap(err, apperrors.CodeValidationFailed,
				"Invalid value.", 422)
		default:
			return apperrors.Internal(err)
		}
	}
}
