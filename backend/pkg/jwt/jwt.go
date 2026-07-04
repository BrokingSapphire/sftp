package jwt

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned when a token fails verification.
var ErrInvalidToken = errors.New("invalid token")

// Manager issues and verifies HS256 access tokens.
type Manager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
}

// NewManager builds a token Manager.
func NewManager(secret, issuer string, accessTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), issuer: issuer, accessTTL: accessTTL}
}

// Issue signs a new access token for the given claims. Registered claims
// (issuer, subject, issued-at, expiry) are populated automatically.
func (m *Manager) Issue(sub, email, username, role, sessionID string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.accessTTL)
	subCopy := sub
	claims := Claims{
		Sub:       &subCopy,
		Email:     email,
		Username:  username,
		Role:      role,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   sub,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// Verify parses and validates a signed token, returning its claims.
func (m *Manager) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// GetToken extracts a bearer token from the Authorization header.
func GetToken(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	const bearerPrefix = "Bearer "
	if len(authHeader) <= len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", false
	}
	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", false
	}
	return token, true
}
