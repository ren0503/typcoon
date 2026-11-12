package staff

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
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
	HiredOn        *time.Time
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
	HiredOn        *time.Time
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
	StartTime  TimeOfDay
	EndTime    TimeOfDay
}

// ShiftTemplateUpdate is the shift template update payload.
type ShiftTemplateUpdate struct {
	Department *string
	Code       *string
	Name       *string
	StartTime  *TimeOfDay
	EndTime    *TimeOfDay
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
	ValidFrom       *time.Time
	ValidTo         *time.Time
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
	ValidFrom       *time.Time
	ValidTo         *time.Time
}

// LaborRuleInput is the labor rule creation payload.
type LaborRuleInput struct {
	ShipID    *uuid.UUID
	RuleCode  string
	Severity  string
	Params    []byte
	ValidFrom *time.Time
	ValidTo   *time.Time
}

// LaborRuleUpdate is the labor rule update payload.
type LaborRuleUpdate struct {
	ShipID    *uuid.UUID
	RuleCode  *string
	Severity  *string
	Params    []byte
	ValidFrom *time.Time
	ValidTo   *time.Time
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
	CreateStaff(ctx context.Context, principal auth.Principal, input StaffInput) (Staff, error)
	GetStaff(ctx context.Context, principal auth.Principal, id uuid.UUID) (Staff, error)
	ListStaff(ctx context.Context, principal auth.Principal, shipID *uuid.UUID, cursor string, limit int32) (Page[Staff], error)
	UpdateStaff(ctx context.Context, principal auth.Principal, id uuid.UUID, input StaffUpdate) (Staff, error)
	DeleteStaff(ctx context.Context, principal auth.Principal, id uuid.UUID) error
	StaffExists(ctx context.Context, id uuid.UUID) (bool, error)

	CreateLeave(ctx context.Context, principal auth.Principal, staffID uuid.UUID, input LeaveInput) (StaffLeafe, error)
	GetLeave(ctx context.Context, principal auth.Principal, id uuid.UUID) (StaffLeafe, error)
	ListLeaves(ctx context.Context, principal auth.Principal, staffID uuid.UUID, cursor string, limit int32) (Page[StaffLeafe], error)
	UpdateLeave(ctx context.Context, principal auth.Principal, id uuid.UUID, input LeaveUpdate) (StaffLeafe, error)
	DeleteLeave(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	SetCredential(ctx context.Context, principal auth.Principal, staffID uuid.UUID, pin string) (CredentialMeta, error)
	CredentialMeta(ctx context.Context, principal auth.Principal, staffID uuid.UUID) (CredentialMeta, error)

	CreateShiftTemplate(ctx context.Context, principal auth.Principal, shipID uuid.UUID, input ShiftTemplateInput) (ShiftTemplate, error)
	GetShiftTemplate(ctx context.Context, principal auth.Principal, id uuid.UUID) (ShiftTemplate, error)
	ListShiftTemplates(ctx context.Context, principal auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[ShiftTemplate], error)
	UpdateShiftTemplate(ctx context.Context, principal auth.Principal, id uuid.UUID, input ShiftTemplateUpdate) (ShiftTemplate, error)
	DeleteShiftTemplate(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateCoverage(ctx context.Context, principal auth.Principal, shipID uuid.UUID, input CoverageInput) (CoverageRequirement, error)
	GetCoverage(ctx context.Context, principal auth.Principal, id uuid.UUID) (CoverageRequirement, error)
	ListCoverage(ctx context.Context, principal auth.Principal, shipID uuid.UUID, cursor string, limit int32) (Page[CoverageRequirement], error)
	UpdateCoverage(ctx context.Context, principal auth.Principal, id uuid.UUID, input CoverageUpdate) (CoverageRequirement, error)
	DeleteCoverage(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateLaborRule(ctx context.Context, principal auth.Principal, input LaborRuleInput) (LaborRule, error)
	GetLaborRule(ctx context.Context, principal auth.Principal, id uuid.UUID) (LaborRule, error)
	ListLaborRules(ctx context.Context, principal auth.Principal, shipID *uuid.UUID, cursor string, limit int32) (Page[LaborRule], error)
	UpdateLaborRule(ctx context.Context, principal auth.Principal, id uuid.UUID, input LaborRuleUpdate) (LaborRule, error)
	DeleteLaborRule(ctx context.Context, principal auth.Principal, id uuid.UUID) error

	CreateAutonomy(ctx context.Context, principal auth.Principal, input AutonomyInput) (AutonomyPolicy, error)
	GetAutonomy(ctx context.Context, principal auth.Principal, id uuid.UUID) (AutonomyPolicy, error)
	ListAutonomy(ctx context.Context, principal auth.Principal, shipID *uuid.UUID, cursor string, limit int32) (Page[AutonomyPolicy], error)
	UpdateAutonomy(ctx context.Context, principal auth.Principal, id uuid.UUID, input AutonomyUpdate) (AutonomyPolicy, error)
	DeleteAutonomy(ctx context.Context, principal auth.Principal, id uuid.UUID) error
}
