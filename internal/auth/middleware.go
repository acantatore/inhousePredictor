package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const (
	contextKeyUserID  contextKey = "user_id"
	contextKeyIsAdmin contextKey = "is_admin"
)

func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			claims, err := ParseToken(strings.TrimPrefix(header, "Bearer "), secret)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, contextKeyIsAdmin, claims.IsAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin, _ := r.Context().Value(contextKeyIsAdmin).(bool)
		if !isAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(contextKeyUserID).(uuid.UUID)
	return id, ok
}

func IsAdminFromContext(ctx context.Context) bool {
	ok, _ := ctx.Value(contextKeyIsAdmin).(bool)
	return ok
}
