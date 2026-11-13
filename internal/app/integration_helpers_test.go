//go:build integration

package app

import (
	"log/slog"

	airepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/ai"
	authrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/auth"
	bookingrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/booking"
	feedbackrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/feedback"
	guestrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/guest"
	partnerrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/partner"
	shiprepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/ship"
	staffrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/staff"
	syncrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/sync"
	tenantrepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/tenant"
	voyagerepo "github.com/ren0503/tycoon/internal/infrastructure/persistence/postgres/voyage"
	"github.com/ren0503/tycoon/internal/platform/testutil"
)

func integrationModuleDeps(h *testutil.Harness, logger *slog.Logger) ModuleDeps {
	return ModuleDeps{
		Logger:       logger,
		Accounts:     authrepo.NewAccountService(h.Pool),
		TenantRepo:   tenantrepo.New(h.Pool),
		ShipRepo:     shiprepo.New(h.Pool),
		VoyageRepo:   voyagerepo.New(h.Pool),
		StaffRepo:    staffrepo.New(h.Pool),
		PartnerRepo:  partnerrepo.New(h.Pool),
		BookingRepo:  bookingrepo.New(h.Pool),
		GuestRepo:    guestrepo.New(h.Pool),
		AIRepo:       airepo.New(h.Pool),
		FeedbackRepo: feedbackrepo.New(h.Pool),
		SyncRepo:     syncrepo.New(h.Pool),
	}
}
