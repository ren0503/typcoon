package guest

import "time"

type guestCreateRequest struct {
	FullName            string  `json:"full_name" validate:"required,max=200"`
	DateOfBirth         *string `json:"date_of_birth"`
	Nationality         *string `json:"nationality" validate:"omitempty,max=100"`
	DocumentType        *string `json:"document_type" validate:"omitempty,max=50"`
	DocumentEnc         *string `json:"document_enc"`
	DocumentKeyID       *string `json:"document_key_id" validate:"omitempty,max=100"`
	EmergencyContactEnc *string `json:"emergency_contact_enc"`
	SpecialNeeds        *string `json:"special_needs" validate:"omitempty,max=2000"`
	IsLead              bool    `json:"is_lead"`
	MusterStation       *string `json:"muster_station" validate:"omitempty,max=50"`
	Document            *string `json:"document"`
}

type guestUpdateRequest struct {
	FullName      *string `json:"full_name" validate:"omitempty,max=200"`
	SpecialNeeds  *string `json:"special_needs" validate:"omitempty,max=2000"`
	IsLead        *bool   `json:"is_lead"`
	MusterStation *string `json:"muster_station" validate:"omitempty,max=50"`
	DocumentType  *string `json:"document_type" validate:"omitempty,max=50"`
	Document      *string `json:"document"`
}

type boardingPassIssueRequest struct {
	GuestID   *string    `json:"guest_id" validate:"omitempty,uuid"`
	ExpiresAt *time.Time `json:"expires_at"`
}
