package httpx

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ren0503/tycoon/internal/apperrors"
)

// MaxUploadBytes is the compressed size limit for one uploaded workbook.
const MaxUploadBytes int64 = 10 << 20

const multipartOverhead int64 = 1 << 20

// ReadUpload reads one bounded multipart file part. Missing parts and malformed
// multipart bodies are bad requests; an oversized file is 413.
func ReadUpload(w http.ResponseWriter, r *http.Request, field string, maxBytes int64) (string, []byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+multipartOverhead)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return "", nil, fileTooLarge()
		}
		return "", nil, apperrors.BadRequest(apperrors.CodeValidationFailed, "Request must be multipart/form-data with one file part.")
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	file, header, err := r.FormFile(field)
	if err != nil {
		return "", nil, apperrors.BadRequest(apperrors.CodeValidationFailed, "A file part named file is required.")
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return "", nil, fileTooLarge()
		}
		return "", nil, apperrors.BadRequest(apperrors.CodeValidationFailed, "Uploaded file could not be read.")
	}
	if int64(len(data)) > maxBytes {
		return "", nil, fileTooLarge()
	}

	name := sanitizeFileName(header.Filename)
	return name, data, nil
}

func fileTooLarge() error {
	return apperrors.New(apperrors.CodeValidationFailed, "Uploaded file exceeds the 10 MiB limit.", http.StatusRequestEntityTooLarge)
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	if name == "." || name == "/" {
		name = ""
	}
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	if len(name) > 255 {
		name = name[:255]
	}
	return name
}
