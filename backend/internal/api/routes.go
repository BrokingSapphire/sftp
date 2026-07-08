package api

import (
	"maps"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	aihandler "sapphirebroking.com/sftp_service/internal/api/handlers/ai"
	apikeyhandler "sapphirebroking.com/sftp_service/internal/api/handlers/apikey"
	audithandler "sapphirebroking.com/sftp_service/internal/api/handlers/audit"
	authhandler "sapphirebroking.com/sftp_service/internal/api/handlers/auth"
	backuphandler "sapphirebroking.com/sftp_service/internal/api/handlers/backup"
	editorhandler "sapphirebroking.com/sftp_service/internal/api/handlers/editor"
	filehandler "sapphirebroking.com/sftp_service/internal/api/handlers/file"
	m "sapphirebroking.com/sftp_service/internal/api/handlers/middleware"
	notifhandler "sapphirebroking.com/sftp_service/internal/api/handlers/notification"
	securityhandler "sapphirebroking.com/sftp_service/internal/api/handlers/security"
	sharehandler "sapphirebroking.com/sftp_service/internal/api/handlers/share"
	ssohandler "sapphirebroking.com/sftp_service/internal/api/handlers/sso"
	teamhandler "sapphirebroking.com/sftp_service/internal/api/handlers/team"
	userhandler "sapphirebroking.com/sftp_service/internal/api/handlers/user"
	"sapphirebroking.com/sftp_service/internal/config"
	"sapphirebroking.com/sftp_service/internal/metrics"
	auditsvc "sapphirebroking.com/sftp_service/internal/service/audit"
	"sapphirebroking.com/sftp_service/pkg/logger"
	"sapphirebroking.com/sftp_service/pkg/ratelimit"
)

// Deps carries everything the router needs. Feature handlers are added here as
// later phases land (users, files, shares, ...).
type Deps struct {
	CORSConfig      config.CORSConfig
	Logger          logger.Logger
	DebugErrors     bool
	Auth            *m.Authenticator
	GlobalRL        *ratelimit.Limiter
	LoginRL         *ratelimit.Limiter
	Perms           *m.Permissions
	Recorder        *auditsvc.Recorder
	HealthHandler   *handlers.HealthHandler
	AuthHandler     *authhandler.Handler
	SSOHandler      *ssohandler.Handler
	UserHandler     *userhandler.Handler
	FileHandler     *filehandler.Handler
	APIKeyHandler   *apikeyhandler.Handler
	AuditHandler    *audithandler.Handler
	SecurityHandler *securityhandler.Handler
	AIHandler       *aihandler.Handler
	EditorHandler   *editorhandler.Handler
	BackupHandler   *backuphandler.Handler
	TeamHandler     *teamhandler.Handler
	ShareHandler    *sharehandler.Handler
	NotifHandler    *notifhandler.Handler
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
	fuego.GetStd(s, "/metrics", func(w http.ResponseWriter, r *http.Request) { metrics.Handler().ServeHTTP(w, r) }, option.Hide())

	g := fuego.Group(s, BaseURL)
	// Global per-IP rate limit across the API.
	if deps.GlobalRL != nil {
		fuego.Use(g, m.RateLimit(deps.GlobalRL))
	}
	fuego.Get(g, "/health-check", deps.HealthHandler.Live, option.Summary("Health check"), option.Tags("Health"))
	fuego.Get(g, "/info", deps.HealthHandler.Info, option.Summary("Build/runtime info"), option.Tags("Health"))

	registerAuthRoutes(g, deps)
	registerUserRoutes(g, deps)
	registerFileRoutes(g, deps)
	registerAPIKeyRoutes(g, deps)
	registerAuditRoutes(g, deps)
	registerShareRoutes(g, deps)
	registerNotificationRoutes(g, deps)
	registerAIRoutes(g, deps)
	registerEditorRoutes(g, deps)
	registerAdminRoutes(g, deps)
	registerTeamRoutes(g, deps)
}

