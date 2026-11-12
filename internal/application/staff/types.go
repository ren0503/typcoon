package staff

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CreateStaffParams struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	ShipID         uuid.NullUUID   `json:"ship_id"`
	EmployeeCode   string          `json:"employee_code"`
	FullName       string          `json:"full_name"`
	Department     string          `json:"department"`
	RoleTitle      *string         `json:"role_title"`
	IsSupervisor   bool            `json:"is_supervisor"`
	Languages      []string        `json:"languages"`
	Phone          *string         `json:"phone"`
	Certifications json.RawMessage `json:"certifications"`
	Status         string          `json:"status"`
	HiredOn        *time.Time      `json:"hired_on"`
}

type UpdateStaffParams struct {
	ShipID         uuid.NullUUID   `json:"ship_id"`
	EmployeeCode   string          `json:"employee_code"`
	FullName       string          `json:"full_name"`
	Department     string          `json:"department"`
	RoleTitle      *string         `json:"role_title"`
	IsSupervisor   bool            `json:"is_supervisor"`
	Languages      []string        `json:"languages"`
	Phone          *string         `json:"phone"`
	Certifications json.RawMessage `json:"certifications"`
	Status         string          `json:"status"`
	HiredOn        *time.Time      `json:"hired_on"`
	ID             uuid.UUID       `json:"id"`
}

type ListStaffParams struct {
	TenantID  uuid.NullUUID `json:"tenant_id"`
	ShipID    uuid.NullUUID `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateStaffLeaveParams struct {
	ID        uuid.UUID `json:"id"`
	StaffID   uuid.UUID `json:"staff_id"`
	LeaveType string    `json:"leave_type"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Note      *string   `json:"note"`
}

type UpdateStaffLeaveParams struct {
	LeaveType string    `json:"leave_type"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Note      *string   `json:"note"`
	ID        uuid.UUID `json:"id"`
}

type ListStaffLeavesParams struct {
	StaffID   uuid.UUID     `json:"staff_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type UpsertStaffCredentialParams struct {
	ID      uuid.UUID `json:"id"`
	StaffID uuid.UUID `json:"staff_id"`
	PinHash string    `json:"pin_hash"`
}

type CreateShiftTemplateParams struct {
	ID         uuid.UUID `json:"id"`
	ShipID     uuid.UUID `json:"ship_id"`
	Department string    `json:"department"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	StartTime  TimeOfDay `json:"start_time"`
	EndTime    TimeOfDay `json:"end_time"`
}

type UpdateShiftTemplateParams struct {
	Department string    `json:"department"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	StartTime  TimeOfDay `json:"start_time"`
	EndTime    TimeOfDay `json:"end_time"`
	ID         uuid.UUID `json:"id"`
}

type ListShiftTemplatesParams struct {
	ShipID    uuid.UUID     `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateCoverageRequirementParams struct {
	ID              uuid.UUID     `json:"id"`
	ShipID          uuid.UUID     `json:"ship_id"`
	Department      string        `json:"department"`
	ShiftTemplateID uuid.UUID     `json:"shift_template_id"`
	PositionCode    string        `json:"position_code"`
	DeckID          uuid.NullUUID `json:"deck_id"`
	AreaID          uuid.NullUUID `json:"area_id"`
	MinStaff        int16         `json:"min_staff"`
	MaxStaff        *int16        `json:"max_staff"`
	DaysOfWeek      []int16       `json:"days_of_week"`
	ValidFrom       *time.Time    `json:"valid_from"`
	ValidTo         *time.Time    `json:"valid_to"`
}

type UpdateCoverageRequirementParams struct {
	Department      string        `json:"department"`
	ShiftTemplateID uuid.UUID     `json:"shift_template_id"`
	PositionCode    string        `json:"position_code"`
	DeckID          uuid.NullUUID `json:"deck_id"`
	AreaID          uuid.NullUUID `json:"area_id"`
	MinStaff        int16         `json:"min_staff"`
	MaxStaff        *int16        `json:"max_staff"`
	DaysOfWeek      []int16       `json:"days_of_week"`
	ValidFrom       *time.Time    `json:"valid_from"`
	ValidTo         *time.Time    `json:"valid_to"`
	ID              uuid.UUID     `json:"id"`
}

type ListCoverageRequirementsParams struct {
	ShipID    uuid.UUID     `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateLaborRuleParams struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	ShipID    uuid.NullUUID   `json:"ship_id"`
	RuleCode  string          `json:"rule_code"`
	Severity  string          `json:"severity"`
	Params    json.RawMessage `json:"params"`
	ValidFrom *time.Time      `json:"valid_from"`
	ValidTo   *time.Time      `json:"valid_to"`
}

