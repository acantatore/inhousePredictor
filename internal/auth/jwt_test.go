package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewTokenRoundTrip(t *testing.T) {
	userID := uuid.New()
	secret := "super-secret"

	token, err := NewToken(userID, true, secret)
	require.NoError(t, err)

	claims, err := ParseToken(token, secret)
	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
	require.True(t, claims.IsAdmin)
	require.NotNil(t, claims.RegisteredClaims.ExpiresAt)
	require.NotNil(t, claims.RegisteredClaims.IssuedAt)
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	token, err := NewToken(uuid.New(), false, "right-secret")
	require.NoError(t, err)

	_, err = ParseToken(token, "wrong-secret")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestParseTokenRejectsGarbage(t *testing.T) {
	_, err := ParseToken("definitely-not-a-token", "secret")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenFromRequestPrefersQueryOverHeaderAndCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "cookie-token"})

	require.Equal(t, "query-token", TokenFromRequest(req))
}

func TestTokenFromRequestFallsBackToCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "cookie-token"})

	require.Equal(t, "cookie-token", TokenFromRequest(req))
}

func TestSetTokenCookieUsesHttpOnlyCookie(t *testing.T) {
	res := httptest.NewRecorder()
	SetTokenCookie(res, "token-value", mustParseRFC3339(t, "2026-03-21T12:00:00Z"))

	cookies := res.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, CookieName, cookies[0].Name)
	require.Equal(t, "token-value", cookies[0].Value)
	require.True(t, cookies[0].HttpOnly)
	require.Equal(t, "/", cookies[0].Path)
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}
