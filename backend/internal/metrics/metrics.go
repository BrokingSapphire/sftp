// Package metrics exposes Prometheus request metrics and a scrape handler.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sftp",
			Name:      "http_requests_total",
			Help:      "Total HTTP requests by method and status class.",
		},
		[]string{"method", "status"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "sftp",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency by method.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method"},
	)
	inFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "sftp",
		Name:      "http_requests_in_flight",
		Help:      "In-flight HTTP requests.",
	})
)

func init() {
	prometheus.MustRegister(requestsTotal, requestDuration, inFlight)
}

// Handler serves the Prometheus scrape endpoint.
func Handler() http.Handler { return promhttp.Handler() }

// Middleware records request count, latency, and in-flight gauge. Labels are
// low-cardinality (method + status class) to keep the series count bounded.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't measure the scrape endpoint itself.
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		inFlight.Inc()
		defer inFlight.Dec()

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(sw, r)

		requestDuration.WithLabelValues(r.Method).Observe(time.Since(start).Seconds())
		requestsTotal.WithLabelValues(r.Method, statusClass(sw.status)).Inc()
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wrote {
		w.status = code
		w.wrote = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	w.wrote = true
	return w.ResponseWriter.Write(b)
}

// Flush/Hijack pass-through would go here if needed; ServeContent uses Write.

func statusClass(code int) string {
	return strconv.Itoa(code/100) + "xx"
}
