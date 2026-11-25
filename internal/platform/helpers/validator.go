package helpers

import (
	"time"

	apperrors "github.com/ren0503/tycoon/internal/platform/errors"
)

func Validation(message string) error {
	return apperrors.Validation(apperrors.CodeValidationFailed, message)
}

func ValidateDateRange(from, to *time.Time) error {
	if from != nil && to != nil && to.Before(*from) {
		return Validation("Valid-to date must not be before valid-from.")
	}
	return nil
}
