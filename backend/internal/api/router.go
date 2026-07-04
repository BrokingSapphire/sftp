package api

import (
	"github.com/BrokingSapphire/sftp/backend/internal/config"
	"github.com/BrokingSapphire/sftp/backend/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Deps carries the dependencies needed to build the HTTP router.
type Deps struct {
	Config  *config.Config
	Logger  *zap.Logger
	DB      *gorm.DB
	Version string
}

// NewRouter builds the fully-wired Gin engine.
func NewRouter(d Deps) *gin.Engine {
	if d.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Logger(d.Logger),
		middleware.Recovery(d.Logger),
		middleware.SecurityHeaders(),
		corsMiddleware(d.Config),
	)

	health := NewHealthHandler(d.DB, d.Version)
	r.GET("/healthz", health.Live)
	r.GET("/readyz", health.Ready)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/info", health.Info)
		// Feature routes are registered here in later phases:
		//   auth, users, roles, files, folders, storage,
		//   shares, search, audit, notifications, api-keys.
	}

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID", "X-API-Key", "X-Upload-Id", "Content-Range"},
		ExposeHeaders:    []string{"X-Request-ID", "Content-Range", "Content-Length"},
		AllowCredentials: true,
	})
}
