package ai

import (
	"time"

	"github.com/google/uuid"
)

// AiParam is one signed AI parameter version.
type AiParam struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Kind         string     `json:"kind"`
	ParamVersion int32      `json:"param_version"`
	StorageKey   string     `json:"storage_key"`
	Sha256       string     `json:"sha256"`
	Kid          string     `json:"kid"`
	Signature    string     `json:"signature"`
	Status       string     `json:"status"`
	ReleasedAt   *time.Time `json:"released_at"`
	Notes        *string    `json:"notes"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Version      int64      `json:"version"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

// CreateAIParamParams is the input for creating an AI parameter.
type CreateAIParamParams struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Kind         string     `json:"kind"`
	ParamVersion int32      `json:"param_version"`
	StorageKey   string     `json:"storage_key"`
	Sha256       string     `json:"sha256"`
	Kid          string     `json:"kid"`
	Signature    string     `json:"signature"`
	Status       string     `json:"status"`
	ReleasedAt   *time.Time `json:"released_at"`
	Notes        *string    `json:"notes"`
}

// ListAIParamsParams filters an AI parameter listing.
type ListAIParamsParams struct {
	TenantID  uuid.UUID     `json:"tenant_id"`
	Kind      *string       `json:"kind"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

// UpdateAIParamMetadataParams updates signed AI parameter metadata.
type UpdateAIParamMetadataParams struct {
	StorageKey string     `json:"storage_key"`
	Sha256     string     `json:"sha256"`
	Kid        string     `json:"kid"`
	Signature  string     `json:"signature"`
	Status     string     `json:"status"`
	ReleasedAt *time.Time `json:"released_at"`
	Notes      *string    `json:"notes"`
	ID         uuid.UUID  `json:"id"`
}

// ParamInput creates an AI parameter metadata record.
type ParamInput struct {
	TenantID     uuid.UUID
	Kind         string
	ParamVersion int32
	StorageKey   string
	Sha256       string
	Kid          string
	Signature    string
	Status       string
	ReleasedAt   *time.Time
	Notes        *string
}

// ParamUpdate updates metadata for an existing AI parameter.
type ParamUpdate struct {
	StorageKey string
	Sha256     string
	Kid        string
	Signature  string
	Status     string
	ReleasedAt *time.Time
	Notes      *string
}

func (row AiParam) toView() ParamView {
	return ParamView{
		ID: row.ID, TenantID: row.TenantID, Kind: row.Kind, ParamVersion: row.ParamVersion,
		StorageKey: row.StorageKey, Sha256: row.Sha256, Kid: row.Kid, Signature: row.Signature,
		Status: row.Status, ReleasedAt: row.ReleasedAt, Notes: row.Notes,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Version: row.Version,
	}
}

func (input ParamInput) toCreateAPIParam(id uuid.UUID) CreateAIParamParams {
	return CreateAIParamParams{
		ID: id, TenantID: input.TenantID, Kind: input.Kind, ParamVersion: input.ParamVersion,
		StorageKey: input.StorageKey, Sha256: input.Sha256, Kid: input.Kid, Signature: input.Signature,
		Status: input.Status, ReleasedAt: input.ReleasedAt, Notes: input.Notes,
	}
}

func (input ParamUpdate) toUpdateAIParamMetadataParams(id uuid.UUID) UpdateAIParamMetadataParams {
	return UpdateAIParamMetadataParams{
		ID: id, StorageKey: input.StorageKey, Sha256: input.Sha256, Kid: input.Kid,
		Signature: input.Signature, Status: input.Status, ReleasedAt: input.ReleasedAt, Notes: input.Notes,
	}
}
