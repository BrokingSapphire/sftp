// Package file wires the file/folder HTTP handlers.
package file

import (
	"context"

	"strconv"
	"time"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/params"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/apperrors"
	models "sapphirebroking.com/sftp_service/internal/models/file"
	filesvc "sapphirebroking.com/sftp_service/internal/service/file"
	"sapphirebroking.com/sftp_service/internal/utils"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves the /files and /folders endpoints.
type Handler struct {
	svc *filesvc.Service
	log logger.Logger
}

// NewHandler constructs the file Handler.
func NewHandler(svc *filesvc.Service, log logger.Logger) *Handler {
	return &Handler{svc: svc, log: log.Named("handler.file")}
}

// ── Folders ───────────────────────────────────────────────

// CreateFolder creates a folder.
func (h *Handler) CreateFolder(c fuego.ContextWithBody[models.CreateFolderRequest]) (*response.Envelope[models.FolderResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return nil, fuego.BadRequestError{Title: "name is required"}
	}
	folder, err := h.svc.CreateFolder(c.Context(), uid, body)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*folder, "Folder created"), nil
}

// List returns the contents of a folder (query: folder_id, limit, offset).
func (h *Handler) List(c fuego.ContextNoBody) (*response.Envelope[models.ListingResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	folderID, err := optionalUUID(c.QueryParam("folder_id"))
	if err != nil {
		return nil, fuego.BadRequestError{Title: "invalid folder_id"}
	}
	limit := params.IntQueryDefault(c, "limit", 100)
	offset := params.IntQueryDefault(c, "offset", 0)
	listing, total, err := h.svc.ListFolder(c.Context(), uid, folderID, limit, offset)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.Paginated(*listing, models.ListMeta{Total: total, Limit: limit, Offset: offset}), nil
}

// RenameFolder renames a folder.
func (h *Handler) RenameFolder(c fuego.ContextWithBody[models.RenameRequest]) (*response.Envelope[response.Any], error) {
	uid, id, body, err := h.idAndBody(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.RenameFolder(c.Context(), uid, id, body.Name); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Folder renamed"), nil
}

// MoveFolder reparents a folder.
func (h *Handler) MoveFolder(c fuego.ContextWithBody[models.MoveRequest]) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, _ := c.Body()
	target, err := optionalUUID(deref(body.TargetID))
	if err != nil {
		return nil, fuego.BadRequestError{Title: "invalid target_id"}
	}
	if err := h.svc.MoveFolder(c.Context(), uid, id, target); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Folder moved"), nil
}

// DeleteFolder soft-deletes an empty folder.
func (h *Handler) DeleteFolder(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteFolder(c.Context(), uid, id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Folder deleted"), nil
}

// StarFolder toggles a folder's star.
func (h *Handler) StarFolder(c fuego.ContextWithBody[models.StarRequest]) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, _ := c.Body()
	if err := h.svc.StarFolder(c.Context(), uid, id, body.Starred); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Folder updated"), nil
}

// SetFolderColor sets a folder's colour.
func (h *Handler) SetFolderColor(c fuego.ContextWithBody[models.ColorRequest]) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, _ := c.Body()
	if err := h.svc.SetFolderColor(c.Context(), uid, id, body.Color); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Folder colour updated"), nil
}

// ── Files ─────────────────────────────────────────────────

// GetFile returns file metadata.
func (h *Handler) GetFile(c fuego.ContextNoBody) (*response.Envelope[models.FileResponse], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	f, err := h.svc.GetFile(c.Context(), uid, id)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(*f), nil
}

// RenameFile renames a file.
func (h *Handler) RenameFile(c fuego.ContextWithBody[models.RenameRequest]) (*response.Envelope[response.Any], error) {
	uid, id, body, err := h.idAndBody(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.RenameFile(c.Context(), uid, id, body.Name); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "File renamed"), nil
}

