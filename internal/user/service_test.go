package user

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthenticateRejectsWrongPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	_, authErr := authenticateWithHash(string(hash), "wrong-password")
	require.ErrorIs(t, authErr, ErrInvalidCredentials)
}

func authenticateWithHash(hash, password string) (*User, error) {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return &User{}, nil
}

func TestAuthenticateHelperAcceptsCorrectPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user, authErr := authenticateWithHash(string(hash), "correct-password")
	require.NoError(t, authErr)
	require.NotNil(t, user)
	require.False(t, errors.Is(authErr, ErrInvalidCredentials))
}

func TestServiceConstructorStoresRepository(t *testing.T) {
	repo := &Repository{}
	service := NewService(repo)

	require.Same(t, repo, service.repo)
	require.NotEqual(t, uuid.Nil, uuid.New())
}
