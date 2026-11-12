package staff

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/auth"
	appstaff "github.com/ren0503/tycoon/internal/application/staff"
	apperrors "github.com/ren0503/tycoon/internal/platform/errors"
	"github.com/ren0503/tycoon/internal/transport/http/httpx"
)

type staffCreateRequest struct {
	ShipID         *string  `json:"ship_id" validate:"omitempty,uuid"`
	EmployeeCode   string   `json:"employee_code" validate:"required,max=64"`
	FullName       string   `json:"full_name" validate:"required,max=255"`
	Department     string   `json:"department" validate:"required,oneof=security cleaning operator"`
	RoleTitle      *string  `json:"role_title" validate:"omitempty,max=255"`
	IsSupervisor   *bool    `json:"is_supervisor"`
	Languages      []string `json:"languages"`
	Phone          *string  `json:"phone" validate:"omitempty,max=32"`
	Certifications []byte   `json:"certifications"`
	HiredOn        *string  `json:"hired_on"`
}

type staffUpdateRequest struct {
	ShipID         *string  `json:"ship_id" validate:"omitempty,uuid"`
	EmployeeCode   *string  `json:"employee_code" validate:"omitempty,min=1,max=64"`
	FullName       *string  `json:"full_name" validate:"omitempty,min=1,max=255"`
	Department     *string  `json:"department" validate:"omitempty,oneof=security cleaning operator"`
	RoleTitle      *string  `json:"role_title" validate:"omitempty,max=255"`
	IsSupervisor   *bool    `json:"is_supervisor"`
	Languages      []string `json:"languages"`
	Phone          *string  `json:"phone" validate:"omitempty,max=32"`
	Certifications []byte   `json:"certifications"`
	Status         *string  `json:"status" validate:"omitempty,oneof=active on_leave sick inactive"`
	HiredOn        *string  `json:"hired_on"`
}

type leaveCreateRequest struct {
	LeaveType string    `json:"leave_type" validate:"required,oneof=annual sick rest training other"`
	StartsAt  time.Time `json:"starts_at" validate:"required"`
	EndsAt    time.Time `json:"ends_at" validate:"required"`
	Note      *string   `json:"note" validate:"omitempty,max=1024"`
}

