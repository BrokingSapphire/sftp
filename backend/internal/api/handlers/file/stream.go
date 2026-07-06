package file

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/apperrors"
	filesvc "sapphirebroking.com/sftp_service/internal/service/file"
	"sapphirebroking.com/sftp_service/pkg/headers"
)

// PutChunk receives one chunk of a resumable upload (raw request body).
// Route: PUT /files/uploads/{id}/chunks/{index}
func (h *Handler) PutChunk(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		handlers.WriteProblem(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	uploadID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid upload id")
		return
	}
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || index < 0 {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid chunk index")
		return
	}
	defer r.Body.Close()

	status, err := h.svc.PutChunk(r.Context(), uid, uploadID, index, r.Body)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Success: true, Message: "Chunk received", Data: status})
}

// SimpleUpload stores a single file from a multipart form.
// Route: POST /files/upload   (form fields: file, folder_id?)
func (h *Handler) SimpleUpload(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		handlers.WriteProblem(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	// 32 MiB in-memory threshold; larger parts spill to temp files.
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid multipart form")
		return
	}
	part, header, err := r.FormFile("file")
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "missing file field")
		return
	}
	defer part.Close()

	var folderID *string
	if v := r.FormValue("folder_id"); v != "" {
		folderID = &v
	}
	f, err := h.svc.SimpleUpload(r.Context(), uid, folderID, header.Filename, part)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Success: true, Message: "File uploaded", Data: f})
}

// CommonUpload stores a file directly into the organisation-wide Common area.
// Route: POST /files/common/upload  (multipart, any authenticated user)
func (h *Handler) CommonUpload(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		handlers.WriteProblem(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid multipart form")
		return
	}
	part, header, err := r.FormFile("file")
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "missing file field")
		return
	}
	defer part.Close()

	f, err := h.svc.UploadCommon(r.Context(), uid, header.Filename, part)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Success: true, Message: "Added to Common", Data: f})
}

// Download streams a file with HTTP range support.
// Route: GET /files/{id}/download
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		handlers.WriteProblem(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid file id")
		return
	}
	dl, err := h.svc.OpenForDownload(r.Context(), uid, id, filesvc.DownloadMeta{
		IP: headers.GetClientIP(r), UserAgent: r.UserAgent(),
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	defer dl.File.Close()

	// inline=1 lets the browser render the file in-page (preview/viewer);
	// otherwise force a download.
	disposition := "attachment"
	if r.URL.Query().Get("inline") == "1" {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", dl.MimeType)
	w.Header().Set("Content-Disposition", disposition+"; filename=\""+dl.Name+"\"")
	w.Header().Set("Accept-Ranges", "bytes")
	// ServeContent handles Range, If-Range, and conditional requests.
	http.ServeContent(w, r, dl.Name, dl.ModTime, dl.File)
}

// DownloadVersion streams a specific archived version of a file.
func (h *Handler) DownloadVersion(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		handlers.WriteProblem(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid file id")
		return
	}
	vn, err := strconv.Atoi(r.PathValue("version"))
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid version")
		return
	}
	dl, err := h.svc.OpenVersionForDownload(r.Context(), uid, id, int32(vn))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	defer dl.File.Close()
	w.Header().Set("Content-Type", dl.MimeType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+dl.Name+"\"")
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, dl.Name, dl.ModTime, dl.File)
}

// ── std-handler helpers ────────────────────────────────────

type envelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	handlers.WriteProblem(w, r, apperrors.HTTPStatus(err), err.Error(), err)
}

func userID(r *http.Request) (uuid.UUID, bool) {
	id, err := currentUserID(r.Context())
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
