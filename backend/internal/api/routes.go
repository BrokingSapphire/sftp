package api

import (
	"maps"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/config"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Deps carries everything the router needs. Feature handlers are added here as
// later phases land (auth, users, files, shares, ...).
type Deps struct {
	CORSConfig    config.CORSConfig
	Logger        logger.Logger
	DebugErrors   bool
	HealthHandler *handlers.HealthHandler
}

var (
	secured          = option.Security(openapi3.SecurityRequirement{"bearerAuth": []string{}})
	respUnauthorized = problemResponse(401, "Missing, invalid, or expired authentication token")
	respForbidden    = problemResponse(403, "Authenticated but lacking the required permission")
)

func problemResponse(code int, desc string) fuego.RouteOption {
	return option.AddResponse(code, desc, fuego.Response{Type: fuego.HTTPError{}})
}

// RegisterRoutes mounts all routes on the Fuego server.
func RegisterRoutes(s *fuego.Server, deps Deps) {
	components := s.OpenAPI.Description().Components
	if components.SecuritySchemes == nil {
		components.SecuritySchemes = openapi3.SecuritySchemes{}
	}
	maps.Copy(components.SecuritySchemes, securitySchemes)

	// Unversioned infra probes.
	fuego.Get(s, "/healthz", deps.HealthHandler.Live, option.Summary("Liveness probe"), option.Hide())
	fuego.Get(s, "/readyz", deps.HealthHandler.Ready, option.Summary("Readiness probe"), option.Hide())

	g := fuego.Group(s, BaseURL)
	fuego.Get(g, "/health-check", deps.HealthHandler.Live, option.Summary("Health check"), option.Tags("Health"))
	fuego.Get(g, "/info", deps.HealthHandler.Info, option.Summary("Build/runtime info"), option.Tags("Health"))

	// Feature route groups are registered here in later phases:
	//   registerAuthRoutes(g, deps)
	//   registerUserRoutes(g, deps)
	//   registerFileRoutes(g, deps)
	//   ...
}
