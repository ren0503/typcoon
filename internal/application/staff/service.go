package staff

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/audit"
	"github.com/ren0503/tycoon/internal/application/auth"
	appship "github.com/ren0503/tycoon/internal/application/ship"
	"github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/platform/pagination"
)

var (
	departments    = map[string]bool{"security": true, "cleaning": true, "operator": true}
	staffStatuses  = map[string]bool{"active": true, "on_leave": true, "sick": true, "inactive": true}
	leaveTypes     = map[string]bool{"annual": true, "sick": true, "rest": true, "training": true, "other": true}
	severities     = map[string]bool{"hard": true, "soft": true}
	autonomyLevels = map[string]bool{"auto": true, "conditional": true, "approval": true, "never": true}
)

type service struct {
	repo   Repository
	ships  appship.Service
	logger *slog.Logger
}

// New builds the staff service.
func New(repo Repository, logger *slog.Logger, ships appship.Service) Service {
	return &service{repo: repo, ships: ships, logger: logger.With("module", "staff")}
}

func (s *service) CreateStaff(ctx context.Context, p auth.Principal, input StaffInput) (Staff, error) {
	if err := s.requireTenant(p); err != nil {
		s.logger.ErrorContext(ctx, "tenant access denied", "principal", p, "error", err)
		return Staff{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			s.logger.ErrorContext(ctx, "ship access denied", "principal", p, "ship_id", *input.ShipID, "error", err)
			return Staff{}, err
		}
	}
	code := strings.TrimSpace(input.EmployeeCode)
	name := strings.TrimSpace(input.FullName)
	if code == "" || name == "" || !departments[input.Department] {
		s.logger.ErrorContext(ctx, "validation failed", "principal", p, "input", input)
		return Staff{}, validation("Employee code, full name, and a recognised department are required.")
	}
	certifications, err := certificationsJSON(input.Certifications)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to parse certifications", "principal", p, "input", input, "error", err)
		return Staff{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate staff ID", "principal", p, "input", input, "error", err)
		return Staff{}, apperrors.Internal(err)
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to begin transaction", "principal", p, "input", input, "error", err)
		return Staff{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	row, err := tx.CreateStaff(ctx, CreateStaffParams{
		ID: id, TenantID: p.TenantID, ShipID: optionalUUID(input.ShipID), EmployeeCode: code, FullName: name,
		Department: input.Department, RoleTitle: trimPtr(input.RoleTitle), IsSupervisor: input.IsSupervisor,
		Languages: emptyStrings(input.Languages), Phone: trimPtr(input.Phone), Certifications: certifications,
		Status: "active", HiredOn: input.HiredOn,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create staff", "principal", p, "input", input, "error", err)
		return Staff{}, err
	}
	shipID := uuid.Nil
	if row.ShipID.Valid {
		shipID = row.ShipID.UUID
	}
	if err := s.appendAudit(ctx, tx, p, "staff.created", row.ID, row.TenantID, shipID, map[string]any{
		"employee_code": row.EmployeeCode, "department": row.Department, "status": row.Status,
	}); err != nil {
		s.logger.ErrorContext(ctx, "failed to append audit entry", "principal", p, "input", input, "error", err)
		return Staff{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		s.logger.ErrorContext(ctx, "failed to commit transaction", "principal", p, "input", input, "error", err)
		return Staff{}, err
	}
	return row, nil
}

func (s *service) GetStaff(ctx context.Context, p auth.Principal, id uuid.UUID) (Staff, error) {
	row, err := s.repo.GetStaff(ctx, id)
	if err != nil {
		return Staff{}, err
	}
	if err := s.checkStaffScope(ctx, p, row); err != nil {
		return Staff{}, err
	}
	return row, nil
}

func (s *service) ListStaff(ctx context.Context, p auth.Principal, shipID *uuid.UUID, rawCursor string, limit int32) (Page[Staff], error) {
	if shipID != nil {
		if err := s.requireShip(ctx, p, *shipID); err != nil {
			return Page[Staff]{}, err
		}
	}
	parts := []string{"staff", p.TenantID.String(), optionalUUIDString(shipID)}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[Staff]{}, err
	}
	rows, err := s.repo.ListStaff(ctx, ListStaffParams{
		TenantID: nullUUID(p.TenantID), ShipID: optionalUUID(shipID), After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[Staff]{}, err
	}
	page := Page[Staff]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateStaff(ctx context.Context, p auth.Principal, id uuid.UUID, input StaffUpdate) (Staff, error) {
	current, err := s.GetStaff(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "staff not found or access denied", "principal", p, "error", err)
		return Staff{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			s.logger.ErrorContext(ctx, "ship not found or access denied", "principal", p, "ship_id", *input.ShipID, "error", err)
			return Staff{}, err
		}
		current.ShipID = optionalUUID(input.ShipID)
	}
	if input.EmployeeCode != nil {
		if strings.TrimSpace(*input.EmployeeCode) == "" {
			s.logger.ErrorContext(ctx, "validation failed: employee code must not be empty", "principal", p, "input", input)
			return Staff{}, validation("Employee code must not be empty.")
		}
		current.EmployeeCode = strings.TrimSpace(*input.EmployeeCode)
	}
	if input.FullName != nil {
		if strings.TrimSpace(*input.FullName) == "" {
			s.logger.ErrorContext(ctx, "validation failed: full name must not be empty", "principal", p, "input", input)
			return Staff{}, validation("Full name must not be empty.")
		}
		current.FullName = strings.TrimSpace(*input.FullName)
	}
	if input.Department != nil {
		if !departments[*input.Department] {
			s.logger.ErrorContext(ctx, "validation failed: unrecognized department", "principal", p, "input", input)
			return Staff{}, validation("Department is not recognised.")
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
			s.logger.ErrorContext(ctx, "failed to parse certifications", "principal", p, "input", input, "error", err)
			return Staff{}, err
		}
		current.Certifications = certifications
	}
	if input.Status != nil {
		if !staffStatuses[*input.Status] {
			s.logger.ErrorContext(ctx, "validation failed: unrecognized staff status", "principal", p, "input", input)
			return Staff{}, validation("Staff status is not recognised.")
		}
		current.Status = *input.Status
	}
	if input.HiredOn != nil {
		current.HiredOn = input.HiredOn
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to begin transaction", "principal", p, "input", input, "error", err)
		return Staff{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	row, err := tx.UpdateStaff(ctx, UpdateStaffParams{
		ID: id, ShipID: current.ShipID, EmployeeCode: current.EmployeeCode, FullName: current.FullName,
		Department: current.Department, RoleTitle: current.RoleTitle, IsSupervisor: current.IsSupervisor,
		Languages: current.Languages, Phone: current.Phone, Certifications: current.Certifications,
		Status: current.Status, HiredOn: current.HiredOn,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update staff", "principal", p, "input", input, "error", err)
		return Staff{}, err
	}
	shipID := uuid.Nil
	if row.ShipID.Valid {
		shipID = row.ShipID.UUID
	}
	if err := s.appendAudit(ctx, tx, p, "staff.updated", row.ID, row.TenantID, shipID, map[string]any{
		"employee_code": row.EmployeeCode, "department": row.Department, "status": row.Status,
	}); err != nil {
		s.logger.ErrorContext(ctx, "failed to append audit entry", "principal", p, "input", input, "error", err)
		return Staff{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		s.logger.ErrorContext(ctx, "failed to commit transaction", "principal", p, "input", input, "error", err)
		return Staff{}, err
	}
	return row, nil
}

func (s *service) DeleteStaff(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	current, err := s.GetStaff(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "staff not found or access denied", "principal", p, "error", err)
		return err
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to begin transaction", "principal", p, "staff_id", id, "error", err)
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := tx.SoftDeleteStaff(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete staff", "principal", p, "staff_id", id, "error", err)
		return err
	}
	shipID := uuid.Nil
	if current.ShipID.Valid {
		shipID = current.ShipID.UUID
	}
	if err := s.appendAudit(ctx, tx, p, "staff.deleted", id, current.TenantID, shipID, nil); err != nil {
		s.logger.ErrorContext(ctx, "failed to append audit entry", "principal", p, "staff_id", id, "error", err)
		return err
	}
	return tx.Commit(ctx)
}

func (s *service) appendAudit(ctx context.Context, tx Tx, p auth.Principal, action string, entityID, tenantID, shipID uuid.UUID, details map[string]any) error {
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
	return tx.AppendAudit(ctx, entry)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (s *service) StaffExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.repo.StaffExists(ctx, id)
}

func (s *service) CreateLeave(ctx context.Context, p auth.Principal, staffID uuid.UUID, input LeaveInput) (StaffLeafe, error) {
	if _, err := s.GetStaff(ctx, p, staffID); err != nil {
		s.logger.ErrorContext(ctx, "staff not found or access denied", "principal", p, "staff_id", staffID, "error", err)
		return StaffLeafe{}, err
	}
	if !leaveTypes[input.LeaveType] || !input.EndsAt.After(input.StartsAt) {
		s.logger.ErrorContext(ctx, "validation failed: unrecognized leave type or end date not after start date", "principal", p, "staff_id", staffID, "input", input)
		return StaffLeafe{}, validation("Leave type must be recognised and end after start.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate leave ID", "principal", p, "staff_id", staffID, "input", input, "error", err)
		return StaffLeafe{}, apperrors.Internal(err)
	}
	leave, err := s.repo.CreateStaffLeave(ctx, CreateStaffLeaveParams{
		ID: id, StaffID: staffID, LeaveType: input.LeaveType, StartsAt: input.StartsAt, EndsAt: input.EndsAt,
		Note: trimPtr(input.Note),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create leave", "principal", p, "staff_id", staffID, "input", input, "error", err)
		return StaffLeafe{}, apperrors.Internal(err)
	}
	return leave, nil
}

func (s *service) GetLeave(ctx context.Context, p auth.Principal, id uuid.UUID) (StaffLeafe, error) {
	row, err := s.repo.GetStaffLeave(ctx, id)
	if err != nil {
		return StaffLeafe{}, err
	}
	if _, err := s.GetStaff(ctx, p, row.StaffID); err != nil {
		return StaffLeafe{}, err
	}
	return row, nil
}

func (s *service) ListLeaves(ctx context.Context, p auth.Principal, staffID uuid.UUID, rawCursor string, limit int32) (Page[StaffLeafe], error) {
	if _, err := s.GetStaff(ctx, p, staffID); err != nil {
		return Page[StaffLeafe]{}, err
	}
	parts := []string{"staff_leaves", staffID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[StaffLeafe]{}, err
	}
	rows, err := s.repo.ListStaffLeaves(ctx, ListStaffLeavesParams{StaffID: staffID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[StaffLeafe]{}, err
	}
	page := Page[StaffLeafe]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateLeave(ctx context.Context, p auth.Principal, id uuid.UUID, input LeaveUpdate) (StaffLeafe, error) {
	current, err := s.GetLeave(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "leave not found or access denied", "principal", p, "leave_id", id, "error", err)
		return StaffLeafe{}, err
	}
	if input.LeaveType != nil {
		if !leaveTypes[*input.LeaveType] {
			s.logger.ErrorContext(ctx, "validation failed: unrecognized leave type", "principal", p, "leave_id", id, "input", input)
			return StaffLeafe{}, validation("Leave type is not recognised.")
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
		s.logger.ErrorContext(ctx, "validation failed: leave end date not after start date", "principal", p, "leave_id", id, "input", input)
		return StaffLeafe{}, validation("Leave end must be after start.")
	}
	leave, err := s.repo.UpdateStaffLeave(ctx, UpdateStaffLeaveParams{
		ID: id, LeaveType: current.LeaveType, StartsAt: current.StartsAt, EndsAt: current.EndsAt, Note: current.Note,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update leave", "principal", p, "leave_id", id, "input", input, "error", err)
		return StaffLeafe{}, apperrors.Internal(err)
	}
	return leave, nil
}

func (s *service) DeleteLeave(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetLeave(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "leave not found or access denied", "principal", p, "leave_id", id, "error", err)
		return err
	}
	return s.repo.DeleteStaffLeave(ctx, id)
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
	row, err := s.repo.UpsertStaffCredential(ctx, UpsertStaffCredentialParams{ID: id, StaffID: staffID, PinHash: hash})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to upsert credential", "principal", p, "staff_id", staffID, "pin_hash", hash, "error", err)
		return CredentialMeta{}, apperrors.Internal(err)
	}
	return CredentialMeta{StaffID: row.StaffID, PinUpdatedAt: row.PinUpdatedAt}, nil
}

func (s *service) CredentialMeta(ctx context.Context, p auth.Principal, staffID uuid.UUID) (CredentialMeta, error) {
	if _, err := s.GetStaff(ctx, p, staffID); err != nil {
		s.logger.ErrorContext(ctx, "staff not found or access denied", "principal", p, "staff_id", staffID, "error", err)
		return CredentialMeta{}, apperrors.Internal(err)
	}
	row, err := s.repo.GetStaffCredential(ctx, staffID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get credential", "principal", p, "staff_id", staffID, "error", err)
		return CredentialMeta{}, apperrors.Internal(err)
	}
	return CredentialMeta{StaffID: row.StaffID, PinUpdatedAt: row.PinUpdatedAt}, nil
}

func (s *service) CreateShiftTemplate(ctx context.Context, p auth.Principal, shipID uuid.UUID, input ShiftTemplateInput) (ShiftTemplate, error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		s.logger.ErrorContext(ctx, "ship access denied", "principal", p, "ship_id", shipID, "error", err)
		return ShiftTemplate{}, err
	}
	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
	if code == "" || name == "" || !departments[input.Department] || !input.StartTime.Valid || !input.EndTime.Valid {
		s.logger.ErrorContext(ctx, "validation failed for shift template", "principal", p, "ship_id", shipID, "input", input)
		return ShiftTemplate{}, validation("Shift template code, name, department, and times are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate shift template ID", "principal", p, "ship_id", shipID, "input", input, "error", err)
		return ShiftTemplate{}, apperrors.Internal(err)
	}
	shiftTemplate, err := s.repo.CreateShiftTemplate(ctx, CreateShiftTemplateParams{
		ID: id, ShipID: shipID, Department: input.Department, Code: code, Name: name,
		StartTime: input.StartTime, EndTime: input.EndTime,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create shift template", "principal", p, "ship_id", shipID, "input", input, "error", err)
		return ShiftTemplate{}, apperrors.Internal(err)
	}
	return shiftTemplate, nil
}

func (s *service) GetShiftTemplate(ctx context.Context, p auth.Principal, id uuid.UUID) (ShiftTemplate, error) {
	row, err := s.repo.GetShiftTemplate(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get shift template", "principal", p, "shift_template_id", id, "error", err)
		return ShiftTemplate{}, apperrors.Internal(err)
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		s.logger.ErrorContext(ctx, "ship access denied", "principal", p, "shift_template_id", id, "ship_id", row.ShipID, "error", err)
		return ShiftTemplate{}, apperrors.Internal(err)
	}
	return row, nil
}

func (s *service) ListShiftTemplates(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[ShiftTemplate], error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return Page[ShiftTemplate]{}, err
	}
	parts := []string{"shift_templates", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[ShiftTemplate]{}, err
	}
	rows, err := s.repo.ListShiftTemplates(ctx, ListShiftTemplatesParams{ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[ShiftTemplate]{}, err
	}
	page := Page[ShiftTemplate]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateShiftTemplate(ctx context.Context, p auth.Principal, id uuid.UUID, input ShiftTemplateUpdate) (ShiftTemplate, error) {
	current, err := s.GetShiftTemplate(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "shift template not found or access denied", "principal", p, "shift_template_id", id, "error", err)
		return ShiftTemplate{}, err
	}
	if input.Department != nil {
		if !departments[*input.Department] {
			s.logger.ErrorContext(ctx, "validation failed: unrecognized department", "principal", p, "shift_template_id", id, "input", input)
			return ShiftTemplate{}, validation("Department is not recognised.")
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
		s.logger.ErrorContext(ctx, "validation failed: shift template code, name, and times are required", "principal", p, "shift_template_id", id, "input", input)
		return ShiftTemplate{}, validation("Shift template code, name, and times are required.")
	}
	shiftTemplate, err := s.repo.UpdateShiftTemplate(ctx, UpdateShiftTemplateParams{
		ID: id, Department: current.Department, Code: current.Code, Name: current.Name,
		StartTime: current.StartTime, EndTime: current.EndTime,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update shift template", "principal", p, "shift_template_id", id, "error", err)
		return ShiftTemplate{}, apperrors.Internal(err)
	}
	return shiftTemplate, nil
}

func (s *service) DeleteShiftTemplate(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetShiftTemplate(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to get shift template", "principal", p, "shift_template_id", id, "error", err)
		return apperrors.Internal(err)
	}
	return s.repo.DeleteShiftTemplate(ctx, id)
}

func (s *service) CreateCoverage(ctx context.Context, p auth.Principal, shipID uuid.UUID, input CoverageInput) (CoverageRequirement, error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		s.logger.ErrorContext(ctx, "ship access denied", "principal", p, "ship_id", shipID, "error", err)
		return CoverageRequirement{}, apperrors.Internal(err)
	}
	template, err := s.repo.GetShiftTemplate(ctx, input.ShiftTemplateID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get shift template", "principal", p, "ship_id", shipID, "shift_template_id", input.ShiftTemplateID, "error", err)
		return CoverageRequirement{}, apperrors.Validation(apperrors.CodeInvalidReference, "Shift template does not exist on this ship.")
	}
	if template.ShipID != shipID {
		s.logger.ErrorContext(ctx, "shift template does not belong to ship", "principal", p, "ship_id", shipID, "shift_template_id", input.ShiftTemplateID)
		return CoverageRequirement{}, apperrors.Validation(apperrors.CodeInvalidReference, "Shift template does not exist on this ship.")
	}
	if err := validateCoverage(input.Department, input.MinStaff, input.MaxStaff, input.DaysOfWeek, input.ValidFrom, input.ValidTo); err != nil {
		s.logger.ErrorContext(ctx, "validation failed for coverage requirement", "principal", p, "ship_id", shipID, "input", input, "error", err)
		return CoverageRequirement{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate coverage requirement ID", "principal", p, "ship_id", shipID, "input", input, "error", err)
		return CoverageRequirement{}, apperrors.Internal(err)
	}
	coverage, err := s.repo.CreateCoverageRequirement(ctx, CreateCoverageRequirementParams{
		ID: id, ShipID: shipID, Department: input.Department, ShiftTemplateID: input.ShiftTemplateID,
		PositionCode: strings.TrimSpace(input.PositionCode), DeckID: optionalUUID(input.DeckID), AreaID: optionalUUID(input.AreaID),
		MinStaff: input.MinStaff, MaxStaff: input.MaxStaff, DaysOfWeek: input.DaysOfWeek,
		ValidFrom: input.ValidFrom, ValidTo: input.ValidTo,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create coverage requirement", "principal", p, "ship_id", shipID, "input", input, "error", err)
		return CoverageRequirement{}, apperrors.Internal(err)
	}
	return coverage, nil
}

func (s *service) GetCoverage(ctx context.Context, p auth.Principal, id uuid.UUID) (CoverageRequirement, error) {
	row, err := s.repo.GetCoverageRequirement(ctx, id)
	if err != nil {
		return CoverageRequirement{}, err
	}
	if err := s.requireShip(ctx, p, row.ShipID); err != nil {
		return CoverageRequirement{}, err
	}
	return row, nil
}

func (s *service) ListCoverage(ctx context.Context, p auth.Principal, shipID uuid.UUID, rawCursor string, limit int32) (Page[CoverageRequirement], error) {
	if err := s.requireShip(ctx, p, shipID); err != nil {
		return Page[CoverageRequirement]{}, err
	}
	parts := []string{"coverage_requirements", shipID.String()}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[CoverageRequirement]{}, err
	}
	rows, err := s.repo.ListCoverageRequirements(ctx, ListCoverageRequirementsParams{ShipID: shipID, After: nullUUID(cursor.ID), PageLimit: limit + 1})
	if err != nil {
		return Page[CoverageRequirement]{}, err
	}
	page := Page[CoverageRequirement]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateCoverage(ctx context.Context, p auth.Principal, id uuid.UUID, input CoverageUpdate) (CoverageRequirement, error) {
	current, err := s.GetCoverage(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get coverage requirement", "principal", p, "coverage_id", id, "error", err)
		return CoverageRequirement{}, apperrors.Internal(err)
	}
	if input.Department != nil {
		current.Department = *input.Department
	}
	if input.ShiftTemplateID != nil {
		template, err := s.repo.GetShiftTemplate(ctx, *input.ShiftTemplateID)
		if err != nil || template.ShipID != current.ShipID {
			s.logger.ErrorContext(ctx, "shift template does not exist on this ship", "principal", p, "coverage_id", id, "shift_template_id", *input.ShiftTemplateID, "error", err)
			return CoverageRequirement{}, apperrors.Validation(apperrors.CodeInvalidReference, "Shift template does not exist on this ship.")
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
		current.ValidFrom = input.ValidFrom
	}
	if input.ValidTo != nil {
		current.ValidTo = input.ValidTo
	}
	if err := validateCoverage(current.Department, current.MinStaff, current.MaxStaff, current.DaysOfWeek, current.ValidFrom, current.ValidTo); err != nil {
		s.logger.ErrorContext(ctx, "validation failed for coverage requirement", "principal", p, "coverage_id", id, "input", input, "error", err)
		return CoverageRequirement{}, err
	}
	coverage, err := s.repo.UpdateCoverageRequirement(ctx, UpdateCoverageRequirementParams{
		ID: id, Department: current.Department, ShiftTemplateID: current.ShiftTemplateID, PositionCode: current.PositionCode,
		DeckID: current.DeckID, AreaID: current.AreaID, MinStaff: current.MinStaff, MaxStaff: current.MaxStaff,
		DaysOfWeek: current.DaysOfWeek, ValidFrom: current.ValidFrom, ValidTo: current.ValidTo,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update coverage requirement", "principal", p, "coverage_id", id, "error", err)
		return CoverageRequirement{}, apperrors.Internal(err)
	}
	return coverage, nil
}

func (s *service) DeleteCoverage(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetCoverage(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to get coverage requirement", "principal", p, "coverage_id", id, "error", err)
		return err
	}
	return s.repo.DeleteCoverageRequirement(ctx, id)
}

func (s *service) CreateLaborRule(ctx context.Context, p auth.Principal, input LaborRuleInput) (LaborRule, error) {
	if err := s.requireTenant(p); err != nil {
		s.logger.ErrorContext(ctx, "tenant access denied", "principal", p, "error", err)
		return LaborRule{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			s.logger.ErrorContext(ctx, "ship access denied", "principal", p, "ship_id", *input.ShipID, "error", err)
			return LaborRule{}, err
		}
	}
	params, err := objectJSON(input.Params)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to parse labor rule params", "principal", p, "input", input, "error", err)
		return LaborRule{}, err
	}
	if strings.TrimSpace(input.RuleCode) == "" || !severities[input.Severity] {
		s.logger.ErrorContext(ctx, "invalid labor rule input", "principal", p, "input", input, "error", err)
		return LaborRule{}, validation("Rule code and a recognised severity are required.")
	}
	if err := validateDateRange(input.ValidFrom, input.ValidTo); err != nil {
		s.logger.ErrorContext(ctx, "invalid labor rule date range", "principal", p, "input", input, "error", err)
		return LaborRule{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate labor rule ID", "principal", p, "input", input, "error", err)
		return LaborRule{}, apperrors.Internal(err)
	}
	laborRule, err := s.repo.CreateLaborRule(ctx, CreateLaborRuleParams{
		ID: id, TenantID: p.TenantID, ShipID: optionalUUID(input.ShipID), RuleCode: strings.TrimSpace(input.RuleCode),
		Severity: input.Severity, Params: params, ValidFrom: input.ValidFrom, ValidTo: input.ValidTo,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create labor rule", "principal", p, "input", input, "error", err)
		return LaborRule{}, apperrors.Internal(err)
	}
	return laborRule, nil
}

func (s *service) GetLaborRule(ctx context.Context, p auth.Principal, id uuid.UUID) (LaborRule, error) {
	row, err := s.repo.GetLaborRule(ctx, id)
	if err != nil {
		return LaborRule{}, err
	}
	if !p.CanAccessTenant(row.TenantID) {
		return LaborRule{}, apperrors.Forbidden("Labor rule access is not permitted.")
	}
	if row.ShipID.Valid && !p.CanAccessShip(row.ShipID.UUID) {
		return LaborRule{}, apperrors.Forbidden("Labor rule access is not permitted.")
	}
	return row, nil
}

func (s *service) ListLaborRules(ctx context.Context, p auth.Principal, shipID *uuid.UUID, rawCursor string, limit int32) (Page[LaborRule], error) {
	if shipID != nil {
		if err := s.requireShip(ctx, p, *shipID); err != nil {
			return Page[LaborRule]{}, err
		}
	}
	parts := []string{"labor_rules", p.TenantID.String(), optionalUUIDString(shipID)}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[LaborRule]{}, err
	}
	rows, err := s.repo.ListLaborRules(ctx, ListLaborRulesParams{
		TenantID: nullUUID(p.TenantID), ShipID: optionalUUID(shipID), After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[LaborRule]{}, err
	}
	page := Page[LaborRule]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateLaborRule(ctx context.Context, p auth.Principal, id uuid.UUID, input LaborRuleUpdate) (LaborRule, error) {
	current, err := s.GetLaborRule(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get labor rule", "principal", p, "labor_rule_id", id, "error", err)
		return LaborRule{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			s.logger.ErrorContext(ctx, "ship access denied", "principal", p, "labor_rule_id", id, "ship_id", *input.ShipID, "error", err)
			return LaborRule{}, err
		}
		current.ShipID = optionalUUID(input.ShipID)
	}
	if input.RuleCode != nil {
		current.RuleCode = strings.TrimSpace(*input.RuleCode)
	}
	if input.Severity != nil {
		if !severities[*input.Severity] {
			s.logger.ErrorContext(ctx, "validation failed: unrecognized severity", "principal", p, "labor_rule_id", id, "input", input)
			return LaborRule{}, validation("Severity is not recognised.")
		}
		current.Severity = *input.Severity
	}
	if input.Params != nil {
		params, err := objectJSON(input.Params)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to parse labor rule params", "principal", p, "labor_rule_id", id, "input", input, "error", err)
			return LaborRule{}, err
		}
		current.Params = params
	}
	if input.ValidFrom != nil {
		current.ValidFrom = input.ValidFrom
	}
	if input.ValidTo != nil {
		current.ValidTo = input.ValidTo
	}
	if current.RuleCode == "" {
		s.logger.ErrorContext(ctx, "validation failed: labor rule code is required", "principal", p, "labor_rule_id", id, "input", input)
		return LaborRule{}, validation("Rule code is required.")
	}
	if err := validateDateRange(current.ValidFrom, current.ValidTo); err != nil {
		s.logger.ErrorContext(ctx, "invalid labor rule date range", "principal", p, "labor_rule_id", id, "input", input, "error", err)
		return LaborRule{}, err
	}
	laborRule, err := s.repo.UpdateLaborRule(ctx, UpdateLaborRuleParams{
		ID: id, ShipID: current.ShipID, RuleCode: current.RuleCode, Severity: current.Severity,
		Params: current.Params, ValidFrom: current.ValidFrom, ValidTo: current.ValidTo,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update labor rule", "principal", p, "labor_rule_id", id, "error", err)
		return LaborRule{}, apperrors.Internal(err)
	}
	return laborRule, nil
}

func (s *service) DeleteLaborRule(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetLaborRule(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to get labor rule", "principal", p, "labor_rule_id", id, "error", err)
		return err
	}
	return s.repo.DeleteLaborRule(ctx, id)
}

func (s *service) CreateAutonomy(ctx context.Context, p auth.Principal, input AutonomyInput) (AutonomyPolicy, error) {
	if err := s.requireTenant(p); err != nil {
		s.logger.ErrorContext(ctx, "tenant access denied", "principal", p, "error", err)
		return AutonomyPolicy{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			s.logger.ErrorContext(ctx, "ship access denied", "principal", p, "ship_id", *input.ShipID, "error", err)
			return AutonomyPolicy{}, err
		}
	}
	conditions, err := objectJSON(input.Conditions)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to parse autonomy policy conditions", "principal", p, "input", input, "error", err)
		return AutonomyPolicy{}, err
	}
	if strings.TrimSpace(input.ChangeType) == "" || !autonomyLevels[input.Level] {
		s.logger.ErrorContext(ctx, "invalid autonomy policy input", "principal", p, "input", input, "error", err)
		return AutonomyPolicy{}, validation("Change type and a recognised autonomy level are required.")
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to generate autonomy policy ID", "principal", p, "input", input, "error", err)
		return AutonomyPolicy{}, apperrors.Internal(err)
	}
	autonomyPolicy, err := s.repo.CreateAutonomyPolicy(ctx, CreateAutonomyPolicyParams{
		ID: id, TenantID: p.TenantID, ShipID: optionalUUID(input.ShipID), ChangeType: strings.TrimSpace(input.ChangeType),
		Level: input.Level, Conditions: conditions,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create autonomy policy", "principal", p, "input", input, "error", err)
		return AutonomyPolicy{}, apperrors.Internal(err)
	}
	return autonomyPolicy, nil
}

func (s *service) GetAutonomy(ctx context.Context, p auth.Principal, id uuid.UUID) (AutonomyPolicy, error) {
	row, err := s.repo.GetAutonomyPolicy(ctx, id)
	if err != nil {
		return AutonomyPolicy{}, err
	}
	if !p.CanAccessTenant(row.TenantID) || (row.ShipID.Valid && !p.CanAccessShip(row.ShipID.UUID)) {
		return AutonomyPolicy{}, apperrors.Forbidden("Autonomy policy access is not permitted.")
	}
	return row, nil
}

func (s *service) ListAutonomy(ctx context.Context, p auth.Principal, shipID *uuid.UUID, rawCursor string, limit int32) (Page[AutonomyPolicy], error) {
	if shipID != nil {
		if err := s.requireShip(ctx, p, *shipID); err != nil {
			return Page[AutonomyPolicy]{}, err
		}
	}
	parts := []string{"autonomy_policies", p.TenantID.String(), optionalUUIDString(shipID)}
	cursor, err := pagination.Decode(rawCursor, pagination.FilterHash(parts...))
	if err != nil {
		return Page[AutonomyPolicy]{}, err
	}
	rows, err := s.repo.ListAutonomyPolicies(ctx, ListAutonomyPoliciesParams{
		TenantID: nullUUID(p.TenantID), ShipID: optionalUUID(shipID), After: nullUUID(cursor.ID), PageLimit: limit + 1,
	})
	if err != nil {
		return Page[AutonomyPolicy]{}, err
	}
	page := Page[AutonomyPolicy]{Items: rows}
	if len(rows) > int(limit) {
		page.Items = rows[:limit]
		page.NextCursor = pagination.Cursor{Filter: pagination.FilterHash(parts...), ID: page.Items[len(page.Items)-1].ID}.Encode()
	}
	return page, nil
}

func (s *service) UpdateAutonomy(ctx context.Context, p auth.Principal, id uuid.UUID, input AutonomyUpdate) (AutonomyPolicy, error) {
	current, err := s.GetAutonomy(ctx, p, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get autonomy policy", "principal", p, "autonomy_id", id, "error", err)
		return AutonomyPolicy{}, err
	}
	if input.ShipID != nil {
		if err := s.requireShip(ctx, p, *input.ShipID); err != nil {
			s.logger.ErrorContext(ctx, "ship access denied", "principal", p, "autonomy_id", id, "ship_id", *input.ShipID, "error", err)
			return AutonomyPolicy{}, err
		}
		current.ShipID = optionalUUID(input.ShipID)
	}
	if input.ChangeType != nil {
		current.ChangeType = strings.TrimSpace(*input.ChangeType)
	}
	if input.Level != nil {
		if !autonomyLevels[*input.Level] {
			s.logger.ErrorContext(ctx, "validation failed: unrecognized autonomy level", "principal", p, "autonomy_id", id, "input", input)
			return AutonomyPolicy{}, validation("Autonomy level is not recognised.")
		}
		current.Level = *input.Level
	}
	if input.Conditions != nil {
		conditions, err := objectJSON(input.Conditions)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to parse autonomy policy conditions", "principal", p, "autonomy_id", id, "input", input, "error", err)
			return AutonomyPolicy{}, err
		}
		current.Conditions = conditions
	}
	if current.ChangeType == "" {
		s.logger.ErrorContext(ctx, "validation failed: change type is required", "principal", p, "autonomy_id", id, "input", input)
		return AutonomyPolicy{}, validation("Change type is required.")
	}
	autonomyPolicy, err := s.repo.UpdateAutonomyPolicy(ctx, UpdateAutonomyPolicyParams{
		ID: id, ShipID: current.ShipID, ChangeType: current.ChangeType, Level: current.Level, Conditions: current.Conditions,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update autonomy policy", "principal", p, "autonomy_id", id, "input", input, "error", err)
		return AutonomyPolicy{}, apperrors.Internal(err)
	}
	return autonomyPolicy, nil
}

func (s *service) DeleteAutonomy(ctx context.Context, p auth.Principal, id uuid.UUID) error {
	if _, err := s.GetAutonomy(ctx, p, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to get autonomy policy", "principal", p, "autonomy_id", id, "error", err)
		return err
	}
	return s.repo.DeleteAutonomyPolicy(ctx, id)
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

func (s *service) checkStaffScope(ctx context.Context, p auth.Principal, row Staff) error {
	if !p.CanAccessTenant(row.TenantID) {
		return apperrors.Forbidden("Staff access is not permitted.")
	}
	if row.ShipID.Valid {
		return s.requireShip(ctx, p, row.ShipID.UUID)
	}
	return nil
}

func validateCoverage(department string, minStaff int16, maxStaff *int16, days []int16, from, to *time.Time) error {
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

func validateDateRange(from, to *time.Time) error {
	if from != nil && to != nil && to.Before(*from) {
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

var _ = time.Now

func nullUUID(id uuid.UUID) uuid.NullUUID {
	if id == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}
