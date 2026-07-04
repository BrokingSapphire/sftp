package jwt

import "context"

type claimsContextKey struct{}

// NewContextWithClaims returns a copy of ctx carrying the verified claims.
func NewContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// GetClaimsFromContext returns the claims stored in ctx, or nil if none.
func GetClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsContextKey{}).(*Claims)
	return claims
}