func registerTeamRoutes(g *fuego.Server, deps Deps) {
	gt := fuego.Group(g, "/teams", option.Tags("Teams"), secured, respUnauthorized, respForbidden)
	fuego.Use(gt, deps.Auth.Require)
	fuego.Get(gt, "/", deps.TeamHandler.List, option.Summary("List my teams"))
	fuego.Post(gt, "/", deps.TeamHandler.Create, option.Summary("Create a team"))
	fuego.Get(gt, "/{id}", deps.TeamHandler.Get, option.Summary("Get a team"))
	fuego.Put(gt, "/{id}", deps.TeamHandler.Update, option.Summary("Update a team (admin+)"))
	fuego.Delete(gt, "/{id}", deps.TeamHandler.Delete, option.Summary("Delete a team (owner)"))
	fuego.Get(gt, "/{id}/members", deps.TeamHandler.Members, option.Summary("List team members"))
	fuego.Post(gt, "/{id}/members", deps.TeamHandler.AddMember, option.Summary("Add a member (admin+)"))
	fuego.Delete(gt, "/{id}/members/{uid}", deps.TeamHandler.RemoveMember, option.Summary("Remove a member (admin+)"))
}

func registerAdminRoutes(g *fuego.Server, deps Deps) {
	// Super-admin backup/restore. The role is enforced inside the handlers.
	gadm := fuego.Group(g, "/admin", option.Tags("Admin"), secured, respUnauthorized, respForbidden)
	fuego.Use(gadm, deps.Auth.Require)
	fuego.Post(gadm, "/backup", deps.BackupHandler.Run, option.Summary("Run a full/incremental backup (super admin)"))
	fuego.Get(gadm, "/backup/status", deps.BackupHandler.Status, option.Summary("Backup status for a target (super admin)"))
	fuego.Post(gadm, "/restore", deps.BackupHandler.Restore, option.Summary("Restore from a backup target (super admin)"))
}

func registerEditorRoutes(g *fuego.Server, deps Deps) {
	// Config: authenticated + files.write.
	ge := fuego.Group(g, "/editor", option.Tags("Editor"), secured, respUnauthorized, respForbidden)
	fuego.Use(ge, deps.Auth.Require)
	fuego.Get(ge, "/{id}/config", deps.EditorHandler.Config,
		option.Middleware(deps.Perms.Require("files.write")), option.Summary("OnlyOffice editor config for a file"))

	// Callback: called by the Document Server (no user auth; secured by a signed token).
	fuego.PostStd(g, "/editor/{id}/callback", deps.EditorHandler.Callback, option.Summary("OnlyOffice save callback"))
}

func registerAIRoutes(g *fuego.Server, deps Deps) {
	gai := fuego.Group(g, "/ai", option.Tags("AI"), secured, respUnauthorized)
	fuego.Use(gai, deps.Auth.Require)
	fuego.Get(gai, "/status", deps.AIHandler.Status, option.Summary("AI availability"))
	fuego.Post(gai, "/ask", deps.AIHandler.Ask, option.Summary("Ask a question about your files"))
	fuego.Get(gai, "/search", deps.AIHandler.Search, option.Summary("Semantic search over your files"))
}

