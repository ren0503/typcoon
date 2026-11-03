//go:build integration

package sqlc_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ren0503/tycoon/internal/database/sqlc"
	"github.com/ren0503/tycoon/internal/testutil"
)

type modelSpec struct {
	table string
	model any
}

var models = []modelSpec{
	{"tenants", &sqlc.Tenant{}},
	{"admin_users", &sqlc.AdminUser{}},
	{"admin_ship_access", &sqlc.AdminShipAccess{}},
	{"accounts", &sqlc.Account{}},
	{"account_identities", &sqlc.AccountIdentity{}},
	{"refresh_tokens", &sqlc.RefreshToken{}},
	{"ships", &sqlc.Ship{}},
	{"decks", &sqlc.Deck{}},
	{"areas", &sqlc.Area{}},
	{"rooms", &sqlc.Room{}},
	{"ports", &sqlc.Port{}},
	{"voyages", &sqlc.Voyage{}},
	{"port_calls", &sqlc.PortCall{}},
	{"staff", &sqlc.Staff{}},
	{"staff_leaves", &sqlc.StaffLeafe{}},
	{"staff_credentials", &sqlc.StaffCredential{}},
	{"shift_templates", &sqlc.ShiftTemplate{}},
	{"coverage_requirements", &sqlc.CoverageRequirement{}},
	{"labor_rules", &sqlc.LaborRule{}},
	{"autonomy_policies", &sqlc.AutonomyPolicy{}},
	{"ai_params", &sqlc.AiParam{}},
	{"partners", &sqlc.Partner{}},
	{"partner_contracts", &sqlc.PartnerContract{}},
	{"partner_staff", &sqlc.PartnerStaff{}},
	{"partner_services", &sqlc.PartnerService{}},
	{"partner_slots", &sqlc.PartnerSlot{}},
	{"bookings", &sqlc.Booking{}},
	{"guests", &sqlc.Guest{}},
	{"boarding_passes", &sqlc.BoardingPass{}},
	{"service_bookings", &sqlc.ServiceBooking{}},
	{"ship_seq", &sqlc.ShipSeq{}},
	{"change_log", &sqlc.ChangeLog{}},
	{"hub_devices", &sqlc.HubDevice{}},
	{"ship_devices", &sqlc.ShipDevice{}},
	{"revocations", &sqlc.Revocation{}},
	{"snapshots", &sqlc.Snapshot{}},
	{"job_runs", &sqlc.JobRun{}},
	{"ops_events", &sqlc.OpsEvent{}},
	{"audit_log", &sqlc.AuditLog{}},
	{"feedback_responses", &sqlc.FeedbackResponse{}},
	{"feedback_insights", &sqlc.FeedbackInsight{}},
}

type column struct {
	name     string
	nullable bool
}

func TestGeneratedModelsMatchSchema(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if len(models) != 41 {
		t.Fatalf("model map covers %d tables, want 41", len(models))
	}

	for _, spec := range models {
		columns := tableColumns(t, ctx, h, spec.table)
		if len(columns) == 0 {
			t.Fatalf("table %s has no columns or does not exist", spec.table)
		}
		fields := modelFields(t, spec.model)

		if len(fields) != len(columns) {
			t.Fatalf("%s: model has %d fields, schema has %d columns", spec.table, len(fields), len(columns))
		}
		for name, column := range columns {
			fieldType, ok := fields[name]
			if !ok {
				t.Fatalf("%s: model field for column %s is missing", spec.table, name)
			}
			if column.nullable && !isNullableGoType(fieldType) {
				t.Fatalf("%s.%s is nullable but model type %s is not nullable", spec.table, name, fieldType)
			}
		}
	}
}

func tableColumns(t *testing.T, ctx context.Context, h *testutil.Harness, table string) map[string]column {
	t.Helper()
	rows, err := h.Pool.Query(ctx, `
		SELECT column_name, is_nullable = 'YES'
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1`, table)
	if err != nil {
		t.Fatalf("read columns for %s: %v", table, err)
	}
	defer rows.Close()
	out := map[string]column{}
	for rows.Next() {
		var name string
		var nullable bool
		if err := rows.Scan(&name, &nullable); err != nil {
			t.Fatalf("scan column for %s: %v", table, err)
		}
		out[name] = column{name: name, nullable: nullable}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("columns for %s: %v", table, err)
	}
	return out
}

func modelFields(t *testing.T, model any) map[string]reflect.Type {
	t.Helper()
	value := reflect.TypeOf(model).Elem()
	out := map[string]reflect.Type{}
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		tag := field.Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			continue
		}
		out[name] = field.Type
	}
	return out
}

func isNullableGoType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		return true
	}
	name := t.Name()
	if name == "NullUUID" || name == "NullDecimal" {
		return true
	}
	return strings.HasPrefix(t.PkgPath(), "github.com/jackc/pgx/v5/pgtype")
}
