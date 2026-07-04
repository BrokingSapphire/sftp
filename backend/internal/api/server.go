package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/BrokingSapphire/sftp/backend/internal/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server wraps the HTTP server with graceful lifecycle helpers.
type Server struct {
	http *http.Server
	log  *zap.Logger
}

// NewServer builds an *http.Server from config and a handler.
func NewServer(cfg *config.Config, handler *gin.Engine, log *zap.Logger) *Server {
	return &Server{
		http: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:      handler,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		},
		log: log,
	}
}

// Start begins serving and blocks until the server stops.
func (s *Server) Start() error {
	s.log.Info("http server listening", zap.String("addr", s.http.Addr))
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully drains in-flight requests.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http server shutting down")
	return s.http.Shutdown(ctx)
}
