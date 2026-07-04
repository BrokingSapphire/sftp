package handlers

import (
	"context"
	"runtime"
	"time"

	"github.com/go-fuego/fuego"
	"github.com/jackc/pgx/v5/pgxpool"

	"sapphirebroking.com/sftp_service/internal/api/response"
)

var startedAt = time.Now()

// HealthResponse is the bare liveness body (non-enveloped so probes can match
// a top-level "status" field).
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// InfoResponse holds build/runtime metadata.
type InfoResponse struct {
	Version    string `json:"version"`
	Uptime     string `json:"uptime"`
	GoVersion  string `json:"go_version"`
	Goroutines int    `json:"goroutines"`
	NumCPU     int    `json:"num_cpu"`
	StartedAt  string `json:"started_at"`
}

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
func (h *HealthHandler) Live(_ fuego.ContextNoBody) (*HealthResponse, error) {
	return &HealthResponse{Status: "ok", Timestamp: time.Now().UnixMilli()}, nil
}

// Ready verifies the database is reachable.
func (h *HealthHandler) Ready(c fuego.ContextNoBody) (*HealthResponse, error) {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		return nil, fuego.HTTPError{Status: 503, Title: "database unreachable"}
	}
	return &HealthResponse{Status: "ready", Timestamp: time.Now().UnixMilli()}, nil
}

// Info returns build/runtime metadata wrapped in the standard envelope.
func (h *HealthHandler) Info(_ fuego.ContextNoBody) (*response.Envelope[InfoResponse], error) {
	return response.OK(InfoResponse{
		Version:    h.version,
		Uptime:     time.Since(startedAt).String(),
		GoVersion:  runtime.Version(),
		Goroutines: runtime.NumGoroutine(),
		NumCPU:     runtime.NumCPU(),
		StartedAt:  startedAt.UTC().Format(time.RFC3339),
	}), nil
}
