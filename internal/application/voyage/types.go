package voyage

import (
	"time"

	"github.com/google/uuid"
)

type Port struct {
	ID          uuid.UUID  `json:"id"`
	UnLocode    string     `json:"un_locode"`
	Name        string     `json:"name"`
	CountryCode string     `json:"country_code"`
	Latitude    *float64   `json:"latitude"`
	Longitude   *float64   `json:"longitude"`
	Timezone    string     `json:"timezone"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Version     int64      `json:"version"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type Voyage struct {
	ID               uuid.UUID  `json:"id"`
	ShipID           uuid.UUID  `json:"ship_id"`
	Code             string     `json:"code"`
	Status           string     `json:"status"`
	DepartureAt      time.Time  `json:"departure_at"`
	ArrivalAt        time.Time  `json:"arrival_at"`
	BoardingOpensAt  *time.Time `json:"boarding_opens_at"`
	ManifestFrozenAt *time.Time `json:"manifest_frozen_at"`
	Ownership        string     `json:"ownership"`
	ScheduleVersion  int32      `json:"schedule_version"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Version          int64      `json:"version"`
	DeletedAt        *time.Time `json:"deleted_at"`
}

type PortCall struct {
	ID              uuid.UUID  `json:"id"`
	ShipID          uuid.UUID  `json:"ship_id"`
	VoyageID        uuid.UUID  `json:"voyage_id"`
	PortID          uuid.UUID  `json:"port_id"`
	Sequence        int32      `json:"sequence"`
	Eta             *time.Time `json:"eta"`
	Etd             *time.Time `json:"etd"`
	Ata             *time.Time `json:"ata"`
	Atd             *time.Time `json:"atd"`
	AllowsEmbark    bool       `json:"allows_embark"`
	AllowsDisembark bool       `json:"allows_disembark"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Version         int64      `json:"version"`
	DeletedAt       *time.Time `json:"deleted_at"`
}

type CreatePortParams struct {
	ID          uuid.UUID
	UnLocode    string
	Name        string
	CountryCode string
	Latitude    *float64
	Longitude   *float64
	Timezone    string
}

type UpdatePortParams struct {
	Name      string
	Latitude  *float64
	Longitude *float64
	Timezone  string
	ID        uuid.UUID
}

type ListPortsParams struct {
	After     uuid.NullUUID
	PageLimit int32
}

type CreateVoyageParams struct {
	ID               uuid.UUID
	ShipID           uuid.UUID
	Code             string
	DepartureAt      time.Time
	ArrivalAt        time.Time
	BoardingOpensAt  *time.Time
	ManifestFrozenAt *time.Time
}

type UpdateVoyageParams struct {
	Code             string
	DepartureAt      time.Time
	ArrivalAt        time.Time
	BoardingOpensAt  *time.Time
	ManifestFrozenAt *time.Time
	ID               uuid.UUID
}

type ListVoyagesParams struct {
	ShipID    uuid.NullUUID
	AfterSort *time.Time
	After     uuid.NullUUID
	PageLimit int32
}

type ListVoyagesForTenantParams struct {
	TenantID  uuid.UUID
	AfterSort *time.Time
	After     uuid.NullUUID
	PageLimit int32
}

type CreatePortCallParams struct {
	ID              uuid.UUID
	ShipID          uuid.UUID
	VoyageID        uuid.UUID
	PortID          uuid.UUID
	Sequence        int32
	Eta             *time.Time
	Etd             *time.Time
	Ata             *time.Time
	Atd             *time.Time
	AllowsEmbark    bool
	AllowsDisembark bool
	Status          string
}

type UpdatePortCallParams struct {
	Sequence        int32
	Eta             *time.Time
	Etd             *time.Time
	Ata             *time.Time
	Atd             *time.Time
	AllowsEmbark    bool
	AllowsDisembark bool
	Status          string
	ID              uuid.UUID
}

type ListPortCallsParams struct {
	VoyageID  uuid.UUID
	AfterSeq  *int64
	After     uuid.NullUUID
	PageLimit int32
}
