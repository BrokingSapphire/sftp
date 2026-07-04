// Package user wires the user-administration HTTP handlers.
package user

import (
	"context"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/params"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/apperrors"
	models "sapphirebroking.com/sftp_service/internal/models/user"
	usersvc "sapphirebroking.com/sftp_service/internal/service/user"
	"sapphirebroking.com/sftp_service/internal/utils"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves the /users endpoints.
type Handler struct {
	svc *usersvc.Service
	log logger.Logger
}

// NewHandler constructs the user Handler.
func NewHandler(svc *usersvc.Service, log logger.Logger) *Handler {
	return &Handler{svc: svc, log: log.Named("handler.user")}
}

// Create provisions a new user.
func (h *Handler) Create(c fuego.ContextWithBody[models.CreateRequest]) (*response.Envelope[models.Response], error) {
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return nil, fuego.BadRequestError{Title: "invalid user payload", Err: err}
	}
	creator, _ := currentUserID(c.Context())
	user, err := h.svc.Create(c.Context(), body, creator)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*user, "User created"), nil
}

// List returns a paginated list of users.
func (h *Handler) List(c fuego.ContextNoBody) (*response.Envelope[[]models.Response], error) {
	limit := params.IntQueryDefault(c, "limit", 50)
	offset := params.IntQueryDefault(c, "offset", 0)
	users, total, err := h.svc.List(c.Context(), limit, offset)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.Paginated(users, models.ListMeta{Total: total, Limit: limit, Offset: offset}), nil
}

// Get returns one user by ID.
func (h *Handler) Get(c fuego.ContextNoBody) (*response.Envelope[models.Response], error) {
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	user, err := h.svc.Get(c.Context(), id)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(*user), nil
}

// Update changes profile fields.
func (h *Handler) Update(c fuego.ContextWithBody[models.UpdateRequest]) (*response.Envelope[models.Response], error) {
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	user, err := h.svc.Update(c.Context(), id, body)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*user, "User updated"), nil
}

// SetRole reassigns a user's role.
func (h *Handler) SetRole(c fuego.ContextWithBody[models.SetRoleRequest]) (*response.Envelope[response.Any], error) {
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return nil, fuego.BadRequestError{Title: "role is required"}
	}
	if err := h.svc.SetRole(c.Context(), id, body.RoleSlug); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Role updated"), nil
}

// SetQuota changes a user's storage quota.
func (h *Handler) SetQuota(c fuego.ContextWithBody[models.SetQuotaRequest]) (*response.Envelope[response.Any], error) {
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := h.svc.SetQuota(c.Context(), id, body.StorageQuota); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Quota updated"), nil
}

// SetActive enables or disables a user.
func (h *Handler) SetActive(c fuego.ContextWithBody[models.SetActiveRequest]) (*response.Envelope[response.Any], error) {
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := h.svc.SetActive(c.Context(), id, body.IsActive); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "User status updated"), nil
}

// ResetPassword sets a new password for a user.
func (h *Handler) ResetPassword(c fuego.ContextWithBody[models.ResetPasswordRequest]) (*response.Envelope[response.Any], error) {
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return nil, fuego.BadRequestError{Title: "invalid password payload", Err: err}
	}
	if err := h.svc.ResetPassword(c.Context(), id, body.NewPassword); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Password reset"), nil
}

// Delete soft-deletes a user.
func (h *Handler) Delete(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(c.Context(), id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "User deleted"), nil
}

// ListRoles returns all roles with permissions.
func (h *Handler) ListRoles(c fuego.ContextNoBody) (*response.Envelope[[]usersvc.RoleInfo], error) {
	roles, err := h.svc.ListRoles(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(roles), nil
}

func currentUserID(ctx context.Context) (uuid.UUID, error) {
	claims := jwt.GetClaimsFromContext(ctx)
	if claims == nil || claims.Sub == nil {
		return uuid.Nil, apperrors.ErrUnauthorized
	}
	return uuid.Parse(*claims.Sub)
}
