// Package middleware holds chi-compatible HTTP middleware.
package middleware

import (
	"net/http"
	"regexp"

	"github.com/google/uuid"
	"sapphirebroking.com/sftp_service/pkg/reqctx"
)

// validRequestID matches a canonical UUID.
var validRequestID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// RequestID assigns a correlation ID (honouring a valid inbound X-Request-ID)
// and echoes it back on the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validRequestID.MatchString(id) {
			id = uuid.NewString()
		}
		ctx := reqctx.NewContextWithRequestID(r.Context(), id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
