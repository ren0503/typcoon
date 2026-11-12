package guest

import (
	"context"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/application/audit"
)

type Repository interface {
	Begin(context.Context) (Tx, error)
	GetGuest(context.Context, uuid.UUID) (Guest, error)
	ListGuests(context.Context, ListGuestsParams) ([]Guest, error)
	GetBoardingPass(context.Context, uuid.UUID) (BoardingPass, error)
	ListBoardingPasses(context.Context, ListBoardingPassesParams) ([]BoardingPass, error)
}

type Tx interface {
	CreateGuest(context.Context, CreateGuestParams) (Guest, error)
	UpdateGuest(context.Context, UpdateGuestParams) (Guest, error)
	SoftDeleteGuest(context.Context, uuid.UUID) error
	CreateBoardingPass(context.Context, CreateBoardingPassParams) (BoardingPass, error)
	RevokeBoardingPass(context.Context, uuid.UUID) (BoardingPass, error)
	AppendAudit(context.Context, audit.Entry) error
	Commit(context.Context) error
	Rollback(context.Context) error
}
