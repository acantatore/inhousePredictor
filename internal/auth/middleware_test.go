package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareAcceptsAuthCookie(t *testing.T) {
	secret := "secret"
	userID := uuid.New()
	token, err := NewToken(userID, true, secret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	res := httptest.NewRecorder()

	called := false
	Middleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		ctxUserID, ok := UserIDFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, userID, ctxUserID)
		require.True(t, IsAdminFromContext(r.Context()))
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(res, req)

	require.True(t, called)
	require.Equal(t, http.StatusNoContent, res.Code)
}
