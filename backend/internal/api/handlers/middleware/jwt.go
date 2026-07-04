package middleware

import (
	"net/http"

	"sapphirebroking.com/sftp_service/internal/api/handlers"
	"sapphirebroking.com/sftp_service/pkg/jwt"
)

// JWT verifies bearer access tokens and injects the claims into the context.
type JWT struct {
	manager *jwt.Manager
}

// NewJWT constructs the JWT middleware provider.
func NewJWT(manager *jwt.Manager) *JWT {
	return &JWT{manager: manager}
}

// Require validates the Authorization bearer token; on success it stores the
// claims on the request context, otherwise responds 401.
func (j *JWT) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := jwt.GetToken(r)
		if !ok {
			handlers.WriteProblem(w, r, http.StatusUnauthorized, "missing authentication token")
			return
		}
		claims, err := j.manager.Verify(token)
		if err != nil {
			handlers.WriteProblem(w, r, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		ctx := jwt.NewContextWithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
