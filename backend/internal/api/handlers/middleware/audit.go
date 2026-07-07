package middleware

import (
	"net/http"
	"strings"

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
			case http.MethodGet:
				// Reads are normally not audited, but downloads (data leaving the
				// system — including the Common area) must always be logged.
				if !isDownloadPath(r.URL.Path) {
					return
				}
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
			action, category, objectID := deriveAction(r.Method, r.URL.Path)
			e := audit.Entry{
				Action:    action,
				Category:  category,
				ObjectID:  objectID,
				Result:    result,
				IP:        headers.GetClientIP(r),
				UserAgent: r.UserAgent(),
				Browser:   dev.Browser,
				OS:        dev.OS,
				RequestID: reqctx.GetRequestID(r.Context()),
				Metadata:  map[string]any{"status": sr.status, "method": r.Method, "path": r.URL.Path},
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

// deriveAction turns an HTTP method + path into a semantic (action, category,
// object_id) triple so the audit trail reads like events ("file.upload",
// "share.create") rather than raw routes.
func deriveAction(method, path string) (action, category, objectID string) {
	p := strings.TrimPrefix(path, "/api/v1/")
	segs := strings.Split(strings.Trim(p, "/"), "/")
	if len(segs) == 0 || segs[0] == "" {
		return method + " " + path, "http", ""
	}
	resource := segs[0]

	// Capture the first UUID-looking segment as the object id.
	for _, s := range segs {
		if _, err := uuid.Parse(s); err == nil {
			objectID = s
			break
		}
	}

	// Named sub-action = last non-UUID segment (e.g. "rename", "trash").
	sub := ""
	for i := len(segs) - 1; i >= 1; i-- {
		if _, err := uuid.Parse(segs[i]); err != nil && !isIndex(segs[i]) {
			sub = segs[i]
			break
		}
	}

	category = singular(resource)
	switch resource {
	case "auth":
		category = "auth"
		action = "auth." + orDefault(sub, "request")
	case "files", "folders":
		verb := map[string]string{
			http.MethodPost: "create", http.MethodPut: "update",
			http.MethodPatch: "update", http.MethodDelete: "delete",
		}[method]
		if sub != "" && sub != resource {
			verb = sub
		}
		action = category + "." + orDefault(verb, "access")
	case "uploads":
		category = "file"
		action = "file.upload." + orDefault(sub, "session")
	case "shares", "share":
		category = "share"
		if sub == "download" {
			action = "share.download"
		} else {
			action = "share." + map[string]string{http.MethodPost: "create", http.MethodDelete: "revoke"}[method]
		}
	case "users":
		category = "user"
		action = "user." + orDefault(sub, map[string]string{http.MethodPost: "create", http.MethodPatch: "update", http.MethodDelete: "delete"}[method])
	case "api-keys":
		category = "apikey"
		action = "apikey." + map[string]string{http.MethodPost: "create", http.MethodDelete: "revoke"}[method]
	case "activity":
		category, action = "activity", "activity.track"
	case "admin":
		category = "backup"
		if sub == "restore" {
			action = "backup.restore"
		} else {
			action = "backup.run"
		}
	default:
		action = category + "." + strings.ToLower(method)
	}
	if strings.HasSuffix(action, ".") {
		action += strings.ToLower(method)
	}
	return action, category, objectID
}

func isIndex(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func singular(s string) string {
	return strings.TrimSuffix(s, "s")
}

// isDownloadPath reports whether a GET path represents content leaving the
// system (a download), which must be audited even though reads generally aren't.
func isDownloadPath(path string) bool {
	return strings.HasSuffix(path, "/download")
}
