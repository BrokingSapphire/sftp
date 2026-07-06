// Package notification serves the per-user notification feed.
package notification

import (
	"context"
	"time"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/params"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves /notifications.
type Handler struct {
	q   *sftpdb.Queries
	log logger.Logger
}

// NewHandler constructs the notification Handler.
func NewHandler(q *sftpdb.Queries, log logger.Logger) *Handler {
	return &Handler{q: q, log: log.Named("handler.notification")}
}

// Response is the public projection of a notification.
type Response struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Link      string `json:"link,omitempty"`
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}

// List returns the caller's notifications.
func (h *Handler) List(c fuego.ContextNoBody) (*response.Envelope[[]Response], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	rows, err := h.q.ListNotifications(c.Context(), sftpdb.ListNotificationsParams{
		UserID: uid, Limit: int32(params.IntQueryDefault(c, "limit", 30)), Offset: int32(params.IntQueryDefault(c, "offset", 0)),
	})
	if err != nil {
		return nil, handlers.Fail(err)
	}
	out := make([]Response, 0, len(rows))
	for _, n := range rows {
		r := Response{ID: n.ID.String(), Type: n.Type, Title: n.Title, Body: n.Body, IsRead: n.IsRead}
		if n.Link != nil {
			r.Link = *n.Link
		}
		if n.CreatedAt.Valid {
			r.CreatedAt = n.CreatedAt.Time.Format(time.RFC3339)
		}
		out = append(out, r)
	}
	return response.OK(out), nil
}

// UnreadCount returns the number of unread notifications.
func (h *Handler) UnreadCount(c fuego.ContextNoBody) (*response.Envelope[map[string]int64], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	n, err := h.q.CountUnreadNotifications(c.Context(), uid)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(map[string]int64{"unread": n}), nil
}

// MarkRead marks one notification read.
func (h *Handler) MarkRead(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	if err := h.q.MarkNotificationRead(c.Context(), sftpdb.MarkNotificationReadParams{ID: id, UserID: uid}); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Marked read"), nil
}

// MarkAllRead marks all the caller's notifications read.
func (h *Handler) MarkAllRead(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	uid, err := currentUserID(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	if err := h.q.MarkAllNotificationsRead(c.Context(), uid); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "All marked read"), nil
}

func currentUserID(ctx context.Context) (uuid.UUID, error) {
	claims := jwt.GetClaimsFromContext(ctx)
	if claims == nil || claims.Sub == nil {
		return uuid.Nil, apperrors.ErrUnauthorized
	}
	return uuid.Parse(*claims.Sub)
}
