package staff

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

// Page is a cursor-paginated collection.
type Page[T any] struct {
	Items      []T
	NextCursor string
}

// StaffInput is the staff creation payload.
type StaffInput struct {
	ShipID         *uuid.UUID
	EmployeeCode   string
	FullName       string
	Department     string
	RoleTitle      *string
	IsSupervisor   bool
	Languages      []string
	Phone          *string
	Certifications []byte
	HiredOn        pgtype.Date
}

// StaffUpdate is the staff update payload.
type StaffUpdate struct {
	ShipID         *uuid.UUID
	EmployeeCode   *string
	FullName       *string
	Department     *string
	RoleTitle      *string
	IsSupervisor   *bool
	Languages      []string
	Phone          *string
	Certifications []byte
	Status         *string
	HiredOn        *pgtype.Date
}

// LeaveInput is the staff leave creation payload.
type LeaveInput struct {
	LeaveType string
	StartsAt  time.Time
	EndsAt    time.Time
	Note      *string
}

// LeaveUpdate is the staff leave update payload.
type LeaveUpdate struct {
	LeaveType *string
	StartsAt  *time.Time
	EndsAt    *time.Time
	Note      *string
}

// CredentialMeta never exposes the PIN hash.
type CredentialMeta struct {
	StaffID      uuid.UUID `json:"staff_id"`
	PinUpdatedAt time.Time `json:"pin_updated_at"`
}

// ShiftTemplateInput is the shift template creation payload.
type ShiftTemplateInput struct {
	Department string
	Code       string
	Name       string
	StartTime  pgtype.Time
	EndTime    pgtype.Time
}

// ShiftTemplateUpdate is the shift template update payload.
type ShiftTemplateUpdate struct {
	Department *string
	Code       *string
	Name       *string
	StartTime  *pgtype.Time
	EndTime    *pgtype.Time
}

// CoverageInput is the coverage requirement creation payload.
type CoverageInput struct {
	Department      string
	ShiftTemplateID uuid.UUID
	PositionCode    string
	DeckID          *uuid.UUID
	AreaID          *uuid.UUID
	MinStaff        int16
	MaxStaff        *int16
	DaysOfWeek      []int16
	ValidFrom       pgtype.Date
	ValidTo         pgtype.Date
}

// CoverageUpdate is the coverage requirement update payload.
type CoverageUpdate struct {
	Department      *string
	ShiftTemplateID *uuid.UUID
	PositionCode    *string
	DeckID          *uuid.UUID
	AreaID          *uuid.UUID
	MinStaff        *int16
	MaxStaff        *int16
	DaysOfWeek      []int16
	ValidFrom       *pgtype.Date
	ValidTo         *pgtype.Date
}

// LaborRuleInput is the labor rule creation payload.
type LaborRuleInput struct {
	ShipID    *uuid.UUID
	RuleCode  string
	Severity  string
	Params    []byte
	ValidFrom pgtype.Date
	ValidTo   pgtype.Date
}

// LaborRuleUpdate is the labor rule update payload.
type LaborRuleUpdate struct {
	ShipID    *uuid.UUID
	RuleCode  *string
	Severity  *string
	Params    []byte
	ValidFrom *pgtype.Date
	ValidTo   *pgtype.Date
}

// AutonomyInput is the autonomy policy creation payload.
type AutonomyInput struct {
	ShipID     *uuid.UUID
	ChangeType string
	Level      string
	Conditions []byte
}

// AutonomyUpdate is the autonomy policy update payload.
type AutonomyUpdate struct {
	ShipID     *uuid.UUID
	ChangeType *string
	Level      *string
	Conditions []byte
}

// Service is the public surface of the staff module.
type Service interface {
	CreateStaff(ctx context.Context, principal auth.Principal, input StaffInput) (sqlc.Staff, error)
	GetStaff(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.Staff, error)
	ListStaff(ctx context.Context, principal auth.Principal, shipID *uuid.UUID, cursor string, limit int32) (Page[sqlc.Staff], error)
	UpdateStaff(ctx context.Context, principal auth.Principal, id uuid.UUID, input StaffUpdate) (sqlc.Staff, error)
	DeleteStaff(ctx context.Context, principal auth.Principal, id uuid.UUID) error
	StaffExists(ctx context.Context, id uuid.UUID) (bool, error)

	CreateLeave(ctx context.Context, principal auth.Principal, staffID uuid.UUID, input LeaveInput) (sqlc.StaffLeafe, error)
	GetLeave(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.StaffLeafe, error)
	ListLeaves(ctx context.Context, principal auth.Principal, staffID uuid.UUID, cursor string, limit int32) (Page[sqlc.StaffLeafe], error)
	UpdateLeave(ctx context.Context, principal auth.Principal, id uuid.UUID, input LeaveUpdate) (sqlc.StaffLeafe, error)
	DeleteLeave(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	SetCredential(ctx context.Context, principal auth.Principal, staffID uuid.UUID, pin string) (CredentialMeta, error)
	CredentialMeta(ctx context.Context, principal auth.Principal, staffID uuid.UUID) (CredentialMeta, error)

	CreateShiftTemplate(ctx context.Context, principal auth.Principal, shipID uuid.UUID, input ShiftTemplateInput) (sqlc.ShiftTemplate, error)
	GetShiftTemplate(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.ShiftTemplate, error)
	ListShiftTemplates(ctx context.Context, principal auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[sqlc.ShiftTemplate], error)
	UpdateShiftTemplate(ctx context.Context, principal auth.Principal, id uuid.UUID, input ShiftTemplateUpdate) (sqlc.ShiftTemplate, error)
	DeleteShiftTemplate(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateCoverage(ctx context.Context, principal auth.Principal, shipID uuid.UUID, input CoverageInput) (sqlc.CoverageRequirement, error)
	GetCoverage(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.CoverageRequirement, error)
	ListCoverage(ctx context.Context, principal auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[sqlc.CoverageRequirement], error)
	UpdateCoverage(ctx context.Context, principal auth.Principal, id uuid.UUID, input CoverageUpdate) (sqlc.CoverageRequirement, error)
	DeleteCoverage(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateLaborRule(ctx context.Context, principal auth.Principal, input LaborRuleInput) (sqlc.LaborRule, error)
	GetLaborRule(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.LaborRule, error)
	ListLaborRules(ctx context.Context, principal auth.Principal, shipID *uuid.UUID, cursor string, limit int32) (Page[sqlc.LaborRule], error)
	UpdateLaborRule(ctx context.Context, principal auth.Principal, id uuid.UUID, input LaborRuleUpdate) (sqlc.LaborRule, error)
	DeleteLaborRule(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateAutonomy(ctx context.Context, principal auth.Principal, input AutonomyInput) (sqlc.AutonomyPolicy, error)
	GetAutonomy(ctx context.Context, principal auth.Principal, id uuid.UUID) (sqlc.AutonomyPolicy, error)
	ListAutonomy(ctx context.Context, principal auth.Principal, shipID *uuid.UUID, cursor string, limit int32) (Page[sqlc.AutonomyPolicy], error)
	UpdateAutonomy(ctx context.Context, principal auth.Principal, id uuid.UUID, input AutonomyUpdate) (sqlc.AutonomyPolicy, error)
	DeleteAutonomy(ctx context.Context, principal auth.Principal, id uuid.UUID) error
}
