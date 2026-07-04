// Package sso implements OpenID Connect single-sign-on providers.
package sso

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"sapphirebroking.com/sftp_service/internal/config"
	authsvc "sapphirebroking.com/sftp_service/internal/service/auth"
)

// Microsoft wraps the Entra ID (Azure AD) OIDC provider and OAuth2 config.
type Microsoft struct {
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
	cfg      config.MicrosoftSSOConfig
}

// NewMicrosoft performs OIDC discovery against the configured tenant and
// returns a ready-to-use provider. Returns (nil, nil) when SSO is disabled.
func NewMicrosoft(ctx context.Context, cfg config.MicrosoftSSOConfig) (*Microsoft, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("microsoft sso enabled but client_id/client_secret missing")
	}

	issuer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", cfg.TenantID)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}

	return &Microsoft{
		oauth: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		cfg:      cfg,
	}, nil
}

// AuthCodeURL builds the provider authorization URL for the given state.
func (m *Microsoft) AuthCodeURL(state string) string {
	return m.oauth.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange swaps an auth code for tokens and verifies the ID token, returning
// the normalised profile.
func (m *Microsoft) Exchange(ctx context.Context, code string) (authsvc.SSOProfile, error) {
	tok, err := m.oauth.Exchange(ctx, code)
	if err != nil {
		return authsvc.SSOProfile{}, fmt.Errorf("token exchange: %w", err)
	}
	rawID, ok := tok.Extra("id_token").(string)
	if !ok || rawID == "" {
		return authsvc.SSOProfile{}, fmt.Errorf("no id_token in response")
	}
	idToken, err := m.verifier.Verify(ctx, rawID)
	if err != nil {
		return authsvc.SSOProfile{}, fmt.Errorf("verify id_token: %w", err)
	}

	var claims struct {
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return authsvc.SSOProfile{}, fmt.Errorf("parse claims: %w", err)
	}

	email := claims.Email
	if email == "" {
		email = claims.PreferredUsername // Entra often puts UPN here
	}
	return authsvc.SSOProfile{Email: email, FullName: claims.Name, Provider: "microsoft"}, nil
}

// Config returns the underlying provider config.
func (m *Microsoft) Config() config.MicrosoftSSOConfig { return m.cfg }
