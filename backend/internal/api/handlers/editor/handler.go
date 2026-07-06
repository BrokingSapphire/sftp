// Package editor integrates a self-hosted OnlyOffice Document Server for live
// Office (docx/xlsx/pptx) co-editing. It is inert unless enabled in config.
//
// Flow: the browser asks GET /editor/{id}/config → we return a signed editor
// config; the browser hands it to OnlyOffice's api.js. The Document Server then
// (a) fetches the file from a backend download URL carrying a short-lived JWT,
// and (b) on save POSTs the edited document to our callback, which stores it as
// a new version.
package editor

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/api/response"
	"sapphirebroking.com/sftp_service/internal/config"
	filesvc "sapphirebroking.com/sftp_service/internal/service/file"
	"sapphirebroking.com/sftp_service/pkg/jwt"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Handler serves /editor.
type Handler struct {
	files *filesvc.Service
	jwt   *jwt.Manager
	cfg   config.EditorConfig
	http  *http.Client
	log   logger.Logger
}

// NewHandler builds the editor handler.
func NewHandler(files *filesvc.Service, jwtMgr *jwt.Manager, cfg config.EditorConfig, log logger.Logger) *Handler {
	return &Handler{files: files, jwt: jwtMgr, cfg: cfg, http: &http.Client{Timeout: 60 * time.Second}, log: log.Named("handler.editor")}
}

// Enabled reports whether Office editing is configured.
func (h *Handler) Enabled() bool { return h.cfg.Enabled && h.cfg.DocServerURL != "" }

// Session is returned to the browser to boot the OnlyOffice editor.
type Session struct {
	DocServerURL string         `json:"doc_server_url"`
	Config       map[string]any `json:"config"`
}

var docType = map[string]string{
	"docx": "word", "doc": "word", "odt": "word", "rtf": "word", "txt": "word",
	"xlsx": "cell", "xls": "cell", "ods": "cell", "csv": "cell",
	"pptx": "slide", "ppt": "slide", "odp": "slide",
}

// EditableExt reports whether OnlyOffice can edit this extension.
func EditableExt(ext string) bool { _, ok := docType[strings.ToLower(ext)]; return ok }

// Config builds the signed OnlyOffice editor configuration for a file.
func (h *Handler) Config(c fuego.ContextNoBody) (*response.Envelope[Session], error) {
	if !h.Enabled() {
		return nil, fuego.HTTPError{Title: "Office editing is not enabled", Status: 503}
	}
	uid, err := currentUser(c.Context())
	if err != nil {
		return nil, handlers.Fail(err)
	}
	id, err := uuid.Parse(c.PathParam("id"))
	if err != nil {
		return nil, fuego.BadRequestError{Title: "invalid file id"}
	}
	f, err := h.files.GetFile(c.Context(), uid, id)
	if err != nil {
		return nil, handlers.Fail(err)
	}
	dtype, ok := docType[strings.ToLower(f.Extension)]
	if !ok {
		return nil, fuego.BadRequestError{Title: "this file type is not editable"}
	}

	// Short-lived token the Document Server uses to fetch the file.
	fetchTok, _, err := h.jwt.Issue(uid.String(), "", "", "", "")
	if err != nil {
		return nil, handlers.Fail(err)
	}
	base := strings.TrimRight(h.cfg.InternalBaseURL, "/")
	// Callback token binds this callback to (file, owner) and expires.
	cbTok := h.sign(map[string]any{"f": id.String(), "o": uid.String(), "exp": time.Now().Add(4 * time.Hour).Unix()})

	cfg := map[string]any{
		"document": map[string]any{
			"fileType": strings.ToLower(f.Extension),
			"key":      fmt.Sprintf("%s_%d", f.ID, f.VersionNo),
			"title":    f.Name,
			"url":      fmt.Sprintf("%s/api/v1/files/%s/download?access_token=%s", base, id, fetchTok),
		},
		"documentType": dtype,
		"editorConfig": map[string]any{
			"callbackUrl": fmt.Sprintf("%s/api/v1/editor/%s/callback?t=%s", base, id, cbTok),
			"user":        map[string]any{"id": uid.String(), "name": f.Name},
			"mode":        "edit",
		},
	}
	// OnlyOffice validates this JWT of the config with the shared secret.
	cfg["token"] = h.sign(cfg)

	return response.OK(Session{DocServerURL: strings.TrimRight(h.cfg.DocServerURL, "/"), Config: cfg}), nil
}

// callbackBody is the OnlyOffice save callback payload.
type callbackBody struct {
	Status int    `json:"status"`
	URL    string `json:"url"`
}

// Callback receives edited documents from the Document Server (unauthenticated;
// secured by the signed `t` token) and stores them as a new version.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	// OnlyOffice ignores non-zero errors and retries, so always answer {"error":0}.
	reply := func() { w.Header().Set("Content-Type", "application/json"); _, _ = w.Write([]byte(`{"error":0}`)) }

	claims, err := h.verify(r.URL.Query().Get("t"))
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":1}`))
		return
	}
	owner, _ := uuid.Parse(str(claims["o"]))
	fileID, _ := uuid.Parse(str(claims["f"]))

	var body callbackBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		reply()
		return
	}
	// 2 = ready to save, 6 = force save. Other statuses need no action.
	if (body.Status == 2 || body.Status == 6) && body.URL != "" {
		if err := h.saveFromURL(r.Context(), owner, fileID, body.URL); err != nil {
			h.log.Error("editor save failed", "file", fileID, "err", err)
		} else {
			h.log.Info("editor saved new version", "file", fileID)
		}
	}
	reply()
}

func (h *Handler) saveFromURL(ctx context.Context, owner, fileID uuid.UUID, url string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := h.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch edited doc: %s", resp.Status)
	}
	_, err = h.files.OverwriteContent(ctx, owner, fileID, resp.Body)
	return err
}

// ── minimal HS256 JWT (OnlyOffice shared-secret) ──────────────────────────────

func (h *Handler) sign(payload map[string]any) string {
	header := b64(`{"alg":"HS256","typ":"JWT"}`)
	pj, _ := json.Marshal(payload)
	body := b64(string(pj))
	sig := b64(string(hmacSHA256(header+"."+body, h.cfg.JWTSecret)))
	return header + "." + body + "." + sig
}

func (h *Handler) verify(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token")
	}
	expected := b64(string(hmacSHA256(parts[0]+"."+parts[1], h.cfg.JWTSecret)))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, fmt.Errorf("bad signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, err
	}
	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return nil, fmt.Errorf("expired")
	}
	return claims, nil
}

func hmacSHA256(msg, secret string) []byte {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(msg))
	return m.Sum(nil)
}
func b64(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
func str(v any) string    { s, _ := v.(string); return s }

func currentUser(ctx context.Context) (uuid.UUID, error) {
	claims := jwt.GetClaimsFromContext(ctx)
	if claims == nil || claims.Sub == nil {
		return uuid.Nil, fmt.Errorf("unauthenticated")
	}
	return uuid.Parse(*claims.Sub)
}
