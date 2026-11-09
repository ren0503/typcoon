package app

import (
	"log/slog"

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

type Modules struct {
	AuthSvc     auth.Service
	TenantSvc   tenant.TenantService
	ShipSvc     ship.Service
	VoyageSvc   voyage.Service
	StaffSvc    staff.Service
	PartnerSvc  partner.Service
	BookingSvc  booking.Service
	GuestSvc    guest.Service
	AccountsSvc auth.AccountService
	AIParamsSvc ai.AIParamService
	FeedbackSvc feedback.FeedbackService
	SyncSvc     syncmodule.Service
}

func BuildModules(pool *pgxpool.Pool, logger *slog.Logger, authService auth.Service) Modules {
	auditStore := audit.NewStore(pool, logger)
	shipSvc := ship.New(pool, logger)
	voyageSvc := voyage.New(pool, logger, shipSvc)
	staffSvc := staff.New(pool, logger, shipSvc, auditStore)
	partnerSvc := partner.New(pool, shipSvc)
	accountSvc := auth.NewAccountService(pool)
	bookingSvc := booking.New(pool, shipSvc, voyageSvc, partnerSvc, accountSvc, auditStore)
	documentCipher, err := guest.NewDevDocumentCipher()
	if err != nil {
		panic("guest document cipher: " + err.Error())
	}
	return Modules{
		AuthSvc:     authService,
		TenantSvc:   tenant.New(pool),
		ShipSvc:     shipSvc,
		VoyageSvc:   voyageSvc,
		StaffSvc:    staffSvc,
		PartnerSvc:  partnerSvc,
		BookingSvc:  bookingSvc,
		GuestSvc:    guest.New(pool, bookingSvc, auditStore, documentCipher),
		AccountsSvc: accountSvc,
		AIParamsSvc: ai.New(pool),
		FeedbackSvc: feedback.New(pool, shipSvc, voyageSvc),
		SyncSvc:     syncmodule.New(pool, shipSvc, voyageSvc),
	}
}

// RegisterRoutes mounts every module that exposes an HTTP surface under /v1.
func (m Modules) RegisterRoutes(r chi.Router) {
	tenant.RegisterRoutes(r, m.TenantSvc)
	ship.RegisterRoutes(r, m.ShipSvc)
	voyage.RegisterRoutes(r, m.VoyageSvc)
	staff.RegisterRoutes(r, m.StaffSvc)
	partner.RegisterRoutes(r, m.PartnerSvc)
	booking.RegisterRoutes(r, m.BookingSvc)
	guest.RegisterRoutes(r, m.GuestSvc)
	auth.RegisterProtectedRoutes(r, m.AuthSvc)
}

// RegisterPublicRoutes mounts public auth routes outside the bearer middleware.
func (m Modules) RegisterPublicRoutes(r chi.Router) {
	auth.RegisterPublicRoutes(r, m.AuthSvc)
}
