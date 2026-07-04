package api

import (
	"maps"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	authhandler "sapphirebroking.com/sftp_service/internal/api/handlers/auth"
	m "sapphirebroking.com/sftp_service/internal/api/handlers/middleware"
	ssohandler "sapphirebroking.com/sftp_service/internal/api/handlers/sso"
	userhandler "sapphirebroking.com/sftp_service/internal/api/handlers/user"
	"sapphirebroking.com/sftp_service/internal/config"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Deps carries everything the router needs. Feature handlers are added here as
// later phases land (users, files, shares, ...).
type Deps struct {
	CORSConfig    config.CORSConfig
	Logger        logger.Logger
	DebugErrors   bool
	JWT           *m.JWT
	Perms         *m.Permissions
	HealthHandler *handlers.HealthHandler
	AuthHandler   *authhandler.Handler
	SSOHandler    *ssohandler.Handler
	UserHandler   *userhandler.Handler
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

	registerAuthRoutes(g, deps)
	registerUserRoutes(g, deps)
	// Feature route groups are registered here in later phases:
	//   registerFileRoutes(g, deps)
	//   ...
}

func registerUserRoutes(g *fuego.Server, deps Deps) {
	read := option.Middleware(deps.Perms.Require("users.read"))
	manage := option.Middleware(deps.Perms.Require("users.manage"))

	gu := fuego.Group(g, "/users", option.Tags("Users"), secured, respUnauthorized, respForbidden)
	fuego.Use(gu, deps.JWT.Require)

	fuego.Get(gu, "/", deps.UserHandler.List, read, option.Summary("List users"))
	fuego.Post(gu, "/", deps.UserHandler.Create, manage, option.Summary("Create user"))
	fuego.Get(gu, "/{id}", deps.UserHandler.Get, read, option.Summary("Get a user"))
	fuego.Patch(gu, "/{id}", deps.UserHandler.Update, manage, option.Summary("Update a user"))
	fuego.Delete(gu, "/{id}", deps.UserHandler.Delete, manage, option.Summary("Delete a user"))
	fuego.Put(gu, "/{id}/role", deps.UserHandler.SetRole, manage, option.Summary("Set a user's role"))
	fuego.Put(gu, "/{id}/quota", deps.UserHandler.SetQuota, manage, option.Summary("Set a user's storage quota"))
	fuego.Put(gu, "/{id}/status", deps.UserHandler.SetActive, manage, option.Summary("Enable/disable a user"))
	fuego.Post(gu, "/{id}/reset-password", deps.UserHandler.ResetPassword, manage, option.Summary("Reset a user's password"))

	gr := fuego.Group(g, "/roles", option.Tags("Roles"), secured, respUnauthorized, respForbidden)
	fuego.Use(gr, deps.JWT.Require)
	fuego.Get(gr, "/", deps.UserHandler.ListRoles, read, option.Summary("List roles and permissions"))
}

func registerAuthRoutes(g *fuego.Server, deps Deps) {
	ga := fuego.Group(g, "/auth", option.Tags("Auth"))

	fuego.Post(ga, "/login", deps.AuthHandler.Login, option.Summary("Log in with email/username and password"))
	fuego.Post(ga, "/refresh", deps.AuthHandler.Refresh, option.Summary("Refresh access token"))

	// Microsoft Entra ID (Azure AD) single sign-on.
	if deps.SSOHandler != nil && deps.SSOHandler.Enabled() {
		fuego.GetStd(ga, "/sso/microsoft/login", deps.SSOHandler.MicrosoftLogin,
			option.Summary("Begin Microsoft SSO login"))
		fuego.GetStd(ga, "/sso/microsoft/callback", deps.SSOHandler.MicrosoftCallback,
			option.Summary("Microsoft SSO callback"))
	}

	gsec := fuego.Group(ga, "", secured, respUnauthorized)
	fuego.Use(gsec, deps.JWT.Require)
	fuego.Post(gsec, "/logout", deps.AuthHandler.Logout, option.Summary("Log out (revoke refresh token)"))
	fuego.Get(gsec, "/me", deps.AuthHandler.Me, option.Summary("Get current user profile"))
	fuego.Post(gsec, "/change-password", deps.AuthHandler.ChangePassword, option.Summary("Change password"))
}
