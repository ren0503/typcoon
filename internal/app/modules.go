package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ren0503/tycoon/internal/ai"
	"github.com/ren0503/tycoon/internal/audit"
	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/booking"
	"github.com/ren0503/tycoon/internal/feedback"
	"github.com/ren0503/tycoon/internal/guest"
	"github.com/ren0503/tycoon/internal/partner"
	"github.com/ren0503/tycoon/internal/ship"
	"github.com/ren0503/tycoon/internal/staff"
	syncmodule "github.com/ren0503/tycoon/internal/sync"
	"github.com/ren0503/tycoon/internal/tenant"
	"github.com/ren0503/tycoon/internal/voyage"
)

// Modules holds every business module constructed behind its public interface.
type Modules struct {
	Tenant   tenant.TenantService
	Ship     ship.Service
	Voyage   voyage.Service
	Staff    staff.Service
	Partner  partner.Service
	Booking  booking.Service
	Guest    guest.Service
	Accounts auth.AccountService
	AIParams ai.AIParamService
	Feedback feedback.FeedbackService
	Sync     syncmodule.Service
}

// BuildModules constructs the acyclic module graph described in data-model.md.
func BuildModules(pool *pgxpool.Pool) Modules {
	auditStore := audit.NewStore(pool)
	shipSvc := ship.New(pool)
	voyageSvc := voyage.New(pool, shipSvc)
	staffSvc := staff.New(pool, shipSvc, auditStore)
	partnerSvc := partner.New(pool, shipSvc)
	accountSvc := auth.NewAccountService(pool)
	bookingSvc := booking.New(pool, shipSvc, voyageSvc, partnerSvc, accountSvc, auditStore)
	documentCipher, err := guest.NewDevDocumentCipher()
	if err != nil {
		panic("guest document cipher: " + err.Error())
	}
	return Modules{
		Tenant:   tenant.New(pool),
		Ship:     shipSvc,
		Voyage:   voyageSvc,
		Staff:    staffSvc,
		Partner:  partnerSvc,
		Booking:  bookingSvc,
		Guest:    guest.New(pool, bookingSvc, auditStore, documentCipher),
		Accounts: accountSvc,
		AIParams: ai.New(pool),
		Feedback: feedback.New(pool, shipSvc, voyageSvc),
		Sync:     syncmodule.New(pool, shipSvc, voyageSvc),
	}
}

// RegisterRoutes mounts every module that exposes an HTTP surface under /v1.
func (m Modules) RegisterRoutes(r chi.Router) {
	tenant.RegisterRoutes(r, m.Tenant)
	ship.RegisterRoutes(r, m.Ship)
	voyage.RegisterRoutes(r, m.Voyage)
	staff.RegisterRoutes(r, m.Staff)
	partner.RegisterRoutes(r, m.Partner)
	booking.RegisterRoutes(r, m.Booking)
	guest.RegisterRoutes(r, m.Guest)
}
