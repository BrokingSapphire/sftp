package middleware

import (
	"context"

	"sapphirebroking.com/sftp_service/pkg/jwt"
)

// actorHolder is a mutable, request-scoped slot the auth middleware fills once
// it resolves the caller. It lets an *outer* middleware (AuditLog) read the
// authenticated actor even though claims are set by an *inner* middleware —
// context values only propagate downward, but a pointer's contents do not.
type actorHolder struct{ claims *jwt.Claims }

type actorKey struct{}

// withActorHolder installs an empty holder on the context.
func withActorHolder(ctx context.Context) (context.Context, *actorHolder) {
	h := &actorHolder{}
	return context.WithValue(ctx, actorKey{}, h), h
}

// setActor records the resolved claims in the holder, if one is present.
func setActor(ctx context.Context, claims *jwt.Claims) {
	if h, ok := ctx.Value(actorKey{}).(*actorHolder); ok {
		h.claims = claims
	}
}
