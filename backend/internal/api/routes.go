package api

import (
	"maps"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	apikeyhandler "sapphirebroking.com/sftp_service/internal/api/handlers/apikey"
	audithandler "sapphirebroking.com/sftp_service/internal/api/handlers/audit"
	authhandler "sapphirebroking.com/sftp_service/internal/api/handlers/auth"
	filehandler "sapphirebroking.com/sftp_service/internal/api/handlers/file"
	m "sapphirebroking.com/sftp_service/internal/api/handlers/middleware"
	sharehandler "sapphirebroking.com/sftp_service/internal/api/handlers/share"
	ssohandler "sapphirebroking.com/sftp_service/internal/api/handlers/sso"
	userhandler "sapphirebroking.com/sftp_service/internal/api/handlers/user"
	"sapphirebroking.com/sftp_service/internal/config"
	auditsvc "sapphirebroking.com/sftp_service/internal/service/audit"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Deps carries everything the router needs. Feature handlers are added here as
// later phases land (users, files, shares, ...).
type Deps struct {
	CORSConfig    config.CORSConfig
	Logger        logger.Logger
	DebugErrors   bool
	Auth          *m.Authenticator
	Perms         *m.Permissions
	Recorder      *auditsvc.Recorder
	HealthHandler *handlers.HealthHandler
	AuthHandler   *authhandler.Handler
	SSOHandler    *ssohandler.Handler
	UserHandler   *userhandler.Handler
	FileHandler   *filehandler.Handler
	APIKeyHandler *apikeyhandler.Handler
	AuditHandler  *audithandler.Handler
	ShareHandler  *sharehandler.Handler
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
	registerFileRoutes(g, deps)
	registerAPIKeyRoutes(g, deps)
	registerAuditRoutes(g, deps)
	registerShareRoutes(g, deps)
}

func registerShareRoutes(g *fuego.Server, deps Deps) {
	// Owner-managed shares (authenticated, files.share).
	gs := fuego.Group(g, "/shares", option.Tags("Shares"), secured, respUnauthorized, respForbidden)
	fuego.Use(gs, deps.Auth.Require)
	share := option.Middleware(deps.Perms.Require("files.share"))
	fuego.Get(gs, "/", deps.ShareHandler.List, share, option.Summary("List your share links"))
	fuego.Post(gs, "/", deps.ShareHandler.Create, share, option.Summary("Create a share link"))
	fuego.Delete(gs, "/{id}", deps.ShareHandler.Revoke, share, option.Summary("Revoke a share link"))

	// Public share access (no authentication).
	gp := fuego.Group(g, "/share", option.Tags("Public Share"))
	fuego.Get(gp, "/{token}", deps.ShareHandler.PublicInfo, option.Summary("Get public share metadata"))
	fuego.GetStd(gp, "/{token}/download", deps.ShareHandler.PublicDownload, option.Summary("Download a shared file"))
}

func registerAuditRoutes(g *fuego.Server, deps Deps) {
	// Telemetry: any authenticated user may report their own UI activity.
	ga := fuego.Group(g, "/activity", option.Tags("Activity"), secured, respUnauthorized)
	fuego.Use(ga, deps.Auth.Require)
	fuego.Post(ga, "/", deps.AuditHandler.Telemetry, option.Summary("Record a UI activity/click event"))

	// Audit log: restricted to holders of audit.read.
	gl := fuego.Group(g, "/audit", option.Tags("Audit"), secured, respUnauthorized, respForbidden)
	fuego.Use(gl, deps.Auth.Require)
	fuego.Get(gl, "/", deps.AuditHandler.List,
		option.Middleware(deps.Perms.Require("audit.read")), option.Summary("List audit log"))
}

func registerAPIKeyRoutes(g *fuego.Server, deps Deps) {
	manage := option.Middleware(deps.Perms.Require("apikeys.manage"))
	gk := fuego.Group(g, "/api-keys", option.Tags("API Keys"), secured, respUnauthorized, respForbidden)
	fuego.Use(gk, deps.Auth.Require)
	fuego.Get(gk, "/", deps.APIKeyHandler.List, manage, option.Summary("List your API keys"))
	fuego.Post(gk, "/", deps.APIKeyHandler.Create, manage, option.Summary("Create an API key"))
	fuego.Delete(gk, "/{id}", deps.APIKeyHandler.Revoke, manage, option.Summary("Revoke an API key"))
}

func registerFileRoutes(g *fuego.Server, deps Deps) {
	read := option.Middleware(deps.Perms.Require("files.read"))
	upload := option.Middleware(deps.Perms.Require("files.upload"))
	write := option.Middleware(deps.Perms.Require("files.write"))
	del := option.Middleware(deps.Perms.Require("files.delete"))
	fwrite := option.Middleware(deps.Perms.Require("folders.write"))
	fdel := option.Middleware(deps.Perms.Require("folders.delete"))
	h := deps.FileHandler

	// Folders.
	gd := fuego.Group(g, "/folders", option.Tags("Folders"), secured, respUnauthorized, respForbidden)
	fuego.Use(gd, deps.Auth.Require)
	fuego.Post(gd, "/", h.CreateFolder, fwrite, option.Summary("Create folder"))
	fuego.Put(gd, "/{id}/rename", h.RenameFolder, fwrite, option.Summary("Rename folder"))
	fuego.Put(gd, "/{id}/move", h.MoveFolder, fwrite, option.Summary("Move folder"))
	fuego.Put(gd, "/{id}/star", h.StarFolder, fwrite, option.Summary("Star/unstar folder"))
	fuego.Put(gd, "/{id}/color", h.SetFolderColor, fwrite, option.Summary("Set folder colour"))
	fuego.Delete(gd, "/{id}", h.DeleteFolder, fdel, option.Summary("Delete folder"))

	// Files.
	gf := fuego.Group(g, "/files", option.Tags("Files"), secured, respUnauthorized, respForbidden)
	fuego.Use(gf, deps.Auth.Require)

	fuego.Get(gf, "/", h.List, read, option.Summary("List folder contents"))
	fuego.Get(gf, "/trash", h.Trash, read, option.Summary("List recycle bin"))
	fuego.Get(gf, "/recent", h.Recent, read, option.Summary("List recent files"))
	fuego.Get(gf, "/starred", h.Starred, read, option.Summary("List starred files"))
	fuego.Get(gf, "/search", h.Search, read, option.Summary("Search files by name"))

	// Organisation-wide Common area.
	fuego.Get(gf, "/common", h.CommonList, read, option.Summary("List organisation-wide Common files"))
	fuego.PostStd(gf, "/common/upload", h.CommonUpload, upload, option.Summary("Upload a file to Common"))
	fuego.Delete(gf, "/common/{id}", h.CommonDelete, option.Summary("Delete a Common file (uploader or admin)"))

	// Simple single-request multipart upload.
	fuego.PostStd(gf, "/upload", h.SimpleUpload, upload, option.Summary("Upload a single file (multipart)"))

	// Single file.
	fuego.Get(gf, "/{id}", h.GetFile, read, option.Summary("Get file metadata"))
	fuego.GetStd(gf, "/{id}/download", h.Download, read, option.Summary("Download a file"))
	fuego.Put(gf, "/{id}/rename", h.RenameFile, write, option.Summary("Rename file"))
	fuego.Put(gf, "/{id}/move", h.MoveFile, write, option.Summary("Move file"))
	fuego.Put(gf, "/{id}/star", h.StarFile, write, option.Summary("Star/unstar file"))
	fuego.Post(gf, "/{id}/trash", h.TrashFile, write, option.Summary("Move file to trash"))
	fuego.Post(gf, "/{id}/restore", h.RestoreFile, write, option.Summary("Restore file from trash"))
	fuego.Post(gf, "/{id}/make-common", h.MakeCommon, option.Middleware(deps.Perms.Require("files.share")), option.Summary("Share a file to Common"))
	fuego.Delete(gf, "/{id}", h.DeleteFile, del, option.Summary("Permanently delete file"))

	// Resumable uploads (own group to avoid mux wildcard collisions with /files/{id}).
	gu := fuego.Group(g, "/uploads", option.Tags("Uploads"), secured, respUnauthorized, respForbidden)
	fuego.Use(gu, deps.Auth.Require)
	fuego.Post(gu, "/", h.InitUpload, upload, option.Summary("Start a resumable upload"))
	fuego.Get(gu, "/{id}", h.UploadStatus, read, option.Summary("Get upload progress"))
	fuego.PutStd(gu, "/{id}/chunks/{index}", h.PutChunk, upload, option.Summary("Upload a chunk"))
	fuego.Post(gu, "/{id}/complete", h.CompleteUpload, upload, option.Summary("Complete an upload"))
	fuego.Delete(gu, "/{id}", h.AbortUpload, upload, option.Summary("Abort an upload"))
}

func registerUserRoutes(g *fuego.Server, deps Deps) {
	read := option.Middleware(deps.Perms.Require("users.read"))
	manage := option.Middleware(deps.Perms.Require("users.manage"))

	gu := fuego.Group(g, "/users", option.Tags("Users"), secured, respUnauthorized, respForbidden)
	fuego.Use(gu, deps.Auth.Require)

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
	fuego.Use(gr, deps.Auth.Require)
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
	fuego.Use(gsec, deps.Auth.Require)
	fuego.Post(gsec, "/logout", deps.AuthHandler.Logout, option.Summary("Log out (revoke refresh token)"))
	fuego.Get(gsec, "/me", deps.AuthHandler.Me, option.Summary("Get current user profile"))
	fuego.Post(gsec, "/change-password", deps.AuthHandler.ChangePassword, option.Summary("Change password"))
}
