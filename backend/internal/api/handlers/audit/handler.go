// Package audit wires the audit-log and telemetry HTTP handlers.
package audit

import (
	"time"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/params"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/apperrors"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	models "sapphirebroking.com/sftp_service/internal/models/audit"
	auditsvc "sapphirebroking.com/sftp_service/internal/service/audit"
	"sapphirebroking.com/sftp_service/internal/utils"
	"sapphirebroking.com/sftp_service/pkg/headers"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves /audit and /activity endpoints.
type Handler struct {
	rec *auditsvc.Recorder
	log logger.Logger
}

// NewHandler constructs the audit Handler.
func NewHandler(rec *auditsvc.Recorder, log logger.Logger) *Handler {
	return &Handler{rec: rec, log: log.Named("handler.audit")}
}

// Telemetry ingests a UI interaction event (click/view/navigate/…).
func (h *Handler) Telemetry(c fuego.ContextWithBody[models.TelemetryRequest]) (*response.Envelope[response.Any], error) {
	body, err := c.Body()
	if err != nil {
		return nil, handlers.Fail(apperrors.ErrInvalidRequest)
	}
	if err := utils.Validate(body); err != nil {
		return nil, fuego.BadRequestError{Title: "event_type is required"}
	}
	r := c.Request()
	entry := auditsvc.ActivityEntry{
		EventType: body.EventType, Element: body.Element, Path: body.Path,
		IP: headers.GetClientIP(r), UserAgent: r.UserAgent(), Metadata: body.Metadata,
	}
	if claims := jwt.GetClaimsFromContext(c.Context()); claims != nil && claims.Sub != nil {
		if id, err := uuid.Parse(*claims.Sub); err == nil {
			entry.UserID = &id
		}
	}
	h.rec.RecordActivity(c.Context(), entry)
	return response.OKWithMessage[response.Any](nil, "recorded"), nil
}

// List returns recent audit-log entries.
func (h *Handler) List(c fuego.ContextNoBody) (*response.Envelope[[]models.LogResponse], error) {
	limit := params.IntQueryDefault(c, "limit", 100)
	offset := params.IntQueryDefault(c, "offset", 0)
	rows, err := h.rec.List(c.Context(), limit, offset)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	out := make([]models.LogResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, toLogResponse(r))
	}
	return response.Paginated(out, models.ListMeta{Limit: limit, Offset: offset}), nil
}

func toLogResponse(a sftpdb.AuditLog) models.LogResponse {
	r := models.LogResponse{
		ID: a.ID, Action: a.Action, Category: a.Category, Result: a.Result,
	}
	if a.ActorID != nil {
		r.ActorID = a.ActorID.String()
	}
	r.ActorEmail = deref(a.ActorEmail)
	r.ObjectType = deref(a.ObjectType)
	r.ObjectID = deref(a.ObjectID)
	r.ObjectName = deref(a.ObjectName)
	r.Browser = deref(a.Browser)
	r.OS = deref(a.Os)
	if a.IpAddress != nil {
		r.IPAddress = a.IpAddress.String()
	}
	if a.CreatedAt.Valid {
		r.CreatedAt = a.CreatedAt.Time.Format(time.RFC3339)
	}
	return r
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