// MoveFile moves a file to another folder.
func (h *Handler) MoveFile(c fuego.ContextWithBody[models.MoveRequest]) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, _ := c.Body()
	target, err := optionalUUID(deref(body.TargetID))
	if err != nil {
		return nil, fuego.BadRequestError{Title: "invalid target_id"}
	}
	if err := h.svc.MoveFile(c.Context(), uid, id, target); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "File moved"), nil
}

// StarFile toggles a file's star.
func (h *Handler) StarFile(c fuego.ContextWithBody[models.StarRequest]) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, _ := c.Body()
	if err := h.svc.StarFile(c.Context(), uid, id, body.Starred); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "File updated"), nil
}

// TrashFile moves a file to the recycle bin.
func (h *Handler) TrashFile(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.TrashFile(c.Context(), uid, id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "File moved to trash"), nil
}

// RestoreFile restores a file from the recycle bin.
func (h *Handler) RestoreFile(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.RestoreFile(c.Context(), uid, id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "File restored"), nil
}

// DeleteFile permanently deletes a file.
func (h *Handler) DeleteFile(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeletePermanent(c.Context(), uid, id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "File permanently deleted"), nil
}

// Trash lists soft-deleted files.
func (h *Handler) Trash(c fuego.ContextNoBody) (*response.Envelope[[]models.FileResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	files, err := h.svc.ListTrash(c.Context(), uid, params.IntQueryDefault(c, "limit", 100), params.IntQueryDefault(c, "offset", 0))
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(files), nil
}

// CopyFile duplicates a file.
func (h *Handler) CopyFile(c fuego.ContextNoBody) (*response.Envelope[models.FileResponse], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	f, err := h.svc.CopyFile(c.Context(), uid, id)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*f, "File copied"), nil
}

// EmptyTrash permanently deletes all of the caller's trashed files.
func (h *Handler) EmptyTrash(c fuego.ContextNoBody) (*response.Envelope[map[string]int], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	n, err := h.svc.EmptyTrash(c.Context(), uid)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(map[string]int{"deleted": n}, "Trash emptied"), nil
}

