// Package auth wires the authentication HTTP handlers.
package auth

import (
	"context"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/apperrors"
	models "sapphirebroking.com/sftp_service/internal/models/auth"
	authsvc "sapphirebroking.com/sftp_service/internal/service/auth"
	"sapphirebroking.com/sftp_service/internal/utils"
	"sapphirebroking.com/sftp_service/pkg/headers"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves the /auth endpoints.
type Handler struct {
	svc *authsvc.Service
	log logger.Logger
}

// NewHandler constructs the auth Handler.
func NewHandler(svc *authsvc.Service, log logger.Logger) *Handler {
	return &Handler{svc: svc, log: log.Named("handler.auth")}
}

// currentUserID extracts and parses the authenticated user's ID from claims.
func currentUserID(ctx context.Context) (uuid.UUID, error) {
	claims := jwt.GetClaimsFromContext(ctx)
	if claims == nil || claims.Sub == nil {
		return uuid.Nil, apperrors.ErrUnauthorized
	}
	id, err := uuid.Parse(*claims.Sub)
	if err != nil {
		return uuid.Nil, apperrors.ErrUnauthorized
	}
	return id, nil
}

// Login authenticates a user and returns a token pair.
func (h *Handler) Login(c fuego.ContextWithBody[models.LoginRequest]) (*response.Envelope[models.TokenPair], error) {
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return nil, fuego.BadRequestError{Title: "invalid credentials payload", Err: err}
	}
	r := c.Request()
	pair, err := h.svc.Login(c.Context(), body, authsvc.RequestMeta{
		IP:        headers.GetClientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*pair, "Login successful"), nil
}

// Refresh rotates a refresh token.
func (h *Handler) Refresh(c fuego.ContextWithBody[models.RefreshRequest]) (*response.Envelope[models.TokenPair], error) {
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return nil, fuego.BadRequestError{Title: "refresh_token is required"}
	}
	r := c.Request()
	pair, err := h.svc.Refresh(c.Context(), body.RefreshToken, authsvc.RequestMeta{
		IP:        headers.GetClientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*pair, "Token refreshed"), nil
}

// Logout revokes the given refresh token.
func (h *Handler) Logout(c fuego.ContextWithBody[models.LogoutRequest]) (*response.Envelope[response.Any], error) {
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := h.svc.Logout(c.Context(), body.RefreshToken); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Logged out"), nil
}

// Me returns the authenticated user's profile.
func (h *Handler) Me(c fuego.ContextNoBody) (*response.Envelope[models.UserInfo], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	info, err := h.svc.Me(c.Context(), uid)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(*info), nil
}

// SetLanguage persists the user's preferred UI language so it follows them
// across devices.
func (h *Handler) SetLanguage(c fuego.ContextWithBody[models.LanguageRequest]) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := h.svc.SetLanguage(c.Context(), uid, body.Language); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Language updated"), nil
}

// ChangePassword updates the authenticated user's password.
func (h *Handler) ChangePassword(c fuego.ContextWithBody[models.ChangePasswordRequest]) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return nil, fuego.BadRequestError{Title: "invalid password payload", Err: err}
	}
	if err := h.svc.ChangePassword(c.Context(), uid, body); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Password changed; please sign in again"), nil
}
