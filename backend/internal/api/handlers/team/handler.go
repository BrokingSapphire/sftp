// Package team wires the Team Spaces HTTP handlers.
package team

import (
	"context"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/params"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/apperrors"
	models "sapphirebroking.com/sftp_service/internal/models/team"
	teamsvc "sapphirebroking.com/sftp_service/internal/service/team"
	"sapphirebroking.com/sftp_service/internal/utils"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves /teams.
type Handler struct {
	svc *teamsvc.Service
	log logger.Logger
}

// NewHandler builds the team Handler.
func NewHandler(svc *teamsvc.Service, log logger.Logger) *Handler {
	return &Handler{svc: svc, log: log.Named("handler.team")}
}

// List returns the caller's teams.
func (h *Handler) List(c fuego.ContextNoBody) (*response.Envelope[[]models.TeamResponse], error) {
	uid, err := caller(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	teams, err := h.svc.ListForUser(c.Context(), uid)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(teams), nil
}

// Create makes a new team (caller becomes owner).
func (h *Handler) Create(c fuego.ContextWithBody[models.CreateRequest]) (*response.Envelope[models.TeamResponse], error) {
	uid, err := caller(c.Context())
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
	t, err := h.svc.Create(c.Context(), uid, body)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*t, "Team created"), nil
}

// Get returns a team.
func (h *Handler) Get(c fuego.ContextNoBody) (*response.Envelope[models.TeamResponse], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	t, err := h.svc.Get(c.Context(), uid, id)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(*t), nil
}

// Update edits a team.
func (h *Handler) Update(c fuego.ContextWithBody[models.CreateRequest]) (*response.Envelope[response.Any], error) {
	uid, err := caller(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, _ := c.Body()
	if err := h.svc.Update(c.Context(), uid, id, body); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Team updated"), nil
}

// Delete removes a team.
func (h *Handler) Delete(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(c.Context(), uid, id); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Team deleted"), nil
}

// Members lists a team's members.
func (h *Handler) Members(c fuego.ContextNoBody) (*response.Envelope[[]models.MemberResponse], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	members, err := h.svc.ListMembers(c.Context(), uid, id)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(members), nil
}

// AddMember adds a member by email.
func (h *Handler) AddMember(c fuego.ContextWithBody[models.AddMemberRequest]) (*response.Envelope[models.MemberResponse], error) {
	uid, err := caller(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	body, err := c.Body()
	if err != nil || body.Email == "" {
		return nil, fuego.BadRequestError{Title: "email is required"}
	}
	mem, err := h.svc.AddMember(c.Context(), uid, id, body.Email, body.Role)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage(*mem, "Member added"), nil
}

// RemoveMember removes a member.
func (h *Handler) RemoveMember(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, id, err := h.idOnly(c)
	if err != nil {
		return nil, err
	}
	target, err := params.UUIDPath(c, "uid")
	if err != nil {
		return nil, err
	}
	if err := h.svc.RemoveMember(c.Context(), uid, id, target); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Member removed"), nil
}

func (h *Handler) idOnly(c fuego.ContextNoBody) (uuid.UUID, uuid.UUID, error) {
	uid, err := caller(c.Context())
	if err != nil {
		return uuid.Nil, uuid.Nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return uid, id, nil
}

func caller(ctx context.Context) (uuid.UUID, error) {
	claims := jwt.GetClaimsFromContext(ctx)
	if claims == nil || claims.Sub == nil {
		return uuid.Nil, apperrors.ErrUnauthorized
	}
	return uuid.Parse(*claims.Sub)
}
