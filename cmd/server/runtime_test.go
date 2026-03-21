package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/naranjax/inhousepredictor/internal/testutil"
)

func TestAllowedOriginsDefaultsForLocalDevelopment(t *testing.T) {
	prev := os.Getenv("ALLOWED_ORIGINS")
	t.Cleanup(func() { _ = os.Setenv("ALLOWED_ORIGINS", prev) })
	require.NoError(t, os.Unsetenv("ALLOWED_ORIGINS"))

	allowed := allowedOrigins()
	_, ok := allowed["http://localhost:3000"]
	require.True(t, ok)
	_, ok = allowed["http://127.0.0.1:8080"]
	require.True(t, ok)
}

func TestBootstrapAdminPromotesFirstAdminOnly(t *testing.T) {
	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `
		INSERT INTO users (name, email, password_hash)
		VALUES ('Admin Candidate', 'admin@example.com', $1)
	`, string(hash))
	require.NoError(t, err)

	require.NoError(t, bootstrapAdmin(context.Background(), pool, "admin@example.com"))

	var isAdmin bool
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT is_admin FROM users WHERE email = 'admin@example.com'`).Scan(&isAdmin))
	require.True(t, isAdmin)

	require.NoError(t, bootstrapAdmin(context.Background(), pool, "missing@example.com"))
	var adminCount int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM users WHERE is_admin = true`).Scan(&adminCount))
	require.Equal(t, 1, adminCount)
}
