package middleware

import (
	"net/http"
	"time"

	"sapphirebroking.com/sftp_service/pkg/headers"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// statusRecorder captures the response status code and bytes written.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	n, err := sr.ResponseWriter.Write(b)
	sr.bytes += n
	return n, err
}

// InjectLogger attaches a request-scoped child logger (enriched with
// request_id) to the context. Must run after RequestID.
func InjectLogger(root logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			child := root.WithContext(r.Context())
			ctx := logger.NewContext(r.Context(), child)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AccessLog logs one structured line per completed request. Must run after
// InjectLogger. Server errors log at Error, client errors at Warn.
func AccessLog(fallback logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.FromContext(r.Context(), fallback)
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rec, r)

			fields := []interface{}{
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"latency_ms", time.Since(start).Milliseconds(),
				"bytes", rec.bytes,
				"ip", headers.GetClientIP(r),
				"user_agent", r.UserAgent(),
			}
			switch {
			case rec.status >= 500:
				log.Error("request completed", fields...)
			case rec.status >= 400:
				log.Warn("request completed", fields...)
			default:
				log.Info("request completed", fields...)
			}
		})
	}
}
