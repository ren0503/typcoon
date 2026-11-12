// Package staff implements staff persistence.
package staff

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appstaff "github.com/ren0503/tycoon/internal/application/staff"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/convert"
	"github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sqlc"
)

type Repo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool, q: sqlc.New(pool)} }

var _ appstaff.Repository = (*Repo)(nil)

func (r *Repo) Begin(ctx context.Context) (appstaff.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, postgres.MapPgError(err)
	}
	return &Tx{tx: tx, q: sqlc.New(tx)}, nil
}
func (r *Repo) GetStaff(ctx context.Context, id uuid.UUID) (appstaff.Staff, error) {
	row, err := r.q.GetStaff(ctx, id)
	return convert.To[appstaff.Staff](row), postgres.MapPgError(err)
}
func (r *Repo) ListStaff(ctx context.Context, p appstaff.ListStaffParams) ([]appstaff.Staff, error) {
	rows, err := r.q.ListStaff(ctx, convert.To[sqlc.ListStaffParams](p))
	return convert.Slice[appstaff.Staff](rows), postgres.MapPgError(err)
}
func (r *Repo) StaffExists(ctx context.Context, id uuid.UUID) (bool, error) {
	ok, err := r.q.StaffExists(ctx, id)
	return ok, postgres.MapPgError(err)
}
func (r *Repo) CreateStaffLeave(ctx context.Context, p appstaff.CreateStaffLeaveParams) (appstaff.StaffLeafe, error) {
	row, err := r.q.CreateStaffLeave(ctx, convert.To[sqlc.CreateStaffLeaveParams](p))
	return convert.To[appstaff.StaffLeafe](row), postgres.MapPgError(err)
}
func (r *Repo) GetStaffLeave(ctx context.Context, id uuid.UUID) (appstaff.StaffLeafe, error) {
	row, err := r.q.GetStaffLeave(ctx, id)
	return convert.To[appstaff.StaffLeafe](row), postgres.MapPgError(err)
}
func (r *Repo) ListStaffLeaves(ctx context.Context, p appstaff.ListStaffLeavesParams) ([]appstaff.StaffLeafe, error) {
	rows, err := r.q.ListStaffLeaves(ctx, convert.To[sqlc.ListStaffLeavesParams](p))
	return convert.Slice[appstaff.StaffLeafe](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateStaffLeave(ctx context.Context, p appstaff.UpdateStaffLeaveParams) (appstaff.StaffLeafe, error) {
	row, err := r.q.UpdateStaffLeave(ctx, convert.To[sqlc.UpdateStaffLeaveParams](p))
	return convert.To[appstaff.StaffLeafe](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteStaffLeave(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeleteStaffLeave(ctx, id))
}
func (r *Repo) UpsertStaffCredential(ctx context.Context, p appstaff.UpsertStaffCredentialParams) (appstaff.StaffCredential, error) {
	row, err := r.q.UpsertStaffCredential(ctx, convert.To[sqlc.UpsertStaffCredentialParams](p))
	return convert.To[appstaff.StaffCredential](row), postgres.MapPgError(err)
}
func (r *Repo) GetStaffCredential(ctx context.Context, id uuid.UUID) (appstaff.StaffCredential, error) {
	row, err := r.q.GetStaffCredential(ctx, id)
	return convert.To[appstaff.StaffCredential](row), postgres.MapPgError(err)
}
func (r *Repo) CreateShiftTemplate(ctx context.Context, p appstaff.CreateShiftTemplateParams) (appstaff.ShiftTemplate, error) {
	row, err := r.q.CreateShiftTemplate(ctx, convert.To[sqlc.CreateShiftTemplateParams](p))
	return convert.To[appstaff.ShiftTemplate](row), postgres.MapPgError(err)
}
func (r *Repo) GetShiftTemplate(ctx context.Context, id uuid.UUID) (appstaff.ShiftTemplate, error) {
	row, err := r.q.GetShiftTemplate(ctx, id)
	return convert.To[appstaff.ShiftTemplate](row), postgres.MapPgError(err)
}
func (r *Repo) ListShiftTemplates(ctx context.Context, p appstaff.ListShiftTemplatesParams) ([]appstaff.ShiftTemplate, error) {
	rows, err := r.q.ListShiftTemplates(ctx, convert.To[sqlc.ListShiftTemplatesParams](p))
	return convert.Slice[appstaff.ShiftTemplate](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateShiftTemplate(ctx context.Context, p appstaff.UpdateShiftTemplateParams) (appstaff.ShiftTemplate, error) {
	row, err := r.q.UpdateShiftTemplate(ctx, convert.To[sqlc.UpdateShiftTemplateParams](p))
	return convert.To[appstaff.ShiftTemplate](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteShiftTemplate(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeleteShiftTemplate(ctx, id))
}
func (r *Repo) CreateCoverageRequirement(ctx context.Context, p appstaff.CreateCoverageRequirementParams) (appstaff.CoverageRequirement, error) {
	row, err := r.q.CreateCoverageRequirement(ctx, convert.To[sqlc.CreateCoverageRequirementParams](p))
	return convert.To[appstaff.CoverageRequirement](row), postgres.MapPgError(err)
}
func (r *Repo) GetCoverageRequirement(ctx context.Context, id uuid.UUID) (appstaff.CoverageRequirement, error) {
	row, err := r.q.GetCoverageRequirement(ctx, id)
	return convert.To[appstaff.CoverageRequirement](row), postgres.MapPgError(err)
}
func (r *Repo) ListCoverageRequirements(ctx context.Context, p appstaff.ListCoverageRequirementsParams) ([]appstaff.CoverageRequirement, error) {
	rows, err := r.q.ListCoverageRequirements(ctx, convert.To[sqlc.ListCoverageRequirementsParams](p))
	return convert.Slice[appstaff.CoverageRequirement](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateCoverageRequirement(ctx context.Context, p appstaff.UpdateCoverageRequirementParams) (appstaff.CoverageRequirement, error) {
	row, err := r.q.UpdateCoverageRequirement(ctx, convert.To[sqlc.UpdateCoverageRequirementParams](p))
	return convert.To[appstaff.CoverageRequirement](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteCoverageRequirement(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeleteCoverageRequirement(ctx, id))
}
func (r *Repo) CreateLaborRule(ctx context.Context, p appstaff.CreateLaborRuleParams) (appstaff.LaborRule, error) {
	row, err := r.q.CreateLaborRule(ctx, convert.To[sqlc.CreateLaborRuleParams](p))
	return convert.To[appstaff.LaborRule](row), postgres.MapPgError(err)
}
func (r *Repo) GetLaborRule(ctx context.Context, id uuid.UUID) (appstaff.LaborRule, error) {
	row, err := r.q.GetLaborRule(ctx, id)
	return convert.To[appstaff.LaborRule](row), postgres.MapPgError(err)
}
func (r *Repo) ListLaborRules(ctx context.Context, p appstaff.ListLaborRulesParams) ([]appstaff.LaborRule, error) {
	rows, err := r.q.ListLaborRules(ctx, convert.To[sqlc.ListLaborRulesParams](p))
	return convert.Slice[appstaff.LaborRule](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateLaborRule(ctx context.Context, p appstaff.UpdateLaborRuleParams) (appstaff.LaborRule, error) {
	row, err := r.q.UpdateLaborRule(ctx, convert.To[sqlc.UpdateLaborRuleParams](p))
	return convert.To[appstaff.LaborRule](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteLaborRule(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeleteLaborRule(ctx, id))
}
func (r *Repo) CreateAutonomyPolicy(ctx context.Context, p appstaff.CreateAutonomyPolicyParams) (appstaff.AutonomyPolicy, error) {
	row, err := r.q.CreateAutonomyPolicy(ctx, convert.To[sqlc.CreateAutonomyPolicyParams](p))
	return convert.To[appstaff.AutonomyPolicy](row), postgres.MapPgError(err)
}
func (r *Repo) GetAutonomyPolicy(ctx context.Context, id uuid.UUID) (appstaff.AutonomyPolicy, error) {
	row, err := r.q.GetAutonomyPolicy(ctx, id)
	return convert.To[appstaff.AutonomyPolicy](row), postgres.MapPgError(err)
}
func (r *Repo) ListAutonomyPolicies(ctx context.Context, p appstaff.ListAutonomyPoliciesParams) ([]appstaff.AutonomyPolicy, error) {
	rows, err := r.q.ListAutonomyPolicies(ctx, convert.To[sqlc.ListAutonomyPoliciesParams](p))
	return convert.Slice[appstaff.AutonomyPolicy](rows), postgres.MapPgError(err)
}
func (r *Repo) UpdateAutonomyPolicy(ctx context.Context, p appstaff.UpdateAutonomyPolicyParams) (appstaff.AutonomyPolicy, error) {
	row, err := r.q.UpdateAutonomyPolicy(ctx, convert.To[sqlc.UpdateAutonomyPolicyParams](p))
	return convert.To[appstaff.AutonomyPolicy](row), postgres.MapPgError(err)
}
func (r *Repo) DeleteAutonomyPolicy(ctx context.Context, id uuid.UUID) error {
	return mapDelete(r.q.SoftDeleteAutonomyPolicy(ctx, id))
}
func mapDelete(_ int64, err error) error {
	if err != nil {
		return postgres.MapPgError(err)
	}
	return nil
}
