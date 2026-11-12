//go:build integration

package staff_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ren0503/tycoon/internal/platform/testutil"
	"github.com/ren0503/tycoon/internal/ship"
	"github.com/ren0503/tycoon/internal/staff"
)

func TestStaffModuleValidationAndCredentials(t *testing.T) {
	h := testutil.StartPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := h.SeedTenant(t)
	principal := testutil.Principal(tenantID)
	shipSvc := ship.New(h.Pool, testutil.Logger(t), nil)
	shipRow, err := shipSvc.CreateShip(ctx, principal, ship.ShipInput{Name: "Staff Ship", Timezone: "UTC"})
	if err != nil {
		t.Fatalf("create ship: %v", err)
	}
	svc := staff.New(h.Pool, testutil.Logger(t), shipSvc, nil)

	member, err := svc.CreateStaff(ctx, principal, staff.StaffInput{
		ShipID: &shipRow.ID, EmployeeCode: "EMP-1", FullName: "Ada Crew", Department: "security",
		Languages: []string{"en"}, Certifications: []byte(`[{"code":"STCW"}]`),
	})
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}

	if _, err := svc.CreateLeave(ctx, principal, member.ID, staff.LeaveInput{
		LeaveType: "annual", StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now(),
	}); err == nil {
		t.Fatal("expected leave time-order validation")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}
	if _, err := svc.CreateStaff(ctx, principal, staff.StaffInput{
		EmployeeCode: "EMP-2", FullName: "Bad Cert", Department: "cleaning",
		Certifications: []byte(`{"not":"an array"}`),
	}); err == nil {
		t.Fatal("expected non-array certifications validation")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}

	template, err := svc.CreateShiftTemplate(ctx, principal, shipRow.ID, staff.ShiftTemplateInput{
		Department: "security", Code: "NIGHT", Name: "Night",
		StartTime: pgtype.Time{Microseconds: 22 * 3600 * 1e6, Valid: true},
		EndTime:   pgtype.Time{Microseconds: 6 * 3600 * 1e6, Valid: true},
	})
	if err != nil {
		t.Fatalf("create shift template: %v", err)
	}
	if _, err := svc.CreateCoverage(ctx, principal, shipRow.ID, staff.CoverageInput{
		Department: "security", ShiftTemplateID: template.ID, PositionCode: "GATE",
		MinStaff: 1, DaysOfWeek: []int16{9},
	}); err == nil {
		t.Fatal("expected days_of_week validation")
	} else {
		testutil.RequireErrorCode(t, err, "validation_failed")
	}

	meta, err := svc.SetCredential(ctx, principal, member.ID, "4821")
	if err != nil {
		t.Fatalf("set credential: %v", err)
	}
	if meta.StaffID != member.ID || meta.PinUpdatedAt.IsZero() {
		t.Fatalf("unexpected credential metadata: %+v", meta)
	}
	if _, err := svc.CredentialMeta(ctx, principal, member.ID); err != nil {
		t.Fatalf("read credential metadata: %v", err)
	}
	var storedHash string
	if err := h.Pool.QueryRow(ctx, `SELECT pin_hash FROM staff_credentials WHERE staff_id = $1`, member.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read stored pin hash: %v", err)
	}
	if storedHash == "" || storedHash == "4821" {
		t.Fatalf("PIN was not hashed before persistence")
	}

	autonomy, err := svc.CreateAutonomy(ctx, principal, staff.AutonomyInput{
		ChangeType: "shift_swap", Level: "approval", Conditions: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("create tenant-wide autonomy policy: %v", err)
	}
	if autonomy.ShipID.Valid {
		t.Fatalf("tenant-wide autonomy policy must not use a placeholder ship id")
	}
}
