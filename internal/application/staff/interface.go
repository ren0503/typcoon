package staff

import (
	"context"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/audit"
)

type Repository interface {
	Begin(context.Context) (Tx, error)
	GetStaff(context.Context, uuid.UUID) (Staff, error)
	ListStaff(context.Context, ListStaffParams) ([]Staff, error)
	StaffExists(context.Context, uuid.UUID) (bool, error)
	CreateStaffLeave(context.Context, CreateStaffLeaveParams) (StaffLeafe, error)
	GetStaffLeave(context.Context, uuid.UUID) (StaffLeafe, error)
	ListStaffLeaves(context.Context, ListStaffLeavesParams) ([]StaffLeafe, error)
	UpdateStaffLeave(context.Context, UpdateStaffLeaveParams) (StaffLeafe, error)
	DeleteStaffLeave(context.Context, uuid.UUID) error
	UpsertStaffCredential(context.Context, UpsertStaffCredentialParams) (StaffCredential, error)
	GetStaffCredential(context.Context, uuid.UUID) (StaffCredential, error)
	CreateShiftTemplate(context.Context, CreateShiftTemplateParams) (ShiftTemplate, error)
	GetShiftTemplate(context.Context, uuid.UUID) (ShiftTemplate, error)
	ListShiftTemplates(context.Context, ListShiftTemplatesParams) ([]ShiftTemplate, error)
	UpdateShiftTemplate(context.Context, UpdateShiftTemplateParams) (ShiftTemplate, error)
	DeleteShiftTemplate(context.Context, uuid.UUID) error
	CreateCoverageRequirement(context.Context, CreateCoverageRequirementParams) (CoverageRequirement, error)
	GetCoverageRequirement(context.Context, uuid.UUID) (CoverageRequirement, error)
	ListCoverageRequirements(context.Context, ListCoverageRequirementsParams) ([]CoverageRequirement, error)
	UpdateCoverageRequirement(context.Context, UpdateCoverageRequirementParams) (CoverageRequirement, error)
	DeleteCoverageRequirement(context.Context, uuid.UUID) error
	CreateLaborRule(context.Context, CreateLaborRuleParams) (LaborRule, error)
	GetLaborRule(context.Context, uuid.UUID) (LaborRule, error)
	ListLaborRules(context.Context, ListLaborRulesParams) ([]LaborRule, error)
	UpdateLaborRule(context.Context, UpdateLaborRuleParams) (LaborRule, error)
	DeleteLaborRule(context.Context, uuid.UUID) error
	CreateAutonomyPolicy(context.Context, CreateAutonomyPolicyParams) (AutonomyPolicy, error)
	GetAutonomyPolicy(context.Context, uuid.UUID) (AutonomyPolicy, error)
	ListAutonomyPolicies(context.Context, ListAutonomyPoliciesParams) ([]AutonomyPolicy, error)
	UpdateAutonomyPolicy(context.Context, UpdateAutonomyPolicyParams) (AutonomyPolicy, error)
	DeleteAutonomyPolicy(context.Context, uuid.UUID) error
}

type Tx interface {
	CreateStaff(context.Context, CreateStaffParams) (Staff, error)
	UpdateStaff(context.Context, UpdateStaffParams) (Staff, error)
	SoftDeleteStaff(context.Context, uuid.UUID) error
	AppendAudit(context.Context, audit.Entry) error
	Commit(context.Context) error
	Rollback(context.Context) error
}
