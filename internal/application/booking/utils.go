package booking

import (
	"strings"

	apperrors "github.com/ren0503/tycoon/internal/platform/errors"
)

func isNotFound(err error) bool {
	appErr, ok := apperrors.As(err)
	return ok && appErr.HTTPStatus == 404
}

func trimPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func validation(message string) error {
	return apperrors.Validation(apperrors.CodeValidationFailed, message)
}
