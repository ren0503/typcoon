package staff

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
)

type repo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func newRepo(pool *pgxpool.Pool) *repo {
	return &repo{pool: pool, q: sqlc.New(pool)}
}

func (r *repo) begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, database.MapPgError(err)
	}
	return tx, nil
}

func (r *repo) getStaff(ctx context.Context, id uuid.UUID) (sqlc.Staff, error) {
	row, err := r.q.GetStaff(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listStaff(ctx context.Context, p sqlc.ListStaffParams) ([]sqlc.Staff, error) {
	rows, err := r.q.ListStaff(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) staffExists(ctx context.Context, id uuid.UUID) (bool, error) {
	ok, err := r.q.StaffExists(ctx, id)
	return ok, database.MapPgError(err)
}

func (r *repo) createLeave(ctx context.Context, p sqlc.CreateStaffLeaveParams) (sqlc.StaffLeafe, error) {
	row, err := r.q.CreateStaffLeave(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getLeave(ctx context.Context, id uuid.UUID) (sqlc.StaffLeafe, error) {
	row, err := r.q.GetStaffLeave(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listLeaves(ctx context.Context, p sqlc.ListStaffLeavesParams) ([]sqlc.StaffLeafe, error) {
	rows, err := r.q.ListStaffLeaves(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateLeave(ctx context.Context, p sqlc.UpdateStaffLeaveParams) (sqlc.StaffLeafe, error) {
	row, err := r.q.UpdateStaffLeave(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteLeave(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteStaffLeave(ctx, id)
	return execResult(n, err)
}

func (r *repo) upsertCredential(ctx context.Context, p sqlc.UpsertStaffCredentialParams) (sqlc.StaffCredential, error) {
	row, err := r.q.UpsertStaffCredential(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getCredential(ctx context.Context, staffID uuid.UUID) (sqlc.StaffCredential, error) {
	row, err := r.q.GetStaffCredential(ctx, staffID)
	return row, database.MapPgError(err)
}

func (r *repo) createShiftTemplate(ctx context.Context, p sqlc.CreateShiftTemplateParams) (sqlc.ShiftTemplate, error) {
	row, err := r.q.CreateShiftTemplate(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getShiftTemplate(ctx context.Context, id uuid.UUID) (sqlc.ShiftTemplate, error) {
	row, err := r.q.GetShiftTemplate(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listShiftTemplates(ctx context.Context, p sqlc.ListShiftTemplatesParams) ([]sqlc.ShiftTemplate, error) {
	rows, err := r.q.ListShiftTemplates(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateShiftTemplate(ctx context.Context, p sqlc.UpdateShiftTemplateParams) (sqlc.ShiftTemplate, error) {
	row, err := r.q.UpdateShiftTemplate(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteShiftTemplate(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteShiftTemplate(ctx, id)
	return execResult(n, err)
}

func (r *repo) createCoverage(ctx context.Context, p sqlc.CreateCoverageRequirementParams) (sqlc.CoverageRequirement, error) {
	row, err := r.q.CreateCoverageRequirement(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getCoverage(ctx context.Context, id uuid.UUID) (sqlc.CoverageRequirement, error) {
	row, err := r.q.GetCoverageRequirement(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listCoverage(ctx context.Context, p sqlc.ListCoverageRequirementsParams) ([]sqlc.CoverageRequirement, error) {
	rows, err := r.q.ListCoverageRequirements(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateCoverage(ctx context.Context, p sqlc.UpdateCoverageRequirementParams) (sqlc.CoverageRequirement, error) {
	row, err := r.q.UpdateCoverageRequirement(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteCoverage(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteCoverageRequirement(ctx, id)
	return execResult(n, err)
}

func (r *repo) createLaborRule(ctx context.Context, p sqlc.CreateLaborRuleParams) (sqlc.LaborRule, error) {
	row, err := r.q.CreateLaborRule(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getLaborRule(ctx context.Context, id uuid.UUID) (sqlc.LaborRule, error) {
	row, err := r.q.GetLaborRule(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listLaborRules(ctx context.Context, p sqlc.ListLaborRulesParams) ([]sqlc.LaborRule, error) {
	rows, err := r.q.ListLaborRules(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateLaborRule(ctx context.Context, p sqlc.UpdateLaborRuleParams) (sqlc.LaborRule, error) {
	row, err := r.q.UpdateLaborRule(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteLaborRule(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteLaborRule(ctx, id)
	return execResult(n, err)
}

func (r *repo) createAutonomy(ctx context.Context, p sqlc.CreateAutonomyPolicyParams) (sqlc.AutonomyPolicy, error) {
	row, err := r.q.CreateAutonomyPolicy(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) getAutonomy(ctx context.Context, id uuid.UUID) (sqlc.AutonomyPolicy, error) {
	row, err := r.q.GetAutonomyPolicy(ctx, id)
	return row, database.MapPgError(err)
}

func (r *repo) listAutonomy(ctx context.Context, p sqlc.ListAutonomyPoliciesParams) ([]sqlc.AutonomyPolicy, error) {
	rows, err := r.q.ListAutonomyPolicies(ctx, p)
	return rows, database.MapPgError(err)
}

func (r *repo) updateAutonomy(ctx context.Context, p sqlc.UpdateAutonomyPolicyParams) (sqlc.AutonomyPolicy, error) {
	row, err := r.q.UpdateAutonomyPolicy(ctx, p)
	return row, database.MapPgError(err)
}

func (r *repo) deleteAutonomy(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.SoftDeleteAutonomyPolicy(ctx, id)
	return execResult(n, err)
}

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}
