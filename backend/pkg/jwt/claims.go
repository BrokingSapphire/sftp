// Package jwt handles issuing and verifying the platform's access tokens.
package jwt

import (
	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload for an authenticated user.
//
// Sub is a pointer so downstream code (logger enrichment, middleware) can
// distinguish "no subject" from the zero value.
type Claims struct {
	Sub       *string `json:"sub,omitempty"`
	Email     string  `json:"email,omitempty"`
	Username  string  `json:"username,omitempty"`
	Role      string  `json:"role,omitempty"`
	SessionID string  `json:"sid,omitempty"`
	jwt.RegisteredClaims
}
