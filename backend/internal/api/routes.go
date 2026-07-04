package api

import (
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	m "sapphirebroking.com/sftp_service/internal/api/handlers/middleware"
	"sapphirebroking.com/sftp_service/internal/config"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Deps carries the dependencies needed to build the router. Feature handlers
// are added here as later phases land (auth, users, files, shares, ...).
type Deps struct {
	Config        *config.Config
	Logger        logger.Logger
	HealthHandler *handlers.HealthHandler
}

// SetupRoutes registers global middleware and all routes on the router.
func SetupRoutes(router *chi.Mux, deps Deps) {
	router.Use(m.RequestID)
	router.Use(chimw.RealIP)
	router.Use(m.InjectLogger(deps.Logger))
	router.Use(m.AccessLog(deps.Logger))
	router.Use(m.SecurityHeaders)
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   deps.Config.CORS.AllowedOrigins,
		AllowedMethods:   deps.Config.CORS.AllowedMethods,
		AllowedHeaders:   deps.Config.CORS.AllowedHeaders,
		ExposedHeaders:   deps.Config.CORS.ExposedHeaders,
		AllowCredentials: deps.Config.CORS.AllowCredentials,
		MaxAge:           deps.Config.CORS.MaxAgeSecs,
	}).Handler)
	router.Use(chimw.Recoverer)

	router.NotFound(handlers.NotFoundHandler)
	router.MethodNotAllowed(handlers.MethodNotAllowedHandler)

	// Unversioned infra probes.
	router.Get("/healthz", deps.HealthHandler.Live)
	router.Get("/readyz", deps.HealthHandler.Ready)

	router.Route(BaseURL, func(r chi.Router) {
		r.With(chimw.Timeout(30 * time.Second)).Get("/health-check", deps.HealthHandler.Live)
		r.With(chimw.Timeout(30 * time.Second)).Get("/info", deps.HealthHandler.Info)

		// Feature route groups are mounted here in later phases:
		//   /auth, /users, /roles, /files, /folders, /storage,
		//   /shares, /search, /audit, /notifications, /api-keys, /activity.
	})
}
