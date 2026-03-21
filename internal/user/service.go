package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/naranjax/inhousepredictor/internal/httpx"
	"github.com/naranjax/inhousepredictor/internal/validate"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, name, email, password string) (*User, error) {
	if err := validate.Password(password); err != nil {
		return nil, httpx.NewError(400, httpx.CodeValidation, err.Error(), err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, httpx.NewError(500, httpx.CodeInternal, "Internal server error.", fmt.Errorf("hash password: %w", err))
	}
	return s.repo.Create(ctx, name, email, string(hash))
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (*User, error) {
	u, hash, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, query string, limit int) ([]*Summary, error) {
	return s.repo.List(ctx, strings.TrimSpace(query), limit)
}