type leaveUpdateRequest struct {
	LeaveType *string    `json:"leave_type" validate:"omitempty,oneof=annual sick rest training other"`
	StartsAt  *time.Time `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
	Note      *string    `json:"note" validate:"omitempty,max=1024"`
}

type credentialRequest struct {
	Pin string `json:"pin" validate:"required,min=4,max=12"`
}

type shiftCreateRequest struct {
	Department string `json:"department" validate:"required,oneof=security cleaning operator"`
	Code       string `json:"code" validate:"required,max=64"`
	Name       string `json:"name" validate:"required,max=255"`
	StartTime  string `json:"start_time" validate:"required"`
	EndTime    string `json:"end_time" validate:"required"`
}

type shiftUpdateRequest struct {
	Department *string `json:"department" validate:"omitempty,oneof=security cleaning operator"`
	Code       *string `json:"code" validate:"omitempty,min=1,max=64"`
	Name       *string `json:"name" validate:"omitempty,min=1,max=255"`
	StartTime  *string `json:"start_time"`
	EndTime    *string `json:"end_time"`
}

type coverageCreateRequest struct {
	Department      string  `json:"department" validate:"required,oneof=security cleaning operator"`
	ShiftTemplateID string  `json:"shift_template_id" validate:"required,uuid"`
	PositionCode    string  `json:"position_code" validate:"required,max=64"`
	DeckID          *string `json:"deck_id" validate:"omitempty,uuid"`
	AreaID          *string `json:"area_id" validate:"omitempty,uuid"`
	MinStaff        int16   `json:"min_staff" validate:"min=0"`
	MaxStaff        *int16  `json:"max_staff"`
	DaysOfWeek      []int16 `json:"days_of_week"`
	ValidFrom       *string `json:"valid_from"`
	ValidTo         *string `json:"valid_to"`
}

type coverageUpdateRequest struct {
	Department      *string `json:"department" validate:"omitempty,oneof=security cleaning operator"`
	ShiftTemplateID *string `json:"shift_template_id" validate:"omitempty,uuid"`
	PositionCode    *string `json:"position_code" validate:"omitempty,min=1,max=64"`
	DeckID          *string `json:"deck_id" validate:"omitempty,uuid"`
	AreaID          *string `json:"area_id" validate:"omitempty,uuid"`
	MinStaff        *int16  `json:"min_staff" validate:"omitempty,min=0"`
	MaxStaff        *int16  `json:"max_staff"`
	DaysOfWeek      []int16 `json:"days_of_week"`
	ValidFrom       *string `json:"valid_from"`
	ValidTo         *string `json:"valid_to"`
}

type laborRuleCreateRequest struct {
	ShipID    *string `json:"ship_id" validate:"omitempty,uuid"`
	RuleCode  string  `json:"rule_code" validate:"required,max=64"`
	Severity  string  `json:"severity" validate:"required,oneof=hard soft"`
	Params    []byte  `json:"params"`
	ValidFrom *string `json:"valid_from"`
	ValidTo   *string `json:"valid_to"`
}

type laborRuleUpdateRequest struct {
	ShipID    *string `json:"ship_id" validate:"omitempty,uuid"`
	RuleCode  *string `json:"rule_code" validate:"omitempty,min=1,max=64"`
	Severity  *string `json:"severity" validate:"omitempty,oneof=hard soft"`
	Params    []byte  `json:"params"`
	ValidFrom *string `json:"valid_from"`
	ValidTo   *string `json:"valid_to"`
}

type autonomyCreateRequest struct {
	ShipID     *string `json:"ship_id" validate:"omitempty,uuid"`
	ChangeType string  `json:"change_type" validate:"required,max=64"`
	Level      string  `json:"level" validate:"required,oneof=auto conditional approval never"`
	Conditions []byte  `json:"conditions"`
}

type autonomyUpdateRequest struct {
	ShipID     *string `json:"ship_id" validate:"omitempty,uuid"`
	ChangeType *string `json:"change_type" validate:"omitempty,min=1,max=64"`
	Level      *string `json:"level" validate:"omitempty,oneof=auto conditional approval never"`
	Conditions []byte  `json:"conditions"`
}

type staffResponse struct {
	appstaff.Staff
	HiredOn *string `json:"hired_on"`
}

type shiftResponse struct {
	appstaff.ShiftTemplate
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type coverageResponse struct {
	appstaff.CoverageRequirement
	ValidFrom *string `json:"valid_from"`
	ValidTo   *string `json:"valid_to"`
}

type laborRuleResponse struct {
	appstaff.LaborRule
	ValidFrom *string `json:"valid_from"`
	ValidTo   *string `json:"valid_to"`
}

// RegisterRoutes mounts staff routes on the router.
func RegisterRoutes(r chi.Router, svc appstaff.Service) {
	r.Route("/staff", func(r chi.Router) {
		r.Get("/", listStaff(svc))
		r.Post("/", createStaff(svc))
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getStaff(svc))
			r.Patch("/", updateStaff(svc))
			r.Delete("/", deleteStaff(svc))
		})
		r.Route("/{staffId}/leaves", func(r chi.Router) {
			r.Get("/", listLeaves(svc))
			r.Post("/", createLeave(svc))
		})
		r.Route("/{staffId}/credential", func(r chi.Router) {
			r.Get("/", getCredential(svc))
			r.Put("/", setCredential(svc))
		})
	})
	r.Route("/leaves/{id}", func(r chi.Router) {
		r.Get("/", getLeave(svc))
		r.Patch("/", updateLeave(svc))
		r.Delete("/", deleteLeave(svc))
	})
	r.Route("/ships/{shipId}/shift-templates", func(r chi.Router) {
		r.Get("/", listShiftTemplates(svc))
		r.Post("/", createShiftTemplate(svc))
	})
	r.Route("/shift-templates/{id}", func(r chi.Router) {
		r.Get("/", getShiftTemplate(svc))
		r.Patch("/", updateShiftTemplate(svc))
		r.Delete("/", deleteShiftTemplate(svc))
	})
	r.Route("/ships/{shipId}/coverage-requirements", func(r chi.Router) {
		r.Get("/", listCoverage(svc))
		r.Post("/", createCoverage(svc))
	})
	r.Route("/coverage-requirements/{id}", func(r chi.Router) {
		r.Get("/", getCoverage(svc))
		r.Patch("/", updateCoverage(svc))
		r.Delete("/", deleteCoverage(svc))
	})
	r.Route("/labor-rules", func(r chi.Router) {
		r.Get("/", listLaborRules(svc))
		r.Post("/", createLaborRule(svc))
	})
	r.Route("/labor-rules/{id}", func(r chi.Router) {
		r.Get("/", getLaborRule(svc))
		r.Patch("/", updateLaborRule(svc))
		r.Delete("/", deleteLaborRule(svc))
	})
	r.Route("/autonomy-policies", func(r chi.Router) {
		r.Get("/", listAutonomy(svc))
		r.Post("/", createAutonomy(svc))
	})
	r.Route("/autonomy-policies/{id}", func(r chi.Router) {
		r.Get("/", getAutonomy(svc))
		r.Patch("/", updateAutonomy(svc))
		r.Delete("/", deleteAutonomy(svc))
	})
}

func toStaffResponse(row appstaff.Staff) staffResponse {
	return staffResponse{Staff: row, HiredOn: dateString(row.HiredOn)}
}

func toShiftResponse(row appstaff.ShiftTemplate) shiftResponse {
	return shiftResponse{ShiftTemplate: row, StartTime: timeString(row.StartTime), EndTime: timeString(row.EndTime)}
}

func toCoverageResponse(row appstaff.CoverageRequirement) coverageResponse {
	return coverageResponse{CoverageRequirement: row, ValidFrom: dateString(row.ValidFrom), ValidTo: dateString(row.ValidTo)}
}

func toLaborRuleResponse(row appstaff.LaborRule) laborRuleResponse {
	return laborRuleResponse{LaborRule: row, ValidFrom: dateString(row.ValidFrom), ValidTo: dateString(row.ValidTo)}
}

func dateString(d *time.Time) *string {
	if d == nil {
		return nil
	}
	value := d.Format("2006-01-02")
	return &value
}

func timeString(t appstaff.TimeOfDay) string {
	if !t.Valid {
		return ""
	}
	micros := t.Microseconds
	seconds := micros / 1_000_000
	return formatClock(int(seconds/3600), int(seconds%3600/60), int(seconds%60))
}

func formatClock(hour, minute, second int) string {
	if second == 0 {
		return pad(hour) + ":" + pad(minute)
	}
	return pad(hour) + ":" + pad(minute) + ":" + pad(second)
}

func pad(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func parseDate(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", *raw)
	if err != nil {
		return nil, apperrors.Validation(apperrors.CodeValidationFailed, "Date must use the YYYY-MM-DD format.")
	}
	return &parsed, nil
}

func parseClock(raw string) (appstaff.TimeOfDay, error) {
	layouts := []string{"15:04", "15:04:05", "15:04:05.999999"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			micros := int64(parsed.Hour())*3600*1_000_000 + int64(parsed.Minute())*60*1_000_000 +
				int64(parsed.Second())*1_000_000 + int64(parsed.Nanosecond())/1000
			return appstaff.TimeOfDay{Microseconds: micros, Valid: true}, nil
		}
	}
	return appstaff.TimeOfDay{}, apperrors.Validation(apperrors.CodeValidationFailed, "Time must use the HH:MM or HH:MM:SS format.")
}

func listStaff(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.QueryUUID(r, "ship_id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListStaff(r.Context(), principal, shipID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		items := make([]staffResponse, len(page.Items))
		for i, row := range page.Items {
			items[i] = toStaffResponse(row)
		}
		httpx.Page(w, items, page.NextCursor)
	}
}

func createStaff(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		var req staffCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := parseUUIDPtr(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		hiredOn, err := parseDate(req.HiredOn)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		isSupervisor := false
		if req.IsSupervisor != nil {
			isSupervisor = *req.IsSupervisor
		}
		row, err := svc.CreateStaff(r.Context(), principal, appstaff.StaffInput{
			ShipID: shipID, EmployeeCode: req.EmployeeCode, FullName: req.FullName, Department: req.Department,
			RoleTitle: req.RoleTitle, IsSupervisor: isSupervisor, Languages: req.Languages, Phone: req.Phone,
			Certifications: req.Certifications, HiredOn: hiredOn,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, toStaffResponse(row))
	}
}

func getStaff(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetStaff(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, toStaffResponse(row))
	}
}

func updateStaff(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req staffUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := parseUUIDPtr(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		hiredOn, err := parseDate(req.HiredOn)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateStaff(r.Context(), principal, id, appstaff.StaffUpdate{
			ShipID: shipID, EmployeeCode: req.EmployeeCode, FullName: req.FullName, Department: req.Department,
			RoleTitle: req.RoleTitle, IsSupervisor: req.IsSupervisor, Languages: req.Languages, Phone: req.Phone,
			Certifications: req.Certifications, Status: req.Status, HiredOn: hiredOn,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, toStaffResponse(row))
	}
}

func deleteStaff(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteStaff(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listLeaves(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		staffID, err := httpx.PathID(r, "staffId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListLeaves(r.Context(), principal, staffID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createLeave(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		staffID, err := httpx.PathID(r, "staffId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req leaveCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateLeave(r.Context(), principal, staffID, appstaff.LeaveInput{
			LeaveType: req.LeaveType, StartsAt: req.StartsAt, EndsAt: req.EndsAt, Note: req.Note,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getLeave(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetLeave(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updateLeave(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req leaveUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateLeave(r.Context(), principal, id, appstaff.LeaveUpdate{
			LeaveType: req.LeaveType, StartsAt: req.StartsAt, EndsAt: req.EndsAt, Note: req.Note,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteLeave(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteLeave(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func setCredential(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		staffID, err := httpx.PathID(r, "staffId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req credentialRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		meta, err := svc.SetCredential(r.Context(), principal, staffID, req.Pin)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, meta)
	}
}

func getCredential(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		staffID, err := httpx.PathID(r, "staffId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		meta, err := svc.CredentialMeta(r.Context(), principal, staffID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, meta)
	}
}

func listShiftTemplates(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListShiftTemplates(r.Context(), principal, shipID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		items := make([]shiftResponse, len(page.Items))
		for i, row := range page.Items {
			items[i] = toShiftResponse(row)
		}
		httpx.Page(w, items, page.NextCursor)
	}
}

func createShiftTemplate(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req shiftCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		start, err := parseClock(req.StartTime)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		end, err := parseClock(req.EndTime)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateShiftTemplate(r.Context(), principal, shipID, appstaff.ShiftTemplateInput{
			Department: req.Department, Code: req.Code, Name: req.Name, StartTime: start, EndTime: end,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, toShiftResponse(row))
	}
}

func getShiftTemplate(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetShiftTemplate(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, toShiftResponse(row))
	}
}

func updateShiftTemplate(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req shiftUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		input := appstaff.ShiftTemplateUpdate{Department: req.Department, Code: req.Code, Name: req.Name}
		if req.StartTime != nil {
			start, err := parseClock(*req.StartTime)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.StartTime = &start
		}
		if req.EndTime != nil {
			end, err := parseClock(*req.EndTime)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			input.EndTime = &end
		}
		row, err := svc.UpdateShiftTemplate(r.Context(), principal, id, input)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, toShiftResponse(row))
	}
}

func deleteShiftTemplate(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteShiftTemplate(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listCoverage(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListCoverage(r.Context(), principal, shipID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		items := make([]coverageResponse, len(page.Items))
		for i, row := range page.Items {
			items[i] = toCoverageResponse(row)
		}
		httpx.Page(w, items, page.NextCursor)
	}
}

func createCoverage(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req coverageCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		templateID, err := httpx.ParseUUID(req.ShiftTemplateID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		deckID, err := parseUUIDPtr(req.DeckID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		areaID, err := parseUUIDPtr(req.AreaID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		from, err := parseDate(req.ValidFrom)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		to, err := parseDate(req.ValidTo)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateCoverage(r.Context(), principal, shipID, appstaff.CoverageInput{
			Department: req.Department, ShiftTemplateID: templateID, PositionCode: req.PositionCode,
			DeckID: deckID, AreaID: areaID, MinStaff: req.MinStaff, MaxStaff: req.MaxStaff,
			DaysOfWeek: req.DaysOfWeek, ValidFrom: from, ValidTo: to,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, toCoverageResponse(row))
	}
}

func getCoverage(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetCoverage(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, toCoverageResponse(row))
	}
}

func updateCoverage(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req coverageUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		templateID, err := parseUUIDPtr(req.ShiftTemplateID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		deckID, err := parseUUIDPtr(req.DeckID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		areaID, err := parseUUIDPtr(req.AreaID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		from, err := parseDate(req.ValidFrom)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		to, err := parseDate(req.ValidTo)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateCoverage(r.Context(), principal, id, appstaff.CoverageUpdate{
			Department: req.Department, ShiftTemplateID: templateID, PositionCode: req.PositionCode,
			DeckID: deckID, AreaID: areaID, MinStaff: req.MinStaff, MaxStaff: req.MaxStaff,
			DaysOfWeek: req.DaysOfWeek, ValidFrom: from, ValidTo: to,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, toCoverageResponse(row))
	}
}

func deleteCoverage(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteCoverage(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listLaborRules(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.QueryUUID(r, "ship_id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListLaborRules(r.Context(), principal, shipID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		items := make([]laborRuleResponse, len(page.Items))
		for i, row := range page.Items {
			items[i] = toLaborRuleResponse(row)
		}
		httpx.Page(w, items, page.NextCursor)
	}
}

func createLaborRule(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		var req laborRuleCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := parseUUIDPtr(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		from, err := parseDate(req.ValidFrom)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		to, err := parseDate(req.ValidTo)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateLaborRule(r.Context(), principal, appstaff.LaborRuleInput{
			ShipID: shipID, RuleCode: req.RuleCode, Severity: req.Severity, Params: req.Params,
			ValidFrom: from, ValidTo: to,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, toLaborRuleResponse(row))
	}
}

func getLaborRule(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetLaborRule(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, toLaborRuleResponse(row))
	}
}

func updateLaborRule(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req laborRuleUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := parseUUIDPtr(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		from, err := parseDate(req.ValidFrom)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		to, err := parseDate(req.ValidTo)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateLaborRule(r.Context(), principal, id, appstaff.LaborRuleUpdate{
			ShipID: shipID, RuleCode: req.RuleCode, Severity: req.Severity, Params: req.Params,
			ValidFrom: from, ValidTo: to,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, toLaborRuleResponse(row))
	}
}

func deleteLaborRule(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteLaborRule(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func listAutonomy(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.QueryUUID(r, "ship_id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		limit, err := httpx.ParseLimit(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		page, err := svc.ListAutonomy(r.Context(), principal, shipID, httpx.RawCursor(r), limit)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.Page(w, page.Items, page.NextCursor)
	}
}

func createAutonomy(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		var req autonomyCreateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := parseUUIDPtr(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.CreateAutonomy(r.Context(), principal, appstaff.AutonomyInput{
			ShipID: shipID, ChangeType: req.ChangeType, Level: req.Level, Conditions: req.Conditions,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, row)
	}
}

func getAutonomy(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetAutonomy(r.Context(), principal, id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func updateAutonomy(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req autonomyUpdateRequest
		if err := httpx.Decode(w, r, &req); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		shipID, err := parseUUIDPtr(req.ShipID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.UpdateAutonomy(r.Context(), principal, id, appstaff.AutonomyUpdate{
			ShipID: shipID, ChangeType: req.ChangeType, Level: req.Level, Conditions: req.Conditions,
		})
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func deleteAutonomy(svc appstaff.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		id, err := httpx.PathID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if err := svc.DeleteAutonomy(r.Context(), principal, id); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.NoContent(w)
	}
}

func parseUUIDPtr(raw *string) (*uuid.UUID, error) {
	if raw == nil {
		return nil, nil
	}
	parsed, err := httpx.ParseUUID(*raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
