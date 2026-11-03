package staff

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/apperrors"
	"github.com/ren0503/tycoon/internal/audit"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/database"
	"github.com/ren0503/tycoon/internal/database/sqlc"
	"github.com/ren0503/tycoon/internal/pagination"
	"github.com/ren0503/tycoon/internal/ship"
)

var (
	departments    = map[string]bool{"security": true, "cleaning": true, "operator": true}
	staffStatuses  = map[string]bool{"active": true, "on_leave": true, "sick": true, "inactive": true}
	leaveTypes     = map[string]bool{"annual": true, "sick": true, "rest": true, "training": true, "other": true}
	severities     = map[string]bool{"hard": true, "soft": true}
	autonomyLevels = map[string]bool{"auto": true, "conditional": true, "approval": true, "never": true}
)

type service struct {
	repo  *repo
	ships ship.Service
	audit audit.Appender
}

// New builds the staff service.
func New(pool *pgxpool.Pool, ships ship.Service, appender audit.Appender) Service {
	return &service{repo: newRepo(pool), ships: ships, audit: appender}
}

func (s *service) CreateStaff(ctx context.Context, p auth.Principal, input StaffInput) (sqlc.Staff, error) {
	if err := s.requireTenant(p); err != nil {
		return sqlc.Staff{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			return sqlc.Staff{}, err
		}
	}
	code := strings.TrimSpace(input.EmployeeCode)
	name := strings.TrimSpace(input.FullName)
	if code == "" || name == "" || !departments[input.Department] {
		return sqlc.Staff{}, validation("Employee code, full name, and a recognised department are required.")
	}
	certifications, err := certificationsJSON(input.Certifications)
	if err != nil {
		return sqlc.Staff{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.Staff{}, apperrors.Internal(err)
	}
	tx, err := s.repo.begin(ctx)
	if err != nil {
		return sqlc.Staff{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	row, err := sqlc.New(tx).CreateStaff(ctx, sqlc.CreateStaffParams{
		ID: id, TenantID: p.TenantID, ShipID: optionalUUID(input.ShipID), EmployeeCode: code, FullName: name,
		Department: input.Department, RoleTitle: trimPtr(input.RoleTitle), IsSupervisor: input.IsSupervisor,
		Languages: emptyStrings(input.Languages), Phone: trimPtr(input.Phone), Certifications: certifications,
		Status: "active", HiredOn: input.HiredOn,
	})
	if err != nil {
		return sqlc.Staff{}, database.MapPgError(err)
	}
	shipID := uuid.Nil
	if row.ShipID.Valid {
		shipID = row.ShipID.UUID
	}
	if err := s.appendAudit(ctx, tx, p, "staff.created", row.ID, row.TenantID, shipID, map[string]any{
		"employee_code": row.EmployeeCode, "department": row.Department, "status": row.Status,
	}); err != nil {
		return sqlc.Staff{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.Staff{}, database.MapPgError(err)
	}
	return row, nil
}

func (s *service) GetStaff(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.Staff, error) {
	row, err := s.repo.getStaff(ctx, id)
	if err != nil {
		return sqlc.Staff{}, err
	}
	if err := s.checkStaffScope(ctx, p, row); err != nil {
		return sqlc.Staff{}, err
	}
	return row, nil
}

func (s *service) ListStaff(ctx context.Context, p auth.Principal, shipID *uuid.UUID, rawCursor string, limit int32) (Page[sqlc.Staff], error) {
	if shipID != nil {
		if err := s.requireShip(ctx, p, *shipID); err != nil {
			return Page[sqlc.Staff]{}, err
		}
	}
	parts := []string{"staff", p.TenantID.String(), optionalUUIDString(shipID)}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[sqlc.Staff]{}, err
	}
	rows, err := s.repo.listStaff(ctx, sqlc.ListStaffParams{
		TenantID: nullUUID(p.TenantID), ShipID: optionalUUID(shipID), After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[sqlc.Staff]{}, err
	}
	page := Page[sqlc.Staff]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateStaff(ctx context.Context, p auth.Principal, id uuid.UUID, input StaffUpdate) (sqlc.Staff, error) {
	current, err := s.GetStaff(ctx, p, id)
	if err != nil {
		return sqlc.Staff{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			return sqlc.Staff{}, err
		}
		current.ShipID = optionalUUID(input.ShipID)
	}
	if input.EmployeeCode != nil {
		if strings.TrimSpace(*input.EmployeeCode) == "" {
			return sqlc.Staff{}, validation("Employee code must not be empty.")
		}
		current.EmployeeCode = strings.TrimSpace(*input.EmployeeCode)
	}
	if input.FullName != nil {
		if strings.TrimSpace(*input.FullName) == "" {
			return sqlc.Staff{}, validation("Full name must not be empty.")
		}
		current.FullName = strings.TrimSpace(*input.FullName)
	}
	if input.Department != nil {
		if !departments[*input.Department] {
			return sqlc.Staff{}, validation("Department is not recognised.")
		}
		current.Department = *input.Department
	}
	if input.RoleTitle != nil {
		current.RoleTitle = trimPtr(input.RoleTitle)
	}
	if input.IsSupervisor != nil {
		current.IsSupervisor = *input.IsSupervisor
	}
	if input.Languages != nil {
		current.Languages = emptyStrings(input.Languages)
	}
	if input.Phone != nil {
		current.Phone = trimPtr(input.Phone)
	}
	if input.Certifications != nil {
		certifications, err := certificationsJSON(input.Certifications)
		if err != nil {
			return sqlc.Staff{}, err
		}
		current.Certifications = certifications
	}
	if input.Status != nil {
		if !staffStatuses[*input.Status] {
			return sqlc.Staff{}, validation("Staff status is not recognised.")
		}
		current.Status = *input.Status
	}
	if input.HiredOn != nil {
		current.HiredOn = *input.HiredOn
	}
	tx, err := s.repo.begin(ctx)
	if err != nil {
		return sqlc.Staff{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	row, err := sqlc.New(tx).UpdateStaff(ctx, sqlc.UpdateStaffParams{
		ID: id, ShipID: current.ShipID, EmployeeCode: current.EmployeeCode, FullName: current.FullName,
		Department: current.Department, RoleTitle: current.RoleTitle, IsSupervisor: current.IsSupervisor,
		Languages: current.Languages, Phone: current.Phone, Certifications: current.Certifications,
		Status: current.Status, HiredOn: current.HiredOn,
	})
	if err != nil {
		return sqlc.Staff{}, database.MapPgError(err)
	}
	shipID := uuid.Nil
	if row.ShipID.Valid {
		shipID = row.ShipID.UUID
	}
	if err := s.appendAudit(ctx, tx, p, "staff.updated", row.ID, row.TenantID, shipID, map[string]any{
		"employee_code": row.EmployeeCode, "department": row.Department, "status": row.Status,
	}); err != nil {
		return sqlc.Staff{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.Staff{}, database.MapPgError(err)
	}
	return row, nil
}

func (s *service) DeleteStaff(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	current, err := s.GetStaff(ctx, p, id)
	if err != nil {
		return err
	}
	tx, err := s.repo.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	n, err := sqlc.New(tx).SoftDeleteStaff(ctx, id)
	if err != nil {
		return database.MapPgError(err)
	}
	if n == 0 {
		return database.MapPgError(pgx.ErrNoRows)
	}
	shipID := uuid.Nil
	if current.ShipID.Valid {
		shipID = current.ShipID.UUID
	}
	if err := s.appendAudit(ctx, tx, p, "staff.deleted", id, current.TenantID, shipID, nil); err != nil {
		return err
	}
	return database.MapPgError(tx.Commit(ctx))
}

func (s *service) appendAudit(ctx context.Context, tx pgx.Tx, p auth.Principal, action string, entityID, tenantID, shipID uuid.UUID, details map[string]any) error {
	if s.audit == nil {
		return nil
	}
	payload := json.RawMessage(`{}`)
	if details != nil {
		encoded, err := json.Marshal(details)
		if err != nil {
			return apperrors.Internal(err)
		}
		payload = encoded
	}
	actorID := p.ActorID
	entity := "staff"
	entityRef := entityID.String()
	entry := audit.Entry{
		Source: "cloud", ActorType: string(p.ActorType), ActorID: optionalString(actorID),
		Action: action, Entity: &entity, EntityID: &entityRef, Details: payload,
	}
	if tenantID != uuid.Nil {
		_ = tenantID
	}
	if shipID != uuid.Nil {
		entry.ShipID = &shipID
	}
	return s.audit.AppendTx(ctx, tx, entry)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (s *service) StaffExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.repo.staffExists(ctx, id)
}

func (s *service) CreateLeave(ctx context.Context, p auth.Principal, staffID uuid.UUID, input LeaveInput) (sqlc.StaffLeafe, error) {
	if _, err := s.GetStaff(ctx, p, staffID); err != nil {
		return sqlc.StaffLeafe{}, err
	}
	if !leaveTypes[input.LeaveType] || !input.EndsAt.After(input.StartsAt) {
		return sqlc.StaffLeafe{}, validation("Leave type must be recognised and end after start.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.StaffLeafe{}, apperrors.Internal(err)
	}
	return s.repo.createLeave(ctx, sqlc.CreateStaffLeaveParams{
		ID: id, StaffID: staffID, LeaveType: input.LeaveType, StartsAt: input.StartsAt, EndsAt: input.EndsAt,
		Note: trimPtr(input.Note),
	})
}

func (s *service) GetLeave(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.StaffLeafe, error) {
	row, err := s.repo.getLeave(ctx, id)
	if err != nil {
		return sqlc.StaffLeafe{}, err
	}
	if _, err := s.GetStaff(ctx, p, row.StaffID); err != nil {
		return sqlc.StaffLeafe{}, err
	}
	return row, nil
}

func (s *service) ListLeaves(ctx context.Context, p auth.Principal, staffID uuid.UUID, rawCursor string, limit int32) (Page[sqlc.StaffLeafe], error) {
	if _, err := s.GetStaff(ctx, p, staffID); err != nil {
		return Page[sqlc.StaffLeafe]{}, err
	}
	parts := []string{"staff_leaves", staffID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[sqlc.StaffLeafe]{}, err
	}
	rows, err := s.repo.listLeaves(ctx, sqlc.ListStaffLeavesParams{StaffID: staffID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[sqlc.StaffLeafe]{}, err
	}
	page := Page[sqlc.StaffLeafe]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateLeave(ctx context.Context, p auth.Principal, id uuid.UUID, input LeaveUpdate) (sqlc.StaffLeafe, error) {
	current, err := s.GetLeave(ctx, p, id)
	if err != nil {
		return sqlc.StaffLeafe{}, err
	}
	if input.LeaveType != nil {
		if !leaveTypes[*input.LeaveType] {
			return sqlc.StaffLeafe{}, validation("Leave type is not recognised.")
		}
		current.LeaveType = *input.LeaveType
	}
	if input.StartsAt != nil {
		current.StartsAt = *input.StartsAt
	}
	if input.EndsAt != nil {
		current.EndsAt = *input.EndsAt
	}
	if input.Note != nil {
		current.Note = trimPtr(input.Note)
	}
	if !current.EndsAt.After(current.StartsAt) {
		return sqlc.StaffLeafe{}, validation("Leave end must be after start.")
	}
	return s.repo.updateLeave(ctx, sqlc.UpdateStaffLeaveParams{
		ID: id, LeaveType: current.LeaveType, StartsAt: current.StartsAt, EndsAt: current.EndsAt, Note: current.Note,
	})
}

func (s *service) DeleteLeave(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetLeave(ctx, p, id); err != nil {
		return err
	}
	return s.repo.deleteLeave(ctx, id)
}

func (s *service) SetCredential(ctx context.Context, p auth.Principal, staffID uuid.UUID, pin string) (CredentialMeta, error) {
	if _, err := s.GetStaff(ctx, p, staffID); err != nil {
		return CredentialMeta{}, err
	}
	if len(pin) < 4 || len(pin) > 12 {
		return CredentialMeta{}, validation("PIN must be between 4 and 12 characters.")
	}
	hash, err := hashPIN(pin)
	if err != nil {
		return CredentialMeta{}, apperrors.Internal(err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return CredentialMeta{}, apperrors.Internal(err)
	}
	row, err := s.repo.upsertCredential(ctx, sqlc.UpsertStaffCredentialParams{ID: id, StaffID: staffID, PinHash: hash})
	if err != nil {
		return CredentialMeta{}, err
	}
	return CredentialMeta{StaffID: row.StaffID, PinUpdatedAt: row.PinUpdatedAt}, nil
}

func (s *service) CredentialMeta(ctx context.Context, p auth.Principal, staffID uuid.UUID) (CredentialMeta, error) {
	if _, err := s.GetStaff(ctx, p, staffID); err != nil {
		return CredentialMeta{}, err
	}
	row, err := s.repo.getCredential(ctx, staffID)
	if err != nil {
		return CredentialMeta{}, err
	}
	return CredentialMeta{StaffID: row.StaffID, PinUpdatedAt: row.PinUpdatedAt}, nil
}

func (s *service) CreateShiftTemplate(ctx context.Context, p auth.Principal, shipID uuid.UUID, input ShiftTemplateInput) (sqlc.ShiftTemplate, error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return sqlc.ShiftTemplate{}, err
	}
	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
	if code == "" || name == "" || !departments[input.Department] || !input.StartTime.Valid || !input.EndTime.Valid {
		return sqlc.ShiftTemplate{}, validation("Shift template code, name, department, and times are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.ShiftTemplate{}, apperrors.Internal(err)
	}
	return s.repo.createShiftTemplate(ctx, sqlc.CreateShiftTemplateParams{
		ID: id, ShipID: shipID, Department: input.Department, Code: code, Name: name,
		StartTime: input.StartTime, EndTime: input.EndTime,
	})
}

func (s *service) GetShiftTemplate(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.ShiftTemplate, error) {
	row, err := s.repo.getShiftTemplate(ctx, id)
	if err != nil {
		return sqlc.ShiftTemplate{}, err
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		return sqlc.ShiftTemplate{}, err
	}
	return row, nil
}

func (s *service) ListShiftTemplates(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[sqlc.ShiftTemplate], error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return Page[sqlc.ShiftTemplate]{}, err
	}
	parts := []string{"shift_templates", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[sqlc.ShiftTemplate]{}, err
	}
	rows, err := s.repo.listShiftTemplates(ctx, sqlc.ListShiftTemplatesParams{ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[sqlc.ShiftTemplate]{}, err
	}
	page := Page[sqlc.ShiftTemplate]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateShiftTemplate(ctx context.Context, p auth.Principal, id uuid.UUID, input ShiftTemplateUpdate) (sqlc.ShiftTemplate, error) {
	current, err := s.GetShiftTemplate(ctx, p, id)
	if err != nil {
		return sqlc.ShiftTemplate{}, err
	}
	if input.Department != nil {
		if !departments[*input.Department] {
			return sqlc.ShiftTemplate{}, validation("Department is not recognised.")
		}
		current.Department = *input.Department
	}
	if input.Code != nil {
		current.Code = strings.TrimSpace(*input.Code)
	}
	if input.Name != nil {
		current.Name = strings.TrimSpace(*input.Name)
	}
	if input.StartTime != nil {
		current.StartTime = *input.StartTime
	}
	if input.EndTime != nil {
		current.EndTime = *input.EndTime
	}
	if current.Code == "" || current.Name == "" || !current.StartTime.Valid || !current.EndTime.Valid {
		return sqlc.ShiftTemplate{}, validation("Shift template code, name, and times are required.")
	}
	return s.repo.updateShiftTemplate(ctx, sqlc.UpdateShiftTemplateParams{
		ID: id, Department: current.Department, Code: current.Code, Name: current.Name,
		StartTime: current.StartTime, EndTime: current.EndTime,
	})
}

func (s *service) DeleteShiftTemplate(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetShiftTemplate(ctx, p, id); err != nil {
		return err
	}
	return s.repo.deleteShiftTemplate(ctx, id)
}

func (s *service) CreateCoverage(ctx context.Context, p auth.Principal, shipID uuid.UUID, input CoverageInput) (sqlc.CoverageRequirement, error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return sqlc.CoverageRequirement{}, err
	}
	template, err := s.repo.getShiftTemplate(ctx, input.ShiftTemplateID)
	if err != nil {
		return sqlc.CoverageRequirement{}, apperrors.Validation(apperrors.CodeInvalidReference, "Shift template does not exist on this ship.")
	}
	if template.ShipID != shipID {
		return sqlc.CoverageRequirement{}, apperrors.Validation(apperrors.CodeInvalidReference, "Shift template does not exist on this ship.")
	}
	if err := validateCoverage(input.Department, input.MinStaff, input.MaxStaff, input.DaysOfWeek, input.ValidFrom, input.ValidTo); err != nil {
		return sqlc.CoverageRequirement{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.CoverageRequirement{}, apperrors.Internal(err)
	}
	return s.repo.createCoverage(ctx, sqlc.CreateCoverageRequirementParams{
		ID: id, ShipID: shipID, Department: input.Department, ShiftTemplateID: input.ShiftTemplateID,
		PositionCode: strings.TrimSpace(input.PositionCode), DeckID: optionalUUID(input.DeckID), AreaID: optionalUUID(input.AreaID),
		MinStaff: input.MinStaff, MaxStaff: input.MaxStaff, DaysOfWeek: input.DaysOfWeek,
		ValidFrom: input.ValidFrom, ValidTo: input.ValidTo,
	})
}

func (s *service) GetCoverage(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.CoverageRequirement, error) {
	row, err := s.repo.getCoverage(ctx, id)
	if err != nil {
		return sqlc.CoverageRequirement{}, err
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		return sqlc.CoverageRequirement{}, err
	}
	return row, nil
}

func (s *service) ListCoverage(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[sqlc.CoverageRequirement], error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return Page[sqlc.CoverageRequirement]{}, err
	}
	parts := []string{"coverage_requirements", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[sqlc.CoverageRequirement]{}, err
	}
	rows, err := s.repo.listCoverage(ctx, sqlc.ListCoverageRequirementsParams{ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[sqlc.CoverageRequirement]{}, err
	}
	page := Page[sqlc.CoverageRequirement]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateCoverage(ctx context.Context, p auth.Principal, id uuid.UUID, input CoverageUpdate) (sqlc.CoverageRequirement, error) {
	current, err := s.GetCoverage(ctx, p, id)
	if err != nil {
		return sqlc.CoverageRequirement{}, err
	}
	if input.Department != nil {
		current.Department = *input.Department
	}
	if input.ShiftTemplateID != nil {
		template, err := s.repo.getShiftTemplate(ctx, *input.ShiftTemplateID)
		if err != nil || template.ShipID != current.ShipID {
			return sqlc.CoverageRequirement{}, apperrors.Validation(apperrors.CodeInvalidReference, "Shift template does not exist on this ship.")
		}
		current.ShiftTemplateID = *input.ShiftTemplateID
	}
	if input.PositionCode != nil {
		current.PositionCode = strings.TrimSpace(*input.PositionCode)
	}
	if input.DeckID != nil {
		current.DeckID = optionalUUID(input.DeckID)
	}
	if input.AreaID != nil {
		current.AreaID = optionalUUID(input.AreaID)
	}
	if input.MinStaff != nil {
		current.MinStaff = *input.MinStaff
	}
	if input.MaxStaff != nil {
		current.MaxStaff = input.MaxStaff
	}
	if input.DaysOfWeek != nil {
		current.DaysOfWeek = input.DaysOfWeek
	}
	if input.ValidFrom != nil {
		current.ValidFrom = *input.ValidFrom
	}
	if input.ValidTo != nil {
		current.ValidTo = *input.ValidTo
	}
	if err := validateCoverage(current.Department, current.MinStaff, current.MaxStaff, current.DaysOfWeek, current.ValidFrom, current.ValidTo); err != nil {
		return sqlc.CoverageRequirement{}, err
	}
	return s.repo.updateCoverage(ctx, sqlc.UpdateCoverageRequirementParams{
		ID: id, Department: current.Department, ShiftTemplateID: current.ShiftTemplateID, PositionCode: current.PositionCode,
		DeckID: current.DeckID, AreaID: current.AreaID, MinStaff: current.MinStaff, MaxStaff: current.MaxStaff,
		DaysOfWeek: current.DaysOfWeek, ValidFrom: current.ValidFrom, ValidTo: current.ValidTo,
	})
}

func (s *service) DeleteCoverage(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetCoverage(ctx, p, id); err != nil {
		return err
	}
	return s.repo.deleteCoverage(ctx, id)
}

func (s *service) CreateLaborRule(ctx context.Context, p auth.Principal, input LaborRuleInput) (sqlc.LaborRule, error) {
	if err := s.requireTenant(p); err != nil {
		return sqlc.LaborRule{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			return sqlc.LaborRule{}, err
		}
	}
	params, err := objectJSON(input.Params)
	if err != nil {
		return sqlc.LaborRule{}, err
	}
	if strings.TrimSpace(input.RuleCode) == "" || !severities[input.Severity] {
		return sqlc.LaborRule{}, validation("Rule code and a recognised severity are required.")
	}
	if err := validateDateRange(input.ValidFrom, input.ValidTo); err != nil {
		return sqlc.LaborRule{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.LaborRule{}, apperrors.Internal(err)
	}
	return s.repo.createLaborRule(ctx, sqlc.CreateLaborRuleParams{
		ID: id, TenantID: p.TenantID, ShipID: optionalUUID(input.ShipID), RuleCode: strings.TrimSpace(input.RuleCode),
		Severity: input.Severity, Params: params, ValidFrom: input.ValidFrom, ValidTo: input.ValidTo,
	})
}

func (s *service) GetLaborRule(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.LaborRule, error) {
	row, err := s.repo.getLaborRule(ctx, id)
	if err != nil {
		return sqlc.LaborRule{}, err
	}
	if !p.CanAccessTenant(row.TenantID) {
		return sqlc.LaborRule{}, apperrors.Forbidden("Labor rule access is not permitted.")
	}
	if row.ShipID.Valid && !p.CanAccessShip(row.ShipID.UUID) {
		return sqlc.LaborRule{}, apperrors.Forbidden("Labor rule access is not permitted.")
	}
	return row, nil
}

func (s *service) ListLaborRules(ctx context.Context, p auth.Principal, shipID *uuid.UUID, rawCursor string, limit int32) (Page[sqlc.LaborRule], error) {
	if shipID != nil {
		if err := s.requireShip(ctx, p, *shipID); err != nil {
			return Page[sqlc.LaborRule]{}, err
		}
	}
	parts := []string{"labor_rules", p.TenantID.String(), optionalUUIDString(shipID)}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[sqlc.LaborRule]{}, err
	}
	rows, err := s.repo.listLaborRules(ctx, sqlc.ListLaborRulesParams{
		TenantID: nullUUID(p.TenantID), ShipID: optionalUUID(shipID), After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[sqlc.LaborRule]{}, err
	}
	page := Page[sqlc.LaborRule]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateLaborRule(ctx context.Context, p auth.Principal, id uuid.UUID, input LaborRuleUpdate) (sqlc.LaborRule, error) {
	current, err := s.GetLaborRule(ctx, p, id)
	if err != nil {
		return sqlc.LaborRule{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			return sqlc.LaborRule{}, err
		}
		current.ShipID = optionalUUID(input.ShipID)
	}
	if input.RuleCode != nil {
		current.RuleCode = strings.TrimSpace(*input.RuleCode)
	}
	if input.Severity != nil {
		if !severities[*input.Severity] {
			return sqlc.LaborRule{}, validation("Severity is not recognised.")
		}
		current.Severity = *input.Severity
	}
	if input.Params != nil {
		params, err := objectJSON(input.Params)
		if err != nil {
			return sqlc.LaborRule{}, err
		}
		current.Params = params
	}
	if input.ValidFrom != nil {
		current.ValidFrom = *input.ValidFrom
	}
	if input.ValidTo != nil {
		current.ValidTo = *input.ValidTo
	}
	if current.RuleCode == "" {
		return sqlc.LaborRule{}, validation("Rule code is required.")
	}
	if err := validateDateRange(current.ValidFrom, current.ValidTo); err != nil {
		return sqlc.LaborRule{}, err
	}
	return s.repo.updateLaborRule(ctx, sqlc.UpdateLaborRuleParams{
		ID: id, ShipID: current.ShipID, RuleCode: current.RuleCode, Severity: current.Severity,
		Params: current.Params, ValidFrom: current.ValidFrom, ValidTo: current.ValidTo,
	})
}

func (s *service) DeleteLaborRule(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetLaborRule(ctx, p, id); err != nil {
		return err
	}
	return s.repo.deleteLaborRule(ctx, id)
}

func (s *service) CreateAutonomy(ctx context.Context, p auth.Principal, input AutonomyInput) (sqlc.AutonomyPolicy, error) {
	if err := s.requireTenant(p); err != nil {
		return sqlc.AutonomyPolicy{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			return sqlc.AutonomyPolicy{}, err
		}
	}
	conditions, err := objectJSON(input.Conditions)
	if err != nil {
		return sqlc.AutonomyPolicy{}, err
	}
	if strings.TrimSpace(input.ChangeType) == "" || !autonomyLevels[input.Level] {
		return sqlc.AutonomyPolicy{}, validation("Change type and a recognised autonomy level are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sqlc.AutonomyPolicy{}, apperrors.Internal(err)
	}
	return s.repo.createAutonomy(ctx, sqlc.CreateAutonomyPolicyParams{
		ID: id, TenantID: p.TenantID, ShipID: optionalUUID(input.ShipID), ChangeType: strings.TrimSpace(input.ChangeType),
		Level: input.Level, Conditions: conditions,
	})
}

func (s *service) GetAutonomy(ctx context.Context, p auth.Principal, id uuid.UUID) (sqlc.AutonomyPolicy, error) {
	row, err := s.repo.getAutonomy(ctx, id)
	if err != nil {
		return sqlc.AutonomyPolicy{}, err
	}
	if !p.CanAccessTenant(row.TenantID) || (row.ShipID.Valid && !p.CanAccessShip(row.ShipID.UUID)) {
		return sqlc.AutonomyPolicy{}, apperrors.Forbidden("Autonomy policy access is not permitted.")
	}
	return row, nil
}

func (s *service) ListAutonomy(ctx context.Context, p auth.Principal, shipID *uuid.UUID, rawCursor string, limit int32) (Page[sqlc.AutonomyPolicy], error) {
	if shipID != nil {
		if err := s.requireShip(ctx, p, *shipID); err != nil {
			return Page[sqlc.AutonomyPolicy]{}, err
		}
	}
	parts := []string{"autonomy_policies", p.TenantID.String(), optionalUUIDString(shipID)}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[sqlc.AutonomyPolicy]{}, err
	}
	rows, err := s.repo.listAutonomy(ctx, sqlc.ListAutonomyPoliciesParams{
		TenantID: nullUUID(p.TenantID), ShipID: optionalUUID(shipID), After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[sqlc.AutonomyPolicy]{}, err
	}
	page := Page[sqlc.AutonomyPolicy]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateAutonomy(ctx context.Context, p auth.Principal, id uuid.UUID, input AutonomyUpdate) (sqlc.AutonomyPolicy, error) {
	current, err := s.GetAutonomy(ctx, p, id)
	if err != nil {
		return sqlc.AutonomyPolicy{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			return sqlc.AutonomyPolicy{}, err
		}
		current.ShipID = optionalUUID(input.ShipID)
	}
	if input.ChangeType != nil {
		current.ChangeType = strings.TrimSpace(*input.ChangeType)
	}
	if input.Level != nil {
		if !autonomyLevels[*input.Level] {
			return sqlc.AutonomyPolicy{}, validation("Autonomy level is not recognised.")
		}
		current.Level = *input.Level
	}
	if input.Conditions != nil {
		conditions, err := objectJSON(input.Conditions)
		if err != nil {
			return sqlc.AutonomyPolicy{}, err
		}
		current.Conditions = conditions
	}
	if current.ChangeType == "" {
		return sqlc.AutonomyPolicy{}, validation("Change type is required.")
	}
	return s.repo.updateAutonomy(ctx, sqlc.UpdateAutonomyPolicyParams{
		ID: id, ShipID: current.ShipID, ChangeType: current.ChangeType, Level: current.Level, Conditions: current.Conditions,
	})
}

func (s *service) DeleteAutonomy(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetAutonomy(ctx, p, id); err != nil {
		return err
	}
	return s.repo.deleteAutonomy(ctx, id)
}

func (s *service) requireTenant(p auth.Principal) error {
	if p.TenantID == uuid.Nil {
		return apperrors.Validation(apperrors.CodeValidationFailed, "A tenant is required.")
	}
	return nil
}

func (s *service) requireShip(ctx context.Context, p auth.Principal, shipID uuid.UUID) error {
	tenantID, err := s.ships.TenantID(ctx, shipID)
	if err != nil {
		return err
	}
	if !p.CanAccessTenant(tenantID) || !p.CanAccessShip(shipID) {
		return apperrors.Forbidden("Staff resource access is not permitted.")
	}
	return nil
}

func (s *service) checkStaffScope(ctx context.Context, p auth.Principal, row sqlc.Staff) error {
	if !p.CanAccessTenant(row.TenantID) {
		return apperrors.Forbidden("Staff access is not permitted.")
	}
	if row.ShipID.Valid {
		return s.requireShip(ctx, p, row.ShipID.UUID)
	}
	return nil
}

func validateCoverage(department string, minStaff int16, maxStaff *int16, days []int16, from, to pgtype.Date) error {
	if !departments[department] || minStaff < 0 {
		return validation("Department must be recognised and minimum staff must not be negative.")
	}
	if maxStaff != nil && *maxStaff < minStaff {
		return validation("Maximum staff must be greater than or equal to minimum staff.")
	}
	for _, day := range days {
		if day < 1 || day > 7 {
			return validation("Days of week must be between 1 and 7.")
		}
	}
	return validateDateRange(from, to)
}

func validateDateRange(from, to pgtype.Date) error {
	if from.Valid && to.Valid && to.Time.Before(from.Time) {
		return validation("Valid-to date must not be before valid-from.")
	}
	return nil
}

func certificationsJSON(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("[]"), nil
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, validation("Certifications must be a JSON array.")
	}
	return raw, nil
}

func objectJSON(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("{}"), nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, validation("Value must be a JSON object.")
	}
	return raw, nil
}

func optionalUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil || *id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func optionalUUIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func emptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func trimPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func validation(message string) error {
	return apperrors.Validation(apperrors.CodeValidationFailed, message)
}

func execResult(n int64, err error) error {
	if err != nil {
		return database.MapPgError(err)
	}
	if n == 0 {
		return database.MapPgError(pgx.ErrNoRows)
	}
	return nil
}

var _ = time.Now
