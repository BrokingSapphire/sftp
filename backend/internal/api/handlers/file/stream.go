package file

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

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
	// Stream the file part straight to storage (see CommonUpload) so large files
	// don't fill /tmp. folder_id is sent before the file.
	mr, err := r.MultipartReader()
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid multipart form")
		return
	}
	var folderID *string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			handlers.WriteProblem(w, r, http.StatusBadRequest, "could not read upload")
			return
		}
		switch part.FormName() {
		case "folder_id":
			b, _ := io.ReadAll(io.LimitReader(part, 256))
			part.Close()
			if v := strings.TrimSpace(string(b)); v != "" {
				folderID = &v
			}
		case "file":
			f, uerr := h.svc.SimpleUpload(r.Context(), uid, folderID, part.FileName(), part)
			part.Close()
			if uerr != nil {
				writeServiceError(w, r, uerr)
				return
			}
			writeJSON(w, http.StatusOK, envelope{Success: true, Message: "File uploaded", Data: f})
			return
		default:
			part.Close()
		}
	}
	handlers.WriteProblem(w, r, http.StatusBadRequest, "missing file field")
}

// CommonUpload stores a file directly into the organisation-wide Common area.
// Route: POST /files/common/upload  (multipart, any authenticated user)
func (h *Handler) CommonUpload(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		handlers.WriteProblem(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	// Stream the multipart body part-by-part so large files (hundreds of MB) are
	// piped straight to storage instead of being buffered in memory / a temp file
	// (which fails on containers with a small /tmp). The frontend sends folder_id
	// before the file so the target is known when the file part arrives.
	mr, err := r.MultipartReader()
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid multipart form")
		return
	}
	var folderID *string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			handlers.WriteProblem(w, r, http.StatusBadRequest, "could not read upload")
			return
		}
		switch part.FormName() {
		case "folder_id":
			b, _ := io.ReadAll(io.LimitReader(part, 256))
			part.Close()
			if v := strings.TrimSpace(string(b)); v != "" {
				folderID = &v
			}
		case "file":
			f, uerr := h.svc.UploadCommonTo(r.Context(), uid, folderID, part.FileName(), part)
			part.Close()
			if uerr != nil {
				writeServiceError(w, r, uerr)
				return
			}
			writeJSON(w, http.StatusOK, envelope{Success: true, Message: "Added to Common", Data: f})
			return
		default:
			part.Close()
		}
	}
	handlers.WriteProblem(w, r, http.StatusBadRequest, "missing file field")
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

// FolderDownload streams a folder (recursively) as a zip archive.
func (h *Handler) FolderDownload(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		handlers.WriteProblem(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid folder id")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="folder.zip"`)
	if _, err := h.svc.WriteFolderZip(r.Context(), uid, id, w); err != nil {
		h.log.Error("folder zip failed", "folder", id, "err", err)
	}
}

// SaveContent overwrites a file's content from the raw request body (used by
// the in-app editor). Creates a new version.
func (h *Handler) SaveContent(w http.ResponseWriter, r *http.Request) {
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
	defer r.Body.Close()
	// Editor documents are text — cap the body generously at 25 MiB.
	f, err := h.svc.OverwriteContent(r.Context(), uid, id, http.MaxBytesReader(w, r.Body, 25<<20))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Success: true, Message: "Saved", Data: f})
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
