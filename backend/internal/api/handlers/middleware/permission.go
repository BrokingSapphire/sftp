package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/pkg/cache"
	"sapphirebroking.com/sftp_service/pkg/jwt"
)

// permsTTL is short so role/permission changes propagate quickly.
const permsTTL = 30 * time.Second

// Permissions enforces RBAC by loading the caller's effective permissions
// (derived from their role) and checking required capabilities. The lookup is
// cached (per user) to keep it off the database on every request. Must run
// after JWT.Require.
type Permissions struct {
	q     *sftpdb.Queries
	cache cache.Cache
}

// NewPermissions constructs the RBAC middleware provider. cache may be nil.
func NewPermissions(q *sftpdb.Queries, c cache.Cache) *Permissions {
	return &Permissions{q: q, cache: c}
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

	ctx := r.Context()
	key := "perms:" + uid.String()
	if p.cache != nil {
		if b, ok := p.cache.Get(ctx, key); ok {
			var slugs []string
			if json.Unmarshal(b, &slugs) == nil {
				return toSet(slugs), true
			}
		}
	}

	slugs, err := p.q.GetPermissionsForUser(ctx, uid)
	if err != nil {
		return nil, false
	}
	if p.cache != nil {
		if b, err := json.Marshal(slugs); err == nil {
			p.cache.Set(ctx, key, b, permsTTL)
		}
	}
	return toSet(slugs), true
}

func toSet(slugs []string) map[string]bool {
	set := make(map[string]bool, len(slugs))
	for _, s := range slugs {
		set[s] = true
	}
	return set
}
