package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naranjax/inhousepredictor/internal/httpx"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, name, email, passwordHash string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, balance, is_admin, created_at
	`, name, email, passwordHash).Scan(
		&u.ID, &u.Name, &u.Email, &u.Balance, &u.IsAdmin, &u.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, httpx.NewError(409, httpx.CodeConflict, "That email is already registered.", err)
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// GetByEmail returns the user and their password hash for authentication.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, string, error) {
	u := &User{}
	var hash string
	err := r.db.QueryRow(ctx, `
		SELECT id, name, email, balance, is_admin, created_at, password_hash
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Name, &u.Email, &u.Balance, &u.IsAdmin, &u.CreatedAt, &hash)
	if err != nil {
		return nil, "", fmt.Errorf("get user by email: %w", err)
	}
	return u, hash, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, email, balance, is_admin, created_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Name, &u.Email, &u.Balance, &u.IsAdmin, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (r *Repository) List(ctx context.Context, query string, limit int) ([]*Summary, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	search := strings.TrimSpace(query)
	if search != "" {
		search = "%" + search + "%"
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, name, email, is_admin
		FROM users
		WHERE ($1 = '' OR name ILIKE $1 OR email ILIKE $1)
		ORDER BY LOWER(name) ASC, LOWER(email) ASC
		LIMIT $2
	`, search, limit)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*Summary
	for rows.Next() {
		user := &Summary{}
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.IsAdmin); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	return users, rows.Err()
}