type UpdateLaborRuleParams struct {
	ShipID    uuid.NullUUID   `json:"ship_id"`
	RuleCode  string          `json:"rule_code"`
	Severity  string          `json:"severity"`
	Params    json.RawMessage `json:"params"`
	ValidFrom *time.Time      `json:"valid_from"`
	ValidTo   *time.Time      `json:"valid_to"`
	ID        uuid.UUID       `json:"id"`
}

type ListLaborRulesParams struct {
	TenantID  uuid.NullUUID `json:"tenant_id"`
	ShipID    uuid.NullUUID `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type CreateAutonomyPolicyParams struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	ShipID     uuid.NullUUID   `json:"ship_id"`
	ChangeType string          `json:"change_type"`
	Level      string          `json:"level"`
	Conditions json.RawMessage `json:"conditions"`
}

type UpdateAutonomyPolicyParams struct {
	ShipID     uuid.NullUUID   `json:"ship_id"`
	ChangeType string          `json:"change_type"`
	Level      string          `json:"level"`
	Conditions json.RawMessage `json:"conditions"`
	ID         uuid.UUID       `json:"id"`
}

type ListAutonomyPoliciesParams struct {
	TenantID  uuid.NullUUID `json:"tenant_id"`
	ShipID    uuid.NullUUID `json:"ship_id"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

type Staff struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	ShipID         uuid.NullUUID   `json:"ship_id"`
	EmployeeCode   string          `json:"employee_code"`
	FullName       string          `json:"full_name"`
	Department     string          `json:"department"`
	RoleTitle      *string         `json:"role_title"`
	IsSupervisor   bool            `json:"is_supervisor"`
	Languages      []string        `json:"languages"`
	Phone          *string         `json:"phone"`
	Certifications json.RawMessage `json:"certifications"`
	Status         string          `json:"status"`
	HiredOn        *time.Time      `json:"hired_on"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Version        int64           `json:"version"`
	DeletedAt      *time.Time      `json:"deleted_at"`
}

type StaffLeafe struct {
	ID        uuid.UUID  `json:"id"`
	StaffID   uuid.UUID  `json:"staff_id"`
	LeaveType string     `json:"leave_type"`
	StartsAt  time.Time  `json:"starts_at"`
	EndsAt    time.Time  `json:"ends_at"`
	Note      *string    `json:"note"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Version   int64      `json:"version"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type StaffCredential struct {
	ID           uuid.UUID  `json:"id"`
	StaffID      uuid.UUID  `json:"staff_id"`
	PinHash      string     `json:"pin_hash"`
	PinUpdatedAt time.Time  `json:"pin_updated_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Version      int64      `json:"version"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

type ShiftTemplate struct {
	ID         uuid.UUID  `json:"id"`
	ShipID     uuid.UUID  `json:"ship_id"`
	Department string     `json:"department"`
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	StartTime  TimeOfDay  `json:"start_time"`
	EndTime    TimeOfDay  `json:"end_time"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Version    int64      `json:"version"`
	DeletedAt  *time.Time `json:"deleted_at"`
}

type CoverageRequirement struct {
	ID              uuid.UUID     `json:"id"`
	ShipID          uuid.UUID     `json:"ship_id"`
	Department      string        `json:"department"`
	ShiftTemplateID uuid.UUID     `json:"shift_template_id"`
	PositionCode    string        `json:"position_code"`
	DeckID          uuid.NullUUID `json:"deck_id"`
	AreaID          uuid.NullUUID `json:"area_id"`
	MinStaff        int16         `json:"min_staff"`
	MaxStaff        *int16        `json:"max_staff"`
	DaysOfWeek      []int16       `json:"days_of_week"`
	ValidFrom       *time.Time    `json:"valid_from"`
	ValidTo         *time.Time    `json:"valid_to"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Version         int64         `json:"version"`
	DeletedAt       *time.Time    `json:"deleted_at"`
}

type LaborRule struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	ShipID    uuid.NullUUID   `json:"ship_id"`
	RuleCode  string          `json:"rule_code"`
	Severity  string          `json:"severity"`
	Params    json.RawMessage `json:"params"`
	ValidFrom *time.Time      `json:"valid_from"`
	ValidTo   *time.Time      `json:"valid_to"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Version   int64           `json:"version"`
	DeletedAt *time.Time      `json:"deleted_at"`
}

type AutonomyPolicy struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	ShipID     uuid.NullUUID   `json:"ship_id"`
	ChangeType string          `json:"change_type"`
	Level      string          `json:"level"`
	Conditions json.RawMessage `json:"conditions"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	Version    int64           `json:"version"`
	DeletedAt  *time.Time      `json:"deleted_at"`
}
