package database

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ren0503/tycoon/internal/apperrors"
)

// constraintFields maps well-known database constraint names to the request
// field they protect so clients receive a useful validation message.
var constraintFields = map[string]string{
	"uq_admin_users_email":                             "email",
	"uq_accounts_email":                                "email",
	"uq_decks_level":                                   "level",
	"uq_rooms_number":                                  "number",
	"uq_staff_code":                                    "employee_code",
	"uq_shift_templates_code":                          "code",
	"uq_port_calls_seq":                                "sequence",
	"uq_bookings_idem":                                 "idempotency_key",
	"uq_service_bookings_idem":                         "idempotency_key",
	"uq_autonomy_scope":                                "change_type",
	"uq_ship_structure_imports_idem":                   "idempotency_key",
	"uq_hub_primary":                                   "role",
	"admin_ship_access_pkey":                           "ship_id",
	"staff_credentials_staff_id_key":                   "staff_id",
	"account_identities_provider_provider_sub_key":     "provider_sub",
	"ai_params_tenant_id_kind_param_version_key":       "param_version",
	"job_runs_voyage_id_job_type_schedule_version_key": "schedule_version",
	"ships_imo_number_key":                             "imo_number",
	"ports_un_locode_key":                              "un_locode",
	"voyages_ship_id_code_key":                         "code",
	"refresh_tokens_token_hash_key":                    "token_hash",
	"ship_devices_token_hash_key":                      "token_hash",
	"hub_devices_cert_serial_key":                      "cert_serial",
	"boarding_passes_jti_key":                          "jti",
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
	if hint, ok := constraintFields[pgErr.ConstraintName]; ok {
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