func registerNotificationRoutes(g *fuego.Server, deps Deps) {
	gn := fuego.Group(g, "/notifications", option.Tags("Notifications"), secured, respUnauthorized)
	fuego.Use(gn, deps.Auth.Require)
	fuego.Get(gn, "/", deps.NotifHandler.List, option.Summary("List notifications"))
	fuego.Get(gn, "/unread-count", deps.NotifHandler.UnreadCount, option.Summary("Unread count"))
	fuego.Post(gn, "/{id}/read", deps.NotifHandler.MarkRead, option.Summary("Mark a notification read"))
	fuego.Post(gn, "/read-all", deps.NotifHandler.MarkAllRead, option.Summary("Mark all read"))
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

	// Security alerts (audit anomaly detection): same audit.read gate.
	gs := fuego.Group(g, "/security", option.Tags("Security"), secured, respUnauthorized, respForbidden)
	fuego.Use(gs, deps.Auth.Require)
	sec := option.Middleware(deps.Perms.Require("audit.read"))
	fuego.Get(gs, "/alerts", deps.SecurityHandler.List, sec, option.Summary("List security alerts"))
	fuego.Get(gs, "/alerts/unresolved-count", deps.SecurityHandler.UnresolvedCount, sec, option.Summary("Unresolved alert count"))
	fuego.Post(gs, "/alerts/{id}/resolve", deps.SecurityHandler.Resolve, sec, option.Summary("Resolve a security alert"))
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
	fuego.GetStd(gd, "/{id}/download", h.FolderDownload, read, option.Summary("Download a folder as a zip"))

	// Internal (per-user) folder sharing — mirrors the file endpoints below.
	fuego.Get(gd, "/shared-with-me", h.SharedFoldersWithMe, read, option.Summary("List folders shared with me"))
	fuego.Post(gd, "/{id}/share-user", h.ShareFolderWithUser, fwrite, option.Summary("Share a folder with a specific user"))
	fuego.Get(gd, "/{id}/shares", h.ListFolderGrants, read, option.Summary("List a folder's internal recipients"))
	fuego.Delete(gd, "/{id}/shares/{uid}", h.RevokeFolderGrant, fwrite, option.Summary("Remove a user's access to a folder"))

	// Files.
	gf := fuego.Group(g, "/files", option.Tags("Files"), secured, respUnauthorized, respForbidden)
	fuego.Use(gf, deps.Auth.Require)

	fuego.Get(gf, "/", h.List, read, option.Summary("List folder contents"))
	fuego.Get(gf, "/trash", h.Trash, read, option.Summary("List recycle bin"))
	fuego.Post(gf, "/trash/empty", h.EmptyTrash, del, option.Summary("Permanently empty the recycle bin"))
	fuego.Get(gf, "/recent", h.Recent, read, option.Summary("List recent files"))
	fuego.Get(gf, "/starred", h.Starred, read, option.Summary("List starred files"))
	fuego.Get(gf, "/search", h.Search, read, option.Summary("Search files by name"))
	fuego.Get(gf, "/search/content", h.SearchContent, read, option.Summary("Full-text search inside file contents"))
	fuego.Get(gf, "/inherited", h.Inherited, read, option.Summary("List files inherited from a deleted user"))
	fuego.Post(gf, "/{id}/keep", h.KeepFile, write, option.Summary("Keep an inherited file"))
	fuego.Get(gf, "/shared-with-me", h.SharedWithMe, read, option.Summary("List files shared with me"))
	fuego.Post(gf, "/{id}/share-user", h.ShareWithUser, write, option.Summary("Share a file with a specific user"))
	fuego.Get(gf, "/{id}/shares", h.ListGrants, read, option.Summary("List a file's internal recipients"))
	fuego.Delete(gf, "/{id}/shares/{uid}", h.RevokeGrant, write, option.Summary("Remove a user's access to a file"))

	// Organisation-wide Common area.
	fuego.Get(gf, "/common", h.CommonList, read, option.Summary("List organisation-wide Common files"))
	fuego.Get(gf, "/common/browse", h.CommonBrowse, read, option.Summary("Browse Common folders + files at a level"))
	fuego.Post(gf, "/common/folders", h.CommonFolderCreate, upload, option.Summary("Create a folder in Common"))
	fuego.PostStd(gf, "/common/upload", h.CommonUpload, upload, option.Summary("Upload a file to Common (optional folder_id)"))
	fuego.Delete(gf, "/common/{id}", h.CommonDelete, option.Summary("Delete a Common file (uploader or admin)"))

	// Simple single-request multipart upload.
	fuego.PostStd(gf, "/upload", h.SimpleUpload, upload, option.Summary("Upload a single file (multipart)"))

	// Single file.
	fuego.Get(gf, "/{id}/versions", h.ListVersions, read, option.Summary("List a file's previous versions"))
	fuego.Post(gf, "/{id}/versions/{version}/restore", h.RestoreVersion, write, option.Summary("Restore a previous version"))
	fuego.GetStd(gf, "/{id}/versions/{version}/download", h.DownloadVersion, read, option.Summary("Download a previous version"))
	fuego.Get(gf, "/{id}", h.GetFile, read, option.Summary("Get file metadata"))
	fuego.GetStd(gf, "/{id}/download", h.Download, read, option.Summary("Download a file"))
	fuego.PutStd(gf, "/{id}/content", h.SaveContent, write, option.Summary("Overwrite a file's content (in-app editor)"))
	fuego.Put(gf, "/{id}/rename", h.RenameFile, write, option.Summary("Rename file"))
	fuego.Put(gf, "/{id}/move", h.MoveFile, write, option.Summary("Move file"))
	fuego.Put(gf, "/{id}/star", h.StarFile, write, option.Summary("Star/unstar file"))
	fuego.Post(gf, "/{id}/trash", h.TrashFile, write, option.Summary("Move file to trash"))
	fuego.Post(gf, "/{id}/restore", h.RestoreFile, write, option.Summary("Restore file from trash"))
	fuego.Post(gf, "/{id}/make-common", h.MakeCommon, option.Middleware(deps.Perms.Require("files.share")), option.Summary("Share a file to Common"))
	fuego.Post(gf, "/{id}/copy", h.CopyFile, upload, option.Summary("Duplicate a file"))
	compliance := option.Middleware(deps.Perms.Require("storage.manage"))
	fuego.Post(gf, "/{id}/legal-hold", h.SetLegalHold, compliance, option.Summary("Place/release a legal hold (admin)"))
	fuego.Post(gf, "/{id}/retention", h.SetRetention, compliance, option.Summary("Set/clear a WORM retention lock (admin)"))
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
	fuego.Delete(gu, "/{id}", deps.UserHandler.Delete, manage, option.Summary("Delete a user (transfers their files)"))
	fuego.Post(gu, "/{id}/enable", deps.UserHandler.Enable, manage, option.Summary("Re-enable a disabled account (super admin)"))
	fuego.Put(gu, "/{id}/role", deps.UserHandler.SetRole, manage, option.Summary("Set a user's role"))
	fuego.Put(gu, "/{id}/quota", deps.UserHandler.SetQuota, manage, option.Summary("Set a user's storage quota"))
	fuego.Put(gu, "/{id}/status", deps.UserHandler.SetActive, manage, option.Summary("Enable/disable a user"))
	fuego.Post(gu, "/{id}/reset-password", deps.UserHandler.ResetPassword, manage, option.Summary("Reset a user's password"))
	fuego.Get(gu, "/storage", deps.UserHandler.StorageOverview, option.Middleware(deps.Perms.Require("storage.manage")), option.Summary("Storage usage overview (admin)"))
	fuego.PostStd(gu, "/me/avatar", deps.UserHandler.UploadAvatar, option.Summary("Upload your profile photo"))
	fuego.GetStd(gu, "/{id}/avatar", deps.UserHandler.Avatar, option.Summary("Get a user's profile photo"))

	gr := fuego.Group(g, "/roles", option.Tags("Roles"), secured, respUnauthorized, respForbidden)
	fuego.Use(gr, deps.Auth.Require)
	fuego.Get(gr, "/", deps.UserHandler.ListRoles, read, option.Summary("List roles and permissions"))
}

func registerAuthRoutes(g *fuego.Server, deps Deps) {
	ga := fuego.Group(g, "/auth", option.Tags("Auth"))

	fuego.Post(ga, "/login", deps.AuthHandler.Login, option.Middleware(m.RateLimit(deps.LoginRL)), option.Summary("Log in with email/username and password"))
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
	fuego.Put(gsec, "/me/language", deps.AuthHandler.SetLanguage, option.Summary("Set preferred UI language"))
	fuego.Post(gsec, "/change-password", deps.AuthHandler.ChangePassword, option.Summary("Change password"))
}
