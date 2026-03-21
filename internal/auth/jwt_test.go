package auth

import (
	"testing"

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
