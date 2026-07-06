// Package api wires HTTP routing (Fuego) and the server lifecycle.
package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"github.com/rs/cors"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	m "sapphirebroking.com/sftp_service/internal/api/handlers/middleware"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// BaseURL is the versioned API prefix.
const BaseURL = "/api/v1"

var openAPIInfo = &openapi3.Info{
	Title:   "Sapphire SFTP Platform API",
	Version: "1.0.0",
	Description: "Self-hosted, on-premise enterprise file-transfer platform.\n\n" +
		"Exposes a REST API for authentication, file/folder management, sharing, " +
		"search, audit and administration, alongside a native SFTP-over-SSH endpoint.\n\n" +
		"**Auth**: send a platform-issued JWT as `Authorization: Bearer <token>`, or an " +
		"API key as `X-API-Key: <key>` for programmatic access.\n\n" +
		"**Errors**: failures are returned as RFC 7807 problem+json.",
	Contact: &openapi3.Contact{Name: "Sapphire Broking", Email: "tech@sapphirebroking.com"},
	License: &openapi3.License{Name: "MIT"},
}

var securitySchemes = openapi3.SecuritySchemes{
	"bearerAuth": &openapi3.SecuritySchemeRef{
		Value: openapi3.NewSecurityScheme().
			WithType("http").WithScheme("bearer").WithBearerFormat("JWT").
			WithDescription("Platform-issued access token. Send as: `Authorization: Bearer <token>`"),
	},
	"apiKeyAuth": &openapi3.SecuritySchemeRef{
		Value: openapi3.NewSecurityScheme().
			WithType("apiKey").WithIn("header").WithName("X-API-Key").
			WithDescription("Programmatic access key. Send as: `X-API-Key: <key>`"),
	},
}

// HttpServer wraps the Fuego server with lifecycle helpers.
type HttpServer struct {
	fuego  *fuego.Server
	port   int
	logger logger.Logger
}

// NewHttpServer builds and configures the Fuego server.
func NewHttpServer(port int, deps Deps) *HttpServer {
	handlers.SetDebugErrors(deps.DebugErrors)

	corsMW := cors.New(cors.Options{
		AllowedOrigins:   deps.CORSConfig.AllowedOrigins,
		AllowedMethods:   deps.CORSConfig.AllowedMethods,
		AllowedHeaders:   deps.CORSConfig.AllowedHeaders,
		ExposedHeaders:   deps.CORSConfig.ExposedHeaders,
		AllowCredentials: deps.CORSConfig.AllowCredentials,
		MaxAge:           deps.CORSConfig.MaxAgeSecs,
	}).Handler

	s := fuego.NewServer(
		fuego.WithAddr(fmt.Sprintf("0.0.0.0:%d", port)),
		fuego.WithoutAutoGroupTags(),
		fuego.WithEngineOptions(
			fuego.WithErrorHandler(handlers.ErrorHandler),
			fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
				Disabled:         !deps.DebugErrors,
				DisableLocalSave: !deps.DebugErrors,
				JSONFilePath:     "docs/openapi.json",
				PrettyFormatJSON: true,
				Info:             openAPIInfo,
			}),
		),
		fuego.WithGlobalMiddlewares(
			m.RequestID,
			m.InjectLogger(deps.Logger),
			m.Recover(deps.Logger),
			m.AccessLog(deps.Logger),
			m.RealIP,
			m.SecurityHeaders,
			m.AuditLog(deps.Recorder),
			corsMW,
		),
	)

	// Large files stream for arbitrary durations, so the whole-request read and
	// write timeouts MUST be disabled (Fuego defaults both to 30s, which would
	// abort any upload/download longer than 30 seconds). ReadHeaderTimeout still
	// guards against slow-header (slowloris) attacks.
	s.Server.ReadTimeout = 0
	s.Server.WriteTimeout = 0
	s.Server.ReadHeaderTimeout = 30 * time.Second
	s.Server.IdleTimeout = 120 * time.Second
	s.Server.MaxHeaderBytes = 1 << 20

	RegisterRoutes(s, deps)

	return &HttpServer{fuego: s, port: port, logger: deps.Logger.Named("http.server")}
}

// Handler exposes the underlying mux (useful for tests).
func (hs *HttpServer) Handler() http.Handler { return hs.fuego.Mux }

// Start begins serving; blocks until the server stops.
func (hs *HttpServer) Start() {
	hs.logger.Info("starting HTTP server", "port", hs.port)
	if err := hs.fuego.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		hs.logger.Error("http server stopped", "err", err)
	}
}

// Shutdown gracefully drains in-flight requests.
func (hs *HttpServer) Shutdown(ctx context.Context) error {
	hs.logger.Info("shutting down HTTP server")
	return hs.fuego.Shutdown(ctx)
}
