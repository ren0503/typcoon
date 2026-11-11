package ship

import (
	"net/http"

	"github.com/ren0503/tycoon/internal/auth"
	"github.com/ren0503/tycoon/internal/httpx"
)

func uploadStructureImport(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		name, data, err := httpx.ReadUpload(w, r, "file", httpx.MaxUploadBytes)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		preview, err := svc.UploadStructureImport(r.Context(), principal, shipID, name, data, httpx.Header(r))
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, preview)
	}
}

func getStructureImport(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		importID, err := httpx.PathID(r, "importId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.GetStructureImport(r.Context(), principal, shipID, importID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func previewStructureImport(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		importID, err := httpx.PathID(r, "importId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		preview, err := svc.PreviewStructureImport(r.Context(), principal, shipID, importID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, preview)
	}
}

func confirmStructureImport(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		shipID, err := httpx.PathID(r, "shipId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		importID, err := httpx.PathID(r, "importId")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		row, err := svc.ConfirmStructureImport(r.Context(), principal, shipID, importID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, row)
	}
}

func structureImportTemplate(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := svc.StructureImportTemplate(r.Context())
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="ship-structure-import-template.xlsx"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}
