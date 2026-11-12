package ai

import (
	"time"

	"github.com/google/uuid"
)

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

type ListAIParamsParams struct {
	TenantID  uuid.UUID     `json:"tenant_id"`
	Kind      *string       `json:"kind"`
	After     uuid.NullUUID `json:"after"`
	PageLimit int32         `json:"page_limit"`
}

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
