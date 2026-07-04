// Package sso wires the OIDC single-sign-on HTTP handlers.
package sso

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	authsvc "sapphirebroking.com/sftp_service/internal/service/auth"
	ssosvc "sapphirebroking.com/sftp_service/internal/service/sso"
	"sapphirebroking.com/sftp_service/pkg/headers"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

const stateCookie = "sftp_sso_state"

// Handler serves the Microsoft SSO login and callback endpoints.
type Handler struct {
	ms     *ssosvc.Microsoft
	auth   *authsvc.Service
	secure bool
	log    logger.Logger
}

// NewHandler constructs the SSO handler. ms may be nil (SSO disabled).
func NewHandler(ms *ssosvc.Microsoft, auth *authsvc.Service, secure bool, log logger.Logger) *Handler {
	return &Handler{ms: ms, auth: auth, secure: secure, log: log.Named("handler.sso")}
}

// Enabled reports whether Microsoft SSO is configured.
func (h *Handler) Enabled() bool { return h.ms != nil }

// MicrosoftLogin starts the OIDC auth-code flow (302 to Microsoft).
func (h *Handler) MicrosoftLogin(w http.ResponseWriter, r *http.Request) {
	if h.ms == nil {
		handlers.WriteProblem(w, r, http.StatusServiceUnavailable, "microsoft sso is not configured")
		return
	}
	state, err := randomState()
	if err != nil {
		handlers.WriteProblem(w, r, http.StatusInternalServerError, "could not start sso")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state,
		Path:     "/api/v1/auth/sso",
		MaxAge:   int((5 * time.Minute).Seconds()),
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, h.ms.AuthCodeURL(state), http.StatusFound)
}

// MicrosoftCallback completes the flow, provisions/logs in the user and
// redirects to the frontend success URL with tokens in the fragment.
func (h *Handler) MicrosoftCallback(w http.ResponseWriter, r *http.Request) {
	if h.ms == nil {
		handlers.WriteProblem(w, r, http.StatusServiceUnavailable, "microsoft sso is not configured")
		return
	}
	cfg := h.ms.Config()
	q := r.URL.Query()

	if e := q.Get("error"); e != "" {
		h.redirectError(w, r, cfg.SuccessURL, q.Get("error_description"))
		return
	}

	// CSRF: state query must match the state cookie.
	cookie, err := r.Cookie(stateCookie)
	if err != nil || cookie.Value == "" || cookie.Value != q.Get("state") {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "invalid sso state")
		return
	}
	h.clearState(w)

	code := q.Get("code")
	if code == "" {
		handlers.WriteProblem(w, r, http.StatusBadRequest, "missing authorization code")
		return
	}

	profile, err := h.ms.Exchange(r.Context(), code)
	if err != nil {
		h.log.Warn("sso exchange failed", "err", err)
		h.redirectError(w, r, cfg.SuccessURL, "authentication failed")
		return
	}

	pair, err := h.auth.LoginSSO(r.Context(), profile, cfg.DefaultRole, cfg.AllowedDomains, authsvc.RequestMeta{
		IP:        headers.GetClientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		h.log.Warn("sso login failed", "err", err, "email", profile.Email)
		h.redirectError(w, r, cfg.SuccessURL, "access denied")
		return
	}

	frag := url.Values{}
	frag.Set("access_token", pair.AccessToken)
	frag.Set("refresh_token", pair.RefreshToken)
	frag.Set("expires_in", fmt.Sprintf("%d", pair.ExpiresIn))
	http.Redirect(w, r, cfg.SuccessURL+"#"+frag.Encode(), http.StatusFound)
}

func (h *Handler) redirectError(w http.ResponseWriter, r *http.Request, base, msg string) {
	frag := url.Values{}
	frag.Set("error", msg)
	http.Redirect(w, r, base+"#"+frag.Encode(), http.StatusFound)
}

func (h *Handler) clearState(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: stateCookie, Value: "", Path: "/api/v1/auth/sso",
		MaxAge: -1, HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode,
	})
}

func randomState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
