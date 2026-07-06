// Package backup exposes super-admin-only backup/restore endpoints.
package backup

import (
	"context"

	"github.com/go-fuego/fuego"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/apperrors"
	backupsvc "sapphirebroking.com/sftp_service/internal/service/backup"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves /admin/backup and /admin/restore.
type Handler struct {
	svc *backupsvc.Service
	log logger.Logger
}

// NewHandler builds the backup handler.
func NewHandler(svc *backupsvc.Service, log logger.Logger) *Handler {
	return &Handler{svc: svc, log: log.Named("handler.backup")}
}

// TargetRequest carries the backup/restore target directory (a path the server
// can write to — e.g. a mounted removable disk).
type TargetRequest struct {
	TargetPath string `json:"target_path" validate:"required"`
}

// Run performs a backup (full first time, incremental thereafter).
func (h *Handler) Run(c fuego.ContextWithBody[TargetRequest]) (*response.Envelope[backupsvc.Result], error) {
	if err := requireSuperAdmin(c.Context()); err != nil {
		return nil, handlers.Fail(err)
	}
	body, err := c.Body()
	if err != nil || body.TargetPath == "" {
		return nil, fuego.BadRequestError{Title: "target_path is required"}
	}
	res, err := h.svc.Run(c.Context(), body.TargetPath)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	msg := "Backup complete"
	if res.Mode == "none" {
		msg = "Already up to date — nothing to back up"
	}
	return response.OKWithMessage(*res, msg), nil
}

// Status reports the backups already on a target (query: target_path).
func (h *Handler) Status(c fuego.ContextNoBody) (*response.Envelope[backupsvc.Status], error) {
	if err := requireSuperAdmin(c.Context()); err != nil {
		return nil, handlers.Fail(err)
	}
	target := c.QueryParam("target_path")
	if target == "" {
		return nil, fuego.BadRequestError{Title: "target_path is required"}
	}
	st, err := h.svc.Status(c.Context(), target)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(*st), nil
}

// Restore rebuilds files from a target's archives.
func (h *Handler) Restore(c fuego.ContextWithBody[TargetRequest]) (*response.Envelope[backupsvc.RestoreResult], error) {
	if err := requireSuperAdmin(c.Context()); err != nil {
		return nil, handlers.Fail(err)
	}
	body, err := c.Body()
	if err != nil || body.TargetPath == "" {
		return nil, fuego.BadRequestError{Title: "target_path is required"}
	}
	res, err := h.svc.Restore(c.Context(), body.TargetPath)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*res, "Restore complete"), nil
}

func requireSuperAdmin(ctx context.Context) error {
	claims := jwt.GetClaimsFromContext(ctx)
	if claims == nil || claims.Sub == nil {
		return apperrors.ErrUnauthorized
	}
	if claims.Role != "super_admin" {
		return apperrors.ErrForbidden
	}
	return nil
}
