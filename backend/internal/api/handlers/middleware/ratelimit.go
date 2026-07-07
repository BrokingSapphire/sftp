package middleware

import (
	"net/http"
	"strconv"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/pkg/headers"
	"sapphirebroking.com/sftp_service/pkg/ratelimit"
)

// RateLimit rejects requests from a client (by IP) once its token bucket is
// empty, returning 429 with a Retry-After hint. A nil limiter is a no-op.
func RateLimit(l *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if l == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.Allow(headers.GetClientIP(r)) {
				w.Header().Set("Retry-After", strconv.Itoa(1))
				handlers.WriteProblem(w, r, http.StatusTooManyRequests, "too many requests — slow down")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
