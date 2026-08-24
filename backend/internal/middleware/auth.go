package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/juanbedoya/hnl-bank/backend/pkg/response"
)

type ctxKey string

// UserIDKey is the context key holding the authenticated user id.
const UserIDKey ctxKey = "user_id"

// TokenValidator validates a bearer token and returns the user id.
type TokenValidator func(ctx context.Context, token string) (string, error)

// JWTAuth protects handlers that require an authenticated user.
func JWTAuth(validate TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				response.Error(w, http.StatusUnauthorized, "missing bearer token", "unauthorized")
				return
			}
			token := strings.TrimPrefix(auth, "Bearer ")
			userID, err := validate(r.Context(), token)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid or expired session", "unauthorized")
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID returns the authenticated user id from a request context.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(UserIDKey).(string)
	return id
}