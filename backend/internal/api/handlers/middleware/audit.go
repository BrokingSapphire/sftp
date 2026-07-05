package middleware

import (
	"net/http"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/service/audit"
	"sapphirebroking.com/sftp_service/pkg/headers"
	"sapphirebroking.com/sftp_service/pkg/reqctx"
)

// AuditLog records every state-changing request (POST/PUT/PATCH/DELETE) to the
// compliance audit trail, capturing actor, IP, device, action and outcome.
func AuditLog(rec *audit.Recorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, holder := withActorHolder(r.Context())
			r = r.WithContext(ctx)
			sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sr, r)

			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			default:
				return
			}

			result := audit.ResultSuccess
			switch {
			case sr.status == http.StatusForbidden || sr.status == http.StatusUnauthorized:
				result = audit.ResultDenied
			case sr.status >= 400:
				result = audit.ResultFailure
			}

			dev := headers.ParseDevice(r.UserAgent())
			e := audit.Entry{
				Action:    r.Method + " " + r.URL.Path,
				Category:  "http",
				Result:    result,
				IP:        headers.GetClientIP(r),
				UserAgent: r.UserAgent(),
				Browser:   dev.Browser,
				OS:        dev.OS,
				RequestID: reqctx.GetRequestID(r.Context()),
				Metadata:  map[string]any{"status": sr.status},
			}
			if holder.claims != nil {
				e.ActorEmail = holder.claims.Email
				if holder.claims.Sub != nil {
					if id, err := uuid.Parse(*holder.claims.Sub); err == nil {
						e.ActorID = &id
					}
				}
			}
			rec.Record(e)
		})
	}
}
