package ownership

// Module identifies a business module that owns a subset of the database tables.
type Module string

const (
	ModuleTenant   Module = "tenant"
	ModuleAuth     Module = "auth"
	ModuleShip     Module = "ship"
	ModuleVoyage   Module = "voyage"
	ModuleStaff    Module = "staff"
	ModulePartner  Module = "partner"
	ModuleBooking  Module = "booking"
	ModuleGuest    Module = "guest"
	ModuleSync     Module = "sync"
	ModuleAudit    Module = "audit"
	ModuleAI       Module = "ai"
	ModuleFeedback Module = "feedback"
)

// Modules lists every business module in a stable order.
var Modules = []Module{
	ModuleTenant,
	ModuleAuth,
	ModuleShip,
	ModuleVoyage,
	ModuleStaff,
	ModulePartner,
	ModuleBooking,
	ModuleGuest,
	ModuleSync,
	ModuleAudit,
	ModuleAI,
	ModuleFeedback,
}

// TableOwner assigns every table in the authoritative schema to exactly one module.
var TableOwner = map[string]Module{
	"tenants":               ModuleTenant,
	"admin_users":           ModuleAuth,
	"admin_ship_access":     ModuleAuth,
	"accounts":              ModuleAuth,
	"account_identities":    ModuleAuth,
	"refresh_tokens":        ModuleAuth,
	"ships":                 ModuleShip,
	"decks":                 ModuleShip,
	"areas":                 ModuleShip,
	"rooms":                 ModuleShip,
	"ports":                 ModuleVoyage,
	"voyages":               ModuleVoyage,
	"port_calls":            ModuleVoyage,
	"staff":                 ModuleStaff,
	"staff_leaves":          ModuleStaff,
	"staff_credentials":     ModuleStaff,
	"shift_templates":       ModuleStaff,
	"coverage_requirements": ModuleStaff,
	"labor_rules":           ModuleStaff,
	"autonomy_policies":     ModuleStaff,
	"ai_params":             ModuleAI,
	"partners":              ModulePartner,
	"partner_contracts":     ModulePartner,
	"partner_staff":         ModulePartner,
	"partner_services":      ModulePartner,
	"partner_slots":         ModulePartner,
	"bookings":              ModuleBooking,
	"service_bookings":      ModuleBooking,
	"guests":                ModuleGuest,
	"boarding_passes":       ModuleGuest,
	"ship_seq":              ModuleSync,
	"change_log":            ModuleSync,
	"hub_devices":           ModuleSync,
	"ship_devices":          ModuleSync,
	"revocations":           ModuleSync,
	"snapshots":             ModuleSync,
	"job_runs":              ModuleSync,
	"ops_events":            ModuleSync,
	"audit_log":             ModuleAudit,
	"feedback_responses":    ModuleFeedback,
	"feedback_insights":     ModuleFeedback,
}
