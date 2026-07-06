package user

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/pkg/jwt"
)

// UploadAvatar stores the authenticated user's profile photo (multipart field:
// "avatar"). Route: POST /users/me/avatar
func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	claims := jwt.GetClaimsFromContext(r.Context())
	if claims == nil || claims.Sub == nil {
		handlers.WriteProblem(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	uid, err := uuid.Parse(*claims.Sub)
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusUnauthorized, "invalid session")
		return
	}
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid multipart form")
		return
	}
	part, _, err := r.FormFile("avatar")
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "missing avatar field")
		return
	}
	defer part.Close()

	if err := h.svc.SetAvatar(r.Context(), uid, part); err != nil {
		handlers.WriteProblem(w, r, http.StatusInternalServerError, "could not save avatar")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true,"message":"Avatar updated"}`))
}

// Avatar streams a user's profile photo. Route: GET /users/{id}/avatar
func (h *Handler) Avatar(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid user id")
		return
	}
	rsc, err := h.svc.OpenAvatar(r.Context(), id)
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusNotFound, "no avatar")
		return
	}
	defer rsc.Close()
	w.Header().Set("Cache-Control", "private, max-age=300")
	// ServeContent sniffs the content type from the data.
	http.ServeContent(w, r, "avatar", time.Time{}, rsc)
}
