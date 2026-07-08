// Package share wires the share-link HTTP handlers (owner + public).
package share

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/params"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/apperrors"
	models "sapphirebroking.com/sftp_service/internal/models/share"
	sharesvc "sapphirebroking.com/sftp_service/internal/service/share"
	"sapphirebroking.com/sftp_service/internal/utils"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves the /shares (owner) and /share (public) endpoints.
type Handler struct {
	svc *sharesvc.Service
	log logger.Logger
}

// NewHandler constructs the share Handler.
func NewHandler(svc *sharesvc.Service, log logger.Logger) *Handler {
	return &Handler{svc: svc, log: log.Named("handler.share")}
}

// Create makes a new share link.
func (h *Handler) Create(c fuego.ContextWithBody[models.CreateRequest]) (*response.Envelope[models.CreateResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return nil, fuego.BadRequestError{Title: "a valid file_id or folder_id is required"}
	}
	sh, err := h.svc.Create(c.Context(), uid, body)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*sh, "Share created"), nil
}

// List returns the caller's shares.
func (h *Handler) List(c fuego.ContextNoBody) (*response.Envelope[[]models.Response], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	shares, err := h.svc.List(c.Context(), uid)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(shares), nil
}

// Revoke deactivates a share.
func (h *Handler) Revoke(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	if err := h.svc.Revoke(c.Context(), uid, id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Share revoked"), nil
}

// PublicInfo returns unauthenticated metadata for a share token.
func (h *Handler) PublicInfo(c fuego.ContextNoBody) (*response.Envelope[models.PublicInfo], error) {
	token, err := params.StringPath(c, "token", 0)
	if err != nil {
		return nil, err
	}
	info, err := h.svc.Info(c.Context(), token)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(*info), nil
}

// PublicDownload streams a shared file, or a shared folder as a zip archive
// (std handler; files support range + password).
// Route: GET /share/{token}/download?password=...
func (h *Handler) PublicDownload(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	password := r.URL.Query().Get("password")

	kind, err := h.svc.ShareKind(r.Context(), token)
	if err != nil {
		handlers.WriteProblem(w, r, apperrors.HTTPStatus(err), err.Error(), err)
		return
	}

	if kind == "folder" {
		ft, err := h.svc.AccessFolder(r.Context(), token, password)
		if err != nil {
			handlers.WriteProblem(w, r, apperrors.HTTPStatus(err), err.Error(), err)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+zipFilename(ft.Name)+"\"")
		if _, err := h.svc.WriteFolderZip(r.Context(), ft.Owner, ft.FolderID, ft.Name, w); err != nil {
			h.log.Error("share folder zip failed", "folder", ft.FolderID, "err", err)
		}
		return
	}

	dl, err := h.svc.Access(r.Context(), token, password)
	if err != nil {
		handlers.WriteProblem(w, r, apperrors.HTTPStatus(err), err.Error(), err)
		return
	}
	defer dl.File.Close()

	w.Header().Set("Content-Type", dl.MimeType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+dl.Name+"\"")
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, dl.Name, dl.ModTime, dl.File)
}

// zipFilename turns a folder name into a safe "<name>.zip" download filename.
func zipFilename(name string) string {
	name = strings.NewReplacer("/", "-", "\\", "-", "\"", "", "\n", "", "\r", "").Replace(name)
	name = strings.TrimSpace(name)
	if name == "" {
		name = "folder"
	}
	return name + ".zip"
}

func currentUserID(ctx context.Context) (uuid.UUID, error) {
	claims := jwt.GetClaimsFromContext(ctx)
	if claims == nil || claims.Sub == nil {
		return uuid.Nil, apperrors.ErrUnauthorized
	}
	return uuid.Parse(*claims.Sub)
}
