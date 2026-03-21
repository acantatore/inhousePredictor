package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/naranjax/inhousepredictor/internal/httpx"
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
				httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", nil))
				return
			}
			claims, err := ParseToken(strings.TrimPrefix(header, "Bearer "), secret)
			if err != nil {
				httpx.WriteError(w, httpx.NewError(http.StatusUnauthorized, httpx.CodeUnauthorized, "Unauthorized.", err))
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
			httpx.WriteError(w, httpx.NewError(http.StatusForbidden, httpx.CodeForbidden, "Forbidden.", nil))
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
