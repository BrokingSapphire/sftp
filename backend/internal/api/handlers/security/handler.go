// Package security serves the audit anomaly alert feed for administrators.
package security

import (
	"time"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/params"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves /security.
type Handler struct {
	q   *sftpdb.Queries
	log logger.Logger
}

// NewHandler builds the security alerts handler.
func NewHandler(q *sftpdb.Queries, log logger.Logger) *Handler {
	return &Handler{q: q, log: log.Named("handler.security")}
}

// Alert is the public projection of a security alert.
type Alert struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	ActorEmail  string `json:"actor_email,omitempty"`
	Summary     string `json:"summary"`
	EventCount  int32  `json:"event_count"`
	WindowStart string `json:"window_start,omitempty"`
	WindowEnd   string `json:"window_end,omitempty"`
	Resolved    bool   `json:"resolved"`
	CreatedAt   string `json:"created_at"`
}

func toAlert(a sftpdb.SecurityAlert) Alert {
	out := Alert{
		ID: a.ID.String(), Type: a.Type, Severity: a.Severity, Summary: a.Summary,
		EventCount: a.EventCount, Resolved: a.Resolved,
	}
	if a.ActorEmail != nil {
		out.ActorEmail = *a.ActorEmail
	}
	if a.WindowStart.Valid {
		out.WindowStart = a.WindowStart.Time.Format(time.RFC3339)
	}
	if a.WindowEnd.Valid {
		out.WindowEnd = a.WindowEnd.Time.Format(time.RFC3339)
	}
	if a.CreatedAt.Valid {
		out.CreatedAt = a.CreatedAt.Time.Format(time.RFC3339)
	}
	return out
}

// List returns security alerts (unresolved first, newest first).
func (h *Handler) List(c fuego.ContextNoBody) (*response.Envelope[[]Alert], error) {
	rows, err := h.q.ListSecurityAlerts(c.Context(), sftpdb.ListSecurityAlertsParams{
		Limit: int32(params.IntQueryDefault(c, "limit", 100)), Offset: int32(params.IntQueryDefault(c, "offset", 0)),
	})
	if err != nil {
		return nil, handlers.Fail(err)
	}
	out := make([]Alert, 0, len(rows))
	for _, a := range rows {
		out = append(out, toAlert(a))
	}
	return response.OK(out), nil
}

// UnresolvedCount returns how many alerts are open.
func (h *Handler) UnresolvedCount(c fuego.ContextNoBody) (*response.Envelope[map[string]int64], error) {
	n, err := h.q.CountUnresolvedAlerts(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OK(map[string]int64{"unresolved": n}), nil
}

// Resolve marks an alert handled.
func (h *Handler) Resolve(c fuego.ContextNoBody) (*response.Envelope[response.Any], error) {
	id, err := params.UUIDPath(c, "id")
	if err != nil {
		return nil, err
	}
	var by *uuid.UUID
	if claims := jwt.GetClaimsFromContext(c.Context()); claims != nil && claims.Sub != nil {
		if uid, err := uuid.Parse(*claims.Sub); err == nil {
			by = &uid
		}
	}
	if err := h.q.ResolveSecurityAlert(c.Context(), sftpdb.ResolveSecurityAlertParams{ID: id, ResolvedBy: by}); err != nil {
		return nil, handlers.Fail(err)
	}
	return response.OKWithMessage[response.Any](nil, "Alert resolved"), nil
}
