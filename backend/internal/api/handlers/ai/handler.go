// Package ai wires the AI (semantic search + ask-your-files) HTTP handlers.
package ai

import (
	"context"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/apperrors"
	aisvc "sapphirebroking.com/sftp_service/internal/service/ai"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves /ai.
type Handler struct {
	svc *aisvc.Service
	log logger.Logger
}

// NewHandler builds the AI handler.
func NewHandler(svc *aisvc.Service, log logger.Logger) *Handler {
	return &Handler{svc: svc, log: log.Named("handler.ai")}
}

// AskRequest is a natural-language question over the caller's files.
type AskRequest struct {
	Question string `json:"question" validate:"required"`
}

// Status reports whether AI is enabled (used to gate the UI).
func (h *Handler) Status(c fuego.ContextNoBody) (*response.Envelope[map[string]bool], error) {
	return response.OK(map[string]bool{"enabled": h.svc.Enabled()}), nil
}

// Ask answers a question using the caller's documents.
func (h *Handler) Ask(c fuego.ContextWithBody[AskRequest]) (*response.Envelope[aisvc.Answer], error) {
	if !h.svc.Enabled() {
		return nil, fuego.HTTPError{Title: "AI features are not enabled", Status: 503}
	}
	uid, err := currentUser(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	body, err := c.Body()
	if err != nil || body.Question == "" {
		return nil, fuego.BadRequestError{Title: "question is required"}
	}
	ans, err := h.svc.Ask(c.Context(), uid, body.Question)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(*ans), nil
}

// Search runs semantic search over the caller's files.
func (h *Handler) Search(c fuego.ContextNoBody) (*response.Envelope[[]aisvc.Hit], error) {
	if !h.svc.Enabled() {
		return nil, fuego.HTTPError{Title: "AI features are not enabled", Status: 503}
	}
	uid, err := currentUser(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	q := c.QueryParam("q")
	if q == "" {
		return nil, fuego.BadRequestError{Title: "q is required"}
	}
	hits, err := h.svc.SemanticSearch(c.Context(), uid, q)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(hits), nil
}

func currentUser(ctx context.Context) (uuid.UUID, error) {
	claims := jwt.GetClaimsFromContext(ctx)
	if claims == nil || claims.Sub == nil {
		return uuid.Nil, apperrors.ErrUnauthorized
	}
	return uuid.Parse(*claims.Sub)
}