// Recent lists recently created files.
func (h *Handler) Recent(c fuego.ContextNoBody) (*response.Envelope[[]models.FileResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	files, err := h.svc.ListRecent(c.Context(), uid, params.IntQueryDefault(c, "limit", 20))
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(files), nil
}

// Starred lists starred files.
func (h *Handler) Starred(c fuego.ContextNoBody) (*response.Envelope[[]models.FileResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	files, err := h.svc.ListStarred(c.Context(), uid)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(files), nil
}

// Search finds files by name.
func (h *Handler) Search(c fuego.ContextNoBody) (*response.Envelope[[]models.FileResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	q := c.QueryParam("q")
	if q == "" {
		return nil, fuego.BadRequestError{Title: "q is required"}
	}
	files, err := h.svc.Search(c.Context(), uid, q, params.IntQueryDefault(c, "limit", 50), params.IntQueryDefault(c, "offset", 0))
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(files), nil
}

// SearchContent runs full-text search over file contents (returns snippets).
func (h *Handler) SearchContent(c fuego.ContextNoBody) (*response.Envelope[[]models.SearchHit], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	q := c.QueryParam("q")
	if q == "" {
		return nil, fuego.BadRequestError{Title: "q is required"}
	}
	hits, err := h.svc.SearchContent(c.Context(), uid, q, params.IntQueryDefault(c, "limit", 30))
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(hits), nil
}

// ListVersions returns a file's previous versions.
func (h *Handler) ListVersions(c fuego.ContextNoBody) (*response.Envelope[[]models.FileVersionResponse], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	vs, err := h.svc.ListVersions(c.Context(), uid, id)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(vs), nil
}

// RestoreVersion makes a previous version current.
func (h *Handler) RestoreVersion(c fuego.ContextNoBody) (*response.Envelope[models.FileResponse], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	vn, err := strconv.Atoi(c.PathParam("version"))
	if err != nil {
		return nil, fuego.BadRequestError{Title: "invalid version"}
	}
	f, err := h.svc.RestoreVersion(c.Context(), uid, id, int32(vn))
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*f, "Version restored"), nil
}

// SetLegalHold places or releases a legal hold on a file (admin).
func (h *Handler) SetLegalHold(c fuego.ContextWithBody[models.LegalHoldRequest]) (*response.Envelope[response.Any], error) {
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, _ := c.Body()
	if err := h.svc.SetLegalHold(c.Context(), id, body.Hold); err != nil {
		return nil, handlers.Fail(err)
	}
	msg := "Legal hold released"
	if body.Hold {
		msg = "Legal hold placed"
	}
	return response.OKWithMessage[response.Any](nil, msg), nil
}

// SetRetention sets or clears a WORM retention lock on a file (admin).
func (h *Handler) SetRetention(c fuego.ContextWithBody[models.RetentionRequest]) (*response.Envelope[response.Any], error) {
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, _ := c.Body()
	var until *time.Time
	if body.Until != nil && *body.Until != "" {
		t, perr := time.Parse(time.RFC3339, *body.Until)
		if perr != nil {
			return nil, fuego.BadRequestError{Title: "until must be an RFC3339 timestamp"}
		}
		until = &t
	}
	if err := h.svc.SetRetention(c.Context(), id, until); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Retention updated"), nil
}

// ── Uploads (session control) ─────────────────────────────

// InitUpload starts a resumable upload session.
func (h *Handler) InitUpload(c fuego.ContextWithBody[models.InitUploadRequest]) (*response.Envelope[models.InitUploadResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return nil, fuego.BadRequestError{Title: "invalid upload init payload", Err: err}
	}
	res, err := h.svc.InitUpload(c.Context(), uid, body)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*res, "Upload session created"), nil
}

// UploadStatus reports progress for resume.
func (h *Handler) UploadStatus(c fuego.ContextNoBody) (*response.Envelope[models.UploadStatusResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	st, err := h.svc.UploadStatus(c.Context(), uid, id)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(*st), nil
}

// CompleteUpload assembles the chunks into a stored file.
func (h *Handler) CompleteUpload(c fuego.ContextNoBody) (*response.Envelope[models.FileResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	f, err := h.svc.CompleteUpload(c.Context(), uid, id)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*f, "Upload complete"), nil
}

// AbortUpload cancels an upload session.
func (h *Handler) AbortUpload(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	if err := h.svc.AbortUpload(c.Context(), uid, id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Upload aborted"), nil
}

// ── shared helpers ────────────────────────────────────────

// ShareWithUser grants a specific internal user access to a file.
func (h *Handler) ShareWithUser(c fuego.ContextWithBody[models.ShareUserRequest]) (*response.Envelope[models.FileGrantResponse], error) {
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(err)
	}
	if err := utils.Validate(body); err != nil {
		return nil, handlers.Fail(err)
	}
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	grant, err := h.svc.ShareWithUser(c.Context(), uid, id, body.RecipientEmail, body.CanWrite)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*grant, "File shared"), nil
}

// ListGrants lists a file's internal recipients (owner only).
func (h *Handler) ListGrants(c fuego.ContextNoBody) (*response.Envelope[[]models.FileGrantResponse], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	grants, err := h.svc.ListFileGrants(c.Context(), uid, id)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(grants), nil
}

// RevokeGrant removes a user's access to a file (owner only).
func (h *Handler) RevokeGrant(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	rid, err := params.UUIDPath(c, "uid")
	if err != nil {
		return nil, err
	}
	if err := h.svc.RevokeUserShare(c.Context(), uid, id, rid); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Access removed"), nil
}

// SharedWithMe lists files other users have shared with the caller.
func (h *Handler) SharedWithMe(c fuego.ContextNoBody) (*response.Envelope[[]models.SharedFileResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	files, err := h.svc.ListSharedWithMe(c.Context(), uid)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(files), nil
}

func (h *Handler) idOnly(c fuego.ContextNoBody) (uuid.UUID, uuid.UUID, error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return uuid.Nil, uuid.Nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return uid, id, nil
}

func (h *Handler) idAndBody(c fuego.ContextWithBody[models.RenameRequest]) (uuid.UUID, uuid.UUID, models.RenameRequest, error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return uuid.Nil, uuid.Nil, models.RenameRequest{}, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return uuid.Nil, uuid.Nil, models.RenameRequest{}, err
	}
	body, err := c.Body()
	if err != nil {
		return uuid.Nil, uuid.Nil, models.RenameRequest{}, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return uuid.Nil, uuid.Nil, models.RenameRequest{}, fuego.BadRequestError{Title: "name is required"}
	}
	return uid, id, body, nil
}

func currentUserID(ctx context.Context) (uuid.UUID, error) {
	claims := jwt.GetClaimsFromContext(ctx)
	if claims == nil || claims.Sub == nil {
		return uuid.Nil, apperrors.ErrUnauthorized
	}
	return uuid.Parse(*claims.Sub)
}

// isAdmin reports whether the caller holds an administrative role.
func isAdmin(ctx context.Context) bool {
	claims := jwt.GetClaimsFromContext(ctx)
	return claims != nil && (claims.Role == "super_admin" || claims.Role == "admin")
}

// ── Common (organisation-wide) files ──────────────────────

// CommonList returns the organisation-wide Common files.
func (h *Handler) CommonList(c fuego.ContextNoBody) (*response.Envelope[[]models.CommonFileResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	limit := params.IntQueryDefault(c, "limit", 200)
	offset := params.IntQueryDefault(c, "offset", 0)
	files, total, err := h.svc.ListCommon(c.Context(), uid, isAdmin(c.Context()), limit, offset)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.Paginated(files, models.ListMeta{Total: total, Limit: limit, Offset: offset}), nil
}

// CommonBrowseResponse is a navigable level of the Common area.
type CommonBrowseResponse struct {
	Folders []models.FolderResponse     `json:"folders"`
	Files   []models.CommonFileResponse `json:"files"`
}

// CommonBrowse lists Common folders + files at a level (parent_id query; empty = root).
func (h *Handler) CommonBrowse(c fuego.ContextNoBody) (*response.Envelope[CommonBrowseResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	var parent *string
	if p := c.QueryParam("parent_id"); p != "" {
		parent = &p
	}
	folders, files, err := h.svc.ListCommonAt(c.Context(), uid, isAdmin(c.Context()), parent)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(CommonBrowseResponse{Folders: folders, Files: files}), nil
}

// CommonFolderCreate creates a folder in the Common area.
func (h *Handler) CommonFolderCreate(c fuego.ContextWithBody[models.CreateFolderRequest]) (*response.Envelope[models.FolderResponse], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	body, err := c.Body()
	if err != nil || body.Name == "" {
		return nil, fuego.BadRequestError{Title: "name is required"}
	}
	f, err := h.svc.CreateCommonFolder(c.Context(), uid, body.ParentID, body.Name)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*f, "Folder created"), nil
}

// MakeCommon shares the caller's file into the Common area.
func (h *Handler) MakeCommon(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.MakeCommon(c.Context(), uid, id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Shared to Common"), nil
}

// Inherited lists files transferred to the caller, grouped by source user.
func (h *Handler) Inherited(c fuego.ContextNoBody) (*response.Envelope[[]models.InheritedGroup], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	groups, err := h.svc.ListInheritedGrouped(c.Context(), uid)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(groups), nil
}

// KeepFile clears the inherited-pending flag (heir keeps the file).
func (h *Handler) KeepFile(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.KeepInherited(c.Context(), uid, id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "File kept"), nil
}

// CommonDelete removes a Common file (uploader or admin only).
func (h *Handler) CommonDelete(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteCommon(c.Context(), uid, isAdmin(c.Context()), id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Deleted from Common"), nil
}

func optionalUUID(s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
