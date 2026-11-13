package app

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	appai "github.com/ren0503/tycoon/internal/application/ai"
	appauth "github.com/ren0503/tycoon/internal/application/auth"
	appbooking "github.com/ren0503/tycoon/internal/application/booking"
	appfeedback "github.com/ren0503/tycoon/internal/application/feedback"
	appguest "github.com/ren0503/tycoon/internal/application/guest"
	apppartner "github.com/ren0503/tycoon/internal/application/partner"
	appship "github.com/ren0503/tycoon/internal/application/ship"
	appstaff "github.com/ren0503/tycoon/internal/application/staff"
	appsync "github.com/ren0503/tycoon/internal/application/sync"
	apptenant "github.com/ren0503/tycoon/internal/application/tenant"
	appvoyage "github.com/ren0503/tycoon/internal/application/voyage"
	"github.com/ren0503/tycoon/internal/transport/http/handlers/auth"
	"github.com/ren0503/tycoon/internal/transport/http/handlers/booking"
	"github.com/ren0503/tycoon/internal/transport/http/handlers/guest"
	"github.com/ren0503/tycoon/internal/transport/http/handlers/partner"
	"github.com/ren0503/tycoon/internal/transport/http/handlers/ship"
	"github.com/ren0503/tycoon/internal/transport/http/handlers/staff"
	"github.com/ren0503/tycoon/internal/transport/http/handlers/tenant"
	"github.com/ren0503/tycoon/internal/transport/http/handlers/voyage"
)

// ModuleDeps supplies application ports to the composition root.
type ModuleDeps struct {
	Logger *slog.Logger

	Auth         appauth.Service
	Accounts     appauth.AccountService
	TenantRepo   apptenant.Repository
	ShipRepo     appship.Repository
	VoyageRepo   appvoyage.Repository
	StaffRepo    appstaff.Repository
	PartnerRepo  apppartner.Repository
	BookingRepo  appbooking.Repository
	GuestRepo    appguest.Repository
	AIRepo       appai.Repository
	FeedbackRepo appfeedback.Repository
	SyncRepo     appsync.Repository

	DocumentCipher appguest.DocumentCipher
}

type Modules struct {
	AuthSvc     appauth.Service
	TenantSvc   apptenant.TenantService
	ShipSvc     appship.Service
	VoyageSvc   appvoyage.Service
	StaffSvc    appstaff.Service
	PartnerSvc  apppartner.Service
	BookingSvc  appbooking.Service
	GuestSvc    appguest.Service
	AIParamsSvc appai.AIParamService
	FeedbackSvc appfeedback.FeedbackService
	SyncSvc     appsync.Service
}

// BuildModules constructs application services from injected ports only.
func BuildModules(deps ModuleDeps) Modules {
	shipSvc := appship.New(deps.ShipRepo, deps.Logger)
	voyageSvc := appvoyage.New(deps.VoyageRepo, deps.Logger, shipSvc)
	staffSvc := appstaff.New(deps.StaffRepo, deps.Logger, shipSvc)
	partnerSvc := apppartner.New(deps.PartnerRepo, shipSvc)
	bookingSvc := appbooking.New(deps.BookingRepo, shipSvc, voyageSvc, partnerSvc, deps.Accounts)
	return Modules{
		AuthSvc:     deps.Auth,
		TenantSvc:   apptenant.New(deps.TenantRepo),
		ShipSvc:     shipSvc,
		VoyageSvc:   voyageSvc,
		StaffSvc:    staffSvc,
		PartnerSvc:  partnerSvc,
		BookingSvc:  bookingSvc,
		GuestSvc:    appguest.New(deps.GuestRepo, bookingSvc, deps.DocumentCipher),
		AIParamsSvc: appai.New(deps.AIRepo),
		FeedbackSvc: appfeedback.New(deps.FeedbackRepo, shipSvc, voyageSvc),
		SyncSvc:     appsync.New(deps.SyncRepo, shipSvc, voyageSvc),
	}
}

// RegisterRoutes mounts every module that exposes an HTTP surface under /api.
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
