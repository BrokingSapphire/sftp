// Package handlers implements HTTP request handlers.
package handlers

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"sapphirebroking.com/sftp_service/internal/httpresponse"
)

var startedAt = time.Now()

// HealthHandler serves liveness, readiness and info probes.
type HealthHandler struct {
	pool    *pgxpool.Pool
	version string
}

// NewHealthHandler constructs a HealthHandler.
func NewHealthHandler(pool *pgxpool.Pool, version string) *HealthHandler {
	return &HealthHandler{pool: pool, version: version}
}

// Live is a cheap liveness probe.
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	httpresponse.NewResponse(w, r).OK(map[string]any{"status": "alive"})
}

// Ready verifies the database is reachable.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		httpresponse.NewResponse(w, r).
			Error(0, httpresponse.ErrTypeInternalServer, "database unreachable").
			StatusCode(http.StatusServiceUnavailable).Send()
		return
	}
	httpresponse.NewResponse(w, r).OK(map[string]any{"status": "ready", "database": "ok"})
}

// Info returns build/runtime metadata.
func (h *HealthHandler) Info(w http.ResponseWriter, r *http.Request) {
	httpresponse.NewResponse(w, r).OK(map[string]any{
		"version":    h.version,
		"uptime":     time.Since(startedAt).String(),
		"go_version": runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
		"num_cpu":    runtime.NumCPU(),
		"started_at": startedAt.UTC().Format(time.RFC3339),
	})
}
