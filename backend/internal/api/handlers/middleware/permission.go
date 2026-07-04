package middleware

import (
	"net/http"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/pkg/jwt"
)

// Permissions enforces RBAC by loading the caller's effective permissions
// (derived from their role) and checking required capabilities. Must run
// after JWT.Require.
type Permissions struct {
	q *sftpdb.Queries
}

// NewPermissions constructs the RBAC middleware provider.
func NewPermissions(q *sftpdb.Queries) *Permissions {
	return &Permissions{q: q}
}

// Require returns middleware that allows the request only if the caller holds
// ALL of the given permission slugs. "admin.all" acts as a wildcard.
func (p *Permissions) Require(perms ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			granted, ok := p.load(r)
			if !ok {
				handlers.WriteProblem(w, r, http.StatusUnauthorized, "authentication required")
				return
			}
			if granted["admin.all"] {
				next.ServeHTTP(w, r)
				return
			}
			for _, need := range perms {
				if !granted[need] {
					handlers.WriteProblem(w, r, http.StatusForbidden, "insufficient permissions")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (p *Permissions) load(r *http.Request) (map[string]bool, bool) {
	claims := jwt.GetClaimsFromContext(r.Context())
	if claims == nil || claims.Sub == nil {
		return nil, false
	}
	uid, err := uuid.Parse(*claims.Sub)
	if err != nil {
		return nil, false
	}
	slugs, err := p.q.GetPermissionsForUser(r.Context(), uid)
	if err != nil {
		return nil, false
	}
	set := make(map[string]bool, len(slugs))
	for _, s := range slugs {
		set[s] = true
	}
	return set, true
}
