// Package api wires HTTP routing and the server lifecycle.
package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// BaseURL is the versioned API prefix.
const BaseURL = "/api/v1"

// HttpServer wraps the standard-library server with lifecycle helpers.
type HttpServer struct {
	logger logger.Logger
	server *http.Server
}

// NewHttpServer builds the router and configures the HTTP server.
//
// WriteTimeout is intentionally 0 (disabled) because the service streams
// arbitrarily large files; per-handler timeouts guard the non-streaming routes.
func NewHttpServer(port int, deps Deps) *HttpServer {
	router := chi.NewRouter()
	SetupRoutes(router, deps)

	srv := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", port),
		Handler:           router,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	return &HttpServer{logger: deps.Logger, server: srv}
}

// Start begins serving; blocks until the server stops.
func (hs *HttpServer) Start() {
	hs.logger.Info("starting HTTP server", "addr", hs.server.Addr)
	if err := hs.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		hs.logger.Fatal("could not start HTTP server", "err", err)
	}
}

// Shutdown gracefully drains in-flight requests.
func (hs *HttpServer) Shutdown(ctx context.Context) error {
	hs.logger.Info("shutting down HTTP server")
	return hs.server.Shutdown(ctx)
}
