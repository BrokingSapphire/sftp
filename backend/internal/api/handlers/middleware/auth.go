package middleware

import (
	"context"
	"net/http"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	apikeysvc "sapphirebroking.com/sftp_service/internal/service/apikey"
	"sapphirebroking.com/sftp_service/pkg/headers"
	"sapphirebroking.com/sftp_service/pkg/jwt"
)

// Authenticator authenticates requests via either a Bearer JWT or an API key
// (X-API-Key), injecting equivalent claims so downstream handlers and the RBAC
// middleware work identically for web and programmatic clients.
type Authenticator struct {
	jwt    *jwt.Manager
	apiKey *apikeysvc.Service
	// sessionActive reports whether a JWT's session is still valid (not revoked /
	// expired). Enforces single-active-session: a token whose session was kicked
	// stops working immediately. Nil disables the check.
	sessionActive func(ctx context.Context, sessionID string) bool
}

// NewAuthenticator builds the unified auth middleware provider.
func NewAuthenticator(jwtMgr *jwt.Manager, apiKey *apikeysvc.Service, sessionActive func(context.Context, string) bool) *Authenticator {
	return &Authenticator{jwt: jwtMgr, apiKey: apiKey, sessionActive: sessionActive}
}

// Require rejects unauthenticated requests.
func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claims := a.resolve(r); claims != nil {
			// Single-session enforcement: reject a JWT whose session was revoked.
			if claims.SessionID != "" && a.sessionActive != nil && !a.sessionActive(r.Context(), claims.SessionID) {
				handlers.WriteProblem(w, r, http.StatusUnauthorized, "session ended (signed in elsewhere)")
				return
			}
			setActor(r.Context(), claims) // surface actor to the audit middleware
			next.ServeHTTP(w, r.WithContext(jwt.NewContextWithClaims(r.Context(), claims)))
			return
		}
		handlers.WriteProblem(w, r, http.StatusUnauthorized, "missing or invalid authentication")
	})
}

func (a *Authenticator) resolve(r *http.Request) *jwt.Claims {
	if token, ok := jwt.GetToken(r); ok {
		if claims, err := a.jwt.Verify(token); err == nil {
			return claims
		}
	}
	// Access token as a query param — required for browser-initiated file
	// downloads (an <a>/window.location navigation cannot set headers). Only
	// short-lived access tokens are accepted here, never refresh tokens.
	if token := r.URL.Query().Get("access_token"); token != "" {
		if claims, err := a.jwt.Verify(token); err == nil {
			return claims
		}
	}
	if key := r.Header.Get("X-API-Key"); key != "" && a.apiKey != nil {
		if p, err := a.apiKey.Authenticate(r.Context(), key, headers.GetClientIP(r)); err == nil {
			sub := p.UserID.String()
			return &jwt.Claims{Sub: &sub, Email: p.Email, Role: p.Role}
		}
	}
	return nil
}
