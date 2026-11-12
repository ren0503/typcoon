package httpx

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ren0503/tycoon/internal/platform/errors"
)

func TestReadUploadReturnsSanitizedFile(t *testing.T) {
	body, contentType := multipartBody(t, "file", "../../tmp/Workbook.xlsx", []byte("workbook"))
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	name, data, err := ReadUpload(rec, req, "file", MaxUploadBytes)
	if err != nil {
		t.Fatalf("read upload: %v", err)
	}
	if name != "Workbook.xlsx" || string(data) != "workbook" {
		t.Fatalf("upload = %q %q", name, data)
	}
}

func TestReadUploadRejectsMalformedAndMissingFiles(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("not multipart"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	rec := httptest.NewRecorder()
	if _, _, err := ReadUpload(rec, req, "file", MaxUploadBytes); errorCode(err) != apperrors.CodeValidationFailed {
		t.Fatalf("malformed error = %v", err)
	}

	body, contentType := multipartBody(t, "other", "other.xlsx", []byte("workbook"))
	req = httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec = httptest.NewRecorder()
	if _, _, err := ReadUpload(rec, req, "file", MaxUploadBytes); errorCode(err) != apperrors.CodeValidationFailed {
		t.Fatalf("missing file error = %v", err)
	}
}

func TestReadUploadRejectsOversizedFile(t *testing.T) {
	body, contentType := multipartBody(t, "file", "large.xlsx", bytes.Repeat([]byte("x"), int(MaxUploadBytes)+1))
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	_, _, err := ReadUpload(rec, req, "file", MaxUploadBytes)
	appErr, ok := apperrors.As(err)
	if !ok || appErr.HTTPStatus != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized error = %v", err)
	}
}

func multipartBody(t *testing.T, field, name string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, name)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return &body, writer.FormDataContentType()
}

func errorCode(err error) string {
	appErr, ok := apperrors.As(err)
	if !ok {
		return ""
	}
	return appErr.Code
}
